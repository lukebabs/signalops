import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  parseDashboardStreamData,
  subscribeDashboardStream,
  toDashboardStreamEvent,
  type DashboardStreamEventType,
} from './stream';

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

  it('subscribes with default channels and closes the EventSource', () => {
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
