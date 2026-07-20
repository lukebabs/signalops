package history

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

type fakeRepository struct {
	events        []storage.NormalizedEventLedgerRecord
	distributions []storage.MarketOpsOptionsDistributionRecord
	chain         []storage.MarketOpsOptionsChainRecord
	writes        int
}

func (f *fakeRepository) ListMarketOpsBacktestNormalizedEvents(context.Context, storage.MarketOpsBacktestEventFilter) ([]storage.NormalizedEventLedgerRecord, error) {
	return f.events, nil
}
func (f *fakeRepository) ListMarketOpsOptionsDistributions(context.Context, storage.MarketOpsOptionsDistributionFilter) ([]storage.MarketOpsOptionsDistributionRecord, error) {
	return f.distributions, nil
}
func (f *fakeRepository) ListMarketOpsOptionsChain(context.Context, storage.MarketOpsOptionsChainFilter) ([]storage.MarketOpsOptionsChainRecord, error) {
	return f.chain, nil
}
func (f *fakeRepository) UpsertMarketOpsFeatureDefinition(context.Context, storage.MarketOpsFeatureDefinitionRecord) error {
	f.writes++
	return nil
}
func (f *fakeRepository) UpsertMarketOpsFeatureObservation(context.Context, storage.MarketOpsFeatureObservationRecord) error {
	f.writes++
	return nil
}
func (f *fakeRepository) UpsertMarketOpsMarketState(context.Context, storage.MarketOpsMarketStateRecord) error {
	f.writes++
	return nil
}
func (f *fakeRepository) UpsertMarketOpsStateTransition(context.Context, storage.MarketOpsStateTransitionRecord) error {
	f.writes++
	return nil
}
func (f *fakeRepository) UpsertMarketOpsEvidence(context.Context, storage.MarketOpsEvidenceRecord) error {
	f.writes++
	return nil
}
func (f *fakeRepository) UpsertMarketOpsHypothesisDefinition(context.Context, storage.MarketOpsHypothesisDefinitionRecord) error {
	f.writes++
	return nil
}
func (f *fakeRepository) UpsertMarketOpsHypothesisEvaluation(context.Context, storage.MarketOpsHypothesisEvaluationRecord) error {
	f.writes++
	return nil
}
func (f *fakeRepository) UpsertMarketOpsOpportunity(context.Context, storage.MarketOpsOpportunityRecord) error {
	f.writes++
	return nil
}
func (f *fakeRepository) UpsertMarketOpsSignalOutcome(context.Context, storage.MarketOpsSignalOutcomeRecord) error {
	f.writes++
	return nil
}

