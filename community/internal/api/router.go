package api

import (
	"net/http"

	"github.com/braccet/community/internal/api/handlers"
	"github.com/braccet/community/internal/api/middleware"
	"github.com/braccet/community/internal/repository"
	"github.com/braccet/community/internal/service"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

func NewRouter(
	communityRepo repository.CommunityRepository,
	memberRepo repository.MemberRepository,
	eloService service.EloService,
	storageService service.StorageService,
) *chi.Mux {
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

	// Initialize handlers
	communityHandler := handlers.NewCommunityHandler(communityRepo, memberRepo)
	memberHandler := handlers.NewMemberHandler(memberRepo, communityRepo)
	eloHandler := handlers.NewEloHandler(eloService, communityRepo, memberRepo)
	uploadHandler := handlers.NewUploadHandler(storageService, memberRepo, communityRepo)

	// Internal routes (service-to-service, no auth required)
	r.Route("/internal", func(r chi.Router) {
		r.Route("/communities", func(r chi.Router) {
			r.Get("/{id}", communityHandler.GetByID)
			r.Get("/{id}/members/{memberId}", memberHandler.GetByID)
			r.Post("/{id}/members", memberHandler.CreateGhostMember)
			r.Post("/{id}/members/find-or-create", memberHandler.FindOrCreateGhostMember)
			r.Post("/{id}/members/search", memberHandler.SearchMembers)
			r.Post("/{id}/members/delete", memberHandler.DeleteMembersInternal)
		})
		r.Route("/members", func(r chi.Router) {
			r.Post("/bulk-icons", memberHandler.GetBulkMemberIcons)
			r.Post("/bulk-data", memberHandler.GetBulkMemberData)
		})
		r.Route("/elo", func(r chi.Router) {
			r.Post("/process-match", eloHandler.ProcessMatch)
			r.Get("/systems/{id}", eloHandler.GetSystemByID)
			r.Post("/bulk-ratings", eloHandler.GetBulkMemberRatings)
			r.Delete("/tournament/{tournamentId}", eloHandler.RevertTournamentElo)
		})
	})

	// Internal route accessible via gateway (no auth required)
	r.Get("/communities/internal/{id}", communityHandler.GetByID)

	// Public routes (auth required)
	r.Route("/communities", func(r chi.Router) {
		r.Use(middleware.Auth)

		// Community CRUD
		r.Get("/", communityHandler.List)
		r.Post("/", communityHandler.Create)
		r.Get("/{slug}", communityHandler.Get)
		r.Put("/{slug}", communityHandler.Update)
		r.Delete("/{slug}", communityHandler.Delete)

		// Member routes (nested under community)
		r.Route("/{slug}/members", func(r chi.Router) {
			r.Get("/", memberHandler.List)
			r.Post("/", memberHandler.Add)
			r.Get("/ghosts", memberHandler.ListGhosts)
			r.Get("/{memberId}", memberHandler.Get)
			r.Put("/{memberId}", memberHandler.Update)
			r.Delete("/{memberId}", memberHandler.Remove)
			r.Put("/{memberId}/role", memberHandler.UpdateRole)
			r.Post("/{memberId}/icon/upload-url", uploadHandler.GetMemberIconUploadURL)
			r.Get("/{memberId}/elo", eloHandler.GetMemberRatings)
			r.Get("/{memberId}/elo/{systemId}/history", eloHandler.GetMemberHistory)
		})

		// Leaderboard (legacy)
		r.Get("/{slug}/leaderboard", memberHandler.Leaderboard)

		// Regions (distinct region codes used in community)
		r.Get("/{slug}/regions", memberHandler.GetRegions)

		// ELO systems routes
		r.Route("/{slug}/elo-systems", func(r chi.Router) {
			r.Get("/", eloHandler.ListSystems)
			r.Post("/", eloHandler.CreateSystem)
			r.Get("/{systemId}", eloHandler.GetSystem)
			r.Put("/{systemId}", eloHandler.UpdateSystem)
			r.Delete("/{systemId}", eloHandler.DeleteSystem)
			r.Get("/{systemId}/leaderboard", eloHandler.GetLeaderboard)
		})
	})

	return r
}
