package outcomes

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestBuildMaturesAllHorizonsWithExactLineage(t *testing.T) {
	sessions := businessSessions(time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), 40)
	originIndex := 10
	evaluation := evaluationFixture("eval-up", sessions[originIndex], "upside")
	result, err := Build(BuildConfig{TenantID: "tenant-local", Symbol: "AAPL", RunID: "run-1", AsOf: sessions[len(sessions)-1]},
		BuildInput{Evaluations: []storage.MarketOpsHypothesisEvaluationRecord{evaluation}, EquityEvents: equityFixtures(sessions)})
	if err != nil {
		t.Fatal(err)
	}
	if result.Sources != 1 || len(result.Outcomes) != 4 || result.Matured != 4 || result.Pending != 0 || result.MissingPrice != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	for _, record := range result.Outcomes {
		if record.OutcomeStatus != storage.MarketOpsOutcomeMatured || record.MaturedSessionDate == nil || record.ForwardReturn == nil {
			t.Fatalf("outcome not matured: %+v", record)
		}
		if len(record.OutcomeEventIDs) != record.HorizonSessions || record.OriginEventID != fmt.Sprintf("evt-%02d", originIndex) {
			t.Fatalf("lineage mismatch: %+v", record)
		}
		if record.DirectionalHit == nil || !*record.DirectionalHit {
			t.Fatalf("expected upside directional hit: %+v", record)
		}
		if record.HorizonSessions == 1 && record.RealizedVolChange != nil {
			t.Fatalf("one-session volatility change must be unavailable: %+v", record)
		}
	}
}

func TestBuildDirectionAndThresholdSemantics(t *testing.T) {
	sessions := businessSessions(time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC), 12)
	events := equityFixtures(sessions)
	for i := 6; i < len(events); i++ {
		payload, _ := json.Marshal(map[string]any{"symbol": "AAPL", "open": 100 - float64(i-5)*2, "high": 101 - float64(i-5)*2, "low": 98 - float64(i-5)*2, "close": 100 - float64(i-5)*2})
		events[i].NormalizedPayload = payload
	}
	origin := sessions[5]
	result, err := Build(BuildConfig{TenantID: "tenant-local", Symbol: "AAPL", RunID: "run-down", AsOf: sessions[len(sessions)-1], Horizons: []int{5}},
		BuildInput{Evaluations: []storage.MarketOpsHypothesisEvaluationRecord{evaluationFixture("eval-down", origin, "downside")}, EquityEvents: events})
	if err != nil {
		t.Fatal(err)
	}
	record := result.Outcomes[0]
	if record.ForwardReturn == nil || *record.ForwardReturn >= 0 || record.DirectionalHit == nil || !*record.DirectionalHit {
		t.Fatalf("expected downside hit: %+v", record)
	}
	if record.ThresholdHit == nil || !*record.ThresholdHit || record.DaysToThreshold == nil {
		t.Fatalf("expected downside threshold hit: %+v", record)
	}
	if record.MaxFavorableExcursion == nil || *record.MaxFavorableExcursion <= 0 || record.MaxAdverseExcursion == nil {
		t.Fatalf("expected direction-adjusted excursions: %+v", record)
	}
}

func TestBuildPendingAndMissingPriceRemainExplicit(t *testing.T) {
	sessions := businessSessions(time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC), 7)
	origin := sessions[5]
	input := BuildInput{Evaluations: []storage.MarketOpsHypothesisEvaluationRecord{evaluationFixture("eval-pending", origin, "upside")}, EquityEvents: equityFixtures(sessions)}
	pending, err := Build(BuildConfig{TenantID: "tenant-local", Symbol: "AAPL", RunID: "run-pending", AsOf: sessions[6], Horizons: []int{5}}, input)
	if err != nil {
		t.Fatal(err)
	}
	if pending.Pending != 1 || pending.Outcomes[0].OutcomeStatus != storage.MarketOpsOutcomePending {
		t.Fatalf("expected pending outcome: %+v", pending)
	}
	missing, err := Build(BuildConfig{TenantID: "tenant-local", Symbol: "AAPL", RunID: "run-missing", AsOf: addBusinessDays(origin, 6), Horizons: []int{5}}, input)
	if err != nil {
		t.Fatal(err)
	}
	if missing.MissingPrice != 1 || missing.Outcomes[0].OutcomeStatus != storage.MarketOpsOutcomeMissingPrice {
		t.Fatalf("expected missing price outcome: %+v", missing)
	}
}

