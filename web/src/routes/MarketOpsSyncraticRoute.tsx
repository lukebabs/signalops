import { useState } from 'react';
import { Sparkles, AlertTriangle, RefreshCw } from 'lucide-react';
import {
  useSyncraticInsights,
  useSyncraticInsight,
  useSyncraticContextWindow,
  useMaterializeSyncraticContexts,
  useAskSyncraticContextWindow,
} from '../api/queries';
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
  summarizeSyncraticAsk,
  summarizeSyncraticAskRouteResult,
  detectSyncraticDataQualityWarning,
  classifySyncraticInsightBadge,
  syncraticSeverityStyle,
  syncraticInsightStatusStyle,
  syncraticCurrentnessLabel,
  syncraticCurrentnessStyle,
  shortSyncraticId,
  sortSyncraticMaterializationDecisions,
  countSyncraticMaterializationDecisions,
  syncraticDecisionStyle,
  messageForSyncraticAskError,
  messageForSyncraticMaterializeError,
  SYNCRATIC_ASK_BADGE_LABELS,
  SYNCRATIC_ASK_BADGE_STYLES,
} from '../lib/syncratic';
import { useTenant } from '../auth/session';
import { useAppProfile } from '../apps/AppProfileContext';
import type { SyncraticInsight, SyncraticContextWindow, SyncraticInsightStatus, SyncraticMaterializeRequest } from '../types';

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

// G090 compact source/badge chip. Distinguishes deterministic SignalOps context
// from Ask-enriched, and flags data-quality-blocked results so they never read
// as a valid market thesis. Ask-skipped is transient (latest route result only).
function SyncraticBadgeChip({ badge }: { badge: ReturnType<typeof classifySyncraticInsightBadge> }) {
  return (
    <span
      className={`inline-flex shrink-0 items-center gap-1 whitespace-nowrap rounded border px-1.5 py-0.5 text-[11px] font-medium ${SYNCRATIC_ASK_BADGE_STYLES[badge]}`}
    >
      {badge === 'data_quality' && <AlertTriangle size={11} />}
      {SYNCRATIC_ASK_BADGE_LABELS[badge]}
    </span>
  );
}


function SyncraticCurrentnessChip({ summary }: { summary: ReturnType<typeof summarizeSyncraticInsight> }) {
  return (
    <span
      className={`inline-flex items-center rounded border px-1.5 py-0.5 text-[11px] font-medium ${syncraticCurrentnessStyle(summary.isCurrent)}`}
      title={summary.isCurrent ? 'Current preferred context for this symbol/strategy' : `Historical context superseded by ${summary.supersededBySyncraticInsightId || 'a newer context'}`}
    >
      {syncraticCurrentnessLabel(summary)}
    </span>
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
                        <td className="max-w-[20rem] px-3 py-2">
                          <div className="flex items-center gap-1.5">
                            <SyncraticBadgeChip badge={classifySyncraticInsightBadge(i)} />
                            <SyncraticCurrentnessChip summary={s} />
                            <span className="min-w-0 truncate text-xs font-medium text-gray-800" title={s.title || s.insightId}>
                              {s.title || s.insightId}
                            </span>
                          </div>
                          {s.summary && (
                            <div className="mt-0.5 truncate text-xs text-gray-500" title={s.summary}>
                              {s.summary}
                            </div>
                          )}
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
            <SyncraticInsightDetail
              key={selectedId}
              insight={selected}
              summary={selectedSummary}
              tenantId={TENANT_ID}
            />
          ) : null}
        </div>
      </div>
    </div>
  );
}

