import type {
  HealthResponse,
  SchedulerRunsResponse,
  SchedulerRunResponse,
  ProviderUsageResponse,
  RawEventsResponse,
  RawEventResponse,
  IdempotencyResponse,
  CatalogSourcesResponse,
  CatalogPipelinesResponse,
  CatalogRulesResponse,
  NormalizedEventsResponse,
  NormalizedEventResponse,
  SignalsResponse,
  SignalResponse,
  AlertsResponse,
  AlertResponse,
  InsightsResponse,
  InsightResponse,
  RawEventFilter,
  NormalizedEventFilter,
  SignalFilter,
  AlertFilter,
  InsightFilter,
  AlertLifecycleMutationOptions,
  InsightLifecycleMutationOptions,
  ReplayJobFilter,
  ReplayJobsResponse,
  ReplayJobResponse,
  ReplayJobCreateRequest,
  ReplayJobCancelRequest,
  ReplayOperationsStatusResponse,
  AppProfilesResponse,
  MarketOpsAssetsResponse,
  MarketOpsAsset,
  MarketOpsAssetBackfillJob,
  MarketOpsAssetCreateRequest,
  MarketOpsAssetDisplayNameRequest,
  MarketOpsAssetDisplaySectorRequest,
  MarketOpsTickerValidation,
  MarketOpsAssetOnboardRequest,
  MarketOpsAssetBackfillJobsResponse,
  MarketOpsAssetBackfillCreateRequest,
  MarketOpsAssetQuotesResponse,
  MarketOpsIntradayConditionsResponse,
  MarketOpsAssetFilter,
  MarketOpsOptionsCoverageResponse,
  MarketOpsOptionsDistributionsResponse,
  MarketOpsOptionsChainResponse,
  MarketOpsOptionsChainFilter,
  MarketOpsOptionsDistributionFilter,
  MarketOpsOpportunitiesResponse,
  MarketOpsOpportunityResponse,
  MarketOpsOpportunityFilter,
  MarketOpsHypothesisEvaluationsResponse,
  MarketOpsHypothesisEvaluationFilter,
  MarketOpsAlgorithmAdjudicationFilter,
  MarketOpsAlgorithmAdjudicationsResponse,
  MarketOpsQuantitativeSeriesResponse,
  MarketOpsAssetAlgorithmObservationsResponse,
  MarketOpsRiskRewardSummariesResponse,
  MarketOpsSignalOverviewResponse,
  MarketOpsHypothesisResponse,
  MarketOpsEvidenceResponse,
  MarketOpsMarketStateLineageResponse,
  MarketOpsMarketStatesResponse,
  MarketOpsMarketStateResponse,
  MarketOpsMarketStateFilter,
  MarketOpsFeatureDefinitionsResponse,
  MarketOpsFeatureDefinitionFilter,
  MarketOpsFeatureObservationsResponse,
  MarketOpsFeatureObservationFilter,
  MarketOpsStateTransitionsResponse,
  MarketOpsStateTransitionFilter,
  MarketOpsEvidencesResponse,
  MarketOpsEvidenceFilter,
  MarketOpsHypothesesResponse,
  MarketOpsHypothesisListFilter,
  MarketOpsOutcomesResponse,
  MarketOpsOutcomeResponse,
  MarketOpsOutcomeFilter,
  MarketOpsOpportunityDispositionsResponse,
  MarketOpsOpportunityDispositionResponse,
  MarketOpsOpportunityDispositionRequest,
  MarketOpsOpportunityDispositionFilter,
  MarketOpsIntelligenceReadinessFilter,
  MarketOpsIntelligenceReadinessResponse,
  MarketOpsDSMArtifactsResponse,
  MarketOpsDSMArtifactResponse,
  MarketOpsDSMArtifactFilter,
  MarketOpsDSMGraphProposalsResponse,
  MarketOpsDSMGraphProposalResponse,
  MarketOpsDSMGraphProposalFilter,
  MarketOpsDSMGraphProposalDecisionOptions,
  MarketOpsBacktestRunsResponse,
  MarketOpsBacktestRunResponse,
  MarketOpsBacktestCreateRequest,
  MarketOpsBacktestCreateResponse,
  MarketOpsBacktestSignalsResponse,
  MarketOpsBacktestGraphProposalsResponse,
  MarketOpsBacktestRunFilter,
  MarketOpsBacktestSignalFilter,
  MarketOpsBacktestGraphProposalFilter,
  MarketOpsBacktestCalibrationSummariesResponse,
  MarketOpsBacktestCalibrationSummaryResponse,
  MarketOpsBacktestCalibrationSummaryCreateRequest,
  MarketOpsBacktestCalibrationSummaryFilter,
  MarketOpsBacktestCalibrationBaselinesResponse,
  MarketOpsBacktestCalibrationBaselineResponse,
  MarketOpsBacktestCalibrationBaselineCreateRequest,
  MarketOpsBacktestCalibrationBaselineFilter,
  MarketOpsBacktestCalibrationComparisonsResponse,
  MarketOpsBacktestCalibrationComparisonResponse,
  MarketOpsBacktestCalibrationComparisonCreateRequest,
  MarketOpsBacktestCalibrationComparisonFilter,
  MarketOpsBacktestEvaluationsResponse,
  MarketOpsBacktestEvaluationResponse,
  MarketOpsBacktestEvaluationCreateRequest,
  MarketOpsBacktestEvaluationFilter,
  MarketOpsBacktestPromotionCandidatesResponse,
  MarketOpsBacktestPromotionCandidateResponse,
  MarketOpsBacktestPromotionCandidateCreateRequest,
  MarketOpsBacktestPromotionCandidateDecisionRequest,
  MarketOpsBacktestPromotionCandidateFilter,
  SyncraticInsightsResponse,
  SyncraticInsightResponse,
  SyncraticContextWindowsResponse,
  SyncraticContextWindowResponse,
  SyncraticMaterializationResponse,
  SyncraticInsightFilter,
  SyncraticContextWindowFilter,
  SyncraticMaterializeRequest,
  SyncraticAskRequest,
  SyncraticAskResponse,
  AlgorithmDefinitionFilter,
  AlgorithmDefinitionsResponse,
  AlgorithmDefinitionResponse,
  AlgorithmExecutionRequestFilter,
  AlgorithmExecutionRequestsResponse,
  AlgorithmExecutionRequestResponse,
  AlgorithmExecutionSummaryResponse,
  AlgorithmResultFilter,
  AlgorithmResultsResponse,
  AlgorithmResultResponse,
  AlgorithmSignalProposalFilter,
  AlgorithmSignalProposalsResponse,
  AlgorithmSignalProposalResponse,
  AlgorithmSignalProposalDecisionRequest,
  AlgorithmSignalProposalSummaryResponse,
  AlgorithmSignalMaterializationPreflightFilter,
  AlgorithmSignalMaterializationPreflightResponse,
  AlgorithmSignalMaterializationRequest,
  AlgorithmSignalMaterializationResponse,
  AlgorithmSignalMaterializationsResponse,
  AlgorithmSignalMaterializationFilter,
} from '../types';
import { authConfig } from '../auth/config';
import { getAccessToken } from '../auth/session';

// Typed API error. Maps gateway error bodies {"error":<code>,"message":<text>}
// plus network failures, so the UI can render endpoint + message.
export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
    message: string,
    public readonly endpoint: string,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

