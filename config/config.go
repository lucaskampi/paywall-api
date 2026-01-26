package config

import "os"

// Config holds runtime configuration loaded from environment.
type Config struct {
	DatabaseURL string
	StripeKey   string
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		StripeKey:   os.Getenv("STRIPE_KEY"),
	}
}
