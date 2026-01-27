package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/lucaskampi/paywall-api/db"
	"github.com/lucaskampi/paywall-api/handlers"
)

func main() {
	// connect to DB
	dbConn, err := db.Connect()
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer dbConn.Close()

	// run migrations
	if err := db.RunMigrations(dbConn, "./migrations"); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	// start single-writer
	writeCh, stopWriter := db.StartWriter(dbConn)
	defer stopWriter()

	// initialize handlers with DB writer
	handlers.Init(dbConn, writeCh)

	// register routes
	http.HandleFunc("/health", handlers.Health)
	http.HandleFunc("/leaderboard", handlers.Leaderboard)
	http.HandleFunc("/pay", handlers.Pay)
	http.HandleFunc("/webhook", handlers.Webhook)
	http.HandleFunc("/total", handlers.Total)

	// root handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, Paywall API!"))
	})

	srv := &http.Server{Addr: ":8080"}

	go func() {
		log.Println("Server starting on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	// wait for interrupt and shutdown gracefully
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown: %v", err)
	}
	log.Println("server stopped")
}
