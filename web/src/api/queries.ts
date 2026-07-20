import { useMutation, useQuery, useQueryClient, type QueryClient } from '@tanstack/react-query';
import { api } from './client';
import type {
  RawEventFilter,
  NormalizedEventFilter,
  SignalFilter,
  AlertFilter,
  InsightFilter,
  AlertLifecycleMutationOptions,
  InsightLifecycleMutationOptions,
  ReplayJobFilter,
  ReplayJobCreateRequest,
  ReplayJob,
  MarketOpsAssetFilter,
  MarketOpsOptionsChainFilter,
  MarketOpsOpportunityFilter,
  MarketOpsHypothesisEvaluationFilter,
  MarketOpsOptionsDistributionFilter,
  MarketOpsDSMArtifactFilter,
  MarketOpsDSMGraphProposalFilter,
  MarketOpsDSMGraphProposalDecisionOptions,
  MarketOpsBacktestRunFilter,
  MarketOpsBacktestSignalFilter,
  MarketOpsBacktestGraphProposalFilter,
  MarketOpsBacktestCreateRequest,
  MarketOpsBacktestCreateResponse,
  MarketOpsBacktestCalibrationSummaryFilter,
  MarketOpsBacktestCalibrationSummaryCreateRequest,
  MarketOpsBacktestCalibrationSummaryResponse,
  MarketOpsBacktestCalibrationBaselineFilter,
  MarketOpsBacktestCalibrationBaselineCreateRequest,
  MarketOpsBacktestCalibrationBaselineResponse,
  MarketOpsBacktestCalibrationComparisonFilter,
  MarketOpsBacktestCalibrationComparisonCreateRequest,
  MarketOpsBacktestCalibrationComparisonResponse,
  MarketOpsBacktestEvaluationFilter,
  MarketOpsBacktestEvaluationCreateRequest,
  MarketOpsBacktestEvaluationResponse,
  MarketOpsBacktestPromotionCandidateFilter,
  MarketOpsBacktestPromotionCandidateCreateRequest,
  MarketOpsBacktestPromotionCandidateDecisionRequest,
  MarketOpsBacktestPromotionCandidateResponse,
  SyncraticInsightFilter,
  SyncraticContextWindowFilter,
  SyncraticMaterializeRequest,
  SyncraticMaterializationResponse,
  SyncraticAskRequest,
  SyncraticAskResponse,
  AlgorithmDefinitionFilter,
  AlgorithmExecutionRequestFilter,
  AlgorithmResultFilter,
  AlgorithmSignalProposalFilter,
  AlgorithmSignalProposalDecisionRequest,
  AlgorithmSignalProposalResponse,
  AlgorithmSignalMaterializationPreflightFilter,
  AlgorithmSignalMaterializationRequest,
  AlgorithmSignalMaterializationResponse,
  AlgorithmSignalMaterializationFilter,
} from '../types';

