package db

import (
    "database/sql"
    "fmt"
    "os"

    _ "modernc.org/sqlite"
)

// Connect opens a SQLite database using DATABASE_URL or a sensible default.
func Connect() (*sql.DB, error) {
    dsn := os.Getenv("DATABASE_URL")
    if dsn == "" {
        // file: URI form supports query params
        dsn = "file:./data/paywall.db?_busy_timeout=5000&_foreign_keys=1"
    }

    db, err := sql.Open("sqlite", dsn)
    if err != nil {
        return nil, fmt.Errorf("open sqlite: %w", err)
    }

    // SQLite performs best with a small connection pool in-process.
    db.SetMaxOpenConns(1)
    db.SetMaxIdleConns(1)

    // Apply PRAGMAs to improve integrity and concurrency.
    if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
        db.Close()
        return nil, fmt.Errorf("enable foreign_keys: %w", err)
    }
    if _, err := db.Exec("PRAGMA journal_mode = WAL;"); err != nil {
        db.Close()
        return nil, fmt.Errorf("set journal_mode WAL: %w", err)
    }
    if _, err := db.Exec("PRAGMA busy_timeout = 5000;"); err != nil {
        db.Close()
        return nil, fmt.Errorf("set busy_timeout: %w", err)
    }

    return db, nil
}
