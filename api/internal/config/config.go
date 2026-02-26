package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL        string
	JWTSecret          string
	Port               string
	GitHubClientID     string
	GitHubClientSecret string
	GitHubCallbackURL  string
}

func Load() *Config {
	// Best-effort: load .env from project root (api/../.env) and api/.env
	_ = godotenv.Load("../.env", ".env")

	return &Config{
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://localhost:5432/devpulse_dev?sslmode=disable"),
		JWTSecret:          getEnv("JWT_SECRET", "devpulse-dev-secret-change-me"),
		Port:               getEnv("PORT", "8080"),
		GitHubClientID:     getEnv("GITHUB_CLIENT_ID", ""),
		GitHubClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
		GitHubCallbackURL:  getEnv("GITHUB_CALLBACK_URL", "http://localhost:3000/auth/github/callback"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
