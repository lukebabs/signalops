import { useState } from 'react';
import { Sparkles } from 'lucide-react';
import {
  useSyncraticInsights,
  useSyncraticInsight,
  useSyncraticContextWindow,
  useMaterializeSyncraticContexts,
} from '../api/queries';
import { isApiError } from '../api/client';
import { LoadingState, ErrorState, EmptyState } from '../components/States';
import { MetricTile } from '../components/MetricTile';
import { CopyButton } from '../components/CopyButton';
import { JsonViewer } from '../components/JsonViewer';
import { RefreshButton } from '../components/RefreshButton';
import { formatUtc, orDash, toRfc3339Utc, toDatetimeLocal } from '../lib/format';
import {
  summarizeSyncraticInsight,
  summarizeSyncraticContextWindow,
  summarizeSyncraticMaterialization,
  syncraticSeverityStyle,
  syncraticInsightStatusStyle,
  shortSyncraticId,
} from '../lib/syncratic';
import { useTenant } from '../auth/session';
import { useAppProfile } from '../apps/AppProfileContext';
import type { SyncraticInsight, SyncraticContextWindow, SyncraticInsightStatus } from '../types';

const STATUSES: (SyncraticInsightStatus | '')[] = [
  '',
  'active',
  'reviewed',
  'dismissed',
  'archived',
  'superseded',
];
const LIMITS = [25, 50, 100, 200];

// Fixed materialize defaults — the bounded form does not expose these.
const MATERIALIZE_UNIVERSE_GROUP = 'top50_megacap';
const MATERIALIZE_STRATEGY = 'symbol_signal_cluster_5d';
const MATERIALIZE_BUILDER_VERSION = 'syncratic.context_builder.v1';
const MATERIALIZE_MAX_CANDIDATE_WINDOWS = 50;

function SeverityLabel({ severity }: { severity: string }) {
  return (
    <span className={`text-xs font-medium ${syncraticSeverityStyle(severity)}`}>{severity || '—'}</span>
  );
}

function StatusLabel({ status }: { status: string }) {
  return (
    <span className={`text-xs font-medium ${syncraticInsightStatusStyle(status)}`}>{status || '—'}</span>
  );
}

