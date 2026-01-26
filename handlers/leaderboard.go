package handlers

import (
	"net/http"
)

// Leaderboard returns a placeholder empty list for now.
func Leaderboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("[]"))
}
