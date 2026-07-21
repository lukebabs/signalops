package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestValidateMarketOpsOptionsCapture(t *testing.T) {
	now := time.Date(2026, 7, 20, 20, 0, 0, 0, time.UTC)
	record := storage.MarketOpsOptionsCaptureRecord{
		CaptureID: "optcap_test", TenantID: "tenant-local", Symbol: "AAPL", SessionDate: now,
		RunID: "g142-test", Status: storage.MarketOpsOptionsCaptureAnalyticsReady, AnalyticsReady: true,
		RequiredSurfaceCells: 7, StartedAt: now, CompletedAt: now.Add(time.Minute),
	}
	if err := validateMarketOpsOptionsCapture(record); err != nil {
		t.Fatalf("validate capture: %v", err)
	}
	record.AnalyticsReady = false
	if err := validateMarketOpsOptionsCapture(record); err == nil {
		t.Fatal("expected readiness/status mismatch")
	}
	record.Status = storage.MarketOpsOptionsCapturePartial
	record.RequiredSurfaceCells = 8
	if err := validateMarketOpsOptionsCapture(record); err == nil {
		t.Fatal("expected invalid surface count")
	}
}

func TestMarketOpsOptionsCaptureAgainstPostgres(t *testing.T) {
	if os.Getenv("SIGNALOPS_POSTGRES_INTEGRATION") != "1" {
		t.Skip("set SIGNALOPS_POSTGRES_INTEGRATION=1 to run")
	}
	dsn := os.Getenv("SIGNALOPS_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://signalops:signalops@localhost:15432/signalops?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	repo, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	defer repo.Close()
	defer repo.db.ExecContext(context.Background(), "DELETE FROM marketops_options_capture_sessions WHERE tenant_id=$1", "test-g142")

	now := time.Date(2026, 7, 20, 20, 0, 0, 0, time.UTC)
	record := storage.MarketOpsOptionsCaptureRecord{
		CaptureID: "optcap_integration", TenantID: "test-g142", Symbol: "AAPL", SessionDate: now,
		Provider: "massive", SourceID: "src-massive", RunID: "g142-integration",
		Status: storage.MarketOpsOptionsCapturePartial, RequiredSurfaceCells: 4,
		ContractCount: 500, UsableIVCount: 450, UsableGreeksCount: 440, OpenInterestCount: 500,
		QualityReasonsJSON: []byte(`["missing_required_surface_cells:atm_90d"]`), MetricsJSON: []byte(`{"fetched":500}`),
		StartedAt: now, CompletedAt: now.Add(time.Minute),
	}
	if err := repo.UpsertMarketOpsOptionsCapture(ctx, record); err != nil {
		t.Fatalf("upsert capture: %v", err)
	}
	record.RunID = "g142-integration-rerun"
	if err := repo.UpsertMarketOpsOptionsCapture(ctx, record); err != nil {
		t.Fatalf("upsert capture rerun: %v", err)
	}
	got, err := repo.GetMarketOpsOptionsCapture(ctx, record.TenantID, record.CaptureID)
	if err != nil {
		t.Fatalf("get capture: %v", err)
	}
	if got.AttemptCount != 2 || got.RunID != record.RunID || got.RequiredSurfaceCells != 4 {
		t.Fatalf("capture = %+v", got)
	}
	ready := false
	rows, err := repo.ListMarketOpsOptionsCaptures(ctx, storage.MarketOpsOptionsCaptureFilter{TenantID: record.TenantID, Symbol: "AAPL", Ready: &ready, Limit: 10})
	if err != nil || len(rows) != 1 {
		t.Fatalf("list captures rows=%+v err=%v", rows, err)
	}
}
