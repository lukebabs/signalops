import type {
  HealthResponse,
  SchedulerRunsResponse,
  SchedulerRunResponse,
  ProviderUsageResponse,
  RawEventsResponse,
  RawEventResponse,
  IdempotencyResponse,
  RawEventFilter,
} from '../types';

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

function buildUrl(path: string, params?: Record<string, string | number | undefined>): string {
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

async function get<T>(path: string, params?: Record<string, string | number | undefined>): Promise<T> {
  const endpoint = buildUrl(path, params);
  let res: Response;
  try {
    res = await fetch(endpoint, { headers: { Accept: 'application/json' } });
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
};
