package main

import "testing"

func TestLoadConfigDefaultsAndExplicitBounds(t *testing.T) {
	cfg, err := loadConfig([]string{
		"--session-start", "2026-01-02",
		"--session-end", "2026-06-01",
		"--as-of", "2026-07-01",
		"--run-id", "g141-test",
		"--dry-run",
	})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Symbol != "AAPL" || cfg.AssetID != "ticker:AAPL" || cfg.RunID != "g141-test" {
		t.Fatalf("config = %+v", cfg)
	}
	if !cfg.DryRun || cfg.AllowInsufficientCoverage {
		t.Fatalf("dry-run/allow-insufficient = %v/%v", cfg.DryRun, cfg.AllowInsufficientCoverage)
	}
}

func TestLoadConfigAllowsExplicitDiagnosticMode(t *testing.T) {
	cfg, err := loadConfig([]string{
		"--session-start", "2026-01-02",
		"--session-end", "2026-06-01",
		"--as-of", "2026-07-01",
		"--allow-insufficient-coverage",
	})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.AllowInsufficientCoverage {
		t.Fatal("expected diagnostic partial mode")
	}
}

func TestLoadConfigRejectsInvalidDate(t *testing.T) {
	if _, err := loadConfig([]string{"--session-start", "not-a-date"}); err == nil {
		t.Fatal("expected date error")
	}
}
