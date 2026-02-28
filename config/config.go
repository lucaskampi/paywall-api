package config

import "os"

// Config holds runtime configuration loaded from environment.
type Config struct {
	DatabaseURL             string
	AbacatePayAPIKey        string
	AbacatePayBaseURL       string
	AbacatePayWebhookSecret string
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		DatabaseURL:             os.Getenv("DATABASE_URL"),
		AbacatePayAPIKey:        os.Getenv("ABACATEPAY_API_KEY"),
		AbacatePayBaseURL:       os.Getenv("ABACATEPAY_BASE_URL"),
		AbacatePayWebhookSecret: os.Getenv("ABACATEPAY_WEBHOOK_SECRET"),
	}
}
