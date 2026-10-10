package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("SIGNALOPS_HTTP_ADDR", "")
	t.Setenv("SIGNALOPS_BROKER_PROVIDER", "")
	t.Setenv("SIGNALOPS_BROKER_BROKERS", "")
	t.Setenv("SIGNALOPS_ENV", "")
	t.Setenv("SIGNALOPS_DATABASE_URL", "")
	t.Setenv("SIGNALOPS_TEMPORAL_DATABASE_URL", "")
	t.Setenv("SIGNALOPS_AUTH_ENABLED", "")
	t.Setenv("SIGNALOPS_AUTH_ISSUER", "")
	t.Setenv("SIGNALOPS_AUTH_REALM", "")
	t.Setenv("SIGNALOPS_AUTH_JWKS_URL", "")
	t.Setenv("SIGNALOPS_AUTH_AUDIENCE", "")
	t.Setenv("SIGNALOPS_AUTH_CLIENT_ID", "")

	cfg := Load()

	if cfg.HTTPAddr != defaultHTTPAddr {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, defaultHTTPAddr)
	}
	if cfg.BrokerProvider != defaultBrokerProvider {
		t.Fatalf("BrokerProvider = %q, want %q", cfg.BrokerProvider, defaultBrokerProvider)
	}
	if cfg.BrokerBrokers != defaultBrokerBrokers {
		t.Fatalf("BrokerBrokers = %q, want %q", cfg.BrokerBrokers, defaultBrokerBrokers)
	}
	if cfg.Environment != defaultEnvironment {
		t.Fatalf("Environment = %q, want %q", cfg.Environment, defaultEnvironment)
	}
	if cfg.DatabaseURL != defaultDatabaseURL {
		t.Fatalf("DatabaseURL = %q, want %q", cfg.DatabaseURL, defaultDatabaseURL)
	}
	if cfg.TemporalDatabaseURL != defaultTemporalDatabaseURL {
		t.Fatalf("TemporalDatabaseURL = %q, want %q", cfg.TemporalDatabaseURL, defaultTemporalDatabaseURL)
	}
	if cfg.AuthEnabled {
		t.Fatal("AuthEnabled = true, want false")
	}
	if cfg.AuthIssuer != defaultAuthIssuer || cfg.AuthRealm != defaultAuthRealm || cfg.AuthJWKSURL != defaultAuthJWKSURL || cfg.AuthAudience != defaultAuthAudience || cfg.AuthClientID != defaultAuthClientID {
		t.Fatalf("auth defaults = %+v", cfg)
	}
}

