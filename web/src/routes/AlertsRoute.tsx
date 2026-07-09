import { useState } from 'react';
import { Link } from '@tanstack/react-router';
import { TriangleAlert, CheckCircle2, CircleCheck, BellOff } from 'lucide-react';
import { useAlerts, useAlert, useMutateAlertLifecycle } from '../api/queries';
import { isApiError } from '../api/client';
import { LoadingState, ErrorState, EmptyState } from '../components/States';
import { MetricTile } from '../components/MetricTile';
import { CopyButton } from '../components/CopyButton';
import { JsonViewer } from '../components/JsonViewer';
import { formatUtc } from '../lib/format';
import type { AlertRecord, AlertLifecycleAction } from '../types';
import { useTenant, useCanMutateLifecycle } from '../auth/session';

type LifecycleMeta = {
  action?: string;
  actor?: string;
  note?: string;
  reason?: string;
  mutated_at?: string;
};

function lifecycleOf(metadata: unknown): LifecycleMeta | undefined {
  return metadata && typeof metadata === 'object' && 'lifecycle' in metadata
    ? (metadata as { lifecycle?: LifecycleMeta }).lifecycle
    : undefined;
}

const SEVERITY_STYLES: Record<string, string> = {
  critical: 'text-red-700',
  high: 'text-orange-700',
  medium: 'text-amber-700',
  low: 'text-gray-700',
  info: 'text-gray-500',
};

function SeverityLabel({ severity }: { severity: string }) {
  return (
    <span className={`text-xs font-medium ${SEVERITY_STYLES[severity] ?? 'text-gray-600'}`}>{severity}</span>
  );
}

const STATUS_STYLES: Record<string, string> = {
  open: 'text-orange-700',
  acknowledged: 'text-blue-700',
  resolved: 'text-green-700',
  suppressed: 'text-gray-500',
};

function StatusLabel({ status }: { status: string }) {
  return <span className={`text-xs font-medium ${STATUS_STYLES[status] ?? 'text-gray-600'}`}>{status}</span>;
}

