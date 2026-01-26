package handlers

import (
	"net/http"
)

// Webhook processes payment provider webhooks. For now it just returns 200.
func Webhook(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
