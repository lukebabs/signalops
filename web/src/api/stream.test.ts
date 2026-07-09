import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  parseDashboardStreamData,
  refreshDashboardViaRest,
  REST_FALLBACK_PREFIXES,
  streamMode,
  subscribeDashboardStream,
  toDashboardStreamEvent,
  type DashboardStreamEventType,
} from './stream';

// Hoisted mutable auth state so the mocked config module reads the live value per test.
const state = vi.hoisted(() => ({ authEnabled: false }));

vi.mock('../auth/config', () => ({
  authConfig: {
    get authEnabled() {
      return state.authEnabled;
    },
  },
}));

class FakeEventSource {
  static instances: FakeEventSource[] = [];
  onopen: (() => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  readonly listeners = new Map<string, (event: MessageEvent<string>) => void>();
  closed = false;

  constructor(public readonly url: string) {
    FakeEventSource.instances.push(this);
  }

  addEventListener(type: string, handler: EventListener) {
    this.listeners.set(type, handler as (event: MessageEvent<string>) => void);
  }

  close() {
    this.closed = true;
  }

  emit(type: DashboardStreamEventType, data: string, id = '') {
    this.listeners.get(type)?.({ data, lastEventId: id } as MessageEvent<string>);
  }
}

afterEach(() => {
  FakeEventSource.instances = [];
  vi.unstubAllGlobals();
  state.authEnabled = false;
});

describe('dashboard stream client', () => {
  it('parses JSON, empty, and non-JSON SSE payloads', () => {
    expect(parseDashboardStreamData('{"status":"ok"}')).toEqual({ status: 'ok' });
    expect(parseDashboardStreamData('')).toBeNull();
    expect(parseDashboardStreamData('not-json')).toBe('not-json');
  });

  it('converts MessageEvent data into a typed dashboard stream event', () => {
    const event = toDashboardStreamEvent('raw_event', {
      data: '{"event_id":"evt-1"}',
      lastEventId: 'evt-1',
    } as MessageEvent<string>);

    expect(event).toEqual({
      type: 'raw_event',
      id: 'evt-1',
      data: { event_id: 'evt-1' },
    });
  });

  it('subscribes with default channels and closes the EventSource (auth disabled)', () => {
    state.authEnabled = false;
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    vi.stubGlobal('EventSource', FakeEventSource);
    const onOpen = vi.fn();
    const onError = vi.fn();
    const onEvent = vi.fn();

    const subscription = subscribeDashboardStream({ onOpen, onError, onEvent });
    const source = FakeEventSource.instances[0];

    expect(source.url).toBe(
      'http://localhost:5173/v1/streams/dashboard?channels=health%2Cruns%2Craw_events%2Cprovider_usage%2Cheartbeat',
    );
    // No credential is ever placed in the stream URL.
    expect(source.url).not.toMatch(/token|access_token|authorization/i);

    source.onopen?.();
    source.emit('health', '{"status":"ok"}');
    source.onerror?.(new Event('error'));

    expect(onOpen).toHaveBeenCalledOnce();
    expect(onEvent).toHaveBeenCalledWith({
      type: 'health',
      id: undefined,
      data: { status: 'ok' },
    });
    expect(onError).toHaveBeenCalledOnce();

    subscription.close();
    expect(source.closed).toBe(true);
  });
});

describe('dashboard stream auth fallback (G054)', () => {
  it('streamMode() is eventsource when auth disabled and rest_fallback when auth enabled', () => {
    state.authEnabled = false;
    expect(streamMode()).toBe('eventsource');
    state.authEnabled = true;
    expect(streamMode()).toBe('rest_fallback');
  });

  it('does not open native EventSource under auth, never reports an stream error, and closes safely', () => {
    state.authEnabled = true;
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    vi.stubGlobal('EventSource', FakeEventSource);
    const onOpen = vi.fn();
    const onError = vi.fn();
    const onEvent = vi.fn();

    const subscription = subscribeDashboardStream({ onOpen, onError, onEvent });

    // No native EventSource is constructed, so no request (and no token in any URL) is made.
    expect(FakeEventSource.instances).toHaveLength(0);
    expect(onOpen).not.toHaveBeenCalled();
    expect(onError).not.toHaveBeenCalled();
    // The inert subscription is safely closable.
    expect(() => subscription.close()).not.toThrow();
  });

  it('refreshDashboardViaRest invalidates the dashboard prefixes but not healthz/readyz', () => {
    const invalidateQueries = vi.fn();
    refreshDashboardViaRest({ invalidateQueries });

    const invalidated = invalidateQueries.mock.calls.map((c) => c[0].queryKey[0]);
    expect(invalidated).toEqual([...REST_FALLBACK_PREFIXES]);
    // healthz/readyz already poll on their own refetchInterval — they must not be duplicated.
    expect(invalidated).not.toContain('healthz');
    expect(invalidated).not.toContain('readyz');
  });
});
