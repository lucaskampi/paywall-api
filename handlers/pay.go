package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/lucaskampi/paywall-api/ws"
	"github.com/stripe/stripe-go/v82"
)

var dbConn *sql.DB

// Init sets up handlers package with DB connection.
func Init(conn *sql.DB) {
	dbConn = conn
}

// Pay keeps backward compatibility by delegating to CreatePaymentIntent.
func Pay(w http.ResponseWriter, r *http.Request) {
	CreatePaymentIntent(w, r)
}

type payConfirmRequest struct {
	PaymentIntentID string `json:"payment_intent_id"`
	Name            string `json:"name"`
	Link            string `json:"link"`
	Email           string `json:"email"`
	AmountCents     int64  `json:"amount_cents"`
}

// PayConfirm verifies the Stripe payment intent and stores it in the DB.
func PayConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if dbConn == nil {
		http.Error(w, "server not ready", http.StatusServiceUnavailable)
		return
	}

	var payload payConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	payload.Name = firstNonEmpty(payload.Name)
	payload.Link = firstNonEmpty(payload.Link)
	payload.Email = firstNonEmpty(payload.Email)
	payload.PaymentIntentID = firstNonEmpty(payload.PaymentIntentID)

	if payload.PaymentIntentID == "" || payload.Name == "" || payload.AmountCents <= 0 {
		http.Error(w, "missing fields", http.StatusBadRequest)
		return
	}

	stripeClient, err := newStripeClientFromEnv()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	intent, err := stripeClient.GetPaymentIntent(payload.PaymentIntentID)
	if err != nil {
		http.Error(w, "failed to fetch payment intent", http.StatusBadGateway)
		return
	}

	if intent.Status != stripe.PaymentIntentStatusSucceeded {
		http.Error(w, "payment not completed", http.StatusBadRequest)
		return
	}

	amountCents := payload.AmountCents
	if intent.AmountReceived > 0 {
		amountCents = intent.AmountReceived
	}

	var paymentID int64
	var createdAt time.Time
	if err := dbConn.QueryRow(
		`INSERT INTO payments (name, link, email, amount_cents, status, currency, provider, provider_charge_id, stripe_payment_intent_id, paid_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		 ON CONFLICT (stripe_payment_intent_id)
		 DO UPDATE SET
		  name = EXCLUDED.name,
		  link = EXCLUDED.link,
		  email = EXCLUDED.email,
		  amount_cents = EXCLUDED.amount_cents,
		  status = EXCLUDED.status,
		  provider = EXCLUDED.provider,
		  provider_charge_id = EXCLUDED.provider_charge_id,
		  paid_at = COALESCE(payments.paid_at, CURRENT_TIMESTAMP),
		  updated_at = CURRENT_TIMESTAMP
		 RETURNING id, created_at`,
		payload.Name,
		payload.Link,
		payload.Email,
		amountCents,
		"paid",
		stringsToLowerOrDefault(string(intent.Currency), "usd"),
		"stripe",
		intent.ID,
		intent.ID,
	).Scan(&paymentID, &createdAt); err != nil {
		http.Error(w, "write failed", http.StatusInternalServerError)
		return
	}

	broadcastPaymentCreated(paymentID, payload.Name, payload.Link, payload.Email, amountCents, createdAt)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id":           paymentID,
		"name":         payload.Name,
		"link":         payload.Link,
		"email":        payload.Email,
		"amount_cents": amountCents,
		"status":       "paid",
		"provider":     "stripe",
	})
}

func stringsToLowerOrDefault(value string, fallback string) string {
	v := strings.TrimSpace(strings.ToLower(value))
	if v == "" {
		return fallback
	}
	return v
}

func broadcastPaymentCreated(id int64, name string, link string, email string, amount int64, createdAt time.Time) {
	ws.Broadcast(map[string]interface{}{
		"type":  "payment_created",
		"event": "payment.created",
		"payment": map[string]interface{}{
			"id":           id,
			"name":         name,
			"link":         link,
			"email":        email,
			"amount_cents": amount,
			"created_at":   createdAt.UTC().Format(time.RFC3339),
		},
	})
}
