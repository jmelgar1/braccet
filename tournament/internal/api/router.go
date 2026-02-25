package api

import (
	"net/http"

	"github.com/braccet/tournament/internal/api/handlers"
	"github.com/braccet/tournament/internal/api/middleware"
	"github.com/braccet/tournament/internal/client"
	"github.com/braccet/tournament/internal/repository"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

func NewRouter(tournamentRepo repository.TournamentRepository, participantRepo repository.ParticipantRepository, stageRepo repository.StageRepository, bracketClient client.BracketClient, communityClient client.CommunityClient) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	// Note: CORS is handled by the gateway, not here

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// Tournament handlers
	tournamentHandler := handlers.NewTournamentHandler(tournamentRepo, participantRepo, stageRepo, bracketClient, communityClient)
	participantHandler := handlers.NewParticipantHandler(participantRepo, tournamentRepo, bracketClient, communityClient)
	stageHandler := handlers.NewStageHandler(tournamentRepo, participantRepo, stageRepo, bracketClient, communityClient)

	// Internal routes (service-to-service, no auth required)
	r.Route("/internal/tournaments", func(r chi.Router) {
		r.Get("/{id}", tournamentHandler.GetByID)
	})

	r.Route("/internal/participants", func(r chi.Router) {
		r.Get("/{id}", participantHandler.GetByID)
	})

	r.Route("/internal/communities/{communityId}/tournaments", func(r chi.Router) {
		r.Get("/", tournamentHandler.ListByCommunity)
	})

	r.Route("/tournaments", func(r chi.Router) {
		// Public route - no auth required for listing community tournaments
		r.Get("/community/{communityId}", tournamentHandler.ListByCommunity)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth)

			r.Get("/", tournamentHandler.List)
			r.Post("/", tournamentHandler.Create)
			r.Post("/multi-stage", stageHandler.CreateMultiStage)
			r.Get("/{slug}", tournamentHandler.Get)
			r.Put("/{slug}", tournamentHandler.Update)
			r.Delete("/{slug}", tournamentHandler.Delete)

			// Participant routes (nested under tournament)
			r.Route("/{slug}/participants", func(r chi.Router) {
				r.Get("/", participantHandler.List)
				r.Get("/search", participantHandler.SearchAvailableMembers)
				r.Post("/", participantHandler.Add)
				r.Delete("/{participantId}", participantHandler.Remove)
				r.Post("/{participantId}/withdraw", participantHandler.Withdraw)
				r.Post("/{participantId}/promote", participantHandler.Promote)
				r.Put("/seeding", participantHandler.UpdateSeeding)
			})

			// Stage routes (for multi-stage tournaments)
			r.Get("/{slug}/stages", stageHandler.GetStages)
			r.Put("/{slug}/stages/{stageId}", stageHandler.UpdateStage)
			r.Post("/{slug}/stages/start", stageHandler.StartStage)
			r.Post("/{slug}/stages/advance", stageHandler.AdvanceStage)
			r.Get("/{slug}/stages/{stageId}/groups", stageHandler.GetGroups)
			r.Get("/{slug}/stages/{stageId}/seeds", stageHandler.GetStageSeeds)
			r.Put("/{slug}/stages/{stageId}/seeds", stageHandler.UpdateStageSeeds)

			// Stage participant pool routes (for assigning participants to starting stages)
			r.Get("/{slug}/stages/pool", stageHandler.GetStagePool)
			r.Put("/{slug}/stages/pool", stageHandler.UpdateStagePool)
			r.Delete("/{slug}/stages/pool", stageHandler.ClearStagePool)
		})
	})

	return r
}
