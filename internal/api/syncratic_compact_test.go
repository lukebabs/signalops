package api

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestCompactSyncraticSignalsRetainsHighestRankedTwelve(t *testing.T) {
	now := time.Date(2026, 7, 29, 15, 0, 0, 0, time.UTC)
	records := make([]storage.SignalLedgerRecord, 0, 13)
	for i := 0; i < 12; i++ {
		records = append(records, storage.SignalLedgerRecord{SignalID: "sig-low-" + strconv.Itoa(i), Severity: "low", Confidence: 0.5, SignalTime: now.Add(-time.Duration(i+1) * time.Minute), EntitiesJSON: []byte(`[{"symbol":"NVS"}]`)})
	}
	records = append(records, storage.SignalLedgerRecord{SignalID: "sig-critical", Severity: "critical", Confidence: 0.1, SignalTime: now.Add(-2 * time.Hour), EntitiesJSON: []byte(`[{"symbol":"NVS"}]`)})

	selected, available := compactSyncraticSignals("NVS", records, now.Add(-24*time.Hour), now, nil)
	if available != 13 || len(selected) != maxSyncraticSignals {
		t.Fatalf("available=%d selected=%d", available, len(selected))
	}
	if selected[0].SignalID != "sig-critical" {
		t.Fatalf("highest-severity signal must be first: %#v", selected[0])
	}
	for _, record := range selected {
		if record.SignalID == "sig-low-11" {
			t.Fatal("oldest low-severity signal should be omitted at the retention cap")
		}
	}
}

func TestMaterializeSyncraticContextsLoadsLedgerOnceForBatch(t *testing.T) {
	now := time.Date(2026, 7, 29, 15, 0, 0, 0, time.UTC)
	repo := &fakeQueryRepository{
		marketOpsAssets: []storage.MarketOpsAssetRecord{{TenantID: "tenant-1", UniverseGroup: "top50_megacap", Ticker: "AAPL", IsActive: true}, {TenantID: "tenant-1", UniverseGroup: "top50_megacap", Ticker: "MSFT", IsActive: true}},
		signals: []storage.SignalLedgerRecord{
			{SignalID: "sig-aapl", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SignalTime: now.Add(-time.Hour), EntitiesJSON: []byte(`[{"symbol":"AAPL"}]`)},
			{SignalID: "sig-msft", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SignalTime: now.Add(-time.Hour), EntitiesJSON: []byte(`[{"symbol":"MSFT"}]`)},
		},
	}
	_, err := materializeSyncraticContexts(context.Background(), repo, syncraticMaterializeRequest{TenantID: "tenant-1", WindowStart: now.Add(-24 * time.Hour).Format(time.RFC3339), WindowEnd: now.Format(time.RFC3339), MaxAssets: 2, MaxCandidateWindows: 2, MaxContextWindows: 2, MaxInsights: 2, MinEvidenceCount: 1, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if repo.signalLedgerQueries != 1 || repo.alertLedgerQueries != 1 {
		t.Fatalf("ledger queries signals=%d alerts=%d; want one each", repo.signalLedgerQueries, repo.alertLedgerQueries)
	}
	if !repo.lastSignalLedgerFilter.WindowStart.Equal(now.Add(-24*time.Hour)) || !repo.lastSignalLedgerFilter.WindowEnd.Equal(now) {
		t.Fatalf("signal filter window=%+v", repo.lastSignalLedgerFilter)
	}
}
