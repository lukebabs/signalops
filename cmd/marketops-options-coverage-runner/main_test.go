package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/adapters/marketdata/massive"
	"github.com/lukebabs/signalops/internal/storage"
)

type fakeProvider struct {
	recordsBySymbol map[string][]massive.OptionContractDailyRecord
	calls           []string
	limit           int
	pages           int
	filter          massive.OptionChainSnapshotFilter
}

func (f *fakeProvider) ListOptionChainSnapshotFiltered(_ context.Context, underlying string, filter massive.OptionChainSnapshotFilter) ([]massive.OptionContractDailyRecord, error) {
	f.calls = append(f.calls, underlying)
	f.limit = filter.Limit
	f.pages = filter.MaxPages
	f.filter = filter
	return f.recordsBySymbol[underlying], nil
}

type fakeRepo struct {
	assets           []storage.MarketOpsAssetRecord
	chain            []storage.MarketOpsOptionsChainRecord
	distributions    []storage.MarketOpsOptionsDistributionRecord
	events           []storage.NormalizedEventLedgerRecord
	normalizedEvents []storage.NormalizedEventLedgerRecord
	captures         []storage.MarketOpsOptionsCaptureRecord
	assetLimit       int
	assetUniverse    string
	assetActiveOnly  bool
	lastChainFilters []storage.MarketOpsOptionsChainFilter
}

