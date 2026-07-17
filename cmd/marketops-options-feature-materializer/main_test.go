package main

import (
	"context"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

type fakeRepo struct {
	distributions []storage.MarketOpsOptionsDistributionRecord
	events        []storage.NormalizedEventLedgerRecord
	filter        storage.MarketOpsOptionsDistributionFilter
}

func (r *fakeRepo) ListMarketOpsOptionsDistributions(_ context.Context, filter storage.MarketOpsOptionsDistributionFilter) ([]storage.MarketOpsOptionsDistributionRecord, error) {
	r.filter = filter
	return r.distributions, nil
}

func (r *fakeRepo) UpsertNormalizedEventLedger(_ context.Context, record storage.NormalizedEventLedgerRecord) error {
	r.events = append(r.events, record)
	return nil
}

func TestMaterializeWritesNormalizedFeatureEvents(t *testing.T) {
	tradeDate := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	repo := &fakeRepo{distributions: []storage.MarketOpsOptionsDistributionRecord{{TenantID: "tenant-local", Symbol: "NVDA", TradeDate: tradeDate, WindowName: "10_trade_days", CallPutOpenInterestRatio: 1.5}}}

	result, err := materialize(context.Background(), repo, cliConfig{TenantID: "tenant-local", Symbol: "nvda", WindowName: "10_trade_days", Limit: 5, RunID: "optfeat-test"})
	if err != nil {
		t.Fatalf("materialize error = %v", err)
	}
	if result.Scanned != 1 || result.Upserted != 1 {
		t.Fatalf("metrics = %+v", result)
	}
	if repo.filter.Symbol != "NVDA" || repo.filter.Limit != 5 {
		t.Fatalf("filter = %+v", repo.filter)
	}
	if len(repo.events) != 1 || repo.events[0].Dataset != "options_distribution_daily" {
		t.Fatalf("events = %+v", repo.events)
	}
}

func TestMaterializeDryRunSkipsWrites(t *testing.T) {
	tradeDate := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	repo := &fakeRepo{distributions: []storage.MarketOpsOptionsDistributionRecord{{TenantID: "tenant-local", Symbol: "NVDA", TradeDate: tradeDate, WindowName: "10_trade_days", CallPutOpenInterestRatio: 1.5}}}

	result, err := materialize(context.Background(), repo, cliConfig{TenantID: "tenant-local", Symbol: "NVDA", WindowName: "10_trade_days", Limit: 5, DryRun: true, RunID: "optfeat-test"})
	if err != nil {
		t.Fatalf("materialize error = %v", err)
	}
	if result.Scanned != 1 || result.Upserted != 0 || len(repo.events) != 0 {
		t.Fatalf("dry run result = %+v events=%+v", result, repo.events)
	}
}
