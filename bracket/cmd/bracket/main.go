package main

import (
	"log"
	"net/http"
	"os"

	"github.com/braccet/bracket/internal/api"
	"github.com/braccet/bracket/internal/config"
	"github.com/braccet/bracket/internal/repository"
)

func main() {
	// Load database configuration
	cfg := config.LoadDatabaseConfig()
	db, err := config.NewDatabaseConnection(cfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create repositories
	repo := repository.NewMatchRepository(db)
	setRepo := repository.NewSetRepository(db)

	// Create router
	router := api.NewRouter(repo, setRepo)

	// Get port from environment
	port := os.Getenv("SERVICE_PORT")
	if port == "" {
		port = "8082"
	}

	log.Printf("Bracket service starting on port %s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
