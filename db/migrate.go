package db

import (
    "database/sql"
    "fmt"

    "github.com/golang-migrate/migrate/v4"
    "github.com/golang-migrate/migrate/v4/database/sqlite"
    _ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations runs migrations located in migrationsPath (e.g. "./migrations").
// It attempts to use golang-migrate with the sqlite driver instance.
func RunMigrations(db *sql.DB, migrationsPath string) error {
    driver, err := sqlite.WithInstance(db, &sqlite.Config{})
    if err != nil {
        return fmt.Errorf("sqlite driver instance: %w", err)
    }

    m, err := migrate.NewWithDatabaseInstance("file://"+migrationsPath, "sqlite3", driver)
    if err != nil {
        return fmt.Errorf("new migrate instance: %w", err)
    }

    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return fmt.Errorf("migrate up: %w", err)
    }
    return nil
}
