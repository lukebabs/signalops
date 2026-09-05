package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

const (
	defaultHTTPAddr                          = ":8080"
	defaultBrokerProvider                    = "redpanda"
	defaultBrokerBrokers                     = "redpanda:9092"
	defaultEnvironment                       = "local"
	defaultDatabaseURL                       = ""
	defaultTemporalDatabaseURL               = ""
	defaultAuthEnabled                       = "false"
	defaultAuthIssuer                        = ""
	defaultAuthRealm                         = ""
	defaultAuthJWKSURL                       = ""
	defaultAuthAudience                      = ""
	defaultAuthClientID                      = ""
	defaultNotificationEncryptionKey         = ""
	defaultSubscriberListsEnabled            = "false"
	defaultSubscriberSubscriptionsEnabled    = "false"
	defaultSubscriberListsPilotTenants       = ""
	defaultSubscriberListsDatabaseURL        = ""
	defaultSubscriberB2CTenantID             = "tenant-local"
	defaultSubscriberB2CAutoActivateExplorer = "false"
	defaultSubscriberB2CRequireSubscription  = "true"
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
	SubscriberListsEnabled    bool
	// SubscriberSubscriptionsEnabled turns on commercial feature enforcement.
	// It is deliberately independent from the catalog/watchlist foundation so
	// the latter can remain live while subscriptions are provisioned and
	// reconciled before any customer-facing capability is restricted.
	SubscriberSubscriptionsEnabled    bool
	SubscriberListsPilotTenants       string
	SubscriberListsDatabaseURL        string
	SubscriberB2CTenantID             string
	SubscriberB2CAutoActivateExplorer bool
	SubscriberB2CRequireSubscription  bool
	StripeWebhookSecret               string
	StripeAPIKey                      string
	StripeCheckoutSuccessURL          string
	StripeCheckoutCancelURL           string
	StripePortalReturnURL             string
	MarketOpsDatabaseURL              string
	MarketOpsTemporalDatabaseURL      string
	// MarketOpsDataBoundaryRequired makes the dedicated MarketOps primary and
	// temporal stores mandatory for processes that participate in the
	// production data plane. It is rendered only in the protected cutover
	// environment; development and pre-cutover deployments remain supported.
	MarketOpsDataBoundaryRequired bool
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

// ValidateMarketOpsDataBoundary prevents a process from silently using the
// shared SignalOps stores after the MarketOps data boundary has become
// authoritative. The URLs are a pair: a primary-only or temporal-only route
// is never a valid topology.
func (c Config) ValidateMarketOpsDataBoundary() error {
	primary := strings.TrimSpace(c.MarketOpsDatabaseURL)
	temporal := strings.TrimSpace(c.MarketOpsTemporalDatabaseURL)
	if (primary == "") != (temporal == "") {
		return fmt.Errorf("SIGNALOPS_MARKETOPS_DATABASE_URL and SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL must be configured together")
	}
	if c.MarketOpsDataBoundaryRequired && primary == "" {
		return fmt.Errorf("dedicated MarketOps data boundary is required: configure SIGNALOPS_MARKETOPS_DATABASE_URL and SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL")
	}
	return nil
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		HTTPAddr:                          envOrDefault("SIGNALOPS_HTTP_ADDR", defaultHTTPAddr),
		BrokerProvider:                    envOrDefault("SIGNALOPS_BROKER_PROVIDER", defaultBrokerProvider),
		BrokerBrokers:                     envOrDefault("SIGNALOPS_BROKER_BROKERS", defaultBrokerBrokers),
		Environment:                       envOrDefault("SIGNALOPS_ENV", defaultEnvironment),
		DatabaseURL:                       envOrDefault("SIGNALOPS_DATABASE_URL", defaultDatabaseURL),
		TemporalDatabaseURL:               envOrDefault("SIGNALOPS_TEMPORAL_DATABASE_URL", defaultTemporalDatabaseURL),
		AuthEnabled:                       envBool("SIGNALOPS_AUTH_ENABLED", defaultAuthEnabled),
		AuthIssuer:                        envOrDefault("SIGNALOPS_AUTH_ISSUER", defaultAuthIssuer),
		AuthRealm:                         envOrDefault("SIGNALOPS_AUTH_REALM", defaultAuthRealm),
		AuthJWKSURL:                       envOrDefault("SIGNALOPS_AUTH_JWKS_URL", defaultAuthJWKSURL),
		AuthAudience:                      envOrDefault("SIGNALOPS_AUTH_AUDIENCE", defaultAuthAudience),
		AuthClientID:                      envOrDefault("SIGNALOPS_AUTH_CLIENT_ID", defaultAuthClientID),
		NotificationEncryptionKey:         envOrDefault("SIGNALOPS_NOTIFICATION_ENCRYPTION_KEY", defaultNotificationEncryptionKey),
		SubscriberListsEnabled:            envBool("SIGNALOPS_SUBSCRIBER_LISTS_ENABLED", defaultSubscriberListsEnabled),
		SubscriberSubscriptionsEnabled:    envBool("SIGNALOPS_SUBSCRIPTIONS_ENABLED", defaultSubscriberSubscriptionsEnabled),
		SubscriberListsPilotTenants:       envOrDefault("SIGNALOPS_SUBSCRIBER_LISTS_PILOT_TENANTS", defaultSubscriberListsPilotTenants),
		SubscriberListsDatabaseURL:        envOrDefault("SIGNALOPS_SUBSCRIBER_GATEWAY_DATABASE_URL", defaultSubscriberListsDatabaseURL),
		SubscriberB2CTenantID:             envOrDefault("SIGNALOPS_SUBSCRIBER_B2C_TENANT_ID", defaultSubscriberB2CTenantID),
		SubscriberB2CAutoActivateExplorer: envBool("SIGNALOPS_SUBSCRIBER_B2C_AUTO_ACTIVATE_EXPLORER", defaultSubscriberB2CAutoActivateExplorer),
		SubscriberB2CRequireSubscription:  envBool("SIGNALOPS_SUBSCRIBER_B2C_REQUIRE_SUBSCRIPTION", defaultSubscriberB2CRequireSubscription),
		StripeWebhookSecret:               envOrDefault("STRIPE_WEBHOOK_SECRET", ""),
		StripeAPIKey:                      envOrDefault("STRIPE_API_KEY", envOrDefault("STRIPE_RESTRICTED_API_KEY", "")),
		StripeCheckoutSuccessURL:          envOrDefault("SIGNALOPS_STRIPE_CHECKOUT_SUCCESS_URL", "https://signalops.syncratic.io/marketops/subscription/return?session_id={CHECKOUT_SESSION_ID}"),
		StripeCheckoutCancelURL:           envOrDefault("SIGNALOPS_STRIPE_CHECKOUT_CANCEL_URL", "https://signalops.syncratic.io/marketops/pricing"),
		StripePortalReturnURL:             envOrDefault("SIGNALOPS_STRIPE_PORTAL_RETURN_URL", "https://signalops.syncratic.io/marketops/pricing"),
		MarketOpsDatabaseURL:              envOrDefault("SIGNALOPS_MARKETOPS_DATABASE_URL", ""),
		MarketOpsTemporalDatabaseURL:      envOrDefault("SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL", ""),
		MarketOpsDataBoundaryRequired:     envBool("SIGNALOPS_MARKETOPS_DATA_BOUNDARY_REQUIRED", "false"),
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
