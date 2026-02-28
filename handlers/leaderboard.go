package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type Payment struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Link        string `json:"link,omitempty"`
	Email       string `json:"email,omitempty"`
	AmountCents int64  `json:"amount_cents"`
	CreatedAt   string `json:"created_at"`
}

// Leaderboard returns payments ordered by amount_cents desc (highest first).
func Leaderboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// dbConn is set during handlers.Init
	if dbConn == nil {
		http.Error(w, "server not ready", http.StatusServiceUnavailable)
		return
	}

	rows, err := dbConn.Query("SELECT id, name, link, email, amount_cents, created_at FROM payments WHERE status = 'paid' ORDER BY amount_cents DESC LIMIT 100")
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var out []Payment
	for rows.Next() {
		var p Payment
		var createdAt time.Time
		if err := rows.Scan(&p.ID, &p.Name, &p.Link, &p.Email, &p.AmountCents, &createdAt); err != nil {
			if err == sql.ErrNoRows {
				break
			}
			http.Error(w, "db scan error", http.StatusInternalServerError)
			return
		}
		p.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		out = append(out, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}
