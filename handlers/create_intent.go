package handlers

import (
	"encoding/json"
	"net/http"
)

type createIntentRequest struct {
	Name        string `json:"name"`
	Link        string `json:"link"`
	Email       string `json:"email"`
	AmountCents int64  `json:"amount_cents"`
}

// CreatePaymentIntent creates a Stripe payment intent and returns a client secret.
func CreatePaymentIntent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var payload createIntentRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	payload.Name = firstNonEmpty(payload.Name)
	payload.Link = firstNonEmpty(payload.Link)
	payload.Email = firstNonEmpty(payload.Email)

	if payload.Name == "" || payload.AmountCents <= 0 {
		http.Error(w, "missing fields", http.StatusBadRequest)
		return
	}

	stripeClient, err := newStripeClientFromEnv()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	intent, err := stripeClient.CreatePaymentIntent(payload.AmountCents, payload.Name, payload.Link, payload.Email)
	if err != nil {
		http.Error(w, "failed to create payment intent", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"clientSecret":    intent.ClientSecret,
		"paymentIntentId": intent.ID,
	})
}
