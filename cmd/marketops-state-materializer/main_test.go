package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

type fakeRepository struct {
	events        []storage.NormalizedEventLedgerRecord
	distributions []storage.MarketOpsOptionsDistributionRecord
	chain         []storage.MarketOpsOptionsChainRecord
	definitions   int
	observations  int
	states        int
	transitions   int
	evidence      int
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
	f.definitions++
	return nil
}
func (f *fakeRepository) UpsertMarketOpsFeatureObservation(context.Context, storage.MarketOpsFeatureObservationRecord) error {
	f.observations++
	return nil
}
func (f *fakeRepository) UpsertMarketOpsMarketState(context.Context, storage.MarketOpsMarketStateRecord) error {
	f.states++
	return nil
}
func (f *fakeRepository) UpsertMarketOpsStateTransition(context.Context, storage.MarketOpsStateTransitionRecord) error {
	f.transitions++
	return nil
}
func (f *fakeRepository) UpsertMarketOpsEvidence(context.Context, storage.MarketOpsEvidenceRecord) error {
	f.evidence++
	return nil
}

func TestMaterializeDryRunDoesNotWrite(t *testing.T) {
	repo := &fakeRepository{events: []storage.NormalizedEventLedgerRecord{equityEvent()}}
	cfg := validConfig()
	cfg.DryRun = true
	result, err := materialize(context.Background(), repo, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result.States != 1 || result.Observations != 75 || repo.definitions+repo.observations+repo.states+repo.transitions+repo.evidence != 0 {
		t.Fatalf("unexpected dry-run result: result=%+v repo=%+v", result, repo)
	}
}

func TestMaterializeWritesBuiltRecords(t *testing.T) {
	repo := &fakeRepository{events: []storage.NormalizedEventLedgerRecord{equityEvent()}}
	result, err := materialize(context.Background(), repo, validConfig())
	if err != nil {
		t.Fatal(err)
	}
	if repo.definitions != result.Definitions || repo.observations != result.Observations || repo.states != result.States || repo.transitions != result.Transitions || repo.evidence != result.Evidence {
		t.Fatalf("write counts do not match metrics: result=%+v repo=%+v", result, repo)
	}
}

func TestConfigAcceptsExplicitNonAAPLAndRejectsInvalidWindow(t *testing.T) {
	cfg := validConfig()
	cfg.Symbol = "MSFT"
	if err := cfg.validate(); err != nil {
		t.Fatalf("explicit symbol was rejected: %v", err)
	}
	cfg = validConfig()
	cfg.WindowEnd = cfg.WindowStart
	if err := cfg.validate(); err == nil {
		t.Fatal("expected invalid window error")
	}
}

func validConfig() cliConfig {
	return cliConfig{TenantID: "tenant-local", Symbol: "AAPL", AssetID: "ticker:AAPL", WindowStart: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), WindowEnd: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), MaxSessions: 100, RunID: "g137-test"}
}

func equityEvent() storage.NormalizedEventLedgerRecord {
	payload, _ := json.Marshal(map[string]any{"symbol": "AAPL", "open": 100, "high": 102, "low": 99, "close": 101, "volume": 1_000_000})
	return storage.NormalizedEventLedgerRecord{EventID: "evt-aapl", TenantID: "tenant-local", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", Dataset: "equity_eod_prices", ObservationTime: time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC), ProcessingTime: time.Date(2026, 7, 9, 1, 0, 0, 0, time.UTC), NormalizedPayload: payload}
}
