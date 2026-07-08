package postgres

import (
	"context"
	"errors"
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

func TestValidateIdempotencyRecord(t *testing.T) {
	record := validIdempotencyRecord()
	if err := validateIdempotencyRecord(record); err != nil {
		t.Fatalf("validate idempotency: %v", err)
	}
	record.IdempotencyKey = ""
	if err := validateIdempotencyRecord(record); err == nil {
		t.Fatal("expected idempotency key validation error")
	}
}

func TestValidateRawEventLedger(t *testing.T) {
	record := validRawEventLedgerRecord()
	if err := validateRawEventLedger(record); err != nil {
		t.Fatalf("validate raw event ledger: %v", err)
	}
	record.PayloadJSON = nil
	if err := validateRawEventLedger(record); err == nil {
		t.Fatal("expected payload validation error")
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
	ledger := validRawEventLedgerRecord()
	ledger.EventID = "test-g028-event"
	ledger.IdempotencyKey = "test-g028-idem"
	if err := repo.UpsertRawEventLedger(ctx, ledger); err != nil {
		t.Fatalf("upsert raw event ledger: %v", err)
	}
	idem := validIdempotencyRecord()
	idem.EventID = ledger.EventID
	idem.IdempotencyKey = ledger.IdempotencyKey
	if err := repo.UpsertIdempotencyRecord(ctx, idem); err != nil {
		t.Fatalf("upsert idempotency: %v", err)
	}

	var status string
	var built int
	if err := repo.db.QueryRowContext(ctx, "SELECT status, events_built FROM scheduler_runs WHERE run_id = $1", run.RunID).Scan(&status, &built); err != nil {
		t.Fatalf("query scheduler run: %v", err)
	}
	if status != storage.RunStatusSucceeded || built != 3 {
		t.Fatalf("status/built = %s/%d", status, built)
	}
	var idemStatus string
	var topic string
	if err := repo.db.QueryRowContext(ctx, "SELECT status, topic FROM idempotency_records WHERE tenant_id = $1 AND source_id = $2 AND idempotency_key = $3", idem.TenantID, idem.SourceID, idem.IdempotencyKey).Scan(&idemStatus, &topic); err != nil {
		t.Fatalf("query idempotency record: %v", err)
	}
	if idemStatus != storage.IdempotencyStatusPublished || topic != "signalops.test.raw.v1" {
		t.Fatalf("idempotency status/topic = %s/%s", idemStatus, topic)
	}
	var ledgerDataset string
	var ledgerOffset int64
	if err := repo.db.QueryRowContext(ctx, "SELECT dataset, broker_offset FROM raw_event_ledger WHERE event_id = $1", ledger.EventID).Scan(&ledgerDataset, &ledgerOffset); err != nil {
		t.Fatalf("query raw event ledger: %v", err)
	}
	if ledgerDataset != "equity_eod_prices" || ledgerOffset != 42 {
		t.Fatalf("ledger dataset/offset = %s/%d", ledgerDataset, ledgerOffset)
	}
	runs, err := repo.ListSchedulerRuns(ctx, 5)
	if err != nil {
		t.Fatalf("list scheduler runs: %v", err)
	}
	if len(runs) == 0 {
		t.Fatal("expected scheduler runs")
	}
	gotRun, err := repo.GetSchedulerRun(ctx, run.RunID)
	if err != nil {
		t.Fatalf("get scheduler run: %v", err)
	}
	if gotRun.RunID != run.RunID || len(gotRun.Datasets) != 1 || gotRun.Datasets[0] != "equity_eod_prices" {
		t.Fatalf("got scheduler run = %+v", gotRun)
	}
	usageRows, err := repo.ListProviderUsage(ctx, run.RunID, 5)
	if err != nil {
		t.Fatalf("list provider usage: %v", err)
	}
	if len(usageRows) == 0 || usageRows[0].RunID != run.RunID {
		t.Fatalf("provider usage rows = %+v", usageRows)
	}
	ledgerRows, err := repo.ListRawEventLedger(ctx, storage.RawEventLedgerFilter{TenantID: ledger.TenantID, SourceID: ledger.SourceID, Dataset: ledger.Dataset, Limit: 5})
	if err != nil {
		t.Fatalf("list raw event ledger: %v", err)
	}
	if len(ledgerRows) == 0 {
		t.Fatal("expected raw event ledger rows")
	}
	gotLedger, err := repo.GetRawEventLedger(ctx, ledger.EventID)
	if err != nil {
		t.Fatalf("get raw event ledger: %v", err)
	}
	if gotLedger.EventID != ledger.EventID || gotLedger.BrokerOffset == nil || *gotLedger.BrokerOffset != 42 {
		t.Fatalf("got raw event ledger = %+v", gotLedger)
	}
	gotID, err := repo.GetIdempotencyRecord(ctx, idem.TenantID, idem.SourceID, idem.IdempotencyKey)
	if err != nil {
		t.Fatalf("get idempotency record: %v", err)
	}
	if gotID.EventID != ledger.EventID || gotID.Status != storage.IdempotencyStatusPublished {
		t.Fatalf("got idempotency = %+v", gotID)
	}
	if _, err := repo.GetSchedulerRun(ctx, "missing-g029-run"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("missing scheduler run error = %v", err)
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

func validIdempotencyRecord() storage.IdempotencyRecord {
	partition := int32(2)
	offset := int64(42)
	return storage.IdempotencyRecord{
		TenantID:       "tenant-1",
		SourceID:       "src-massive",
		IdempotencyKey: "idem-1",
		EventID:        "event-1",
		SourceAdapter:  "market_data.massive",
		Dataset:        "equity_eod_prices",
		Topic:          "signalops.test.raw.v1",
		Partition:      &partition,
		Offset:         &offset,
		PayloadHash:    "sha256:abc123",
		Status:         storage.IdempotencyStatusPublished,
		MetadataJSON:   []byte(`{"route":"massive_scheduled_pull"}`),
	}
}

func validRawEventLedgerRecord() storage.RawEventLedgerRecord {
	partition := int32(2)
	offset := int64(42)
	return storage.RawEventLedgerRecord{
		EventID:         "event-1",
		TenantID:        "tenant-1",
		SourceID:        "src-massive",
		SourceAdapter:   "market_data.massive",
		Dataset:         "equity_eod_prices",
		IdempotencyKey:  "idem-1",
		ObservationTime: time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC),
		ProcessingTime:  time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC),
		BrokerTopic:     "signalops.test.raw.v1",
		BrokerPartition: &partition,
		BrokerOffset:    &offset,
		PayloadJSON:     []byte(`{"event_id":"event-1"}`),
		EntityHintsJSON: []byte(`[]`),
	}
}
