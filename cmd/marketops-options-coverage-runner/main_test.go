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
}

func (f *fakeProvider) ListOptionChainSnapshot(_ context.Context, underlying string, limit int, maxPages int) ([]massive.OptionContractDailyRecord, error) {
	f.calls = append(f.calls, underlying)
	f.limit = limit
	f.pages = maxPages
	return f.recordsBySymbol[underlying], nil
}

type fakeRepo struct {
	assets           []storage.MarketOpsAssetRecord
	chain            []storage.MarketOpsOptionsChainRecord
	distributions    []storage.MarketOpsOptionsDistributionRecord
	events           []storage.NormalizedEventLedgerRecord
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
