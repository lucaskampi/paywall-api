package handlers

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/webhook"
)

// Webhook processes payment provider webhooks (Stripe) and marks payments as paid.
func Webhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if dbConn == nil {
		http.Error(w, "server not ready", http.StatusServiceUnavailable)
		return
	}

	webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	if webhookSecret == "" {
		http.Error(w, "missing STRIPE_WEBHOOK_SECRET", http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	sigHeader := r.Header.Get("Stripe-Signature")
	event, err := webhook.ConstructEvent(body, sigHeader, webhookSecret)
	if err != nil {
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}

	tx, err := dbConn.Begin()
	if err != nil {
		http.Error(w, "db begin failed", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Idempotency: record the event id; if already processed, return 200.
	res, err := tx.Exec(
		"INSERT OR IGNORE INTO webhook_events (provider, event_id) VALUES (?, ?)",
		"stripe",
		event.ID,
	)
	if err != nil {
		http.Error(w, "db write failed", http.StatusInternalServerError)
		return
	}
	if ra, _ := res.RowsAffected(); ra == 0 {
		_ = tx.Commit()
		w.WriteHeader(http.StatusOK)
		return
	}

	switch event.Type {
	case "checkout.session.completed":
		var cs stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &cs); err != nil {
			http.Error(w, "invalid event payload", http.StatusBadRequest)
			return
		}

		paymentIntentID := ""
		if cs.PaymentIntent != nil {
			paymentIntentID = cs.PaymentIntent.ID
		}

		upd, err := tx.Exec(
			"UPDATE payments SET status = ?, stripe_payment_intent_id = CASE WHEN ? != '' THEN ? ELSE stripe_payment_intent_id END, paid_at = COALESCE(paid_at, CURRENT_TIMESTAMP), updated_at = CURRENT_TIMESTAMP WHERE stripe_checkout_session_id = ?",
			"paid",
			paymentIntentID,
			paymentIntentID,
			cs.ID,
		)
		if err != nil {
			http.Error(w, "db update failed", http.StatusInternalServerError)
			return
		}
		if ra, _ := upd.RowsAffected(); ra == 0 {
			http.Error(w, "payment not found", http.StatusInternalServerError)
			return
		}

	case "checkout.session.expired":
		var cs stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &cs); err != nil {
			http.Error(w, "invalid event payload", http.StatusBadRequest)
			return
		}

		_, err := tx.Exec(
			"UPDATE payments SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE stripe_checkout_session_id = ?",
			"expired",
			cs.ID,
		)
		if err != nil {
			http.Error(w, "db update failed", http.StatusInternalServerError)
			return
		}

	default:
		// Ignore events we don't care about.
	}

	if err := tx.Commit(); err != nil {
		if err == sql.ErrTxDone {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, "db commit failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
