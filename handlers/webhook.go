package handlers

import (
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/lucaskampi/paywall-api/ws"
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

	webhookSecret := strings.TrimSpace(os.Getenv("STRIPE_WEBHOOK_SECRET"))
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
	if strings.TrimSpace(sigHeader) == "" {
		log.Printf("stripe webhook: missing Stripe-Signature header")
		http.Error(w, "missing signature", http.StatusBadRequest)
		return
	}
	// Allow a bit more tolerance for clock skew (common in WSL/VMs).
	// Accept events with a different API version than this library expects
	// to avoid hard-failing local Stripe CLI deliveries. This sets a
	// tolerance and opts into ignoring API version mismatches. In
	// production you should set your webhook endpoint's API version to
	// match the library or remove the ignore flag.
	event, err := webhook.ConstructEventWithOptions(body, sigHeader, webhookSecret, webhook.ConstructEventOptions{
		Tolerance:                10 * time.Minute,
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		log.Printf("stripe webhook: signature verification failed: %v", err)
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}

	alreadyProcessed := false

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
		alreadyProcessed = true
	}

	// We still parse and broadcast on retries, even if alreadyProcessed,
	// because WS clients may have connected after the first delivery.
	var wsSessionID string
	wsEventType := ""
	wsData := map[string]interface{}{}

	switch event.Type {
	case "checkout.session.completed":
		var cs stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &cs); err != nil {
			log.Printf("stripe webhook: unmarshal checkout.session.completed failed: %v", err)
			http.Error(w, "invalid event payload", http.StatusBadRequest)
			return
		}

		paymentIntentID := ""
		if cs.PaymentIntent != nil {
			paymentIntentID = cs.PaymentIntent.ID
		}

		wsSessionID = cs.ID
		wsEventType = "stripe.checkout.session.completed"
		wsData["session_id"] = cs.ID
		wsData["payment_intent_id"] = paymentIntentID
		wsData["status"] = "paid"

		if !alreadyProcessed {
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
				// Log but don't fail - test events won't have a matching payment row
				log.Printf("stripe webhook: no payment found for session %s (test event?)", cs.ID)
			}
		}

	case "checkout.session.expired":
		var cs stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &cs); err != nil {
			log.Printf("stripe webhook: unmarshal checkout.session.expired failed: %v", err)
			http.Error(w, "invalid event payload", http.StatusBadRequest)
			return
		}

		wsSessionID = cs.ID
		wsEventType = "stripe.checkout.session.expired"
		wsData["session_id"] = cs.ID
		wsData["status"] = "expired"

		if !alreadyProcessed {
			_, err := tx.Exec(
				"UPDATE payments SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE stripe_checkout_session_id = ?",
				"expired",
				cs.ID,
			)
			if err != nil {
				http.Error(w, "db update failed", http.StatusInternalServerError)
				return
			}
		}

	case "payment_intent.succeeded":
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			log.Printf("stripe webhook: unmarshal payment_intent.succeeded failed: %v", err)
			http.Error(w, "invalid event payload", http.StatusBadRequest)
			return
		}

		wsEventType = "stripe.payment_intent.succeeded"
		wsData["payment_intent_id"] = pi.ID
		wsData["status"] = "paid"

		if !alreadyProcessed {
			upd, err := tx.Exec(
				"UPDATE payments SET status = ?, paid_at = COALESCE(paid_at, CURRENT_TIMESTAMP), updated_at = CURRENT_TIMESTAMP WHERE stripe_payment_intent_id = ?",
				"paid",
				pi.ID,
			)
			if err != nil {
				http.Error(w, "db update failed", http.StatusInternalServerError)
				return
			}
			if ra, _ := upd.RowsAffected(); ra == 0 {
				log.Printf("stripe webhook: no payment found for payment_intent %s (test event?)", pi.ID)
			}
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

	if wsEventType != "" && wsSessionID != "" {
		ws.BroadcastToSession(wsSessionID, ws.Event{
			Type: wsEventType,
			Data: wsData,
			At:   time.Now().UTC().Format(time.RFC3339Nano),
		})
	}

	w.WriteHeader(http.StatusOK)
}
