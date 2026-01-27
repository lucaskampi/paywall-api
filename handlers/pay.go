package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/lucaskampi/paywall-api/db"
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

	errCh := make(chan error, 1)
	writeCh <- db.WriteRequest{
		Query: "INSERT INTO payments (name, link, email, amount_cents) VALUES (?, ?, ?, ?)",
		Args:  []interface{}{payload.Name, payload.Link, payload.Email, payload.AmountCents},
		ErrCh: errCh,
	}
	if err := <-errCh; err != nil {
		http.Error(w, "write failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
