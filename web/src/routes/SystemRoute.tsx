import { useHealthz, useReadyz, useRuns } from '../api/queries';
import { useUi } from '../store/ui';
import { MetricTile } from '../components/MetricTile';
import { RefreshButton } from '../components/RefreshButton';
import { isApiError } from '../api/client';
import { formatUtc } from '../lib/format';

const BASE_URL =
  (import.meta.env.VITE_SIGNALOPS_API_BASE_URL ?? '').replace(/\/+$/, '') ||
  '(same-origin via dev proxy)';

export function SystemRoute() {
  const healthz = useHealthz();
  const readyz = useReadyz();
  const probe = useRuns(1); // storage availability probe: 200 = available, 503 = unavailable
  const lastRefresh = useUi((s) => s.lastRefresh);
  const lastStreamEventAt = useUi((s) => s.lastStreamEventAt);
  const streamConnected = useUi((s) => s.streamConnected);
  const streamError = useUi((s) => s.streamError);
  const restFallback = useUi((s) => s.streamMode) === 'rest_fallback';
  const setLastRefresh = useUi((s) => s.setLastRefresh);

  const storageAvailable = probe.isSuccess;
  const storageUnavailable =
    probe.isError && isApiError(probe.error) && probe.error.status === 503;

  function refreshAll() {
    healthz.refetch();
    readyz.refetch();
    probe.refetch();
    setLastRefresh(new Date().toISOString());
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h1 className="text-lg font-semibold">System</h1>
        <RefreshButton
          onClick={refreshAll}
          loading={healthz.isFetching || readyz.isFetching || probe.isFetching}
        />
      </div>
      <div className="grid grid-cols-2 gap-2 md:grid-cols-3">
        <MetricTile
          label="Gateway Health (/healthz)"
          value={healthz.data?.status ?? (healthz.isError ? 'unreachable' : '…')}
          hint={healthz.data?.time}
        />
        <MetricTile
          label="Gateway Ready (/readyz)"
          value={readyz.data?.status ?? (readyz.isError ? 'unreachable' : '…')}
          hint={readyz.data?.time}
        />
        <MetricTile
          label="Storage Query"
          value={storageAvailable ? 'available' : storageUnavailable ? 'unavailable' : 'checking'}
          hint={
            storageUnavailable
              ? '503 storage_unavailable — check SIGNALOPS_DATABASE_URL and Postgres'
              : undefined
          }
        />
        <MetricTile label="API Base URL" value={<code className="text-sm">{BASE_URL}</code>} />
        <MetricTile label="Last Refresh" value={formatUtc(lastRefresh ?? undefined)} />
        <MetricTile
          label="Dashboard Stream"
          value={
            restFallback
              ? 'REST refresh'
              : streamConnected
                ? 'connected'
                : streamError
                  ? 'reconnecting'
                  : 'checking'
          }
          hint={restFallback ? 'SSE disabled under auth; REST polling active' : (streamError ?? undefined)}
        />
        <MetricTile label="Last Stream Event" value={formatUtc(lastStreamEventAt ?? undefined)} />
      </div>
    </div>
  );
}