export const queryKeys = {
  healthz: ['healthz'] as const,
  readyz: ['readyz'] as const,
  runs: (limit: number) => ['runs', limit] as const,
  run: (runId: string) => ['run', runId] as const,
  providerUsage: (runId: string | undefined, limit: number) =>
    ['provider-usage', runId, limit] as const,
  rawEvents: (filter: RawEventFilter) => ['raw-events', filter] as const,
  rawEvent: (eventId: string) => ['raw-event', eventId] as const,
  idempotency: (tenantId: string, sourceId: string, key: string) =>
    ['idempotency', tenantId, sourceId, key] as const,
  catalogSources: (tenantId: string, limit: number) => ['catalog-sources', tenantId, limit] as const,
  catalogPipelines: (tenantId: string, limit: number) => ['catalog-pipelines', tenantId, limit] as const,
  catalogRules: (tenantId: string, limit: number) => ['catalog-rules', tenantId, limit] as const,
  normalizedEvents: (filter: NormalizedEventFilter) => ['normalized-events', filter] as const,
  normalizedEvent: (eventId: string) => ['normalized-event', eventId] as const,
  signals: (filter: SignalFilter) => ['signals', filter] as const,
  signal: (signalId: string) => ['signal', signalId] as const,
  alerts: (filter: AlertFilter) => ['alerts', filter] as const,
  alert: (alertId: string) => ['alert', alertId] as const,
  insights: (filter: InsightFilter) => ['insights', filter] as const,
  insight: (insightId: string) => ['insight', insightId] as const,
  replayJobs: (filter: ReplayJobFilter) => ['replay-jobs', filter] as const,
  replayJob: (replayJobId: string) => ['replay-job', replayJobId] as const,
  replayStatus: (tenantId: string, limit?: number) => ['replay-status', tenantId, limit] as const,
  appProfiles: ['app-profiles'] as const,
  marketOpsAssets: (filter: MarketOpsAssetFilter) => ['marketops-assets', filter] as const,
  marketOpsOptionsCoverage: (tenantId: string, symbol: string) =>
    ['marketops-options-coverage', tenantId, symbol] as const,
  marketOpsOptionsDistributions: (tenantId: string, symbol: string, filter: MarketOpsOptionsDistributionFilter) =>
    ['marketops-options-distributions', tenantId, symbol, filter] as const,
  marketOpsOptionsChain: (tenantId: string, symbol: string, filter: MarketOpsOptionsChainFilter) =>
    ['marketops-options-chain', tenantId, symbol, filter] as const,
  marketOpsOpportunities: (filter: MarketOpsOpportunityFilter) => ['marketops-opportunities', filter] as const,
  marketOpsOpportunity: (opportunityId: string, tenantId: string) =>
    ['marketops-opportunity', opportunityId, tenantId] as const,
  marketOpsHypothesisEvaluations: (filter: MarketOpsHypothesisEvaluationFilter) =>
    ['marketops-hypothesis-evaluations', filter] as const,
  marketOpsHypothesis: (key: string, version: string, tenantId: string) =>
    ['marketops-hypothesis', key, version, tenantId] as const,
  marketOpsEvidence: (evidenceId: string) => ['marketops-evidence', evidenceId] as const,
  marketOpsMarketStateLineage: (marketStateId: string) =>
    ['marketops-market-state-lineage', marketStateId] as const,
  marketOpsDSMArtifacts: (filter: MarketOpsDSMArtifactFilter) => ['marketops-dsm-artifacts', filter] as const,
  marketOpsDSMArtifact: (artifactId: string) => ['marketops-dsm-artifact', artifactId] as const,
  marketOpsDSMGraphProposals: (filter: MarketOpsDSMGraphProposalFilter) =>
    ['marketops-dsm-graph-proposals', filter] as const,
  marketOpsDSMGraphProposal: (proposalId: string) => ['marketops-dsm-graph-proposal', proposalId] as const,
  marketOpsBacktests: (filter: MarketOpsBacktestRunFilter) => ['marketops-backtests', filter] as const,
  marketOpsBacktest: (runId: string, tenantId: string) => ['marketops-backtest', runId, tenantId] as const,
  marketOpsBacktestSignals: (runId: string, filter: MarketOpsBacktestSignalFilter) =>
    ['marketops-backtest-signals', runId, filter] as const,
  marketOpsBacktestGraphProposals: (runId: string, filter: MarketOpsBacktestGraphProposalFilter) =>
    ['marketops-backtest-graph-proposals', runId, filter] as const,
  marketOpsBacktestCalibrationSummaries: (filter: MarketOpsBacktestCalibrationSummaryFilter) =>
    ['marketops-backtest-calibration-summaries', filter] as const,
  marketOpsBacktestCalibrationSummary: (summaryId: string) =>
    ['marketops-backtest-calibration-summary', summaryId] as const,
  marketOpsBacktestCalibrationBaselines: (filter: MarketOpsBacktestCalibrationBaselineFilter) =>
    ['marketops-backtest-calibration-baselines', filter] as const,
  marketOpsBacktestCalibrationBaseline: (baselineId: string) =>
    ['marketops-backtest-calibration-baseline', baselineId] as const,
  marketOpsBacktestCalibrationComparisons: (filter: MarketOpsBacktestCalibrationComparisonFilter) =>
    ['marketops-backtest-calibration-comparisons', filter] as const,
  marketOpsBacktestCalibrationComparison: (comparisonId: string) =>
    ['marketops-backtest-calibration-comparison', comparisonId] as const,
  marketOpsBacktestEvaluations: (filter: MarketOpsBacktestEvaluationFilter) =>
    ['marketops-backtest-evaluations', filter] as const,
  marketOpsBacktestEvaluation: (evaluationId: string) =>
    ['marketops-backtest-evaluation', evaluationId] as const,
  marketOpsBacktestPromotionCandidates: (filter: MarketOpsBacktestPromotionCandidateFilter) =>
    ['marketops-backtest-promotion-candidates', filter] as const,
  marketOpsBacktestPromotionCandidate: (candidateId: string) =>
    ['marketops-backtest-promotion-candidate', candidateId] as const,
  syncraticInsights: (filter: SyncraticInsightFilter) => ['syncratic-insights', filter] as const,
  syncraticInsight: (insightId: string) => ['syncratic-insight', insightId] as const,
  syncraticContextWindows: (filter: SyncraticContextWindowFilter) =>
    ['syncratic-context-windows', filter] as const,
  syncraticContextWindow: (contextWindowId: string) =>
    ['syncratic-context-window', contextWindowId] as const,
  algorithmDefinitions: (filter: AlgorithmDefinitionFilter) => ['algorithm-definitions', filter] as const,
  algorithmDefinition: (algorithmId: string, tenantId: string) =>
    ['algorithm-definition', algorithmId, tenantId] as const,
  algorithmExecutionRequests: (filter: AlgorithmExecutionRequestFilter) =>
    ['algorithm-execution-requests', filter] as const,
  algorithmExecutionRequest: (executionRequestId: string, tenantId: string) =>
    ['algorithm-execution-request', executionRequestId, tenantId] as const,
  algorithmExecutionSummary: (executionRequestId: string, tenantId: string, limit: number) =>
    ['algorithm-execution-summary', executionRequestId, tenantId, limit] as const,
  algorithmResults: (filter: AlgorithmResultFilter) => ['algorithm-results', filter] as const,
  algorithmResult: (algorithmResultId: string, tenantId: string) =>
    ['algorithm-result', algorithmResultId, tenantId] as const,
  algorithmSignalProposals: (filter: AlgorithmSignalProposalFilter) =>
    ['algorithm-signal-proposals', filter] as const,
  algorithmSignalProposal: (proposalId: string, tenantId: string) =>
    ['algorithm-signal-proposal', proposalId, tenantId] as const,
  // Summary key excludes limit on purpose — the summary aggregates the whole
  // matched slice and must not refetch when only the page limit changes.
  algorithmSignalProposalSummary: (filter: AlgorithmSignalProposalFilter) =>
    ['algorithm-signal-proposal-summary', filter] as const,
  // Preflight key includes the full filter (coupled proposal filters + limit +
  // min_reviewed_ratio + policy_version) so it refetches on any filter change.
  algorithmSignalMaterializationPreflight: (filter: AlgorithmSignalMaterializationPreflightFilter) =>
    ['algorithm-signal-materialization-preflight', filter] as const,
  // Materialization ledger rows for one proposal. Keyed by tenant + proposal id
  // (+limit) so it refetches when the selected proposal changes.
  algorithmSignalMaterializations: (filter: AlgorithmSignalMaterializationFilter) =>
    ['algorithm-signal-materializations', filter] as const,
};

export function useHealthz() {
  return useQuery({ queryKey: queryKeys.healthz, queryFn: api.healthz, refetchInterval: 15000 });
}

export function useReadyz() {
  return useQuery({ queryKey: queryKeys.readyz, queryFn: api.readyz, refetchInterval: 15000 });
}

export function useRuns(limit: number) {
  return useQuery({ queryKey: queryKeys.runs(limit), queryFn: () => api.listRuns(limit) });
}

export function useRun(runId: string | null) {
  return useQuery({
    queryKey: queryKeys.run(runId ?? ''),
    queryFn: () => api.getRun(runId!),
    enabled: !!runId,
  });
}

export function useProviderUsage(runId: string | null, limit = 50) {
  return useQuery({
    queryKey: queryKeys.providerUsage(runId ?? undefined, limit),
    queryFn: () => api.listProviderUsage(runId ?? undefined, limit),
    enabled: !!runId,
  });
}

// Unfiltered recent provider usage for the Dashboard (no run_id gating).
// Reuses the same query key prefix so DashboardStreamBridge invalidations refresh it.
export function useRecentProviderUsage(limit = 50) {
  return useQuery({
    queryKey: queryKeys.providerUsage(undefined, limit),
    queryFn: () => api.listProviderUsage(undefined, limit),
  });
}

export function useRawEvents(filter: RawEventFilter) {
  return useQuery({
    queryKey: queryKeys.rawEvents(filter),
    queryFn: () => api.listRawEvents(filter),
  });
}

export function useRawEvent(eventId: string | null) {
  return useQuery({
    queryKey: queryKeys.rawEvent(eventId ?? ''),
    queryFn: () => api.getRawEvent(eventId!),
    enabled: !!eventId,
  });
}