func (f *fakeRepo) ListMarketOpsAssets(_ context.Context, tenantID string, universeGroup string, activeOnly bool, limit int) ([]storage.MarketOpsAssetRecord, error) {
	f.assetLimit = limit
	f.assetUniverse = universeGroup
	f.assetActiveOnly = activeOnly
	out := []storage.MarketOpsAssetRecord{}
	for _, asset := range f.assets {
		if asset.TenantID == tenantID || asset.TenantID == "" {
			out = append(out, asset)
		}
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (f *fakeRepo) UpsertMarketOpsOptionsChain(_ context.Context, record storage.MarketOpsOptionsChainRecord) error {
	f.chain = append(f.chain, record)
	return nil
}

func (f *fakeRepo) ListMarketOpsOptionsChain(_ context.Context, filter storage.MarketOpsOptionsChainFilter) ([]storage.MarketOpsOptionsChainRecord, error) {
	f.lastChainFilters = append(f.lastChainFilters, filter)
	out := []storage.MarketOpsOptionsChainRecord{}
	for _, row := range f.chain {
		if strings.EqualFold(row.Symbol, filter.Symbol) {
			out = append(out, row)
		}
	}
	return out, nil
}

func (f *fakeRepo) UpsertMarketOpsOptionsDistribution(_ context.Context, record storage.MarketOpsOptionsDistributionRecord) error {
	f.distributions = append(f.distributions, record)
	return nil
}

func (f *fakeRepo) ListMarketOpsOptionsDistributions(_ context.Context, filter storage.MarketOpsOptionsDistributionFilter) ([]storage.MarketOpsOptionsDistributionRecord, error) {
	out := []storage.MarketOpsOptionsDistributionRecord{}
	for _, row := range f.distributions {
		if strings.EqualFold(row.Symbol, filter.Symbol) {
			out = append(out, row)
		}
	}
	return out, nil
}

func (f *fakeRepo) UpsertNormalizedEventLedger(_ context.Context, record storage.NormalizedEventLedgerRecord) error {
	f.events = append(f.events, record)
	return nil
}

func (f *fakeRepo) ListMarketOpsBacktestNormalizedEvents(_ context.Context, _ storage.MarketOpsBacktestEventFilter) ([]storage.NormalizedEventLedgerRecord, error) {
	return f.normalizedEvents, nil
}

func (f *fakeRepo) UpsertMarketOpsOptionsCapture(_ context.Context, record storage.MarketOpsOptionsCaptureRecord) error {
	f.captures = append(f.captures, record)
	return nil
}

func (f *fakeRepo) GetMarketOpsOptionsCapture(_ context.Context, tenantID string, captureID string) (storage.MarketOpsOptionsCaptureRecord, error) {
	for _, record := range f.captures {
		if record.TenantID == tenantID && record.CaptureID == captureID {
			return record, nil
		}
	}
	return storage.MarketOpsOptionsCaptureRecord{}, storage.ErrNotFound
}

func TestRunCoverageProcessesExplicitBoundedSymbols(t *testing.T) {
	provider := &fakeProvider{recordsBySymbol: map[string][]massive.OptionContractDailyRecord{
		"NVDA": {providerRecord("NVDA", "O:NVDA260116C00100000", "call", 200), providerRecord("NVDA", "O:NVDA260116P00100000", "put", 100)},
		"AAPL": {providerRecord("AAPL", "O:AAPL260116C00100000", "call", 50), providerRecord("AAPL", "O:AAPL260116P00100000", "put", 25)},
	}}
	repo := &fakeRepo{}
	result, err := runCoverage(context.Background(), provider, repo, cliConfig{TenantID: "tenant-local", Symbols: []string{"nvda", "aapl", "msft"}, MaxSymbols: 2, Limit: 10, MaxPages: 1, WindowDays: 10, ChainScanLimit: 100, DistributionLimit: 10, RunID: "optcov-test"})
	if err != nil {
		t.Fatalf("runCoverage: %v", err)
	}
	if result.SymbolsRequested != 2 || result.SymbolsProcessed != 2 || result.Fetched != 4 || result.ChainUpserted != 4 || result.DistributionsUpserted != 2 || result.FeatureRowsUpserted != 2 {
		t.Fatalf("result = %+v", result)
	}
	if len(provider.calls) != 2 || provider.calls[0] != "NVDA" || provider.calls[1] != "AAPL" || provider.limit != 10 || provider.pages != 1 {
		t.Fatalf("provider = %+v", provider)
	}
	if len(repo.events) != 2 || repo.events[0].Dataset != "options_distribution_daily" {
		t.Fatalf("events = %+v", repo.events)
	}
	if result.QualityCounts["usable"] != 2 {
		t.Fatalf("quality counts = %+v", result.QualityCounts)
	}
}

func TestRunCoverageResolvesUniverseAndDryRunSuppressesWrites(t *testing.T) {
	provider := &fakeProvider{recordsBySymbol: map[string][]massive.OptionContractDailyRecord{
		"NVDA": {providerRecord("NVDA", "O:NVDA260116C00100000", "call", 0), providerRecord("NVDA", "O:NVDA260116P00100000", "put", 0)},
	}}
	repo := &fakeRepo{assets: []storage.MarketOpsAssetRecord{{TenantID: "tenant-local", Ticker: "NVDA"}, {TenantID: "tenant-local", Ticker: "AAPL"}}}
	result, err := runCoverage(context.Background(), provider, repo, cliConfig{TenantID: "tenant-local", UniverseGroup: "top50_megacap", MaxSymbols: 1, DryRun: true, RunID: "optcov-test"})
	if err != nil {
		t.Fatalf("runCoverage dry run: %v", err)
	}
	if repo.assetUniverse != "top50_megacap" || !repo.assetActiveOnly || repo.assetLimit != 1 {
		t.Fatalf("asset lookup = %s/%t/%d", repo.assetUniverse, repo.assetActiveOnly, repo.assetLimit)
	}
	if result.SymbolsProcessed != 1 || result.Fetched != 2 || result.ChainUpserted != 0 || result.DistributionsUpserted != 0 || result.FeatureRowsUpserted != 0 {
		t.Fatalf("result = %+v", result)
	}
	if len(repo.chain) != 0 || len(repo.distributions) != 0 || len(repo.events) != 0 {
		t.Fatalf("dry-run writes chain=%d distributions=%d events=%d", len(repo.chain), len(repo.distributions), len(repo.events))
	}
	if result.QualityCounts["all_zero"] != 1 || result.Symbols[0].OpenInterestQualityCounts["all_zero"] != 1 {
		t.Fatalf("quality = %+v symbol=%+v", result.QualityCounts, result.Symbols[0].OpenInterestQualityCounts)
	}
}

func TestParseSymbolsNormalizesDeduplicatesAndLimits(t *testing.T) {
	got := limitSymbols(parseSymbols(" nvda, AAPL,nvda, msft "), 2)
	if len(got) != 2 || got[0] != "NVDA" || got[1] != "AAPL" {
		t.Fatalf("symbols = %+v", got)
	}
}

func providerRecord(symbol string, ticker string, contractType string, oi int64) massive.OptionContractDailyRecord {
	underlying := 100.0
	closeValue := 10.0
	return massive.OptionContractDailyRecord{
		ProviderContractID: ticker,
		OptionTicker:       ticker,
		UnderlyingSymbol:   symbol,
		ContractType:       contractType,
		ExpirationDate:     time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC),
		StrikePrice:        100,
		ObservationDate:    time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC),
		Close:              &closeValue,
		OpenInterest:       &oi,
		UnderlyingClose:    &underlying,
		Raw:                map[string]any{"ticker": ticker},
	}
}

