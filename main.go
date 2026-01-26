package main

import (
	"log"
	"net/http"

	"github.com/lucaskampi/paywall-api/handlers"
)

func main() {
	http.HandleFunc("/health", handlers.Health)
	http.HandleFunc("/leaderboard", handlers.Leaderboard)
	http.HandleFunc("/pay", handlers.Pay)
	http.HandleFunc("/webhook", handlers.Webhook)
	http.HandleFunc("/total", handlers.Total)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// root handler keeps a simple hello response
func init() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, Paywall API!"))
	})
}
