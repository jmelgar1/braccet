package main

import (
	"log"
	"net/http"

	"github.com/braccet/gateway/internal/api"
	"github.com/braccet/gateway/internal/config"
)

func main() {
	cfg := config.Load()

	router := api.NewRouter(cfg)

	log.Printf("API Gateway starting on port %s", cfg.Port)
	log.Printf("  Auth service:       %s", cfg.AuthServiceURL)
	log.Printf("  Tournament service: %s", cfg.TournamentServiceURL)
	log.Printf("  Bracket service:    %s", cfg.BracketServiceURL)

	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
