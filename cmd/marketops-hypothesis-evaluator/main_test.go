package main

import (
	"context"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

type fakeRepository struct{ definitions, evaluations int }

func (f *fakeRepository) ListMarketOpsMarketStates(context.Context, storage.MarketOpsMarketStateFilter) ([]storage.MarketOpsMarketStateRecord, error) {
	q := .2
	return []storage.MarketOpsMarketStateRecord{{MarketStateID: "mstate-1", TenantID: "tenant-local", AppID: "marketops", AssetID: "ticker:AAPL", Symbol: "AAPL", SessionDate: time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC), AsOfTime: time.Date(2026, 7, 19, 23, 0, 0, 0, time.UTC), QualityState: storage.MarketOpsQualityPartial, QualityScore: &q}}, nil
}
func (f *fakeRepository) ListMarketOpsFeatureObservations(context.Context, storage.MarketOpsFeatureObservationFilter) ([]storage.MarketOpsFeatureObservationRecord, error) {
	return nil, nil
}
func (f *fakeRepository) ListMarketOpsStateTransitions(context.Context, storage.MarketOpsStateTransitionFilter) ([]storage.MarketOpsStateTransitionRecord, error) {
	return nil, nil
}
func (f *fakeRepository) ListMarketOpsEvidence(context.Context, storage.MarketOpsEvidenceFilter) ([]storage.MarketOpsEvidenceRecord, error) {
	return nil, nil
}
func (f *fakeRepository) UpsertMarketOpsHypothesisDefinition(context.Context, storage.MarketOpsHypothesisDefinitionRecord) error {
	f.definitions++
	return nil
}
func (f *fakeRepository) UpsertMarketOpsHypothesisEvaluation(context.Context, storage.MarketOpsHypothesisEvaluationRecord) error {
	f.evaluations++
	return nil
}

func TestEvaluateDryRunAndWrite(t *testing.T) {
	cfg := cliConfig{TenantID: "tenant-local", Symbol: "AAPL", RunID: "g138-test", SessionStart: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), SessionEnd: time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC), MaxSessions: 10, DryRun: true}
	repo := &fakeRepository{}
	dry, err := evaluate(context.Background(), repo, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if dry.Evaluations != 4 || dry.Rejected != 4 || repo.definitions+repo.evaluations != 0 {
		t.Fatalf("dry=%+v repo=%+v", dry, repo)
	}
	cfg.DryRun = false
	written, err := evaluate(context.Background(), repo, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if written.Evaluations != 4 || repo.definitions != 4 || repo.evaluations != 4 {
		t.Fatalf("written=%+v repo=%+v", written, repo)
	}
}

func TestConfigBoundary(t *testing.T) {
	cfg := cliConfig{TenantID: "tenant-local", Symbol: "MSFT", RunID: "run", SessionStart: time.Now(), SessionEnd: time.Now(), MaxSessions: 1}
	if cfg.validate() == nil {
		t.Fatal("expected AAPL boundary")
	}
}
