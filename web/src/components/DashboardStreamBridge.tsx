import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '../api/queries';
import {
  REST_FALLBACK_INTERVAL_MS,
  refreshDashboardViaRest,
  subscribeDashboardStream,
  streamMode,
  type DashboardStreamEvent,
} from '../api/stream';
import { useUi } from '../store/ui';
import type { HealthResponse } from '../types';

export function DashboardStreamBridge() {
  const queryClient = useQueryClient();
  const setStreamConnected = useUi((s) => s.setStreamConnected);
  const recordStreamEvent = useUi((s) => s.recordStreamEvent);
  const setStreamError = useUi((s) => s.setStreamError);
  const setStreamMode = useUi((s) => s.setStreamMode);

  useEffect(() => {
    const mode = streamMode();
    setStreamMode(mode);

    // Auth-enabled fallback: native EventSource cannot carry the Bearer token, so SSE is
    // intentionally not opened. Keep the dashboard fresh by invalidating the operational
    // query prefixes on a modest interval. No stream error is set.
    if (mode === 'rest_fallback') {
      const handle = setInterval(() => refreshDashboardViaRest(queryClient), REST_FALLBACK_INTERVAL_MS);
      refreshDashboardViaRest(queryClient);
      return () => clearInterval(handle);
    }

    // Auth-disabled: native SSE as before.
    const subscription = subscribeDashboardStream({
      onOpen: () => setStreamConnected(true),
      onEvent: (event) => {
        recordStreamEvent();
        applyStreamEvent(queryClient, event);
      },
      onError: () => setStreamError('dashboard stream disconnected'),
    });

    return () => {
      subscription.close();
      setStreamConnected(false);
    };
  }, [queryClient, recordStreamEvent, setStreamConnected, setStreamError, setStreamMode]);

  return null;
}

function applyStreamEvent(
  queryClient: ReturnType<typeof useQueryClient>,
  event: DashboardStreamEvent,
) {
  switch (event.type) {
    case 'health':
      queryClient.setQueryData(queryKeys.healthz, event.data as HealthResponse);
      return;
    case 'scheduler_run':
      queryClient.invalidateQueries({ queryKey: ['runs'] });
      if (event.id) queryClient.invalidateQueries({ queryKey: queryKeys.run(event.id) });
      return;
    case 'raw_event':
      queryClient.invalidateQueries({ queryKey: ['raw-events'] });
      if (event.id) queryClient.invalidateQueries({ queryKey: queryKeys.rawEvent(event.id) });
      return;
    case 'provider_usage':
      queryClient.invalidateQueries({ queryKey: ['provider-usage'] });
      return;
    case 'error':
      queryClient.invalidateQueries({ queryKey: queryKeys.healthz });
      return;
    case 'heartbeat':
    default:
      return;
  }
}
