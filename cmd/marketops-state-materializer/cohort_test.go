package main

import (
	"context"
	"testing"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestMaterializeCohortProcessesOnlyExplicitBoundedSymbols(t *testing.T) {
	repo := &fakeRepository{events: []storage.NormalizedEventLedgerRecord{equityEvent()}}
	cfg := validConfig()
	cfg.Symbols = []string{"aapl", "MSFT", "aapl"}
	cfg.Symbol = ""
	cfg.AssetID = ""
	cfg.MaxSymbols = 2
	cfg.DryRun = true
	result, err := materializeCohort(context.Background(), repo, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result.SymbolsRequested != 2 || result.SymbolsProcessed != 2 || len(result.SymbolResults) != 2 || result.Definitions != 44 || result.States != 1 {
		t.Fatalf("unexpected cohort metrics: %+v", result)
	}
	if result.SymbolResults[0].Symbol != "AAPL" || result.SymbolResults[0].RunID != cfg.RunID+"_aapl" || result.SymbolResults[1].Symbol != "MSFT" {
		t.Fatalf("cohort order or run isolation changed: %+v", result.SymbolResults)
	}
	if repo.definitions+repo.observations+repo.states+repo.transitions+repo.evidence != 0 {
		t.Fatal("dry-run cohort wrote ledger records")
	}
}

func TestMaterializeCohortRejectsFanoutBeyondHardCap(t *testing.T) {
	cfg := validConfig()
	cfg.Symbols = []string{"AAPL", "MSFT", "NVDA"}
	cfg.Symbol = ""
	cfg.AssetID = ""
	cfg.MaxSymbols = 2
	if _, err := materializeCohort(context.Background(), &fakeRepository{}, cfg); err == nil {
		t.Fatal("expected explicit cohort cap error")
	}
	cfg.MaxSymbols = 11
	if _, err := materializeCohort(context.Background(), &fakeRepository{}, cfg); err == nil {
		t.Fatal("expected hard maximum error")
	}
}

func TestParseExplicitSymbolsNormalizesAndDeduplicates(t *testing.T) {
	got := parseExplicitSymbols(" aapl,MSFT,AAPL,, nvda ")
	if len(got) != 3 || got[0] != "AAPL" || got[1] != "MSFT" || got[2] != "NVDA" {
		t.Fatalf("symbols = %+v", got)
	}
}
