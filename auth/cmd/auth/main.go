package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/braccet/auth/internal/api"
	"github.com/braccet/auth/internal/config"
	"github.com/braccet/auth/internal/repository"
	"github.com/braccet/auth/internal/service"
)

func main() {
	// Load database config and connect
	dbConfig := config.LoadDatabaseConfig()
	db, err := config.NewDatabaseConnection(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	pendingRepo := repository.NewPendingRegistrationRepository(db)

	// Initialize email sender
	emailSender := initEmailSender()

	// Get base URL for verification links
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8081"
	}

	// Start cleanup goroutine for expired pending registrations
	go cleanupExpiredRegistrations(pendingRepo)

	// Create router
	router := api.NewRouter(userRepo, pendingRepo, emailSender, baseURL)

	// Get port from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Auth service starting on port %s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// initEmailSender creates the appropriate email sender based on configuration
func initEmailSender() service.EmailSender {
	provider := os.Getenv("EMAIL_PROVIDER")

	switch provider {
	case "sendgrid":
		cfg := service.EmailConfig{
			Provider:  "sendgrid",
			APIKey:    os.Getenv("SENDGRID_API_KEY"),
			FromEmail: getEnv("EMAIL_FROM_ADDRESS", "noreply@braccet.com"),
			FromName:  getEnv("EMAIL_FROM_NAME", "Braccet"),
		}
		return service.NewSendGridEmailSender(cfg)
	default:
		// Use console sender for development
		log.Println("Using console email sender (set EMAIL_PROVIDER=sendgrid for production)")
		return service.NewConsoleEmailSender()
	}
}

// cleanupExpiredRegistrations periodically removes expired pending registrations
func cleanupExpiredRegistrations(repo repository.PendingRegistrationRepository) {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		count, err := repo.DeleteExpired(context.Background())
		if err != nil {
			log.Printf("Failed to cleanup expired registrations: %v", err)
		} else if count > 0 {
			log.Printf("Cleaned up %d expired pending registrations", count)
		}
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
