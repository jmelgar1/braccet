package config

import (
	"os"
)

type Config struct {
	Port            string
	AuthServiceURL       string
	TournamentServiceURL string
	BracketServiceURL    string
}

func Load() Config {
	return Config{
		Port:                 getEnv("SERVICE_PORT", "8080"),
		AuthServiceURL:       getEnv("AUTH_SERVICE_URL", "http://localhost:8081"),
		TournamentServiceURL: getEnv("TOURNAMENT_SERVICE_URL", "http://localhost:8083"),
		BracketServiceURL:    getEnv("BRACKET_SERVICE_URL", "http://localhost:8082"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