// Idempotency lookup is user-triggered; disable retries so a 404 surfaces immediately.
export function useIdempotency(
  tenantId: string,
  sourceId: string,
  key: string,
  enabled: boolean,
) {
  return useQuery({
    queryKey: queryKeys.idempotency(tenantId, sourceId, key),
    queryFn: () => api.getIdempotency(tenantId, sourceId, key),
    enabled: enabled && !!tenantId && !!sourceId && !!key,
    retry: false,
  });
}

export function useCatalogSources(tenantId = 'tenant-local', limit = 50) {
  return useQuery({
    queryKey: queryKeys.catalogSources(tenantId, limit),
    queryFn: () => api.listCatalogSources(tenantId, limit),
  });
}

export function useCatalogPipelines(tenantId = 'tenant-local', limit = 50) {
  return useQuery({
    queryKey: queryKeys.catalogPipelines(tenantId, limit),
    queryFn: () => api.listCatalogPipelines(tenantId, limit),
  });
}

export function useCatalogRules(tenantId = 'tenant-local', limit = 50) {
  return useQuery({
    queryKey: queryKeys.catalogRules(tenantId, limit),
    queryFn: () => api.listCatalogRules(tenantId, limit),
  });
}

export function useNormalizedEvents(filter: NormalizedEventFilter) {
  return useQuery({
    queryKey: queryKeys.normalizedEvents(filter),
    queryFn: () => api.listNormalizedEvents(filter),
  });
}

export function useNormalizedEvent(eventId: string | null) {
  return useQuery({
    queryKey: queryKeys.normalizedEvent(eventId ?? ''),
    queryFn: () => api.getNormalizedEvent(eventId!),
    enabled: !!eventId,
  });
}

export function useSignals(filter: SignalFilter) {
  return useQuery({
    queryKey: queryKeys.signals(filter),
    queryFn: () => api.listSignals(filter),
  });
}

export function useSignal(signalId: string | null) {
  return useQuery({
    queryKey: queryKeys.signal(signalId ?? ''),
    queryFn: () => api.getSignal(signalId!),
    enabled: !!signalId,
  });
}

export function useAlerts(filter: AlertFilter) {
  return useQuery({
    queryKey: queryKeys.alerts(filter),
    queryFn: () => api.listAlerts(filter),
  });
}

export function useAlert(alertId: string | null) {
  return useQuery({
    queryKey: queryKeys.alert(alertId ?? ''),
    queryFn: () => api.getAlert(alertId!),
    enabled: !!alertId,
  });
}

export function useInsights(filter: InsightFilter) {
  return useQuery({
    queryKey: queryKeys.insights(filter),
    queryFn: () => api.listInsights(filter),
  });
}

export function useInsight(insightId: string | null) {
  return useQuery({
    queryKey: queryKeys.insight(insightId ?? ''),
    queryFn: () => api.getInsight(insightId!),
    enabled: !!insightId,
  });
}

// Replay execution is asynchronous: creating a job only queues it. The list
// polls so newly queued/running jobs refresh without a second SSE stream.
export function useReplayJobs(filter: ReplayJobFilter = { tenant_id: 'tenant-local', limit: 50 }) {
  return useQuery({
    queryKey: queryKeys.replayJobs(filter),
    queryFn: () => api.listReplayJobs(filter),
    refetchInterval: 5000,
  });
}

// Detail polls only while a job is still in flight; once terminal it stops.
export function useReplayJob(replayJobId?: string) {
  return useQuery({
    queryKey: queryKeys.replayJob(replayJobId ?? ''),
    queryFn: () => api.getReplayJob(replayJobId!),
    enabled: Boolean(replayJobId),
    refetchInterval: (query) => {
      const status = query.state.data?.replay_job.status;
      return status === 'queued' || status === 'running' ? 3000 : false;
    },
  });
}

// On create, seed the detail cache with the returned (queued) job and
// invalidate the list prefix so filtered tables + Dashboard summary refetch.
export function useCreateReplayJob() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: ReplayJobCreateRequest) => api.createReplayJob(body),
    onSuccess: (data) => {
      queryClient.setQueryData(queryKeys.replayJob(data.replay_job.replay_job_id), data);
      queryClient.invalidateQueries({ queryKey: ['replay-jobs'] });
    },
  });
}

// Replay operations observability (G064): worker health + job counts. REST
// polling at the replay list cadence (5s); participates in manual refresh.
export function useReplayStatus({ tenant_id, limit }: { tenant_id: string; limit?: number }) {
  return useQuery({
    queryKey: queryKeys.replayStatus(tenant_id, limit),
    queryFn: () => api.getReplayStatus({ tenant_id, limit }),
    refetchInterval: 5000,
  });
}

// G066 app profiles (console + marketops). Static backend data; cache 5 min.
export function useAppProfiles() {
  return useQuery({
    queryKey: queryKeys.appProfiles,
    queryFn: api.getAppProfiles,
    staleTime: 5 * 60 * 1000,
  });
}

// G071 MarketOps asset universe (read-only). The seed changes slowly; cache 5 min.
export function useMarketOpsAssets(
  filter: MarketOpsAssetFilter = { tenant_id: 'tenant-local', universe_group: 'top50_megacap', active_only: true, limit: 50 },
) {
  return useQuery({
    queryKey: queryKeys.marketOpsAssets(filter),
    queryFn: () => api.listMarketOpsAssets(filter),
    staleTime: 5 * 60 * 1000,
  });
}

// G128 MarketOps asset options intelligence (read-only). Coverage + distribution
// run only while an asset is selected (tenant + symbol). The chain query also
// waits for a selected trade date. Not polled; short stale time mirrors the
// other MarketOps reads. Performs no ingestion and never calls live-preview.
export function useMarketOpsOptionsCoverage(tenantId: string, symbol: string | null) {
  return useQuery({
    queryKey: queryKeys.marketOpsOptionsCoverage(tenantId, symbol ?? ''),
    queryFn: () => api.getMarketOpsOptionsCoverage(tenantId, symbol!),
    enabled: !!tenantId && !!symbol,
    staleTime: 5 * 60 * 1000,
  });
}

export function useMarketOpsOptionsDistributions(
  tenantId: string,
  symbol: string | null,
  filter: MarketOpsOptionsDistributionFilter = {},
) {
  return useQuery({
    queryKey: queryKeys.marketOpsOptionsDistributions(tenantId, symbol ?? '', filter),
    queryFn: () => api.listMarketOpsOptionsDistributions(tenantId, symbol!, filter),
    enabled: !!tenantId && !!symbol,
    staleTime: 5 * 60 * 1000,
  });
}

