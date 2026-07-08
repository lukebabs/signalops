package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
	"github.com/lukebabs/signalops/pkg/broker"
)

type fakePublisher struct {
	msg broker.Message
	err error
}

func (p *fakePublisher) Publish(ctx context.Context, msg broker.Message) (broker.PublishResult, error) {
	p.msg = msg
	if p.err != nil {
		return broker.PublishResult{}, p.err
	}
	return broker.PublishResult{Topic: msg.Topic, Partition: 2, Offset: 42}, nil
}

func (p *fakePublisher) Close(ctx context.Context) error {
	return nil
}

type fakeQueryRepository struct {
	runs             []storage.SchedulerRunRecord
	usage            []storage.ProviderUsageRecord
	rawEvents        []storage.RawEventLedgerRecord
	idem             storage.IdempotencyRecord
	sources          []storage.CatalogSourceRecord
	pipelines        []storage.CatalogPipelineRecord
	rules            []storage.CatalogRuleRecord
	notFound         bool
	lastFilter       storage.RawEventLedgerFilter
	schedulerQueries int
	rawEventQueries  int
	usageQueries     int
}

func (q *fakeQueryRepository) ListSchedulerRuns(context.Context, int) ([]storage.SchedulerRunRecord, error) {
	q.schedulerQueries++
	return q.runs, nil
}

func (q *fakeQueryRepository) GetSchedulerRun(_ context.Context, runID string) (storage.SchedulerRunRecord, error) {
	if q.notFound {
		return storage.SchedulerRunRecord{}, storage.ErrNotFound
	}
	for _, run := range q.runs {
		if run.RunID == runID {
			return run, nil
		}
	}
	return storage.SchedulerRunRecord{}, storage.ErrNotFound
}

func (q *fakeQueryRepository) ListProviderUsage(context.Context, string, int) ([]storage.ProviderUsageRecord, error) {
	q.usageQueries++
	return q.usage, nil
}

func (q *fakeQueryRepository) ListRawEventLedger(_ context.Context, filter storage.RawEventLedgerFilter) ([]storage.RawEventLedgerRecord, error) {
	q.rawEventQueries++
	q.lastFilter = filter
	return q.rawEvents, nil
}

func (q *fakeQueryRepository) GetRawEventLedger(_ context.Context, eventID string) (storage.RawEventLedgerRecord, error) {
	if q.notFound {
		return storage.RawEventLedgerRecord{}, storage.ErrNotFound
	}
	for _, event := range q.rawEvents {
		if event.EventID == eventID {
			return event, nil
		}
	}
	return storage.RawEventLedgerRecord{}, storage.ErrNotFound
}

func (q *fakeQueryRepository) GetIdempotencyRecord(context.Context, string, string, string) (storage.IdempotencyRecord, error) {
	if q.notFound {
		return storage.IdempotencyRecord{}, storage.ErrNotFound
	}
	return q.idem, nil
}

func (q *fakeQueryRepository) ListCatalogSources(context.Context, string, int) ([]storage.CatalogSourceRecord, error) {
	return q.sources, nil
}

func (q *fakeQueryRepository) ListCatalogPipelines(context.Context, string, int) ([]storage.CatalogPipelineRecord, error) {
	return q.pipelines, nil
}

func (q *fakeQueryRepository) ListCatalogRules(context.Context, string, int) ([]storage.CatalogRuleRecord, error) {
	return q.rules, nil
}