export function AlertsRoute() {
  const TENANT_ID = useTenant();
  const [sourceId, setSourceId] = useState('');
  const [dataset, setDataset] = useState('');
  const [severity, setSeverity] = useState('');
  const [status, setStatus] = useState('open');
  const [limit, setLimit] = useState(50);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const list = useAlerts({
    tenant_id: TENANT_ID,
    source_id: sourceId || undefined,
    dataset: dataset || undefined,
    severity: severity || undefined,
    status: status || undefined,
    limit,
  });
  const detail = useAlert(selectedId);
  const data = list.data?.alerts ?? [];

  const open = data.filter((a) => a.status === 'open').length;
  const highCritical = data.filter((a) => a.severity === 'high' || a.severity === 'critical').length;
  const distinctSources = new Set(data.map((a) => a.source_id)).size;
  const avgConfidence = data.length ? data.reduce((n, a) => n + a.confidence, 0) / data.length : 0;

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h1 className="text-lg font-semibold">Alerts</h1>
        <div className="flex flex-wrap items-center gap-2">
          <input
            placeholder="source id"
            value={sourceId}
            onChange={(e) => setSourceId(e.target.value)}
            className="rounded border border-gray-300 px-2 py-1 text-sm"
          />
          <input
            placeholder="dataset"
            value={dataset}
            onChange={(e) => setDataset(e.target.value)}
            className="rounded border border-gray-300 px-2 py-1 text-sm"
          />
          <select
            value={severity}
            onChange={(e) => setSeverity(e.target.value)}
            className="rounded border border-gray-300 px-2 py-1 text-sm"
          >
            <option value="">any severity</option>
            {['info', 'low', 'medium', 'high', 'critical'].map((s) => (
              <option key={s} value={s}>{s}</option>
            ))}
          </select>
          <select
            value={status}
            onChange={(e) => setStatus(e.target.value)}
            className="rounded border border-gray-300 px-2 py-1 text-sm"
          >
            <option value="">any status</option>
            {['open', 'acknowledged', 'resolved', 'suppressed'].map((s) => (
              <option key={s} value={s}>{s}</option>
            ))}
          </select>
          <select
            value={limit}
            onChange={(e) => setLimit(Number(e.target.value))}
            className="rounded border border-gray-300 px-2 py-1 text-sm"
          >
            {[25, 50, 100, 200].map((n) => (
              <option key={n} value={n}>{n}</option>
            ))}
          </select>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-2 md:grid-cols-5">
        <MetricTile label="Alerts" value={data.length} />
        <MetricTile label="Open" value={open} />
        <MetricTile label="High/Critical" value={highCritical} />
        <MetricTile label="Distinct Sources" value={distinctSources} />
        <MetricTile label="Avg Confidence" value={avgConfidence.toFixed(2)} />
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <div className="lg:col-span-2">
          {list.isLoading ? (
            <LoadingState />
          ) : list.isError ? (
            <ErrorState error={list.error} />
          ) : data.length ? (
            <div className="overflow-x-auto rounded border border-gray-200 bg-white">
              <table className="min-w-full divide-y divide-gray-200 text-sm">
                <thead className="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500">
                  <tr>
                    <th className="px-3 py-2">Alert</th>
                    <th className="px-3 py-2">Severity</th>
                    <th className="px-3 py-2">Status</th>
                    <th className="px-3 py-2">Type</th>
                    <th className="px-3 py-2">Detector</th>
                    <th className="px-3 py-2">Source/Dataset</th>
                    <th className="px-3 py-2">Confidence</th>
                    <th className="px-3 py-2">Events</th>
                    <th className="px-3 py-2">First Observed</th>
                    <th className="px-3 py-2">Last Observed</th>
                    <th className="px-3 py-2">Updated</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {data.map((a) => (
                    <tr
                      key={a.alert_id}
                      onClick={() => setSelectedId(a.alert_id)}
                      className={`cursor-pointer align-top hover:bg-gray-50 ${selectedId === a.alert_id ? 'bg-brand-50' : ''}`}
                    >
                      <td className="px-3 py-2">
                        <div className="flex items-start gap-2">
                          <TriangleAlert size={16} className="mt-0.5 text-orange-700" />
                          <div>
                            <div className="text-xs font-medium text-gray-800">{a.title}</div>
                            <div className="font-mono text-xs text-gray-700">{a.alert_id}</div>
                            <div className="text-xs text-gray-500">{a.summary}</div>
                          </div>
                        </div>
                      </td>
                      <td className="px-3 py-2"><SeverityLabel severity={a.severity} /></td>
                      <td className="px-3 py-2"><StatusLabel status={a.status} /></td>
                      <td className="px-3 py-2 text-xs font-mono">{a.alert_type}</td>
                      <td className="px-3 py-2 text-xs font-mono">{a.detector_id}</td>
                      <td className="px-3 py-2 text-xs">
                        <div className="font-mono">{a.source_id}</div>
                        <div>{a.dataset}</div>
                      </td>
                      <td className="px-3 py-2 text-xs">{a.confidence.toFixed(2)}</td>
                      <td className="px-3 py-2 text-xs">{a.event_ids.length}</td>
                      <td className="px-3 py-2 text-xs text-gray-600">{formatUtc(a.first_observed_at)}</td>
                      <td className="px-3 py-2 text-xs text-gray-600">{formatUtc(a.last_observed_at)}</td>
                      <td className="px-3 py-2 text-xs text-gray-600">{formatUtc(a.updated_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <EmptyState message="No alerts found." />
          )}
        </div>

        <div className="rounded border border-gray-200 bg-white p-3">
          {!selectedId ? (
            <EmptyState message="Select an alert to inspect details." />
          ) : detail.isLoading ? (
            <LoadingState />
          ) : detail.isError ? (
            <ErrorState error={detail.error} />
          ) : detail.data ? (
            <AlertDetailBody key={detail.data.alert.alert_id} alert={detail.data.alert} />
          ) : null}
        </div>
      </div>
    </div>
  );
}

function AlertDetailBody({ alert }: { alert: AlertRecord }) {
  const mutation = useMutateAlertLifecycle();
  const lifecycle = lifecycleOf(alert.metadata);
  const status = alert.status;
  const canAcknowledge = !['acknowledged', 'resolved', 'suppressed'].includes(status);
  const canResolve = !['resolved', 'suppressed'].includes(status);
  const canSuppress = status !== 'suppressed';
  const pending = mutation.isPending;
  const canMutate = useCanMutateLifecycle();
  const run = (action: AlertLifecycleAction) => mutation.mutate({ alertId: alert.alert_id, action });

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        <SeverityLabel severity={alert.severity} />
        <StatusLabel status={alert.status} />
        <code className="break-all text-xs text-gray-700">{alert.alert_id}</code>
        <CopyButton value={alert.alert_id} />
      </div>
      <div className="text-sm text-gray-700">{alert.summary}</div>

      <div className="space-y-1">
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            disabled={pending || !canAcknowledge || !canMutate}
            title={!canMutate ? 'Requires operator or admin role' : undefined}
            onClick={() => run('acknowledge')}
            className="inline-flex items-center gap-1 rounded bg-brand-500 px-2 py-1 text-xs text-white hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <CheckCircle2 size={14} /> Acknowledge
          </button>
          <button
            type="button"
            disabled={pending || !canResolve || !canMutate}
            title={!canMutate ? 'Requires operator or admin role' : undefined}
            onClick={() => run('resolve')}
            className="inline-flex items-center gap-1 rounded bg-brand-500 px-2 py-1 text-xs text-white hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <CircleCheck size={14} /> Resolve
          </button>
          <button
            type="button"
            disabled={pending || !canSuppress || !canMutate}
            title={!canMutate ? 'Requires operator or admin role' : undefined}
            onClick={() => run('suppress')}
            className="inline-flex items-center gap-1 rounded border border-gray-300 bg-white px-2 py-1 text-xs text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <BellOff size={14} /> Suppress
          </button>
        </div>
        {!canMutate && <p className="text-xs text-gray-500">Lifecycle actions require operator or admin role.</p>}
        {mutation.isError && (
          <p className="text-xs text-red-700" role="alert">
            Action failed: {isApiError(mutation.error) ? mutation.error.message : 'unknown error'}. Selection preserved.
          </p>
        )}
        {pending && <p className="text-xs text-gray-500">Applying lifecycle action…</p>}
      </div>

      {lifecycle && (
        <div className="rounded border border-gray-200 bg-gray-50 p-2 text-xs">
          <div><span className="text-gray-500">Last action:</span> {lifecycle.action ?? '—'} by {lifecycle.actor ?? '—'} at {formatUtc(lifecycle.mutated_at)}</div>
          {(lifecycle.note || lifecycle.reason) && (
            <div><span className="text-gray-500">Note:</span> {lifecycle.note ?? lifecycle.reason}</div>
          )}
        </div>
      )}

      <div className="grid grid-cols-2 gap-2 text-sm">
        <div><div className="text-xs text-gray-500">Type</div><div className="text-xs font-mono">{alert.alert_type}</div></div>
        <div><div className="text-xs text-gray-500">Confidence</div><div>{alert.confidence.toFixed(2)}</div></div>
        <div><div className="text-xs text-gray-500">Signal</div><div className="text-xs"><Link to="/signals" className="font-mono text-brand-700 hover:underline">{alert.signal_id}</Link></div></div>
        <div><div className="text-xs text-gray-500">Detector</div><div className="text-xs font-mono">{alert.detector_id}</div></div>
        <div><div className="text-xs text-gray-500">Source</div><div className="text-xs font-mono">{alert.source_id}</div></div>
        <div><div className="text-xs text-gray-500">Dataset</div><div className="text-xs">{alert.dataset}</div></div>
        <div><div className="text-xs text-gray-500">Source Domain</div><div className="text-xs font-mono">{alert.source_domain}</div></div>
        <div><div className="text-xs text-gray-500">Source Adapter</div><div className="text-xs font-mono">{alert.source_adapter}</div></div>
        <div><div className="text-xs text-gray-500">First Observed</div><div className="text-xs">{formatUtc(alert.first_observed_at)}</div></div>
        <div><div className="text-xs text-gray-500">Last Observed</div><div className="text-xs">{formatUtc(alert.last_observed_at)}</div></div>
        <div><div className="text-xs text-gray-500">Correlation</div><div className="break-all text-xs font-mono">{alert.correlation_id}</div></div>
        <div><div className="text-xs text-gray-500">Tenant</div><div className="text-xs font-mono">{alert.tenant_id}</div></div>
        {alert.acknowledged_at && (
          <div><div className="text-xs text-gray-500">Acknowledged</div><div className="text-xs">{formatUtc(alert.acknowledged_at)}{alert.acknowledged_by ? ` by ${alert.acknowledged_by}` : ''}</div></div>
        )}
        {alert.resolved_at && (
          <div><div className="text-xs text-gray-500">Resolved</div><div className="text-xs">{formatUtc(alert.resolved_at)}{alert.resolved_by ? ` by ${alert.resolved_by}` : ''}</div></div>
        )}
      </div>
      <div>
        <div className="mb-1 text-xs font-medium text-gray-600">Event IDs</div>
        {alert.event_ids.length ? (
          <div className="flex flex-wrap gap-x-3 gap-y-1">
            {alert.event_ids.map((id) => (
              <Link key={id} to="/normalized-events" className="break-all text-xs text-brand-700 hover:underline">{id}</Link>
            ))}
          </div>
        ) : (
          <span className="text-xs text-gray-400">—</span>
        )}
      </div>
      <JsonViewer label="Entities" value={alert.entities} />
      <JsonViewer label="Evidence" value={alert.evidence} />
      <JsonViewer label="Recommendation" value={alert.recommendation} />
      <JsonViewer label="Metadata" value={alert.metadata} />
    </div>
  );
}
