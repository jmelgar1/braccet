package main

import (
	"log"
	"net/http"
	"os"

	"github.com/braccet/tournament/internal/api"
	"github.com/braccet/tournament/internal/client"
	"github.com/braccet/tournament/internal/config"
	"github.com/braccet/tournament/internal/repository"
)

func main() {
	// Load database config and connect
	dbConfig := config.LoadDatabaseConfig()
	db, err := config.NewDatabaseConnection(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize repositories
	tournamentRepo := repository.NewTournamentRepository(db)
	participantRepo := repository.NewParticipantRepository(db)

	// Initialize bracket service client
	bracketServiceURL := os.Getenv("BRACKET_SERVICE_URL")
	if bracketServiceURL == "" {
		bracketServiceURL = "http://localhost:8082"
	}
	bracketClient := client.NewBracketClient(bracketServiceURL)

	// Create router
	router := api.NewRouter(tournamentRepo, participantRepo, bracketClient)

	// Get port from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	log.Printf("Tournament service starting on port %s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