export function useMarketOpsOptionsChain(
  tenantId: string,
  symbol: string | null,
  filter: MarketOpsOptionsChainFilter = {},
) {
  return useQuery({
    queryKey: queryKeys.marketOpsOptionsChain(tenantId, symbol ?? '', filter),
    queryFn: () => api.listMarketOpsOptionsChain(tenantId, symbol!, filter),
    enabled: !!tenantId && !!symbol && !!filter.trade_date,
    staleTime: 5 * 60 * 1000,
  });
}

// G139 MarketOps Opportunities workbench (read-only). The list runs with the
// active filters; detail + linked records are lazy (enabled only when an
// opportunity / contribution / evidence row is opened). hypothesis-evaluations
// powers both empty-queue diagnostics and contribution reason-code enrichment,
// so its `enabled` flag is caller-controlled. Short stale time keeps the queue
// responsive without refetch churn. No mutations.
export function useMarketOpsOpportunities(filter: MarketOpsOpportunityFilter = { tenant_id: 'tenant-local' }) {
  return useQuery({
    queryKey: queryKeys.marketOpsOpportunities(filter),
    queryFn: () => api.listMarketOpsOpportunities(filter),
    staleTime: 60 * 1000,
  });
}

export function useMarketOpsOpportunity(opportunityId: string | null, tenantId: string) {
  return useQuery({
    queryKey: queryKeys.marketOpsOpportunity(opportunityId ?? '', tenantId),
    queryFn: () => api.getMarketOpsOpportunity(opportunityId!, tenantId),
    enabled: !!opportunityId,
    staleTime: 60 * 1000,
  });
}

export function useMarketOpsHypothesisEvaluations(filter: MarketOpsHypothesisEvaluationFilter = {}, enabled = true) {
  return useQuery({
    queryKey: queryKeys.marketOpsHypothesisEvaluations(filter),
    queryFn: () => api.listMarketOpsHypothesisEvaluations(filter),
    enabled,
    staleTime: 60 * 1000,
  });
}

export function useMarketOpsHypothesis(key: string | null, version: string | null, tenantId: string) {
  return useQuery({
    queryKey: queryKeys.marketOpsHypothesis(key ?? '', version ?? '', tenantId),
    queryFn: () => api.getMarketOpsHypothesis(key!, version!, tenantId),
    enabled: !!key && !!version,
    staleTime: 5 * 60 * 1000,
  });
}

export function useMarketOpsEvidence(evidenceId: string | null) {
  return useQuery({
    queryKey: queryKeys.marketOpsEvidence(evidenceId ?? ''),
    queryFn: () => api.getMarketOpsEvidence(evidenceId!),
    enabled: !!evidenceId,
    staleTime: 5 * 60 * 1000,
  });
}

export function useMarketOpsMarketStateLineage(marketStateId: string | null) {
  return useQuery({
    queryKey: queryKeys.marketOpsMarketStateLineage(marketStateId ?? ''),
    queryFn: () => api.getMarketOpsMarketStateLineage(marketStateId!),
    enabled: !!marketStateId,
    staleTime: 5 * 60 * 1000,
  });
}

// G078 first-class MarketOps DSM artifact ledger. Artifacts are materialized
// from signal semantic evidence by the backend, so a short stale time keeps the
// workbench responsive while avoiding unnecessary refetch churn.
export function useMarketOpsDSMArtifacts(filter: MarketOpsDSMArtifactFilter = { tenant_id: 'tenant-local', limit: 50 }) {
  return useQuery({
    queryKey: queryKeys.marketOpsDSMArtifacts(filter),
    queryFn: () => api.listMarketOpsDSMArtifacts(filter),
    staleTime: 60 * 1000,
  });
}

export function useMarketOpsDSMArtifact(artifactId: string | null) {
  return useQuery({
    queryKey: queryKeys.marketOpsDSMArtifact(artifactId ?? ''),
    queryFn: () => api.getMarketOpsDSMArtifact(artifactId!),
    enabled: !!artifactId,
    staleTime: 60 * 1000,
  });
}

// G079 MarketOps DSM graph proposal ledger (read-only). Like artifacts, the
// ledger is materialized by the backend; a short stale time keeps the workbench
// responsive without refetch churn. The list is signal-scoped and only runs
// while a signal is selected. Detail is fetched on demand when a proposal row
// is expanded (guarded by a truthy proposal id).
export function useMarketOpsDSMGraphProposals(filter: MarketOpsDSMGraphProposalFilter = { tenant_id: 'tenant-local', limit: 50 }) {
  return useQuery({
    queryKey: queryKeys.marketOpsDSMGraphProposals(filter),
    queryFn: () => api.listMarketOpsDSMGraphProposals(filter),
    enabled: !!filter.signal_id || !!filter.artifact_id,
    staleTime: 60 * 1000,
  });
}

export function useMarketOpsDSMGraphProposal(proposalId: string | null) {
  return useQuery({
    queryKey: queryKeys.marketOpsDSMGraphProposal(proposalId ?? ''),
    queryFn: () => api.getMarketOpsDSMGraphProposal(proposalId!),
    enabled: !!proposalId,
    staleTime: 60 * 1000,
  });
}

export function useMutateMarketOpsDSMGraphProposalDecision() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (options: MarketOpsDSMGraphProposalDecisionOptions) =>
      api.mutateMarketOpsDSMGraphProposalDecision(options),
    onSuccess: (data, variables) => {
      queryClient.setQueryData(queryKeys.marketOpsDSMGraphProposal(variables.proposalId), data);
      queryClient.invalidateQueries({ queryKey: ['marketops-dsm-graph-proposals'] });
      queryClient.invalidateQueries({ queryKey: ['marketops-dsm-graph-proposal', variables.proposalId] });
    },
  });
}

// G081 MarketOps back-test workspace (isolated experimental runs). The runner is
// synchronous, so the list/detail do not poll — a create invalidation is enough
// for a new run to appear without a manual reload. Detail/signals/graph-proposals
// only run while a run is selected (guarded by a truthy run id).
export function useMarketOpsBacktests(filter: MarketOpsBacktestRunFilter = { tenant_id: 'tenant-local', limit: 50 }) {
  return useQuery({
    queryKey: queryKeys.marketOpsBacktests(filter),
    queryFn: () => api.listMarketOpsBacktests(filter),
  });
}

export function useMarketOpsBacktest(runId: string | null, tenantId: string = 'tenant-local') {
  return useQuery({
    queryKey: queryKeys.marketOpsBacktest(runId ?? '', tenantId),
    queryFn: () => api.getMarketOpsBacktest(runId!, tenantId),
    enabled: !!runId,
  });
}

