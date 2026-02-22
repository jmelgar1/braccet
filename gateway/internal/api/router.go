package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/braccet/gateway/internal/api/handlers"
	"github.com/braccet/gateway/internal/config"
	"github.com/braccet/gateway/internal/proxy"
)

func NewRouter(cfg config.Config) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:4200", "http://localhost:*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Gateway health check
	r.Get("/health", handlers.Health)

	// Auth service routes
	r.Route("/api/auth", func(r chi.Router) {
		authProxy := proxy.NewServiceProxy(cfg.AuthServiceURL, "/api")
		r.HandleFunc("/", authProxy)
		r.HandleFunc("/*", authProxy)
	})

	// Tournament service routes
	r.Route("/api/tournaments", func(r chi.Router) {
		tournamentProxy := proxy.NewServiceProxy(cfg.TournamentServiceURL, "/api")
		r.HandleFunc("/", tournamentProxy)
		r.HandleFunc("/*", tournamentProxy)
	})

	// Bracket service routes (includes /api/brackets/matches/*)
	r.Route("/api/brackets", func(r chi.Router) {
		bracketProxy := proxy.NewServiceProxy(cfg.BracketServiceURL, "/api")
		r.HandleFunc("/", bracketProxy)  // Handles /api/brackets
		r.HandleFunc("/*", bracketProxy) // Handles /api/brackets/*
	})

	// Community service routes
	r.Route("/api/communities", func(r chi.Router) {
		communityProxy := proxy.NewServiceProxy(cfg.CommunityServiceURL, "/api")
		r.HandleFunc("/", communityProxy)  // Handles /api/communities
		r.HandleFunc("/*", communityProxy) // Handles /api/communities/*
	})

	return r
}
