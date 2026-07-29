import { buildUrl } from './client';
import { authConfig } from '../auth/config';
import { getAccessToken } from '../auth/session';
import type { CyberOpsLiveTrafficSnapshot } from '../types';

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

export type CyberOpsLiveTrafficStatus = "connecting" | "open" | "reconnecting" | "closed";

export interface CyberOpsLiveTrafficSubscription {
  close: () => void;
}

interface CyberOpsLiveTrafficSubscribeOptions {
  tenantId: string;
  onSnapshot: (snapshot: CyberOpsLiveTrafficSnapshot) => void;
  onStatus?: (status: CyberOpsLiveTrafficStatus) => void;
  onError?: (error: Error) => void;
}

// Unlike EventSource, fetch streaming carries the bearer token in an Authorization
// header. This keeps credentials out of the URL while preserving true SSE delivery.
export function subscribeCyberOpsLiveTraffic({ tenantId, onSnapshot, onStatus, onError }: CyberOpsLiveTrafficSubscribeOptions): CyberOpsLiveTrafficSubscription {
  const controller = new AbortController();
  let closed = false;
  let retryMs = 1_000;

  const run = async () => {
    let firstAttempt = true;
    while (!closed) {
      onStatus?.(firstAttempt ? "connecting" : "reconnecting");
      firstAttempt = false;
      try {
        const token = authConfig.authEnabled ? getAccessToken() : null;
        const response = await fetch(buildUrl("/v1/tenants/" + encodeURIComponent(tenantId) + "/cyberops/live-traffic"), {
          cache: "no-store",
          headers: { Accept: "text/event-stream", ...(token ? { Authorization: "Bearer " + token } : {}) },
          signal: controller.signal,
        });
        if (!response.ok) {
          throw new Error("Live CyberOps traffic stream failed (" + response.status + ")");
        }
        if (!response.body) {
          throw new Error("Live CyberOps traffic stream has no response body");
        }
        onStatus?.("open");
        retryMs = 1_000;
        await consumeCyberOpsLiveTrafficStream(response.body, onSnapshot);
        if (!closed) {
          throw new Error("Live CyberOps traffic stream closed");
        }
      } catch (error) {
        if (closed || controller.signal.aborted) {
          break;
        }
        onError?.(error instanceof Error ? error : new Error("Live CyberOps traffic stream failed"));
        await sleep(retryMs, controller.signal);
        retryMs = Math.min(retryMs * 2, 10_000);
      }
    }
    onStatus?.("closed");
  };
  void run();
  return { close: () => { closed = true; controller.abort(); } };
}

export async function consumeCyberOpsLiveTrafficStream(stream: ReadableStream<Uint8Array>, onSnapshot: (snapshot: CyberOpsLiveTrafficSnapshot) => void): Promise<void> {
  const reader = stream.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  try {
    while (true) {
      const result = await reader.read();
      if (result.done) {
        break;
      }
      buffer += decoder.decode(result.value, { stream: true });
      buffer = consumeCyberOpsLiveTrafficFrames(buffer, onSnapshot);
    }
    buffer += decoder.decode();
    consumeCyberOpsLiveTrafficFrames(buffer + "\n\n", onSnapshot);
  } finally {
    reader.releaseLock();
  }
}

export function consumeCyberOpsLiveTrafficFrames(buffer: string, onSnapshot: (snapshot: CyberOpsLiveTrafficSnapshot) => void): string {
  const frames = buffer.split(/\r?\n\r?\n/);
  const remaining = frames.pop() ?? "";
  for (const frame of frames) {
    const event = parseCyberOpsLiveTrafficFrame(frame);
    if (event) {
      onSnapshot(event);
    }
  }
  return remaining;
}

function parseCyberOpsLiveTrafficFrame(frame: string): CyberOpsLiveTrafficSnapshot | null {
  const lines = frame.split(/\r?\n/);
  let event = "message";
  const data: string[] = [];
  for (const line of lines) {
    if (line.startsWith("event:")) {
      event = line.slice("event:".length).trim();
    } else if (line.startsWith("data:")) {
      data.push(line.slice("data:".length).trimStart());
    }
  }
  if ((event !== "snapshot" && event !== "traffic") || data.length === 0) {
    return null;
  }
  try {
    const value = JSON.parse(data.join("\n")) as CyberOpsLiveTrafficSnapshot;
    if (!Array.isArray(value.points) || typeof value.generated_at !== "string") {
      return null;
    }
    return value;
  } catch {
    return null;
  }
}

function sleep(duration: number, signal: AbortSignal): Promise<void> {
  return new Promise((resolve) => {
    const timer = window.setTimeout(resolve, duration);
    signal.addEventListener("abort", () => { window.clearTimeout(timer); resolve(); }, { once: true });
  });
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
