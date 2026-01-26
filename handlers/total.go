package handlers

import (
	"net/http"
)

// Total returns the total invested amount placeholder.
func Total(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"total_cents":0}`))
}
