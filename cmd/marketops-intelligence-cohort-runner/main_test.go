package main

import (
	"strings"
	"testing"
	"time"
)

func validConfig() cliConfig {
	day := time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC)
	return cliConfig{TenantID: "tenant-1", Symbols: []string{"AAPL", "MSFT"}, MaxSymbols: 2, SessionStart: day, SessionEnd: day, Stages: []string{"preflight"}, DryRun: true, RunID: "cohort-test"}
}
func TestValidateCohortBoundsAndWriteAcknowledgement(t *testing.T) {
	cfg := validConfig()
	if err := validate(cfg); err != nil {
		t.Fatal(err)
	}
	cfg.MaxSymbols = 11
	if err := validate(cfg); err == nil || !strings.Contains(err.Error(), "10") {
		t.Fatalf("expected hard cap error, got %v", err)
	}
	cfg = validConfig()
	cfg.DryRun = false
	if err := validate(cfg); err == nil || !strings.Contains(err.Error(), "acknowledge") {
		t.Fatalf("expected write acknowledgement error, got %v", err)
	}
}
func TestValidateRequiresExclusiveSymbolOrUniverseScope(t *testing.T) {
	cfg := validConfig()
	cfg.UniverseGroup = "top50_megacap"
	if err := validate(cfg); err == nil {
		t.Fatal("accepted both explicit symbols and universe")
	}
	cfg = validConfig()
	cfg.Symbols = nil
	if err := validate(cfg); err == nil {
		t.Fatal("accepted empty scope")
	}
}