export function useMarketOpsBacktestSignals(runId: string | null, filter: MarketOpsBacktestSignalFilter = {}) {
  return useQuery({
    queryKey: queryKeys.marketOpsBacktestSignals(runId ?? '', filter),
    queryFn: () => api.listMarketOpsBacktestSignals(runId!, filter),
    enabled: !!runId,
  });
}

export function useMarketOpsBacktestGraphProposals(
  runId: string | null,
  filter: MarketOpsBacktestGraphProposalFilter = {},
) {
  return useQuery({
    queryKey: queryKeys.marketOpsBacktestGraphProposals(runId ?? '', filter),
    queryFn: () => api.listMarketOpsBacktestGraphProposals(runId!, filter),
    enabled: !!runId,
  });
}

// On create, seed the detail cache with the returned (terminal) run and
// invalidate the list + detail prefixes so filtered tables refetch. The run is
// already terminal when the 201 returns (synchronous runner). The cache effect
// is extracted so it can be exercised against a real QueryClient in tests
// without rendering the hook.
export function applyBacktestCreateResult(queryClient: QueryClient, data: MarketOpsBacktestCreateResponse) {
  queryClient.setQueryData(
    queryKeys.marketOpsBacktest(data.backtest_run.run_id, data.backtest_run.tenant_id),
    { backtest_run: data.backtest_run },
  );
  queryClient.invalidateQueries({ queryKey: ['marketops-backtests'] });
  queryClient.invalidateQueries({ queryKey: ['marketops-backtest'] });
}

export function useCreateMarketOpsBacktest() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: MarketOpsBacktestCreateRequest) => api.createMarketOpsBacktest(body),
    onSuccess: (data) => applyBacktestCreateResult(queryClient, data),
  });
}

export function useMarketOpsBacktestCalibrationSummaries(filter: MarketOpsBacktestCalibrationSummaryFilter = { tenant_id: 'tenant-local', limit: 25 }) {
  return useQuery({
    queryKey: queryKeys.marketOpsBacktestCalibrationSummaries(filter),
    queryFn: () => api.listMarketOpsBacktestCalibrationSummaries(filter),
  });
}

export function useMarketOpsBacktestCalibrationSummary(summaryId: string | null) {
  return useQuery({
    queryKey: queryKeys.marketOpsBacktestCalibrationSummary(summaryId ?? ''),
    queryFn: () => api.getMarketOpsBacktestCalibrationSummary(summaryId!),
    enabled: !!summaryId,
  });
}

export function applyBacktestCalibrationSummaryCreateResult(queryClient: QueryClient, data: MarketOpsBacktestCalibrationSummaryResponse) {
  queryClient.setQueryData(
    queryKeys.marketOpsBacktestCalibrationSummary(data.calibration_summary.summary_id),
    data,
  );
  queryClient.invalidateQueries({ queryKey: ['marketops-backtest-calibration-summaries'] });
}

export function useCreateMarketOpsBacktestCalibrationSummary() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: MarketOpsBacktestCalibrationSummaryCreateRequest) => api.createMarketOpsBacktestCalibrationSummary(body),
    onSuccess: (data) => applyBacktestCalibrationSummaryCreateResult(queryClient, data),
  });
}

// G083 persisted calibration baselines + stored comparisons. Like the G082
// summary list, these are not polled; a create invalidation is enough for a new
// baseline/comparison to appear without a manual reload. Detail only runs while
// a baseline/comparison id is selected (guarded by a truthy id).
export function useMarketOpsBacktestCalibrationBaselines(filter: MarketOpsBacktestCalibrationBaselineFilter = { tenant_id: 'tenant-local', limit: 50 }) {
  return useQuery({
    queryKey: queryKeys.marketOpsBacktestCalibrationBaselines(filter),
    queryFn: () => api.listMarketOpsBacktestCalibrationBaselines(filter),
  });
}

export function useMarketOpsBacktestCalibrationBaseline(baselineId: string | null) {
  return useQuery({
    queryKey: queryKeys.marketOpsBacktestCalibrationBaseline(baselineId ?? ''),
    queryFn: () => api.getMarketOpsBacktestCalibrationBaseline(baselineId!),
    enabled: !!baselineId,
  });
}

// On create, seed the baseline detail cache and invalidate baseline list/detail
// prefixes. Only calibration baseline queries are touched — never production
// signal, DSM artifact, or graph proposal queries.
export function applyBacktestCalibrationBaselineCreateResult(
  queryClient: QueryClient,
  data: MarketOpsBacktestCalibrationBaselineResponse,
) {
  queryClient.setQueryData(
    queryKeys.marketOpsBacktestCalibrationBaseline(data.calibration_baseline.baseline_id),
    data,
  );
  queryClient.invalidateQueries({ queryKey: ['marketops-backtest-calibration-baselines'] });
  queryClient.invalidateQueries({ queryKey: ['marketops-backtest-calibration-baseline'] });
}

export function useCreateMarketOpsBacktestCalibrationBaseline() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: MarketOpsBacktestCalibrationBaselineCreateRequest) =>
      api.createMarketOpsBacktestCalibrationBaseline(body),
    onSuccess: (data) => applyBacktestCalibrationBaselineCreateResult(queryClient, data),
  });
}

export function useMarketOpsBacktestCalibrationComparisons(filter: MarketOpsBacktestCalibrationComparisonFilter = { tenant_id: 'tenant-local', limit: 50 }) {
  return useQuery({
    queryKey: queryKeys.marketOpsBacktestCalibrationComparisons(filter),
    queryFn: () => api.listMarketOpsBacktestCalibrationComparisons(filter),
    // The comparisons list is baseline-scoped; only run while a baseline is selected.
    enabled: !!filter.baseline_id,
  });
}

export function useMarketOpsBacktestCalibrationComparison(comparisonId: string | null) {
  return useQuery({
    queryKey: queryKeys.marketOpsBacktestCalibrationComparison(comparisonId ?? ''),
    queryFn: () => api.getMarketOpsBacktestCalibrationComparison(comparisonId!),
    enabled: !!comparisonId,
  });
}

// On create, seed the comparison detail cache and invalidate comparison
// list/detail prefixes. Only calibration comparison queries are touched.
export function applyBacktestCalibrationComparisonCreateResult(
  queryClient: QueryClient,
  data: MarketOpsBacktestCalibrationComparisonResponse,
) {
  queryClient.setQueryData(
    queryKeys.marketOpsBacktestCalibrationComparison(data.calibration_comparison.comparison_id),
    data,
  );
  queryClient.invalidateQueries({ queryKey: ['marketops-backtest-calibration-comparisons'] });
  queryClient.invalidateQueries({ queryKey: ['marketops-backtest-calibration-comparison'] });
}