func TestPostRawEventPublishesMessage(t *testing.T) {
	publisher := &fakePublisher{}
	router := NewRouter(RouterConfig{
		ServiceName: "test-gateway",
		Publisher:   publisher,
		RawTopic:    "signalops.test.raw.v1",
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/events/raw", bytes.NewBufferString(`{
		"event_id":"evt-123",
		"idempotency_key":"idem-123",
		"correlation_id":"corr-payload",
		"payload":{"symbol":"SPY"}
	}`))
	req.Header.Set("X-Correlation-ID", "corr-header")
	req.Header.Set("X-Causation-ID", "cause-header")
	req.Header.Set("X-Trace-ID", "trace-header")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if publisher.msg.Topic != "signalops.test.raw.v1" {
		t.Fatalf("topic = %q", publisher.msg.Topic)
	}
	if publisher.msg.Key != "idem-123" {
		t.Fatalf("key = %q", publisher.msg.Key)
	}
	if publisher.msg.CorrelationID != "corr-header" {
		t.Fatalf("correlation id = %q", publisher.msg.CorrelationID)
	}
	if publisher.msg.CausationID != "cause-header" {
		t.Fatalf("causation id = %q", publisher.msg.CausationID)
	}
	if publisher.msg.TraceID != "trace-header" {
		t.Fatalf("trace id = %q", publisher.msg.TraceID)
	}
	if publisher.msg.Headers["signalops_event_id"] != "evt-123" {
		t.Fatalf("event header = %q", publisher.msg.Headers["signalops_event_id"])
	}

	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("response JSON error = %v", err)
	}
	if response["status"] != "accepted" {
		t.Fatalf("response status = %v", response["status"])
	}
	if response["topic"] != "signalops.test.raw.v1" {
		t.Fatalf("response topic = %v", response["topic"])
	}
	if response["offset"].(float64) != 42 {
		t.Fatalf("response offset = %v", response["offset"])
	}
}

func TestPostRawEventRejectsInvalidJSON(t *testing.T) {
	router := NewRouter(RouterConfig{
		Publisher: &fakePublisher{},
		RawTopic:  "signalops.test.raw.v1",
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/events/raw", bytes.NewBufferString(`[]`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestPostRawEventRequiresPublisher(t *testing.T) {
	router := NewRouter(RouterConfig{RawTopic: "signalops.test.raw.v1"})

	req := httptest.NewRequest(http.MethodPost, "/v1/events/raw", bytes.NewBufferString(`{"event_id":"evt-123"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestPostRawEventHandlesPublishFailure(t *testing.T) {
	router := NewRouter(RouterConfig{
		Publisher: &fakePublisher{err: errors.New("publish failed")},
		RawTopic:  "signalops.test.raw.v1",
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/events/raw", bytes.NewBufferString(`{"event_id":"evt-123"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestGetSchedulerRuns(t *testing.T) {
	repo := &fakeQueryRepository{runs: []storage.SchedulerRunRecord{validSchedulerRunRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})

	req := httptest.NewRequest(http.MethodGet, "/v1/scheduler/runs", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var response map[string][]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("response JSON error = %v", err)
	}
	if len(response["runs"]) != 1 || response["runs"][0]["run_id"] != "run-1" {
		t.Fatalf("response = %+v", response)
	}
	config, ok := response["runs"][0]["config"].(map[string]any)
	if !ok || config["dry_run"] != true {
		t.Fatalf("config = %+v", response["runs"][0]["config"])
	}
}

func TestGetSchedulerRunNotFound(t *testing.T) {
	router := NewRouter(RouterConfig{QueryRepository: &fakeQueryRepository{notFound: true}})

	req := httptest.NewRequest(http.MethodGet, "/v1/scheduler/runs/missing", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestGetRawEventsUsesFilters(t *testing.T) {
	repo := &fakeQueryRepository{rawEvents: []storage.RawEventLedgerRecord{validRawEventLedgerRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})

	req := httptest.NewRequest(http.MethodGet, "/v1/raw-events?tenant_id=tenant-1&source_id=src-1&dataset=equity_eod_prices&limit=3", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if repo.lastFilter.TenantID != "tenant-1" || repo.lastFilter.SourceID != "src-1" || repo.lastFilter.Dataset != "equity_eod_prices" || repo.lastFilter.Limit != 3 {
		t.Fatalf("filter = %+v", repo.lastFilter)
	}
	var response map[string][]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("response JSON error = %v", err)
	}
	if response["raw_events"][0]["payload"].(map[string]any)["event_id"] != "event-1" {
		t.Fatalf("response = %+v", response)
	}
}

func TestGetCatalogSources(t *testing.T) {
	repo := &fakeQueryRepository{sources: []storage.CatalogSourceRecord{validCatalogSourceRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})

	req := httptest.NewRequest(http.MethodGet, "/v1/tenants/tenant-1/catalog/sources", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var response map[string][]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("response JSON error = %v", err)
	}
	if len(response["sources"]) != 1 || response["sources"][0]["source_id"] != "src-massive" {
		t.Fatalf("response = %+v", response)
	}
	if response["sources"][0]["source_domain"] != "market_data" {
		t.Fatalf("source domain = %+v", response["sources"][0])
	}
}

func TestGetCatalogPipelines(t *testing.T) {
	repo := &fakeQueryRepository{pipelines: []storage.CatalogPipelineRecord{validCatalogPipelineRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})

	req := httptest.NewRequest(http.MethodGet, "/v1/tenants/tenant-1/catalog/pipelines", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var response map[string][]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("response JSON error = %v", err)
	}
	if len(response["pipelines"]) != 1 || response["pipelines"][0]["pipeline_id"] != "pipeline-massive-raw-ingest" {
		t.Fatalf("response = %+v", response)
	}
	if response["pipelines"][0]["source_domain"] != "market_data" {
		t.Fatalf("source domain = %+v", response["pipelines"][0])
	}
}

func TestGetCatalogRules(t *testing.T) {
	repo := &fakeQueryRepository{rules: []storage.CatalogRuleRecord{validCatalogRuleRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})

	req := httptest.NewRequest(http.MethodGet, "/v1/tenants/tenant-1/catalog/rules", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var response map[string][]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("response JSON error = %v", err)
	}
	if len(response["rules"]) != 1 || response["rules"][0]["rule_id"] != "rule-marketdata-eod-price-quality" {
		t.Fatalf("response = %+v", response)
	}
	if response["rules"][0]["rule_type"] != "quality_check" {
		t.Fatalf("rule type = %+v", response["rules"][0])
	}
}

func TestGetIdempotencyRequiresQueryParams(t *testing.T) {
	router := NewRouter(RouterConfig{QueryRepository: &fakeQueryRepository{}})

	req := httptest.NewRequest(http.MethodGet, "/v1/idempotency?tenant_id=tenant-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestQueryRoutesRequireRepository(t *testing.T) {
	router := NewRouter(RouterConfig{})

	req := httptest.NewRequest(http.MethodGet, "/v1/raw-events", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestWriteSSEFormatsEvent(t *testing.T) {
	var buf bytes.Buffer
	err := writeSSE(&buf, sseEvent{
		Event: "raw_event",
		ID:    "event-1",
		Data:  map[string]string{"event_id": "event-1"},
	})
	if err != nil {
		t.Fatalf("writeSSE error = %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "event: raw_event\n") || !strings.Contains(got, "id: event-1\n") || !strings.Contains(got, `data: {"event_id":"event-1"}`) || !strings.HasSuffix(got, "\n\n") {
		t.Fatalf("SSE frame = %q", got)
	}
}

func TestDashboardStreamRejectsUnknownChannel(t *testing.T) {
	router := NewRouter(RouterConfig{QueryRepository: &fakeQueryRepository{}})

	req := httptest.NewRequest(http.MethodGet, "/v1/streams/dashboard?channels=bogus", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "invalid_channel") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestEmitDashboardSnapshotFiltersChannelsAndDeduplicates(t *testing.T) {
	repo := &fakeQueryRepository{
		runs:      []storage.SchedulerRunRecord{validSchedulerRunRecord()},
		rawEvents: []storage.RawEventLedgerRecord{validRawEventLedgerRecord()},
		usage:     []storage.ProviderUsageRecord{validProviderUsageRecord()},
	}
	seen := map[string]struct{}{}
	var events []sseEvent
	emit := func(event sseEvent) bool {
		events = append(events, event)
		return true
	}

	channels := streamChannelSet{"scheduler_run": true, "raw_event": true}
	if !emitDashboardSnapshot(context.Background(), repo, "test-gateway", channels, seen, emit) {
		t.Fatal("first snapshot failed")
	}
	if !emitDashboardSnapshot(context.Background(), repo, "test-gateway", channels, seen, emit) {
		t.Fatal("second snapshot failed")
	}

	if repo.schedulerQueries != 2 || repo.rawEventQueries != 2 || repo.usageQueries != 0 {
		t.Fatalf("query counts scheduler=%d raw=%d usage=%d", repo.schedulerQueries, repo.rawEventQueries, repo.usageQueries)
	}
	if len(events) != 2 {
		t.Fatalf("events = %+v", events)
	}
	if events[0].Event != "scheduler_run" || events[0].ID != "run-1" {
		t.Fatalf("event[0] = %+v", events[0])
	}
	if events[1].Event != "raw_event" || events[1].ID != "event-1" {
		t.Fatalf("event[1] = %+v", events[1])
	}
}

func TestDashboardStreamNoRepositoryEmitsSSEError(t *testing.T) {
	router := NewRouter(RouterConfig{ServiceName: "test-gateway"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/v1/streams/dashboard?channels=health", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		router.ServeHTTP(rec, req)
		close(done)
	}()

	deadline := time.After(500 * time.Millisecond)
	for {
		if strings.Contains(rec.Body.String(), "storage_unavailable") {
			cancel()
			<-done
			break
		}
		select {
		case <-deadline:
			cancel()
			<-done
			t.Fatalf("stream body = %q", rec.Body.String())
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("content type = %q", got)
	}
}

func validCatalogSourceRecord() storage.CatalogSourceRecord {
	return storage.CatalogSourceRecord{
		TenantID:       "tenant-1",
		SourceID:       "src-massive",
		SourceDomain:   "market_data",
		SourceAdapter:  "market_data.massive",
		DisplayName:    "Massive Market Data",
		Description:    "Scheduled Massive market-data source.",
		Status:         storage.CatalogSourceStatusActive,
		IngestionModes: []string{"scheduled_pull"},
		Datasets:       []string{"equity_eod_prices", "option_contracts_daily"},
		MetadataJSON:   []byte(`{"provider":"massive"}`),
		CreatedAt:      time.Date(2026, 7, 8, 0, 0, 3, 0, time.UTC),
		UpdatedAt:      time.Date(2026, 7, 8, 0, 0, 4, 0, time.UTC),
	}
}

func validCatalogPipelineRecord() storage.CatalogPipelineRecord {
	return storage.CatalogPipelineRecord{
		TenantID:      "tenant-1",
		PipelineID:    "pipeline-massive-raw-ingest",
		SourceID:      "src-massive",
		SourceDomain:  "market_data",
		PipelineName:  "Massive Raw Ingest",
		Description:   "Scheduled Massive market-data raw ingest pipeline.",
		Status:        storage.CatalogPipelineStatusActive,
		Stages:        []string{"scheduled_pull", "raw_event_build", "broker_publish"},
		InputDatasets: []string{"equity_eod_prices", "option_contracts_daily"},
		OutputTopics:  []string{"signalops.local.raw.v1"},
		MetadataJSON:  []byte(`{"provider":"massive"}`),
		CreatedAt:     time.Date(2026, 7, 8, 0, 0, 5, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 7, 8, 0, 0, 6, 0, time.UTC),
	}
}

func validCatalogRuleRecord() storage.CatalogRuleRecord {
	return storage.CatalogRuleRecord{
		TenantID:       "tenant-1",
		RuleID:         "rule-marketdata-eod-price-quality",
		RuleName:       "Market Data EOD Price Quality",
		Description:    "Flags records with missing or non-positive close prices.",
		RuleType:       "quality_check",
		Severity:       "medium",
		Status:         storage.CatalogRuleStatusActive,
		Version:        1,
		SourceID:       "src-massive",
		PipelineID:     "pipeline-massive-raw-ingest",
		DatasetScope:   []string{"equity_eod_prices"},
		EntityScope:    []string{"ticker"},
		ExpressionJSON: []byte(`{"language":"json_logic"}`),
		Actions:        []string{"emit_alert"},
		MetadataJSON:   []byte(`{"execution":"catalog_only"}`),
		CreatedAt:      time.Date(2026, 7, 8, 0, 0, 7, 0, time.UTC),
		UpdatedAt:      time.Date(2026, 7, 8, 0, 0, 8, 0, time.UTC),
	}
}

func validSchedulerRunRecord() storage.SchedulerRunRecord {
	completed := time.Date(2026, 7, 8, 0, 1, 0, 0, time.UTC)
	return storage.SchedulerRunRecord{
		RunID:            "run-1",
		TenantID:         "tenant-1",
		SourceID:         "src-1",
		SourceAdapter:    "market_data.massive",
		Datasets:         []string{"equity_eod_prices"},
		ObservationDate:  time.Date(2026, 7, 7, 0, 0, 0, 0, time.UTC),
		DryRun:           true,
		Status:           storage.RunStatusSucceeded,
		StartedAt:        time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC),
		CompletedAt:      &completed,
		EventsBuilt:      1,
		EventsPublished:  0,
		ProviderRequests: 1,
		ConfigJSON:       []byte(`{"dry_run":true}`),
		ReportJSON:       []byte(`{"events_built":1}`),
		CreatedAt:        time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC),
		UpdatedAt:        completed,
	}
}

func validRawEventLedgerRecord() storage.RawEventLedgerRecord {
	partition := int32(2)
	offset := int64(42)
	return storage.RawEventLedgerRecord{
		EventID:         "event-1",
		TenantID:        "tenant-1",
		SourceID:        "src-1",
		SourceAdapter:   "market_data.massive",
		Dataset:         "equity_eod_prices",
		IdempotencyKey:  "idem-1",
		ObservationTime: time.Date(2026, 7, 7, 0, 0, 0, 0, time.UTC),
		ProcessingTime:  time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC),
		BrokerTopic:     "signalops.local.raw.v1",
		BrokerPartition: &partition,
		BrokerOffset:    &offset,
		PayloadJSON:     []byte(`{"event_id":"event-1"}`),
		EntityHintsJSON: []byte(`[]`),
		CreatedAt:       time.Date(2026, 7, 8, 0, 0, 1, 0, time.UTC),
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
		BudgetJSON:   []byte(`{}`),
		CreatedAt:    time.Date(2026, 7, 8, 0, 0, 2, 0, time.UTC),
	}
}
