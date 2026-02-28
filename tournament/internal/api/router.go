package api

import (
	"net/http"

	"github.com/braccet/tournament/internal/api/handlers"
	"github.com/braccet/tournament/internal/api/middleware"
	"github.com/braccet/tournament/internal/client"
	"github.com/braccet/tournament/internal/repository"
	"github.com/braccet/tournament/internal/service"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

func NewRouter(tournamentRepo repository.TournamentRepository, participantRepo repository.ParticipantRepository, stageRepo repository.StageRepository, eventRepo repository.EventRepository, bracketClient client.BracketClient, communityClient client.CommunityClient, storageService service.StorageService) *chi.Mux {
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
	participantHandler := handlers.NewParticipantHandler(participantRepo, tournamentRepo, stageRepo, eventRepo, bracketClient, communityClient)
	stageHandler := handlers.NewStageHandler(tournamentRepo, participantRepo, stageRepo, bracketClient, communityClient)
	eventHandler := handlers.NewEventHandler(eventRepo, tournamentRepo, participantRepo, communityClient)
	uploadHandler := handlers.NewUploadHandler(storageService, eventRepo, tournamentRepo)

	// Internal routes (service-to-service, no auth required)
	r.Route("/internal/tournaments", func(r chi.Router) {
		r.Get("/{id}", tournamentHandler.GetByID)
		r.Post("/{id}/advance", eventHandler.TriggerAdvancementInternal)
	})

	r.Route("/internal/participants", func(r chi.Router) {
		r.Get("/{id}", participantHandler.GetByID)
	})

	r.Route("/internal/communities/{communityId}/tournaments", func(r chi.Router) {
		r.Get("/", tournamentHandler.ListByCommunity)
	})

	// Internal stage routes (for bracket service)
	r.Route("/internal/stages/{stageId}", func(r chi.Router) {
		r.Get("/groups", stageHandler.GetStageGroupsInternal)
		r.Get("/participants", stageHandler.GetStageParticipantsInternal)
		r.Put("/assignments", stageHandler.UpdateStageAssignmentsInternal)
	})

	// Event routes
	r.Route("/events", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth)

			r.Get("/", eventHandler.List)
			r.Post("/", eventHandler.Create)
			r.Get("/{slug}", eventHandler.Get)
			r.Put("/{slug}", eventHandler.Update)
			r.Delete("/{slug}", eventHandler.Delete)
			r.Get("/{slug}/qualification-status", eventHandler.GetQualificationStatus)
			r.Get("/{slug}/available-tournaments", eventHandler.ListAvailableTournaments)

			// Event tournament management
			r.Post("/{slug}/tournaments", eventHandler.AddTournament)
			r.Put("/{slug}/tournaments/{tournamentId}", eventHandler.UpdateTournament)
			r.Delete("/{slug}/tournaments/{tournamentId}", eventHandler.RemoveTournament)
			r.Post("/{slug}/tournaments/{tournamentId}/advance", eventHandler.TriggerAdvancement)

			// DAG editor endpoints
			r.Get("/{slug}/graph", eventHandler.GetGraph)
			r.Put("/{slug}/connections", eventHandler.UpdateConnections)

			// Event logo upload
			r.Post("/{slug}/logo/upload-url", uploadHandler.GetEventLogoUploadURL)
			r.Put("/{slug}/logo", uploadHandler.UpdateEventLogoURL)
		})
	})

	// Community events (public)
	r.Get("/communities/{communityId}/events", eventHandler.ListByCommunity)

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
			r.Get("/{slug}/prize-tiers", tournamentHandler.GetSuggestedPrizeTiers)

			// Tournament logo upload
			r.Post("/{slug}/logo/upload-url", uploadHandler.GetTournamentLogoUploadURL)
			r.Put("/{slug}/logo", uploadHandler.UpdateTournamentLogoURL)

			// Participant routes (nested under tournament)
			r.Route("/{slug}/participants", func(r chi.Router) {
				r.Get("/", participantHandler.List)
				r.Get("/search", participantHandler.SearchAvailableMembers)
				r.Get("/invite-by-region", participantHandler.GetTopMembersForRegionInvite)
				r.Post("/", participantHandler.Add)
				r.Delete("/{participantId}", participantHandler.Remove)
				r.Post("/{participantId}/withdraw", participantHandler.Withdraw)
				r.Post("/{participantId}/promote", participantHandler.Promote)
				r.Put("/seeding", participantHandler.UpdateSeeding)
			})

			// Stage routes (for multi-stage tournaments)
			r.Get("/{slug}/stages", stageHandler.GetStages)
			r.Post("/{slug}/stages", stageHandler.AddStage)
			r.Put("/{slug}/stages/{stageId}", stageHandler.UpdateStage)
			r.Delete("/{slug}/stages/{stageId}", stageHandler.DeleteStage)
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
