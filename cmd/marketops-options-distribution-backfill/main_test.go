package main

import (
	"context"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

type fakeRepo struct {
	chain         []storage.MarketOpsOptionsChainRecord
	distributions []storage.MarketOpsOptionsDistributionRecord
	filter        storage.MarketOpsOptionsChainFilter
}

func (f *fakeRepo) ListMarketOpsOptionsChain(_ context.Context, filter storage.MarketOpsOptionsChainFilter) ([]storage.MarketOpsOptionsChainRecord, error) {
	f.filter = filter
	return append([]storage.MarketOpsOptionsChainRecord{}, f.chain...), nil
}

func (f *fakeRepo) UpsertMarketOpsOptionsDistribution(_ context.Context, record storage.MarketOpsOptionsDistributionRecord) error {
	f.distributions = append(f.distributions, record)
	return nil
}

func TestBackfillWritesDistributionForEachTradeDate(t *testing.T) {
	repo := &fakeRepo{chain: []storage.MarketOpsOptionsChainRecord{
		chainRow("2026-07-15", "call", 100), chainRow("2026-07-15", "put", 50),
		chainRow("2026-07-16", "call", 200), chainRow("2026-07-16", "put", 100),
		chainRow("2026-07-17", "call", 300), chainRow("2026-07-17", "put", 100),
	}}
	result, err := backfill(context.Background(), repo, cliConfig{TenantID: "tenant-local", Symbol: "nvda", WindowDays: 10, RunID: "run-1"})
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if result.ChainRowsScanned != 6 || result.TradeDatesScanned != 3 || result.DistributionsBuilt != 3 || result.DistributionsUpserted != 3 {
		t.Fatalf("result = %+v", result)
	}
	if result.FirstTradeDate != "2026-07-15" || result.LastTradeDate != "2026-07-17" {
		t.Fatalf("date range = %+v", result)
	}
	if repo.filter.Symbol != "NVDA" || repo.filter.Limit != 5000 {
		t.Fatalf("filter = %+v", repo.filter)
	}
	latest := repo.distributions[len(repo.distributions)-1]
	if latest.TradeDate.Format("2006-01-02") != "2026-07-17" || latest.CallPutOpenInterestRatio != 3 {
		t.Fatalf("latest distribution = %+v", latest)
	}
}

func TestBackfillDryRunSkipsWrites(t *testing.T) {
	repo := &fakeRepo{chain: []storage.MarketOpsOptionsChainRecord{chainRow("2026-07-17", "call", 100)}}
	result, err := backfill(context.Background(), repo, cliConfig{TenantID: "tenant-local", Symbol: "NVDA", DryRun: true, RunID: "run-1"})
	if err != nil {
		t.Fatalf("backfill dry run: %v", err)
	}
	if result.DistributionsBuilt != 1 || result.DistributionsUpserted != 0 || len(repo.distributions) != 0 {
		t.Fatalf("dry-run result = %+v writes=%d", result, len(repo.distributions))
	}
}

func chainRow(date string, contractType string, oi int64) storage.MarketOpsOptionsChainRecord {
	parsed, _ := time.Parse("2006-01-02", date)
	underlying := 100.0
	return storage.MarketOpsOptionsChainRecord{TenantID: "tenant-local", Symbol: "NVDA", TradeDate: parsed, OptionTicker: "O:NVDA" + date + contractType, ContractType: contractType, ExpirationDate: parsed.AddDate(0, 0, 30), StrikePrice: 100, UnderlyingClose: &underlying, OpenInterest: &oi}
}
