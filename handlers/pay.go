package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/url"
	"os"

	"github.com/lucaskampi/paywall-api/db"
	"github.com/stripe/stripe-go/v79"
	checkoutsession "github.com/stripe/stripe-go/v79/checkout/session"
	stripeprice "github.com/stripe/stripe-go/v79/price"
)

var writeCh chan<- db.WriteRequest
var dbConn *sql.DB

// Init sets up handlers package with DB connection and writer channel.
func Init(conn *sql.DB, ch chan<- db.WriteRequest) {
	dbConn = conn
	writeCh = ch
}

// Pay handles POST /pay and writes a payment row into the payments table.
func Pay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	stripeSecretKey := os.Getenv("STRIPE_SECRET_KEY")
	if stripeSecretKey == "" {
		// Backward compatibility with earlier config name.
		stripeSecretKey = os.Getenv("STRIPE_KEY")
	}
	stripePriceID := os.Getenv("STRIPE_PRICE_ID")
	stripeProductID := os.Getenv("STRIPE_PRODUCT_ID")
	successURL := os.Getenv("STRIPE_SUCCESS_URL")
	cancelURL := os.Getenv("STRIPE_CANCEL_URL")
	currency := os.Getenv("STRIPE_CURRENCY")
	if currency == "" {
		currency = "usd"
	}
	productName := os.Getenv("STRIPE_PRODUCT_NAME")
	if productName == "" {
		productName = "Paywall payment"
	}

	if stripeSecretKey == "" {
		http.Error(w, "missing STRIPE_SECRET_KEY", http.StatusInternalServerError)
		return
	}
	if successURL == "" || cancelURL == "" {
		http.Error(w, "missing STRIPE_SUCCESS_URL or STRIPE_CANCEL_URL", http.StatusInternalServerError)
		return
	}
	if _, err := url.ParseRequestURI(successURL); err != nil {
		http.Error(w, "invalid STRIPE_SUCCESS_URL", http.StatusInternalServerError)
		return
	}
	if _, err := url.ParseRequestURI(cancelURL); err != nil {
		http.Error(w, "invalid STRIPE_CANCEL_URL", http.StatusInternalServerError)
		return
	}

	var payload struct {
		Name        string `json:"name"`
		Link        string `json:"link"`
		Email       string `json:"email"`
		AmountCents int64  `json:"amount_cents"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if payload.Name == "" {
		http.Error(w, "missing fields", http.StatusBadRequest)
		return
	}
	if stripePriceID == "" && payload.AmountCents <= 0 {
		http.Error(w, "missing fields", http.StatusBadRequest)
		return
	}

	if writeCh == nil {
		http.Error(w, "server not ready", http.StatusServiceUnavailable)
		return
	}

	stripe.Key = stripeSecretKey
	params := &stripe.CheckoutSessionParams{
		Mode:               stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL:         stripe.String(successURL),
		CancelURL:          stripe.String(cancelURL),
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		Metadata: map[string]string{
			"name":  payload.Name,
			"link":  payload.Link,
			"email": payload.Email,
		},
	}
	if payload.Email != "" {
		params.CustomerEmail = stripe.String(payload.Email)
	}

	amountCentsForDB := payload.AmountCents
	currencyForDB := currency

	// Prefer a pre-created Stripe Price (fixed amount) if provided.
	if stripePriceID != "" {
		p, err := stripeprice.Get(stripePriceID, nil)
		if err != nil {
			http.Error(w, "invalid STRIPE_PRICE_ID", http.StatusBadGateway)
			return
		}
		if p != nil {
			// Keep DB consistent with Stripe's configured amount/currency.
			if p.UnitAmount > 0 {
				amountCentsForDB = p.UnitAmount
			}
			if p.Currency != "" {
				currencyForDB = string(p.Currency)
			}
		}
		params.LineItems = []*stripe.CheckoutSessionLineItemParams{
			{
				Quantity: stripe.Int64(1),
				Price:    stripe.String(stripePriceID),
			},
		}
	} else {
		// Dynamic amount: create price data on the fly.
		pd := &stripe.CheckoutSessionLineItemPriceDataParams{
			Currency:   stripe.String(currency),
			UnitAmount: stripe.Int64(payload.AmountCents),
		}
		if stripeProductID != "" {
			pd.Product = stripe.String(stripeProductID)
		} else {
			pd.ProductData = &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
				Name: stripe.String(productName),
			}
		}

		params.LineItems = []*stripe.CheckoutSessionLineItemParams{
			{
				Quantity:  stripe.Int64(1),
				PriceData: pd,
			},
		}
	}

	cs, err := checkoutsession.New(params)
	if err != nil {
		http.Error(w, "failed to create checkout session", http.StatusBadGateway)
		return
	}

	errCh := make(chan error, 1)
	writeCh <- db.WriteRequest{
		Query: "INSERT INTO payments (name, link, email, amount_cents, status, currency, provider, stripe_checkout_session_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		Args:  []interface{}{payload.Name, payload.Link, payload.Email, amountCentsForDB, "pending", currencyForDB, "stripe", cs.ID},
		ErrCh: errCh,
	}
	if err := <-errCh; err != nil {
		http.Error(w, "write failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status":       "created",
		"checkout_url": cs.URL,
		"session_id":   cs.ID,
	})
}
