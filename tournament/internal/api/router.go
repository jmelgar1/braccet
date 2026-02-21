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

func NewRouter(tournamentRepo repository.TournamentRepository, participantRepo repository.ParticipantRepository, bracketClient client.BracketClient) *chi.Mux {
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
	tournamentHandler := handlers.NewTournamentHandler(tournamentRepo)
	participantHandler := handlers.NewParticipantHandler(participantRepo, tournamentRepo, bracketClient)

	r.Route("/tournaments", func(r chi.Router) {
		r.Use(middleware.Auth)

		r.Get("/", tournamentHandler.List)
		r.Post("/", tournamentHandler.Create)
		r.Get("/{slug}", tournamentHandler.Get)
		r.Put("/{slug}", tournamentHandler.Update)
		r.Delete("/{slug}", tournamentHandler.Delete)

		// Participant routes (nested under tournament)
		r.Route("/{slug}/participants", func(r chi.Router) {
			r.Get("/", participantHandler.List)
			r.Post("/", participantHandler.Add)
			r.Delete("/{participantId}", participantHandler.Remove)
			r.Post("/{participantId}/withdraw", participantHandler.Withdraw)
			r.Put("/seeding", participantHandler.UpdateSeeding)
		})
	})

	return r
}
