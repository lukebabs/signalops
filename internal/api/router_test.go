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
	"github.com/lukebabs/signalops/internal/syncratic/userapi"
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

type fakeSyncraticAskClient struct {
	calls int
	req   userapi.AskRequest
	resp  userapi.AskResponse
	err   error
}

func (c *fakeSyncraticAskClient) Ask(_ context.Context, req userapi.AskRequest) (userapi.AskResponse, error) {
	c.calls++
	c.req = req
	if c.err != nil {
		return userapi.AskResponse{}, c.err
	}
	if c.resp.Answer == "" && len(c.resp.Raw) == 0 {
		c.resp = userapi.AskResponse{QueryID: "ask-test", Answer: "AAPL shows a bounded multi-event volatility pattern worth operator review.", Confidence: userapi.NumericFloat(0.82), EvidenceCount: 3}
	}
	return c.resp, nil
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
	runs                           []storage.SchedulerRunRecord
	replayJobs                     []storage.ReplayJobRecord
	replayCounts                   []storage.ReplayJobStatusCount
	replayWorkers                  []storage.ReplayWorkerHeartbeatRecord
	lastReplayFilter               storage.ReplayJobFilter
	usage                          []storage.ProviderUsageRecord
	rawEvents                      []storage.RawEventLedgerRecord
	idem                           storage.IdempotencyRecord
	sources                        []storage.CatalogSourceRecord
	pipelines                      []storage.CatalogPipelineRecord
	rules                          []storage.CatalogRuleRecord
	marketOpsAssets                []storage.MarketOpsAssetRecord
	dsmArtifacts                   []storage.MarketOpsDSMArtifactRecord
	dsmGraphProposals              []storage.MarketOpsDSMGraphProposalRecord
	backtestRuns                   []storage.MarketOpsBacktestRunRecord
	backtestCoverage               []storage.MarketOpsBacktestCoverageRecord
	backtestCampaigns              []storage.MarketOpsBacktestCampaignRecord
	backtestSignals                []storage.MarketOpsBacktestSignalRecord
	backtestGraphProposals         []storage.MarketOpsBacktestGraphProposalRecord
	backtestPolicyResults          []storage.MarketOpsBacktestPolicyResultRecord
	backtestCalibrationSummaries   []storage.MarketOpsBacktestCalibrationSummaryRecord
	backtestCalibrationBaselines   []storage.MarketOpsBacktestCalibrationBaselineRecord
	backtestCalibrationComparisons []storage.MarketOpsBacktestCalibrationComparisonRecord
	backtestEvaluationLabels       []storage.MarketOpsBacktestEvaluationLabelRecord
	backtestEvaluations            []storage.MarketOpsBacktestEvaluationRecord
	backtestPromotionCandidates    []storage.MarketOpsBacktestPromotionCandidateRecord
	backtestCalibrationReadiness   []storage.MarketOpsBacktestCalibrationReadinessRecord
	syncraticContextWindows        []storage.SyncraticContextWindowRecord
	syncraticInsights              []storage.SyncraticInsightRecord
	algorithmDefinitions           []storage.AlgorithmDefinitionRecord
	algorithmExecutionRequests     []storage.AlgorithmExecutionRequestRecord
	algorithmResults               []storage.AlgorithmResultRecord
	lastBacktestRunFilter          storage.MarketOpsBacktestRunFilter
	lastBacktestCoverageFilter     storage.MarketOpsBacktestCoverageFilter
	lastBacktestCampaignFilter     storage.MarketOpsBacktestCampaignFilter
	lastBacktestSignalFilter       storage.MarketOpsBacktestSignalFilter
	lastBacktestGraphFilter        storage.MarketOpsBacktestGraphProposalFilter
	lastBacktestCalibrationFilter  storage.MarketOpsBacktestCalibrationSummaryFilter
	lastBacktestBaselineFilter     storage.MarketOpsBacktestCalibrationBaselineFilter
	lastBacktestComparisonFilter   storage.MarketOpsBacktestCalibrationComparisonFilter
	lastEvaluationLabelFilter      storage.MarketOpsBacktestEvaluationLabelFilter
	lastBacktestEvaluationFilter   storage.MarketOpsBacktestEvaluationFilter
	lastBacktestPromotionFilter    storage.MarketOpsBacktestPromotionCandidateFilter
	lastBacktestReadinessFilter    storage.MarketOpsBacktestCalibrationReadinessFilter
	lastAlgorithmDefinitionFilter  storage.AlgorithmDefinitionFilter
	lastAlgorithmExecutionFilter   storage.AlgorithmExecutionRequestFilter
	lastAlgorithmResultFilter      storage.AlgorithmResultFilter
	lastDSMFilter                  storage.MarketOpsDSMArtifactFilter
	lastGraphProposalFilter        storage.MarketOpsDSMGraphProposalFilter
	lastGraphProposalMutation      storage.MarketOpsDSMGraphProposalMutation
	lastUniverseGroup              string
	lastActiveOnly                 bool
	signals                        []storage.SignalLedgerRecord
	alerts                         []storage.AlertLedgerRecord
	insights                       []storage.InsightLedgerRecord
	notFound                       bool
	lastFilter                     storage.RawEventLedgerFilter
	schedulerQueries               int
	rawEventQueries                int
	usageQueries                   int
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
	return q.signals, nil
}
func (q *fakeQueryRepository) GetSignalLedger(_ context.Context, signalID string) (storage.SignalLedgerRecord, error) {
	for _, signal := range q.signals {
		if signal.SignalID == signalID {
			return signal, nil
		}
	}
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

func (q *fakeQueryRepository) ListMarketOpsBacktestCoverage(_ context.Context, filter storage.MarketOpsBacktestCoverageFilter) ([]storage.MarketOpsBacktestCoverageRecord, error) {
	q.lastBacktestCoverageFilter = filter
	return q.backtestCoverage, nil
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

func (q *fakeQueryRepository) UpsertMarketOpsBacktestCampaign(_ context.Context, record storage.MarketOpsBacktestCampaignRecord) error {
	for i, existing := range q.backtestCampaigns {
		if existing.CampaignID == record.CampaignID {
			q.backtestCampaigns[i] = record
			return nil
		}
	}
	q.backtestCampaigns = append(q.backtestCampaigns, record)
	return nil
}

func (q *fakeQueryRepository) ListMarketOpsBacktestCampaigns(_ context.Context, filter storage.MarketOpsBacktestCampaignFilter) ([]storage.MarketOpsBacktestCampaignRecord, error) {
	q.lastBacktestCampaignFilter = filter
	return q.backtestCampaigns, nil
}

func (q *fakeQueryRepository) GetMarketOpsBacktestCampaign(_ context.Context, campaignID string) (storage.MarketOpsBacktestCampaignRecord, error) {
	if q.notFound {
		return storage.MarketOpsBacktestCampaignRecord{}, storage.ErrNotFound
	}
	for _, campaign := range q.backtestCampaigns {
		if campaign.CampaignID == campaignID {
			return campaign, nil
		}
	}
	return storage.MarketOpsBacktestCampaignRecord{}, storage.ErrNotFound
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

func (q *fakeQueryRepository) UpsertMarketOpsBacktestCalibrationSummary(_ context.Context, record storage.MarketOpsBacktestCalibrationSummaryRecord) error {
	for i, existing := range q.backtestCalibrationSummaries {
		if existing.SummaryID == record.SummaryID {
			record.CreatedAt = existing.CreatedAt
			q.backtestCalibrationSummaries[i] = record
			return nil
		}
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}
	q.backtestCalibrationSummaries = append(q.backtestCalibrationSummaries, record)
	return nil
}

func (q *fakeQueryRepository) ListMarketOpsBacktestCalibrationSummaries(_ context.Context, filter storage.MarketOpsBacktestCalibrationSummaryFilter) ([]storage.MarketOpsBacktestCalibrationSummaryRecord, error) {
	q.lastBacktestCalibrationFilter = filter
	return q.backtestCalibrationSummaries, nil
}

func (q *fakeQueryRepository) GetMarketOpsBacktestCalibrationSummary(_ context.Context, summaryID string) (storage.MarketOpsBacktestCalibrationSummaryRecord, error) {
	if q.notFound {
		return storage.MarketOpsBacktestCalibrationSummaryRecord{}, storage.ErrNotFound
	}
	for _, summary := range q.backtestCalibrationSummaries {
		if summary.SummaryID == summaryID {
			return summary, nil
		}
	}
	return storage.MarketOpsBacktestCalibrationSummaryRecord{}, storage.ErrNotFound
}

func (q *fakeQueryRepository) UpsertMarketOpsBacktestCalibrationBaseline(_ context.Context, record storage.MarketOpsBacktestCalibrationBaselineRecord) error {
	for i, existing := range q.backtestCalibrationBaselines {
		if existing.BaselineID == record.BaselineID {
			record.CreatedAt = existing.CreatedAt
			if record.UpdatedAt.IsZero() {
				record.UpdatedAt = time.Now().UTC()
			}
			q.backtestCalibrationBaselines[i] = record
			return nil
		}
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}
	if record.UpdatedAt.IsZero() {
		record.UpdatedAt = record.CreatedAt
	}
	q.backtestCalibrationBaselines = append(q.backtestCalibrationBaselines, record)
	return nil
}

func (q *fakeQueryRepository) ListMarketOpsBacktestCalibrationBaselines(_ context.Context, filter storage.MarketOpsBacktestCalibrationBaselineFilter) ([]storage.MarketOpsBacktestCalibrationBaselineRecord, error) {
	q.lastBacktestBaselineFilter = filter
	return q.backtestCalibrationBaselines, nil
}

func (q *fakeQueryRepository) GetMarketOpsBacktestCalibrationBaseline(_ context.Context, baselineID string) (storage.MarketOpsBacktestCalibrationBaselineRecord, error) {
	if q.notFound {
		return storage.MarketOpsBacktestCalibrationBaselineRecord{}, storage.ErrNotFound
	}
	for _, baseline := range q.backtestCalibrationBaselines {
		if baseline.BaselineID == baselineID {
			return baseline, nil
		}
	}
	return storage.MarketOpsBacktestCalibrationBaselineRecord{}, storage.ErrNotFound
}

func (q *fakeQueryRepository) UpsertMarketOpsBacktestCalibrationComparison(_ context.Context, record storage.MarketOpsBacktestCalibrationComparisonRecord) error {
	for i, existing := range q.backtestCalibrationComparisons {
		if existing.ComparisonID == record.ComparisonID {
			record.CreatedAt = existing.CreatedAt
			q.backtestCalibrationComparisons[i] = record
			return nil
		}
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}
	q.backtestCalibrationComparisons = append(q.backtestCalibrationComparisons, record)
	return nil
}

func (q *fakeQueryRepository) ListMarketOpsBacktestCalibrationComparisons(_ context.Context, filter storage.MarketOpsBacktestCalibrationComparisonFilter) ([]storage.MarketOpsBacktestCalibrationComparisonRecord, error) {
	q.lastBacktestComparisonFilter = filter
	return q.backtestCalibrationComparisons, nil
}

func (q *fakeQueryRepository) GetMarketOpsBacktestCalibrationComparison(_ context.Context, comparisonID string) (storage.MarketOpsBacktestCalibrationComparisonRecord, error) {
	if q.notFound {
		return storage.MarketOpsBacktestCalibrationComparisonRecord{}, storage.ErrNotFound
	}
	for _, comparison := range q.backtestCalibrationComparisons {
		if comparison.ComparisonID == comparisonID {
			return comparison, nil
		}
	}
	return storage.MarketOpsBacktestCalibrationComparisonRecord{}, storage.ErrNotFound
}

func (q *fakeQueryRepository) UpsertMarketOpsBacktestEvaluation(_ context.Context, record storage.MarketOpsBacktestEvaluationRecord) error {
	for i, existing := range q.backtestEvaluations {
		if existing.EvaluationID == record.EvaluationID {
			record.CreatedAt = existing.CreatedAt
			q.backtestEvaluations[i] = record
			return nil
		}
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}
	q.backtestEvaluations = append(q.backtestEvaluations, record)
	return nil
}

func (q *fakeQueryRepository) ListMarketOpsBacktestEvaluations(_ context.Context, filter storage.MarketOpsBacktestEvaluationFilter) ([]storage.MarketOpsBacktestEvaluationRecord, error) {
	q.lastBacktestEvaluationFilter = filter
	return q.backtestEvaluations, nil
}

func (q *fakeQueryRepository) GetMarketOpsBacktestEvaluation(_ context.Context, evaluationID string) (storage.MarketOpsBacktestEvaluationRecord, error) {
	if q.notFound {
		return storage.MarketOpsBacktestEvaluationRecord{}, storage.ErrNotFound
	}
	for _, evaluation := range q.backtestEvaluations {
		if evaluation.EvaluationID == evaluationID {
			return evaluation, nil
		}
	}
	return storage.MarketOpsBacktestEvaluationRecord{}, storage.ErrNotFound
}

func (q *fakeQueryRepository) UpsertMarketOpsBacktestPromotionCandidate(_ context.Context, record storage.MarketOpsBacktestPromotionCandidateRecord) error {
	for i, existing := range q.backtestPromotionCandidates {
		if existing.CandidateID == record.CandidateID {
			record.CreatedAt = existing.CreatedAt
			if record.UpdatedAt.IsZero() {
				record.UpdatedAt = time.Now().UTC()
			}
			q.backtestPromotionCandidates[i] = record
			return nil
		}
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}
	if record.UpdatedAt.IsZero() {
		record.UpdatedAt = record.CreatedAt
	}
	q.backtestPromotionCandidates = append(q.backtestPromotionCandidates, record)
	return nil
}

func (q *fakeQueryRepository) ListMarketOpsBacktestPromotionCandidates(_ context.Context, filter storage.MarketOpsBacktestPromotionCandidateFilter) ([]storage.MarketOpsBacktestPromotionCandidateRecord, error) {
	q.lastBacktestPromotionFilter = filter
	return q.backtestPromotionCandidates, nil
}

func (q *fakeQueryRepository) GetMarketOpsBacktestPromotionCandidate(_ context.Context, candidateID string) (storage.MarketOpsBacktestPromotionCandidateRecord, error) {
	if q.notFound {
		return storage.MarketOpsBacktestPromotionCandidateRecord{}, storage.ErrNotFound
	}
	for _, candidate := range q.backtestPromotionCandidates {
		if candidate.CandidateID == candidateID {
			return candidate, nil
		}
	}
	return storage.MarketOpsBacktestPromotionCandidateRecord{}, storage.ErrNotFound
}

func (q *fakeQueryRepository) MutateMarketOpsBacktestPromotionCandidateDecision(_ context.Context, mutation storage.MarketOpsBacktestPromotionCandidateDecisionMutation) (storage.MarketOpsBacktestPromotionCandidateRecord, error) {
	for i, candidate := range q.backtestPromotionCandidates {
		if candidate.CandidateID == mutation.CandidateID {
			candidate.Status = mutation.Status
			candidate.ReviewedBy = mutation.ReviewedBy
			candidate.ReviewedAt = &mutation.ReviewedAt
			candidate.DecisionNote = mutation.DecisionNote
			candidate.UpdatedAt = mutation.ReviewedAt
			q.backtestPromotionCandidates[i] = candidate
			return candidate, nil
		}
	}
	return storage.MarketOpsBacktestPromotionCandidateRecord{}, storage.ErrNotFound
}

func (q *fakeQueryRepository) UpsertMarketOpsBacktestCalibrationReadiness(_ context.Context, record storage.MarketOpsBacktestCalibrationReadinessRecord) error {
	for i, existing := range q.backtestCalibrationReadiness {
		if existing.ReadinessID == record.ReadinessID {
			record.CreatedAt = existing.CreatedAt
			q.backtestCalibrationReadiness[i] = record
			return nil
		}
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}
	q.backtestCalibrationReadiness = append(q.backtestCalibrationReadiness, record)
	return nil
}

func (q *fakeQueryRepository) ListMarketOpsBacktestCalibrationReadiness(_ context.Context, filter storage.MarketOpsBacktestCalibrationReadinessFilter) ([]storage.MarketOpsBacktestCalibrationReadinessRecord, error) {
	q.lastBacktestReadinessFilter = filter
	return q.backtestCalibrationReadiness, nil
}

func (q *fakeQueryRepository) GetMarketOpsBacktestCalibrationReadiness(_ context.Context, readinessID string) (storage.MarketOpsBacktestCalibrationReadinessRecord, error) {
	if q.notFound {
		return storage.MarketOpsBacktestCalibrationReadinessRecord{}, storage.ErrNotFound
	}
	for _, record := range q.backtestCalibrationReadiness {
		if record.ReadinessID == readinessID {
			return record, nil
		}
	}
	return storage.MarketOpsBacktestCalibrationReadinessRecord{}, storage.ErrNotFound
}

func (q *fakeQueryRepository) UpsertSyncraticContextWindow(_ context.Context, record storage.SyncraticContextWindowRecord) error {
	for i, existing := range q.syncraticContextWindows {
		if existing.ContextWindowID == record.ContextWindowID || existing.IdempotencyKey == record.IdempotencyKey {
			record.CreatedAt = existing.CreatedAt
			if record.UpdatedAt.IsZero() {
				record.UpdatedAt = time.Now().UTC()
			}
			q.syncraticContextWindows[i] = record
			return nil
		}
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}
	if record.UpdatedAt.IsZero() {
		record.UpdatedAt = record.CreatedAt
	}
	q.syncraticContextWindows = append(q.syncraticContextWindows, record)
	return nil
}

func (q *fakeQueryRepository) ListSyncraticContextWindows(_ context.Context, filter storage.SyncraticContextWindowFilter) ([]storage.SyncraticContextWindowRecord, error) {
	out := []storage.SyncraticContextWindowRecord{}
	for _, record := range q.syncraticContextWindows {
		if filter.TenantID != "" && record.TenantID != filter.TenantID {
			continue
		}
		if filter.SubjectSymbol != "" && record.SubjectSymbol != filter.SubjectSymbol {
			continue
		}
		if filter.ContextStrategy != "" && record.ContextStrategy != filter.ContextStrategy {
			continue
		}
		out = append(out, record)
	}
	return out, nil
}

func (q *fakeQueryRepository) GetSyncraticContextWindow(_ context.Context, contextWindowID string) (storage.SyncraticContextWindowRecord, error) {
	if q.notFound {
		return storage.SyncraticContextWindowRecord{}, storage.ErrNotFound
	}
	for _, record := range q.syncraticContextWindows {
		if record.ContextWindowID == contextWindowID {
			return record, nil
		}
	}
	return storage.SyncraticContextWindowRecord{}, storage.ErrNotFound
}

func (q *fakeQueryRepository) UpsertSyncraticInsight(_ context.Context, record storage.SyncraticInsightRecord) error {
	for i, existing := range q.syncraticInsights {
		if existing.SyncraticInsightID == record.SyncraticInsightID || (existing.ContextWindowID == record.ContextWindowID && existing.InsightType == record.InsightType && existing.BuilderVersion == record.BuilderVersion) {
			record.CreatedAt = existing.CreatedAt
			if record.UpdatedAt.IsZero() {
				record.UpdatedAt = time.Now().UTC()
			}
			q.syncraticInsights[i] = record
			return nil
		}
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}
	if record.UpdatedAt.IsZero() {
		record.UpdatedAt = record.CreatedAt
	}
	q.syncraticInsights = append(q.syncraticInsights, record)
	return nil
}

func (q *fakeQueryRepository) ListSyncraticInsights(_ context.Context, filter storage.SyncraticInsightFilter) ([]storage.SyncraticInsightRecord, error) {
	out := []storage.SyncraticInsightRecord{}
	for _, record := range q.syncraticInsights {
		if filter.TenantID != "" && record.TenantID != filter.TenantID {
			continue
		}
		if filter.ContextWindowID != "" && record.ContextWindowID != filter.ContextWindowID {
			continue
		}
		if filter.SubjectSymbol != "" && record.SubjectSymbol != filter.SubjectSymbol {
			continue
		}
		out = append(out, record)
	}
	return out, nil
}

func (q *fakeQueryRepository) GetSyncraticInsight(_ context.Context, syncraticInsightID string) (storage.SyncraticInsightRecord, error) {
	if q.notFound {
		return storage.SyncraticInsightRecord{}, storage.ErrNotFound
	}
	for _, record := range q.syncraticInsights {
		if record.SyncraticInsightID == syncraticInsightID {
			return record, nil
		}
	}
	return storage.SyncraticInsightRecord{}, storage.ErrNotFound
}

func (q *fakeQueryRepository) UpsertMarketOpsBacktestEvaluationLabel(_ context.Context, record storage.MarketOpsBacktestEvaluationLabelRecord) error {
	for i, existing := range q.backtestEvaluationLabels {
		if existing.LabelID == record.LabelID || (existing.SourceProposalID == record.SourceProposalID && existing.LabelVersion == record.LabelVersion) {
			record.CreatedAt = existing.CreatedAt
			if record.UpdatedAt.IsZero() {
				record.UpdatedAt = time.Now().UTC()
			}
			q.backtestEvaluationLabels[i] = record
			return nil
		}
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}
	if record.UpdatedAt.IsZero() {
		record.UpdatedAt = record.CreatedAt
	}
	q.backtestEvaluationLabels = append(q.backtestEvaluationLabels, record)
	return nil
}

func (q *fakeQueryRepository) ListMarketOpsBacktestEvaluationLabels(_ context.Context, filter storage.MarketOpsBacktestEvaluationLabelFilter) ([]storage.MarketOpsBacktestEvaluationLabelRecord, error) {
	q.lastEvaluationLabelFilter = filter
	return q.backtestEvaluationLabels, nil
}

func (q *fakeQueryRepository) GetMarketOpsBacktestEvaluationLabel(_ context.Context, labelID string) (storage.MarketOpsBacktestEvaluationLabelRecord, error) {
	if q.notFound {
		return storage.MarketOpsBacktestEvaluationLabelRecord{}, storage.ErrNotFound
	}
	for _, label := range q.backtestEvaluationLabels {
		if label.LabelID == labelID {
			return label, nil
		}
	}
	return storage.MarketOpsBacktestEvaluationLabelRecord{}, storage.ErrNotFound
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

func (q *fakeQueryRepository) UpsertAlgorithmDefinition(_ context.Context, record storage.AlgorithmDefinitionRecord) error {
	for i, existing := range q.algorithmDefinitions {
		if existing.TenantID == record.TenantID && existing.AlgorithmID == record.AlgorithmID {
			record.CreatedAt = existing.CreatedAt
			q.algorithmDefinitions[i] = record
			return nil
		}
	}
	q.algorithmDefinitions = append(q.algorithmDefinitions, record)
	return nil
}

func (q *fakeQueryRepository) ListAlgorithmDefinitions(_ context.Context, filter storage.AlgorithmDefinitionFilter) ([]storage.AlgorithmDefinitionRecord, error) {
	q.lastAlgorithmDefinitionFilter = filter
	return q.algorithmDefinitions, nil
}

func (q *fakeQueryRepository) GetAlgorithmDefinition(_ context.Context, tenantID string, algorithmID string) (storage.AlgorithmDefinitionRecord, error) {
	for _, record := range q.algorithmDefinitions {
		if record.TenantID == tenantID && record.AlgorithmID == algorithmID {
			return record, nil
		}
	}
	return storage.AlgorithmDefinitionRecord{}, storage.ErrNotFound
}

func (q *fakeQueryRepository) UpsertAlgorithmExecutionRequest(_ context.Context, record storage.AlgorithmExecutionRequestRecord) error {
	for i, existing := range q.algorithmExecutionRequests {
		if existing.TenantID == record.TenantID && existing.ExecutionRequestID == record.ExecutionRequestID {
			record.CreatedAt = existing.CreatedAt
			q.algorithmExecutionRequests[i] = record
			return nil
		}
	}
	q.algorithmExecutionRequests = append(q.algorithmExecutionRequests, record)
	return nil
}

func (q *fakeQueryRepository) ListAlgorithmExecutionRequests(_ context.Context, filter storage.AlgorithmExecutionRequestFilter) ([]storage.AlgorithmExecutionRequestRecord, error) {
	q.lastAlgorithmExecutionFilter = filter
	return q.algorithmExecutionRequests, nil
}

func (q *fakeQueryRepository) GetAlgorithmExecutionRequest(_ context.Context, tenantID string, executionRequestID string) (storage.AlgorithmExecutionRequestRecord, error) {
	for _, record := range q.algorithmExecutionRequests {
		if record.TenantID == tenantID && record.ExecutionRequestID == executionRequestID {
			return record, nil
		}
	}
	return storage.AlgorithmExecutionRequestRecord{}, storage.ErrNotFound
}

func (q *fakeQueryRepository) InsertAlgorithmResult(_ context.Context, record storage.AlgorithmResultRecord) error {
	for _, existing := range q.algorithmResults {
		if existing.TenantID == record.TenantID && existing.AlgorithmResultID == record.AlgorithmResultID {
			return nil
		}
	}
	q.algorithmResults = append(q.algorithmResults, record)
	return nil
}

func (q *fakeQueryRepository) ListAlgorithmResults(_ context.Context, filter storage.AlgorithmResultFilter) ([]storage.AlgorithmResultRecord, error) {
	q.lastAlgorithmResultFilter = filter
	return q.algorithmResults, nil
}

func (q *fakeQueryRepository) GetAlgorithmResult(_ context.Context, tenantID string, algorithmResultID string) (storage.AlgorithmResultRecord, error) {
	for _, record := range q.algorithmResults {
		if record.TenantID == tenantID && record.AlgorithmResultID == algorithmResultID {
			return record, nil
		}
	}
	return storage.AlgorithmResultRecord{}, storage.ErrNotFound
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

func TestGetMarketOpsBacktestCoverageUsesDefaultMarketOpsFilters(t *testing.T) {
	now := time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC)
	repo := &fakeQueryRepository{backtestCoverage: []storage.MarketOpsBacktestCoverageRecord{{TenantID: "tenant-local", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SourceID: "src-massive", SourceAdapter: "market_data.massive", Dataset: "equity_eod_prices", SubjectSymbol: "AAPL", EventCount: 2, FirstObserved: now, LastObserved: now.Add(24 * time.Hour)}}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodGet, "/v1/marketops/backtest-coverage?tenant_id=tenant-local&symbols=aapl,msft&window_start=2026-07-09T00:00:00Z&window_end=2026-07-11T00:00:00Z&limit=7", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	filter := repo.lastBacktestCoverageFilter
	if filter.TenantID != "tenant-local" || filter.AppID != "marketops" || filter.Domain != "market_data" || filter.UseCase != "daily_market_surveillance" || filter.Limit != 7 {
		t.Fatalf("filter = %+v", filter)
	}
	if len(filter.Symbols) != 2 || filter.Symbols[0] != "AAPL" || filter.Symbols[1] != "MSFT" {
		t.Fatalf("symbols = %+v", filter.Symbols)
	}
	var response map[string][]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("response JSON error = %v", err)
	}
	if len(response["coverage"]) != 1 || response["coverage"][0]["event_count"].(float64) != 2 {
		t.Fatalf("response = %+v", response)
	}
}

func TestGetMarketOpsBacktestCoverageRejectsInvalidWindow(t *testing.T) {
	router := NewRouter(RouterConfig{QueryRepository: &fakeQueryRepository{}})
	req := httptest.NewRequest(http.MethodGet, "/v1/marketops/backtest-coverage?tenant_id=tenant-local&window_start=2026-07-11T00:00:00Z&window_end=2026-07-09T00:00:00Z", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
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

func TestPostMarketOpsBacktestCampaignCreatesBoundedChildRuns(t *testing.T) {
	repo := &fakeQueryRepository{}
	seen := []marketopsbacktest.Config{}
	runner := func(ctx context.Context, repo storage.MarketOpsBacktestRepository, cfg marketopsbacktest.Config) (marketopsbacktest.Result, error) {
		seen = append(seen, cfg)
		now := time.Date(2026, 7, 12, 10, 0, 0, 0, time.UTC)
		completed := now.Add(time.Second)
		record := storage.MarketOpsBacktestRunRecord{RunID: cfg.RunID, TenantID: cfg.TenantID, AppID: cfg.AppID, Domain: cfg.Domain, UseCase: cfg.UseCase, SourceID: cfg.SourceID, SourceAdapter: cfg.SourceAdapter, Dataset: cfg.Dataset, DetectorID: cfg.DetectorID, Status: storage.RunStatusSucceeded, RequestedBy: cfg.RequestedBy, WindowStart: cfg.WindowStart, WindowEnd: cfg.WindowEnd, StartedAt: now, CompletedAt: &completed, MetricsJSON: []byte(`{"scanned":2}`), CreatedAt: now, UpdatedAt: completed}
		return marketopsbacktest.Result{Run: record, Metrics: marketopsbacktest.Metrics{RunID: cfg.RunID, Scanned: 2, Signals: 1, Artifacts: 1, GraphProposals: 1, PolicyResults: 1, RecommendationCounts: map[string]int{storage.MarketOpsBacktestPolicyManualReviewRequired: 1}}}, nil
	}
	router := NewRouter(RouterConfig{QueryRepository: repo, MarketOpsBacktestRunner: runner})
	body := `{"campaign_id":"campaign-1","tenant_id":"tenant-local","source_id":"src-massive","dataset_scope":["equity_eod_prices"],"symbols":["aapl","msft"],"window_start":"2026-07-09T00:00:00Z","window_end":"2026-07-11T00:00:00Z","max_symbols":2,"max_windows":2,"max_runs":3,"max_records":5}`
	req := httptest.NewRequest(http.MethodPost, "/v1/marketops/backtest-campaigns", strings.NewReader(body))
	req.Header.Set("X-SignalOps-Actor", "operator-api")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(seen) != 3 {
		t.Fatalf("child run count = %d", len(seen))
	}
	if seen[0].Symbols[0] != "AAPL" || seen[2].Symbols[0] != "MSFT" {
		t.Fatalf("child configs = %+v", seen)
	}
	var response marketOpsBacktestCampaignCreateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("response JSON error = %v", err)
	}
	if response.Campaign.Status != storage.RunStatusSucceeded || len(response.Campaign.ChildRunIDs) != 3 {
		t.Fatalf("campaign response = %+v", response.Campaign)
	}
	var metrics map[string]any
	if err := json.Unmarshal(response.Campaign.Metrics, &metrics); err != nil {
		t.Fatalf("metrics JSON error = %v", err)
	}
	if metrics["completed_runs"].(float64) != 3 || metrics["scanned"].(float64) != 6 {
		t.Fatalf("metrics = %+v", metrics)
	}
}

func TestPostMarketOpsBacktestCampaignResolvesUniverseSymbols(t *testing.T) {
	repo := &fakeQueryRepository{marketOpsAssets: []storage.MarketOpsAssetRecord{{Ticker: "NVDA"}, {Ticker: "MSFT"}}}
	seen := []marketopsbacktest.Config{}
	runner := func(ctx context.Context, repo storage.MarketOpsBacktestRepository, cfg marketopsbacktest.Config) (marketopsbacktest.Result, error) {
		seen = append(seen, cfg)
		now := time.Now().UTC()
		return marketopsbacktest.Result{Run: storage.MarketOpsBacktestRunRecord{RunID: cfg.RunID, TenantID: cfg.TenantID, AppID: cfg.AppID, Domain: cfg.Domain, UseCase: cfg.UseCase, DetectorID: cfg.DetectorID, Status: storage.RunStatusSucceeded, StartedAt: now, CreatedAt: now, UpdatedAt: now}, Metrics: marketopsbacktest.Metrics{RunID: cfg.RunID, RecommendationCounts: map[string]int{}}}, nil
	}
	router := NewRouter(RouterConfig{QueryRepository: repo, MarketOpsBacktestRunner: runner})
	body := `{"tenant_id":"tenant-local","universe_group":"top50_megacap","window_start":"2026-07-09T00:00:00Z","window_end":"2026-07-10T00:00:00Z","max_symbols":1,"max_runs":1}`
	req := httptest.NewRequest(http.MethodPost, "/v1/marketops/backtest-campaigns", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if repo.lastUniverseGroup != "top50_megacap" || !repo.lastActiveOnly {
		t.Fatalf("universe lookup = %q/%t", repo.lastUniverseGroup, repo.lastActiveOnly)
	}
	if len(seen) != 1 || seen[0].Symbols[0] != "NVDA" {
		t.Fatalf("seen = %+v", seen)
	}
}

func TestPostMarketOpsBacktestCampaignRejectsUnboundedRequest(t *testing.T) {
	router := NewRouter(RouterConfig{QueryRepository: &fakeQueryRepository{}})
	body := `{"tenant_id":"tenant-local","symbols":["AAPL"],"window_start":"2026-07-09T00:00:00Z","window_end":"2026-07-10T00:00:00Z","max_runs":251}`
	req := httptest.NewRequest(http.MethodPost, "/v1/marketops/backtest-campaigns", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestGetMarketOpsBacktestCampaigns(t *testing.T) {
	now := time.Date(2026, 7, 12, 10, 0, 0, 0, time.UTC)
	repo := &fakeQueryRepository{backtestCampaigns: []storage.MarketOpsBacktestCampaignRecord{{CampaignID: "campaign-1", TenantID: "tenant-local", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", DetectorID: "marketops.dsm.taxonomy_v1", UniverseGroup: "top50_megacap", DatasetScope: []string{"equity_eod_prices"}, Symbols: []string{"AAPL"}, WindowStart: now, WindowEnd: now.Add(24 * time.Hour), Status: storage.RunStatusSucceeded, MetricsJSON: []byte(`{}`), StartedAt: now, CreatedAt: now, UpdatedAt: now}}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodGet, "/v1/marketops/backtest-campaigns?tenant_id=tenant-local&universe_group=top50_megacap&status=succeeded&limit=5", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if repo.lastBacktestCampaignFilter.TenantID != "tenant-local" || repo.lastBacktestCampaignFilter.UniverseGroup != "top50_megacap" || repo.lastBacktestCampaignFilter.Status != storage.RunStatusSucceeded || repo.lastBacktestCampaignFilter.Limit != 5 {
		t.Fatalf("filter = %+v", repo.lastBacktestCampaignFilter)
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

func TestPostMarketOpsBacktestEvaluationScoresRunAgainstLabels(t *testing.T) {
	proposal := validMarketOpsBacktestGraphProposalRecord()
	proposal.NodeID = "ticker:AAPL"
	policy := validMarketOpsBacktestPolicyResultRecord()
	policy.ProposalID = proposal.ProposalID
	policy.Recommendation = storage.MarketOpsBacktestPolicyAutoAcceptCandidate
	repo := &fakeQueryRepository{backtestRuns: []storage.MarketOpsBacktestRunRecord{validMarketOpsBacktestRunRecord()}, backtestGraphProposals: []storage.MarketOpsBacktestGraphProposalRecord{proposal}, backtestPolicyResults: []storage.MarketOpsBacktestPolicyResultRecord{policy}, backtestEvaluationLabels: []storage.MarketOpsBacktestEvaluationLabelRecord{validMarketOpsBacktestEvaluationLabelRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodPost, "/v1/marketops/backtest-evaluations", strings.NewReader(`{"evaluation_id":"bteval-1","tenant_id":"tenant-1","run_id":"bt-marketops-1"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	evaluation := response["backtest_evaluation"]
	if evaluation["evaluation_id"] != "bteval-1" || evaluation["true_positive"].(float64) != 1 || evaluation["precision"].(float64) != 1 {
		t.Fatalf("evaluation = %#v", evaluation)
	}
}

func TestGetMarketOpsBacktestEvaluations(t *testing.T) {
	repo := &fakeQueryRepository{backtestEvaluations: []storage.MarketOpsBacktestEvaluationRecord{validMarketOpsBacktestEvaluationRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodGet, "/v1/marketops/backtest-evaluations?tenant_id=tenant-1&run_id=bt-marketops-1&limit=10", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if repo.lastBacktestEvaluationFilter.TenantID != "tenant-1" || repo.lastBacktestEvaluationFilter.RunID != "bt-marketops-1" || repo.lastBacktestEvaluationFilter.Limit != 10 {
		t.Fatalf("filter = %+v", repo.lastBacktestEvaluationFilter)
	}
	var response map[string][]map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if len(response["backtest_evaluations"]) != 1 || response["backtest_evaluations"][0]["evaluation_id"] != "bteval-1" {
		t.Fatalf("response = %#v", response)
	}
}

func TestGetMarketOpsBacktestEvaluation(t *testing.T) {
	repo := &fakeQueryRepository{backtestEvaluations: []storage.MarketOpsBacktestEvaluationRecord{validMarketOpsBacktestEvaluationRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodGet, "/v1/marketops/backtest-evaluations/bteval-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response["backtest_evaluation"]["evaluation_id"] != "bteval-1" {
		t.Fatalf("response = %#v", response)
	}
}

func TestPostMarketOpsBacktestPromotionCandidateCreatesReadyForReviewCandidate(t *testing.T) {
	repo := &fakeQueryRepository{backtestCalibrationBaselines: []storage.MarketOpsBacktestCalibrationBaselineRecord{validMarketOpsBacktestCalibrationBaselineRecord()}, backtestCalibrationComparisons: []storage.MarketOpsBacktestCalibrationComparisonRecord{validMarketOpsBacktestCalibrationComparisonRecord()}, backtestEvaluations: []storage.MarketOpsBacktestEvaluationRecord{validMarketOpsBacktestEvaluationRecord()}, backtestRuns: []storage.MarketOpsBacktestRunRecord{validMarketOpsBacktestRunRecord()}, backtestPolicyResults: []storage.MarketOpsBacktestPolicyResultRecord{validMarketOpsBacktestPolicyResultRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	body := `{"candidate_id":"btpromo-1","tenant_id":"tenant-1","baseline_id":"btbase-1","comparison_id":"btcmp-1","evaluation_id":"bteval-1","candidate_version":"taxonomy-v1-policy-v1-test"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/marketops/backtest-promotion-candidates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	candidate := response["promotion_candidate"]
	if candidate["candidate_id"] != "btpromo-1" || candidate["readiness_status"] != storage.MarketOpsBacktestPromotionReadinessReadyForReview || candidate["status"] != storage.MarketOpsBacktestPromotionCandidateStatusProposed {
		t.Fatalf("candidate = %#v", candidate)
	}
	if candidate["policy_version"] != "marketops.backtest.policy_v1" || candidate["run_id"] != "bt-marketops-1" {
		t.Fatalf("candidate evidence refs = %#v", candidate)
	}
}

func TestPostMarketOpsBacktestPromotionCandidateWithoutEvaluationRequiresManualReview(t *testing.T) {
	repo := &fakeQueryRepository{backtestCalibrationBaselines: []storage.MarketOpsBacktestCalibrationBaselineRecord{validMarketOpsBacktestCalibrationBaselineRecord()}, backtestCalibrationComparisons: []storage.MarketOpsBacktestCalibrationComparisonRecord{validMarketOpsBacktestCalibrationComparisonRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	body := `{"candidate_id":"btpromo-1","tenant_id":"tenant-1","baseline_id":"btbase-1","comparison_id":"btcmp-1"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/marketops/backtest-promotion-candidates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	candidate := response["promotion_candidate"]
	if candidate["readiness_status"] != storage.MarketOpsBacktestPromotionReadinessManualReviewRequired {
		t.Fatalf("candidate = %#v", candidate)
	}
}

func TestGetMarketOpsBacktestPromotionCandidates(t *testing.T) {
	repo := &fakeQueryRepository{backtestPromotionCandidates: []storage.MarketOpsBacktestPromotionCandidateRecord{validMarketOpsBacktestPromotionCandidateRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodGet, "/v1/marketops/backtest-promotion-candidates?tenant_id=tenant-1&baseline_id=btbase-1&readiness_status=ready_for_review&status=proposed&limit=10", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if repo.lastBacktestPromotionFilter.TenantID != "tenant-1" || repo.lastBacktestPromotionFilter.BaselineID != "btbase-1" || repo.lastBacktestPromotionFilter.Limit != 10 {
		t.Fatalf("filter = %+v", repo.lastBacktestPromotionFilter)
	}
	var response map[string][]map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if len(response["promotion_candidates"]) != 1 || response["promotion_candidates"][0]["candidate_id"] != "btpromo-1" {
		t.Fatalf("response = %#v", response)
	}
}

func TestGetMarketOpsBacktestPromotionCandidate(t *testing.T) {
	repo := &fakeQueryRepository{backtestPromotionCandidates: []storage.MarketOpsBacktestPromotionCandidateRecord{validMarketOpsBacktestPromotionCandidateRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodGet, "/v1/marketops/backtest-promotion-candidates/btpromo-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response["promotion_candidate"]["candidate_id"] != "btpromo-1" {
		t.Fatalf("response = %#v", response)
	}
}

func TestPostMarketOpsBacktestPromotionCandidateDecision(t *testing.T) {
	repo := &fakeQueryRepository{backtestPromotionCandidates: []storage.MarketOpsBacktestPromotionCandidateRecord{validMarketOpsBacktestPromotionCandidateRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodPost, "/v1/marketops/backtest-promotion-candidates/btpromo-1/decision", strings.NewReader(`{"status":"approved_for_promotion","reviewed_by":"operator-test","decision_note":"approved for deployment planning only"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	candidate := response["promotion_candidate"]
	if candidate["status"] != storage.MarketOpsBacktestPromotionCandidateStatusApprovedForPromotion || candidate["decision_note"] != "approved for deployment planning only" {
		t.Fatalf("candidate = %#v", candidate)
	}
}

func TestPostMarketOpsBacktestPromotionCandidateDecisionRejectsProposedStatus(t *testing.T) {
	repo := &fakeQueryRepository{backtestPromotionCandidates: []storage.MarketOpsBacktestPromotionCandidateRecord{validMarketOpsBacktestPromotionCandidateRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodPost, "/v1/marketops/backtest-promotion-candidates/btpromo-1/decision", strings.NewReader(`{"status":"proposed"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPostMarketOpsBacktestCalibrationReadinessCreatesNeedsMoreDataSnapshot(t *testing.T) {
	repo := &fakeQueryRepository{
		backtestCalibrationBaselines:   []storage.MarketOpsBacktestCalibrationBaselineRecord{validMarketOpsBacktestCalibrationBaselineRecord()},
		backtestCalibrationComparisons: []storage.MarketOpsBacktestCalibrationComparisonRecord{validMarketOpsBacktestCalibrationComparisonRecord()},
		backtestEvaluations:            []storage.MarketOpsBacktestEvaluationRecord{validMarketOpsBacktestEvaluationRecord()},
		backtestPromotionCandidates:    []storage.MarketOpsBacktestPromotionCandidateRecord{validMarketOpsBacktestPromotionCandidateRecord()},
		backtestRuns:                   []storage.MarketOpsBacktestRunRecord{validMarketOpsBacktestRunRecord()},
		backtestEvaluationLabels:       []storage.MarketOpsBacktestEvaluationLabelRecord{validMarketOpsBacktestEvaluationLabelRecord()},
		marketOpsAssets:                []storage.MarketOpsAssetRecord{{TenantID: "tenant-1", UniverseGroup: "top50_megacap", Ticker: "AAPL", IsActive: true}, {TenantID: "tenant-1", UniverseGroup: "top50_megacap", Ticker: "MSFT", IsActive: true}},
	}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	body := `{"readiness_id":"btready-1","tenant_id":"tenant-1","baseline_id":"btbase-1","comparison_id":"btcmp-1","evaluation_id":"bteval-1","candidate_id":"btpromo-1","dataset_scope":["equity_eod_prices","options_contracts_daily"],"universe_group":"top50_megacap","thresholds":{"min_reviewed_labels":100}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/marketops/backtest-calibration-readiness", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	readiness := response["calibration_readiness"]
	if readiness["readiness_id"] != "btready-1" || readiness["readiness_status"] != storage.MarketOpsBacktestCalibrationReadinessNeedsMoreHistoricalData {
		t.Fatalf("readiness = %#v", readiness)
	}
	if repo.lastBacktestRunFilter.Status != storage.RunStatusSucceeded || repo.lastUniverseGroup != "top50_megacap" {
		t.Fatalf("filters run=%+v universe=%s", repo.lastBacktestRunFilter, repo.lastUniverseGroup)
	}
}

func TestGetMarketOpsBacktestCalibrationReadiness(t *testing.T) {
	repo := &fakeQueryRepository{backtestCalibrationReadiness: []storage.MarketOpsBacktestCalibrationReadinessRecord{validMarketOpsBacktestCalibrationReadinessRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodGet, "/v1/marketops/backtest-calibration-readiness/btready-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["calibration_readiness"]["readiness_id"] != "btready-1" {
		t.Fatalf("response = %s", rec.Body.String())
	}
}

func TestGetMarketOpsBacktestCalibrationReadinessList(t *testing.T) {
	repo := &fakeQueryRepository{backtestCalibrationReadiness: []storage.MarketOpsBacktestCalibrationReadinessRecord{validMarketOpsBacktestCalibrationReadinessRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodGet, "/v1/marketops/backtest-calibration-readiness?tenant_id=tenant-1&readiness_status=needs_more_labels&limit=10", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if repo.lastBacktestReadinessFilter.TenantID != "tenant-1" || repo.lastBacktestReadinessFilter.ReadinessStatus != storage.MarketOpsBacktestCalibrationReadinessNeedsMoreLabels || repo.lastBacktestReadinessFilter.Limit != 10 {
		t.Fatalf("filter = %+v", repo.lastBacktestReadinessFilter)
	}
}

func TestPostMarketOpsBacktestEvaluationLabelsSyncCreatesLabels(t *testing.T) {
	proposal := validMarketOpsDSMGraphProposalRecord()
	now := time.Date(2026, 7, 12, 20, 0, 0, 0, time.UTC)
	proposal.Status = storage.MarketOpsDSMGraphProposalStatusAccepted
	proposal.ReviewedBy = "operator-g080"
	proposal.DecisionNote = "approved"
	proposal.DecidedAt = &now
	proposal.UpdatedAt = now
	repo := &fakeQueryRepository{dsmGraphProposals: []storage.MarketOpsDSMGraphProposalRecord{proposal}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodPost, "/v1/marketops/backtest-evaluation-labels/sync", strings.NewReader(`{"tenant_id":"tenant-1","status":"accepted","limit":5}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if repo.lastGraphProposalFilter.TenantID != "tenant-1" || repo.lastGraphProposalFilter.Status != storage.MarketOpsDSMGraphProposalStatusAccepted || repo.lastGraphProposalFilter.Limit != 5 {
		t.Fatalf("graph filter = %+v", repo.lastGraphProposalFilter)
	}
	var response map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response["synced"].(float64) != 1 {
		t.Fatalf("response = %#v", response)
	}
	labels := response["labels"].([]any)
	label := labels[0].(map[string]any)
	if label["source_proposal_id"] != proposal.ProposalID || label["label"] != "positive" || label["decision_status"] != storage.MarketOpsDSMGraphProposalStatusAccepted {
		t.Fatalf("label = %#v", label)
	}
	if label["graph_fact_key"] != "node:ticker:AAPL" {
		t.Fatalf("graph fact key = %#v", label["graph_fact_key"])
	}
}

func TestGetMarketOpsBacktestEvaluationLabels(t *testing.T) {
	repo := &fakeQueryRepository{backtestEvaluationLabels: []storage.MarketOpsBacktestEvaluationLabelRecord{validMarketOpsBacktestEvaluationLabelRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodGet, "/v1/marketops/backtest-evaluation-labels?tenant_id=tenant-1&label=positive&decision_status=accepted&limit=10", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if repo.lastEvaluationLabelFilter.TenantID != "tenant-1" || repo.lastEvaluationLabelFilter.Label != "positive" || repo.lastEvaluationLabelFilter.Limit != 10 {
		t.Fatalf("filter = %+v", repo.lastEvaluationLabelFilter)
	}
	var response map[string][]map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if len(response["evaluation_labels"]) != 1 || response["evaluation_labels"][0]["label_id"] != "btlabel-1" {
		t.Fatalf("response = %#v", response)
	}
}

func TestGetMarketOpsBacktestEvaluationLabel(t *testing.T) {
	repo := &fakeQueryRepository{backtestEvaluationLabels: []storage.MarketOpsBacktestEvaluationLabelRecord{validMarketOpsBacktestEvaluationLabelRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodGet, "/v1/marketops/backtest-evaluation-labels/btlabel-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response["evaluation_label"]["label_id"] != "btlabel-1" {
		t.Fatalf("response = %#v", response)
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

func validMarketOpsBacktestCalibrationReadinessRecord() storage.MarketOpsBacktestCalibrationReadinessRecord {
	now := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	return storage.MarketOpsBacktestCalibrationReadinessRecord{ReadinessID: "btready-1", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", BaselineID: "btbase-1", ComparisonID: "btcmp-1", EvaluationID: "bteval-1", CandidateID: "btpromo-1", DetectorID: "marketops.dsm.taxonomy_v1", DatasetScope: []string{"equity_eod_prices"}, UniverseGroup: "top50_megacap", ReadinessStatus: storage.MarketOpsBacktestCalibrationReadinessNeedsMoreLabels, ReadinessReasons: []string{"reviewed label volume or label coverage is below readiness thresholds"}, CoverageMetricsJSON: []byte(`{"symbol_coverage_ratio":0.02}`), LabelMetricsJSON: []byte(`{"matched_label_count":1}`), EvaluationMetricsJSON: []byte(`{"label_coverage":1}`), ThresholdsJSON: []byte(`{"min_reviewed_labels":100}`), EvidenceJSON: []byte(`{"deployment_block":"calibration readiness is advisory"}`), RequestedBy: "operator-test", CreatedAt: now}
}

func validMarketOpsBacktestPromotionCandidateRecord() storage.MarketOpsBacktestPromotionCandidateRecord {
	now := time.Date(2026, 7, 12, 21, 30, 0, 0, time.UTC)
	return storage.MarketOpsBacktestPromotionCandidateRecord{CandidateID: "btpromo-1", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", BaselineID: "btbase-1", ComparisonID: "btcmp-1", EvaluationID: "bteval-1", RunID: "bt-marketops-1", DetectorID: "marketops.dsm.taxonomy_v1", DetectorVersion: "0.1.0", Dataset: "equity_eod_prices", PolicyVersion: "marketops.backtest.policy_v1", CandidateVersion: "taxonomy-v1-policy-v1-test", ReadinessStatus: storage.MarketOpsBacktestPromotionReadinessReadyForReview, ReadinessReasons: []string{"comparison and evaluation evidence meet review thresholds"}, EvidenceJSON: []byte(`{"readiness":{"status":"ready_for_review"}}`), Status: storage.MarketOpsBacktestPromotionCandidateStatusProposed, RequestedBy: "operator-test", CreatedAt: now, UpdatedAt: now}
}

func validMarketOpsBacktestEvaluationRecord() storage.MarketOpsBacktestEvaluationRecord {
	now := time.Date(2026, 7, 12, 20, 30, 0, 0, time.UTC)
	return storage.MarketOpsBacktestEvaluationRecord{EvaluationID: "bteval-1", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", RunID: "bt-marketops-1", DetectorID: "marketops.dsm.taxonomy_v1", Dataset: "equity_eod_prices", LabelSource: "g080_graph_proposal_decision", LabelVersion: "marketops.eval_label.v1", ScoringVersion: "marketops.eval_scoring.v1", RequestedBy: "operator-test", CandidateCount: 1, LabeledCount: 1, PositiveCount: 1, TruePositive: 1, Precision: 1, Recall: 1, Accuracy: 1, LabelCoverage: 1, Recommendation: storage.MarketOpsBacktestCalibrationRecommendationImprovement, RecommendationNote: "automatic recommendations align with available labels", MetricsJSON: []byte(`{"matched_samples":[]}`), CreatedAt: now}
}

func validMarketOpsBacktestEvaluationLabelRecord() storage.MarketOpsBacktestEvaluationLabelRecord {
	now := time.Date(2026, 7, 12, 20, 5, 0, 0, time.UTC)
	return storage.MarketOpsBacktestEvaluationLabelRecord{LabelID: "btlabel-1", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SourceProposalID: "graphprop_marketops_dsm_v1_test", ArtifactID: "artifact_marketops_dsm_v1_test", SignalID: "signal-1", SubjectSymbol: "AAPL", CandidateType: "node_candidate", GraphFactKey: "node:ticker:AAPL", DecisionStatus: storage.MarketOpsDSMGraphProposalStatusAccepted, Label: "positive", LabelSource: "g080_graph_proposal_decision", LabeledBy: "operator-g080", LabeledAt: now, LabelVersion: "marketops.eval_label.v1", MetadataJSON: []byte(`{"decision_note":"approved"}`), CreatedAt: now, UpdatedAt: now}
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

func TestPostMarketOpsBacktestCalibrationSummaryCreatesStoredSnapshot(t *testing.T) {
	repo := &fakeQueryRepository{backtestRuns: []storage.MarketOpsBacktestRunRecord{
		{RunID: "bt-1", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SourceID: "src-massive", Dataset: "equity_eod_prices", DetectorID: "marketops.dsm.taxonomy_v1", Status: storage.RunStatusSucceeded, MetricsJSON: []byte(`{"scanned":2,"signals":1,"artifacts":1,"graph_proposals":5,"policy_results":5,"recommendation_counts":{"auto_accept_candidate":5}}`)},
		{RunID: "bt-2", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SourceID: "src-massive", Dataset: "equity_eod_prices", DetectorID: "marketops.dsm.taxonomy_v1", Status: storage.RunStatusSucceeded, MetricsJSON: []byte(`{"scanned":0,"signals":0,"artifacts":0,"graph_proposals":0,"policy_results":0,"recommendation_counts":{}}`)},
	}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	body := `{"summary_id":"btcal-1","tenant_id":"tenant-1","dataset":"equity_eod_prices","detector_id":"marketops.dsm.taxonomy_v1","status":"succeeded","limit":25}`
	req := httptest.NewRequest(http.MethodPost, "/v1/marketops/backtest-calibration-summaries", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if repo.lastBacktestRunFilter.TenantID != "tenant-1" || repo.lastBacktestRunFilter.Dataset != "equity_eod_prices" || repo.lastBacktestRunFilter.Status != "succeeded" || repo.lastBacktestRunFilter.Limit != 25 {
		t.Fatalf("filter = %+v", repo.lastBacktestRunFilter)
	}
	var response map[string]map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	summary := response["calibration_summary"]
	if summary["summary_id"] != "btcal-1" || summary["run_count"].(float64) != 2 || summary["zero_input_count"].(float64) != 1 {
		t.Fatalf("summary = %#v", summary)
	}
	if summary["signal_yield"].(float64) != 0.5 || summary["policy_results_per_signal"].(float64) != 5 {
		t.Fatalf("rates = %#v", summary)
	}
	dominant := summary["dominant_recommendation"].(map[string]any)
	if dominant["key"] != storage.MarketOpsBacktestPolicyAutoAcceptCandidate || dominant["count"].(float64) != 5 {
		t.Fatalf("dominant = %#v", dominant)
	}
}

func TestGetMarketOpsBacktestCalibrationSummaries(t *testing.T) {
	repo := &fakeQueryRepository{backtestCalibrationSummaries: []storage.MarketOpsBacktestCalibrationSummaryRecord{validMarketOpsBacktestCalibrationSummaryRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodGet, "/v1/marketops/backtest-calibration-summaries?tenant_id=tenant-1&dataset=equity_eod_prices&detector_id=marketops.dsm.taxonomy_v1&limit=10", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if repo.lastBacktestCalibrationFilter.TenantID != "tenant-1" || repo.lastBacktestCalibrationFilter.Dataset != "equity_eod_prices" || repo.lastBacktestCalibrationFilter.Limit != 10 {
		t.Fatalf("filter = %+v", repo.lastBacktestCalibrationFilter)
	}
	var response map[string][]map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if len(response["calibration_summaries"]) != 1 || response["calibration_summaries"][0]["summary_id"] != "btcal-1" {
		t.Fatalf("response = %#v", response)
	}
}

func TestGetMarketOpsBacktestCalibrationSummary(t *testing.T) {
	repo := &fakeQueryRepository{backtestCalibrationSummaries: []storage.MarketOpsBacktestCalibrationSummaryRecord{validMarketOpsBacktestCalibrationSummaryRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodGet, "/v1/marketops/backtest-calibration-summaries/btcal-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response["calibration_summary"]["summary_id"] != "btcal-1" {
		t.Fatalf("response = %#v", response)
	}
}

func TestPostMarketOpsBacktestCalibrationBaselineCreatesStoredBaseline(t *testing.T) {
	repo := &fakeQueryRepository{backtestCalibrationSummaries: []storage.MarketOpsBacktestCalibrationSummaryRecord{validMarketOpsBacktestCalibrationSummaryRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	body := `{"baseline_id":"btbase-1","tenant_id":"tenant-1","name":"Taxonomy baseline","summary_id":"btcal-1","scope":{"symbols":["AAPL"]}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/marketops/backtest-calibration-baselines", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	baseline := response["calibration_baseline"]
	if baseline["baseline_id"] != "btbase-1" || baseline["summary_id"] != "btcal-1" || baseline["status"] != storage.MarketOpsBacktestCalibrationBaselineStatusActive {
		t.Fatalf("baseline = %#v", baseline)
	}
	if baseline["detector_id"] != "marketops.dsm.taxonomy_v1" || baseline["dataset"] != "equity_eod_prices" {
		t.Fatalf("baseline summary linkage = %#v", baseline)
	}
}

func TestGetMarketOpsBacktestCalibrationBaselines(t *testing.T) {
	repo := &fakeQueryRepository{backtestCalibrationBaselines: []storage.MarketOpsBacktestCalibrationBaselineRecord{validMarketOpsBacktestCalibrationBaselineRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodGet, "/v1/marketops/backtest-calibration-baselines?tenant_id=tenant-1&dataset=equity_eod_prices&detector_id=marketops.dsm.taxonomy_v1&status=active&limit=10", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if repo.lastBacktestBaselineFilter.TenantID != "tenant-1" || repo.lastBacktestBaselineFilter.Status != "active" || repo.lastBacktestBaselineFilter.Limit != 10 {
		t.Fatalf("filter = %+v", repo.lastBacktestBaselineFilter)
	}
	var response map[string][]map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if len(response["calibration_baselines"]) != 1 || response["calibration_baselines"][0]["baseline_id"] != "btbase-1" {
		t.Fatalf("response = %#v", response)
	}
}

func TestGetMarketOpsBacktestCalibrationBaseline(t *testing.T) {
	repo := &fakeQueryRepository{backtestCalibrationBaselines: []storage.MarketOpsBacktestCalibrationBaselineRecord{validMarketOpsBacktestCalibrationBaselineRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodGet, "/v1/marketops/backtest-calibration-baselines/btbase-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response["calibration_baseline"]["baseline_id"] != "btbase-1" {
		t.Fatalf("response = %#v", response)
	}
}

func TestPostMarketOpsBacktestCalibrationComparisonCreatesStoredComparison(t *testing.T) {
	repo := &fakeQueryRepository{backtestCalibrationSummaries: []storage.MarketOpsBacktestCalibrationSummaryRecord{validMarketOpsBacktestCalibrationSummaryRecord()}, backtestCalibrationBaselines: []storage.MarketOpsBacktestCalibrationBaselineRecord{validMarketOpsBacktestCalibrationBaselineRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	body := `{"comparison_id":"btcmp-1","tenant_id":"tenant-1","baseline_id":"btbase-1","candidate_summary_id":"btcal-1"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/marketops/backtest-calibration-comparisons", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	comparison := response["calibration_comparison"]
	if comparison["comparison_id"] != "btcmp-1" || comparison["recommendation"] != storage.MarketOpsBacktestCalibrationRecommendationNeutral {
		t.Fatalf("comparison = %#v", comparison)
	}
	metrics := comparison["comparison_metrics"].(map[string]any)
	deltas := metrics["deltas"].(map[string]any)
	if deltas["dominant_recommendation_changed"] != false {
		t.Fatalf("metrics = %#v", metrics)
	}
}

func TestGetMarketOpsBacktestCalibrationComparisons(t *testing.T) {
	repo := &fakeQueryRepository{backtestCalibrationComparisons: []storage.MarketOpsBacktestCalibrationComparisonRecord{validMarketOpsBacktestCalibrationComparisonRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodGet, "/v1/marketops/backtest-calibration-comparisons?tenant_id=tenant-1&baseline_id=btbase-1&recommendation=neutral_candidate&limit=10", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if repo.lastBacktestComparisonFilter.TenantID != "tenant-1" || repo.lastBacktestComparisonFilter.BaselineID != "btbase-1" || repo.lastBacktestComparisonFilter.Limit != 10 {
		t.Fatalf("filter = %+v", repo.lastBacktestComparisonFilter)
	}
	var response map[string][]map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if len(response["calibration_comparisons"]) != 1 || response["calibration_comparisons"][0]["comparison_id"] != "btcmp-1" {
		t.Fatalf("response = %#v", response)
	}
}

func TestGetMarketOpsBacktestCalibrationComparison(t *testing.T) {
	repo := &fakeQueryRepository{backtestCalibrationComparisons: []storage.MarketOpsBacktestCalibrationComparisonRecord{validMarketOpsBacktestCalibrationComparisonRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodGet, "/v1/marketops/backtest-calibration-comparisons/btcmp-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response["calibration_comparison"]["comparison_id"] != "btcmp-1" {
		t.Fatalf("response = %#v", response)
	}
}

func validMarketOpsBacktestCalibrationBaselineRecord() storage.MarketOpsBacktestCalibrationBaselineRecord {
	now := time.Date(2026, 7, 12, 6, 30, 0, 0, time.UTC)
	return storage.MarketOpsBacktestCalibrationBaselineRecord{BaselineID: "btbase-1", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", Name: "Taxonomy baseline", SummaryID: "btcal-1", DetectorID: "marketops.dsm.taxonomy_v1", Dataset: "equity_eod_prices", ScopeJSON: []byte(`{"symbols":["AAPL"]}`), Status: storage.MarketOpsBacktestCalibrationBaselineStatusActive, CreatedBy: "operator-test", CreatedAt: now, UpdatedAt: now}
}

func validMarketOpsBacktestCalibrationComparisonRecord() storage.MarketOpsBacktestCalibrationComparisonRecord {
	now := time.Date(2026, 7, 12, 6, 35, 0, 0, time.UTC)
	return storage.MarketOpsBacktestCalibrationComparisonRecord{ComparisonID: "btcmp-1", TenantID: "tenant-1", BaselineID: "btbase-1", BaselineSummaryID: "btcal-1", CandidateSummaryID: "btcal-1", DetectorID: "marketops.dsm.taxonomy_v1", Dataset: "equity_eod_prices", ComparisonMetricsJSON: []byte(`{"deltas":{"dominant_recommendation_changed":false}}`), Recommendation: storage.MarketOpsBacktestCalibrationRecommendationNeutral, RecommendationReason: "candidate is within baseline tolerance bands", CreatedBy: "operator-test", CreatedAt: now}
}

func validMarketOpsBacktestCalibrationSummaryRecord() storage.MarketOpsBacktestCalibrationSummaryRecord {
	now := time.Date(2026, 7, 12, 5, 55, 0, 0, time.UTC)
	return storage.MarketOpsBacktestCalibrationSummaryRecord{SummaryID: "btcal-1", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SourceID: "src-massive", Dataset: "equity_eod_prices", DetectorID: "marketops.dsm.taxonomy_v1", StatusFilter: "succeeded", RequestedBy: "operator-test", RunIDs: []string{"bt-1"}, RunCount: 1, SucceededCount: 1, Scanned: 2, Signals: 1, Artifacts: 1, GraphProposals: 5, PolicyResults: 5, SignalYield: 0.5, PolicyResultsPerSignal: 5, RecommendationCounts: []byte(`{"auto_accept_candidate":5}`), RecommendationShares: []byte(`{"auto_accept_candidate":1}`), DominantRecommendation: []byte(`{"key":"auto_accept_candidate","count":5,"share":1}`), FiltersJSON: []byte(`{"tenant_id":"tenant-1"}`), ParametersJSON: []byte(`{"summary_version":"marketops.backtest.calibration_summary.v1"}`), CreatedAt: now}
}

func TestPostSyncraticContextWindowCreatesFromPersistedEvidence(t *testing.T) {
	now := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	repo := &fakeQueryRepository{
		signals: []storage.SignalLedgerRecord{{SignalID: "sig-aapl-1", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SignalType: "marketops.dsm.volatility_expansion", DetectorID: "marketops.dsm.taxonomy_v1", SignalTime: now.Add(-time.Hour), EventIDs: []string{"evt-aapl-1"}, EntitiesJSON: []byte(`[{"symbol":"AAPL"}]`)}},
		alerts:  []storage.AlertLedgerRecord{{AlertID: "alert-aapl-1", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", AlertType: "marketops.dsm.volatility_expansion", DetectorID: "marketops.dsm.taxonomy_v1", Severity: "medium", LastObservedAt: now.Add(-30 * time.Minute), EventIDs: []string{"evt-aapl-2"}, EntitiesJSON: []byte(`[{"symbol":"AAPL"}]`)}},
	}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	body := `{"tenant_id":"tenant-1","subject_symbol":"AAPL","context_strategy":"symbol_signal_cluster_5d","window_start":"2026-07-12T00:00:00Z","window_end":"2026-07-14T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/syncratic/context-windows", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	ctx := response["context_window"]
	if ctx["subject_symbol"] != "AAPL" || ctx["evidence_digest"] == "" || ctx["idempotency_key"] == "" {
		t.Fatalf("context window = %#v", ctx)
	}
	if len(repo.syncraticContextWindows) != 1 || len(repo.syncraticContextWindows[0].SignalIDs) != 1 || len(repo.syncraticContextWindows[0].AlertIDs) != 1 {
		t.Fatalf("stored context = %+v", repo.syncraticContextWindows)
	}
}

func TestPostSyncraticInsightCreatesFromContextWindow(t *testing.T) {
	ctx := storage.SyncraticContextWindowRecord{ContextWindowID: "synctx-test", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SubjectType: "ticker", SubjectID: "AAPL", SubjectSymbol: "AAPL", ContextStrategy: "symbol_signal_cluster_5d", ContextBuilderVersion: "syncratic.context_builder.v1", SignalIDs: []string{"sig-aapl-1", "sig-aapl-2"}, AlertIDs: []string{"alert-aapl-1"}, EventIDs: []string{"evt-aapl-1"}, SummaryMetricsJSON: []byte(`{"signal_count":2,"alert_count":1}`), EvidenceDigest: "digest", IdempotencyKey: "idem", Status: "active", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	repo := &fakeQueryRepository{syncraticContextWindows: []storage.SyncraticContextWindowRecord{ctx}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodPost, "/v1/syncratic/insights", strings.NewReader(`{"tenant_id":"tenant-1","context_window_id":"synctx-test","insight_type":"marketops.syncratic.recurring_volatility_cluster"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if len(repo.syncraticInsights) != 1 || repo.syncraticInsights[0].ContextWindowID != "synctx-test" || len(repo.syncraticInsights[0].SupportingSignalIDs) != 2 {
		t.Fatalf("stored insights = %+v", repo.syncraticInsights)
	}
}

func TestListSyncraticInsightsIncludesReadTimeCurrentness(t *testing.T) {
	olderTime := time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC)
	newerTime := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	olderContext := storage.SyncraticContextWindowRecord{ContextWindowID: "synctx-old", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SubjectType: "ticker", SubjectID: "AAPL", SubjectSymbol: "AAPL", WindowStart: olderTime.Add(-48 * time.Hour), WindowEnd: olderTime, ContextStrategy: "symbol_signal_cluster_5d", ContextBuilderVersion: defaultSyncraticBuilderVersion, EvidenceDigest: "digest-old", Status: "active", UpdatedAt: olderTime}
	newerContext := storage.SyncraticContextWindowRecord{ContextWindowID: "synctx-new", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SubjectType: "ticker", SubjectID: "AAPL", SubjectSymbol: "AAPL", WindowStart: newerTime.Add(-48 * time.Hour), WindowEnd: newerTime, ContextStrategy: "symbol_signal_cluster_5d", ContextBuilderVersion: defaultSyncraticBuilderVersion, EvidenceDigest: "digest-new", Status: "active", UpdatedAt: newerTime}
	olderInsight := buildSyncraticInsight(olderContext, defaultSyncraticInsightType, defaultSyncraticBuilderVersion)
	olderInsight.SyncraticInsightID = "synins-old"
	olderInsight.MetricsJSON = []byte(`{"signal_count":2,"syncratic_ask":{"ask_status":"completed"}}`)
	olderInsight.UpdatedAt = olderTime
	newerInsight := buildSyncraticInsight(newerContext, defaultSyncraticInsightType, defaultSyncraticBuilderVersion)
	newerInsight.SyncraticInsightID = "synins-new"
	newerInsight.MetricsJSON = []byte(`{"signal_count":3}`)
	newerInsight.UpdatedAt = newerTime
	repo := &fakeQueryRepository{syncraticContextWindows: []storage.SyncraticContextWindowRecord{olderContext, newerContext}, syncraticInsights: []storage.SyncraticInsightRecord{olderInsight, newerInsight}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodGet, "/v1/syncratic/insights?tenant_id=tenant-1&subject_symbol=AAPL&limit=10", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response map[string][]map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	byID := map[string]map[string]any{}
	for _, item := range response["syncratic_insights"] {
		byID[item["syncratic_insight_id"].(string)] = item
	}
	oldCurrentness := byID["synins-old"]["currentness"].(map[string]any)
	newCurrentness := byID["synins-new"]["currentness"].(map[string]any)
	if oldCurrentness["is_current"].(bool) || oldCurrentness["superseded_by_syncratic_insight_id"] != "synins-new" || oldCurrentness["reason"] != "newer_context_window" {
		t.Fatalf("old currentness = %#v", oldCurrentness)
	}
	if !newCurrentness["is_current"].(bool) || newCurrentness["reason"] != "latest_window_end" {
		t.Fatalf("new currentness = %#v", newCurrentness)
	}
	metrics := byID["synins-old"]["metrics"].(map[string]any)
	if _, ok := metrics["syncratic_ask"]; !ok {
		t.Fatalf("older ask metadata missing from metrics = %#v", metrics)
	}
}

func TestGetSyncraticInsightIncludesReadTimeCurrentness(t *testing.T) {
	olderTime := time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC)
	newerTime := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	olderContext := storage.SyncraticContextWindowRecord{ContextWindowID: "synctx-old", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SubjectType: "ticker", SubjectID: "AAPL", SubjectSymbol: "AAPL", WindowStart: olderTime.Add(-48 * time.Hour), WindowEnd: olderTime, ContextStrategy: "symbol_signal_cluster_5d", ContextBuilderVersion: defaultSyncraticBuilderVersion, EvidenceDigest: "digest-old", Status: "active", UpdatedAt: olderTime}
	newerContext := storage.SyncraticContextWindowRecord{ContextWindowID: "synctx-new", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SubjectType: "ticker", SubjectID: "AAPL", SubjectSymbol: "AAPL", WindowStart: newerTime.Add(-48 * time.Hour), WindowEnd: newerTime, ContextStrategy: "symbol_signal_cluster_5d", ContextBuilderVersion: defaultSyncraticBuilderVersion, EvidenceDigest: "digest-new", Status: "active", UpdatedAt: newerTime}
	olderInsight := buildSyncraticInsight(olderContext, defaultSyncraticInsightType, defaultSyncraticBuilderVersion)
	olderInsight.SyncraticInsightID = "synins-old"
	olderInsight.UpdatedAt = olderTime
	newerInsight := buildSyncraticInsight(newerContext, defaultSyncraticInsightType, defaultSyncraticBuilderVersion)
	newerInsight.SyncraticInsightID = "synins-new"
	newerInsight.UpdatedAt = newerTime
	repo := &fakeQueryRepository{syncraticContextWindows: []storage.SyncraticContextWindowRecord{olderContext, newerContext}, syncraticInsights: []storage.SyncraticInsightRecord{olderInsight, newerInsight}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodGet, "/v1/syncratic/insights/synins-old", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	currentness := response["syncratic_insight"]["currentness"].(map[string]any)
	if currentness["is_current"].(bool) || currentness["superseded_by_context_window_id"] != "synctx-new" || currentness["superseded_by_syncratic_insight_id"] != "synins-new" {
		t.Fatalf("currentness = %#v", currentness)
	}
}

func TestPostSyncraticMaterializeScansTop50AndSkipsQuietAssets(t *testing.T) {
	now := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	repo := &fakeQueryRepository{
		marketOpsAssets: []storage.MarketOpsAssetRecord{{TenantID: "tenant-1", UniverseGroup: "top50_megacap", Ticker: "AAPL", IsActive: true}, {TenantID: "tenant-1", UniverseGroup: "top50_megacap", Ticker: "MSFT", IsActive: true}},
		signals: []storage.SignalLedgerRecord{
			{SignalID: "sig-aapl-1", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SignalType: "marketops.dsm.volatility_expansion", DetectorID: "marketops.dsm.taxonomy_v1", SignalTime: now.Add(-time.Hour), EventIDs: []string{"evt-aapl-1"}, EntitiesJSON: []byte(`[{"symbol":"AAPL"}]`)},
			{SignalID: "sig-aapl-2", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SignalType: "marketops.dsm.price_quality_exception", DetectorID: "marketops.dsm.taxonomy_v1", SignalTime: now.Add(-2 * time.Hour), EventIDs: []string{"evt-aapl-2"}, EntitiesJSON: []byte(`[{"symbol":"AAPL"}]`)},
		},
	}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	body := `{"tenant_id":"tenant-1","universe_group":"top50_megacap","context_strategy":"symbol_signal_cluster_5d","window_start":"2026-07-12T00:00:00Z","window_end":"2026-07-14T00:00:00Z","min_evidence_count":2,"max_assets":50,"max_context_windows":10,"max_insights":10}`
	req := httptest.NewRequest(http.MethodPost, "/v1/syncratic/materialize", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	materialization := response["materialization"]
	if materialization["scanned_assets"].(float64) != 2 || materialization["materialized_context_windows"].(float64) != 1 || materialization["skipped_below_threshold"].(float64) != 1 {
		t.Fatalf("materialization = %#v", materialization)
	}
	if len(repo.syncraticContextWindows) != 1 || repo.syncraticContextWindows[0].SubjectSymbol != "AAPL" || len(repo.syncraticInsights) != 1 {
		t.Fatalf("contexts=%+v insights=%+v", repo.syncraticContextWindows, repo.syncraticInsights)
	}
}

func TestPostSyncraticMaterializeRejectsSubjectMismatchedEvidence(t *testing.T) {
	now := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	repo := &fakeQueryRepository{
		marketOpsAssets: []storage.MarketOpsAssetRecord{{TenantID: "tenant-1", UniverseGroup: "top50_megacap", Ticker: "MS", IsActive: true}},
		signals: []storage.SignalLedgerRecord{
			{SignalID: "sig-ms-tainted-1", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SignalType: "marketops.dsm.volatility_expansion", DetectorID: "marketops.dsm.taxonomy_v1", SignalTime: now.Add(-time.Hour), EventIDs: []string{"evt-ms-1"}, EntitiesJSON: []byte(`[{"symbol":"MS"}]`), EvidenceJSON: []byte(`[{"summary":"volatility expansion threshold crossed for AAPL"}]`)},
			{SignalID: "sig-ms-tainted-2", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SignalType: "marketops.dsm.accumulation", DetectorID: "marketops.dsm.taxonomy_v1", SignalTime: now.Add(-2 * time.Hour), EventIDs: []string{"evt-ms-2"}, EntitiesJSON: []byte(`[{"symbol":"MS"}]`), EvidenceJSON: []byte(`[{"summary":"accumulation pressure detected for SPY"}]`)},
		},
	}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	body := `{"tenant_id":"tenant-1","universe_group":"top50_megacap","context_strategy":"symbol_signal_cluster_5d","window_start":"2026-07-12T00:00:00Z","window_end":"2026-07-14T00:00:00Z","min_evidence_count":2}`
	req := httptest.NewRequest(http.MethodPost, "/v1/syncratic/materialize", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	materialization := response["materialization"]
	if materialization["materialized_context_windows"].(float64) != 0 || materialization["skipped_below_threshold"].(float64) != 1 {
		t.Fatalf("materialization = %#v", materialization)
	}
	if len(repo.syncraticContextWindows) != 0 || len(repo.syncraticInsights) != 0 {
		t.Fatalf("contexts=%+v insights=%+v", repo.syncraticContextWindows, repo.syncraticInsights)
	}
}

func TestPostSyncraticMaterializeDryRunReturnsDecisionPreview(t *testing.T) {
	now := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	repo := &fakeQueryRepository{
		marketOpsAssets: []storage.MarketOpsAssetRecord{{TenantID: "tenant-1", UniverseGroup: "top50_megacap", Ticker: "AAPL", IsActive: true}, {TenantID: "tenant-1", UniverseGroup: "top50_megacap", Ticker: "MSFT", IsActive: true}},
		signals: []storage.SignalLedgerRecord{
			{SignalID: "sig-aapl-1", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SignalType: "marketops.dsm.volatility_expansion", DetectorID: "marketops.dsm.taxonomy_v1", SignalTime: now.Add(-time.Hour), EventIDs: []string{"evt-aapl-1"}, EntitiesJSON: []byte(`[{"symbol":"AAPL"}]`)},
			{SignalID: "sig-aapl-2", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SignalType: "marketops.dsm.price_quality_exception", DetectorID: "marketops.dsm.taxonomy_v1", SignalTime: now.Add(-2 * time.Hour), EventIDs: []string{"evt-aapl-2"}, EntitiesJSON: []byte(`[{"symbol":"AAPL"}]`)},
		},
	}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	body := `{"tenant_id":"tenant-1","universe_group":"top50_megacap","context_strategy":"symbol_signal_cluster_5d","window_start":"2026-07-12T00:00:00Z","window_end":"2026-07-14T00:00:00Z","min_evidence_count":2,"dry_run":true}`
	req := httptest.NewRequest(http.MethodPost, "/v1/syncratic/materialize", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	materialization := response["materialization"]
	if materialization["dry_run"] != true || materialization["candidate_windows"].(float64) != 1 || materialization["materialized_context_windows"].(float64) != 0 || materialization["skipped_below_threshold"].(float64) != 1 {
		t.Fatalf("materialization = %#v", materialization)
	}
	decisions := materialization["decisions"].([]any)
	if len(decisions) != 2 {
		t.Fatalf("decisions = %#v", decisions)
	}
	first := decisions[0].(map[string]any)
	second := decisions[1].(map[string]any)
	if first["subject_symbol"] != "AAPL" || first["action"] != "would_materialize" || first["reason"] != "eligible" || second["action"] != "skipped" || second["reason"] != "below_threshold" {
		t.Fatalf("decisions = %#v", decisions)
	}
	if len(repo.syncraticContextWindows) != 0 || len(repo.syncraticInsights) != 0 {
		t.Fatalf("dry run persisted contexts=%+v insights=%+v", repo.syncraticContextWindows, repo.syncraticInsights)
	}
}

func TestPostSyncraticMaterializeDryRunAppliesMaterializationBudget(t *testing.T) {
	now := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	repo := &fakeQueryRepository{
		marketOpsAssets: []storage.MarketOpsAssetRecord{{TenantID: "tenant-1", UniverseGroup: "top50_megacap", Ticker: "AAPL", IsActive: true}, {TenantID: "tenant-1", UniverseGroup: "top50_megacap", Ticker: "MSFT", IsActive: true}},
		signals: []storage.SignalLedgerRecord{
			{SignalID: "sig-aapl-1", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SignalType: "marketops.dsm.volatility_expansion", DetectorID: "marketops.dsm.taxonomy_v1", SignalTime: now.Add(-time.Hour), EventIDs: []string{"evt-aapl-1"}, EntitiesJSON: []byte(`[{"symbol":"AAPL"}]`)},
			{SignalID: "sig-aapl-2", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SignalType: "marketops.dsm.price_quality_exception", DetectorID: "marketops.dsm.taxonomy_v1", SignalTime: now.Add(-2 * time.Hour), EventIDs: []string{"evt-aapl-2"}, EntitiesJSON: []byte(`[{"symbol":"AAPL"}]`)},
			{SignalID: "sig-msft-1", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SignalType: "marketops.dsm.volatility_expansion", DetectorID: "marketops.dsm.taxonomy_v1", SignalTime: now.Add(-time.Hour), EventIDs: []string{"evt-msft-1"}, EntitiesJSON: []byte(`[{"symbol":"MSFT"}]`)},
			{SignalID: "sig-msft-2", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SignalType: "marketops.dsm.price_quality_exception", DetectorID: "marketops.dsm.taxonomy_v1", SignalTime: now.Add(-2 * time.Hour), EventIDs: []string{"evt-msft-2"}, EntitiesJSON: []byte(`[{"symbol":"MSFT"}]`)},
		},
	}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	body := `{"tenant_id":"tenant-1","universe_group":"top50_megacap","context_strategy":"symbol_signal_cluster_5d","window_start":"2026-07-12T00:00:00Z","window_end":"2026-07-14T00:00:00Z","min_evidence_count":2,"max_context_windows":1,"max_insights":1,"dry_run":true}`
	req := httptest.NewRequest(http.MethodPost, "/v1/syncratic/materialize", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response map[string]map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	materialization := response["materialization"]
	if materialization["candidate_windows"].(float64) != 2 || materialization["skipped_budget_cap"].(float64) != 1 {
		t.Fatalf("materialization = %#v", materialization)
	}
	decisions := materialization["decisions"].([]any)
	if decisions[0].(map[string]any)["action"] != "would_materialize" || decisions[1].(map[string]any)["reason"] != "materialization_budget_cap" {
		t.Fatalf("decisions = %#v", decisions)
	}
	if len(repo.syncraticContextWindows) != 0 || len(repo.syncraticInsights) != 0 {
		t.Fatalf("dry run persisted contexts=%+v insights=%+v", repo.syncraticContextWindows, repo.syncraticInsights)
	}
}

func TestPostSyncraticMaterializeSkipsUnchangedEvidenceDigest(t *testing.T) {
	now := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	repo := &fakeQueryRepository{
		marketOpsAssets: []storage.MarketOpsAssetRecord{{TenantID: "tenant-1", UniverseGroup: "top50_megacap", Ticker: "AAPL", IsActive: true}},
		signals: []storage.SignalLedgerRecord{
			{SignalID: "sig-aapl-1", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SignalType: "marketops.dsm.volatility_expansion", DetectorID: "marketops.dsm.taxonomy_v1", SignalTime: now.Add(-time.Hour), EventIDs: []string{"evt-aapl-1"}, EntitiesJSON: []byte(`[{"symbol":"AAPL"}]`)},
			{SignalID: "sig-aapl-2", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SignalType: "marketops.dsm.price_quality_exception", DetectorID: "marketops.dsm.taxonomy_v1", SignalTime: now.Add(-2 * time.Hour), EventIDs: []string{"evt-aapl-2"}, EntitiesJSON: []byte(`[{"symbol":"AAPL"}]`)},
		},
	}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	body := `{"tenant_id":"tenant-1","universe_group":"top50_megacap","context_strategy":"symbol_signal_cluster_5d","window_start":"2026-07-12T00:00:00Z","window_end":"2026-07-14T00:00:00Z","min_evidence_count":2}`
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/v1/syncratic/materialize", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("run %d status = %d body=%s", i, rec.Code, rec.Body.String())
		}
		if i == 1 {
			var response map[string]map[string]any
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatal(err)
			}
			if response["materialization"]["skipped_unchanged"].(float64) != 1 || response["materialization"]["materialized_context_windows"].(float64) != 0 {
				t.Fatalf("second materialization = %#v", response["materialization"])
			}
		}
	}
	if len(repo.syncraticContextWindows) != 1 || len(repo.syncraticInsights) != 1 {
		t.Fatalf("contexts=%d insights=%d", len(repo.syncraticContextWindows), len(repo.syncraticInsights))
	}
}

func TestBuildSyncraticAskPromptIsStableAndCapped(t *testing.T) {
	cw := storage.SyncraticContextWindowRecord{ContextWindowID: "synctx-test", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SubjectType: "ticker", SubjectID: "AAPL", SubjectSymbol: "AAPL", WindowStart: time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC), WindowEnd: time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC), ContextStrategy: "symbol_signal_cluster_5d", ContextBuilderVersion: defaultSyncraticBuilderVersion, SignalTypes: []string{"b", "a"}, DetectorIDs: []string{"det-1"}, SignalIDs: []string{"sig-3", "sig-1", "sig-2"}, AlertIDs: []string{"alert-1"}, SummaryMetricsJSON: []byte(`{"signal_count":3,"alert_count":1}`), EvidenceDigest: "digest"}
	prompt1, meta1, err := buildSyncraticAskPrompt(cw, syncraticAskRequest{MaxPromptBytes: 12000}, []syncraticAskSignalDetail{{SignalID: "sig-1", SignalType: "marketops.dsm.volatility_expansion", Severity: "high", Confidence: 0.91, SupportingMetrics: json.RawMessage(`{"daily_return_pct":6.9}`)}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	prompt2, meta2, err := buildSyncraticAskPrompt(cw, syncraticAskRequest{MaxPromptBytes: 12000}, []syncraticAskSignalDetail{{SignalID: "sig-1", SignalType: "marketops.dsm.volatility_expansion", Severity: "high", Confidence: 0.91, SupportingMetrics: json.RawMessage(`{"daily_return_pct":6.9}`)}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if prompt1 != prompt2 || meta1.PromptDigest != meta2.PromptDigest || meta1.PromptBuilderVersion != defaultSyncraticAskPromptVersion {
		t.Fatalf("prompt not stable meta1=%+v meta2=%+v", meta1, meta2)
	}
	if !strings.Contains(prompt1, "operator-useful interpretation") || !strings.Contains(prompt1, "top_drivers") || !strings.Contains(prompt1, "daily_return_pct") || !strings.Contains(prompt1, "synctx-test") || !strings.Contains(prompt1, "digest") {
		t.Fatalf("prompt missing required context: %s", prompt1)
	}
	if _, _, err := buildSyncraticAskPrompt(cw, syncraticAskRequest{MaxPromptBytes: 999}, nil, nil); err == nil {
		t.Fatal("expected max_prompt_bytes validation error")
	}
}

func TestPostSyncraticContextWindowAskPersistsGeneratedExplanation(t *testing.T) {
	cw := storage.SyncraticContextWindowRecord{ContextWindowID: "synctx-test", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SubjectType: "ticker", SubjectID: "AAPL", SubjectSymbol: "AAPL", WindowStart: time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC), WindowEnd: time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC), ContextStrategy: "symbol_signal_cluster_5d", ContextBuilderVersion: defaultSyncraticBuilderVersion, SignalIDs: []string{"sig-aapl-1", "sig-aapl-2"}, AlertIDs: []string{"alert-aapl-1"}, EventIDs: []string{"evt-aapl-1"}, SummaryMetricsJSON: []byte(`{"signal_count":2,"alert_count":1}`), EvidenceDigest: "digest", IdempotencyKey: "idem", Status: "active", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	insight := buildSyncraticInsight(cw, defaultSyncraticInsightType, defaultSyncraticBuilderVersion)
	repo := &fakeQueryRepository{syncraticContextWindows: []storage.SyncraticContextWindowRecord{cw}, syncraticInsights: []storage.SyncraticInsightRecord{insight}, signals: []storage.SignalLedgerRecord{{SignalID: "sig-aapl-1", SignalType: "marketops.dsm.volatility_expansion", DetectorID: "marketops.dsm.taxonomy_v1", Severity: "high", Confidence: 0.91, EventIDs: []string{"evt-aapl-1"}, EntitiesJSON: []byte(`[{"symbol":"AAPL"}]`), SupportingMetrics: []byte(`{"daily_return_pct":6.9,"intraday_range_pct":13}`), EvidenceJSON: []byte(`[{"summary":"volatility expansion threshold crossed"}]`)}}}
	ask := &fakeSyncraticAskClient{resp: userapi.AskResponse{QueryID: "ask-1", Answer: "Generated Syncratic Ask explanation for AAPL.", Confidence: userapi.NumericFloat(0.91), EvidenceCount: 3, Raw: []byte(`{"title":"Ask title","summary":"Ask summary","action":"review"}`)}}
	router := NewRouter(RouterConfig{QueryRepository: repo, SyncraticAskClient: ask})
	req := httptest.NewRequest(http.MethodPost, "/v1/syncratic/context-windows/synctx-test/ask", strings.NewReader(`{"tenant_id":"tenant-1","max_prompt_bytes":12000}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if ask.calls != 1 || ask.req.Scope != defaultSyncraticAskScope || ask.req.K != 1 || ask.req.ThreadMode != "off" || ask.req.IncludeRefs == nil || *ask.req.IncludeRefs || len(ask.req.Filters) != 0 {
		t.Fatalf("ask call = %+v calls=%d", ask.req, ask.calls)
	}
	if ask.req.DirectReasoning == nil || !*ask.req.DirectReasoning || ask.req.GraphEnabled == nil || *ask.req.GraphEnabled || ask.req.KEEEnabled == nil || *ask.req.KEEEnabled {
		t.Fatalf("ask reasoning flags = %+v", ask.req)
	}
	if ask.req.ExternalContext == nil || len(ask.req.ExternalContext.Items) != 1 || ask.req.ExternalContext.Items[0].SourceID != "synctx-test" || !strings.Contains(ask.req.ExternalContext.Items[0].Text, "synctx-test") {
		t.Fatalf("ask external context = %+v", ask.req.ExternalContext)
	}
	stored := repo.syncraticInsights[0]
	if stored.Explanation != "Generated Syncratic Ask explanation for AAPL." || stored.Title != "Ask title" || stored.Summary != "Ask summary" || stored.Confidence != 0.91 {
		t.Fatalf("stored insight = %+v", stored)
	}
	var metrics map[string]any
	if err := json.Unmarshal(stored.MetricsJSON, &metrics); err != nil {
		t.Fatal(err)
	}
	askMeta, ok := metrics["syncratic_ask"].(map[string]any)
	if !ok || askMeta["ask_query_id"] != "ask-1" || askMeta["context_evidence_digest"] != "digest" {
		t.Fatalf("ask metrics = %#v", metrics["syncratic_ask"])
	}
}

func TestPostSyncraticContextWindowAskSkipsUnchangedEvidence(t *testing.T) {
	cw := storage.SyncraticContextWindowRecord{ContextWindowID: "synctx-test", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SubjectType: "ticker", SubjectID: "AAPL", SubjectSymbol: "AAPL", WindowStart: time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC), WindowEnd: time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC), ContextStrategy: "symbol_signal_cluster_5d", ContextBuilderVersion: defaultSyncraticBuilderVersion, SignalIDs: []string{"sig-aapl-1"}, AlertIDs: []string{"alert-aapl-1"}, SummaryMetricsJSON: []byte(`{"signal_count":1,"alert_count":1}`), EvidenceDigest: "digest", IdempotencyKey: "idem", Status: "active"}
	repo := &fakeQueryRepository{syncraticContextWindows: []storage.SyncraticContextWindowRecord{cw}}
	details, missing, err := syncraticAskSignalDetails(context.Background(), repo, cw, 12)
	if err != nil {
		t.Fatal(err)
	}
	_, meta, err := buildSyncraticAskPrompt(cw, syncraticAskRequest{MaxPromptBytes: 12000}, details, missing)
	if err != nil {
		t.Fatal(err)
	}
	insight := buildSyncraticInsight(cw, defaultSyncraticInsightType, defaultSyncraticBuilderVersion)
	insight.MetricsJSON = mustJSON(map[string]any{"syncratic_ask": map[string]any{"ask_status": "completed", "prompt_digest": meta.PromptDigest, "context_evidence_digest": meta.ContextEvidenceDigest}})
	repo.syncraticInsights = []storage.SyncraticInsightRecord{insight}
	ask := &fakeSyncraticAskClient{}
	router := NewRouter(RouterConfig{QueryRepository: repo, SyncraticAskClient: ask})
	req := httptest.NewRequest(http.MethodPost, "/v1/syncratic/context-windows/synctx-test/ask", strings.NewReader(`{"tenant_id":"tenant-1","max_prompt_bytes":12000}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if ask.calls != 0 {
		t.Fatalf("expected ask skip, calls=%d", ask.calls)
	}
	if !strings.Contains(rec.Body.String(), "unchanged_prompt_and_evidence") {
		t.Fatalf("response = %s", rec.Body.String())
	}
}

func TestPostSyncraticContextWindowAskSanitizesUpstreamError(t *testing.T) {
	cw := storage.SyncraticContextWindowRecord{ContextWindowID: "synctx-test", TenantID: "tenant-1", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SubjectType: "ticker", SubjectID: "AAPL", SubjectSymbol: "AAPL", WindowStart: time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC), WindowEnd: time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC), ContextStrategy: "symbol_signal_cluster_5d", ContextBuilderVersion: defaultSyncraticBuilderVersion, SignalIDs: []string{"sig-aapl-1"}, AlertIDs: []string{"alert-aapl-1"}, SummaryMetricsJSON: []byte(`{"signal_count":1,"alert_count":1}`), EvidenceDigest: "digest", IdempotencyKey: "idem", Status: "active"}
	repo := &fakeQueryRepository{syncraticContextWindows: []storage.SyncraticContextWindowRecord{cw}, syncraticInsights: []storage.SyncraticInsightRecord{buildSyncraticInsight(cw, defaultSyncraticInsightType, defaultSyncraticBuilderVersion)}}
	ask := &fakeSyncraticAskClient{err: errors.New("secret upstream token failure details")}
	router := NewRouter(RouterConfig{QueryRepository: repo, SyncraticAskClient: ask})
	req := httptest.NewRequest(http.MethodPost, "/v1/syncratic/context-windows/synctx-test/ask", strings.NewReader(`{"tenant_id":"tenant-1"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "secret upstream") {
		t.Fatalf("upstream details leaked: %s", rec.Body.String())
	}
}

func TestAlgorithmDefinitionCreateListAndGet(t *testing.T) {
	repo := &fakeQueryRepository{}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	body := `{"tenant_id":"tenant-local","algorithm_id":"signalops.algorithms.zscore_anomaly_v1","name":"Z-Score Anomaly","algorithm_type":"anomaly_detection","runtime_type":"python_plugin","input_features":["daily_return_pct"],"input_event_types":["normalized_event"],"output_schema":{"type":"object"},"config_schema":{"type":"object"},"default_config":{"z_threshold":3},"version":"v1","status":"draft"}`

	req := httptest.NewRequest(http.MethodPost, "/v1/algorithms/definitions", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/algorithms/definitions?tenant_id=tenant-local&algorithm_type=anomaly_detection&runtime_type=python_plugin&status=draft&limit=5", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", rec.Code, rec.Body.String())
	}
	if repo.lastAlgorithmDefinitionFilter.TenantID != "tenant-local" || repo.lastAlgorithmDefinitionFilter.Limit != 5 {
		t.Fatalf("filter = %+v", repo.lastAlgorithmDefinitionFilter)
	}
	var list map[string][]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if got := list["algorithm_definitions"][0]["algorithm_id"]; got != "signalops.algorithms.zscore_anomaly_v1" {
		t.Fatalf("algorithm_id = %v", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/algorithms/definitions/signalops.algorithms.zscore_anomaly_v1?tenant_id=tenant-local", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAlgorithmExecutionRequestCreateListAndGet(t *testing.T) {
	repo := &fakeQueryRepository{}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	body := `{"tenant_id":"tenant-local","execution_request_id":"algexec-1","algorithm_id":"signalops.algorithms.zscore_anomaly_v1","algorithm_version":"v1","event_ids":["evt-1"],"feature_refs":["feature:daily_return_pct"],"entity_refs":["ticker:AAPL"],"window_ref":"2026-07-09/2026-07-14","config":{"z_threshold":3},"correlation_id":"corr-1"}`

	req := httptest.NewRequest(http.MethodPost, "/v1/algorithms/execution-requests", strings.NewReader(body))
	req.Header.Set("X-SignalOps-Actor", "analyst-1")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", rec.Code, rec.Body.String())
	}
	if repo.algorithmExecutionRequests[0].Status != storage.AlgorithmExecutionStatusQueued || repo.algorithmExecutionRequests[0].RequestedBy != "analyst-1" {
		t.Fatalf("execution request = %+v", repo.algorithmExecutionRequests[0])
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/algorithms/execution-requests?tenant_id=tenant-local&algorithm_id=signalops.algorithms.zscore_anomaly_v1&status=queued&correlation_id=corr-1&limit=10", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", rec.Code, rec.Body.String())
	}
	if repo.lastAlgorithmExecutionFilter.AlgorithmID != "signalops.algorithms.zscore_anomaly_v1" || repo.lastAlgorithmExecutionFilter.Status != "queued" {
		t.Fatalf("filter = %+v", repo.lastAlgorithmExecutionFilter)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/algorithms/execution-requests/algexec-1?tenant_id=tenant-local", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAlgorithmResultsListAndGet(t *testing.T) {
	now := time.Now().UTC()
	repo := &fakeQueryRepository{algorithmResults: []storage.AlgorithmResultRecord{{AlgorithmResultID: "algres-1", TenantID: "tenant-local", AlgorithmID: "signalops.algorithms.zscore_anomaly_v1", AlgorithmVersion: "v1", ExecutionRequestID: "algexec-1", ResultType: "anomaly_score", Score: 3.4, Confidence: 0.82, Severity: "medium", ResultPayloadJSON: []byte(`{"z_score":3.4}`), SourceEventIDs: []string{"evt-1"}, EvidenceRefs: []string{"evt-1"}, CorrelationID: "corr-1", CreatedAt: now}}}
	router := NewRouter(RouterConfig{QueryRepository: repo})

	req := httptest.NewRequest(http.MethodGet, "/v1/algorithms/results?tenant_id=tenant-local&algorithm_id=signalops.algorithms.zscore_anomaly_v1&execution_request_id=algexec-1&result_type=anomaly_score&severity=medium&correlation_id=corr-1&limit=10", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", rec.Code, rec.Body.String())
	}
	if repo.lastAlgorithmResultFilter.ResultType != "anomaly_score" || repo.lastAlgorithmResultFilter.Severity != "medium" {
		t.Fatalf("filter = %+v", repo.lastAlgorithmResultFilter)
	}
	var list map[string][]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if list["algorithm_results"][0]["algorithm_result_id"] != "algres-1" {
		t.Fatalf("list = %+v", list)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/algorithms/results/algres-1?tenant_id=tenant-local", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAlgorithmExecutionSummaryIncludesResultRollup(t *testing.T) {
	now := time.Now().UTC()
	repo := &fakeQueryRepository{
		algorithmExecutionRequests: []storage.AlgorithmExecutionRequestRecord{{ExecutionRequestID: "algexec-1", TenantID: "tenant-local", AlgorithmID: "signalops.algorithms.zscore_anomaly_v1", AlgorithmVersion: "v1", Status: storage.AlgorithmExecutionStatusSucceeded, CorrelationID: "corr-1", RequestedBy: "operator-test", ConfigJSON: []byte(`{"feature":"daily_return_pct"}`), ResultJSON: []byte(`{"results":3}`), CreatedAt: now, UpdatedAt: now}},
		algorithmResults: []storage.AlgorithmResultRecord{
			{AlgorithmResultID: "algres-low", TenantID: "tenant-local", AlgorithmID: "signalops.algorithms.zscore_anomaly_v1", AlgorithmVersion: "v1", ExecutionRequestID: "algexec-1", ResultType: "z_score", Score: 1.2, Confidence: 0.2, Severity: "low", ResultPayloadJSON: []byte(`{"z_score":1.2}`), CorrelationID: "corr-1", CreatedAt: now},
			{AlgorithmResultID: "algres-high", TenantID: "tenant-local", AlgorithmID: "signalops.algorithms.zscore_anomaly_v1", AlgorithmVersion: "v1", ExecutionRequestID: "algexec-1", ResultType: "z_score", Score: 3.7, Confidence: 0.9, Severity: "high", ResultPayloadJSON: []byte(`{"z_score":3.7}`), CorrelationID: "corr-1", CreatedAt: now.Add(time.Second)},
			{AlgorithmResultID: "algres-medium", TenantID: "tenant-local", AlgorithmID: "signalops.algorithms.zscore_anomaly_v1", AlgorithmVersion: "v1", ExecutionRequestID: "algexec-1", ResultType: "z_score", Score: 2.1, Confidence: 0.5, Severity: "medium", ResultPayloadJSON: []byte(`{"z_score":2.1}`), CorrelationID: "corr-1", CreatedAt: now.Add(2 * time.Second)},
		},
	}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodGet, "/v1/algorithms/execution-requests/algexec-1/summary?tenant_id=tenant-local&limit=2", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if repo.lastAlgorithmResultFilter.ExecutionRequestID != "algexec-1" || repo.lastAlgorithmResultFilter.Limit != 200 {
		t.Fatalf("filter = %+v", repo.lastAlgorithmResultFilter)
	}
	var response map[string]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	summary := response["algorithm_execution_summary"]
	if summary["result_count"].(float64) != 3 || summary["max_score"].(float64) != 3.7 || summary["max_confidence"].(float64) != 0.9 {
		t.Fatalf("summary = %#v", summary)
	}
	severityCounts := summary["severity_counts"].(map[string]any)
	if severityCounts["high"].(float64) != 1 || severityCounts["medium"].(float64) != 1 || severityCounts["low"].(float64) != 1 {
		t.Fatalf("severity_counts = %#v", severityCounts)
	}
	topResults := summary["top_results"].([]any)
	if len(topResults) != 2 || topResults[0].(map[string]any)["algorithm_result_id"] != "algres-high" || topResults[1].(map[string]any)["algorithm_result_id"] != "algres-medium" {
		t.Fatalf("top_results = %#v", topResults)
	}
}
