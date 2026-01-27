package db

import (
    "database/sql"
    "errors"
    "strings"
    "time"
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
                err := ExecWithRetry(db, wr.Query, wr.Args...)
                if wr.ErrCh != nil {
                    wr.ErrCh <- err
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
func ExecWithRetry(db *sql.DB, query string, args ...interface{}) error {
    var err error
    backoff := 50 * time.Millisecond
    for i := 0; i < 6; i++ { // ~6 attempts
        _, err = db.Exec(query, args...)
        if err == nil {
            return nil
        }
        if !isBusyErr(err) {
            return err
        }
        time.Sleep(backoff)
        backoff *= 2
    }
    return err
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
