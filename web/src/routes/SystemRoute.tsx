import { useHealthz, useReadyz, useRuns, useReplayStatus } from '../api/queries';
import { useUi } from '../store/ui';
import { MetricTile } from '../components/MetricTile';
import { RefreshButton } from '../components/RefreshButton';
import { StatusBadge } from '../components/StatusBadge';
import { ErrorState, EmptyState } from '../components/States';
import { isApiError } from '../api/client';
import { formatUtc } from '../lib/format';
import { replayJobCount, worstReplayWorkerHealth, latestReplayWorkerSeenAt } from '../lib/replayStatus';
import { useTenant } from '../auth/session';

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

  const TENANT_ID = useTenant();
  const replayStatus = useReplayStatus({ tenant_id: TENANT_ID, limit: 10 });

  const storageAvailable = probe.isSuccess;
  const storageUnavailable =
    probe.isError && isApiError(probe.error) && probe.error.status === 503;

  const replayStatusOk = replayStatus.data?.replay_status;
  const replayWorkers = replayStatusOk?.workers ?? [];
  const replayWorstHealth = worstReplayWorkerHealth(replayWorkers);
  const replayLastSeen = latestReplayWorkerSeenAt(replayWorkers);

  function refreshAll() {
    healthz.refetch();
    readyz.refetch();
    probe.refetch();
    replayStatus.refetch();
    setLastRefresh(new Date().toISOString());
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h1 className="text-lg font-semibold">System</h1>
        <RefreshButton
          onClick={refreshAll}
          loading={healthz.isFetching || readyz.isFetching || probe.isFetching || replayStatus.isFetching}
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

      <h2 className="text-sm font-semibold text-gray-900">Replay Operations</h2>
      <div className="grid grid-cols-2 gap-2 md:grid-cols-3">
        <MetricTile
          label="Replay Worker"
          value={replayStatus.isError ? 'unreachable' : replayWorstHealth}
        />
        <MetricTile
          label="Replay Queue"
          value={replayJobCount(replayStatusOk, 'queued')}
          hint={replayStatus.isError ? 'status unavailable' : undefined}
        />
        <MetricTile label="Replay Running" value={replayJobCount(replayStatusOk, 'running')} />
        <MetricTile label="Replay Failed" value={replayJobCount(replayStatusOk, 'failed')} />
        <MetricTile label="Replay Last Seen" value={formatUtc(replayLastSeen)} />
      </div>
      {replayStatus.isError ? (
        <ErrorState error={replayStatus.error} />
      ) : replayStatus.isLoading ? (
        <div className="text-sm text-gray-500">Loading replay worker status…</div>
      ) : replayWorkers.length ? (
        <div className="overflow-x-auto rounded border border-gray-200 bg-white">
          <table className="min-w-full divide-y divide-gray-200 text-xs">
            <thead className="bg-gray-50 text-left text-gray-500">
              <tr>
                <th className="px-2 py-1">Worker ID</th>
                <th className="px-2 py-1">Health</th>
                <th className="px-2 py-1">Status</th>
                <th className="px-2 py-1">Last seen</th>
                <th className="px-2 py-1">Last claimed</th>
                <th className="px-2 py-1">Last completed</th>
                <th className="px-2 py-1">Last error</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {replayWorkers.map((w) => (
                <tr key={w.worker_id}>
                  <td className="break-all px-2 py-1 font-mono">{w.worker_id}</td>
                  <td className="px-2 py-1"><StatusBadge status={w.health} /></td>
                  <td className="px-2 py-1"><StatusBadge status={w.status} /></td>
                  <td className="px-2 py-1 text-gray-600">{formatUtc(w.last_seen_at)}</td>
                  <td className="px-2 py-1">
                    {w.last_claimed_replay_job_id ? (
                      <>
                        <code className="break-all font-mono">{w.last_claimed_replay_job_id}</code>
                        <div className="text-gray-500">{formatUtc(w.last_claimed_at)}</div>
                      </>
                    ) : (
                      '—'
                    )}
                  </td>
                  <td className="px-2 py-1">
                    {w.last_completed_replay_job_id ? (
                      <>
                        <code className="break-all font-mono">{w.last_completed_replay_job_id}</code>
                        <div className="text-gray-500">{formatUtc(w.last_completed_at)}</div>
                      </>
                    ) : (
                      '—'
                    )}
                  </td>
                  <td className="px-2 py-1 text-red-700">
                    {w.last_error_message ? (
                      <>
                        <div className="break-all">{w.last_error_message}</div>
                        <div className="text-gray-500">{formatUtc(w.last_error_at)}</div>
                      </>
                    ) : (
                      '—'
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <EmptyState message="No replay worker heartbeat recorded." />
      )}
    </div>
  );
}
