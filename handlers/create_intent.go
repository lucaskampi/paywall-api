package handlers

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/lucaskampi/paywall-api/db"
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/paymentintent"
)

// CreatePaymentIntent creates a Stripe PaymentIntent and returns the client secret.
// This endpoint is for in-app payment flows using Stripe Elements.
func CreatePaymentIntent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	stripeSecretKey := os.Getenv("STRIPE_SECRET_KEY")
	if stripeSecretKey == "" {
		stripeSecretKey = os.Getenv("STRIPE_KEY")
	}
	currency := os.Getenv("STRIPE_CURRENCY")
	if currency == "" {
		currency = "usd"
	}

	if stripeSecretKey == "" {
		http.Error(w, "missing STRIPE_SECRET_KEY", http.StatusInternalServerError)
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
	if payload.Name == "" || payload.AmountCents <= 0 {
		http.Error(w, "missing fields", http.StatusBadRequest)
		return
	}

	if writeCh == nil {
		http.Error(w, "server not ready", http.StatusServiceUnavailable)
		return
	}

	stripe.Key = stripeSecretKey

	// Create PaymentIntent
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(payload.AmountCents),
		Currency: stripe.String(currency),
		Metadata: map[string]string{
			"name":  payload.Name,
			"link":  payload.Link,
			"email": payload.Email,
		},
	}
	if payload.Email != "" {
		params.ReceiptEmail = stripe.String(payload.Email)
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		http.Error(w, "failed to create payment intent", http.StatusBadGateway)
		return
	}

	// Insert payment row with status='pending' and store payment_intent_id
	errCh := make(chan error, 1)
	idCh := make(chan int64, 1)
	writeCh <- db.WriteRequest{
		Query: "INSERT INTO payments (name, link, email, amount_cents, status, currency, provider, stripe_payment_intent_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		Args:  []interface{}{payload.Name, payload.Link, payload.Email, payload.AmountCents, "pending", currency, "stripe", pi.ID},
		ErrCh: errCh,
		IDCh:  idCh,
	}
	if err := <-errCh; err != nil {
		http.Error(w, "write failed", http.StatusInternalServerError)
		return
	}

	var paymentID int64
	select {
	case paymentID = <-idCh:
		// ok
	default:
		// no id available
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"clientSecret": pi.ClientSecret,
		"paymentId":    paymentID,
	})
}
