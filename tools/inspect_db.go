//go:build tools
// +build tools

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/lucaskampi/paywall-api/db"
)

func main() {
	dbConn, err := db.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer dbConn.Close()

	// list tables
	fmt.Println("Tables:")
	rows, err := dbConn.Query("SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' ORDER BY table_name")
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
	pr, err := dbConn.Query("SELECT id, name, link, amount_cents, created_at FROM payments ORDER BY created_at DESC LIMIT 10")
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
		var created time.Time
		if err := pr.Scan(&id, &name, &link, &amount, &created); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  %d | %s | %s | %d | %s\n", id, name, link, amount, created.UTC().Format(time.RFC3339))
	}
}
