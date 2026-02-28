package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/lucaskampi/paywall-api/ws"
)

// Webhook processes payment provider webhooks (AbacatePay) and marks payments as paid.
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

	webhookSecret := strings.TrimSpace(os.Getenv("ABACATEPAY_WEBHOOK_SECRET"))
	if webhookSecret != "" {
		signatureHeader := firstNonEmpty(
			r.Header.Get("X-AbacatePay-Signature"),
			r.Header.Get("AbacatePay-Signature"),
		)
		if !verifyAbacatePaySignature(body, signatureHeader, webhookSecret) {
			log.Printf("abacatepay webhook: invalid signature")
			http.Error(w, "invalid signature", http.StatusBadRequest)
			return
		}
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	eventType := firstNonEmpty(
		nestedString(payload, []string{"event"}),
		nestedString(payload, []string{"type"}),
	)
	billingID := firstNonEmpty(
		nestedString(payload, []string{"data", "billing", "id"}),
		nestedString(payload, []string{"data", "id"}),
		nestedString(payload, []string{"billing", "id"}),
		nestedString(payload, []string{"id"}),
	)
	status := normalizeProviderStatus(firstNonEmpty(
		nestedString(payload, []string{"data", "status"}),
		nestedString(payload, []string{"status"}),
		eventType,
	))

	eventID := firstNonEmpty(
		nestedString(payload, []string{"eventId"}),
		nestedString(payload, []string{"event_id"}),
		nestedString(payload, []string{"id"}),
	)
	if eventID == "" {
		hash := sha256.Sum256(body)
		eventID = fmt.Sprintf("body:%x", hash)
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
		"abacatepay",
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
	wsSessionID := billingID
	wsEventType := "abacatepay.webhook"
	if eventType != "" {
		wsEventType = "abacatepay." + strings.ToLower(strings.ReplaceAll(eventType, " ", "_"))
	}
	wsData := map[string]interface{}{}
	wsData["billing_id"] = billingID
	wsData["status"] = status
	wsData["event"] = eventType

	if !alreadyProcessed && billingID != "" {
		if status == "paid" {
			if _, err := tx.Exec(
				"UPDATE payments SET status = $1, paid_at = COALESCE(paid_at, CURRENT_TIMESTAMP), updated_at = CURRENT_TIMESTAMP WHERE provider_charge_id = $2",
				status,
				billingID,
			); err != nil {
				http.Error(w, "db update failed", http.StatusInternalServerError)
				return
			}
		} else {
			if _, err := tx.Exec(
				"UPDATE payments SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE provider_charge_id = $2",
				status,
				billingID,
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
