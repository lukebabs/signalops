package main

import (
	"context"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/adapters/marketdata/massive"
	"github.com/lukebabs/signalops/internal/storage"
)

type fakeProvider struct {
	records []massive.OptionContractDailyRecord
	limit   int
	pages   int
	symbol  string
}

func (f *fakeProvider) ListOptionChainSnapshot(_ context.Context, underlying string, limit int, maxPages int) ([]massive.OptionContractDailyRecord, error) {
	f.symbol = underlying
	f.limit = limit
	f.pages = maxPages
	return f.records, nil
}

type fakeRepo struct {
	chain         []storage.MarketOpsOptionsChainRecord
	distributions []storage.MarketOpsOptionsDistributionRecord
	listedFilter  storage.MarketOpsOptionsChainFilter
}

func (f *fakeRepo) UpsertMarketOpsOptionsChain(_ context.Context, record storage.MarketOpsOptionsChainRecord) error {
	f.chain = append(f.chain, record)
	return nil
}

func (f *fakeRepo) ListMarketOpsOptionsChain(_ context.Context, filter storage.MarketOpsOptionsChainFilter) ([]storage.MarketOpsOptionsChainRecord, error) {
	f.listedFilter = filter
	return append([]storage.MarketOpsOptionsChainRecord{}, f.chain...), nil
}

func (f *fakeRepo) UpsertMarketOpsOptionsDistribution(_ context.Context, record storage.MarketOpsOptionsDistributionRecord) error {
	f.distributions = append(f.distributions, record)
	return nil
}

func TestIngestWritesChainRowsAndDistribution(t *testing.T) {
	callOI := int64(200)
	putOI := int64(100)
	underlying := 170.0
	provider := &fakeProvider{records: []massive.OptionContractDailyRecord{
		providerRecord("O:NVDA260116C00100000", "call", callOI, underlying),
		providerRecord("O:NVDA260116P00100000", "put", putOI, underlying),
	}}
	repo := &fakeRepo{}
	result, err := ingest(context.Background(), provider, repo, cliConfig{TenantID: "tenant-local", Symbol: "nvda", Limit: 10, MaxPages: 2, WindowDays: 10, RunID: "run-1"})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if provider.symbol != "NVDA" || provider.limit != 10 || provider.pages != 2 {
		t.Fatalf("provider args = %s/%d/%d", provider.symbol, provider.limit, provider.pages)
	}
	if result.Fetched != 2 || result.Converted != 2 || result.ChainUpserted != 2 || !result.DistributionWritten {
		t.Fatalf("result = %+v", result)
	}
	if len(repo.chain) != 2 || len(repo.distributions) != 1 {
		t.Fatalf("writes = %d/%d", len(repo.chain), len(repo.distributions))
	}
	distribution := repo.distributions[0]
	if distribution.Symbol != "NVDA" || distribution.ContractCount != 2 || distribution.CallPutOpenInterestRatio != 2 {
		t.Fatalf("distribution = %+v", distribution)
	}
	if repo.listedFilter.Symbol != "NVDA" || repo.listedFilter.Limit != 100000 {
		t.Fatalf("listed filter = %+v", repo.listedFilter)
	}
}

func TestIngestDryRunSkipsWritesButBuildsDistributionMetrics(t *testing.T) {
	callOI := int64(300)
	underlying := 150.0
	provider := &fakeProvider{records: []massive.OptionContractDailyRecord{providerRecord("O:NVDA260116C00100000", "call", callOI, underlying)}}
	repo := &fakeRepo{}
	result, err := ingest(context.Background(), provider, repo, cliConfig{TenantID: "tenant-local", Symbol: "NVDA", DryRun: true, RunID: "run-1"})
	if err != nil {
		t.Fatalf("ingest dry run: %v", err)
	}
	if result.ChainUpserted != 0 || result.DistributionWritten || len(repo.chain) != 0 || len(repo.distributions) != 0 {
		t.Fatalf("dry-run writes = %+v chain=%d distributions=%d", result, len(repo.chain), len(repo.distributions))
	}
	if result.DistributionContractCount != 1 || result.TradeDate != "2026-07-17" {
		t.Fatalf("dry-run metrics = %+v", result)
	}
}

func providerRecord(ticker string, contractType string, oi int64, underlying float64) massive.OptionContractDailyRecord {
	closeValue := 10.0
	return massive.OptionContractDailyRecord{
		ProviderContractID: ticker,
		OptionTicker:       ticker,
		UnderlyingSymbol:   "NVDA",
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
