package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("SIGNALOPS_HTTP_ADDR", "")
	t.Setenv("SIGNALOPS_BROKER_PROVIDER", "")
	t.Setenv("SIGNALOPS_BROKER_BROKERS", "")
	t.Setenv("SIGNALOPS_ENV", "")
	t.Setenv("SIGNALOPS_DATABASE_URL", "")

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
}

func TestLoadEnvironment(t *testing.T) {
	t.Setenv("SIGNALOPS_HTTP_ADDR", ":9000")
	t.Setenv("SIGNALOPS_BROKER_PROVIDER", "kafka")
	t.Setenv("SIGNALOPS_BROKER_BROKERS", "localhost:19092")
	t.Setenv("SIGNALOPS_ENV", "test")
	t.Setenv("SIGNALOPS_DATABASE_URL", "postgres://example")

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
}
