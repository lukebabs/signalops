package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/marketops/opportunities"
	"github.com/lukebabs/signalops/internal/storage"
)

type fakeRepository struct {
	definitions   []storage.MarketOpsHypothesisDefinitionRecord
	evaluations   []storage.MarketOpsHypothesisEvaluationRecord
	opportunities []storage.MarketOpsOpportunityRecord
}

func (f *fakeRepository) ListMarketOpsHypothesisDefinitions(context.Context, storage.MarketOpsHypothesisDefinitionFilter) ([]storage.MarketOpsHypothesisDefinitionRecord, error) {
	return f.definitions, nil
}
func (f *fakeRepository) ListMarketOpsHypothesisEvaluations(context.Context, storage.MarketOpsHypothesisEvaluationFilter) ([]storage.MarketOpsHypothesisEvaluationRecord, error) {
	return f.evaluations, nil
}
func (f *fakeRepository) UpsertMarketOpsOpportunity(_ context.Context, record storage.MarketOpsOpportunityRecord) error {
	f.opportunities = append(f.opportunities, record)
	return nil
}

func TestBuildDryRunAndWrite(t *testing.T) {
	score := .8
	session := time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC)
	payload, _ := json.Marshal(map[string]any{"resolved_direction": "downside", "horizon": opportunities.DefaultHorizon})
	repo := &fakeRepository{
		definitions: []storage.MarketOpsHypothesisDefinitionRecord{{TenantID: "tenant-local", HypothesisKey: "H001", HypothesisVersion: "v1", Domain: "momentum_exhaustion", Direction: "downside", LifecycleStatus: storage.MarketOpsHypothesisLifecycleResearch}},
		evaluations: []storage.MarketOpsHypothesisEvaluationRecord{{EvaluationID: "eval-1", TenantID: "tenant-local", AppID: "marketops", HypothesisKey: "H001", HypothesisVersion: "v1", AssetID: "ticker:AAPL", Symbol: "AAPL", SessionDate: session, Eligible: true, Triggered: true, TriggerScore: &score, ConfidenceScore: &score, QualityScore: &score, EvaluationPayloadJSON: payload}},
	}
	cfg := cliConfig{TenantID: "tenant-local", Symbol: "AAPL", RunID: "run-1", SessionStart: session, SessionEnd: session, MaxSessions: 10, DryRun: true}
	dry, err := build(context.Background(), repo, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if dry.Opportunities != 1 || dry.Emerging != 1 || len(repo.opportunities) != 0 {
		t.Fatalf("dry=%+v writes=%d", dry, len(repo.opportunities))
	}
	cfg.DryRun = false
	written, err := build(context.Background(), repo, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if written.Opportunities != 1 || len(repo.opportunities) != 1 {
		t.Fatalf("written=%+v writes=%d", written, len(repo.opportunities))
	}
}

func TestConfigBoundary(t *testing.T) {
	now := time.Now().UTC()
	cfg := cliConfig{TenantID: "tenant-local", Symbol: "MSFT", RunID: "run", SessionStart: now, SessionEnd: now, MaxSessions: 1}
	if cfg.validate() == nil {
		t.Fatal("expected AAPL boundary")
	}
}
