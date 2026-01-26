package handlers

import (
	"net/http"
)

// Pay initiates a payment flow — placeholder that returns a checkout URL.
func Pay(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"checkout_url":"https://example.com/checkout"}`))
}
