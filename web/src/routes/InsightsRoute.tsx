import { useState } from 'react';
import { Link } from '@tanstack/react-router';
import { Lightbulb, Eye, XCircle, Archive } from 'lucide-react';
import { useInsights, useInsight, useMutateInsightLifecycle } from '../api/queries';
import { isApiError } from '../api/client';
import { LoadingState, ErrorState, EmptyState } from '../components/States';
import { MetricTile } from '../components/MetricTile';
import { CopyButton } from '../components/CopyButton';
import { JsonViewer } from '../components/JsonViewer';
import { formatUtc } from '../lib/format';
import type { InsightRecord, InsightLifecycleAction } from '../types';
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
  active: 'text-blue-700',
  reviewed: 'text-green-700',
  dismissed: 'text-gray-500',
  archived: 'text-gray-400',
};

function StatusLabel({ status }: { status: string }) {
  return <span className={`text-xs font-medium ${STATUS_STYLES[status] ?? 'text-gray-600'}`}>{status}</span>;
}

export function InsightsRoute() {
  const TENANT_ID = useTenant();
  const [sourceId, setSourceId] = useState('');
  const [dataset, setDataset] = useState('');
  const [insightType, setInsightType] = useState('');
  const [status, setStatus] = useState('active');
  const [limit, setLimit] = useState(50);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const list = useInsights({
    tenant_id: TENANT_ID,
    source_id: sourceId || undefined,
    dataset: dataset || undefined,
    insight_type: insightType || undefined,
    status: status || undefined,
    limit,
  });
  const detail = useInsight(selectedId);
  const data = list.data?.insights ?? [];

  const active = data.filter((i) => i.status === 'active').length;
  const insightTypes = new Set(data.map((i) => i.insight_type)).size;
  const highCritical = data.filter((i) => i.severity === 'high' || i.severity === 'critical').length;
  const avgConfidence = data.length ? data.reduce((n, i) => n + i.confidence, 0) / data.length : 0;

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h1 className="text-lg font-semibold">Insights</h1>
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
          <input
            placeholder="insight type"
            value={insightType}
            onChange={(e) => setInsightType(e.target.value)}
            className="rounded border border-gray-300 px-2 py-1 text-sm"
          />
          <select
            value={status}
            onChange={(e) => setStatus(e.target.value)}
            className="rounded border border-gray-300 px-2 py-1 text-sm"
          >
            <option value="">any status</option>
            {['active', 'reviewed', 'dismissed', 'archived'].map((s) => (
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
        <MetricTile label="Insights" value={data.length} />
        <MetricTile label="Active" value={active} />
        <MetricTile label="Insight Types" value={insightTypes} />
        <MetricTile label="High/Critical" value={highCritical} />
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
                    <th className="px-3 py-2">Insight</th>
                    <th className="px-3 py-2">Status</th>
                    <th className="px-3 py-2">Severity</th>
                    <th className="px-3 py-2">Type</th>
                    <th className="px-3 py-2">Detector</th>
                    <th className="px-3 py-2">Source/Dataset</th>
                    <th className="px-3 py-2">Confidence</th>
                    <th className="px-3 py-2">Events</th>
                    <th className="px-3 py-2">Observed</th>
                    <th className="px-3 py-2">Updated</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {data.map((i) => (
                    <tr
                      key={i.insight_id}
                      onClick={() => setSelectedId(i.insight_id)}
                      className={`cursor-pointer align-top hover:bg-gray-50 ${selectedId === i.insight_id ? 'bg-brand-50' : ''}`}
                    >
                      <td className="px-3 py-2">
                        <div className="flex items-start gap-2">
                          <Lightbulb size={16} className="mt-0.5 text-amber-500" />
                          <div>
                            <div className="text-xs font-medium text-gray-800">{i.title}</div>
                            <div className="font-mono text-xs text-gray-700">{i.insight_id}</div>
                            <div className="text-xs text-gray-500">{i.summary}</div>
                          </div>
                        </div>
                      </td>
                      <td className="px-3 py-2"><StatusLabel status={i.status} /></td>
                      <td className="px-3 py-2"><SeverityLabel severity={i.severity} /></td>
                      <td className="px-3 py-2 text-xs font-mono">{i.insight_type}</td>
                      <td className="px-3 py-2 text-xs font-mono">{i.detector_id}</td>
                      <td className="px-3 py-2 text-xs">
                        <div className="font-mono">{i.source_id}</div>
                        <div>{i.dataset}</div>
                      </td>
                      <td className="px-3 py-2 text-xs">{i.confidence.toFixed(2)}</td>
                      <td className="px-3 py-2 text-xs">{i.event_ids.length}</td>
                      <td className="px-3 py-2 text-xs text-gray-600">{formatUtc(i.observed_at)}</td>
                      <td className="px-3 py-2 text-xs text-gray-600">{formatUtc(i.updated_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <EmptyState message="No insights found." />
          )}
        </div>

        <div className="rounded border border-gray-200 bg-white p-3">
          {!selectedId ? (
            <EmptyState message="Select an insight to inspect details." />
          ) : detail.isLoading ? (
            <LoadingState />
          ) : detail.isError ? (
            <ErrorState error={detail.error} />
          ) : detail.data ? (
            <InsightDetailBody key={detail.data.insight.insight_id} insight={detail.data.insight} />
          ) : null}
        </div>
      </div>
    </div>
  );
}

function InsightDetailBody({ insight }: { insight: InsightRecord }) {
  const mutation = useMutateInsightLifecycle();
  const lifecycle = lifecycleOf(insight.metadata);
  const status = insight.status;
  const canReview = !['reviewed', 'dismissed', 'archived'].includes(status);
  const canDismiss = !['dismissed', 'archived'].includes(status);
  const canArchive = status !== 'archived';
  const pending = mutation.isPending;
  const canMutate = useCanMutateLifecycle();
  const run = (action: InsightLifecycleAction) => mutation.mutate({ insightId: insight.insight_id, action });

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        <StatusLabel status={insight.status} />
        <SeverityLabel severity={insight.severity} />
        <code className="break-all text-xs text-gray-700">{insight.insight_id}</code>
        <CopyButton value={insight.insight_id} />
      </div>
      <div className="text-sm text-gray-700">{insight.summary}</div>

      <div className="space-y-1">
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            disabled={pending || !canReview || !canMutate}
            title={!canMutate ? 'Requires operator or admin role' : undefined}
            onClick={() => run('review')}
            className="inline-flex items-center gap-1 rounded bg-brand-500 px-2 py-1 text-xs text-white hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <Eye size={14} /> Review
          </button>
          <button
            type="button"
            disabled={pending || !canDismiss || !canMutate}
            title={!canMutate ? 'Requires operator or admin role' : undefined}
            onClick={() => run('dismiss')}
            className="inline-flex items-center gap-1 rounded border border-gray-300 bg-white px-2 py-1 text-xs text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <XCircle size={14} /> Dismiss
          </button>
          <button
            type="button"
            disabled={pending || !canArchive || !canMutate}
            title={!canMutate ? 'Requires operator or admin role' : undefined}
            onClick={() => run('archive')}
            className="inline-flex items-center gap-1 rounded border border-gray-300 bg-white px-2 py-1 text-xs text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <Archive size={14} /> Archive
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
        <div><div className="text-xs text-gray-500">Type</div><div className="text-xs font-mono">{insight.insight_type}</div></div>
        <div><div className="text-xs text-gray-500">Confidence</div><div>{insight.confidence.toFixed(2)}</div></div>
        <div><div className="text-xs text-gray-500">Signal</div><div className="text-xs"><Link to="/signals" className="font-mono text-brand-700 hover:underline">{insight.signal_id}</Link></div></div>
        <div><div className="text-xs text-gray-500">Detector</div><div className="text-xs font-mono">{insight.detector_id}</div></div>
        <div><div className="text-xs text-gray-500">Source</div><div className="text-xs font-mono">{insight.source_id}</div></div>
        <div><div className="text-xs text-gray-500">Dataset</div><div className="text-xs">{insight.dataset}</div></div>
        <div><div className="text-xs text-gray-500">Source Domain</div><div className="text-xs font-mono">{insight.source_domain}</div></div>
        <div><div className="text-xs text-gray-500">Source Adapter</div><div className="text-xs font-mono">{insight.source_adapter}</div></div>
        <div><div className="text-xs text-gray-500">Observed</div><div className="text-xs">{formatUtc(insight.observed_at)}</div></div>
        <div><div className="text-xs text-gray-500">Correlation</div><div className="break-all text-xs font-mono">{insight.correlation_id}</div></div>
        <div><div className="text-xs text-gray-500">Tenant</div><div className="text-xs font-mono">{insight.tenant_id}</div></div>
        <div><div className="text-xs text-gray-500">Updated</div><div className="text-xs">{formatUtc(insight.updated_at)}</div></div>
        {insight.reviewed_at && (
          <div><div className="text-xs text-gray-500">Reviewed</div><div className="text-xs">{formatUtc(insight.reviewed_at)}{insight.reviewed_by ? ` by ${insight.reviewed_by}` : ''}</div></div>
        )}
      </div>
      <div>
        <div className="mb-1 text-xs font-medium text-gray-600">Event IDs</div>
        {insight.event_ids.length ? (
          <div className="flex flex-wrap gap-x-3 gap-y-1">
            {insight.event_ids.map((id) => (
              <Link key={id} to="/normalized-events" className="break-all text-xs text-brand-700 hover:underline">{id}</Link>
            ))}
          </div>
        ) : (
          <span className="text-xs text-gray-400">—</span>
        )}
      </div>
      <JsonViewer label="Entities" value={insight.entities} />
      <JsonViewer label="Supporting Metrics" value={insight.supporting_metrics} />
      <JsonViewer label="Semantic Evidence" value={insight.semantic_evidence} />
      <JsonViewer label="Recommendation" value={insight.recommendation} />
      <JsonViewer label="Metadata" value={insight.metadata} />
    </div>
  );
}