function SyncraticInsightDetail({
  insight,
  summary,
  tenantId,
}: {
  insight: SyncraticInsight;
  summary: ReturnType<typeof summarizeSyncraticInsight>;
  tenantId: string;
}) {
  // Fetch the context window detail by context_window_id (read-only review).
  const contextWindow = useSyncraticContextWindow(summary.contextWindowId || null);
  const cwSummary = contextWindow.data ? summarizeSyncraticContextWindow(contextWindow.data.context_window) : null;

  // G090 Ask enrichment state. `ask` is read from persisted metrics; `askMutation`
  // carries the latest operator-triggered route result (transient skip/success).
  const askMutation = useAskSyncraticContextWindow();
  const ask = summarizeSyncraticAsk(insight);
  const dataQuality = detectSyncraticDataQualityWarning(insight);
  const latestRoute = askMutation.data?.ask_result
    ? summarizeSyncraticAskRouteResult(askMutation.data.ask_result)
    : null;
  const badge = classifySyncraticInsightBadge(insight, latestRoute?.askStatus);

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        <SyncraticBadgeChip badge={badge} />
        <SyncraticCurrentnessChip summary={summary} />
        <StatusLabel status={summary.status} />
        <SeverityLabel severity={summary.severity} />
        <span className="text-xs text-gray-600">conf {summary.confidence.toFixed(2)}</span>
        <code className="break-all text-xs text-gray-700">{summary.insightId}</code>
        <CopyButton value={summary.insightId} />
        {!summary.isCurrent && summary.supersededBySyncraticInsightId && (
          <span className="text-[11px] text-gray-500">
            superseded by {shortSyncraticId(summary.supersededBySyncraticInsightId)}
          </span>
        )}
      </div>

      {dataQuality && (
        <div className="flex items-start gap-2 rounded border border-red-300 bg-red-50 p-2" role="alert">
          <AlertTriangle size={14} className="mt-0.5 shrink-0 text-red-700" />
          <div>
            <div className="text-xs font-semibold text-red-800">Data Quality Warning</div>
            <div className="text-[11px] text-red-700">
              Evidence does not support this context subject. Treat this as an evidence-quality issue, not a market insight.
            </div>
          </div>
        </div>
      )}

      <div>
        <div className="text-sm font-medium text-gray-900">{summary.title}</div>
        {summary.summary && <div className="mt-0.5 text-xs text-gray-600">{summary.summary}</div>}
      </div>
      {summary.explanation && (
        <div>
          <div className="mb-1 flex flex-wrap items-center gap-2">
            <span className="text-xs font-semibold text-gray-700">
              {ask.present ? 'Ask Explanation' : 'Deterministic Explanation'}
            </span>
            {ask.present && (
              <span className="text-[11px] text-gray-500">
                generated by Syncratic Ask over the bounded context window
              </span>
            )}
          </div>
          <p
            className={`whitespace-pre-wrap break-words rounded border p-2 text-xs ${
              dataQuality
                ? 'border-red-300 bg-red-50 text-red-800'
                : ask.present
                  ? 'border-brand-200 bg-brand-50 text-gray-800'
                  : 'border-gray-200 bg-gray-50 text-gray-700'
            }`}
          >
            {summary.explanation}
          </p>
          {ask.present && (
            <div className="mt-0.5 text-[11px] text-gray-500">
              SignalOps deterministic context and evidence references are shown below. This explanation is Syncratic Ask output, not deterministic synthesis.
            </div>
          )}
        </div>
      )}

      {ask.present && (
        <div className="rounded border border-brand-100 bg-brand-50/40 p-2">
          <div className="mb-1 text-xs font-semibold text-gray-700">Syncratic Ask metadata</div>
          <div className="grid grid-cols-2 gap-2">
            {([
              ['Ask status', ask.askStatus || '—'],
              ['Direct reasoning', ask.directReasoning ? 'on' : 'off'],
              ['Graph', ask.graphEnabled ? 'on' : 'off'],
              ['KEE', ask.keeEnabled ? 'on' : 'off'],
              ['Prompt bytes', orDash(ask.promptBytes)],
              ['Latency', ask.latencyMs ? `${ask.latencyMs} ms` : '—'],
              ['Confidence', ask.responseConfidence ? ask.responseConfidence.toFixed(2) : '—'],
              ['Evidence / citations', `${orDash(ask.responseEvidenceCount)} / ${orDash(ask.responseCitationCount)}`],
            ] as const).map(([label, value]) => (
              <div key={label}>
                <div className="text-xs text-gray-500">{label}</div>
                <div className="text-xs">{value}</div>
              </div>
            ))}
          </div>
          <div className="mt-1">
            <div className="text-xs text-gray-500">Prompt builder</div>
            <code className="break-all text-xs font-mono text-gray-700">{ask.promptBuilderVersion || '—'}</code>
          </div>
          {latestRoute && (
            <div className="mt-1 text-[11px] text-gray-500">
              Last route: {latestRoute.updated ? 'updated' : 'skipped'}
              {latestRoute.skippedReason ? ` · ${latestRoute.skippedReason}` : ''}
            </div>
          )}
        </div>
      )}

      <SyncraticAskControls
        contextWindowId={summary.contextWindowId}
        tenantId={tenantId}
        askMutation={askMutation}
      />

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

// G090 operator-triggered Syncratic Ask controls. Never runs automatically —
// only on explicit click. Ask sends force=false (skips on unchanged prompt +
// evidence); Regenerate sends force=true. Disabled while pending. Errors are
// sanitized server-side and here (no raw prompt, tokens, or upstream bodies).
function SyncraticAskControls({
  contextWindowId,
  tenantId,
  askMutation,
}: {
  contextWindowId: string;
  tenantId: string;
  askMutation: ReturnType<typeof useAskSyncraticContextWindow>;
}) {
  if (!contextWindowId || !tenantId) return null;
  const disabled = askMutation.isPending;

  function trigger(force: boolean) {
    askMutation.mutate({
      contextWindowId,
      request: { tenant_id: tenantId, max_prompt_bytes: 12000, force },
    });
  }

  return (
    <div className="rounded border border-gray-200 bg-white p-2">
      <div className="mb-1 flex items-center gap-1 text-xs font-semibold text-gray-700">
        <Sparkles size={12} /> Ask Syncratic AI
      </div>
      <p className="mb-2 text-[11px] text-gray-500">
        Exploratory analyst drill-down over the official EOD context. It never changes the official daily overview.
        Ask sends force=false; Regenerate sends force=true.
      </p>
      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          onClick={() => trigger(false)}
          disabled={disabled}
          className="inline-flex items-center gap-1 rounded bg-brand-500 px-2.5 py-1 text-xs text-white hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-50"
        >
          <Sparkles size={12} /> {disabled ? 'Asking…' : 'Ask Syncratic AI'}
        </button>
        <button
          type="button"
          onClick={() => trigger(true)}
          disabled={disabled}
          className="inline-flex items-center gap-1 rounded border border-gray-300 bg-white px-2.5 py-1 text-xs text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
        >
          <RefreshCw size={12} /> Regenerate
        </button>
      </div>

      {askMutation.isSuccess && askMutation.data?.ask_result && (
        <AskResultBanner
          status={askMutation.data.ask_result.ask_status}
          updated={askMutation.data.ask_result.updated}
          skippedReason={askMutation.data.ask_result.skipped_reason}
        />
      )}
      {askMutation.isError && (
        <p className="mt-2 text-xs text-red-700" role="alert">
          {messageForSyncraticAskError(askMutation.error)}
        </p>
      )}
    </div>
  );
}

// Transient route-result banner. A skip (updated=false / ask_status=skipped) is a
// normal outcome, rendered as amber info — never red.
function AskResultBanner({
  status,
  updated,
  skippedReason,
}: {
  status: string;
  updated: boolean;
  skippedReason: string;
}) {
  if (status === 'skipped' || !updated) {
    return (
      <p className="mt-2 rounded border border-amber-200 bg-amber-50 px-2 py-1 text-xs text-amber-700">
        Skipped — {skippedReason || 'unchanged prompt and evidence'}. No insight update was written.
      </p>
    );
  }
  return (
    <p className="mt-2 rounded border border-emerald-200 bg-emerald-50 px-2 py-1 text-xs text-emerald-700">
      Ask completed. Insight explanation refreshed.
    </p>
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
  // Safe preview/write defaults: small asset scan and a single context/insight cap.
  const [maxAssets, setMaxAssets] = useState(10);
  const [maxContextWindows, setMaxContextWindows] = useState(1);
  const [maxInsights, setMaxInsights] = useState(1);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [minEvidenceCount, setMinEvidenceCount] = useState(2);
  const [maxCandidateWindows, setMaxCandidateWindows] = useState(MATERIALIZE_MAX_CANDIDATE_WINDOWS);
  const [universeGroup, setUniverseGroup] = useState(MATERIALIZE_UNIVERSE_GROUP);
  const [contextStrategy, setContextStrategy] = useState(MATERIALIZE_STRATEGY);
  // Preview-first gating: write is disabled until a dry-run has succeeded, and
  // needs a two-click confirm so context/insight rows are never created by accident.
  const [previewDone, setPreviewDone] = useState(false);
  const [confirmWrite, setConfirmWrite] = useState(false);

  const inputCls = 'rounded border border-gray-300 px-2 py-1 text-sm';
  const labelCls = 'text-xs text-gray-500';
  const canSubmit =
    !materialize.isPending &&
    windowStart.trim() !== '' &&
    windowEnd.trim() !== '';

  function buildRequest(dryRun: boolean): SyncraticMaterializeRequest {
    return {
      tenant_id: tenantId,
      universe_group: universeGroup,
      context_strategy: contextStrategy,
      context_builder_version: MATERIALIZE_BUILDER_VERSION,
      window_start: toRfc3339Utc(windowStart),
      window_end: toRfc3339Utc(windowEnd),
      min_evidence_count: minEvidenceCount,
      max_assets: maxAssets,
      max_candidate_windows: maxCandidateWindows,
      max_context_windows: maxContextWindows,
      max_insights: maxInsights,
      dry_run: dryRun,
    };
  }

  // Any input change clears the stale result and re-arms the preview gate, so the
  // write action always reflects a fresh preview of the current parameters.
  function resetTransient<T>(setter: (v: T) => void) {
    return (v: T) => {
      materialize.reset();
      setPreviewDone(false);
      setConfirmWrite(false);
      setter(v);
    };
  }

  // Primary action: dry-run preview (form submit). Writes nothing.
  function onPreview(ev: React.FormEvent) {
    ev.preventDefault();
    if (!canSubmit) return;
    setConfirmWrite(false);
    materialize.mutate(buildRequest(true), { onSuccess: () => setPreviewDone(true) });
  }

  // Secondary action: confirmed write (dry_run=false). First click arms the
  // confirm; second click performs the write with the last preview's parameters.
  function onWrite() {
    if (!canSubmit || !previewDone) return;
    if (!confirmWrite) {
      setConfirmWrite(true);
      return;
    }
    materialize.mutate(buildRequest(false), {
      onSuccess: () => setConfirmWrite(false),
      // Disarm the confirm on failure so the operator re-confirms intent after
      // reading the error, rather than one-click retrying a write.
      onError: () => setConfirmWrite(false),
    });
  }

  const result = materialize.data?.materialization ?? null;
  const counters = result ? summarizeSyncraticMaterialization(result) : [];
  const decisionCounts = result ? countSyncraticMaterializationDecisions(result.decisions) : null;
  const sortedDecisions = result ? sortSyncraticMaterializationDecisions(result.decisions) : [];
  const isDryRun = result?.dry_run ?? false;
  // In-flight request mode (from the pending mutation's own variables), used for
  // accurate pending labels — distinct from `isDryRun`, which reflects the last
  // *completed* result and would mislabel the first request.
  const pendingDryRun = materialize.variables?.dry_run;
  const writeCreated =
    !!result && !isDryRun && (result.materialized_context_windows > 0 || result.materialized_insights > 0);
  const idempotentUnchanged =
    !!result &&
    !isDryRun &&
    result.materialized_context_windows === 0 &&
    result.materialized_insights === 0 &&
    result.skipped_unchanged > 0;

  return (
    <form onSubmit={onPreview} className="rounded border border-gray-200 bg-white p-3" aria-label="Materialize Syncratic contexts">
      <div className="mb-2 flex items-center gap-1 text-sm font-semibold text-gray-900">
        <Sparkles size={14} /> Materialize Contexts
        <span className="ml-1 text-xs font-normal text-gray-500">
          Preview-first · bounded synthesis over {universeGroup} · {contextStrategy} · operator-triggered only
        </span>
      </div>
      <p className="-mt-1 mb-2 text-[11px] text-gray-500">
        Subject-scoped contexts require evidence that matches the selected ticker. Evidence mentioning another known ticker is excluded (purity filtering).
      </p>
      <div className="grid grid-cols-2 gap-2 md:grid-cols-3 lg:grid-cols-5">
        <label className="block">
          <span className={labelCls}>Window start</span>
          <input
            type="datetime-local"
            value={windowStart}
            onChange={(e) => resetTransient(setWindowStart)(e.target.value)}
            className={`${inputCls} mt-0.5 w-full`}
            aria-label="Materialize window start"
          />
        </label>
        <label className="block">
          <span className={labelCls}>Window end</span>
          <input
            type="datetime-local"
            value={windowEnd}
            onChange={(e) => resetTransient(setWindowEnd)(e.target.value)}
            className={`${inputCls} mt-0.5 w-full`}
            aria-label="Materialize window end"
          />
        </label>
        <label className="block">
          <span className={labelCls}>Max assets</span>
          <input
            type="number"
            min={1}
            max={5000}
            value={maxAssets}
            onChange={(e) => resetTransient(setMaxAssets)(Number(e.target.value))}
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
            onChange={(e) => resetTransient(setMaxContextWindows)(Number(e.target.value))}
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
            onChange={(e) => resetTransient(setMaxInsights)(Number(e.target.value))}
            className={`${inputCls} mt-0.5 w-full`}
            aria-label="Max insights"
          />
        </label>
      </div>

      <button
        type="button"
        onClick={() => setShowAdvanced((v) => !v)}
        className="mt-1 text-[11px] text-brand-700 hover:underline"
        aria-expanded={showAdvanced}
      >
        {showAdvanced ? '▾ Hide advanced' : '▸ Advanced'}
      </button>
      {showAdvanced && (
        <div className="mt-1 grid grid-cols-2 gap-2 md:grid-cols-4">
          <label className="block">
            <span className={labelCls}>Min evidence</span>
            <input
              type="number"
              min={1}
              value={minEvidenceCount}
              onChange={(e) => resetTransient(setMinEvidenceCount)(Number(e.target.value))}
              className={`${inputCls} mt-0.5 w-full`}
              aria-label="Minimum evidence count"
            />
          </label>
          <label className="block">
            <span className={labelCls}>Max candidate windows</span>
            <input
              type="number"
              min={1}
              max={5000}
              value={maxCandidateWindows}
              onChange={(e) => resetTransient(setMaxCandidateWindows)(Number(e.target.value))}
              className={`${inputCls} mt-0.5 w-full`}
              aria-label="Max candidate windows"
            />
          </label>
          <label className="block">
            <span className={labelCls}>Universe group</span>
            <input
              type="text"
              value={universeGroup}
              onChange={(e) => resetTransient(setUniverseGroup)(e.target.value)}
              className={`${inputCls} mt-0.5 w-full`}
              aria-label="Universe group"
            />
          </label>
          <label className="block">
            <span className={labelCls}>Context strategy</span>
            <input
              type="text"
              value={contextStrategy}
              onChange={(e) => resetTransient(setContextStrategy)(e.target.value)}
              className={`${inputCls} mt-0.5 w-full`}
              aria-label="Context strategy"
            />
          </label>
        </div>
      )}

      <div className="mt-2 flex flex-wrap items-center gap-2">
        <button
          type="submit"
          disabled={!canSubmit}
          className="inline-flex items-center gap-1 rounded bg-brand-500 px-3 py-1.5 text-sm text-white hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-50"
        >
          <Sparkles size={14} /> {materialize.isPending && pendingDryRun === true ? 'Previewing…' : 'Preview materialization'}
        </button>
        <button
          type="button"
          onClick={onWrite}
          disabled={!canSubmit || !previewDone}
          className="inline-flex items-center gap-1 rounded border border-gray-300 bg-white px-3 py-1.5 text-sm text-gray-800 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
          title={previewDone ? 'Create context windows and insights with the previewed budget' : 'Run a preview first'}
        >
          {materialize.isPending && pendingDryRun === false
            ? 'Materializing…'
            : confirmWrite
              ? 'Confirm write — create contexts/insights'
              : 'Materialize selected budget'}
        </button>
        <span className="text-xs text-gray-500">
          Caps: {maxAssets} assets · {maxContextWindows} windows · {maxInsights} insights
        </span>
      </div>

      {confirmWrite && (
        <p className="mt-1 text-[11px] text-amber-700">
          Write mode will create or update up to {maxContextWindows} context window(s) and {maxInsights} insight(s). Click again to confirm.
        </p>
      )}

      {result && (
        <MaterializeResultPanel
          isDryRun={isDryRun}
          counters={counters}
          wouldMaterialize={decisionCounts?.would_materialize ?? 0}
          writeCreated={writeCreated}
          idempotentUnchanged={idempotentUnchanged}
          skippedUnchanged={result.skipped_unchanged}
          materializedContextWindows={result.materialized_context_windows}
          materializedInsights={result.materialized_insights}
          decisions={sortedDecisions}
        />
      )}

      {materialize.isError && (
        <p className="mt-2 text-xs text-red-700" role="alert">
          {messageForSyncraticMaterializeError(materialize.error)} Form values preserved.
        </p>
      )}
    </form>
  );
}

// Aggregate counters + mode banner + per-asset decision table for a preview or
// write result. Zero materializations on a successful response is not an error.
function MaterializeResultPanel({
  isDryRun,
  counters,
  wouldMaterialize,
  writeCreated,
  idempotentUnchanged,
  skippedUnchanged,
  materializedContextWindows,
  materializedInsights,
  decisions,
}: {
  isDryRun: boolean;
  counters: ReturnType<typeof summarizeSyncraticMaterialization>;
  wouldMaterialize: number;
  writeCreated: boolean;
  idempotentUnchanged: boolean;
  skippedUnchanged: number;
  materializedContextWindows: number;
  materializedInsights: number;
  decisions: ReturnType<typeof sortSyncraticMaterializationDecisions>;
}) {
  return (
    <div className="mt-3 space-y-2">
      {/* Mode banner: distinguishes preview (no writes) from write outcomes. */}
      <p
        className={`rounded border px-2 py-1 text-xs ${
          writeCreated
            ? 'border-emerald-200 bg-emerald-50 text-emerald-700'
            : idempotentUnchanged
              ? 'border-gray-200 bg-gray-50 text-gray-600'
              : 'border-brand-100 bg-brand-50/50 text-gray-700'
        }`}
      >
        {isDryRun
          ? 'Dry-run preview — no context windows or insights were written.'
          : writeCreated
            ? `Write complete — ${materializedContextWindows} context window(s), ${materializedInsights} insight(s) created.`
            : idempotentUnchanged
              ? `Idempotent success — evidence unchanged, no duplicates written (skipped_unchanged=${skippedUnchanged}).`
              : 'Write complete — no rows created for this window.'}
      </p>

      {counters.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {isDryRun && (
            <span className="inline-flex items-center gap-1 rounded border border-brand-200 bg-brand-50 px-2 py-0.5 text-xs text-brand-700">
              Would materialize <span className="font-semibold">{wouldMaterialize}</span>
            </span>
          )}
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

      {decisions.length > 0 && (
        <div className="overflow-x-auto rounded border border-gray-200">
          <table className="min-w-full divide-y divide-gray-200 text-[11px]">
            <thead className="bg-gray-50 text-left uppercase tracking-wide text-gray-500">
              <tr>
                <th className="whitespace-nowrap px-2 py-1">Symbol</th>
                <th className="whitespace-nowrap px-2 py-1">Action</th>
                <th className="whitespace-nowrap px-2 py-1">Reason</th>
                <th className="whitespace-nowrap px-2 py-1">Evidence</th>
                <th className="whitespace-nowrap px-2 py-1">Signals</th>
                <th className="whitespace-nowrap px-2 py-1">Alerts</th>
                <th className="whitespace-nowrap px-2 py-1">Artifacts</th>
                <th className="whitespace-nowrap px-2 py-1">Graph</th>
                <th className="whitespace-nowrap px-2 py-1">Labels</th>
                <th className="whitespace-nowrap px-2 py-1">Critical</th>
                <th className="whitespace-nowrap px-2 py-1">Related</th>
                <th className="whitespace-nowrap px-2 py-1">Context</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {decisions.map((d, idx) => (
                <tr key={d.contextWindowId || d.subjectSymbol || idx} className="align-top">
                  <td className="whitespace-nowrap px-2 py-1 font-semibold text-gray-900">{d.subjectSymbol || '—'}</td>
                  <td className="px-2 py-1">
                    <span
                      className={`inline-flex shrink-0 items-center whitespace-nowrap rounded border px-1.5 py-0.5 font-medium ${syncraticDecisionStyle(
                        d.action,
                        d.reason,
                      )}`}
                    >
                      {d.action || '—'}
                    </span>
                  </td>
                  <td className="whitespace-nowrap px-2 py-1 text-gray-600">{d.reason || '—'}</td>
                  <td className="whitespace-nowrap px-2 py-1">{d.evidenceCount}</td>
                  <td className="whitespace-nowrap px-2 py-1">{d.signalCount}</td>
                  <td className="whitespace-nowrap px-2 py-1">{d.alertCount}</td>
                  <td className="whitespace-nowrap px-2 py-1">{d.artifactCount}</td>
                  <td className="whitespace-nowrap px-2 py-1">{d.graphProposalCount}</td>
                  <td className="whitespace-nowrap px-2 py-1">{d.labelCount}</td>
                  <td className="whitespace-nowrap px-2 py-1">{d.criticalAlert ? 'yes' : 'no'}</td>
                  <td className="whitespace-nowrap px-2 py-1">{d.relatedEvidence ? 'yes' : 'no'}</td>
                  <td className="px-2 py-1">
                    <code className="text-gray-600" title={d.contextWindowId}>
                      {d.contextWindowId ? shortSyncraticId(d.contextWindowId) : '—'}
                    </code>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
