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
  MarketOpsAssetFilter,
  MarketOpsDSMArtifactsResponse,
  MarketOpsDSMArtifactResponse,
  MarketOpsDSMArtifactFilter,
  MarketOpsDSMGraphProposalsResponse,
  MarketOpsDSMGraphProposalResponse,
  MarketOpsDSMGraphProposalFilter,
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

async function get<T>(path: string, params?: Record<string, string | number | undefined>): Promise<T> {
  const endpoint = buildUrl(path, params);
  let res: Response;
  try {
    res = await fetch(endpoint, { headers: { Accept: 'application/json', ...authHeaders() } });
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
  listCatalogPipelines: (tenantId = 'tenant-local', limit = 50) =>
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
    get<MarketOpsAssetsResponse>(
      `/v1/tenants/${encodeURIComponent(filter.tenant_id ?? 'tenant-local')}/marketops/assets`,
      {
        universe_group: filter.universe_group || 'top50_megacap',
        active_only: filter.active_only === false ? 'false' : 'true',
        limit: filter.limit ?? 50,
      },
    ),
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
  // G079 first-class MarketOps DSM graph proposals (read-only). The ledger
  // mirrors the artifact API: server-side filtering by signal/artifact context,
  // default limit 50, and only defined filter values are serialized. The UI
  // never calls the decision endpoint — that remains backend/operator-only.
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
};
