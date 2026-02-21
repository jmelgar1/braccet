package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/braccet/bracket/internal/api/handlers"
	"github.com/braccet/bracket/internal/repository"
	"github.com/braccet/bracket/internal/service"
)

func NewRouter(repo repository.MatchRepository, setRepo repository.SetRepository) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	// CORS is handled by the gateway - don't add it here to avoid duplicate headers
	r.Use(middleware.SetHeader("Content-Type", "application/json"))

	// Create services
	bracketSvc := service.NewBracketService(repo)
	matchSvc := service.NewMatchService(repo, setRepo)
	forfeitSvc := service.NewForfeitService(repo)

	// Create handlers
	bracketHandler := handlers.NewBracketHandler(bracketSvc, matchSvc, repo, setRepo)
	matchHandler := handlers.NewMatchHandler(matchSvc, repo, setRepo)
	forfeitHandler := handlers.NewForfeitHandler(forfeitSvc)

	// Health check
	r.Get("/health", handlers.Health)

	// Bracket routes
	r.Post("/brackets", bracketHandler.Generate)
	r.Get("/brackets/{tournamentId}", bracketHandler.GetState)
	r.Get("/brackets/{tournamentId}/matches", bracketHandler.ListMatches)

	// Match routes (nested under /brackets)
	r.Get("/brackets/matches/{id}", matchHandler.Get)
	r.Post("/brackets/matches/{id}/result", matchHandler.ReportResult)
	r.Post("/brackets/matches/{id}/start", matchHandler.Start)

	// Forfeit route (internal, called by tournament service)
	r.Post("/brackets/forfeit-participant", forfeitHandler.ForfeitParticipant)

	return r
}
