package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestNewRequiresDB(t *testing.T) {
	_, err := New(nil)
	if err == nil {
		t.Fatal("expected db error")
	}
}

func TestValidateSchedulerRun(t *testing.T) {
	record := validSchedulerRunRecord()
	if err := validateSchedulerRun(record); err != nil {
		t.Fatalf("validate scheduler run: %v", err)
	}
	record.RunID = ""
	if err := validateSchedulerRun(record); err == nil {
		t.Fatal("expected run id validation error")
	}
}

func TestValidateProviderUsage(t *testing.T) {
	record := validProviderUsageRecord()
	if err := validateProviderUsage(record); err != nil {
		t.Fatalf("validate provider usage: %v", err)
	}
	record.Provider = ""
	if err := validateProviderUsage(record); err == nil {
		t.Fatal("expected provider validation error")
	}
}

func TestStringArrayValue(t *testing.T) {
	value, err := stringArray([]string{"equity_eod_prices", "options_contracts_daily"}).Value()
	if err != nil {
		t.Fatalf("array value: %v", err)
	}
	if value != `{"equity_eod_prices","options_contracts_daily"}` {
		t.Fatalf("array value = %v", value)
	}
	escaped, err := stringArray([]string{`quote"and\slash`}).Value()
	if err != nil {
		t.Fatalf("escaped array value: %v", err)
	}
	if escaped != `{"quote\"and\\slash"}` {
		t.Fatalf("escaped array value = %v", escaped)
	}
}

func TestRepositoryAgainstPostgres(t *testing.T) {
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

	run := validSchedulerRunRecord()
	run.RunID = "test-g027-run"
	if err := repo.UpsertSchedulerRun(ctx, run); err != nil {
		t.Fatalf("upsert started run: %v", err)
	}
	completed := run.StartedAt.Add(time.Minute)
	run.CompletedAt = &completed
	run.Status = storage.RunStatusSucceeded
	run.EventsBuilt = 3
	run.EventsPublished = 2
	run.ProviderRequests = 4
	run.ReportJSON = []byte(`{"ok":true}`)
	if err := repo.UpsertSchedulerRun(ctx, run); err != nil {
		t.Fatalf("upsert completed run: %v", err)
	}
	usage := validProviderUsageRecord()
	usage.RunID = run.RunID
	usage.UsageID = "test-g027-usage"
	if err := repo.InsertProviderUsage(ctx, usage); err != nil {
		t.Fatalf("insert usage: %v", err)
	}

	var status string
	var built int
	if err := repo.db.QueryRowContext(ctx, "SELECT status, events_built FROM scheduler_runs WHERE run_id = $1", run.RunID).Scan(&status, &built); err != nil {
		t.Fatalf("query scheduler run: %v", err)
	}
	if status != storage.RunStatusSucceeded || built != 3 {
		t.Fatalf("status/built = %s/%d", status, built)
	}
}

func validSchedulerRunRecord() storage.SchedulerRunRecord {
	return storage.SchedulerRunRecord{
		RunID:            "run-1",
		TenantID:         "tenant-1",
		SourceID:         "src-massive",
		SourceAdapter:    "market_data.massive",
		Datasets:         []string{"equity_eod_prices"},
		ObservationDate:  time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC),
		DryRun:           true,
		Status:           storage.RunStatusStarted,
		StartedAt:        time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC),
		ConfigJSON:       []byte(`{"dry_run":true}`),
		ReportJSON:       []byte(`{}`),
		EventsBuilt:      1,
		ProviderRequests: 1,
	}
}

func validProviderUsageRecord() storage.ProviderUsageRecord {
	return storage.ProviderUsageRecord{
		UsageID:      "usage-1",
		RunID:        "run-1",
		Provider:     "massive",
		Dataset:      "equity_eod_prices",
		RequestCount: 1,
		RetryCount:   0,
		EventCount:   1,
		BudgetJSON:   []byte(`{"max_provider_requests":2}`),
	}
}
