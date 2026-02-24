package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/braccet/bracket/internal/api/handlers"
	authmw "github.com/braccet/bracket/internal/api/middleware"
	"github.com/braccet/bracket/internal/client"
	"github.com/braccet/bracket/internal/repository"
	"github.com/braccet/bracket/internal/service"
)

func NewRouter(
	repo repository.MatchRepository,
	setRepo repository.SetRepository,
	stageRepo repository.StageRepository,
	swissRepo repository.SwissRepository,
	tiebreakerRepo repository.TiebreakerRepository,
	groupRepo repository.GroupRepository,
	tournamentClient client.TournamentClient,
	communityClient client.CommunityClient,
) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	// CORS is handled by the gateway - don't add it here to avoid duplicate headers
	r.Use(middleware.SetHeader("Content-Type", "application/json"))

	// Create services
	bracketSvc := service.NewBracketServiceFull(repo, stageRepo, swissRepo, groupRepo, communityClient)
	matchSvc := service.NewMatchServiceWithTiebreakers(repo, setRepo, swissRepo, tiebreakerRepo, tournamentClient, communityClient)
	swissSvc := service.NewSwissServiceWithTiebreakers(swissRepo, repo, stageRepo, tiebreakerRepo)
	tiebreakerSvc := service.NewTiebreakerService(tiebreakerRepo, swissRepo, repo)
	forfeitSvc := service.NewForfeitService(repo)
	stageSvc := service.NewStageService(stageRepo)

	// Create handlers
	bracketHandler := handlers.NewBracketHandlerWithTiebreakers(bracketSvc, matchSvc, swissSvc, tiebreakerSvc, repo, setRepo, stageRepo)
	matchHandler := handlers.NewMatchHandler(matchSvc, repo, setRepo)
	forfeitHandler := handlers.NewForfeitHandler(forfeitSvc)
	stageHandler := handlers.NewStageHandler(stageSvc)

	// Health check
	r.Get("/health", handlers.Health)

	// Bracket routes
	r.Post("/brackets", bracketHandler.Generate)
	r.Post("/brackets/preview", bracketHandler.Preview) // Preview without persistence (same BYE logic as Generate)
	r.Post("/brackets/groups", bracketHandler.GenerateGroup) // Generate bracket for a specific group
	r.Get("/brackets/{tournamentId}", bracketHandler.GetState)
	r.Get("/brackets/{tournamentId}/matches", bracketHandler.ListMatches)
	r.Get("/brackets/{tournamentId}/stages/{stageId}/groups/{groupId}", bracketHandler.GetGroupBracket) // Get bracket for a group
	r.Delete("/brackets/{tournamentId}", bracketHandler.Delete) // Delete bracket and revert ELO (for tournament reset)

	// Standings routes (works for both Swiss and elimination formats)
	r.Get("/brackets/{tournamentId}/standings", bracketHandler.GetSwissStandings)
	r.Get("/brackets/{tournamentId}/elimination-standings", bracketHandler.GetEliminationStandings)

	// Swiss-specific routes
	r.Post("/brackets/{tournamentId}/advance-round", bracketHandler.AdvanceSwissRound)

	// Match routes (nested under /brackets)
	r.Get("/brackets/matches/{id}", matchHandler.Get)
	r.Post("/brackets/matches/{id}/result", matchHandler.ReportResult)
	r.Post("/brackets/matches/{id}/start", matchHandler.Start)

	// Protected match routes (require auth)
	r.Group(func(r chi.Router) {
		r.Use(authmw.Auth)
		r.Post("/brackets/matches/{id}/reopen", matchHandler.Reopen)
		r.Put("/brackets/matches/{id}/result", matchHandler.EditResult)
	})

	// Stage routes
	r.Get("/brackets/{tournamentId}/stages", stageHandler.GetStages)
	r.Group(func(r chi.Router) {
		r.Use(authmw.Auth)
		r.Put("/brackets/{tournamentId}/stages/{round}", stageHandler.UpdateStage)
	})

	// Forfeit route (internal, called by tournament service)
	r.Post("/brackets/forfeit-participant", forfeitHandler.ForfeitParticipant)

	return r
}
