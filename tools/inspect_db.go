package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "file:./data/paywall.db?_busy_timeout=5000&_foreign_keys=1"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// list tables
	fmt.Println("Tables:")
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name;")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			log.Fatal(err)
		}
		fmt.Println(" -", name)
	}

	// show payments rows (if table exists)
	fmt.Println("\nPayments (latest 10):")
	pr, err := db.Query("SELECT id, name, link, amount_cents, created_at FROM payments ORDER BY created_at DESC LIMIT 10;")
	if err != nil {
		// if payments doesn't exist, print a note
		fmt.Println("  (payments table may not exist yet)", err)
		return
	}
	defer pr.Close()
	for pr.Next() {
		var id int64
		var name, link string
		var amount int64
		var created string
		if err := pr.Scan(&id, &name, &link, &amount, &created); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  %d | %s | %s | %d | %s\n", id, name, link, amount, created)
	}
}
