package handlers

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/lucaskampi/paywall-api/ws"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
)

// Webhook processes Stripe webhooks and marks payments as paid.
func Webhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if dbConn == nil {
		http.Error(w, "server not ready", http.StatusServiceUnavailable)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	webhookSecret := strings.TrimSpace(os.Getenv("STRIPE_WEBHOOK_SECRET"))
	if webhookSecret != "" {
		signatureHeader := strings.TrimSpace(r.Header.Get("Stripe-Signature"))
		if signatureHeader == "" {
			http.Error(w, "missing signature", http.StatusBadRequest)
			return
		}
		if _, err := webhook.ConstructEvent(body, signatureHeader, webhookSecret); err != nil {
			http.Error(w, "invalid signature", http.StatusBadRequest)
			return
		}
	}

	var event stripe.Event
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	eventType := strings.TrimSpace(string(event.Type))
	eventID := strings.TrimSpace(event.ID)
	if eventID == "" {
		eventID = "no-event-id"
	}

	intentID := ""
	status := "pending"
	amountReceived := int64(0)

	if event.Data.Raw != nil {
		var intent stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &intent); err == nil {
			intentID = strings.TrimSpace(intent.ID)
			status = normalizeProviderStatus(string(intent.Status))
			if intent.AmountReceived > 0 {
				amountReceived = intent.AmountReceived
			}
		}
	}

	if intentID == "" {
		w.WriteHeader(http.StatusOK)
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
		"INSERT INTO webhook_events (provider, event_id) VALUES ($1, $2) ON CONFLICT (provider, event_id) DO NOTHING",
		"stripe",
		eventID,
	)
	if err != nil {
		http.Error(w, "db write failed", http.StatusInternalServerError)
		return
	}
	if ra, _ := res.RowsAffected(); ra == 0 {
		alreadyProcessed = true
	}

	// We still broadcast on retries because WS clients may connect after the first delivery.
	wsSessionID := intentID
	wsEventType := "stripe." + strings.ToLower(strings.ReplaceAll(eventType, " ", "_"))
	wsData := map[string]interface{}{}
	wsData["payment_intent_id"] = intentID
	wsData["status"] = status
	wsData["event"] = eventType

	if !alreadyProcessed {
		if status == "paid" {
			if _, err := tx.Exec(
				"UPDATE payments SET status = $1, amount_cents = CASE WHEN $2 > 0 THEN $2 ELSE amount_cents END, paid_at = COALESCE(paid_at, CURRENT_TIMESTAMP), updated_at = CURRENT_TIMESTAMP WHERE stripe_payment_intent_id = $3 OR provider_charge_id = $3",
				status,
				amountReceived,
				intentID,
			); err != nil {
				http.Error(w, "db update failed", http.StatusInternalServerError)
				return
			}
		} else {
			if _, err := tx.Exec(
				"UPDATE payments SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE stripe_payment_intent_id = $2 OR provider_charge_id = $2",
				status,
				intentID,
			); err != nil {
				http.Error(w, "db update failed", http.StatusInternalServerError)
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		if err == sql.ErrTxDone {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, "db commit failed", http.StatusInternalServerError)
		return
	}

	if wsSessionID != "" {
		ws.BroadcastToSession(wsSessionID, ws.Event{
			Type: wsEventType,
			Data: wsData,
			At:   time.Now().UTC().Format(time.RFC3339Nano),
		})
	}

	w.WriteHeader(http.StatusOK)
}