export function isApiError(e: unknown): e is ApiError {
  return e instanceof ApiError;
}

// Browser-facing base URL. In dev, leave UNSET so requests are same-origin and
// the Vite proxy forwards them to the gateway (the gateway has no CORS).
// Set an absolute URL only when the gateway emits CORS headers.
const BASE_URL = (import.meta.env.VITE_SIGNALOPS_API_BASE_URL ?? '').replace(/\/+$/, '');

export function buildUrl(path: string, params?: Record<string, string | number | undefined>): string {
  const base = BASE_URL || window.location.origin;
  const url = new URL(path, base);
  if (params) {
    for (const [key, value] of Object.entries(params)) {
      if (value !== undefined && value !== '') {
        url.searchParams.set(key, String(value));
      }
    }
  }
  return url.toString();
}

// Bearer token attached to every request when auth is enabled and a token is held.
// /healthz and /readyz stay usable unauthenticated because the token is absent until login.
function authHeaders(): Record<string, string> {
  const token = authConfig.authEnabled ? getAccessToken() : null;
  return token ? { Authorization: `Bearer ${token}` } : {};
}

async function get<T>(path: string, params?: Record<string, string | number | undefined>, cache: RequestCache = 'default'): Promise<T> {
  const endpoint = buildUrl(path, params);
  let res: Response;
  try {
    res = await fetch(endpoint, { cache, headers: { Accept: 'application/json', ...authHeaders() } });
  } catch {
    throw new ApiError(0, 'network_error', 'Gateway unreachable', endpoint);
  }
  if (!res.ok) {
    let code = 'http_error';
    let message = res.statusText || `HTTP ${res.status}`;
    try {
      const body = await res.json();
      if (body && typeof body.error === 'string') code = body.error;
      if (body && typeof body.message === 'string') message = body.message;
    } catch {
      /* non-JSON error body */
    }
    throw new ApiError(res.status, code, message, endpoint);
  }
  return (await res.json()) as T;
}

