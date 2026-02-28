//go:build tools
// +build tools

package main

import (
	"log"

	"github.com/lucaskampi/paywall-api/db"
)

func main() {
	dbConn, err := db.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer dbConn.Close()

	if _, err := dbConn.Exec("TRUNCATE TABLE payments RESTART IDENTITY CASCADE"); err != nil {
		log.Fatal(err)
	}

	log.Println("payments cleared")
}
