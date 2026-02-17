package api

import (
	"net/http"

	"github.com/braccet/auth/internal/api/handlers"
	"github.com/braccet/auth/internal/repository"
	"github.com/braccet/auth/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(
	userRepo repository.UserRepository,
	pendingRepo repository.PendingRegistrationRepository,
	emailSender service.EmailSender,
	baseURL string,
) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// Auth handlers
	authHandler := handlers.NewAuthHandler(userRepo, pendingRepo, emailSender, baseURL)

	r.Route("/auth", func(r chi.Router) {
		r.Post("/signup", authHandler.Signup)
		r.Post("/login", authHandler.Login)
		r.Get("/verify-email", authHandler.VerifyEmail)   // GET for email links
		r.Post("/verify-email", authHandler.VerifyEmail)  // POST for API calls
		r.Post("/resend-verification", authHandler.ResendVerification)
	})

	return r
}