func TestRunCoveragePersistsAnalyticsReadyCapture(t *testing.T) {
	session := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	provider := &fakeProvider{recordsBySymbol: map[string][]massive.OptionContractDailyRecord{
		"AAPL": {
			surfaceProviderRecord("AAPL", "O:AAPL30ATM", "call", session, 30, .50),
			surfaceProviderRecord("AAPL", "O:AAPL60ATM", "put", session, 60, -.50),
			surfaceProviderRecord("AAPL", "O:AAPL90ATM", "call", session, 90, .50),
			surfaceProviderRecord("AAPL", "O:AAPL30P25", "put", session, 30, -.25),
			surfaceProviderRecord("AAPL", "O:AAPL30C25", "call", session, 30, .25),
			surfaceProviderRecord("AAPL", "O:AAPL30DEEP", "call", session, 30, .90),
		},
	}}
	repo := &fakeRepo{normalizedEvents: []storage.NormalizedEventLedgerRecord{equityEvent(session)}}
	result, err := runCoverage(context.Background(), provider, repo, cliConfig{TenantID: "tenant-local", Symbols: []string{"AAPL"}, MaxSymbols: 1, Limit: 10, MaxPages: 1, SessionDate: session, RunID: "g142-test"})
	if err != nil {
		t.Fatalf("runCoverage: %v", err)
	}
	if result.AnalyticsReady != 1 || result.Partial != 0 || len(repo.captures) != 1 {
		t.Fatalf("result=%+v captures=%+v", result, repo.captures)
	}
	if result.Fetched != 6 || result.SelectedEvidence != 5 || result.DiscardedCandidates != 1 || len(repo.chain) != 5 || result.Symbols[0].DistributionContractCount != 6 {
		t.Fatalf("bounded result=%+v chain=%d", result, len(repo.chain))
	}
	if provider.filter.ExpirationDateGTE != session.AddDate(0, 0, 14) || provider.filter.ExpirationDateLTE != session.AddDate(0, 0, 120) || provider.filter.StrikePriceGTE == nil || *provider.filter.StrikePriceGTE != 70 || provider.filter.StrikePriceLTE == nil || *provider.filter.StrikePriceLTE != 130 {
		t.Fatalf("provider filter = %+v", provider.filter)
	}
	capture := repo.captures[0]
	if !capture.AnalyticsReady || capture.Status != storage.MarketOpsOptionsCaptureAnalyticsReady || capture.RequiredSurfaceCells != 5 || capture.CaptureID != optionsCaptureID("tenant-local", "src-massive", "AAPL", session) {
		t.Fatalf("capture = %+v", capture)
	}
}

func TestRunCoverageRejectsProviderDateBeforeWritesAndRecordsFailure(t *testing.T) {
	session := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	provider := &fakeProvider{recordsBySymbol: map[string][]massive.OptionContractDailyRecord{
		"AAPL": {surfaceProviderRecord("AAPL", "O:AAPL30ATM", "call", session.AddDate(0, 0, 1), 30, .50)},
	}}
	repo := &fakeRepo{normalizedEvents: []storage.NormalizedEventLedgerRecord{equityEvent(session)}}
	result, err := runCoverage(context.Background(), provider, repo, cliConfig{TenantID: "tenant-local", Symbols: []string{"AAPL"}, MaxSymbols: 1, SessionDate: session, RunID: "g142-test", ContinueOnError: true})
	if err != nil {
		t.Fatalf("runCoverage: %v", err)
	}
	if result.Failed != 1 || len(repo.chain) != 0 || len(repo.captures) != 1 || repo.captures[0].Status != storage.MarketOpsOptionsCaptureFailed {
		t.Fatalf("result=%+v chain=%d captures=%+v", result, len(repo.chain), repo.captures)
	}
}

