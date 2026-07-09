import type { ReactNode } from 'react';
import { Link } from '@tanstack/react-router';
import { RefreshCw } from 'lucide-react';
import {
  useHealthz,
  useReadyz,
  useRuns,
  useRawEvents,
  useRecentProviderUsage,
  useCatalogSources,
  useCatalogPipelines,
  useCatalogRules,
  useNormalizedEvents,
  useSignals,
  useAlerts,
  useInsights,
} from '../api/queries';
import { useUi } from '../store/ui';
import { MetricTile } from '../components/MetricTile';
import { StatusBadge } from '../components/StatusBadge';
import { LoadingState, ErrorState, EmptyState } from '../components/States';
import { formatUtc, orDash } from '../lib/format';
import type { ProviderUsage } from '../types';
import { useTenant } from '../auth/session';

type RouteLink = '/runs' | '/raw-events' | '/normalized-events' | '/signals' | '/alerts' | '/insights' | '/sources' | '/pipelines' | '/rules' | '/system';

function Panel({ title, linkTo, children }: { title: string; linkTo?: RouteLink; children: ReactNode }) {
  return (
    <section className="rounded border border-gray-200 bg-white p-3">
      <div className="mb-2">
        {linkTo ? (
          <Link to={linkTo} className="text-sm font-semibold text-gray-900 hover:underline">
            {title}
          </Link>
        ) : (
          <h2 className="text-sm font-semibold text-gray-900">{title}</h2>
        )}
      </div>
      {children}
    </section>
  );
}

function aggregateProviderUsage(rows: ProviderUsage[]) {
  const map = new Map<
    string,
    { provider: string; request_count: number; retry_count: number; event_count: number }
  >();
  for (const r of rows) {
    const cur = map.get(r.provider) ?? {
      provider: r.provider,
      request_count: 0,
      retry_count: 0,
      event_count: 0,
    };
    cur.request_count += r.request_count;
    cur.retry_count += r.retry_count;
    cur.event_count += r.event_count;
    map.set(r.provider, cur);
  }
  return [...map.values()];
}

