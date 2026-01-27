package db

import (
	"database/sql"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/lucaskampi/paywall-api/ws"
)

// WriteRequest represents a request to execute a write SQL statement.
type WriteRequest struct {
	Query string
	Args  []interface{}
	ErrCh chan error
}

// StartWriter starts a single goroutine that serializes write requests to the DB.
// It returns the channel to send WriteRequest values and a stop function.
func StartWriter(db *sql.DB) (chan<- WriteRequest, func()) {
	ch := make(chan WriteRequest)
	stop := make(chan struct{})

	go func() {
		for {
			select {
			case wr, ok := <-ch:
				if !ok {
					return
				}
				// execute with a retry helper for SQLITE_BUSY
				res, err := ExecWithRetry(db, wr.Query, wr.Args...)
				if wr.ErrCh != nil {
					wr.ErrCh <- err
				}
				if err == nil && res != nil {
					// if this was an insert into payments, broadcast the created payment
					q := strings.ToLower(strings.TrimSpace(wr.Query))
					if strings.HasPrefix(q, "insert into payments") {
						id, err := res.LastInsertId()
						if err == nil {
							// fetch the inserted row
							var name string
							var link sql.NullString
							var email sql.NullString
							var createdAt string
							var amount int64
							row := db.QueryRow("SELECT name, link, email, amount_cents, created_at FROM payments WHERE id = ?", id)
							if err := row.Scan(&name, &link, &email, &amount, &createdAt); err != nil {
								log.Printf("failed to load inserted payment id=%d for broadcast: %v", id, err)
								break
							}

							payload := map[string]interface{}{
								// keep backward compatibility: some clients may expect either key
								"type":  "payment_created",
								"event": "payment.created",
								"payment": map[string]interface{}{
									"id":           id,
									"name":         name,
									"link":         link.String,
									"email":        email.String,
									"amount_cents": amount,
									"created_at":   createdAt,
								},
							}
							log.Printf("broadcasting payment_created id=%d name=%q amount_cents=%d", id, name, amount)
							ws.Broadcast(payload)
						}
					}
				}
			case <-stop:
				return
			}
		}
	}()

	stopper := func() {
		close(stop)
	}
	return ch, stopper
}

// ExecWithRetry executes a statement and retries when encountering a busy error.
func ExecWithRetry(db *sql.DB, query string, args ...interface{}) (sql.Result, error) {
	var err error
	var res sql.Result
	backoff := 50 * time.Millisecond
	for i := 0; i < 6; i++ { // ~6 attempts
		res, err = db.Exec(query, args...)
		if err == nil {
			return res, nil
		}
		if !isBusyErr(err) {
			return nil, err
		}
		time.Sleep(backoff)
		backoff *= 2
	}
	return nil, err
}

func isBusyErr(err error) bool {
	if err == nil {
		return false
	}
	// driver errors typically include "BUSY" or "busy"
	if errors.Is(err, sql.ErrConnDone) {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "busy") || strings.Contains(s, "database is locked")
}
