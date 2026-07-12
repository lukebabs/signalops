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

	marketopsbacktest "github.com/lukebabs/signalops/internal/marketops/backtest"
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

type fakePublishRepository struct {
	ledger      storage.RawEventLedgerRecord
	idempotency storage.IdempotencyRecord
	err         error
}

func (p *fakePublishRepository) UpsertIdempotencyRecord(context.Context, storage.IdempotencyRecord) error {
	return nil
}
func (p *fakePublishRepository) UpsertRawEventLedger(context.Context, storage.RawEventLedgerRecord) error {
	return nil
}
func (p *fakePublishRepository) PersistPublishedRawEvent(_ context.Context, ledger storage.RawEventLedgerRecord, idempotency storage.IdempotencyRecord) error {
	p.ledger = ledger
	p.idempotency = idempotency
	return p.err
}

type fakeQueryRepository struct {
	runs                      []storage.SchedulerRunRecord
	replayJobs                []storage.ReplayJobRecord
	replayCounts              []storage.ReplayJobStatusCount
	replayWorkers             []storage.ReplayWorkerHeartbeatRecord
	lastReplayFilter          storage.ReplayJobFilter
	usage                     []storage.ProviderUsageRecord
	rawEvents                 []storage.RawEventLedgerRecord
	idem                      storage.IdempotencyRecord
	sources                   []storage.CatalogSourceRecord
	pipelines                 []storage.CatalogPipelineRecord
	rules                     []storage.CatalogRuleRecord
	marketOpsAssets           []storage.MarketOpsAssetRecord
	dsmArtifacts              []storage.MarketOpsDSMArtifactRecord
	dsmGraphProposals         []storage.MarketOpsDSMGraphProposalRecord
	backtestRuns              []storage.MarketOpsBacktestRunRecord
	backtestSignals           []storage.MarketOpsBacktestSignalRecord
	backtestGraphProposals    []storage.MarketOpsBacktestGraphProposalRecord
	backtestPolicyResults     []storage.MarketOpsBacktestPolicyResultRecord
	lastBacktestRunFilter     storage.MarketOpsBacktestRunFilter
	lastBacktestSignalFilter  storage.MarketOpsBacktestSignalFilter
	lastBacktestGraphFilter   storage.MarketOpsBacktestGraphProposalFilter
	lastDSMFilter             storage.MarketOpsDSMArtifactFilter
	lastGraphProposalFilter   storage.MarketOpsDSMGraphProposalFilter
	lastGraphProposalMutation storage.MarketOpsDSMGraphProposalMutation
	lastUniverseGroup         string
	lastActiveOnly            bool
	alerts                    []storage.AlertLedgerRecord
	insights                  []storage.InsightLedgerRecord
	notFound                  bool
	lastFilter                storage.RawEventLedgerFilter
	schedulerQueries          int
	rawEventQueries           int
	usageQueries              int
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

func (q *fakeQueryRepository) UpsertReplayJob(_ context.Context, record storage.ReplayJobRecord) error {
	q.replayJobs = append(q.replayJobs, record)
	return nil
}

func (q *fakeQueryRepository) ListReplayJobs(_ context.Context, filter storage.ReplayJobFilter) ([]storage.ReplayJobRecord, error) {
	q.lastReplayFilter = filter
	return q.replayJobs, nil
}

func (q *fakeQueryRepository) GetReplayJob(_ context.Context, replayJobID string) (storage.ReplayJobRecord, error) {
	if q.notFound {
		return storage.ReplayJobRecord{}, storage.ErrNotFound
	}
	for _, job := range q.replayJobs {
		if job.ReplayJobID == replayJobID {
			return job, nil
		}
	}
	return storage.ReplayJobRecord{}, storage.ErrNotFound
}

func (q *fakeQueryRepository) CountReplayJobsByStatus(context.Context, string) ([]storage.ReplayJobStatusCount, error) {
	if q.replayCounts != nil {
		return q.replayCounts, nil
	}
	counts := map[string]int{}
	for _, job := range q.replayJobs {
		counts[job.Status]++
	}
	items := make([]storage.ReplayJobStatusCount, 0, len(counts))
	for status, count := range counts {
		items = append(items, storage.ReplayJobStatusCount{Status: status, Count: count})
	}
	return items, nil
}

func (q *fakeQueryRepository) UpsertReplayWorkerHeartbeat(context.Context, storage.ReplayWorkerHeartbeatRecord) error {
	return nil
}

func (q *fakeQueryRepository) ListReplayWorkerHeartbeats(context.Context, int) ([]storage.ReplayWorkerHeartbeatRecord, error) {
	return q.replayWorkers, nil
}

func (q *fakeQueryRepository) ClaimNextReplayJob(context.Context, string, time.Time) (storage.ReplayJobRecord, error) {
	return storage.ReplayJobRecord{}, storage.ErrNotFound
}

func (q *fakeQueryRepository) CompleteReplayJob(context.Context, string, time.Time, []byte) (storage.ReplayJobRecord, error) {
	return storage.ReplayJobRecord{}, storage.ErrNotFound
}

func (q *fakeQueryRepository) FailReplayJob(context.Context, string, time.Time, string, []byte) (storage.ReplayJobRecord, error) {
	return storage.ReplayJobRecord{}, storage.ErrNotFound
}

func (q *fakeQueryRepository) CancelReplayJob(_ context.Context, replayJobID string, actor string, canceledAt time.Time, reason string, resultJSON []byte) (storage.ReplayJobRecord, error) {
	for index, job := range q.replayJobs {
		if job.ReplayJobID == replayJobID {
			job.Status = storage.ReplayJobStatusCanceled
			completedAt := canceledAt.UTC()
			job.CompletedAt = &completedAt
			job.ErrorMessage = "canceled by " + actor
			job.ResultJSON = []byte(`{"canceled":true}`)
			if len(resultJSON) > 0 {
				job.ResultJSON = resultJSON
			}
			_ = reason
			q.replayJobs[index] = job
			return job, nil
		}
	}
	return storage.ReplayJobRecord{}, storage.ErrNotFound
}

func (q *fakeQueryRepository) ListReplayRawEvents(context.Context, storage.ReplayJobRecord, int, int) ([]storage.RawEventLedgerRecord, error) {
	return nil, nil
}

func (q *fakeQueryRepository) ListReplayNormalizedEvents(context.Context, storage.ReplayJobRecord, int, int) ([]storage.NormalizedEventLedgerRecord, error) {
	return nil, nil
}

func (q *fakeQueryRepository) ListReplaySignals(context.Context, storage.ReplayJobRecord, int, int) ([]storage.SignalLedgerRecord, error) {
	return nil, nil
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

func (q *fakeQueryRepository) ListNormalizedEventLedger(context.Context, storage.RawEventLedgerFilter) ([]storage.NormalizedEventLedgerRecord, error) {
	return nil, nil
}
func (q *fakeQueryRepository) GetNormalizedEventLedger(context.Context, string) (storage.NormalizedEventLedgerRecord, error) {
	return storage.NormalizedEventLedgerRecord{}, storage.ErrNotFound
}

func (q *fakeQueryRepository) ListSignalLedger(context.Context, storage.SignalLedgerFilter) ([]storage.SignalLedgerRecord, error) {
	return nil, nil
}
func (q *fakeQueryRepository) GetSignalLedger(context.Context, string) (storage.SignalLedgerRecord, error) {
	return storage.SignalLedgerRecord{}, storage.ErrNotFound
}

func (q *fakeQueryRepository) CreateMarketOpsBacktestRun(_ context.Context, record storage.MarketOpsBacktestRunRecord) error {
	q.backtestRuns = append(q.backtestRuns, record)
	return nil
}

func (q *fakeQueryRepository) CompleteMarketOpsBacktestRun(_ context.Context, runID string, completedAt time.Time, metricsJSON []byte) (storage.MarketOpsBacktestRunRecord, error) {
	for i, run := range q.backtestRuns {
		if run.RunID == runID {
			run.Status = storage.RunStatusSucceeded
			run.CompletedAt = &completedAt
			run.MetricsJSON = metricsJSON
			q.backtestRuns[i] = run
			return run, nil
		}
	}
	return storage.MarketOpsBacktestRunRecord{}, storage.ErrNotFound
}

func (q *fakeQueryRepository) FailMarketOpsBacktestRun(_ context.Context, runID string, failedAt time.Time, errorMessage string, metricsJSON []byte) (storage.MarketOpsBacktestRunRecord, error) {
	for i, run := range q.backtestRuns {
		if run.RunID == runID {
			run.Status = storage.RunStatusFailed
			run.CompletedAt = &failedAt
			run.ErrorMessage = errorMessage
			run.MetricsJSON = metricsJSON
			q.backtestRuns[i] = run
			return run, nil
		}
	}
	return storage.MarketOpsBacktestRunRecord{}, storage.ErrNotFound
}

func (q *fakeQueryRepository) PersistMarketOpsBacktestBatch(_ context.Context, _ storage.MarketOpsBacktestRunRecord, signals []storage.MarketOpsBacktestSignalRecord, artifacts []storage.MarketOpsBacktestArtifactRecord, proposals []storage.MarketOpsBacktestGraphProposalRecord, policyResults []storage.MarketOpsBacktestPolicyResultRecord) error {
	q.backtestSignals = append(q.backtestSignals, signals...)
	_ = artifacts
	q.backtestGraphProposals = append(q.backtestGraphProposals, proposals...)
	q.backtestPolicyResults = append(q.backtestPolicyResults, policyResults...)
	return nil
}

func (q *fakeQueryRepository) ListMarketOpsBacktestRuns(_ context.Context, filter storage.MarketOpsBacktestRunFilter) ([]storage.MarketOpsBacktestRunRecord, error) {
	q.lastBacktestRunFilter = filter
	return q.backtestRuns, nil
}

func (q *fakeQueryRepository) GetMarketOpsBacktestRun(_ context.Context, runID string) (storage.MarketOpsBacktestRunRecord, error) {
	if q.notFound {
		return storage.MarketOpsBacktestRunRecord{}, storage.ErrNotFound
	}
	for _, run := range q.backtestRuns {
		if run.RunID == runID {
			return run, nil
		}
	}
	return storage.MarketOpsBacktestRunRecord{}, storage.ErrNotFound
}

func (q *fakeQueryRepository) ListMarketOpsBacktestSignals(_ context.Context, filter storage.MarketOpsBacktestSignalFilter) ([]storage.MarketOpsBacktestSignalRecord, error) {
	q.lastBacktestSignalFilter = filter
	return q.backtestSignals, nil
}

func (q *fakeQueryRepository) ListMarketOpsBacktestGraphProposals(_ context.Context, filter storage.MarketOpsBacktestGraphProposalFilter) ([]storage.MarketOpsBacktestGraphProposalRecord, error) {
	q.lastBacktestGraphFilter = filter
	return q.backtestGraphProposals, nil
}

func (q *fakeQueryRepository) ListMarketOpsBacktestPolicyResults(_ context.Context, filter storage.MarketOpsBacktestGraphProposalFilter) ([]storage.MarketOpsBacktestPolicyResultRecord, error) {
	q.lastBacktestGraphFilter = filter
	return q.backtestPolicyResults, nil
}

func (q *fakeQueryRepository) ListMarketOpsBacktestNormalizedEvents(context.Context, storage.MarketOpsBacktestEventFilter) ([]storage.NormalizedEventLedgerRecord, error) {
	return nil, nil
}

func (q *fakeQueryRepository) ListMarketOpsDSMArtifacts(_ context.Context, filter storage.MarketOpsDSMArtifactFilter) ([]storage.MarketOpsDSMArtifactRecord, error) {
	q.lastDSMFilter = filter
	return q.dsmArtifacts, nil
}

func (q *fakeQueryRepository) GetMarketOpsDSMArtifact(_ context.Context, artifactID string) (storage.MarketOpsDSMArtifactRecord, error) {
	if q.notFound {
		return storage.MarketOpsDSMArtifactRecord{}, storage.ErrNotFound
	}
	for _, artifact := range q.dsmArtifacts {
		if artifact.ArtifactID == artifactID {
			return artifact, nil
		}
	}
	return storage.MarketOpsDSMArtifactRecord{}, storage.ErrNotFound
}

func (q *fakeQueryRepository) ListMarketOpsDSMGraphProposals(_ context.Context, filter storage.MarketOpsDSMGraphProposalFilter) ([]storage.MarketOpsDSMGraphProposalRecord, error) {
	q.lastGraphProposalFilter = filter
	return q.dsmGraphProposals, nil
}

func (q *fakeQueryRepository) GetMarketOpsDSMGraphProposal(_ context.Context, proposalID string) (storage.MarketOpsDSMGraphProposalRecord, error) {
	if q.notFound {
		return storage.MarketOpsDSMGraphProposalRecord{}, storage.ErrNotFound
	}
	for _, proposal := range q.dsmGraphProposals {
		if proposal.ProposalID == proposalID {
			return proposal, nil
		}
	}
	return storage.MarketOpsDSMGraphProposalRecord{}, storage.ErrNotFound
}

func (q *fakeQueryRepository) MutateMarketOpsDSMGraphProposal(_ context.Context, mutation storage.MarketOpsDSMGraphProposalMutation) (storage.MarketOpsDSMGraphProposalRecord, error) {
	q.lastGraphProposalMutation = mutation
	if mutation.Status != storage.MarketOpsDSMGraphProposalStatusProposed && mutation.Status != storage.MarketOpsDSMGraphProposalStatusAccepted && mutation.Status != storage.MarketOpsDSMGraphProposalStatusRejected && mutation.Status != storage.MarketOpsDSMGraphProposalStatusSuperseded {
		return storage.MarketOpsDSMGraphProposalRecord{}, errors.New("marketops dsm graph proposal status is invalid")
	}
	for index, proposal := range q.dsmGraphProposals {
		if proposal.ProposalID == mutation.ProposalID {
			proposal.Status = mutation.Status
			proposal.ReviewedBy = mutation.ReviewedBy
			proposal.DecisionNote = mutation.DecisionNote
			decidedAt := mutation.DecidedAt.UTC()
			proposal.DecidedAt = &decidedAt
			q.dsmGraphProposals[index] = proposal
			return proposal, nil
		}
	}
	return storage.MarketOpsDSMGraphProposalRecord{}, storage.ErrNotFound
}

func (q *fakeQueryRepository) ListAlertLedger(context.Context, storage.AlertLedgerFilter) ([]storage.AlertLedgerRecord, error) {
	return q.alerts, nil
}

func (q *fakeQueryRepository) GetAlertLedger(_ context.Context, alertID string) (storage.AlertLedgerRecord, error) {
	if q.notFound {
		return storage.AlertLedgerRecord{}, storage.ErrNotFound
	}
	for _, alert := range q.alerts {
		if alert.AlertID == alertID {
			return alert, nil
		}
	}
	return storage.AlertLedgerRecord{}, storage.ErrNotFound
}

func (q *fakeQueryRepository) MutateAlertLifecycle(_ context.Context, mutation storage.AlertLifecycleMutation) (storage.AlertLedgerRecord, error) {
	if q.notFound {
		return storage.AlertLedgerRecord{}, storage.ErrNotFound
	}
	for i, alert := range q.alerts {
		if alert.AlertID == mutation.AlertID {
			alert.Status = mutation.Status
			alert.UpdatedAt = mutation.MutatedAt
			alert.MetadataJSON = mutation.MetadataJSON
			if mutation.Status == storage.AlertStatusAcknowledged {
				alert.AcknowledgedAt = &mutation.MutatedAt
				alert.AcknowledgedBy = mutation.Actor
			}
			if mutation.Status == storage.AlertStatusResolved {
				alert.ResolvedAt = &mutation.MutatedAt
				alert.ResolvedBy = mutation.Actor
			}
			q.alerts[i] = alert
			return alert, nil
		}
	}
	return storage.AlertLedgerRecord{}, storage.ErrNotFound
}

func (q *fakeQueryRepository) ListInsightLedger(context.Context, storage.InsightLedgerFilter) ([]storage.InsightLedgerRecord, error) {
	return q.insights, nil
}

func (q *fakeQueryRepository) GetInsightLedger(_ context.Context, insightID string) (storage.InsightLedgerRecord, error) {
	if q.notFound {
		return storage.InsightLedgerRecord{}, storage.ErrNotFound
	}
	for _, insight := range q.insights {
		if insight.InsightID == insightID {
			return insight, nil
		}
	}
	return storage.InsightLedgerRecord{}, storage.ErrNotFound
}

func (q *fakeQueryRepository) MutateInsightLifecycle(_ context.Context, mutation storage.InsightLifecycleMutation) (storage.InsightLedgerRecord, error) {
	if q.notFound {
		return storage.InsightLedgerRecord{}, storage.ErrNotFound
	}
	for i, insight := range q.insights {
		if insight.InsightID == mutation.InsightID {
			insight.Status = mutation.Status
			insight.UpdatedAt = mutation.MutatedAt
			insight.ReviewedAt = &mutation.MutatedAt
			insight.ReviewedBy = mutation.Actor
			insight.MetadataJSON = mutation.MetadataJSON
			q.insights[i] = insight
			return insight, nil
		}
	}
	return storage.InsightLedgerRecord{}, storage.ErrNotFound
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

func (q *fakeQueryRepository) ListMarketOpsAssets(_ context.Context, tenantID string, universeGroup string, activeOnly bool, limit int) ([]storage.MarketOpsAssetRecord, error) {
	q.lastUniverseGroup = universeGroup
	q.lastActiveOnly = activeOnly
	_ = tenantID
	_ = limit
	return q.marketOpsAssets, nil
}

func TestPostRawEventPublishesMessage(t *testing.T) {
	publisher := &fakePublisher{}
	repository := &fakePublishRepository{}
	router := NewRouter(RouterConfig{
		ServiceName:       "test-gateway",
		Publisher:         publisher,
		RawTopic:          "signalops.test.raw.v1",
		PublishRepository: repository,
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/events/raw", bytes.NewBufferString(`{
		"event_id":"evt-123",
		"idempotency_key":"idem-123",
		"correlation_id":"corr-payload",
		"tenant_id":"tenant-test",
		"source_id":"source-test",
		"source_adapter":"manual-test",
		"dataset":"equity-test",
		"observation_time":"2026-07-08T06:00:00Z",
		"entity_hints":[{"entity_type":"security","entity_id":"SPY"}],
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
	if repository.ledger.EventID != "evt-123" || repository.ledger.TenantID != "tenant-test" {
		t.Fatalf("persisted ledger = %+v", repository.ledger)
	}
	if repository.idempotency.Status != storage.IdempotencyStatusPublished || repository.idempotency.Offset == nil || *repository.idempotency.Offset != 42 {
		t.Fatalf("persisted idempotency = %+v", repository.idempotency)
	}
	if !strings.HasPrefix(repository.idempotency.PayloadHash, "sha256:") {
		t.Fatalf("payload hash = %q", repository.idempotency.PayloadHash)
	}
}

func TestPostRawEventRejectsInvalidJSON(t *testing.T) {
	router := NewRouter(RouterConfig{
		Publisher:         &fakePublisher{},
		RawTopic:          "signalops.test.raw.v1",
		PublishRepository: &fakePublishRepository{},
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
		Publisher:         &fakePublisher{err: errors.New("publish failed")},
		RawTopic:          "signalops.test.raw.v1",
		PublishRepository: &fakePublishRepository{},
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/events/raw", bytes.NewBufferString(validRawIngestBody()))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestPostRawEventRejectsMissingPersistenceFieldsBeforePublish(t *testing.T) {
	publisher := &fakePublisher{}
	router := NewRouter(RouterConfig{Publisher: publisher, RawTopic: "signalops.test.raw.v1", PublishRepository: &fakePublishRepository{}})
	req := httptest.NewRequest(http.MethodPost, "/v1/events/raw", bytes.NewBufferString(`{"event_id":"evt-123"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || publisher.msg.Topic != "" {
		t.Fatalf("status = %d, published topic = %q, body = %s", rec.Code, publisher.msg.Topic, rec.Body.String())
	}
}

func TestPostRawEventReportsPersistenceFailureAfterPublish(t *testing.T) {
	publisher := &fakePublisher{}
	repository := &fakePublishRepository{err: errors.New("database unavailable")}
	router := NewRouter(RouterConfig{Publisher: publisher, RawTopic: "signalops.test.raw.v1", PublishRepository: repository})
	req := httptest.NewRequest(http.MethodPost, "/v1/events/raw", bytes.NewBufferString(validRawIngestBody()))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable || publisher.msg.Topic == "" {
		t.Fatalf("status = %d, published topic = %q, body = %s", rec.Code, publisher.msg.Topic, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "persistence_failed") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func validRawIngestBody() string {
	return `{"event_id":"evt-123","tenant_id":"tenant-test","source_id":"source-test","source_adapter":"manual-test","dataset":"equity-test","observation_time":"2026-07-08T06:00:00Z","payload":{"symbol":"SPY"}}`
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

func TestReplayJobCreateListAndDetail(t *testing.T) {
	repo := &fakeQueryRepository{}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	body := `{"tenant_id":"tenant-local","source_id":"src-massive","dataset":"equity_eod_prices","source_kind":"raw_events","replay_mode":"original","window_start":"2026-07-09T00:00:00Z","window_end":"2026-07-10T00:00:00Z","filters":{"symbol":"AAPL"},"options":{"publish":false}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/replay/jobs", bytes.NewBufferString(body))
	req.Header.Set("X-SignalOps-Actor", "operator-g058")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(repo.replayJobs) != 1 {
		t.Fatalf("replay jobs = %+v", repo.replayJobs)
	}
	created := repo.replayJobs[0]
	if created.Status != storage.ReplayJobStatusQueued || created.RequestedBy != "operator-g058" || created.SourceKind != storage.ReplaySourceRaw {
		t.Fatalf("created replay job = %+v", created)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/replay/jobs?tenant_id=tenant-local&status=queued&source_kind=raw_events&limit=3", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if repo.lastReplayFilter.TenantID != "tenant-local" || repo.lastReplayFilter.Status != "queued" || repo.lastReplayFilter.SourceKind != "raw_events" || repo.lastReplayFilter.Limit != 3 {
		t.Fatalf("filter = %+v", repo.lastReplayFilter)
	}
	var listResponse map[string][]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &listResponse); err != nil {
		t.Fatalf("list JSON error = %v", err)
	}
	if len(listResponse["replay_jobs"]) != 1 || listResponse["replay_jobs"][0]["replay_job_id"] != created.ReplayJobID {
		t.Fatalf("list response = %+v", listResponse)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/replay/jobs/"+created.ReplayJobID, nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("detail status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestCancelReplayJob(t *testing.T) {
	job := storage.ReplayJobRecord{
		ReplayJobID: "replay-1",
		TenantID:    "tenant-local",
		SourceKind:  storage.ReplaySourceRaw,
		ReplayMode:  storage.ReplayModeOriginal,
		Status:      storage.ReplayJobStatusQueued,
		WindowStart: time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC),
		WindowEnd:   time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	repo := &fakeQueryRepository{replayJobs: []storage.ReplayJobRecord{job}}
	router := NewRouter(RouterConfig{QueryRepository: repo})

	req := httptest.NewRequest(http.MethodPost, "/v1/replay/jobs/replay-1/cancel", bytes.NewBufferString(`{"actor":"operator-g061","reason":"validation"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if repo.replayJobs[0].Status != storage.ReplayJobStatusCanceled || repo.replayJobs[0].CompletedAt == nil {
		t.Fatalf("canceled replay job = %+v", repo.replayJobs[0])
	}
	var response map[string]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("cancel JSON error = %v", err)
	}
	if response["replay_job"]["status"] != storage.ReplayJobStatusCanceled {
		t.Fatalf("cancel response = %+v", response)
	}
}

func TestReplayStatus(t *testing.T) {
	now := time.Now().UTC()
	job := storage.ReplayJobRecord{
		ReplayJobID: "replay-1",
		TenantID:    "tenant-local",
		SourceKind:  storage.ReplaySourceRaw,
		ReplayMode:  storage.ReplayModeOriginal,
		Status:      storage.ReplayJobStatusRunning,
		WindowStart: now.Add(-time.Hour),
		WindowEnd:   now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	repo := &fakeQueryRepository{
		replayJobs:   []storage.ReplayJobRecord{job},
		replayCounts: []storage.ReplayJobStatusCount{{Status: storage.ReplayJobStatusRunning, Count: 1}},
		replayWorkers: []storage.ReplayWorkerHeartbeatRecord{{
			WorkerID: "worker-1", Status: "idle", ProcessStartedAt: now.Add(-time.Minute), LastSeenAt: now,
			LastCompletedReplayJobID: "replay-0", MetadataJSON: []byte(`{"poll_interval":"5s"}`), CreatedAt: now.Add(-time.Minute), UpdatedAt: now,
		}},
	}
	router := NewRouter(RouterConfig{QueryRepository: repo})

	req := httptest.NewRequest(http.MethodGet, "/v1/replay/status?tenant_id=tenant-local", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("status JSON error = %v", err)
	}
	statusPayload := response["replay_status"].(map[string]any)
	counts := statusPayload["job_counts"].(map[string]any)
	if counts[storage.ReplayJobStatusRunning].(float64) != 1 || counts[storage.ReplayJobStatusQueued].(float64) != 0 {
		t.Fatalf("counts = %+v", counts)
	}
	workers := statusPayload["workers"].([]any)
	if len(workers) != 1 || workers[0].(map[string]any)["health"] != "online" {
		t.Fatalf("workers = %+v", workers)
	}
	latest := statusPayload["latest_jobs"].([]any)
	if len(latest) != 1 || latest[0].(map[string]any)["replay_job_id"] != "replay-1" {
		t.Fatalf("latest jobs = %+v", latest)
	}
}

func TestPostReplayJobRejectsInvalidWindow(t *testing.T) {
	router := NewRouter(RouterConfig{QueryRepository: &fakeQueryRepository{}})
	body := `{"tenant_id":"tenant-local","source_kind":"raw_events","window_start":"2026-07-10T00:00:00Z","window_end":"2026-07-09T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/replay/jobs", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
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

func TestGetMarketOpsAssets(t *testing.T) {
	repo := &fakeQueryRepository{marketOpsAssets: []storage.MarketOpsAssetRecord{validMarketOpsAssetRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})

	req := httptest.NewRequest(http.MethodGet, "/v1/tenants/tenant-1/marketops/assets?universe_group=top50_megacap&active_only=false&limit=10", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if repo.lastUniverseGroup != "top50_megacap" || repo.lastActiveOnly {
		t.Fatalf("universe/active filter = %q/%t", repo.lastUniverseGroup, repo.lastActiveOnly)
	}
	var response map[string][]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("response JSON error = %v", err)
	}
	if len(response["assets"]) != 1 || response["assets"][0]["ticker"] != "NVDA" {
		t.Fatalf("response = %+v", response)
	}
	if response["assets"][0]["app_id"] != "marketops" || response["assets"][0]["use_case"] != "daily_market_surveillance" {
		t.Fatalf("asset metadata = %+v", response["assets"][0])
	}
}

func TestPostMarketOpsBacktestCreatesRun(t *testing.T) {
	repo := &fakeQueryRepository{}
	runner := func(ctx context.Context, repo storage.MarketOpsBacktestRepository, cfg marketopsbacktest.Config) (marketopsbacktest.Result, error) {
		now := time.Date(2026, 7, 12, 10, 0, 0, 0, time.UTC)
		completed := now.Add(time.Second)
		record := storage.MarketOpsBacktestRunRecord{RunID: cfg.RunID, TenantID: cfg.TenantID, AppID: cfg.AppID, Domain: cfg.Domain, UseCase: cfg.UseCase, SourceID: cfg.SourceID, SourceAdapter: cfg.SourceAdapter, Dataset: cfg.Dataset, DetectorID: cfg.DetectorID, Status: storage.RunStatusSucceeded, RequestedBy: cfg.RequestedBy, WindowStart: cfg.WindowStart, WindowEnd: cfg.WindowEnd, StartedAt: now, CompletedAt: &completed, FiltersJSON: []byte(`{"symbols":["SPY"]}`), ParametersJSON: []byte(`{"detector_id":"marketops.dsm.taxonomy_v1"}`), MetricsJSON: []byte(`{"scanned":1}`), CreatedAt: now, UpdatedAt: completed}
		return marketopsbacktest.Result{Run: record, Metrics: marketopsbacktest.Metrics{RunID: cfg.RunID, Scanned: 1, Signals: 1, RecommendationCounts: map[string]int{storage.MarketOpsBacktestPolicyAutoAcceptCandidate: 5}}}, nil
	}
	router := NewRouter(RouterConfig{QueryRepository: repo, MarketOpsBacktestRunner: runner})
	body := `{"run_id":"bt-api-1","tenant_id":"tenant-local","source_id":"src-massive","dataset":"equity_eod_prices","symbols":["spy"],"window_start":"2026-07-09T00:00:00Z","window_end":"2026-07-10T00:00:00Z","max_records":5}`
	req := httptest.NewRequest(http.MethodPost, "/v1/marketops/backtests", strings.NewReader(body))
	req.Header.Set("X-SignalOps-Actor", "operator-api")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var response marketOpsBacktestCreateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("response JSON error = %v", err)
	}
	if response.BacktestRun.RunID != "bt-api-1" || response.BacktestRun.RequestedBy != "operator-api" {
		t.Fatalf("response = %+v", response)
	}
}

func TestPostMarketOpsBacktestRejectsInvalidWindow(t *testing.T) {
	router := NewRouter(RouterConfig{QueryRepository: &fakeQueryRepository{}})
	body := `{"tenant_id":"tenant-local","window_start":"2026-07-10T00:00:00Z","window_end":"2026-07-09T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/marketops/backtests", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestGetMarketOpsBacktestRuns(t *testing.T) {
	repo := &fakeQueryRepository{backtestRuns: []storage.MarketOpsBacktestRunRecord{validMarketOpsBacktestRunRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})

	req := httptest.NewRequest(http.MethodGet, "/v1/marketops/backtests?tenant_id=tenant-1&detector_id=marketops.dsm.taxonomy_v1&status=succeeded&limit=10", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if repo.lastBacktestRunFilter.TenantID != "tenant-1" || repo.lastBacktestRunFilter.DetectorID != "marketops.dsm.taxonomy_v1" || repo.lastBacktestRunFilter.Status != "succeeded" || repo.lastBacktestRunFilter.Limit != 10 {
		t.Fatalf("filter = %+v", repo.lastBacktestRunFilter)
	}
	var response map[string][]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("response JSON error = %v", err)
	}
	if len(response["backtest_runs"]) != 1 || response["backtest_runs"][0]["run_id"] != "bt-marketops-1" {
		t.Fatalf("response = %+v", response)
	}
}

func TestGetMarketOpsBacktestRun(t *testing.T) {
	repo := &fakeQueryRepository{backtestRuns: []storage.MarketOpsBacktestRunRecord{validMarketOpsBacktestRunRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})

	req := httptest.NewRequest(http.MethodGet, "/v1/marketops/backtests/bt-marketops-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var response map[string]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("response JSON error = %v", err)
	}
	if response["backtest_run"]["metrics"].(map[string]any)["signals"].(float64) != 1 {
		t.Fatalf("response = %+v", response)
	}
}

func TestGetMarketOpsBacktestSignals(t *testing.T) {
	repo := &fakeQueryRepository{backtestSignals: []storage.MarketOpsBacktestSignalRecord{validMarketOpsBacktestSignalRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})

	req := httptest.NewRequest(http.MethodGet, "/v1/marketops/backtests/bt-marketops-1/signals?tenant_id=tenant-1&signal_type=marketops.dsm.pinning_risk&limit=10", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if repo.lastBacktestSignalFilter.RunID != "bt-marketops-1" || repo.lastBacktestSignalFilter.SignalType != "marketops.dsm.pinning_risk" {
		t.Fatalf("filter = %+v", repo.lastBacktestSignalFilter)
	}
	var response map[string][]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("response JSON error = %v", err)
	}
	if response["backtest_signals"][0]["signal"].(map[string]any)["signal_id"] != "signal-1" {
		t.Fatalf("response = %+v", response)
	}
}

func TestGetMarketOpsBacktestGraphProposalsIncludesPolicyResults(t *testing.T) {
	repo := &fakeQueryRepository{backtestGraphProposals: []storage.MarketOpsBacktestGraphProposalRecord{validMarketOpsBacktestGraphProposalRecord()}, backtestPolicyResults: []storage.MarketOpsBacktestPolicyResultRecord{validMarketOpsBacktestPolicyResultRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})

	req := httptest.NewRequest(http.MethodGet, "/v1/marketops/backtests/bt-marketops-1/graph-proposals?tenant_id=tenant-1&subject_symbol=AAPL&candidate_type=node_candidate&recommendation=auto_accept_candidate&limit=10", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if repo.lastBacktestGraphFilter.RunID != "bt-marketops-1" || repo.lastBacktestGraphFilter.Recommendation != storage.MarketOpsBacktestPolicyAutoAcceptCandidate {
		t.Fatalf("filter = %+v", repo.lastBacktestGraphFilter)
	}
	var response map[string][]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("response JSON error = %v", err)
	}
	if response["policy_results"][0]["recommendation"] != storage.MarketOpsBacktestPolicyAutoAcceptCandidate {
		t.Fatalf("response = %+v", response)
	}
}

func TestGetMarketOpsDSMArtifacts(t *testing.T) {
	repo := &fakeQueryRepository{dsmArtifacts: []storage.MarketOpsDSMArtifactRecord{validMarketOpsDSMArtifactRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})

	req := httptest.NewRequest(http.MethodGet, "/v1/marketops/dsm/artifacts?tenant_id=tenant-1&app_id=marketops&domain=market_data&use_case=daily_market_surveillance&signal_type=marketops.dsm.pinning_risk&severity=high&subject_symbol=AAPL&limit=10", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if repo.lastDSMFilter.TenantID != "tenant-1" || repo.lastDSMFilter.SignalType != "marketops.dsm.pinning_risk" || repo.lastDSMFilter.SubjectSymbol != "AAPL" || repo.lastDSMFilter.Limit != 10 {
		t.Fatalf("filter = %+v", repo.lastDSMFilter)
	}
	var response map[string][]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("response JSON error = %v", err)
	}
	artifact := response["artifacts"][0]
	if artifact["artifact_id"] != "artifact_marketops_dsm_v1_test" || artifact["subject_symbol"] != "AAPL" {
		t.Fatalf("artifact = %+v", artifact)
	}
	if artifact["artifact"].(map[string]any)["artifact_type"] != "marketops.dsm.signal_artifact.v1" {
		t.Fatalf("artifact payload = %+v", artifact["artifact"])
	}
}

func TestGetMarketOpsDSMArtifact(t *testing.T) {
	repo := &fakeQueryRepository{dsmArtifacts: []storage.MarketOpsDSMArtifactRecord{validMarketOpsDSMArtifactRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})

	req := httptest.NewRequest(http.MethodGet, "/v1/marketops/dsm/artifacts/artifact_marketops_dsm_v1_test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var response map[string]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("response JSON error = %v", err)
	}
	if response["artifact"]["signal_id"] != "signal-1" {
		t.Fatalf("response = %+v", response)
	}
}

func TestGetMarketOpsDSMGraphProposals(t *testing.T) {
	repo := &fakeQueryRepository{dsmGraphProposals: []storage.MarketOpsDSMGraphProposalRecord{validMarketOpsDSMGraphProposalRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})

	req := httptest.NewRequest(http.MethodGet, "/v1/marketops/dsm/graph-proposals?tenant_id=tenant-1&app_id=marketops&domain=market_data&use_case=daily_market_surveillance&artifact_id=artifact_marketops_dsm_v1_test&signal_id=signal-1&signal_type=marketops.dsm.pinning_risk&subject_symbol=AAPL&candidate_type=node_candidate&status=proposed&limit=10", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	filter := repo.lastGraphProposalFilter
	if filter.TenantID != "tenant-1" || filter.ArtifactID != "artifact_marketops_dsm_v1_test" || filter.SignalID != "signal-1" || filter.CandidateType != "node_candidate" || filter.Status != "proposed" || filter.Limit != 10 {
		t.Fatalf("filter = %+v", filter)
	}
	var response map[string][]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("response JSON error = %v", err)
	}
	proposal := response["graph_proposals"][0]
	if proposal["proposal_id"] != "graphprop_marketops_dsm_v1_test" || proposal["node_id"] != "ticker:AAPL" || proposal["status"] != "proposed" {
		t.Fatalf("proposal = %+v", proposal)
	}
}

func TestGetMarketOpsDSMGraphProposal(t *testing.T) {
	repo := &fakeQueryRepository{dsmGraphProposals: []storage.MarketOpsDSMGraphProposalRecord{validMarketOpsDSMGraphProposalRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})

	req := httptest.NewRequest(http.MethodGet, "/v1/marketops/dsm/graph-proposals/graphprop_marketops_dsm_v1_test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var response map[string]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("response JSON error = %v", err)
	}
	if response["graph_proposal"]["artifact_id"] != "artifact_marketops_dsm_v1_test" {
		t.Fatalf("response = %+v", response)
	}
}

func TestPostMarketOpsDSMGraphProposalDecision(t *testing.T) {
	repo := &fakeQueryRepository{dsmGraphProposals: []storage.MarketOpsDSMGraphProposalRecord{validMarketOpsDSMGraphProposalRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})

	req := httptest.NewRequest(http.MethodPost, "/v1/marketops/dsm/graph-proposals/graphprop_marketops_dsm_v1_test/decision", strings.NewReader(`{"status":"accepted","note":"Approved for materialization."}`))
	req.Header.Set("X-SignalOps-Actor", "operator-g079")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if repo.lastGraphProposalMutation.Status != storage.MarketOpsDSMGraphProposalStatusAccepted || repo.lastGraphProposalMutation.ReviewedBy != "operator-g079" || repo.lastGraphProposalMutation.DecisionNote == "" {
		t.Fatalf("mutation = %+v", repo.lastGraphProposalMutation)
	}
	var response map[string]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("response JSON error = %v", err)
	}
	if response["graph_proposal"]["status"] != "accepted" || response["graph_proposal"]["reviewed_by"] != "operator-g079" {
		t.Fatalf("response = %+v", response)
	}
}

func TestPostMarketOpsDSMGraphProposalDecisionRejectsInvalidStatus(t *testing.T) {
	repo := &fakeQueryRepository{dsmGraphProposals: []storage.MarketOpsDSMGraphProposalRecord{validMarketOpsDSMGraphProposalRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})

	req := httptest.NewRequest(http.MethodPost, "/v1/marketops/dsm/graph-proposals/graphprop_marketops_dsm_v1_test/decision", strings.NewReader(`{"status":"invalid"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
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

func TestListAlertsReturnsLifecycleEnvelope(t *testing.T) {
	repo := &fakeQueryRepository{alerts: []storage.AlertLedgerRecord{validAlertRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/alerts?tenant_id=tenant-1&status=open", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Alerts []alertDTO `json:"alerts"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Alerts) != 1 || body.Alerts[0].AlertID != "alert-1" || body.Alerts[0].Status != storage.AlertStatusOpen {
		t.Fatalf("body = %+v", body)
	}
}

func TestListInsightsReturnsLifecycleEnvelope(t *testing.T) {
	repo := &fakeQueryRepository{insights: []storage.InsightLedgerRecord{validInsightRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/insights?tenant_id=tenant-1&status=active", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Insights []insightDTO `json:"insights"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Insights) != 1 || body.Insights[0].InsightID != "insight-1" || body.Insights[0].Status != storage.InsightStatusActive {
		t.Fatalf("body = %+v", body)
	}
}

func TestAcknowledgeAlertMutatesLifecycle(t *testing.T) {
	repo := &fakeQueryRepository{alerts: []storage.AlertLedgerRecord{validAlertRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodPost, "/v1/alerts/alert-1/acknowledge", bytes.NewBufferString(`{"note":"triaged"}`))
	req.Header.Set("X-SignalOps-Actor", "operator-test")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Alert alertDTO `json:"alert"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Alert.Status != storage.AlertStatusAcknowledged || body.Alert.AcknowledgedBy != "operator-test" || body.Alert.AcknowledgedAt == nil {
		t.Fatalf("alert = %+v", body.Alert)
	}
	if !bytes.Contains(body.Alert.Metadata, []byte(`"action":"acknowledge"`)) {
		t.Fatalf("metadata = %s", string(body.Alert.Metadata))
	}
}

func TestArchiveInsightMutatesLifecycle(t *testing.T) {
	repo := &fakeQueryRepository{insights: []storage.InsightLedgerRecord{validInsightRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodPost, "/v1/insights/insight-1/archive", bytes.NewBufferString(`{"actor":"operator-body","reason":"closed"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Insight insightDTO `json:"insight"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Insight.Status != storage.InsightStatusArchived || body.Insight.ReviewedBy != "operator-body" || body.Insight.ReviewedAt == nil {
		t.Fatalf("insight = %+v", body.Insight)
	}
	if !bytes.Contains(body.Insight.Metadata, []byte(`"action":"archive"`)) {
		t.Fatalf("metadata = %s", string(body.Insight.Metadata))
	}
}

func validMarketOpsAssetRecord() storage.MarketOpsAssetRecord {
	return storage.MarketOpsAssetRecord{
		TenantID:      "tenant-1",
		AppID:         "marketops",
		Domain:        "market_data",
		UseCase:       "daily_market_surveillance",
		SourceID:      "src-massive",
		UniverseGroup: "top50_megacap",
		Rank:          1,
		Ticker:        "NVDA",
		TickerKey:     "nvda",
		Company:       "NVIDIA",
		CompanyKey:    "nvidia",
		AssetType:     "equity",
		Sector:        "Technology",
		SectorKey:     "technology",
		Industry:      "Semiconductors",
		IndustryKey:   "semiconductors",
		IsActive:      true,
		MetadataJSON:  []byte(`{"seed":"top50megacap.normalized.csv"}`),
		CreatedAt:     time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 7, 10, 0, 0, 1, 0, time.UTC),
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

func validMarketOpsDSMArtifactRecord() storage.MarketOpsDSMArtifactRecord {
	return storage.MarketOpsDSMArtifactRecord{
		ArtifactID:           "artifact_marketops_dsm_v1_test",
		TenantID:             "tenant-1",
		AppID:                "marketops",
		Domain:               "market_data",
		UseCase:              "daily_market_surveillance",
		SourceID:             "src-massive",
		SourceAdapter:        "market_data.massive",
		Dataset:              "options_contracts_daily",
		SignalID:             "signal-1",
		SignalType:           "marketops.dsm.pinning_risk",
		DetectorID:           "marketops.dsm.taxonomy_v1",
		Severity:             "high",
		Confidence:           0.84,
		EventIDs:             []string{"event-1"},
		SubjectSymbol:        "AAPL",
		ArtifactType:         "marketops.dsm.signal_artifact.v1",
		ArtifactJSON:         []byte(`{"artifact_id":"artifact_marketops_dsm_v1_test","artifact_type":"marketops.dsm.signal_artifact.v1","subject":{"symbol":"AAPL"}}`),
		SemanticEvidenceJSON: []byte(`{"type":"dsm_artifact_proposal","artifact_id":"artifact_marketops_dsm_v1_test"}`),
		GraphTargetsJSON:     []byte(`[{"type":"node_candidate"}]`),
		SupportingMetrics:    []byte(`{"open_interest":2000}`),
		QualityIssues:        []string{},
		CreatedAt:            time.Date(2026, 7, 8, 0, 2, 0, 0, time.UTC),
		UpdatedAt:            time.Date(2026, 7, 8, 0, 3, 0, 0, time.UTC),
	}
}

func validMarketOpsDSMGraphProposalRecord() storage.MarketOpsDSMGraphProposalRecord {
	return storage.MarketOpsDSMGraphProposalRecord{
		ProposalID:     "graphprop_marketops_dsm_v1_test",
		TenantID:       "tenant-1",
		AppID:          "marketops",
		Domain:         "market_data",
		UseCase:        "daily_market_surveillance",
		SourceID:       "src-massive",
		SourceAdapter:  "market_data.massive",
		Dataset:        "options_contracts_daily",
		ArtifactID:     "artifact_marketops_dsm_v1_test",
		SignalID:       "signal-1",
		SignalType:     "marketops.dsm.pinning_risk",
		DetectorID:     "marketops.dsm.taxonomy_v1",
		Severity:       "high",
		Confidence:     0.84,
		EventIDs:       []string{"event-1"},
		SubjectSymbol:  "AAPL",
		CandidateType:  "node_candidate",
		NodeID:         "ticker:AAPL",
		Labels:         []string{"MarketAsset", "Ticker"},
		PropertiesJSON: []byte(`{"symbol":"AAPL"}`),
		RawCandidate:   []byte(`{"type":"node_candidate","node_id":"ticker:AAPL"}`),
		Status:         storage.MarketOpsDSMGraphProposalStatusProposed,
		CreatedAt:      time.Date(2026, 7, 8, 0, 4, 0, 0, time.UTC),
		UpdatedAt:      time.Date(2026, 7, 8, 0, 5, 0, 0, time.UTC),
	}
}

func validAlertRecord() storage.AlertLedgerRecord {
	return storage.AlertLedgerRecord{
		AlertID:            "alert-1",
		TenantID:           "tenant-1",
		SourceID:           "src-1",
		SourceDomain:       "market_data",
		SourceAdapter:      "market_data.massive",
		Dataset:            "equity_eod_prices",
		SignalID:           "signal-1",
		DetectorID:         "detector-1",
		AlertType:          "price.quality",
		Severity:           "high",
		Status:             storage.AlertStatusOpen,
		Title:              "High price quality alert",
		Summary:            "Detector emitted a high signal.",
		Confidence:         0.9,
		EventIDs:           []string{"event-1"},
		EntitiesJSON:       []byte(`[]`),
		EvidenceJSON:       []byte(`[]`),
		RecommendationJSON: []byte(`{"action":"inspect"}`),
		CorrelationID:      "corr-1",
		FirstObservedAt:    time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC),
		LastObservedAt:     time.Date(2026, 7, 8, 0, 1, 0, 0, time.UTC),
		MetadataJSON:       []byte(`{"derived_from":"signal_ledger"}`),
		CreatedAt:          time.Date(2026, 7, 8, 0, 2, 0, 0, time.UTC),
		UpdatedAt:          time.Date(2026, 7, 8, 0, 3, 0, 0, time.UTC),
	}
}

func validInsightRecord() storage.InsightLedgerRecord {
	return storage.InsightLedgerRecord{
		InsightID:            "insight-1",
		TenantID:             "tenant-1",
		SourceID:             "src-1",
		SourceDomain:         "market_data",
		SourceAdapter:        "market_data.massive",
		Dataset:              "equity_eod_prices",
		SignalID:             "signal-1",
		DetectorID:           "detector-1",
		InsightType:          "price.quality",
		Status:               storage.InsightStatusActive,
		Title:                "High signal from detector-1",
		Summary:              "Detector emitted a high signal.",
		Confidence:           0.9,
		Severity:             "high",
		EventIDs:             []string{"event-1"},
		EntitiesJSON:         []byte(`[]`),
		SupportingMetrics:    []byte(`{}`),
		SemanticEvidenceJSON: []byte(`[]`),
		RecommendationJSON:   []byte(`null`),
		CorrelationID:        "corr-1",
		ObservedAt:           time.Date(2026, 7, 8, 0, 1, 0, 0, time.UTC),
		MetadataJSON:         []byte(`{"derived_from":"signal_ledger"}`),
		CreatedAt:            time.Date(2026, 7, 8, 0, 2, 0, 0, time.UTC),
		UpdatedAt:            time.Date(2026, 7, 8, 0, 3, 0, 0, time.UTC),
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

func validMarketOpsBacktestRunRecord() storage.MarketOpsBacktestRunRecord {
	now := time.Date(2026, 7, 12, 10, 0, 0, 0, time.UTC)
	completed := now.Add(time.Minute)
	return storage.MarketOpsBacktestRunRecord{RunID: "bt-marketops-1", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SourceID: "src-massive", SourceAdapter: "market_data.massive", Dataset: "equity_eod_prices", DetectorID: "marketops.dsm.taxonomy_v1", DetectorVersion: "0.1.0", Status: storage.RunStatusSucceeded, RequestedBy: "operator-test", WindowStart: now.Add(-24 * time.Hour), WindowEnd: now, StartedAt: now, CompletedAt: &completed, FiltersJSON: []byte(`{"symbols":["AAPL"]}`), ParametersJSON: []byte(`{"detector_id":"marketops.dsm.taxonomy_v1"}`), MetricsJSON: []byte(`{"signals":1}`), CreatedAt: now, UpdatedAt: completed}
}

func validMarketOpsBacktestSignalRecord() storage.MarketOpsBacktestSignalRecord {
	now := time.Date(2026, 7, 12, 10, 0, 0, 0, time.UTC)
	return storage.MarketOpsBacktestSignalRecord{RunID: "bt-marketops-1", SignalLedgerRecord: storage.SignalLedgerRecord{SignalID: "signal-1", TenantID: "tenant-1", SourceID: "src-massive", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SourceDomain: "market_data", SourceAdapter: "market_data.massive", IngestionMode: "scheduled_pull", Dataset: "equity_eod_prices", EventIDs: []string{"event-1"}, ArtifactIDs: []string{"artifact_marketops_dsm_v1_test"}, SignalType: "marketops.dsm.pinning_risk", DetectorID: "marketops.dsm.taxonomy_v1", DetectorVersion: "0.1.0", ModelVersion: "deterministic-v0", SignalTime: now, ObservationTime: now, EffectiveTime: now, ProcessingTime: now, WindowStart: now, WindowEnd: now, Confidence: 0.84, Severity: "high", EntitiesJSON: []byte(`[]`), SupportingMetrics: []byte(`{}`), GraphTargetsJSON: []byte(`[]`), SemanticEvidenceJSON: []byte(`[]`), EvidenceJSON: []byte(`[]`), RecommendationJSON: []byte(`{"action":"review"}`), CorrelationID: "corr-1", EventJSON: []byte(`{"signal_id":"signal-1"}`), CreatedAt: now, UpdatedAt: now}}
}

func validMarketOpsBacktestGraphProposalRecord() storage.MarketOpsBacktestGraphProposalRecord {
	return storage.MarketOpsBacktestGraphProposalRecord{RunID: "bt-marketops-1", MarketOpsDSMGraphProposalRecord: validMarketOpsDSMGraphProposalRecord()}
}

func validMarketOpsBacktestPolicyResultRecord() storage.MarketOpsBacktestPolicyResultRecord {
	now := time.Date(2026, 7, 12, 10, 0, 0, 0, time.UTC)
	return storage.MarketOpsBacktestPolicyResultRecord{RunID: "bt-marketops-1", PolicyResultID: "btpolicy-1", ProposalID: "graphprop_marketops_dsm_v1_test", ArtifactID: "artifact_marketops_dsm_v1_test", SignalID: "signal-1", TenantID: "tenant-1", SubjectSymbol: "AAPL", CandidateType: "node_candidate", Recommendation: storage.MarketOpsBacktestPolicyAutoAcceptCandidate, Reason: "candidate matches deterministic auto-accept policy", PolicyVersion: "marketops.backtest.policy_v1", Confidence: 1, DecisionInputsJSON: []byte(`{"node_id":"ticker:AAPL"}`), CreatedAt: now}
}
