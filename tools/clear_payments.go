//go:build tools
// +build tools

package main

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "file:./data/paywall.db?_busy_timeout=5000")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec("DELETE FROM payments"); err != nil {
		log.Fatal(err)
	}
	if _, err := db.Exec("DELETE FROM sqlite_sequence WHERE name='payments'"); err != nil {
		log.Fatal(err)
	}
	if _, err := db.Exec("VACUUM"); err != nil {
		log.Fatal(err)
	}

	log.Println("payments cleared")
}