export function useCreateMarketOpsBacktestCalibrationComparison() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: MarketOpsBacktestCalibrationComparisonCreateRequest) =>
      api.createMarketOpsBacktestCalibrationComparison(body),
    onSuccess: (data) => applyBacktestCalibrationComparisonCreateResult(queryClient, data),
  });
}

// G085 label-aware back-test evaluations. The list is run-scoped and only runs
// while a run is selected (guarded by a truthy run id); detail only runs while
// an evaluation id is selected. A create invalidation is enough for a new
// evaluation to appear without a manual reload.
export function useMarketOpsBacktestEvaluations(filter: MarketOpsBacktestEvaluationFilter = { tenant_id: 'tenant-local', limit: 50 }) {
  return useQuery({
    queryKey: queryKeys.marketOpsBacktestEvaluations(filter),
    queryFn: () => api.listMarketOpsBacktestEvaluations(filter),
    enabled: !!filter.run_id,
  });
}

export function useMarketOpsBacktestEvaluation(evaluationId: string | null) {
  return useQuery({
    queryKey: queryKeys.marketOpsBacktestEvaluation(evaluationId ?? ''),
    queryFn: () => api.getMarketOpsBacktestEvaluation(evaluationId!),
    enabled: !!evaluationId,
  });
}

// On create, seed the evaluation detail cache and invalidate evaluation
// list/detail prefixes. Only evaluation queries are touched — never production
// DSM graph proposal, signal, or back-test run/graph-proposal queries.
export function applyBacktestEvaluationCreateResult(
  queryClient: QueryClient,
  data: MarketOpsBacktestEvaluationResponse,
) {
  queryClient.setQueryData(
    queryKeys.marketOpsBacktestEvaluation(data.backtest_evaluation.evaluation_id),
    data,
  );
  queryClient.invalidateQueries({ queryKey: ['marketops-backtest-evaluations'] });
  queryClient.invalidateQueries({ queryKey: ['marketops-backtest-evaluation'] });
}

export function useCreateMarketOpsBacktestEvaluation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: MarketOpsBacktestEvaluationCreateRequest) => api.createMarketOpsBacktestEvaluation(body),
    onSuccess: (data) => applyBacktestEvaluationCreateResult(queryClient, data),
  });
}

// G086 promotion review candidates. Not polled; a create or decision
// invalidation is enough for the panel to refresh without a manual reload.
// Detail only runs while a candidate id is selected (guarded by a truthy id).
export function useMarketOpsBacktestPromotionCandidates(filter: MarketOpsBacktestPromotionCandidateFilter = { tenant_id: 'tenant-local', limit: 50 }) {
  return useQuery({
    queryKey: queryKeys.marketOpsBacktestPromotionCandidates(filter),
    queryFn: () => api.listMarketOpsBacktestPromotionCandidates(filter),
  });
}

export function useMarketOpsBacktestPromotionCandidate(candidateId: string | null) {
  return useQuery({
    queryKey: queryKeys.marketOpsBacktestPromotionCandidate(candidateId ?? ''),
    queryFn: () => api.getMarketOpsBacktestPromotionCandidate(candidateId!),
    enabled: !!candidateId,
  });
}

// Seed the candidate detail cache and invalidate promotion candidate list/detail
// prefixes. Only promotion queries are touched — never production DSM graph
// proposal, signal, alert, insight, or policy queries. Shared by create and
// decision: both return the full candidate and need the same refresh.
function applyBacktestPromotionCandidateResult(
  queryClient: QueryClient,
  data: MarketOpsBacktestPromotionCandidateResponse,
) {
  queryClient.setQueryData(
    queryKeys.marketOpsBacktestPromotionCandidate(data.promotion_candidate.candidate_id),
    data,
  );
  queryClient.invalidateQueries({ queryKey: ['marketops-backtest-promotion-candidates'] });
  queryClient.invalidateQueries({ queryKey: ['marketops-backtest-promotion-candidate'] });
}

// Extracted so it can be exercised against a real QueryClient in tests without
// rendering the hook. Frames the shared refresh as the create mutation.
export function applyBacktestPromotionCandidateCreateResult(
  queryClient: QueryClient,
  data: MarketOpsBacktestPromotionCandidateResponse,
) {
  applyBacktestPromotionCandidateResult(queryClient, data);
}

// Frames the same shared refresh as the decision mutation.
export function applyBacktestPromotionCandidateDecisionResult(
  queryClient: QueryClient,
  data: MarketOpsBacktestPromotionCandidateResponse,
) {
  applyBacktestPromotionCandidateResult(queryClient, data);
}

export function useCreateMarketOpsBacktestPromotionCandidate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: MarketOpsBacktestPromotionCandidateCreateRequest) =>
      api.createMarketOpsBacktestPromotionCandidate(body),
    onSuccess: (data) => applyBacktestPromotionCandidateCreateResult(queryClient, data),
  });
}

export function useDecideMarketOpsBacktestPromotionCandidate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (vars: { candidateId: string; request: MarketOpsBacktestPromotionCandidateDecisionRequest }) =>
      api.decideMarketOpsBacktestPromotionCandidate(vars.candidateId, vars.request),
    onSuccess: (data) => applyBacktestPromotionCandidateDecisionResult(queryClient, data),
  });
}

// Cancel with an optimistic update: mark the matching queued/running job as
// `canceled` in every list cache immediately, roll back on error, then write
// the authoritative returned job into the detail cache and refetch.
type ReplayJobsCache = { replay_jobs: ReplayJob[] };

export function useCancelReplayJob() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (vars: { replayJobId: string; reason?: string; note?: string }) =>
      api.cancelReplayJob(vars.replayJobId, { reason: vars.reason, note: vars.note }),
    onMutate: async ({ replayJobId }) => {
      // Stop in-flight list refetches from clobbering the optimistic update.
      await queryClient.cancelQueries({ queryKey: ['replay-jobs'] });
      const previous = queryClient.getQueriesData<ReplayJobsCache>({ queryKey: ['replay-jobs'] });
      queryClient.setQueriesData<ReplayJobsCache>({ queryKey: ['replay-jobs'] }, (old) => {
        if (!old) return old;
        return {
          replay_jobs: old.replay_jobs.map((j) =>
            j.replay_job_id === replayJobId && (j.status === 'queued' || j.status === 'running')
              ? { ...j, status: 'canceled' }
              : j,
          ),
        };
      });
      return { previous };
    },
    onError: (_err, _vars, context) => {
      // Restore the pre-mutation list caches.
      context?.previous?.forEach(([key, data]) => {
        if (data !== undefined) queryClient.setQueryData(key, data);
      });
    },
    onSuccess: (data, { replayJobId }) => {
      queryClient.setQueryData(queryKeys.replayJob(replayJobId), data);
    },
    onSettled: async (_data, _err, { replayJobId }) => {
      await queryClient.invalidateQueries({ queryKey: ['replay-jobs'] });
      queryClient.invalidateQueries({ queryKey: ['replay-job', replayJobId] });
    },
  });
}