func TestBuildOpportunityNonDirectionalAndStableIdentity(t *testing.T) {
	sessions := businessSessions(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), 10)
	opportunity := storage.MarketOpsOpportunityRecord{OpportunityID: "mopp-1", TenantID: "tenant-local", AppID: "marketops", AssetID: "ticker:AAPL", Symbol: "AAPL", Direction: "non_directional", LastEvaluatedDate: sessions[3]}
	input := BuildInput{Opportunities: []storage.MarketOpsOpportunityRecord{opportunity}, EquityEvents: equityFixtures(sessions)}
	first, err := Build(BuildConfig{TenantID: "tenant-local", Symbol: "AAPL", RunID: "run-a", AsOf: sessions[9], Horizons: []int{5}}, input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Build(BuildConfig{TenantID: "tenant-local", Symbol: "AAPL", RunID: "run-b", AsOf: sessions[9], Horizons: []int{5}}, input)
	if err != nil {
		t.Fatal(err)
	}
	left, right := first.Outcomes[0], second.Outcomes[0]
	if left.OutcomeID != right.OutcomeID || left.DeterministicKey != right.DeterministicKey || left.CalculationRunID == right.CalculationRunID {
		t.Fatalf("identity is not stable across runs: %+v %+v", left, right)
	}
	if left.DirectionalHit != nil || left.MaxAdverseExcursion != nil {
		t.Fatalf("non-directional outcome should avoid directional claims: %+v", left)
	}
}

func TestBuildSkipsRejectedEvaluationAndRejectsWrongSymbol(t *testing.T) {
	sessions := businessSessions(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), 4)
	evaluation := evaluationFixture("eval-rejected", sessions[1], "upside")
	evaluation.Triggered = false
	result, err := Build(BuildConfig{TenantID: "tenant-local", Symbol: "AAPL", RunID: "run", AsOf: sessions[3]},
		BuildInput{Evaluations: []storage.MarketOpsHypothesisEvaluationRecord{evaluation}, EquityEvents: equityFixtures(sessions)})
	if err != nil {
		t.Fatal(err)
	}
	if result.Sources != 0 || len(result.Outcomes) != 0 || result.SkippedReasons["evaluation_not_triggered"] != 1 {
		t.Fatalf("rejected evaluation admitted: %+v", result)
	}
	_, err = Build(BuildConfig{TenantID: "tenant-local", Symbol: "MSFT", RunID: "run", AsOf: sessions[3]}, BuildInput{})
	if err == nil {
		t.Fatal("expected AAPL boundary error")
	}
}

func businessSessions(start time.Time, count int) []time.Time {
	out := make([]time.Time, 0, count)
	for current := start; len(out) < count; current = current.AddDate(0, 0, 1) {
		if current.Weekday() != time.Saturday && current.Weekday() != time.Sunday {
			out = append(out, current)
		}
	}
	return out
}

func equityFixtures(sessions []time.Time) []storage.NormalizedEventLedgerRecord {
	out := make([]storage.NormalizedEventLedgerRecord, 0, len(sessions))
	for i, session := range sessions {
		closeValue := 100 + float64(i)*1.5 + math.Sin(float64(i))
		payload, _ := json.Marshal(map[string]any{"symbol": "AAPL", "open": closeValue - .5, "high": closeValue + 1, "low": closeValue - 1, "close": closeValue})
		out = append(out, storage.NormalizedEventLedgerRecord{EventID: fmt.Sprintf("evt-%02d", i), TenantID: "tenant-local", AppID: "marketops", Dataset: "equity_eod_prices", ObservationTime: session, ProcessingTime: session.Add(time.Hour), NormalizedPayload: payload})
	}
	return out
}

func evaluationFixture(id string, session time.Time, direction string) storage.MarketOpsHypothesisEvaluationRecord {
	payload, _ := json.Marshal(map[string]any{"resolved_direction": direction, "horizon": "5_to_20_sessions"})
	score := .8
	return storage.MarketOpsHypothesisEvaluationRecord{
		EvaluationID: id, TenantID: "tenant-local", AppID: "marketops", HypothesisKey: "H001",
		HypothesisVersion: "v1", MarketStateID: "state-1", AssetID: "ticker:AAPL", Symbol: "AAPL",
		SessionDate: session, Eligible: true, Triggered: true, TriggerScore: &score,
		ConfidenceScore: &score, EvaluationPayloadJSON: payload,
	}
}
