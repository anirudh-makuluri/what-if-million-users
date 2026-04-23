package main

import (
	"log"
	"net/http"
	"os"

	"github.com/anirudh-makuluri/what-if-million-users/ticketmaster/internal/handler"
	"github.com/anirudh-makuluri/what-if-million-users/ticketmaster/internal/store"
)

func main() {
	postgresURL := os.Getenv("DATABASE_URL")
	if postgresURL == "" {
		postgresURL = "postgres://postgres:postgres@localhost:5432/ticketmaster?sslmode=disable"
	}

	postgresStore, err := store.NewPostgresStore(postgresURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer postgresStore.Close()

	if err := postgresStore.InitSchema(); err != nil {
		log.Fatalf("Unable to initialize schema: %v", err)
	}

	h := handler.NewHandler(postgresStore)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	log.Printf("ticketmaster server starting on %s", addr)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}