func TestRunCoverageSkipsExistingAnalyticsReadyCapture(t *testing.T) {
	session := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	captureID := optionsCaptureID("tenant-local", "src-massive", "AAPL", session)
	repo := &fakeRepo{captures: []storage.MarketOpsOptionsCaptureRecord{{CaptureID: captureID, TenantID: "tenant-local", AnalyticsReady: true}}}
	provider := &fakeProvider{recordsBySymbol: map[string][]massive.OptionContractDailyRecord{}}
	result, err := runCoverage(context.Background(), provider, repo, cliConfig{TenantID: "tenant-local", Symbols: []string{"AAPL"}, MaxSymbols: 1, SessionDate: session, SkipComplete: true, RunID: "g142-test"})
	if err != nil {
		t.Fatalf("runCoverage: %v", err)
	}
	if result.SkippedComplete != 1 || len(provider.calls) != 0 || len(repo.captures) != 1 {
		t.Fatalf("result=%+v calls=%+v captures=%d", result, provider.calls, len(repo.captures))
	}
}

func TestRunCoverageRequiresCanonicalCloseBeforeProviderAcquisition(t *testing.T) {
	session := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	provider := &fakeProvider{recordsBySymbol: map[string][]massive.OptionContractDailyRecord{}}
	repo := &fakeRepo{}
	result, err := runCoverage(context.Background(), provider, repo, cliConfig{
		TenantID: "tenant-local", Symbols: []string{"AAPL"}, MaxSymbols: 1,
		SessionDate: session, RunID: "g142-missing-equity", ContinueOnError: true,
	})
	if err != nil {
		t.Fatalf("runCoverage: %v", err)
	}
	if result.Failed != 1 || len(provider.calls) != 0 || len(repo.chain) != 0 || len(repo.captures) != 1 {
		t.Fatalf("result=%+v provider_calls=%v chain=%d captures=%d", result, provider.calls, len(repo.chain), len(repo.captures))
	}
	if !strings.Contains(repo.captures[0].ErrorMessage, "canonical same-session underlying close is required") {
		t.Fatalf("capture = %+v", repo.captures[0])
	}
}

func equityEvent(session time.Time) storage.NormalizedEventLedgerRecord {
	return storage.NormalizedEventLedgerRecord{EventID: "evt-aapl-equity", ObservationTime: session, NormalizedPayload: []byte(`{"symbol":"AAPL","close":100}`)}
}

func surfaceProviderRecord(symbol, ticker, contractType string, session time.Time, dte int, delta float64) massive.OptionContractDailyRecord {
	record := providerRecord(symbol, ticker, contractType, 100)
	iv, gamma, theta, vega := .30, .02, -.01, .10
	record.ObservationDate = session
	record.ExpirationDate = session.AddDate(0, 0, dte)
	record.ImpliedVolatility = &iv
	record.Delta = &delta
	record.Gamma = &gamma
	record.Theta = &theta
	record.Vega = &vega
	return record
}

func TestRunCoverageEnrichesMissingUnderlyingFromCanonicalEquityEvent(t *testing.T) {
	session := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	records := []massive.OptionContractDailyRecord{
		surfaceProviderRecord("AAPL", "O:AAPL30ATM", "call", session, 30, .50),
		surfaceProviderRecord("AAPL", "O:AAPL60ATM", "put", session, 60, -.50),
		surfaceProviderRecord("AAPL", "O:AAPL90ATM", "call", session, 90, .50),
		surfaceProviderRecord("AAPL", "O:AAPL30P25", "put", session, 30, -.25),
		surfaceProviderRecord("AAPL", "O:AAPL30C25", "call", session, 30, .25),
	}
	for index := range records {
		records[index].UnderlyingClose = nil
	}
	provider := &fakeProvider{recordsBySymbol: map[string][]massive.OptionContractDailyRecord{"AAPL": records}}
	repo := &fakeRepo{normalizedEvents: []storage.NormalizedEventLedgerRecord{{EventID: "evt-aapl-equity", ObservationTime: session, NormalizedPayload: []byte(`{"symbol":"AAPL","close":225.5}`)}}}
	result, err := runCoverage(context.Background(), provider, repo, cliConfig{TenantID: "tenant-local", Symbols: []string{"AAPL"}, MaxSymbols: 1, SessionDate: session, RunID: "g142-test", DryRun: true})
	if err != nil {
		t.Fatalf("runCoverage: %v", err)
	}
	if result.AnalyticsReady != 1 || len(result.Symbols) != 1 || result.Symbols[0].UnderlyingPriceSourceEventID != "evt-aapl-equity" {
		t.Fatalf("result = %+v", result)
	}
}
