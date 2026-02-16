package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/braccet/bracket/internal/api/handlers"
	"github.com/braccet/bracket/internal/repository"
	"github.com/braccet/bracket/internal/service"
)

func NewRouter(repo repository.MatchRepository) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.SetHeader("Content-Type", "application/json"))

	// Create services
	bracketSvc := service.NewBracketService(repo)
	matchSvc := service.NewMatchService(repo)

	// Create handlers
	bracketHandler := handlers.NewBracketHandler(bracketSvc, matchSvc, repo)
	matchHandler := handlers.NewMatchHandler(matchSvc, repo)

	// Health check
	r.Get("/health", handlers.Health)

	// Bracket routes
	r.Post("/brackets", bracketHandler.Generate)
	r.Get("/brackets/{tournamentId}", bracketHandler.GetState)
	r.Get("/brackets/{tournamentId}/matches", bracketHandler.ListMatches)

	// Match routes
	r.Get("/matches/{id}", matchHandler.Get)
	r.Post("/matches/{id}/result", matchHandler.ReportResult)
	r.Post("/matches/{id}/start", matchHandler.Start)

	return r
}
