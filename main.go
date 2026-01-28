package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lucaskampi/paywall-api/db"
	"github.com/lucaskampi/paywall-api/handlers"
	"github.com/lucaskampi/paywall-api/ws"
)

// corsMiddleware sets simple CORS headers allowing the frontend origin.
// Set FRONTEND_ORIGIN env var to restrict origin (e.g. http://localhost:3001).
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := os.Getenv("FRONTEND_ORIGIN")
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// connect to DB
	dbConn, err := db.Connect()
	if err != nil {
		log.Printf("db connect: %v", err)
		return
	}
	defer func() {
		if err := dbConn.Close(); err != nil {
			log.Printf("db close: %v", err)
		}
	}()

	// run migrations
	if err := db.RunMigrations(dbConn, "./migrations"); err != nil {
		log.Printf("run migrations: %v", err)
		return
	}

	// start single-writer
	writeCh, stopWriter := db.StartWriter(dbConn)
	defer stopWriter()

	// initialize handlers with DB writer
	handlers.Init(dbConn, writeCh)

	// register websocket endpoint
	http.HandleFunc("/ws", ws.ServeWS)

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

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           corsMiddleware(http.DefaultServeMux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("Server starting on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("listen: %v", err)
			os.Exit(1)
		}
	}()

	// wait for interrupt and shutdown gracefully
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown: %v", err)
	}
	log.Println("server stopped")
}
