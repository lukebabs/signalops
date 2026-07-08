import { useRun, useProviderUsage } from '../api/queries';
import { StatusBadge } from './StatusBadge';
import { JsonViewer } from './JsonViewer';
import { CopyButton } from './CopyButton';
import { MetricTile } from './MetricTile';
import { LoadingState, ErrorState, EmptyState } from './States';
import { formatUtc } from '../lib/format';

export function RunDetail({ runId }: { runId: string | null }) {
  const run = useRun(runId);
  const usage = useProviderUsage(runId);

  if (!runId) return <EmptyState message="Select a run to inspect details." />;
  if (run.isLoading) return <LoadingState />;
  if (run.isError) return <ErrorState error={run.error} />;
  if (!run.data) return <EmptyState message="No run data." />;
  const r = run.data.run;

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        <StatusBadge status={r.status} />
        <code className="break-all text-xs text-gray-700">{r.run_id}</code>
        <CopyButton value={r.run_id} />
      </div>
      <div className="grid grid-cols-2 gap-2 md:grid-cols-3">
        <MetricTile label="Observation" value={formatUtc(r.observation_date)} />
        <MetricTile label="Started" value={formatUtc(r.started_at)} />
        <MetricTile label="Completed" value={formatUtc(r.completed_at)} />
        <MetricTile label="Source" value={r.source_id} />
        <MetricTile label="Built" value={r.events_built} />
        <MetricTile label="Published" value={r.events_published} />
        <MetricTile label="Provider Reqs" value={r.provider_requests} />
        <MetricTile label="Failures" value={r.failures} />
      </div>
      {r.error_message && (
        <div className="rounded border border-red-200 bg-red-50 p-2 text-sm text-red-800">
          {r.error_message}
        </div>
      )}
      <JsonViewer label="Datasets" value={r.datasets} />
      <JsonViewer label="Config" value={r.config} />
      <JsonViewer label="Report" value={r.report} />
      <div>
        <div className="mb-1 text-xs font-medium text-gray-600">Provider Usage</div>
        {usage.isLoading ? (
          <LoadingState />
        ) : usage.isError ? (
          <ErrorState error={usage.error} />
        ) : usage.data && usage.data.provider_usage.length ? (
          <table className="w-full text-xs">
            <thead className="text-left text-gray-500">
              <tr>
                <th className="p-1">Provider</th>
                <th className="p-1">Dataset</th>
                <th className="p-1">Req</th>
                <th className="p-1">Retry</th>
                <th className="p-1">Events</th>
                <th className="p-1">Created</th>
              </tr>
            </thead>
            <tbody>
              {usage.data.provider_usage.map((u) => (
                <tr key={u.usage_id} className="border-t border-gray-100">
                  <td className="p-1">{u.provider}</td>
                  <td className="p-1">{u.dataset}</td>
                  <td className="p-1">{u.request_count}</td>
                  <td className="p-1">{u.retry_count}</td>
                  <td className="p-1">{u.event_count}</td>
                  <td className="p-1">{formatUtc(u.created_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <EmptyState message="No provider usage for this run." />
        )}
      </div>
    </div>
  );
}
