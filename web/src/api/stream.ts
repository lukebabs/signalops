import { buildUrl } from './client';
import { authConfig } from '../auth/config';

export type DashboardStreamChannel =
  | 'health'
  | 'runs'
  | 'raw_events'
  | 'provider_usage'
  | 'heartbeat';

export type DashboardStreamEventType =
  | 'heartbeat'
  | 'health'
  | 'scheduler_run'
  | 'raw_event'
  | 'provider_usage'
  | 'error';

export interface DashboardStreamEvent<T = unknown> {
  type: DashboardStreamEventType;
  id?: string;
  data: T;
}

export interface DashboardStreamSubscription {
  close: () => void;
}

// Native EventSource cannot set an Authorization header, and putting a Bearer token in
// the stream URL would leak it into logs/history. Under frontend auth we therefore do
// NOT open SSE to the protected /v1/streams/dashboard; the dashboard stays fresh via
// REST polling (see DashboardStreamBridge). Auth-disabled keeps native SSE as-is.
export type DashboardStreamMode = 'eventsource' | 'rest_fallback';

export function streamMode(): DashboardStreamMode {
  return authConfig.authEnabled ? 'rest_fallback' : 'eventsource';
}

// Dashboard query prefixes refreshed on the REST fallback clock. `healthz`/`readyz` are
// intentionally excluded: they already poll on their own refetchInterval.
export const REST_FALLBACK_PREFIXES = [
  'runs',
  'raw-events',
  'provider-usage',
  'catalog-sources',
  'catalog-pipelines',
  'catalog-rules',
  'normalized-events',
  'signals',
  'alerts',
  'insights',
] as const;

// Modest interval to keep dashboard summaries fresh under auth without noisy backend load.
export const REST_FALLBACK_INTERVAL_MS = 15_000;

// Minimal interface so the helper is unit-testable without a real QueryClient.
export interface InvalidateQueries {
  invalidateQueries: (opts: { queryKey: readonly unknown[] }) => void;
}

// Invalidate the dashboard operational prefixes once (the REST fallback "tick"). Called by
// DashboardStreamBridge on a 15s interval when SSE is disabled under auth.
export function refreshDashboardViaRest(qc: InvalidateQueries): void {
  for (const prefix of REST_FALLBACK_PREFIXES) {
    qc.invalidateQueries({ queryKey: [prefix] });
  }
}

interface SubscribeOptions {
  channels?: DashboardStreamChannel[];
  onOpen?: () => void;
  onEvent: (event: DashboardStreamEvent) => void;
  onError: (error: Event) => void;
}

export function subscribeDashboardStream({
  channels = ['health', 'runs', 'raw_events', 'provider_usage', 'heartbeat'],
  onOpen,
  onEvent,
  onError,
}: SubscribeOptions): DashboardStreamSubscription {
  // Auth-enabled: return an inert subscription. No EventSource is constructed, no token
  // is placed in any URL, and onError is never invoked, so the UI does not flag a
  // disconnected stream. Freshness comes from the REST fallback in DashboardStreamBridge.
  if (streamMode() === 'rest_fallback') {
    return { close: () => {} };
  }

  const source = new EventSource(
    buildUrl('/v1/streams/dashboard', { channels: channels.join(',') }),
  );

  source.onopen = () => onOpen?.();
  source.onerror = onError;

  const bind = (type: DashboardStreamEventType) => {
    source.addEventListener(type, (message) => {
      const event = message as MessageEvent<string>;
      onEvent(toDashboardStreamEvent(type, event));
    });
  };

  bind('heartbeat');
  bind('health');
  bind('scheduler_run');
  bind('raw_event');
  bind('provider_usage');
  bind('error');

  return { close: () => source.close() };
}

export function parseDashboardStreamData(data: string): unknown {
  if (!data) return null;
  try {
    return JSON.parse(data);
  } catch {
    return data;
  }
}

export function toDashboardStreamEvent(
  type: DashboardStreamEventType,
  event: MessageEvent<string>,
): DashboardStreamEvent {
  return {
    type,
    id: event.lastEventId || undefined,
    data: parseDashboardStreamData(event.data),
  };
}
