package config

import (
	"os"
	"strings"
)

const (
	defaultHTTPAddr            = ":8080"
	defaultBrokerProvider      = "redpanda"
	defaultBrokerBrokers       = "redpanda:9092"
	defaultEnvironment         = "local"
	defaultDatabaseURL         = ""
	defaultTemporalDatabaseURL = ""
	defaultAuthEnabled         = "false"
	defaultAuthIssuer          = ""
	defaultAuthRealm           = ""
	defaultAuthJWKSURL         = ""
	defaultAuthAudience        = ""
	defaultAuthClientID        = ""
)

// Config contains process-level settings for SignalOps services.
type Config struct {
	HTTPAddr            string
	BrokerProvider      string
	BrokerBrokers       string
	Environment         string
	DatabaseURL         string
	TemporalDatabaseURL string
	AuthEnabled         bool
	AuthIssuer          string
	AuthRealm           string
	AuthJWKSURL         string
	AuthAudience        string
	AuthClientID        string
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		HTTPAddr:            envOrDefault("SIGNALOPS_HTTP_ADDR", defaultHTTPAddr),
		BrokerProvider:      envOrDefault("SIGNALOPS_BROKER_PROVIDER", defaultBrokerProvider),
		BrokerBrokers:       envOrDefault("SIGNALOPS_BROKER_BROKERS", defaultBrokerBrokers),
		Environment:         envOrDefault("SIGNALOPS_ENV", defaultEnvironment),
		DatabaseURL:         envOrDefault("SIGNALOPS_DATABASE_URL", defaultDatabaseURL),
		TemporalDatabaseURL: envOrDefault("SIGNALOPS_TEMPORAL_DATABASE_URL", defaultTemporalDatabaseURL),
		AuthEnabled:         envBool("SIGNALOPS_AUTH_ENABLED", defaultAuthEnabled),
		AuthIssuer:          envOrDefault("SIGNALOPS_AUTH_ISSUER", defaultAuthIssuer),
		AuthRealm:           envOrDefault("SIGNALOPS_AUTH_REALM", defaultAuthRealm),
		AuthJWKSURL:         envOrDefault("SIGNALOPS_AUTH_JWKS_URL", defaultAuthJWKSURL),
		AuthAudience:        envOrDefault("SIGNALOPS_AUTH_AUDIENCE", defaultAuthAudience),
		AuthClientID:        envOrDefault("SIGNALOPS_AUTH_CLIENT_ID", defaultAuthClientID),
	}
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envBool(key, fallback string) bool {
	value := strings.ToLower(strings.TrimSpace(envOrDefault(key, fallback)))
	return value == "1" || value == "true" || value == "yes" || value == "on"
}
