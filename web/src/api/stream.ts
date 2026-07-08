import { buildUrl } from './client';

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
