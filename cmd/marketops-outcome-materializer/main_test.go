package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

type fakeRepository struct {
	evaluations   []storage.MarketOpsHypothesisEvaluationRecord
	opportunities []storage.MarketOpsOpportunityRecord
	events        []storage.NormalizedEventLedgerRecord
	outcomes      []storage.MarketOpsSignalOutcomeRecord
}

func (f *fakeRepository) ListMarketOpsHypothesisEvaluations(context.Context, storage.MarketOpsHypothesisEvaluationFilter) ([]storage.MarketOpsHypothesisEvaluationRecord, error) {
	return f.evaluations, nil
}
func (f *fakeRepository) ListMarketOpsOpportunities(context.Context, storage.MarketOpsOpportunityFilter) ([]storage.MarketOpsOpportunityRecord, error) {
	return f.opportunities, nil
}
func (f *fakeRepository) ListMarketOpsBacktestNormalizedEvents(context.Context, storage.MarketOpsBacktestEventFilter) ([]storage.NormalizedEventLedgerRecord, error) {
	return f.events, nil
}
func (f *fakeRepository) UpsertMarketOpsSignalOutcome(_ context.Context, record storage.MarketOpsSignalOutcomeRecord) error {
	f.outcomes = append(f.outcomes, record)
	return nil
}

func TestMaterializeWritesOneRowPerSourceHorizon(t *testing.T) {
	sessions := cliBusinessSessions(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), 9)
	repo := &fakeRepository{evaluations: []storage.MarketOpsHypothesisEvaluationRecord{cliEvaluation(sessions[2])}, events: cliEquityEvents(sessions)}
	cfg := cliConfig{TenantID: "tenant-local", Symbol: "AAPL", RunID: "run-1", SessionStart: sessions[0], SessionEnd: sessions[3], AsOf: sessions[8], MaxSessions: 10, Threshold: .02}
	result, err := materialize(context.Background(), repo, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result.OutcomeSources != 1 || result.Outcomes != 4 || len(repo.outcomes) != 4 || result.Matured != 2 || result.Pending != 2 {
		t.Fatalf("unexpected metrics=%+v writes=%d", result, len(repo.outcomes))
	}
}

func TestMaterializeDryRunWritesNothing(t *testing.T) {
	sessions := cliBusinessSessions(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), 9)
	repo := &fakeRepository{evaluations: []storage.MarketOpsHypothesisEvaluationRecord{cliEvaluation(sessions[2])}, events: cliEquityEvents(sessions)}
	cfg := cliConfig{TenantID: "tenant-local", Symbol: "AAPL", RunID: "run-2", SessionStart: sessions[0], SessionEnd: sessions[3], AsOf: sessions[8], MaxSessions: 10, Threshold: .02, DryRun: true}
	result, err := materialize(context.Background(), repo, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcomes != 4 || len(repo.outcomes) != 0 {
		t.Fatalf("dry run wrote outcomes: metrics=%+v writes=%d", result, len(repo.outcomes))
	}
}

func TestCLIConfigBoundary(t *testing.T) {
	cfg := cliConfig{TenantID: "tenant-local", Symbol: "MSFT", RunID: "run", SessionStart: time.Now(), SessionEnd: time.Now(), AsOf: time.Now(), MaxSessions: 10, Threshold: .02}
	if cfg.validate() == nil {
		t.Fatal("expected AAPL boundary error")
	}
}

func cliBusinessSessions(start time.Time, count int) []time.Time {
	out := []time.Time{}
	for current := start; len(out) < count; current = current.AddDate(0, 0, 1) {
		if current.Weekday() != time.Saturday && current.Weekday() != time.Sunday {
			out = append(out, current)
		}
	}
	return out
}

func cliEquityEvents(sessions []time.Time) []storage.NormalizedEventLedgerRecord {
	out := []storage.NormalizedEventLedgerRecord{}
	for i, session := range sessions {
		closeValue := 100 + float64(i)
		payload, _ := json.Marshal(map[string]any{"symbol": "AAPL", "open": closeValue, "high": closeValue + 1, "low": closeValue - 1, "close": closeValue})
		out = append(out, storage.NormalizedEventLedgerRecord{EventID: fmt.Sprintf("evt-%d", i), TenantID: "tenant-local", AppID: "marketops", Dataset: "equity_eod_prices", ObservationTime: session, ProcessingTime: session.Add(time.Hour), NormalizedPayload: payload})
	}
	return out
}

func cliEvaluation(session time.Time) storage.MarketOpsHypothesisEvaluationRecord {
	score := .8
	payload, _ := json.Marshal(map[string]any{"resolved_direction": "upside"})
	return storage.MarketOpsHypothesisEvaluationRecord{EvaluationID: "eval-1", TenantID: "tenant-local", AppID: "marketops", HypothesisKey: "H001", HypothesisVersion: "v1", MarketStateID: "state-1", AssetID: "ticker:AAPL", Symbol: "AAPL", SessionDate: session, Eligible: true, Triggered: true, TriggerScore: &score, ConfidenceScore: &score, EvaluationPayloadJSON: payload}
}
