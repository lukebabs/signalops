package config

import "os"

const (
	defaultHTTPAddr       = ":8080"
	defaultBrokerProvider = "redpanda"
	defaultBrokerBrokers  = "redpanda:9092"
	defaultEnvironment    = "local"
	defaultDatabaseURL    = ""
)

// Config contains process-level settings for SignalOps services.
type Config struct {
	HTTPAddr       string
	BrokerProvider string
	BrokerBrokers  string
	Environment    string
	DatabaseURL    string
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		HTTPAddr:       envOrDefault("SIGNALOPS_HTTP_ADDR", defaultHTTPAddr),
		BrokerProvider: envOrDefault("SIGNALOPS_BROKER_PROVIDER", defaultBrokerProvider),
		BrokerBrokers:  envOrDefault("SIGNALOPS_BROKER_BROKERS", defaultBrokerBrokers),
		Environment:    envOrDefault("SIGNALOPS_ENV", defaultEnvironment),
		DatabaseURL:    envOrDefault("SIGNALOPS_DATABASE_URL", defaultDatabaseURL),
	}
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
