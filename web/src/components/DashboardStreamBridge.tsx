import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '../api/queries';
import { subscribeDashboardStream, type DashboardStreamEvent } from '../api/stream';
import { useUi } from '../store/ui';
import type { HealthResponse } from '../types';

export function DashboardStreamBridge() {
  const queryClient = useQueryClient();
  const setStreamConnected = useUi((s) => s.setStreamConnected);
  const recordStreamEvent = useUi((s) => s.recordStreamEvent);
  const setStreamError = useUi((s) => s.setStreamError);

  useEffect(() => {
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
  }, [queryClient, recordStreamEvent, setStreamConnected, setStreamError]);

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
