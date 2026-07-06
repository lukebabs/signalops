package config

import "os"

const defaultHTTPAddr = ":8080"

// Config contains process-level settings for SignalOps services.
type Config struct {
	HTTPAddr string
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		HTTPAddr: envOrDefault("SIGNALOPS_HTTP_ADDR", defaultHTTPAddr),
	}
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

