package api

import (
	"net/http"

	"github.com/braccet/tournament/internal/api/handlers"
	"github.com/braccet/tournament/internal/api/middleware"
	"github.com/braccet/tournament/internal/repository"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func NewRouter(tournamentRepo repository.TournamentRepository) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:4200", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// Tournament handlers
	tournamentHandler := handlers.NewTournamentHandler(tournamentRepo)

	r.Route("/tournaments", func(r chi.Router) {
		r.Use(middleware.Auth)

		r.Get("/", tournamentHandler.List)
		r.Post("/", tournamentHandler.Create)
		r.Get("/{id}", tournamentHandler.Get)
		r.Put("/{id}", tournamentHandler.Update)
		r.Delete("/{id}", tournamentHandler.Delete)
	})

	return r
}
