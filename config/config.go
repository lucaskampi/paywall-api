package config

import "os"

// Config holds runtime configuration loaded from environment.
type Config struct {
	DatabaseURL         string
	StripeKey           string
	StripeWebhookSecret string
	StripeSuccessURL    string
	StripeCancelURL     string
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		StripeKey:           os.Getenv("STRIPE_KEY"),
		StripeWebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
		StripeSuccessURL:    os.Getenv("STRIPE_SUCCESS_URL"),
		StripeCancelURL:     os.Getenv("STRIPE_CANCEL_URL"),
	}
}
