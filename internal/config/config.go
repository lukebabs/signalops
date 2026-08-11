package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

const (
	defaultHTTPAddr                  = ":8080"
	defaultBrokerProvider            = "redpanda"
	defaultBrokerBrokers             = "redpanda:9092"
	defaultEnvironment               = "local"
	defaultDatabaseURL               = ""
	defaultTemporalDatabaseURL       = ""
	defaultAuthEnabled               = "false"
	defaultAuthIssuer                = ""
	defaultAuthRealm                 = ""
	defaultAuthJWKSURL               = ""
	defaultAuthAudience              = ""
	defaultAuthClientID              = ""
	defaultNotificationEncryptionKey = ""
)

// Config contains process-level settings for SignalOps services.
type Config struct {
	HTTPAddr                  string
	BrokerProvider            string
	BrokerBrokers             string
	Environment               string
	DatabaseURL               string
	TemporalDatabaseURL       string
	AuthEnabled               bool
	AuthIssuer                string
	AuthRealm                 string
	AuthJWKSURL               string
	AuthAudience              string
	AuthClientID              string
	NotificationEncryptionKey string
}

// ValidateAuthConfiguration fails closed when JWT enforcement is enabled without
// the minimum issuer, JWKS, and audience contract required to validate tokens.
// It validates configuration shape only; deployment readiness still requires a
// live JWKS and browser-session validation.
func (c Config) ValidateAuthConfiguration() error {
	if !c.AuthEnabled {
		return nil
	}
	if strings.TrimSpace(c.AuthIssuer) == "" {
		return fmt.Errorf("SIGNALOPS_AUTH_ISSUER is required when authentication is enabled")
	}
	if strings.TrimSpace(c.AuthJWKSURL) == "" {
		return fmt.Errorf("SIGNALOPS_AUTH_JWKS_URL is required when authentication is enabled")
	}
	if strings.TrimSpace(c.AuthAudience) == "" {
		return fmt.Errorf("SIGNALOPS_AUTH_AUDIENCE is required when authentication is enabled")
	}
	for name, raw := range map[string]string{
		"SIGNALOPS_AUTH_ISSUER":   c.AuthIssuer,
		"SIGNALOPS_AUTH_JWKS_URL": c.AuthJWKSURL,
	} {
		parsed, err := url.ParseRequestURI(strings.TrimSpace(raw))
		if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
			return fmt.Errorf("%s must be an absolute HTTP(S) URL when authentication is enabled", name)
		}
	}
	return nil
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		HTTPAddr:                  envOrDefault("SIGNALOPS_HTTP_ADDR", defaultHTTPAddr),
		BrokerProvider:            envOrDefault("SIGNALOPS_BROKER_PROVIDER", defaultBrokerProvider),
		BrokerBrokers:             envOrDefault("SIGNALOPS_BROKER_BROKERS", defaultBrokerBrokers),
		Environment:               envOrDefault("SIGNALOPS_ENV", defaultEnvironment),
		DatabaseURL:               envOrDefault("SIGNALOPS_DATABASE_URL", defaultDatabaseURL),
		TemporalDatabaseURL:       envOrDefault("SIGNALOPS_TEMPORAL_DATABASE_URL", defaultTemporalDatabaseURL),
		AuthEnabled:               envBool("SIGNALOPS_AUTH_ENABLED", defaultAuthEnabled),
		AuthIssuer:                envOrDefault("SIGNALOPS_AUTH_ISSUER", defaultAuthIssuer),
		AuthRealm:                 envOrDefault("SIGNALOPS_AUTH_REALM", defaultAuthRealm),
		AuthJWKSURL:               envOrDefault("SIGNALOPS_AUTH_JWKS_URL", defaultAuthJWKSURL),
		AuthAudience:              envOrDefault("SIGNALOPS_AUTH_AUDIENCE", defaultAuthAudience),
		AuthClientID:              envOrDefault("SIGNALOPS_AUTH_CLIENT_ID", defaultAuthClientID),
		NotificationEncryptionKey: envOrDefault("SIGNALOPS_NOTIFICATION_ENCRYPTION_KEY", defaultNotificationEncryptionKey),
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