// Lifecycle mutations: on success, write the returned record into the detail cache
// (instant update) and invalidate the list prefix so filtered tables + Dashboard
// summaries (which sit under ['alerts']/['insights']) refetch. When auth is enabled
// the gateway derives the actor from the token; the operator-local placeholder is
// only sent in auth-disabled (local dev) mode — see api/client.ts.
export function useMutateAlertLifecycle() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (options: AlertLifecycleMutationOptions) => api.mutateAlertLifecycle(options),
    onSuccess: (data, variables) => {
      queryClient.setQueryData(queryKeys.alert(variables.alertId), data);
      queryClient.invalidateQueries({ queryKey: ['alerts'] });
    },
  });
}

export function useMutateInsightLifecycle() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (options: InsightLifecycleMutationOptions) => api.mutateInsightLifecycle(options),
    onSuccess: (data, variables) => {
      queryClient.setQueryData(queryKeys.insight(variables.insightId), data);
      queryClient.invalidateQueries({ queryKey: ['insights'] });
    },
  });
}

// G088 Syncratic synthesized insights + context windows (read-only review).
// Lists are not polled; detail hooks only run while an id is truthy.
export function useSyncraticInsights(filter: SyncraticInsightFilter = { tenant_id: 'tenant-local', limit: 50 }) {
  return useQuery({
    queryKey: queryKeys.syncraticInsights(filter),
    queryFn: () => api.listSyncraticInsights(filter),
  });
}

export function useSyncraticInsight(insightId: string | null) {
  return useQuery({
    queryKey: queryKeys.syncraticInsight(insightId ?? ''),
    queryFn: () => api.getSyncraticInsight(insightId!),
    enabled: !!insightId,
  });
}

export function useSyncraticContextWindows(filter: SyncraticContextWindowFilter = { tenant_id: 'tenant-local', limit: 50 }) {
  return useQuery({
    queryKey: queryKeys.syncraticContextWindows(filter),
    queryFn: () => api.listSyncraticContextWindows(filter),
  });
}

export function useSyncraticContextWindow(contextWindowId: string | null) {
  return useQuery({
    queryKey: queryKeys.syncraticContextWindow(contextWindowId ?? ''),
    queryFn: () => api.getSyncraticContextWindow(contextWindowId!),
    enabled: !!contextWindowId,
  });
}

// After a bounded materialization, invalidate Syncratic insight + context-window
// list/detail prefixes so newly materialized rows appear without a manual reload.
// Only Syncratic queries are touched — never alert lifecycle, graph proposal,
// back-test, calibration, promotion, or production signal queries. A dry-run
// preview (G091) writes nothing, so it skips invalidation entirely.
export function applySyncraticMaterializeResult(
  queryClient: QueryClient,
  data: SyncraticMaterializationResponse,
) {
  if (data?.materialization?.dry_run) return;
  queryClient.invalidateQueries({ queryKey: ['syncratic-insights'] });
  queryClient.invalidateQueries({ queryKey: ['syncratic-insight'] });
  queryClient.invalidateQueries({ queryKey: ['syncratic-context-windows'] });
  queryClient.invalidateQueries({ queryKey: ['syncratic-context-window'] });
}

export function useMaterializeSyncraticContexts() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (request: SyncraticMaterializeRequest) => api.materializeSyncraticContexts(request),
    onSuccess: (data) => applySyncraticMaterializeResult(queryClient, data),
  });
}

// G090 operator-triggered Syncratic Ask. On success the route returns the full
// refreshed insight, so seed the detail cache for an instant update (the skip path
// returns the pre-existing insight too), then invalidate Syncratic insight +
// context-window list/detail prefixes so badges and counts refresh. Only Syncratic
// queries are touched — never alert, signal, graph proposal, or production queries.
export function applySyncraticAskResult(queryClient: QueryClient, data: SyncraticAskResponse) {
  const insightId = data.syncratic_insight?.syncratic_insight_id;
  if (insightId) {
    queryClient.setQueryData(queryKeys.syncraticInsight(insightId), {
      syncratic_insight: data.syncratic_insight,
    });
  }
  queryClient.invalidateQueries({ queryKey: ['syncratic-insights'] });
  queryClient.invalidateQueries({ queryKey: ['syncratic-insight'] });
  queryClient.invalidateQueries({ queryKey: ['syncratic-context-windows'] });
  queryClient.invalidateQueries({ queryKey: ['syncratic-context-window'] });
}

export function useAskSyncraticContextWindow() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (vars: { contextWindowId: string; request: SyncraticAskRequest }) =>
      api.askSyncraticContextWindow(vars.contextWindowId, vars.request),
    onSuccess: (data) => applySyncraticAskResult(queryClient, data),
  });
}

// G109 algorithm execution visibility (read-only). Lists/detail only run while
// their selector is truthy; the summary is scoped to a selected execution request.
// No mutations, polling, or automatic execution.
export function useAlgorithmDefinitions(filter: AlgorithmDefinitionFilter = { tenant_id: 'tenant-local' }) {
  return useQuery({
    queryKey: queryKeys.algorithmDefinitions(filter),
    queryFn: () => api.listAlgorithmDefinitions(filter),
  });
}

export function useAlgorithmDefinition(algorithmId: string | null, tenantId: string = 'tenant-local') {
  return useQuery({
    queryKey: queryKeys.algorithmDefinition(algorithmId ?? '', tenantId),
    queryFn: () => api.getAlgorithmDefinition(algorithmId!, tenantId),
    enabled: !!algorithmId,
  });
}

export function useAlgorithmExecutionRequests(filter: AlgorithmExecutionRequestFilter = { tenant_id: 'tenant-local' }) {
  return useQuery({
    queryKey: queryKeys.algorithmExecutionRequests(filter),
    queryFn: () => api.listAlgorithmExecutionRequests(filter),
    // The visibility workflow drills from a selected algorithm; do not fetch the
    // unfiltered execution-request universe before one is chosen.
    enabled: !!filter.algorithm_id,
  });
}