func TestRunBlocksBeforeWritesWhenCoverageIsInsufficient(t *testing.T) {
	start := testDay(0)
	repo := &fakeRepository{events: equityEvents(3)}
	result, err := Run(context.Background(), repo, Config{
		TenantID: "tenant-local", Symbol: "AAPL", RunID: "history-blocked",
		SessionStart: start, SessionEnd: testDay(4), AsOf: testDay(30),
		MaxSessions: 30, MinimumEquitySessions: 3, MinimumOptionsSessions: 2,
		AllowInsufficientCoverage: false,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Status != "blocked_insufficient_coverage" || result.Coverage.Ready {
		t.Fatalf("status/coverage = %s/%+v", result.Status, result.Coverage)
	}
	if len(result.Coverage.Warnings) != 1 || result.Coverage.Warnings[0] != "insufficient_option_distribution_sessions_for_positioning_hypotheses" {
		t.Fatalf("coverage warnings = %+v", result.Coverage.Warnings)
	}
	if repo.writes != 0 || result.MarketStates != 0 {
		t.Fatalf("writes/states = %d/%d", repo.writes, result.MarketStates)
	}
}

func TestRunHistoricalPipelineProducesTriggeredSourcesAndMaturedOutcomes(t *testing.T) {
	repo := positiveRepository()
	result, err := Run(context.Background(), repo, Config{
		TenantID: "tenant-local", Symbol: "AAPL", RunID: "history-positive",
		SessionStart: testDay(0), SessionEnd: testDay(5), AsOf: testDay(30),
		MaxSessions: 30, MinimumEquitySessions: 5, MinimumOptionsSessions: 5,
		AllowInsufficientCoverage: false, DryRun: true,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Status != "completed" || !result.Coverage.Ready {
		t.Fatalf("status/coverage = %s/%+v", result.Status, result.Coverage)
	}
	if result.MarketStates != 5 || result.HypothesisEvaluations != 20 {
		t.Fatalf("states/evaluations = %d/%d", result.MarketStates, result.HypothesisEvaluations)
	}
	if result.TriggeredEvaluations == 0 || result.Opportunities == 0 || result.MaturedOutcomes == 0 {
		t.Fatalf("triggered/opportunities/matured = %d/%d/%d; reasons=%+v", result.TriggeredEvaluations, result.Opportunities, result.MaturedOutcomes, result.EvaluationReasons)
	}
	if repo.writes != 0 {
		t.Fatalf("dry run writes = %d", repo.writes)
	}
}

func TestRunPersistsEachBuiltLayer(t *testing.T) {
	repo := positiveRepository()
	result, err := Run(context.Background(), repo, Config{
		TenantID: "tenant-local", Symbol: "AAPL", RunID: "history-write",
		SessionStart: testDay(0), SessionEnd: testDay(5), AsOf: testDay(30),
		MaxSessions: 30, MinimumEquitySessions: 5, MinimumOptionsSessions: 5,
		AllowInsufficientCoverage: false,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	expected := result.FeatureDefinitions + result.FeatureObservations + result.MarketStates +
		result.StateTransitions + result.Evidence + result.HypothesisDefinitions +
		result.HypothesisEvaluations + result.Opportunities + result.Outcomes
	if repo.writes != expected {
		t.Fatalf("writes = %d, expected %d", repo.writes, expected)
	}
}

func TestValidateRejectsUnboundedOrWrongSymbol(t *testing.T) {
	cfg := Config{
		TenantID: "tenant-local", Symbol: "NVDA", RunID: "bad",
		SessionStart: testDay(0), SessionEnd: testDay(5), AsOf: testDay(30),
		MaxSessions: 30, MinimumEquitySessions: 5, MinimumOptionsSessions: 5,
	}
	if _, err := Run(context.Background(), &fakeRepository{}, cfg); err == nil {
		t.Fatal("expected AAPL boundary error")
	}
	cfg.Symbol = "AAPL"
	cfg.MaxSessions = 201
	if _, err := Run(context.Background(), &fakeRepository{}, cfg); err == nil {
		t.Fatal("expected maximum-session boundary error")
	}
	cfg.MaxSessions = 30
	cfg.AllowInsufficientCoverage = true
	if _, err := Run(context.Background(), &fakeRepository{}, cfg); err == nil {
		t.Fatal("expected partial write boundary error")
	}
}

func TestHasRequiredSurfaceChecksCellsNotOnlyContractCount(t *testing.T) {
	session := testDay(0)
	complete := optionSurface(session, 0)
	if !hasRequiredSurface(session, complete) {
		t.Fatal("expected complete required surface")
	}
	incomplete := append([]storage.MarketOpsOptionsChainRecord{}, complete[:4]...)
	incomplete = append(incomplete, complete[0])
	if hasRequiredSurface(session, incomplete) {
		t.Fatal("contract count must not replace the missing call 25-delta cell")
	}
}

func positiveRepository() *fakeRepository {
	repo := &fakeRepository{events: equityEvents(31)}
	for index := 0; index < 5; index++ {
		session := testDay(index)
		repo.chain = append(repo.chain, optionSurface(session, index)...)
		metrics, _ := json.Marshal(map[string]any{"call_put_oi_ratio_quality": "usable", "open_interest_quality": "complete"})
		repo.distributions = append(repo.distributions, storage.MarketOpsOptionsDistributionRecord{
			TenantID: "tenant-local", Symbol: "AAPL", TradeDate: session, WindowName: "10_trade_days",
			TotalCallOpenInterest: 1000, TotalPutOpenInterest: int64(1000 + index*10),
			TotalCallVolume: 100, TotalPutVolume: 120, MetricsJSON: metrics,
		})
	}
	return repo
}

func equityEvents(count int) []storage.NormalizedEventLedgerRecord {
	records := make([]storage.NormalizedEventLedgerRecord, 0, count)
	for index := 0; index < count; index++ {
		session := testDay(index)
		closeValue := 100.0 + float64(index)
		payload, _ := json.Marshal(map[string]any{
			"symbol": "AAPL", "open": closeValue - .5, "high": closeValue + 1,
			"low": closeValue - 1, "close": closeValue, "volume": 1000000 + index,
		})
		records = append(records, storage.NormalizedEventLedgerRecord{
			EventID: fmt.Sprintf("evt-aapl-%02d", index), TenantID: "tenant-local", AppID: "marketops",
			Domain: "market_data", UseCase: "daily_market_surveillance", Dataset: "equity_eod_prices",
			ObservationTime: session, ProcessingTime: session.Add(time.Hour), NormalizedPayload: payload,
		})
	}
	return records
}

func optionSurface(session time.Time, index int) []storage.MarketOpsOptionsChainRecord {
	value30 := .40 + float64(index)*.02
	value60 := .34 + float64(index)*.02
	value90 := .28 + float64(index)*.02
	return []storage.MarketOpsOptionsChainRecord{
		contract(session, "C30ATM", "call", 30, .50, value30),
		contract(session, "C60ATM", "call", 60, .50, value60),
		contract(session, "C90ATM", "call", 90, .50, value90),
		contract(session, "P30D25", "put", 30, -.25, value30+.03),
		contract(session, "C30D25", "call", 30, .25, value30-.03),
	}
}

func contract(session time.Time, suffix, optionType string, dte int, deltaValue, ivValue float64) storage.MarketOpsOptionsChainRecord {
	underlying := 100.0
	openInterest := int64(1000)
	return storage.MarketOpsOptionsChainRecord{
		TenantID: "tenant-local", Symbol: "AAPL", TradeDate: session,
		OptionTicker: fmt.Sprintf("O:AAPL%s%s", session.Format("060102"), suffix),
		ContractType: optionType, ExpirationDate: session.AddDate(0, 0, dte), StrikePrice: 100,
		UnderlyingClose: &underlying, OpenInterest: &openInterest, ImpliedVolatility: &ivValue, Delta: &deltaValue,
	}
}

func testDay(offset int) time.Time {
	return time.Date(2026, 1, 2+offset, 0, 0, 0, 0, time.UTC)
}
