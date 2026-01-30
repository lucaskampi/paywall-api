//go:build tools
// +build tools

package main

import (
	"fmt"
	"log"

	"github.com/lucaskampi/paywall-api/db"
	_ "modernc.org/sqlite"
)

func main() {
	d, err := db.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer d.Close()

	if err := db.RunMigrations(d, "./migrations"); err != nil {
		log.Fatal(err)
	}

	writeCh, stop := db.StartWriter(d)
	defer stop()

	errCh := make(chan error)
	writeCh <- db.WriteRequest{
		Query: "INSERT INTO payments (name, link, email, amount_cents, status, currency, provider) VALUES (?, ?, ?, ?, ?, ?, ?)",
		Args:  []interface{}{"Test User", "/pay/test", "test@example.com", 1500, "paid", "usd", "manual"},
		ErrCh: errCh,
	}
	if err := <-errCh; err != nil {
		log.Fatal("write failed:", err)
	}
	fmt.Println("inserted")

	rows, err := d.Query("SELECT id, name, link, amount_cents, created_at FROM payments ORDER BY created_at DESC LIMIT 10")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var name, link string
		var amount int64
		var created string
		if err := rows.Scan(&id, &name, &link, &amount, &created); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%d | %s | %s | %d | %s\n", id, name, link, amount, created)
	}
}