func TestLoadEnvironment(t *testing.T) {
	t.Setenv("SIGNALOPS_HTTP_ADDR", ":9000")
	t.Setenv("SIGNALOPS_BROKER_PROVIDER", "kafka")
	t.Setenv("SIGNALOPS_BROKER_BROKERS", "localhost:19092")
	t.Setenv("SIGNALOPS_ENV", "test")
	t.Setenv("SIGNALOPS_DATABASE_URL", "postgres://example")
	t.Setenv("SIGNALOPS_TEMPORAL_DATABASE_URL", "postgres://temporal")
	t.Setenv("SIGNALOPS_AUTH_ENABLED", "true")
	t.Setenv("SIGNALOPS_AUTH_ISSUER", "https://auth.syncratic.co/realms/syncratic")
	t.Setenv("SIGNALOPS_AUTH_REALM", "syncratic")
	t.Setenv("SIGNALOPS_AUTH_JWKS_URL", "https://auth.syncratic.co/realms/syncratic/protocol/openid-connect/certs")
	t.Setenv("SIGNALOPS_AUTH_AUDIENCE", "signalops-api")
	t.Setenv("SIGNALOPS_AUTH_CLIENT_ID", "signalops-web")

	cfg := Load()

	if cfg.HTTPAddr != ":9000" {
		t.Fatalf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.BrokerProvider != "kafka" {
		t.Fatalf("BrokerProvider = %q", cfg.BrokerProvider)
	}
	if cfg.BrokerBrokers != "localhost:19092" {
		t.Fatalf("BrokerBrokers = %q", cfg.BrokerBrokers)
	}
	if cfg.Environment != "test" {
		t.Fatalf("Environment = %q", cfg.Environment)
	}
	if cfg.DatabaseURL != "postgres://example" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.TemporalDatabaseURL != "postgres://temporal" {
		t.Fatalf("TemporalDatabaseURL = %q", cfg.TemporalDatabaseURL)
	}
	if !cfg.AuthEnabled {
		t.Fatal("AuthEnabled = false, want true")
	}
	if cfg.AuthIssuer != "https://auth.syncratic.co/realms/syncratic" || cfg.AuthRealm != "syncratic" || cfg.AuthJWKSURL != "https://auth.syncratic.co/realms/syncratic/protocol/openid-connect/certs" || cfg.AuthAudience != "signalops-api" || cfg.AuthClientID != "signalops-web" {
		t.Fatalf("auth env = %+v", cfg)
	}
}

func TestValidateAuthConfiguration(t *testing.T) {
	valid := Config{
		AuthEnabled:  true,
		AuthIssuer:   "https://auth.syncratic.co/realms/syncratic",
		AuthJWKSURL:  "https://auth.syncratic.co/realms/syncratic/protocol/openid-connect/certs",
		AuthAudience: "signalops-api",
	}
	if err := valid.ValidateAuthConfiguration(); err != nil {
		t.Fatalf("valid auth configuration: %v", err)
	}

	for _, testCase := range []struct {
		name string
		cfg  Config
	}{
		{name: "issuer missing", cfg: Config{AuthEnabled: true, AuthJWKSURL: valid.AuthJWKSURL, AuthAudience: valid.AuthAudience}},
		{name: "jwks missing", cfg: Config{AuthEnabled: true, AuthIssuer: valid.AuthIssuer, AuthAudience: valid.AuthAudience}},
		{name: "audience missing", cfg: Config{AuthEnabled: true, AuthIssuer: valid.AuthIssuer, AuthJWKSURL: valid.AuthJWKSURL}},
		{name: "issuer malformed", cfg: Config{AuthEnabled: true, AuthIssuer: "not-a-url", AuthJWKSURL: valid.AuthJWKSURL, AuthAudience: valid.AuthAudience}},
		{name: "jwks malformed", cfg: Config{AuthEnabled: true, AuthIssuer: valid.AuthIssuer, AuthJWKSURL: "ftp://auth.syncratic.co/certs", AuthAudience: valid.AuthAudience}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if err := testCase.cfg.ValidateAuthConfiguration(); err == nil {
				t.Fatal("expected authentication configuration error")
			}
		})
	}

	if err := (Config{}).ValidateAuthConfiguration(); err != nil {
		t.Fatalf("disabled authentication configuration: %v", err)
	}
}

func TestValidateMarketOpsDataBoundary(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want bool
	}{
		{name: "pre-cutover shared topology is allowed", cfg: Config{}, want: false},
		{name: "complete dedicated topology is allowed", cfg: Config{MarketOpsDatabaseURL: "postgres://primary", MarketOpsTemporalDatabaseURL: "postgres://temporal"}, want: false},
		{name: "primary without temporal is rejected", cfg: Config{MarketOpsDatabaseURL: "postgres://primary"}, want: true},
		{name: "temporal without primary is rejected", cfg: Config{MarketOpsTemporalDatabaseURL: "postgres://temporal"}, want: true},
		{name: "authoritative boundary cannot fall back to shared", cfg: Config{MarketOpsDataBoundaryRequired: true}, want: true},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := testCase.cfg.ValidateMarketOpsDataBoundary()
			if (got != nil) != testCase.want {
				t.Fatalf("ValidateMarketOpsDataBoundary() error = %v, want error=%t", got, testCase.want)
			}
		})
	}
}
