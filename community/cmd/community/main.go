package main

import (
	"log"
	"net/http"
	"os"

	"github.com/braccet/community/internal/api"
	"github.com/braccet/community/internal/config"
	"github.com/braccet/community/internal/repository"
	"github.com/braccet/community/internal/service"
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
	communityRepo := repository.NewCommunityRepository(db)
	memberRepo := repository.NewMemberRepository(db)
	eloSystemRepo := repository.NewEloSystemRepository(db)
	memberEloRatingRepo := repository.NewMemberEloRatingRepository(db)
	eloHistoryRepo := repository.NewEloHistoryRepository(db)

	// Initialize services
	eloService := service.NewEloService(eloSystemRepo, memberEloRatingRepo, eloHistoryRepo, memberRepo)

	// Create router
	router := api.NewRouter(communityRepo, memberRepo, eloService)

	// Get port from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}

	log.Printf("Community service starting on port %s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