export function DashboardRoute() {
  const TENANT_ID = useTenant();
  const healthz = useHealthz();
  const readyz = useReadyz();
  const runs = useRuns(10);
  const rawEvents = useRawEvents({ tenant_id: TENANT_ID, limit: 10 });
  const providerUsage = useRecentProviderUsage(50);
  const sources = useCatalogSources(TENANT_ID, 50);
  const pipelines = useCatalogPipelines(TENANT_ID, 50);
  const rules = useCatalogRules(TENANT_ID, 50);
  const normalizedEvents = useNormalizedEvents({ tenant_id: TENANT_ID, limit: 50 });
  const signals = useSignals({ tenant_id: TENANT_ID, limit: 50 });
  const alerts = useAlerts({ tenant_id: TENANT_ID, status: 'open', limit: 50 });
  const insights = useInsights({ tenant_id: TENANT_ID, status: 'active', limit: 50 });

  const lastRefresh = useUi((s) => s.lastRefresh);
  const setLastRefresh = useUi((s) => s.setLastRefresh);
  const streamConnected = useUi((s) => s.streamConnected);
  const lastStreamEventAt = useUi((s) => s.lastStreamEventAt);
  const streamError = useUi((s) => s.streamError);
  const restFallback = useUi((s) => s.streamMode) === 'rest_fallback';

  const runsData = runs.data?.runs ?? [];
  const failedRuns = runsData.filter((r) => r.status === 'failed').length;
  const rawData = rawEvents.data?.raw_events ?? [];
  const usageData = providerUsage.data?.provider_usage ?? [];
  const totalRequests = usageData.reduce((n, u) => n + u.request_count, 0);
  const sourcesData = sources.data?.sources ?? [];
  const pipelinesData = pipelines.data?.pipelines ?? [];
  const rulesData = rules.data?.rules ?? [];
  const activeSources = sourcesData.filter((s) => s.status === 'active').length;
  const activePipelines = pipelinesData.filter((p) => p.status === 'active').length;
  const activeRules = rulesData.filter((r) => r.status === 'active').length;
  const normalizedCount = normalizedEvents.data?.normalized_events.length ?? 0;
  const signalsData = signals.data?.signals ?? [];
  const alertsData = alerts.data?.alerts ?? [];
  const insightsData = insights.data?.insights ?? [];

  // Under auth, SSE is intentionally off (native EventSource cannot carry a Bearer token);
  // a REST polling interval keeps the dashboard fresh. Show that distinctly rather than as a
  // reconnecting/connecting stream.
  const streamState = restFallback
    ? 'REST refresh'
    : streamError
      ? 'reconnecting'
      : streamConnected
        ? 'connected'
        : 'connecting';

  function refreshAll() {
    healthz.refetch();
    readyz.refetch();
    runs.refetch();
    rawEvents.refetch();
    providerUsage.refetch();
    sources.refetch();
    pipelines.refetch();
    rules.refetch();
    normalizedEvents.refetch();
    signals.refetch();
    alerts.refetch();
    insights.refetch();
    setLastRefresh(new Date().toISOString());
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-lg font-semibold">Dashboard</h1>
          <p className="text-xs text-gray-500">
            Updated {formatUtc(lastRefresh ?? undefined)} · stream {streamState} · last event{' '}
            {formatUtc(lastStreamEventAt ?? undefined)}
          </p>
        </div>
        <button
          type="button"
          onClick={refreshAll}
          aria-label="Refresh dashboard"
          title="Refresh dashboard"
          className="inline-flex items-center gap-1 rounded border border-gray-300 bg-white px-2 py-1 text-sm text-gray-700 hover:bg-gray-50"
        >
          <RefreshCw size={14} /> Refresh
        </button>
      </div>

      <div className="grid grid-cols-2 gap-2 md:grid-cols-3 lg:grid-cols-[repeat(13,minmax(0,1fr))]">
        <MetricTile label="Gateway" value={healthz.data?.status ?? (healthz.isError ? 'unreachable' : '…')} />
        <MetricTile label="Readiness" value={readyz.data?.status ?? (readyz.isError ? 'unreachable' : '…')} />
        <MetricTile label="Recent Runs" value={runsData.length} />
        <MetricTile label="Failed Runs" value={failedRuns} />
        <MetricTile label="Raw Events" value={rawData.length} />
        <MetricTile label="Provider Reqs" value={totalRequests} />
        <MetricTile label="Active Sources" value={activeSources} />
        <MetricTile label="Active Pipelines" value={activePipelines} />
        <MetricTile label="Active Rules" value={activeRules} />
        <MetricTile label="Normalized" value={normalizedCount} />
        <MetricTile label="Signals" value={signalsData.length} />
        <MetricTile label="Open Alerts" value={alertsData.length} hint={alerts.isError ? 'unreachable' : undefined} />
        <MetricTile label="Active Insights" value={insightsData.length} hint={insights.isError ? 'unreachable' : undefined} />
      </div>

      <div className="grid grid-cols-1 gap-3 lg:grid-cols-3">
        <div className="lg:col-span-2">
          <Panel title="Processing Health" linkTo="/system">
            {healthz.isError && readyz.isError ? (
              <ErrorState error={healthz.error} />
            ) : (
              <div className="grid grid-cols-2 gap-2 text-sm md:grid-cols-3">
                <div>
                  <div className="text-xs text-gray-500">Gateway</div>
                  <div>{healthz.data?.status ?? '…'}</div>
                </div>
                <div>
                  <div className="text-xs text-gray-500">Readiness</div>
                  <div>{readyz.data?.status ?? '…'}</div>
                </div>
                <div>
                  <div className="text-xs text-gray-500">Stream</div>
                  <div>{streamState}</div>
                </div>
                <div>
                  <div className="text-xs text-gray-500">Last heartbeat</div>
                  <div>{formatUtc(lastStreamEventAt ?? undefined)}</div>
                </div>
                <div>
                  <div className="text-xs text-gray-500">Latest run</div>
                  <div>
                    {runsData[0] ? (
                      <>
                        <StatusBadge status={runsData[0].status} />{' '}
                        <span className="text-xs text-gray-500">{formatUtc(runsData[0].started_at)}</span>
                      </>
                    ) : (
                      '—'
                    )}
                  </div>
                </div>
                <div>
                  <div className="text-xs text-gray-500">Failed runs (sample)</div>
                  <div>{failedRuns}</div>
                </div>
              </div>
            )}
            <div className="mt-2 text-xs text-gray-500">
              <Link to="/system" className="hover:underline">System</Link> ·{' '}
              <Link to="/runs" className="hover:underline">Runs</Link>
            </div>
          </Panel>
        </div>
        <Panel title="Catalog Inventory">
          {sources.isError || pipelines.isError || rules.isError ? (
            <ErrorState error={sources.error ?? pipelines.error ?? rules.error} />
          ) : (
            <div className="space-y-1 text-sm">
              <div className="flex justify-between">
                <span>Sources</span>
                <span>
                  {sourcesData.length} ({activeSources} active)
                </span>
              </div>
              <div className="flex justify-between">
                <span>Pipelines</span>
                <span>
                  {pipelinesData.length} ({activePipelines} active)
                </span>
              </div>
              <div className="flex justify-between">
                <span>Rules</span>
                <span>
                  {rulesData.length} ({activeRules} active)
                </span>
              </div>
            </div>
          )}
          <div className="mt-2 text-xs text-gray-500">
            <Link to="/sources" className="hover:underline">Sources</Link> ·{' '}
            <Link to="/pipelines" className="hover:underline">Pipelines</Link> ·{' '}
            <Link to="/rules" className="hover:underline">Rules</Link>
          </div>
        </Panel>
      </div>

      <div className="grid grid-cols-1 gap-3 lg:grid-cols-3">
        <div className="lg:col-span-2">
          <Panel title="Recent Runs" linkTo="/runs">
            {runs.isLoading ? (
              <LoadingState />
            ) : runs.isError ? (
              <ErrorState error={runs.error} />
            ) : runsData.length ? (
              <div className="overflow-x-auto">
                <table className="min-w-full text-xs">
                  <thead className="text-left text-gray-500">
                    <tr>
                      <th className="p-1">Status</th>
                      <th className="p-1">Started</th>
                      <th className="p-1">Adapter</th>
                      <th className="p-1">Datasets</th>
                      <th className="p-1">Events</th>
                      <th className="p-1">Fail</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-100">
                    {runsData.slice(0, 5).map((r) => (
                      <tr key={r.run_id}>
                        <td className="p-1"><StatusBadge status={r.status} /></td>
                        <td className="p-1 text-gray-600">{formatUtc(r.started_at)}</td>
                        <td className="p-1 font-mono">{r.source_adapter}</td>
                        <td className="p-1">{r.datasets.join(', ') || '—'}</td>
                        <td className="p-1">{r.events_published}</td>
                        <td className="p-1">{r.failures}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <EmptyState message="No recent runs." />
            )}
          </Panel>
        </div>
        <Panel title="Provider Usage">
          {providerUsage.isLoading ? (
            <LoadingState />
          ) : providerUsage.isError ? (
            <ErrorState error={providerUsage.error} />
          ) : usageData.length ? (
            <div className="overflow-x-auto">
              <table className="min-w-full text-xs">
                <thead className="text-left text-gray-500">
                  <tr>
                    <th className="p-1">Provider</th>
                    <th className="p-1">Requests</th>
                    <th className="p-1">Retries</th>
                    <th className="p-1">Events</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {aggregateProviderUsage(usageData).map((row) => (
                    <tr key={row.provider}>
                      <td className="p-1 font-mono">{row.provider}</td>
                      <td className="p-1">{row.request_count}</td>
                      <td className="p-1">{row.retry_count}</td>
                      <td className="p-1">{row.event_count}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <EmptyState message="No provider usage." />
          )}
          <p className="mt-1 text-xs text-gray-500">Aggregated from the recent sample; not lifetime totals.</p>
        </Panel>
      </div>

      <Panel title="Recent Event Stream" linkTo="/raw-events">
        {rawEvents.isLoading ? (
          <LoadingState />
        ) : rawEvents.isError ? (
          <ErrorState error={rawEvents.error} />
        ) : rawData.length ? (
          <div className="overflow-x-auto">
            <table className="min-w-full text-xs">
              <thead className="text-left text-gray-500">
                <tr>
                  <th className="p-1">Observation</th>
                  <th className="p-1">Adapter</th>
                  <th className="p-1">Dataset</th>
                  <th className="p-1">Event ID</th>
                  <th className="p-1">Partition</th>
                  <th className="p-1">Offset</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {rawData.slice(0, 8).map((e) => (
                  <tr key={e.event_id}>
                    <td className="p-1 text-gray-600">{formatUtc(e.observation_time)}</td>
                    <td className="p-1 font-mono">{e.source_adapter}</td>
                    <td className="p-1">{e.dataset}</td>
                    <td className="p-1 font-mono">{e.event_id}</td>
                    <td className="p-1">{orDash(e.broker_partition)}</td>
                    <td className="p-1">{orDash(e.broker_offset)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <EmptyState message="No recent raw events." />
        )}
      </Panel>

      <Panel title="Recent Signals" linkTo="/signals">
        {signals.isLoading ? (
          <LoadingState />
        ) : signals.isError ? (
          <ErrorState error={signals.error} />
        ) : signalsData.length ? (
          <div className="overflow-x-auto">
            <table className="min-w-full text-xs">
              <thead className="text-left text-gray-500">
                <tr>
                  <th className="p-1">Signal</th>
                  <th className="p-1">Detector</th>
                  <th className="p-1">Severity</th>
                  <th className="p-1">Confidence</th>
                  <th className="p-1">Created</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {signalsData.slice(0, 5).map((s) => (
                  <tr key={s.signal_id}>
                    <td className="p-1 font-mono">{s.signal_id}</td>
                    <td className="p-1 font-mono">{s.detector_id}</td>
                    <td className="p-1">{s.severity}</td>
                    <td className="p-1">{s.confidence.toFixed(2)}</td>
                    <td className="p-1 text-gray-600">{formatUtc(s.created_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <EmptyState message="No recent signals." />
        )}
      </Panel>

      <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
        <Panel title="Open Alerts" linkTo="/alerts">
          {alerts.isLoading ? (
            <LoadingState />
          ) : alerts.isError ? (
            <ErrorState error={alerts.error} />
          ) : alertsData.length ? (
            <div className="overflow-x-auto">
              <table className="min-w-full text-xs">
                <thead className="text-left text-gray-500">
                  <tr>
                    <th className="p-1">Alert</th>
                    <th className="p-1">Severity</th>
                    <th className="p-1">Status</th>
                    <th className="p-1">Source/Dataset</th>
                    <th className="p-1">Last Observed</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {alertsData.slice(0, 5).map((a) => (
                    <tr key={a.alert_id}>
                      <td className="p-1 font-mono">{a.alert_id}</td>
                      <td className="p-1">{a.severity}</td>
                      <td className="p-1">{a.status}</td>
                      <td className="p-1"><span className="font-mono">{a.source_id}</span> / {a.dataset}</td>
                      <td className="p-1 text-gray-600">{formatUtc(a.last_observed_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <EmptyState message="No open alerts." />
          )}
        </Panel>
        <Panel title="Active Insights" linkTo="/insights">
          {insights.isLoading ? (
            <LoadingState />
          ) : insights.isError ? (
            <ErrorState error={insights.error} />
          ) : insightsData.length ? (
            <div className="overflow-x-auto">
              <table className="min-w-full text-xs">
                <thead className="text-left text-gray-500">
                  <tr>
                    <th className="p-1">Insight</th>
                    <th className="p-1">Severity</th>
                    <th className="p-1">Status</th>
                    <th className="p-1">Source/Dataset</th>
                    <th className="p-1">Observed</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {insightsData.slice(0, 5).map((i) => (
                    <tr key={i.insight_id}>
                      <td className="p-1 font-mono">{i.insight_id}</td>
                      <td className="p-1">{i.severity}</td>
                      <td className="p-1">{i.status}</td>
                      <td className="p-1"><span className="font-mono">{i.source_id}</span> / {i.dataset}</td>
                      <td className="p-1 text-gray-600">{formatUtc(i.observed_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <EmptyState message="No active insights." />
          )}
        </Panel>
      </div>
      <p className="text-xs text-gray-500">
        Signals = detector output · Alerts = medium/high/critical lifecycle · Insights = every signal.
      </p>
    </div>
  );
}