async function post<T>(
  path: string,
  body?: unknown,
  headers?: Record<string, string>,
): Promise<T> {
  const endpoint = buildUrl(path);
  let res: Response;
  try {
    res = await fetch(endpoint, {
      method: 'POST',
      headers: { Accept: 'application/json', 'Content-Type': 'application/json', ...authHeaders(), ...headers },
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch {
    throw new ApiError(0, 'network_error', 'Gateway unreachable', endpoint);
  }
  if (!res.ok) {
    let code = 'http_error';
    let message = res.statusText || `HTTP ${res.status}`;
    try {
      const errBody = await res.json();
      if (errBody && typeof errBody.error === 'string') code = errBody.error;
      if (errBody && typeof errBody.message === 'string') message = errBody.message;
    } catch {
      /* non-JSON error body */
    }
    throw new ApiError(res.status, code, message, endpoint);
  }
  return (await res.json()) as T;
}

async function patch<T>(path: string, body?: unknown): Promise<T> {
  const endpoint = buildUrl(path);
  let res: Response;
  try {
    res = await fetch(endpoint, {
      method: 'PATCH',
      headers: { Accept: 'application/json', 'Content-Type': 'application/json', ...authHeaders() },
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch {
    throw new ApiError(0, 'network_error', 'Gateway unreachable', endpoint);
  }
  if (!res.ok) {
    let code = 'http_error';
    let message = res.statusText || `HTTP ${res.status}`;
    try {
      const errBody = await res.json();
      if (errBody && typeof errBody.error === 'string') code = errBody.error;
      if (errBody && typeof errBody.message === 'string') message = errBody.message;
    } catch {
      /* non-JSON error body */
    }
    throw new ApiError(res.status, code, message, endpoint);
  }
  return (await res.json()) as T;
}

export const api = {
  healthz: () => get<HealthResponse>('/healthz'),
  readyz: () => get<HealthResponse>('/readyz'),
  listRuns: (limit = 50) => get<SchedulerRunsResponse>('/v1/scheduler/runs', { limit }),
  getRun: (runId: string) =>
    get<SchedulerRunResponse>(`/v1/scheduler/runs/${encodeURIComponent(runId)}`),
  listProviderUsage: (runId?: string, limit = 50) =>
    get<ProviderUsageResponse>('/v1/provider-usage', { run_id: runId, limit }),
  listRawEvents: (filter: RawEventFilter = {}) =>
    get<RawEventsResponse>('/v1/raw-events', {
      tenant_id: filter.tenant_id,
      app_id: filter.app_id,
      domain: filter.domain,
      use_case: filter.use_case,
      source_id: filter.source_id,
      dataset: filter.dataset,
      limit: filter.limit ?? 50,
    }),
  getRawEvent: (eventId: string) =>
    get<RawEventResponse>(`/v1/raw-events/${encodeURIComponent(eventId)}`),
  getIdempotency: (tenantId: string, sourceId: string, idempotencyKey: string) =>
    get<IdempotencyResponse>('/v1/idempotency', {
      tenant_id: tenantId,
      source_id: sourceId,
      idempotency_key: idempotencyKey,
    }),
  listCatalogSources: (tenantId = 'tenant-local', limit = 50) =>
    get<CatalogSourcesResponse>(`/v1/tenants/${encodeURIComponent(tenantId)}/catalog/sources`, { limit }),
  listCatalogPipelines: (tenantId = 'tenant-local', limit = 200) =>
    get<CatalogPipelinesResponse>(`/v1/tenants/${encodeURIComponent(tenantId)}/catalog/pipelines`, { limit }),
  listCatalogRules: (tenantId = 'tenant-local', limit = 50) =>
    get<CatalogRulesResponse>(`/v1/tenants/${encodeURIComponent(tenantId)}/catalog/rules`, { limit }),
  listNormalizedEvents: (filter: NormalizedEventFilter = {}) =>
    get<NormalizedEventsResponse>('/v1/normalized-events', {
      tenant_id: filter.tenant_id,
      app_id: filter.app_id,
      domain: filter.domain,
      use_case: filter.use_case,
      source_id: filter.source_id,
      dataset: filter.dataset,
      limit: filter.limit ?? 50,
    }),
  getNormalizedEvent: (eventId: string) =>
    get<NormalizedEventResponse>(`/v1/normalized-events/${encodeURIComponent(eventId)}`),
  listSignals: (filter: SignalFilter = {}) =>
    get<SignalsResponse>('/v1/signals', {
      tenant_id: filter.tenant_id,
      app_id: filter.app_id,
      domain: filter.domain,
      use_case: filter.use_case,
      source_id: filter.source_id,
      dataset: filter.dataset,
      detector_id: filter.detector_id,
      severity: filter.severity,
      limit: filter.limit ?? 50,
    }),
  getSignal: (signalId: string) =>
    get<SignalResponse>(`/v1/signals/${encodeURIComponent(signalId)}`),
  listAlerts: (filter: AlertFilter = {}) =>
    get<AlertsResponse>('/v1/alerts', {
      tenant_id: filter.tenant_id,
      app_id: filter.app_id,
      domain: filter.domain,
      use_case: filter.use_case,
      source_id: filter.source_id,
      dataset: filter.dataset,
      severity: filter.severity,
      status: filter.status,
      limit: filter.limit ?? 50,
    }),
  getAlert: (alertId: string) =>
    get<AlertResponse>(`/v1/alerts/${encodeURIComponent(alertId)}`),
  listInsights: (filter: InsightFilter = {}) =>
    get<InsightsResponse>('/v1/insights', {
      tenant_id: filter.tenant_id,
      app_id: filter.app_id,
      domain: filter.domain,
      use_case: filter.use_case,
      source_id: filter.source_id,
      dataset: filter.dataset,
      insight_type: filter.insight_type,
      status: filter.status,
      limit: filter.limit ?? 50,
    }),
  getInsight: (insightId: string) =>
    get<InsightResponse>(`/v1/insights/${encodeURIComponent(insightId)}`),
  // Replay job control plane (G058/G059). Creating a job only queues it; the
  // replay-worker runs separately, so the UI must read status/result from
  // subsequent GETs. tenant_id defaults to tenant-local (dev) when unset.
  listReplayJobs: (filter: ReplayJobFilter = {}) =>
    get<ReplayJobsResponse>('/v1/replay/jobs', {
      tenant_id: filter.tenant_id ?? 'tenant-local',
      source_id: filter.source_id || undefined,
      dataset: filter.dataset || undefined,
      source_kind: filter.source_kind || undefined,
      status: filter.status || undefined,
      limit: filter.limit ?? 50,
    }),
  getReplayJob: (replayJobId: string) =>
    get<ReplayJobResponse>(`/v1/replay/jobs/${encodeURIComponent(replayJobId)}`),
  createReplayJob: (body: ReplayJobCreateRequest) =>
    post<ReplayJobResponse>('/v1/replay/jobs', body),
  // Cancel mirrors the alert/insight lifecycle mutations: under auth the gateway
  // derives the actor from the token (lifecycleActor reads the principal first),
  // so the operator-local placeholder header is only sent in auth-disabled dev.
  cancelReplayJob: (replayJobId: string, body: ReplayJobCancelRequest = {}) =>
    post<ReplayJobResponse>(
      `/v1/replay/jobs/${encodeURIComponent(replayJobId)}/cancel`,
      { reason: body.reason, note: body.note },
      authConfig.authEnabled ? undefined : { 'X-SignalOps-Actor': 'operator-local' },
    ),
  // G064 replay operations observability: worker health, job counts, latest jobs.
  // workers are not tenant-scoped, but job_counts/latest_jobs are; tenant_id is
  // still sent per the backend contract. limit bounds the workers list.
  getReplayStatus: (filter: { tenant_id?: string; limit?: number } = {}) =>
    get<ReplayOperationsStatusResponse>('/v1/replay/status', {
      tenant_id: filter.tenant_id,
      limit: filter.limit,
    }),
  // G066 static app profiles (console + marketops). Same authenticated path.
  getAppProfiles: () => get<AppProfilesResponse>('/v1/app-profiles'),
  // G071 MarketOps asset universe (read-only). tenant_id is a path segment;
  // active_only is serialized as the string the backend parses ("false" disables it).
  listMarketOpsAssets: (filter: MarketOpsAssetFilter = {}) =>
    get<MarketOpsAssetsResponse>(`/v1/tenants/${encodeURIComponent(filter.tenant_id ?? "tenant-local")}/marketops/assets`, { universe_group: filter.universe_group || "top50_megacap", active_only: filter.active_only === false ? "false" : "true", limit: filter.limit ?? 50 }),
  validateMarketOpsWatchlistTicker: (tenantId: string, ticker: string) => get<{validation: MarketOpsTickerValidation}>(`/v1/tenants/${encodeURIComponent(tenantId)}/marketops/assets/validate`, { ticker }, "no-store"),
  onboardMarketOpsWatchlistAsset: (tenantId: string, body: MarketOpsAssetOnboardRequest) => post<{asset: MarketOpsAsset; backfill_job?: MarketOpsAssetBackfillJob}>(`/v1/tenants/${encodeURIComponent(tenantId)}/marketops/assets/onboard`, body),
  updateMarketOpsAssetDisplayName: (tenantId: string, ticker: string, body: MarketOpsAssetDisplayNameRequest) => patch<{asset: MarketOpsAsset}>(`/v1/tenants/${encodeURIComponent(tenantId)}/marketops/assets/${encodeURIComponent(ticker)}/display-name`, body),
  updateMarketOpsAssetDisplaySector: (tenantId: string, ticker: string, body: MarketOpsAssetDisplaySectorRequest) => patch<{asset: MarketOpsAsset}>(`/v1/tenants/${encodeURIComponent(tenantId)}/marketops/assets/${encodeURIComponent(ticker)}/display-sector`, body),
  listMarketOpsAssetBackfillJobs: (tenantId: string, symbol?: string) => get<MarketOpsAssetBackfillJobsResponse>(`/v1/tenants/${encodeURIComponent(tenantId)}/marketops/assets/backfill-jobs`, symbol ? { symbol } : {}),
  createMarketOpsAssetBackfillJob: (tenantId: string, symbol: string, body: MarketOpsAssetBackfillCreateRequest) => post<{backfill_job: MarketOpsAssetBackfillJob}>(`/v1/tenants/${encodeURIComponent(tenantId)}/marketops/assets/${encodeURIComponent(symbol)}/backfill-jobs`, body),
  getMarketOpsAssetQuotes: (tenantId: string, universeGroup = "top50_megacap") =>
    get<MarketOpsAssetQuotesResponse>(`/v1/tenants/${encodeURIComponent(tenantId)}/marketops/assets/quotes`, { universe_group: universeGroup }),
  getMarketOpsIntradayConditions: (tenantId: string, universeGroup = "top50_megacap", symbol?: string) =>
    get<MarketOpsIntradayConditionsResponse>("/v1/tenants/" + encodeURIComponent(tenantId) + "/marketops/assets/" + (symbol ? encodeURIComponent(symbol) + "/" : "") + "intraday-conditions", { universe_group: universeGroup }, 'no-store'),
  // G128 MarketOps asset options intelligence (read-only). Persisted coverage,
  // derived distribution snapshots, and chain rows for one asset. tenant_id and
  // symbol are path segments (the gateway upper-cases symbol). window defaults to
  // 10_trade_days and distribution limit to 10; chain limit defaults to 500
  // (gateway clamps to 200). trade_date is date-only (YYYY-MM-DD). This surface
  // performs no ingestion and never calls live-preview (which stays 501).
  getMarketOpsQuantitativeSeries: (tenantId: string, symbol: string, window: string) => get<MarketOpsQuantitativeSeriesResponse>(`/v1/tenants/${encodeURIComponent(tenantId)}/marketops/assets/${encodeURIComponent(symbol)}/quantitative-series`, { window }),
  getMarketOpsAssetAlgorithmObservations: (tenantId: string, symbol: string) => get<MarketOpsAssetAlgorithmObservationsResponse>(`/v1/tenants/${encodeURIComponent(tenantId)}/marketops/assets/${encodeURIComponent(symbol)}/algorithm-observations`),
  getMarketOpsRiskRewardSummaries: (tenantId: string, universeGroup = "top50_megacap") => get<MarketOpsRiskRewardSummariesResponse>(`/v1/tenants/${encodeURIComponent(tenantId)}/marketops/assets/risk-reward`, { universe_group: universeGroup }, "no-store"),
  getMarketOpsSignalOverview: (tenantId: string, universeGroup = "all_active", window: import("../types").MarketOpsSignalOverviewWindow = "60_trade_days") => get<MarketOpsSignalOverviewResponse>(`/v1/tenants/${encodeURIComponent(tenantId)}/marketops/assets/signal-overview`, { universe_group: universeGroup, window }, "no-store"),
  getMarketOpsOptionsCoverage: (tenantId: string, symbol: string) =>
    get<MarketOpsOptionsCoverageResponse>(
      `/v1/tenants/${encodeURIComponent(tenantId)}/marketops/assets/${encodeURIComponent(symbol)}/options/coverage`,
    ),
  listMarketOpsOptionsDistributions: (tenantId: string, symbol: string, filter: MarketOpsOptionsDistributionFilter = {}) =>
    get<MarketOpsOptionsDistributionsResponse>(
      `/v1/tenants/${encodeURIComponent(tenantId)}/marketops/assets/${encodeURIComponent(symbol)}/options/distribution`,
      {
        window: filter.window || '10_trade_days',
        limit: filter.limit ?? 10,
      },
    ),
  listMarketOpsOptionsChain: (tenantId: string, symbol: string, filter: MarketOpsOptionsChainFilter = {}) =>
    get<MarketOpsOptionsChainResponse>(
      `/v1/tenants/${encodeURIComponent(tenantId)}/marketops/assets/${encodeURIComponent(symbol)}/options/chain`,
      {
        trade_date: filter.trade_date || undefined,
        contract_type: filter.contract_type || undefined,
        limit: filter.limit ?? 500,
      },
    ),
  // G139 MarketOps Opportunities workbench (read-only). Opportunity list/detail
  // plus supporting linked-record reads (hypothesis-evaluations, hypotheses,
  // evidence, market-state lineage). research_only / eligible / triggered /
  // invalidated serialize as "true"/"false" only when set. session_* are
  // date-only (YYYY-MM-DD). limit defaults to 50 (gateway max 200). This surface
  // performs no review, trade, materialization, or build mutation.
  listMarketOpsOpportunities: (filter: MarketOpsOpportunityFilter = {}) =>
    get<MarketOpsOpportunitiesResponse>('/v1/marketops/opportunities', {
      tenant_id: filter.tenant_id,
      app_id: filter.app_id,
      opportunity_id: filter.opportunity_id,
      asset_id: filter.asset_id,
      symbol: filter.symbol,
      direction: filter.direction || undefined,
      horizon: filter.horizon || undefined,
      lifecycle_status: filter.lifecycle_status || undefined,
      research_only: typeof filter.research_only === 'boolean' ? (filter.research_only ? 'true' : 'false') : undefined,
      session_start: filter.session_start || undefined,
      session_end: filter.session_end || undefined,
      limit: filter.limit ?? 50,
    }),
  getMarketOpsOpportunity: (opportunityId: string, tenantId: string) =>
    get<MarketOpsOpportunityResponse>(
      `/v1/marketops/opportunities/${encodeURIComponent(opportunityId)}`,
      { tenant_id: tenantId },
    ),
  listMarketOpsHypothesisEvaluations: (filter: MarketOpsHypothesisEvaluationFilter = {}) =>
    get<MarketOpsHypothesisEvaluationsResponse>('/v1/marketops/hypothesis-evaluations', {
      tenant_id: filter.tenant_id,
      app_id: filter.app_id,
      hypothesis_key: filter.hypothesis_key || undefined,
      hypothesis_version: filter.hypothesis_version || undefined,
      market_state_id: filter.market_state_id || undefined,
      asset_id: filter.asset_id || undefined,
      symbol: filter.symbol || undefined,
      eligible: typeof filter.eligible === 'boolean' ? (filter.eligible ? 'true' : 'false') : undefined,
      triggered: typeof filter.triggered === 'boolean' ? (filter.triggered ? 'true' : 'false') : undefined,
      invalidated: typeof filter.invalidated === 'boolean' ? (filter.invalidated ? 'true' : 'false') : undefined,
      session_start: filter.session_start || undefined,
      session_end: filter.session_end || undefined,
      limit: filter.limit ?? 50,
    }),
  listMarketOpsAlgorithmAdjudications: (filter: MarketOpsAlgorithmAdjudicationFilter = {}) =>
    get<MarketOpsAlgorithmAdjudicationsResponse>("/v1/marketops/algorithm-adjudications", { tenant_id: filter.tenant_id, symbol: filter.symbol, hypothesis_evaluation_id: filter.hypothesis_evaluation_id, correlation_id: filter.correlation_id, limit: filter.limit ?? 50 }),
  getMarketOpsHypothesis: (key: string, version: string, tenantId: string) =>
    get<MarketOpsHypothesisResponse>(
      `/v1/marketops/hypotheses/${encodeURIComponent(key)}/${encodeURIComponent(version)}`,
      { tenant_id: tenantId },
    ),
  getMarketOpsEvidence: (evidenceId: string) =>
    get<MarketOpsEvidenceResponse>(`/v1/marketops/evidence/${encodeURIComponent(evidenceId)}`),
  getMarketOpsMarketStateLineage: (marketStateId: string) =>
    get<MarketOpsMarketStateLineageResponse>(
      `/v1/marketops/states/${encodeURIComponent(marketStateId)}/lineage`,
    ),
  // G147 Market State analyst experience (read-only composition over G136-G146
  // ledgers). States, feature definitions/observations, transitions, evidence +
  // hypothesis lists, outcomes, and opportunity dispositions. Session_* are
  // date-only; observation `dimensions` is a JSON string; nullable filters
  // serialize as true/false only when set. No provider/state/evaluation/promotion
  // mutation. The disposition POST sends no actor (the gateway derives it).
  listMarketOpsStates: (filter: MarketOpsMarketStateFilter = {}) =>
    get<MarketOpsMarketStatesResponse>('/v1/marketops/states', {
      tenant_id: filter.tenant_id,
      app_id: filter.app_id,
      asset_id: filter.asset_id,
      symbol: filter.symbol || undefined,
      state_schema_version: filter.state_schema_version || undefined,
      quality_state: filter.quality_state || undefined,
      session_start: filter.session_start || undefined,
      session_end: filter.session_end || undefined,
      limit: filter.limit ?? 50,
    }),
  getMarketOpsState: (marketStateId: string) =>
    get<MarketOpsMarketStateResponse>(`/v1/marketops/states/${encodeURIComponent(marketStateId)}`),
  listMarketOpsFeatureDefinitions: (filter: MarketOpsFeatureDefinitionFilter = {}) =>
    get<MarketOpsFeatureDefinitionsResponse>('/v1/marketops/features/definitions', {
      tenant_id: filter.tenant_id,
      feature_key: filter.feature_key || undefined,
      feature_version: filter.feature_version || undefined,
      domain: filter.domain || undefined,
      status: filter.status || undefined,
      limit: filter.limit ?? 50,
    }),
  listMarketOpsFeatureObservations: (filter: MarketOpsFeatureObservationFilter = {}) =>
    get<MarketOpsFeatureObservationsResponse>('/v1/marketops/features/observations', {
      tenant_id: filter.tenant_id,
      app_id: filter.app_id,
      asset_id: filter.asset_id,
      symbol: filter.symbol || undefined,
      feature_key: filter.feature_key || undefined,
      feature_version: filter.feature_version || undefined,
      domain: filter.domain || undefined,
      quality_state: filter.quality_state || undefined,
      dimensions: filter.dimensions || undefined,
      session_start: filter.session_start || undefined,
      session_end: filter.session_end || undefined,
      limit: filter.limit ?? 50,
    }),
  listMarketOpsStateTransitions: (filter: MarketOpsStateTransitionFilter = {}) =>
    get<MarketOpsStateTransitionsResponse>('/v1/marketops/transitions', {
      tenant_id: filter.tenant_id,
      app_id: filter.app_id,
      asset_id: filter.asset_id,
      symbol: filter.symbol || undefined,
      current_state_id: filter.current_state_id || undefined,
      feature_key: filter.feature_key || undefined,
      feature_version: filter.feature_version || undefined,
      transition_type: filter.transition_type || undefined,
      quality_state: filter.quality_state || undefined,
      session_start: filter.session_start || undefined,
      session_end: filter.session_end || undefined,
      limit: filter.limit ?? 50,
    }),
  listMarketOpsEvidence: (filter: MarketOpsEvidenceFilter = {}) =>
    get<MarketOpsEvidencesResponse>('/v1/marketops/evidence', {
      tenant_id: filter.tenant_id,
      app_id: filter.app_id,
      asset_id: filter.asset_id,
      symbol: filter.symbol || undefined,
      evidence_type: filter.evidence_type || undefined,
      evidence_version: filter.evidence_version || undefined,
      domain: filter.domain || undefined,
      direction: filter.direction || undefined,
      session_start: filter.session_start || undefined,
      session_end: filter.session_end || undefined,
      limit: filter.limit ?? 50,
    }),
  listMarketOpsHypotheses: (filter: MarketOpsHypothesisListFilter = {}) =>
    get<MarketOpsHypothesesResponse>('/v1/marketops/hypotheses', {
      tenant_id: filter.tenant_id,
      hypothesis_key: filter.hypothesis_key || undefined,
      hypothesis_version: filter.hypothesis_version || undefined,
      domain: filter.domain || undefined,
      lifecycle_status: filter.lifecycle_status || undefined,
      limit: filter.limit ?? 50,
    }),
  listMarketOpsOutcomes: (filter: MarketOpsOutcomeFilter = {}) =>
    get<MarketOpsOutcomesResponse>('/v1/marketops/outcomes', {
      tenant_id: filter.tenant_id,
      app_id: filter.app_id,
      source_type: filter.source_type || undefined,
      source_id: filter.source_id || undefined,
      hypothesis_key: filter.hypothesis_key || undefined,
      hypothesis_version: filter.hypothesis_version || undefined,
      symbol: filter.symbol || undefined,
      direction: filter.direction || undefined,
      outcome_status: filter.outcome_status || undefined,
      horizon_sessions: filter.horizon_sessions,
      session_start: filter.session_start || undefined,
      session_end: filter.session_end || undefined,
      limit: filter.limit ?? 50,
    }),
  getMarketOpsOutcome: (outcomeId: string, tenantId: string) =>
    get<MarketOpsOutcomeResponse>(`/v1/marketops/outcomes/${encodeURIComponent(outcomeId)}`, {
      tenant_id: tenantId,
    }),
  listMarketOpsOpportunityDispositions: (opportunityId: string, filter: MarketOpsOpportunityDispositionFilter = {}) =>
    get<MarketOpsOpportunityDispositionsResponse>(
      `/v1/marketops/opportunities/${encodeURIComponent(opportunityId)}/dispositions`,
      {
        tenant_id: filter.tenant_id,
        disposition: filter.disposition || undefined,
        limit: filter.limit ?? 50,
      },
    ),
  createMarketOpsOpportunityDisposition: (
    opportunityId: string,
    request: MarketOpsOpportunityDispositionRequest,
  ) =>
    post<MarketOpsOpportunityDispositionResponse>(
      `/v1/marketops/opportunities/${encodeURIComponent(opportunityId)}/dispositions`,
      request,
    ),
  // G148-C MarketOps intelligence readiness aggregate (read-only). ONE request
  // serves the whole view; never fan out per-symbol. tenant_id is required by the
  // gateway (defaults to tenant-local). symbols is a CSV string (the gateway
  // uppercases + sorts); latest_session_date is date-only; limit defaults to 50
  // (gateway max 200). No mutation/provider/Ask/graph/proposal controls.
  getMarketOpsIntelligenceReadiness: (filter: MarketOpsIntelligenceReadinessFilter = {}) =>
    get<MarketOpsIntelligenceReadinessResponse>('/v1/marketops/intelligence/readiness', {
      tenant_id: filter.tenant_id ?? 'tenant-local',
      universe_group: filter.universe_group || undefined,
      symbols: filter.symbols || undefined,
      latest_session_date: filter.latest_session_date || undefined,
      rollout_status: filter.rollout_status || undefined,
      limit: filter.limit ?? 50,
    }),
  listMarketOpsDSMArtifacts: (filter: MarketOpsDSMArtifactFilter = {}) =>
    get<MarketOpsDSMArtifactsResponse>('/v1/marketops/dsm/artifacts', {
      tenant_id: filter.tenant_id,
      app_id: filter.app_id,
      domain: filter.domain,
      use_case: filter.use_case,
      signal_type: filter.signal_type,
      severity: filter.severity,
      subject_symbol: filter.subject_symbol,
      limit: filter.limit ?? 50,
    }),
  getMarketOpsDSMArtifact: (artifactId: string) =>
    get<MarketOpsDSMArtifactResponse>(`/v1/marketops/dsm/artifacts/${encodeURIComponent(artifactId)}`),
  // G079/G080 first-class MarketOps DSM graph proposals. The ledger mirrors
  // the artifact API for list/detail reads. G080 uses the existing decision
  // endpoint for operator review metadata only; it still performs no graph DB writes.
  listMarketOpsDSMGraphProposals: (filter: MarketOpsDSMGraphProposalFilter = {}) =>
    get<MarketOpsDSMGraphProposalsResponse>('/v1/marketops/dsm/graph-proposals', {
      tenant_id: filter.tenant_id,
      app_id: filter.app_id,
      domain: filter.domain,
      use_case: filter.use_case,
      artifact_id: filter.artifact_id,
      signal_id: filter.signal_id,
      signal_type: filter.signal_type,
      subject_symbol: filter.subject_symbol,
      candidate_type: filter.candidate_type,
      status: filter.status,
      limit: filter.limit ?? 50,
    }),
  getMarketOpsDSMGraphProposal: (proposalId: string) =>
    get<MarketOpsDSMGraphProposalResponse>(
      `/v1/marketops/dsm/graph-proposals/${encodeURIComponent(proposalId)}`,
    ),
  mutateMarketOpsDSMGraphProposalDecision: ({ proposalId, status, note }: MarketOpsDSMGraphProposalDecisionOptions) =>
    post<MarketOpsDSMGraphProposalResponse>(
      `/v1/marketops/dsm/graph-proposals/${encodeURIComponent(proposalId)}/decision`,
      { status, note },
      authConfig.authEnabled ? undefined : { 'X-SignalOps-Actor': 'operator-local' },
    ),
  mutateAlertLifecycle: ({ alertId, action, note, reason }: AlertLifecycleMutationOptions) =>
    post<AlertResponse>(
      `/v1/alerts/${encodeURIComponent(alertId)}/${action}`,
      { note, reason },
      // When auth is enabled the backend derives the actor from the token; only send the
      // local-development placeholder header when auth is disabled.
      authConfig.authEnabled ? undefined : { 'X-SignalOps-Actor': 'operator-local' },
    ),
  mutateInsightLifecycle: ({ insightId, action, note, reason }: InsightLifecycleMutationOptions) =>
    post<InsightResponse>(
      `/v1/insights/${encodeURIComponent(insightId)}/${action}`,
      { note, reason },
      authConfig.authEnabled ? undefined : { 'X-SignalOps-Actor': 'operator-local' },
    ),
  // G081 MarketOps back-test workspace (isolated experimental runs). The create
  // endpoint is synchronous: the runner completes (or fails) before 201 returns,
  // so no job queue or polling is involved. tenant_id defaults to tenant-local
  // and detector_id to the DSM taxonomy detector, matching the spec defaults.
  listMarketOpsBacktests: (filter: MarketOpsBacktestRunFilter = {}) =>
    get<MarketOpsBacktestRunsResponse>('/v1/marketops/backtests', {
      tenant_id: filter.tenant_id ?? 'tenant-local',
      app_id: filter.app_id || undefined,
      domain: filter.domain || undefined,
      use_case: filter.use_case || undefined,
      source_id: filter.source_id || undefined,
      dataset: filter.dataset || undefined,
      detector_id: filter.detector_id ?? 'marketops.dsm.taxonomy_v1',
      status: filter.status || undefined,
      limit: filter.limit ?? 50,
    }),
  // tenant_id is accepted by the gateway on the detail path but currently ignored
  // (lookup is by run_id); send it anyway for contract consistency.
  getMarketOpsBacktest: (runId: string, tenantId: string = 'tenant-local') =>
    get<MarketOpsBacktestRunResponse>(
      `/v1/marketops/backtests/${encodeURIComponent(runId)}`,
      { tenant_id: tenantId },
    ),
  createMarketOpsBacktest: (body: MarketOpsBacktestCreateRequest) =>
    post<MarketOpsBacktestCreateResponse>('/v1/marketops/backtests', body),
  listMarketOpsBacktestSignals: (runId: string, filter: MarketOpsBacktestSignalFilter = {}) =>
    get<MarketOpsBacktestSignalsResponse>(
      `/v1/marketops/backtests/${encodeURIComponent(runId)}/signals`,
      {
        tenant_id: filter.tenant_id ?? 'tenant-local',
        signal_type: filter.signal_type || undefined,
        limit: filter.limit ?? 50,
      },
    ),
  listMarketOpsBacktestGraphProposals: (runId: string, filter: MarketOpsBacktestGraphProposalFilter = {}) =>
    get<MarketOpsBacktestGraphProposalsResponse>(
      `/v1/marketops/backtests/${encodeURIComponent(runId)}/graph-proposals`,
      {
        tenant_id: filter.tenant_id ?? 'tenant-local',
        signal_type: filter.signal_type || undefined,
        subject_symbol: filter.subject_symbol || undefined,
        candidate_type: filter.candidate_type || undefined,
        recommendation: filter.recommendation || undefined,
        limit: filter.limit ?? 50,
      },
    ),
  listMarketOpsBacktestCalibrationSummaries: (filter: MarketOpsBacktestCalibrationSummaryFilter = {}) =>
    get<MarketOpsBacktestCalibrationSummariesResponse>('/v1/marketops/backtest-calibration-summaries', {
      tenant_id: filter.tenant_id ?? 'tenant-local',
      app_id: filter.app_id || undefined,
      domain: filter.domain || undefined,
      use_case: filter.use_case || undefined,
      source_id: filter.source_id || undefined,
      dataset: filter.dataset || undefined,
      detector_id: filter.detector_id ?? 'marketops.dsm.taxonomy_v1',
      limit: filter.limit ?? 25,
    }),
  getMarketOpsBacktestCalibrationSummary: (summaryId: string) =>
    get<MarketOpsBacktestCalibrationSummaryResponse>(
      `/v1/marketops/backtest-calibration-summaries/${encodeURIComponent(summaryId)}`,
    ),
  createMarketOpsBacktestCalibrationSummary: (body: MarketOpsBacktestCalibrationSummaryCreateRequest) =>
    post<MarketOpsBacktestCalibrationSummaryResponse>('/v1/marketops/backtest-calibration-summaries', body),
  // G083 persisted calibration baselines + stored comparisons. Like the G082
  // summary endpoint, these are plain same-origin POSTs; the gateway derives
  // the actor via replayActor (header -> body -> operator-local), so no actor
  // header is sent — matching the G082 create path. Filters mirror the backend
  // list query params; defaults match the spec (tenant-local, taxonomy detector).
  listMarketOpsBacktestCalibrationBaselines: (filter: MarketOpsBacktestCalibrationBaselineFilter = {}) =>
    get<MarketOpsBacktestCalibrationBaselinesResponse>('/v1/marketops/backtest-calibration-baselines', {
      tenant_id: filter.tenant_id ?? 'tenant-local',
      app_id: filter.app_id || undefined,
      domain: filter.domain || undefined,
      use_case: filter.use_case || undefined,
      detector_id: filter.detector_id ?? 'marketops.dsm.taxonomy_v1',
      dataset: filter.dataset || undefined,
      status: filter.status || undefined,
      limit: filter.limit ?? 50,
    }),
  getMarketOpsBacktestCalibrationBaseline: (baselineId: string) =>
    get<MarketOpsBacktestCalibrationBaselineResponse>(
      `/v1/marketops/backtest-calibration-baselines/${encodeURIComponent(baselineId)}`,
    ),
  createMarketOpsBacktestCalibrationBaseline: (body: MarketOpsBacktestCalibrationBaselineCreateRequest) =>
    post<MarketOpsBacktestCalibrationBaselineResponse>('/v1/marketops/backtest-calibration-baselines', body),
  listMarketOpsBacktestCalibrationComparisons: (filter: MarketOpsBacktestCalibrationComparisonFilter = {}) =>
    get<MarketOpsBacktestCalibrationComparisonsResponse>('/v1/marketops/backtest-calibration-comparisons', {
      tenant_id: filter.tenant_id ?? 'tenant-local',
      baseline_id: filter.baseline_id || undefined,
      detector_id: filter.detector_id || undefined,
      dataset: filter.dataset || undefined,
      recommendation: filter.recommendation || undefined,
      limit: filter.limit ?? 50,
    }),
  getMarketOpsBacktestCalibrationComparison: (comparisonId: string) =>
    get<MarketOpsBacktestCalibrationComparisonResponse>(
      `/v1/marketops/backtest-calibration-comparisons/${encodeURIComponent(comparisonId)}`,
    ),
  createMarketOpsBacktestCalibrationComparison: (body: MarketOpsBacktestCalibrationComparisonCreateRequest) =>
    post<MarketOpsBacktestCalibrationComparisonResponse>('/v1/marketops/backtest-calibration-comparisons', body),
  // G085 label-aware back-test evaluations. Scores a run against synchronized
  // G084 graph-proposal-decision labels. Like G082/G083 these are plain
  // same-origin reads + a create; the gateway derives the actor (and overrides
  // label_source server-side), so no actor header or label_source is sent.
  // Filters mirror the backend list query params; defaults match the spec
  // (tenant-local, limit 50).
  listMarketOpsBacktestEvaluations: (filter: MarketOpsBacktestEvaluationFilter = {}) =>
    get<MarketOpsBacktestEvaluationsResponse>('/v1/marketops/backtest-evaluations', {
      tenant_id: filter.tenant_id ?? 'tenant-local',
      app_id: filter.app_id || undefined,
      domain: filter.domain || undefined,
      use_case: filter.use_case || undefined,
      run_id: filter.run_id || undefined,
      detector_id: filter.detector_id || undefined,
      dataset: filter.dataset || undefined,
      recommendation: filter.recommendation || undefined,
      limit: filter.limit ?? 50,
    }),
  getMarketOpsBacktestEvaluation: (evaluationId: string) =>
    get<MarketOpsBacktestEvaluationResponse>(
      `/v1/marketops/backtest-evaluations/${encodeURIComponent(evaluationId)}`,
    ),
  createMarketOpsBacktestEvaluation: (body: MarketOpsBacktestEvaluationCreateRequest) =>
    post<MarketOpsBacktestEvaluationResponse>('/v1/marketops/backtest-evaluations', body),
  // G086 promotion review candidates. A candidate packages G083 comparison +
  // optional G085 evaluation evidence into an auditable review record. Like
  // G082/G083/G085 these are plain same-origin reads + writes; the gateway
  // derives the actor via replayActor (header -> body -> operator-local), so no
  // actor header is sent. The decision endpoint mutates only the candidate row
  // (it deploys nothing). Filters mirror the backend list query params.
  listMarketOpsBacktestPromotionCandidates: (filter: MarketOpsBacktestPromotionCandidateFilter = {}) =>
    get<MarketOpsBacktestPromotionCandidatesResponse>('/v1/marketops/backtest-promotion-candidates', {
      tenant_id: filter.tenant_id ?? 'tenant-local',
      app_id: filter.app_id || undefined,
      domain: filter.domain || undefined,
      use_case: filter.use_case || undefined,
      baseline_id: filter.baseline_id || undefined,
      comparison_id: filter.comparison_id || undefined,
      evaluation_id: filter.evaluation_id || undefined,
      run_id: filter.run_id || undefined,
      detector_id: filter.detector_id || undefined,
      dataset: filter.dataset || undefined,
      readiness_status: filter.readiness_status || undefined,
      status: filter.status || undefined,
      limit: filter.limit ?? 50,
    }),
  getMarketOpsBacktestPromotionCandidate: (candidateId: string) =>
    get<MarketOpsBacktestPromotionCandidateResponse>(
      `/v1/marketops/backtest-promotion-candidates/${encodeURIComponent(candidateId)}`,
    ),
  createMarketOpsBacktestPromotionCandidate: (body: MarketOpsBacktestPromotionCandidateCreateRequest) =>
    post<MarketOpsBacktestPromotionCandidateResponse>('/v1/marketops/backtest-promotion-candidates', body),
  decideMarketOpsBacktestPromotionCandidate: (
    candidateId: string,
    body: MarketOpsBacktestPromotionCandidateDecisionRequest,
  ) =>
    post<MarketOpsBacktestPromotionCandidateResponse>(
      `/v1/marketops/backtest-promotion-candidates/${encodeURIComponent(candidateId)}/decision`,
      body,
    ),
  // G088 Syncratic synthesized insights + deterministic context windows. These
  // are read-only review surfaces over /v1/syncratic/*. Same authenticated
  // same-origin pattern as the MarketOps reads; the gateway derives the actor,
  // so materialize sends no actor header. Filters mirror the backend list query
  // params; defaults match the spec (tenant-local, limit 50). This frontend
  // never calls the external Syncratic user facade.
  listSyncraticInsights: (filter: SyncraticInsightFilter = {}) =>
    get<SyncraticInsightsResponse>('/v1/syncratic/insights', {
      tenant_id: filter.tenant_id ?? 'tenant-local',
      app_id: filter.app_id || undefined,
      domain: filter.domain || undefined,
      use_case: filter.use_case || undefined,
      context_window_id: filter.context_window_id || undefined,
      insight_type: filter.insight_type || undefined,
      subject_symbol: filter.subject_symbol || undefined,
      status: filter.status || undefined,
      limit: filter.limit ?? 50,
    }),
  getSyncraticInsight: (insightId: string) =>
    get<SyncraticInsightResponse>(`/v1/syncratic/insights/${encodeURIComponent(insightId)}`),
  listSyncraticContextWindows: (filter: SyncraticContextWindowFilter = {}) =>
    get<SyncraticContextWindowsResponse>('/v1/syncratic/context-windows', {
      tenant_id: filter.tenant_id ?? 'tenant-local',
      app_id: filter.app_id || undefined,
      domain: filter.domain || undefined,
      use_case: filter.use_case || undefined,
      subject_symbol: filter.subject_symbol || undefined,
      context_strategy: filter.context_strategy || undefined,
      status: filter.status || undefined,
      limit: filter.limit ?? 50,
    }),
  getSyncraticContextWindow: (contextWindowId: string) =>
    get<SyncraticContextWindowResponse>(
      `/v1/syncratic/context-windows/${encodeURIComponent(contextWindowId)}`,
    ),
  materializeSyncraticContexts: (request: SyncraticMaterializeRequest) =>
    post<SyncraticMaterializationResponse>('/v1/syncratic/materialize', request),
  // G090 operator-triggered Syncratic Ask enrichment over an existing context
  // window. Same authenticated same-origin pattern; the gateway derives the actor,
  // so no actor header is sent. force=false skips on an unchanged prompt+evidence
  // digest; force=true regenerates. Like materialize, this never calls the external
  // Syncratic user facade from the browser.
  askSyncraticContextWindow: (contextWindowId: string, request: SyncraticAskRequest) =>
    post<SyncraticAskResponse>(
      `/v1/syncratic/context-windows/${encodeURIComponent(contextWindowId)}/ask`,
      request,
    ),
  // G109 algorithm execution visibility (read-only). Same authenticated
  // same-origin GET pattern; tenant_id is required by the gateway (it 400s with
  // invalid_algorithm_filter when absent), so it defaults to tenant-local like
  // the other list endpoints. Flexible JSON fields arrive already-parsed.
  listAlgorithmDefinitions: (filter: AlgorithmDefinitionFilter = { tenant_id: 'tenant-local' }) =>
    get<AlgorithmDefinitionsResponse>('/v1/algorithms/definitions', {
      tenant_id: filter.tenant_id ?? 'tenant-local',
      algorithm_type: filter.algorithm_type || undefined,
      runtime_type: filter.runtime_type || undefined,
      status: filter.status || undefined,
      limit: filter.limit ?? 50,
    }),
  getAlgorithmDefinition: (algorithmId: string, tenantId: string = 'tenant-local') =>
    get<AlgorithmDefinitionResponse>(
      `/v1/algorithms/definitions/${encodeURIComponent(algorithmId)}`,
      { tenant_id: tenantId },
    ),
  listAlgorithmExecutionRequests: (filter: AlgorithmExecutionRequestFilter = { tenant_id: 'tenant-local' }) =>
    get<AlgorithmExecutionRequestsResponse>('/v1/algorithms/execution-requests', {
      tenant_id: filter.tenant_id ?? 'tenant-local',
      algorithm_id: filter.algorithm_id || undefined,
      status: filter.status || undefined,
      correlation_id: filter.correlation_id || undefined,
      limit: filter.limit ?? 50,
    }),
  getAlgorithmExecutionRequest: (executionRequestId: string, tenantId: string = 'tenant-local') =>
    get<AlgorithmExecutionRequestResponse>(
      `/v1/algorithms/execution-requests/${encodeURIComponent(executionRequestId)}`,
      { tenant_id: tenantId },
    ),
  // limit defaults to 10 (the backend's summary top-results cap).
  getAlgorithmExecutionSummary: (executionRequestId: string, tenantId: string = 'tenant-local', limit = 10) =>
    get<AlgorithmExecutionSummaryResponse>(
      `/v1/algorithms/execution-requests/${encodeURIComponent(executionRequestId)}/summary`,
      { tenant_id: tenantId, limit },
    ),
  listAlgorithmResults: (filter: AlgorithmResultFilter = { tenant_id: 'tenant-local' }) =>
    get<AlgorithmResultsResponse>('/v1/algorithms/results', {
      tenant_id: filter.tenant_id ?? 'tenant-local',
      algorithm_id: filter.algorithm_id || undefined,
      execution_request_id: filter.execution_request_id || undefined,
      result_type: filter.result_type || undefined,
      severity: filter.severity || undefined,
      correlation_id: filter.correlation_id || undefined,
      limit: filter.limit ?? 50,
    }),
  getAlgorithmResult: (algorithmResultId: string, tenantId: string = 'tenant-local') =>
    get<AlgorithmResultResponse>(
      `/v1/algorithms/results/${encodeURIComponent(algorithmResultId)}`,
      { tenant_id: tenantId },
    ),
  // G113/G114 algorithm signal proposals review surface (G111/G112 backend).
  // Same authenticated same-origin GET pattern; tenant_id is required by the
  // gateway (it 400s with invalid_algorithm_filter when absent), so it defaults
  // to tenant-local like the other algorithm list endpoints. Default status
  // filter is unset here; the route applies status=proposed per the spec.
  listAlgorithmSignalProposals: (filter: AlgorithmSignalProposalFilter = {}) =>
    get<AlgorithmSignalProposalsResponse>('/v1/algorithms/signal-proposals', {
      tenant_id: filter.tenant_id ?? 'tenant-local',
      proposal_source: filter.proposal_source || undefined,
      algorithm_id: filter.algorithm_id || undefined,
      execution_request_id: filter.execution_request_id || undefined,
      algorithm_result_id: filter.algorithm_result_id || undefined,
      hypothesis_evaluation_id: filter.hypothesis_evaluation_id || undefined,
      hypothesis_key: filter.hypothesis_key || undefined,
      status: filter.status || undefined,
      severity: filter.severity || undefined,
      correlation_id: filter.correlation_id || undefined,
      limit: filter.limit ?? 50,
    }),
  getAlgorithmSignalProposal: (proposalId: string, tenantId: string = 'tenant-local') =>
    get<AlgorithmSignalProposalResponse>(
      `/v1/algorithms/signal-proposals/${encodeURIComponent(proposalId)}`,
      { tenant_id: tenantId },
    ),
  // G115/G116 review-coverage summary. Couples to the same filters as the list
  // (tenant_id required by the gateway, defaults to tenant-local) but NEVER sends
  // limit — the summary endpoint aggregates the whole matched slice.
  getAlgorithmSignalProposalSummary: (filter: AlgorithmSignalProposalFilter = {}) =>
    get<AlgorithmSignalProposalSummaryResponse>('/v1/algorithms/signal-proposals/summary', {
      tenant_id: filter.tenant_id ?? 'tenant-local',
      algorithm_id: filter.algorithm_id || undefined,
      execution_request_id: filter.execution_request_id || undefined,
      algorithm_result_id: filter.algorithm_result_id || undefined,
      status: filter.status || undefined,
      severity: filter.severity || undefined,
      correlation_id: filter.correlation_id || undefined,
    }),
  // G119 algorithm signal materialization preflight (read-only). Couples to the
  // same proposal-list filters as the list/summary (tenant_id required by the
  // gateway, defaults to tenant-local) AND sends limit (the proposal-list limit,
  // so the preflight slice matches the reviewed list). min_reviewed_ratio defaults
  // to 1 and policy_version to materialization_preflight.v1 per the spec; both are
  // sent explicitly so the request is self-describing. This forecasts eligibility
  // only — it materializes no production signal, alert, insight, or graph proposal.
  getAlgorithmSignalMaterializationPreflight: (filter: AlgorithmSignalMaterializationPreflightFilter = {}) =>
    get<AlgorithmSignalMaterializationPreflightResponse>(
      '/v1/algorithms/signal-proposals/materialization-preflight',
      {
        tenant_id: filter.tenant_id ?? 'tenant-local',
        algorithm_id: filter.algorithm_id || undefined,
        execution_request_id: filter.execution_request_id || undefined,
        algorithm_result_id: filter.algorithm_result_id || undefined,
        status: filter.status || undefined,
        severity: filter.severity || undefined,
        correlation_id: filter.correlation_id || undefined,
        limit: filter.limit ?? 50,
        min_reviewed_ratio: filter.min_reviewed_ratio ?? 1,
        policy_version: filter.policy_version ?? 'materialization_preflight.v1',
      },
    ),
  // Decision mutation. The gateway derives the reviewer via replayActor
  // (header -> body -> operator-local), so no actor header is sent — matching
  // the promotion-candidate decision. This only records review metadata; it
  // materializes no production signal, alert, insight, or graph proposal.
  decideAlgorithmSignalProposal: (proposalId: string, request: AlgorithmSignalProposalDecisionRequest) =>
    post<AlgorithmSignalProposalResponse>(
      `/v1/algorithms/signal-proposals/${encodeURIComponent(proposalId)}/decision`,
      request,
    ),
  // G123 single-proposal materialization (G122 backend). The only write control on
  // this surface: materializes one reviewed eligible proposal into a production
  // signal. tenant_id goes in the body (the gateway reads body first, then query);
  // policy_version is the fixed algorithm_materialization.v1 default; requested_by
  // and idempotency_key are derived server-side from the JWT/digest, so — like the
  // sibling decision mutation — no actor header is sent. The gateway requires the
  // signalops:operator/admin role. The POST returns 201/200 with the envelope for
  // succeeded/duplicate/blocked; only not-found/auth/server failures throw.
  materializeAlgorithmSignalProposal: (proposalId: string, request: AlgorithmSignalMaterializationRequest) =>
    post<AlgorithmSignalMaterializationResponse>(
      `/v1/algorithms/signal-proposals/${encodeURIComponent(proposalId)}/materializations`,
      request,
    ),
  // G121 materialization ledger reads. tenant_id is required by the gateway
  // (defaults to tenant-local). The proposal detail scopes this to one proposal;
  // limit defaults to 50 (max 200). Read-only.
  listAlgorithmSignalMaterializations: (filter: AlgorithmSignalMaterializationFilter = {}) =>
    get<AlgorithmSignalMaterializationsResponse>('/v1/algorithms/signal-materializations', {
      tenant_id: filter.tenant_id ?? 'tenant-local',
      proposal_id: filter.proposal_id || undefined,
      algorithm_result_id: filter.algorithm_result_id || undefined,
      execution_request_id: filter.execution_request_id || undefined,
      algorithm_id: filter.algorithm_id || undefined,
      status: filter.status || undefined,
      signal_id: filter.signal_id || undefined,
      limit: filter.limit ?? 50,
    }),
};