export function useAlgorithmExecutionRequest(executionRequestId: string | null, tenantId: string = 'tenant-local') {
  return useQuery({
    queryKey: queryKeys.algorithmExecutionRequest(executionRequestId ?? '', tenantId),
    queryFn: () => api.getAlgorithmExecutionRequest(executionRequestId!, tenantId),
    enabled: !!executionRequestId,
  });
}

export function useAlgorithmExecutionSummary(executionRequestId: string | null, tenantId: string = 'tenant-local', limit = 10) {
  return useQuery({
    queryKey: queryKeys.algorithmExecutionSummary(executionRequestId ?? '', tenantId, limit),
    queryFn: () => api.getAlgorithmExecutionSummary(executionRequestId!, tenantId, limit),
    enabled: !!executionRequestId,
  });
}

export function useAlgorithmResults(filter: AlgorithmResultFilter = { tenant_id: 'tenant-local' }) {
  return useQuery({
    queryKey: queryKeys.algorithmResults(filter),
    queryFn: () => api.listAlgorithmResults(filter),
  });
}

export function useAlgorithmResult(algorithmResultId: string | null, tenantId: string = 'tenant-local') {
  return useQuery({
    queryKey: queryKeys.algorithmResult(algorithmResultId ?? '', tenantId),
    queryFn: () => api.getAlgorithmResult(algorithmResultId!, tenantId),
    enabled: !!algorithmResultId,
  });
}

// G113/G114 algorithm signal proposals review surface (read-only review over the
// G111/G112 backend). The list is not polled; detail only runs while a proposal
// id is selected (guarded by a truthy id). tenant_id defaults to tenant-local.
export function useAlgorithmSignalProposals(filter: AlgorithmSignalProposalFilter = { tenant_id: 'tenant-local' }) {
  return useQuery({
    queryKey: queryKeys.algorithmSignalProposals(filter),
    queryFn: () => api.listAlgorithmSignalProposals(filter),
  });
}

export function useAlgorithmSignalProposal(proposalId: string | null, tenantId: string = 'tenant-local') {
  return useQuery({
    queryKey: queryKeys.algorithmSignalProposal(proposalId ?? '', tenantId),
    queryFn: () => api.getAlgorithmSignalProposal(proposalId!, tenantId),
    enabled: !!proposalId,
  });
}

// G116 review-coverage summary over the G115 endpoint. Couples to the active
// proposal filters (minus limit). Not polled — a decision invalidates the
// proposal list/detail prefixes, and the summary re-runs on filter changes.
export function useAlgorithmSignalProposalSummary(filter: AlgorithmSignalProposalFilter = { tenant_id: 'tenant-local' }) {
  return useQuery({
    queryKey: queryKeys.algorithmSignalProposalSummary(filter),
    queryFn: () => api.getAlgorithmSignalProposalSummary(filter),
  });
}

// G119 read-only materialization preflight over the G118 endpoint. Couples to the
// active proposal filters (including limit) plus the preflight-only params. Not
// polled — filter changes and review decisions both trigger a refetch (the
// decision handler invalidates this prefix because a review flips preflight_status).
export function useAlgorithmSignalMaterializationPreflight(
  filter: AlgorithmSignalMaterializationPreflightFilter = { tenant_id: 'tenant-local' },
) {
  return useQuery({
    queryKey: queryKeys.algorithmSignalMaterializationPreflight(filter),
    queryFn: () => api.getAlgorithmSignalMaterializationPreflight(filter),
  });
}

// On decision, seed the proposal detail cache with the returned (reviewed) row
// and invalidate proposal list/detail/summary/preflight prefixes so filtered
// tables, badges, the G116 coverage summary, and the G119 preflight (whose
// preflight_status depends on review status) refresh. Only algorithm-signal-
// proposal queries are touched — never production signal, alert, insight, graph
// proposal, or algorithm execution queries. The decision records review metadata
// only; it materializes nothing.
export function applyAlgorithmSignalProposalDecisionResult(
  queryClient: QueryClient,
  data: AlgorithmSignalProposalResponse,
  proposalId: string,
  tenantId: string,
) {
  queryClient.setQueryData(queryKeys.algorithmSignalProposal(proposalId, tenantId), data);
  queryClient.invalidateQueries({ queryKey: ['algorithm-signal-proposals'] });
  queryClient.invalidateQueries({ queryKey: ['algorithm-signal-proposal', proposalId, tenantId] });
  queryClient.invalidateQueries({ queryKey: ['algorithm-signal-proposal-summary'] });
  queryClient.invalidateQueries({ queryKey: ['algorithm-signal-materialization-preflight'] });
}

export function useDecideAlgorithmSignalProposal() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (vars: { proposalId: string; tenantId: string; request: AlgorithmSignalProposalDecisionRequest }) =>
      api.decideAlgorithmSignalProposal(vars.proposalId, vars.request),
    onSuccess: (data, variables) =>
      applyAlgorithmSignalProposalDecisionResult(queryClient, data, variables.proposalId, variables.tenantId),
  });
}

// G121 materialization ledger for the selected proposal. Only runs while a
// proposal is selected (guarded by a truthy proposal_id). Read-only.
export function useAlgorithmSignalMaterializations(filter: AlgorithmSignalMaterializationFilter = { tenant_id: 'tenant-local' }) {
  return useQuery({
    queryKey: queryKeys.algorithmSignalMaterializations(filter),
    queryFn: () => api.listAlgorithmSignalMaterializations(filter),
    enabled: !!filter.proposal_id,
  });
}

// G123 single-proposal materialization mutation (G122 backend). On success the
// ledger refetches (the new row appears), the preflight refreshes (the proposal
// is no longer would-write), and the proposal list/detail/summary refresh so
// badges/states stay consistent. The POST records a production signal — this is
// the only materializing control on the surface. The backend is idempotent on
// repeat, so a re-submit returns the existing row rather than erroring.
export function applyMaterializeAlgorithmSignalProposalResult(queryClient: QueryClient) {
  queryClient.invalidateQueries({ queryKey: ['algorithm-signal-materializations'] });
  queryClient.invalidateQueries({ queryKey: ['algorithm-signal-materialization-preflight'] });
  queryClient.invalidateQueries({ queryKey: ['algorithm-signal-proposals'] });
  queryClient.invalidateQueries({ queryKey: ['algorithm-signal-proposal-summary'] });
}

export function useMaterializeAlgorithmSignalProposal() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (vars: { proposalId: string; request: AlgorithmSignalMaterializationRequest }) =>
      api.materializeAlgorithmSignalProposal(vars.proposalId, vars.request),
    onSuccess: () => applyMaterializeAlgorithmSignalProposalResult(queryClient),
  });
}
