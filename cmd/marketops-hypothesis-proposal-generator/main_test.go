package main

import (
	"context"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

type fakeRepository struct {
	definitions []storage.MarketOpsHypothesisDefinitionRecord
	evaluations []storage.MarketOpsHypothesisEvaluationRecord
	inserted    []storage.AlgorithmSignalProposalRecord
}

func (f *fakeRepository) ListMarketOpsHypothesisDefinitions(context.Context, storage.MarketOpsHypothesisDefinitionFilter) ([]storage.MarketOpsHypothesisDefinitionRecord, error) {
	return f.definitions, nil
}

func (f *fakeRepository) ListMarketOpsHypothesisEvaluations(context.Context, storage.MarketOpsHypothesisEvaluationFilter) ([]storage.MarketOpsHypothesisEvaluationRecord, error) {
	return f.evaluations, nil
}

func (f *fakeRepository) InsertAlgorithmSignalProposal(_ context.Context, record storage.AlgorithmSignalProposalRecord) (bool, error) {
	f.inserted = append(f.inserted, record)
	return true, nil
}

func TestGenerateHonorsDryRunAndWritesGovernedProposal(t *testing.T) {
	score := .8
	session := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	repo := &fakeRepository{
		definitions: []storage.MarketOpsHypothesisDefinitionRecord{{TenantID: "tenant-1", HypothesisKey: "H001", HypothesisVersion: "v1", LifecycleStatus: storage.MarketOpsHypothesisLifecycleCandidate}},
		evaluations: []storage.MarketOpsHypothesisEvaluationRecord{{EvaluationID: "eval-1", TenantID: "tenant-1", AppID: "marketops", HypothesisKey: "H001", HypothesisVersion: "v1", MarketStateID: "state-1", Symbol: "AAPL", SessionDate: session, Eligible: true, Triggered: true, TriggerScore: &score, ConfidenceScore: &score}},
	}
	cfg := cliConfig{TenantID: "tenant-1", Symbol: "AAPL", RunID: "run-1", CreatedBy: "analyst-1", SessionStart: session, SessionEnd: session, MaxSessions: 1, DryRun: true}
	result, err := generate(context.Background(), repo, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result.Built != 1 || result.Inserted != 0 || len(repo.inserted) != 0 {
		t.Fatalf("dry-run wrote proposal: result=%+v inserted=%d", result, len(repo.inserted))
	}
	cfg.DryRun = false
	result, err = generate(context.Background(), repo, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result.Inserted != 1 || len(repo.inserted) != 1 || !repo.inserted[0].ResearchOnly {
		t.Fatalf("governed proposal not written correctly: result=%+v records=%+v", result, repo.inserted)
	}
}

func TestCLIConfigBounds(t *testing.T) {
	valid := cliConfig{TenantID: "tenant-1", Symbol: "AAPL", RunID: "run-1", CreatedBy: "analyst", SessionStart: time.Now().UTC(), SessionEnd: time.Now().UTC(), MaxSessions: 50}
	if err := valid.validate(); err != nil {
		t.Fatal(err)
	}
	invalid := valid
	invalid.Symbol = "MSFT"
	if invalid.validate() == nil {
		t.Fatal("non-AAPL symbol accepted")
	}
	invalid = valid
	invalid.MaxSessions = 51
	if invalid.validate() == nil {
		t.Fatal("session cap exceeded")
	}
}
