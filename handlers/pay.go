package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/lucaskampi/paywall-api/ws"
)

var dbConn *sql.DB

// Init sets up handlers package with DB connection.
func Init(conn *sql.DB) {
	dbConn = conn
}

// Pay handles POST /pay and writes a payment row into the payments table.
func Pay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
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
	if payload.AmountCents <= 0 {
		http.Error(w, "missing fields", http.StatusBadRequest)
		return
	}

	if dbConn == nil {
		http.Error(w, "server not ready", http.StatusServiceUnavailable)
		return
	}

	client, err := newAbacatePayClientFromEnv()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	description := fmt.Sprintf("%s - %s", payload.Name, payload.Link)
	billing, err := client.CreateBilling(payload.AmountCents, description, payload.Name, payload.Email)
	if err != nil {
		log.Printf("abacatepay create billing failed: %v", err)
		http.Error(w, "failed to create abacatepay billing", http.StatusBadGateway)
		return
	}

	var paymentID int64
	var createdAt time.Time
	if err := dbConn.QueryRow(
		"INSERT INTO payments (name, link, email, amount_cents, status, currency, provider, provider_charge_id, provider_checkout_url) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id, created_at",
		payload.Name,
		payload.Link,
		payload.Email,
		payload.AmountCents,
		normalizeProviderStatus(billing.Status),
		"brl",
		"abacatepay",
		billing.ID,
		billing.URL,
	).Scan(&paymentID, &createdAt); err != nil {
		http.Error(w, "write failed", http.StatusInternalServerError)
		return
	}

	broadcastPaymentCreated(paymentID, payload.Name, payload.Link, payload.Email, payload.AmountCents, createdAt)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status":       "created",
		"checkout_url": billing.URL,
		"session_id":   billing.ID,
		"billing_id":   billing.ID,
	})
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