export function MarketOpsSyncraticRoute() {
  const TENANT_ID = useTenant();
  const { metadataFilter } = useAppProfile();

  const [status, setStatus] = useState<SyncraticInsightStatus | ''>('');
  const [subjectSymbol, setSubjectSymbol] = useState('');
  const [insightType, setInsightType] = useState('');
  const [limit, setLimit] = useState(50);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const insightsList = useSyncraticInsights({
    tenant_id: TENANT_ID,
    app_id: metadataFilter.app_id,
    domain: metadataFilter.domain,
    use_case: metadataFilter.use_case,
    status: status || undefined,
    subject_symbol: subjectSymbol || undefined,
    insight_type: insightType || undefined,
    limit,
  });
  const insightDetail = useSyncraticInsight(selectedId);

  const data = insightsList.data?.syncratic_insights ?? [];
  const selected: SyncraticInsight | null =
    insightDetail.data?.syncratic_insight ?? data.find((i) => i.syncratic_insight_id === selectedId) ?? null;
  const selectedSummary = selected ? summarizeSyncraticInsight(selected) : null;

  const highCritical = data.filter(
    (i) => i.severity === 'high' || i.severity === 'critical',
  ).length;
  const activeCount = data.filter((i) => i.status === 'active').length;
  const totalEvidence = data.reduce(
    (n, i) => n + summarizeSyncraticInsight(i).signalCount + summarizeSyncraticInsight(i).alertCount,
    0,
  );

  const inputCls = 'rounded border border-gray-300 px-2 py-1 text-sm';

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <Sparkles size={18} className="text-brand-700" />
        <div>
          <h1 className="text-lg font-semibold">Syncratic Insights</h1>
          <p className="text-xs text-gray-500">
            Multi-record pattern explanations over bounded evidence windows · not event-level alerts
          </p>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-2 md:grid-cols-4">
        <MetricTile label="Insights" value={data.length} hint={insightsList.isError ? 'unreachable' : undefined} />
        <MetricTile label="Active" value={activeCount} />
        <MetricTile label="High/Critical" value={highCritical} />
        <MetricTile label="Evidence (signals+alerts)" value={totalEvidence} />
      </div>

      <SyncraticMaterializeForm tenantId={TENANT_ID} />

      <div className="flex flex-wrap items-center gap-2">
        <select
          value={status}
          onChange={(e) => setStatus(e.target.value as SyncraticInsightStatus | '')}
          className={inputCls}
          aria-label="Filter by status"
        >
          {STATUSES.map((s) => (
            <option key={s} value={s}>{s || 'any status'}</option>
          ))}
        </select>
        <input
          value={subjectSymbol}
          onChange={(e) => setSubjectSymbol(e.target.value.toUpperCase())}
          className={inputCls}
          aria-label="Filter by subject symbol"
          placeholder="symbol (e.g. AAPL)"
        />
        <input
          value={insightType}
          onChange={(e) => setInsightType(e.target.value)}
          className={inputCls}
          aria-label="Filter by insight type"
          placeholder="insight type"
        />
        <select
          value={limit}
          onChange={(e) => setLimit(Number(e.target.value))}
          className={inputCls}
          aria-label="Page limit"
        >
          {LIMITS.map((n) => (
            <option key={n} value={n}>{n}</option>
          ))}
        </select>
        <RefreshButton onClick={() => insightsList.refetch()} loading={insightsList.isFetching} />
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        {/* Insight list — left/span-2. Emphasizes evidence/window counts, not incident response. */}
        <div className="lg:col-span-2">
          {insightsList.isLoading ? (
            <LoadingState label="Loading Syncratic insights..." />
          ) : insightsList.isError ? (
            <ErrorState error={insightsList.error} />
          ) : data.length ? (
            <div className="overflow-x-auto rounded border border-gray-200 bg-white">
              <table className="min-w-full divide-y divide-gray-200 text-sm">
                <thead className="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500">
                  <tr>
                    <th className="px-3 py-2">Symbol</th>
                    <th className="px-3 py-2">Title</th>
                    <th className="px-3 py-2">Status</th>
                    <th className="px-3 py-2">Severity</th>
                    <th className="px-3 py-2">Conf.</th>
                    <th className="px-3 py-2">Type</th>
                    <th className="px-3 py-2">Window</th>
                    <th className="px-3 py-2">Alerts</th>
                    <th className="px-3 py-2">Signals</th>
                    <th className="px-3 py-2">Graph</th>
                    <th className="px-3 py-2">Updated</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {data.map((i) => {
                    const s = summarizeSyncraticInsight(i);
                    return (
                      <tr
                        key={i.syncratic_insight_id}
                        onClick={() => setSelectedId(i.syncratic_insight_id)}
                        className={`cursor-pointer align-top hover:bg-gray-50 ${selectedId === i.syncratic_insight_id ? 'bg-brand-50' : ''}`}
                      >
                        <td className="px-3 py-2 text-xs font-semibold text-gray-900">{s.subjectSymbol || '—'}</td>
                        <td className="px-3 py-2">
                          <div className="text-xs font-medium text-gray-800">{s.title || s.insightId}</div>
                          <div className="break-all text-xs text-gray-500">{s.summary}</div>
                        </td>
                        <td className="px-3 py-2"><StatusLabel status={s.status} /></td>
                        <td className="px-3 py-2"><SeverityLabel severity={s.severity} /></td>
                        <td className="px-3 py-2 text-xs">{s.confidence.toFixed(2)}</td>
                        <td className="px-3 py-2"><code className="break-all text-xs text-gray-700">{s.insightType || '—'}</code></td>
                        <td className="px-3 py-2">
                          <code className="break-all text-xs text-gray-600" title={s.contextWindowId}>
                            {s.contextWindowId ? shortSyncraticId(s.contextWindowId) : '—'}
                          </code>
                        </td>
                        <td className="px-3 py-2 text-xs">{s.alertCount}</td>
                        <td className="px-3 py-2 text-xs">{s.signalCount}</td>
                        <td className="px-3 py-2 text-xs">{s.graphProposalCount}</td>
                        <td className="px-3 py-2 text-xs text-gray-600">{formatUtc(s.updatedAt)}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          ) : (
            <EmptyState message="No Syncratic insights for the selected filters. Run a bounded Materialize Contexts action above to synthesize pattern-level explanations." />
          )}
        </div>

        {/* Detail panel — selected insight + its context window + evidence references. */}
        <div className="rounded border border-gray-200 bg-white p-3">
          {!selectedId ? (
            <EmptyState message="Select a Syncratic insight to inspect its context window and evidence references." />
          ) : insightDetail.isLoading ? (
            <LoadingState label="Loading insight..." />
          ) : insightDetail.isError ? (
            <ErrorState error={insightDetail.error} />
          ) : selected && selectedSummary ? (
            <SyncraticInsightDetail insight={selected} summary={selectedSummary} />
          ) : null}
        </div>
      </div>
    </div>
  );
}

function SyncraticInsightDetail({
  insight,
  summary,
}: {
  insight: SyncraticInsight;
  summary: ReturnType<typeof summarizeSyncraticInsight>;
}) {
  // Fetch the context window detail by context_window_id (read-only review).
  const contextWindow = useSyncraticContextWindow(summary.contextWindowId || null);
  const cwSummary = contextWindow.data ? summarizeSyncraticContextWindow(contextWindow.data.context_window) : null;

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        <StatusLabel status={summary.status} />
        <SeverityLabel severity={summary.severity} />
        <span className="text-xs text-gray-600">conf {summary.confidence.toFixed(2)}</span>
        <code className="break-all text-xs text-gray-700">{summary.insightId}</code>
        <CopyButton value={summary.insightId} />
      </div>
      <div>
        <div className="text-sm font-medium text-gray-900">{summary.title}</div>
        {summary.summary && <div className="mt-0.5 text-xs text-gray-600">{summary.summary}</div>}
      </div>
      {summary.explanation && (
        <p className="rounded border border-gray-200 bg-gray-50 p-2 text-xs text-gray-700">{summary.explanation}</p>
      )}

      <div className="grid grid-cols-2 gap-2 text-sm">
        <div><div className="text-xs text-gray-500">Type</div><div className="break-all text-xs">{summary.insightType || '—'}</div></div>
        <div><div className="text-xs text-gray-500">Confidence</div><div>{summary.confidence.toFixed(2)}</div></div>
        <div><div className="text-xs text-gray-500">Subject</div><div className="break-all text-xs">{summary.subjectSymbol || summary.subjectId || '—'}</div></div>
        <div><div className="text-xs text-gray-500">Builder</div><div className="break-all text-xs font-mono">{summary.builderVersion || '—'}</div></div>
        <div><div className="text-xs text-gray-500">Created</div><div className="text-xs">{formatUtc(summary.createdAt)}</div></div>
        <div><div className="text-xs text-gray-500">Updated</div><div className="text-xs">{formatUtc(summary.updatedAt)}</div></div>
      </div>

      {/* Evidence references grouped by type (read-only ids — no new routing). */}
      <div className="space-y-2">
        <SyncraticEvidenceList label="Supporting alerts" ids={summary.supportingAlertIds} />
        <SyncraticEvidenceList label="Supporting signals" ids={summary.supportingSignalIds} />
        <SyncraticEvidenceList label="Supporting events" ids={summary.supportingEventIds} />
        <SyncraticEvidenceList label="Supporting artifacts" ids={summary.supportingArtifactIds} />
        <SyncraticEvidenceList label="Related graph proposals" ids={summary.relatedGraphProposalIds} />
        <SyncraticEvidenceList label="Related labels" ids={summary.relatedLabelIds} />
      </div>

      <JsonViewer label="Metrics" value={insight.metrics} />
      <JsonViewer label="Recommendation" value={insight.recommendation} />

      {/* Context window detail rendered in the same panel. */}
      <div className="rounded border border-gray-200 bg-gray-50 p-2">
        <div className="mb-1 flex flex-wrap items-center justify-between gap-2">
          <div className="text-xs font-semibold text-gray-700">Context Window</div>
          {summary.contextWindowId && <CopyButton value={summary.contextWindowId} />}
        </div>
        {contextWindow.isLoading ? (
          <p className="text-xs text-gray-500">Loading context window...</p>
        ) : contextWindow.isError ? (
          <p className="text-xs text-amber-700">Context window unavailable.</p>
        ) : contextWindow.data && cwSummary ? (
          <ContextWindowBody cw={contextWindow.data.context_window} summary={cwSummary} />
        ) : (
          <p className="text-xs text-gray-500">No context window id on this insight.</p>
        )}
      </div>
    </div>
  );
}

function ContextWindowBody({
  cw,
  summary,
}: {
  cw: SyncraticContextWindow;
  summary: ReturnType<typeof summarizeSyncraticContextWindow>;
}) {
  return (
    <div className="space-y-2">
      <div className="grid grid-cols-2 gap-2 text-sm">
        <div><div className="text-xs text-gray-500">Strategy</div><div className="break-all text-xs">{summary.contextStrategy || '—'}</div></div>
        <div><div className="text-xs text-gray-500">Builder</div><div className="break-all text-xs font-mono">{summary.contextBuilderVersion || '—'}</div></div>
        <div><div className="text-xs text-gray-500">Window</div><div className="text-xs">{formatUtc(summary.windowStart)} → {formatUtc(summary.windowEnd)}</div></div>
        <div><div className="text-xs text-gray-500">Status</div><div className="text-xs">{summary.status || '—'}</div></div>
      </div>
      <div className="grid grid-cols-2 gap-2 text-sm">
        <div>
          <div className="text-xs text-gray-500">Evidence digest</div>
          <code className="break-all text-xs text-gray-700">{summary.evidenceDigest || '—'}</code>
        </div>
        <div>
          <div className="text-xs text-gray-500">Idempotency key</div>
          <code className="break-all text-xs text-gray-700">{summary.idempotencyKey || '—'}</code>
        </div>
      </div>
      <div className="flex flex-wrap gap-1">
        {[['Signals', summary.signalCount], ['Alerts', summary.alertCount], ['Events', summary.eventCount], ['Artifacts', summary.artifactCount], ['Graph', summary.graphProposalCount], ['Labels', summary.labelCount]].map(
          ([label, n]) => (
            <span key={label as string} className="inline-flex items-center gap-1 rounded border border-gray-200 bg-white px-1.5 py-0.5 text-[11px] text-gray-700">
              {label} <span className="font-semibold">{n as number}</span>
            </span>
          ),
        )}
      </div>
      {summary.signalTypes.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {summary.signalTypes.map((t) => (
            <code key={t} className="break-all rounded border border-gray-200 bg-white px-1.5 py-0.5 text-[11px] text-gray-700">{t}</code>
          ))}
        </div>
      )}
      <JsonViewer label="Summary metrics" value={cw.summary_metrics} />
    </div>
  );
}

function SyncraticEvidenceList({ label, ids }: { label: string; ids: string[] }) {
  if (!ids.length) return null;
  return (
    <div>
      <div className="mb-0.5 flex items-center gap-2">
        <span className="text-xs font-medium text-gray-600">{label}</span>
        <span className="rounded border border-gray-200 px-1.5 text-[11px] text-gray-600">{ids.length}</span>
        <CopyButton value={ids.join(', ')} />
      </div>
      <code className="break-all text-xs text-gray-700">{ids.join(', ')}</code>
    </div>
  );
}

// Bounded, operator-triggered materialization. Never runs automatically on page
// load. Caps are shown before submit; skip counters render as normal outcomes.
function SyncraticMaterializeForm({ tenantId }: { tenantId: string }) {
  const materialize = useMaterializeSyncraticContexts();
  // Default window: last ~13 days, UTC wall-clock.
  const [windowStart, setWindowStart] = useState(() => {
    const d = new Date();
    d.setUTCDate(d.getUTCDate() - 13);
    return toDatetimeLocal(d.toISOString());
  });
  const [windowEnd, setWindowEnd] = useState(() => toDatetimeLocal(new Date().toISOString()));
  const [minEvidenceCount, setMinEvidenceCount] = useState(2);
  const [maxAssets, setMaxAssets] = useState(50);
  const [maxContextWindows, setMaxContextWindows] = useState(10);
  const [maxInsights, setMaxInsights] = useState(10);

  const inputCls = 'rounded border border-gray-300 px-2 py-1 text-sm';
  const labelCls = 'text-xs text-gray-500';
  const canSubmit =
    !materialize.isPending &&
    windowStart.trim() !== '' &&
    windowEnd.trim() !== '';

  function onSubmit(ev: React.FormEvent) {
    ev.preventDefault();
    if (!canSubmit) return;
    materialize.mutate({
      tenant_id: tenantId,
      universe_group: MATERIALIZE_UNIVERSE_GROUP,
      context_strategy: MATERIALIZE_STRATEGY,
      context_builder_version: MATERIALIZE_BUILDER_VERSION,
      window_start: toRfc3339Utc(windowStart),
      window_end: toRfc3339Utc(windowEnd),
      min_evidence_count: minEvidenceCount,
      max_assets: maxAssets,
      max_candidate_windows: MATERIALIZE_MAX_CANDIDATE_WINDOWS,
      max_context_windows: maxContextWindows,
      max_insights: maxInsights,
    });
  }

  const counters = materialize.data ? summarizeSyncraticMaterialization(materialize.data.materialization) : [];

  return (
    <form onSubmit={onSubmit} className="rounded border border-gray-200 bg-white p-3" aria-label="Materialize Syncratic contexts">
      <div className="mb-2 flex items-center gap-1 text-sm font-semibold text-gray-900">
        <Sparkles size={14} /> Materialize Contexts
        <span className="ml-1 text-xs font-normal text-gray-500">
          Bounded synthesis over {MATERIALIZE_UNIVERSE_GROUP} · {MATERIALIZE_STRATEGY} · operator-triggered only
        </span>
      </div>
      <div className="grid grid-cols-2 gap-2 md:grid-cols-4 lg:grid-cols-6">
        <label className="block">
          <span className={labelCls}>Window start</span>
          <input
            type="datetime-local"
            value={windowStart}
            onChange={(e) => { materialize.reset(); setWindowStart(e.target.value); }}
            className={`${inputCls} mt-0.5 w-full`}
            aria-label="Materialize window start"
          />
        </label>
        <label className="block">
          <span className={labelCls}>Window end</span>
          <input
            type="datetime-local"
            value={windowEnd}
            onChange={(e) => { materialize.reset(); setWindowEnd(e.target.value); }}
            className={`${inputCls} mt-0.5 w-full`}
            aria-label="Materialize window end"
          />
        </label>
        <label className="block">
          <span className={labelCls}>Min evidence</span>
          <input
            type="number"
            min={1}
            value={minEvidenceCount}
            onChange={(e) => { materialize.reset(); setMinEvidenceCount(Number(e.target.value)); }}
            className={`${inputCls} mt-0.5 w-full`}
            aria-label="Minimum evidence count"
          />
        </label>
        <label className="block">
          <span className={labelCls}>Max assets</span>
          <input
            type="number"
            min={1}
            max={5000}
            value={maxAssets}
            onChange={(e) => { materialize.reset(); setMaxAssets(Number(e.target.value)); }}
            className={`${inputCls} mt-0.5 w-full`}
            aria-label="Max assets"
          />
        </label>
        <label className="block">
          <span className={labelCls}>Max context windows</span>
          <input
            type="number"
            min={1}
            max={5000}
            value={maxContextWindows}
            onChange={(e) => { materialize.reset(); setMaxContextWindows(Number(e.target.value)); }}
            className={`${inputCls} mt-0.5 w-full`}
            aria-label="Max context windows"
          />
        </label>
        <label className="block">
          <span className={labelCls}>Max insights</span>
          <input
            type="number"
            min={1}
            max={5000}
            value={maxInsights}
            onChange={(e) => { materialize.reset(); setMaxInsights(Number(e.target.value)); }}
            className={`${inputCls} mt-0.5 w-full`}
            aria-label="Max insights"
          />
        </label>
      </div>
      <div className="mt-2 flex flex-wrap items-center gap-2">
        <button
          type="submit"
          disabled={!canSubmit}
          className="inline-flex items-center gap-1 rounded bg-brand-500 px-3 py-1.5 text-sm text-white hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-50"
        >
          <Sparkles size={14} /> {materialize.isPending ? 'Materializing...' : 'Materialize Contexts'}
        </button>
        <span className="text-xs text-gray-500">
          Caps: {maxAssets} assets · {maxContextWindows} windows · {maxInsights} insights
        </span>
      </div>

      {materialize.isSuccess && counters.length > 0 && (
        <div className="mt-2 flex flex-wrap gap-1.5">
          {counters.map((c) => (
            <span
              key={c.key}
              title={c.label}
              className={`inline-flex items-center gap-1 rounded border px-2 py-0.5 text-xs ${
                c.kind === 'skipped'
                  ? 'border-gray-200 bg-gray-50 text-gray-600'
                  : c.kind === 'materialized'
                    ? 'border-emerald-200 bg-emerald-50 text-emerald-700'
                    : 'border-gray-300 bg-white text-gray-800'
              }`}
            >
              {c.label} <span className="font-semibold">{c.value}</span>
            </span>
          ))}
        </div>
      )}
      {materialize.isError && (
        <p className="mt-2 text-xs text-red-700" role="alert">
          Materialize failed: {isApiError(materialize.error) ? materialize.error.message : 'unknown error'}. Form values preserved.
        </p>
      )}
    </form>
  );
}
