import { useState } from 'react';
import { Link } from '@tanstack/react-router';
import { Network } from 'lucide-react';
import { useSignals, useAlerts, useInsights, useSignal } from '../api/queries';
import { LoadingState, ErrorState, EmptyState } from '../components/States';
import { MetricTile } from '../components/MetricTile';
import { CopyButton } from '../components/CopyButton';
import { JsonViewer } from '../components/JsonViewer';
import { RefreshButton } from '../components/RefreshButton';
import { formatUtc } from '../lib/format';
import { useTenant } from '../auth/session';
import {
  MARKETOPS_DSM_DETECTOR_ID,
  MARKETOPS_DSM_USE_CASE,
  MARKETOPS_DSM_SIGNAL_TYPES,
  dsmShortType,
  dsmFamily,
  getTicker,
  getMetric,
  getArtifactProposal,
  getArtifactId,
  graphTargetCounts,
  hasLifecycleMatch,
  type DsmFamily,
} from '../lib/marketopsDsm';
import type { SignalRecord, AlertRecord, InsightRecord } from '../types';

const SEVERITIES = ['info', 'low', 'medium', 'high', 'critical'] as const;
const DATASETS = ['equity_eod_prices', 'options_contracts_daily'] as const;
const LIMITS = [25, 50, 100, 200];

const SEVERITY_STYLES: Record<string, string> = {
  critical: 'text-red-700',
  high: 'text-orange-700',
  medium: 'text-amber-700',
  low: 'text-gray-700',
  info: 'text-gray-500',
};

function SeverityLabel({ severity }: { severity: string }) {
  return <span className={`text-xs font-medium ${SEVERITY_STYLES[severity] ?? 'text-gray-600'}`}>{severity}</span>;
}

const FAMILY_STYLES: Record<DsmFamily, string> = {
  equity: 'text-blue-700',
  option: 'text-violet-700',
  quality: 'text-amber-700',
  unknown: 'text-gray-500',
};

type MetricSpec = { key: string; label: string; suffix?: string };
const PRICE_METRICS: MetricSpec[] = [
  { key: 'open_close_move_pct', label: 'Open/Close Move', suffix: '%' },
  { key: 'intraday_range_pct', label: 'Intraday Range', suffix: '%' },
  { key: 'daily_return_pct', label: 'Daily Return', suffix: '%' },
  { key: 'vwap_distance_pct', label: 'VWAP Distance', suffix: '%' },
  { key: 'volume', label: 'Volume' },
];
const OPTION_METRICS: MetricSpec[] = [
  { key: 'open_interest', label: 'Open Interest' },
  { key: 'volume_open_interest_ratio', label: 'Vol/OI' },
  { key: 'days_to_expiration', label: 'Days to Exp' },
  { key: 'moneyness_pct', label: 'Moneyness', suffix: '%' },
  { key: 'contract_type', label: 'Contract Type' },
];
const QUALITY_METRICS: MetricSpec[] = [
  { key: 'quality_issue_count', label: 'Quality Issues' },
  { key: 'detector_score', label: 'Detector Score' },
];

// Concise one-line metric summary for the table cell, chosen by family.
function keyMetricsSummary(signal: SignalRecord, family: DsmFamily): string {
  const m = (k: string) => getMetric(signal, k);
  const parts: string[] = [];
  if (family === 'equity') {
    if (m('open_close_move_pct') != null) parts.push(`move ${m('open_close_move_pct')}%`);
    if (m('intraday_range_pct') != null) parts.push(`range ${m('intraday_range_pct')}%`);
    if (m('daily_return_pct') != null) parts.push(`ret ${m('daily_return_pct')}%`);
    if (m('vwap_distance_pct') != null) parts.push(`vwap ${m('vwap_distance_pct')}%`);
  } else if (family === 'option') {
    if (m('open_interest') != null) parts.push(`OI ${m('open_interest')}`);
    if (m('volume_open_interest_ratio') != null) parts.push(`vol/OI ${m('volume_open_interest_ratio')}`);
    if (m('days_to_expiration') != null) parts.push(`DTE ${m('days_to_expiration')}`);
    if (m('moneyness_pct') != null) parts.push(`mny ${m('moneyness_pct')}%`);
    if (m('contract_type') != null) parts.push(String(m('contract_type')));
  } else if (family === 'quality') {
    if (m('quality_issue_count') != null) parts.push(`issues ${m('quality_issue_count')}`);
    if (m('detector_score') != null) parts.push(`score ${m('detector_score')}`);
  }
  return parts.join(' · ');
}

function MetricList({ signal, spec }: { signal: SignalRecord; spec: MetricSpec[] }) {
  const present = spec.filter((s) => getMetric(signal, s.key) != null);
  if (!present.length) return <p className="text-xs text-gray-400">None reported.</p>;
  return (
    <div className="grid grid-cols-2 gap-2">
      {present.map((s) => {
        const v = getMetric(signal, s.key);
        return (
          <div key={s.key}>
            <div className="text-xs text-gray-500">{s.label}</div>
            <div className="text-xs">
              {String(v)}
              {s.suffix ?? ''}
            </div>
          </div>
        );
      })}
    </div>
  );
}

export function MarketOpsDsmRoute() {
  const TENANT_ID = useTenant();
  const [taxonomy, setTaxonomy] = useState('');
  const [severity, setSeverity] = useState('');
  const [dataset, setDataset] = useState('');
  const [limit, setLimit] = useState(50);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const signalsQ = useSignals({
    tenant_id: TENANT_ID,
    app_id: 'marketops',
    domain: 'market_data',
    use_case: MARKETOPS_DSM_USE_CASE,
    detector_id: MARKETOPS_DSM_DETECTOR_ID,
    severity: severity || undefined,
    dataset: dataset || undefined,
    limit,
  });
  const alertsQ = useAlerts({
    tenant_id: TENANT_ID,
    app_id: 'marketops',
    domain: 'market_data',
    use_case: MARKETOPS_DSM_USE_CASE,
    status: 'open',
    limit: 100,
  });
  const insightsQ = useInsights({
    tenant_id: TENANT_ID,
    app_id: 'marketops',
    domain: 'market_data',
    use_case: MARKETOPS_DSM_USE_CASE,
    status: 'active',
    limit: 100,
  });
  const detailQ = useSignal(selectedId);

  // Taxonomy type is not a backend filter; apply it client-side.
  const raw = signalsQ.data?.signals ?? [];
  const signals = taxonomy ? raw.filter((s) => s.signal_type === taxonomy) : raw;

  const alertSignalIds = new Set(
    (alertsQ.data?.alerts ?? []).map((a) => a.signal_id).filter((v): v is string => typeof v === 'string'),
  );
  const insightSignalIds = new Set(
    (insightsQ.data?.insights ?? []).map((i) => i.signal_id).filter((v): v is string => typeof v === 'string'),
  );

  const highCritical = signals.filter((s) => s.severity === 'high' || s.severity === 'critical').length;
  const withAlert = signals.filter((s) => hasLifecycleMatch(s, alertSignalIds)).length;
  const withInsight = signals.filter((s) => hasLifecycleMatch(s, insightSignalIds)).length;
  const taxonomyTypes = new Set(signals.map((s) => s.signal_type)).size;

  const matchedAlert = (alertsQ.data?.alerts ?? []).find((a) => a.signal_id === selectedId) ?? null;
  const matchedInsight = (insightsQ.data?.insights ?? []).find((i) => i.signal_id === selectedId) ?? null;
  const selected: SignalRecord | null =
    detailQ.data?.signal ?? signals.find((s) => s.signal_id === selectedId) ?? null;

  function refresh() {
    signalsQ.refetch();
    alertsQ.refetch();
    insightsQ.refetch();
    if (selectedId) detailQ.refetch();
  }

  const inputCls = 'rounded border border-gray-300 px-2 py-1 text-sm';

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h1 className="flex items-center gap-1 text-lg font-semibold">
            <Network size={18} className="text-brand-700" /> DSM Workbench
          </h1>
          <p className="text-xs text-gray-500">marketops / market_data / daily_market_surveillance</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <select value={taxonomy} onChange={(e) => setTaxonomy(e.target.value)} className={inputCls} aria-label="Filter by taxonomy type">
            <option value="">all taxonomy</option>
            {MARKETOPS_DSM_SIGNAL_TYPES.map((t) => (
              <option key={t} value={t}>{dsmShortType(t)}</option>
            ))}
          </select>
          <select value={severity} onChange={(e) => setSeverity(e.target.value)} className={inputCls} aria-label="Filter by severity">
            <option value="">any severity</option>
            {SEVERITIES.map((s) => (
              <option key={s} value={s}>{s}</option>
            ))}
          </select>
          <select value={dataset} onChange={(e) => setDataset(e.target.value)} className={inputCls} aria-label="Filter by dataset">
            <option value="">any dataset</option>
            {DATASETS.map((d) => (
              <option key={d} value={d}>{d}</option>
            ))}
          </select>
          <select value={limit} onChange={(e) => setLimit(Number(e.target.value))} className={inputCls} aria-label="Page limit">
            {LIMITS.map((n) => (
              <option key={n} value={n}>{n}</option>
            ))}
          </select>
          <RefreshButton onClick={refresh} loading={signalsQ.isFetching || alertsQ.isFetching || insightsQ.isFetching} />
        </div>
      </div>

      <div className="grid grid-cols-2 gap-2 md:grid-cols-5">
        <MetricTile label="DSM Signals" value={signals.length} />
        <MetricTile label="High/Critical" value={highCritical} />
        <MetricTile label="Open Alerts" value={withAlert} hint={alertsQ.isError ? 'unreachable' : undefined} />
        <MetricTile label="Active Insights" value={withInsight} hint={insightsQ.isError ? 'unreachable' : undefined} />
        <MetricTile label="Taxonomy Types" value={taxonomyTypes} />
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <div className="lg:col-span-2">
          {signalsQ.isLoading ? (
            <LoadingState />
          ) : signalsQ.isError ? (
            <ErrorState error={signalsQ.error} />
          ) : signals.length ? (
            <div className="overflow-x-auto rounded border border-gray-200 bg-white">
              <table className="min-w-full divide-y divide-gray-200 text-sm">
                <thead className="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500">
                  <tr>
                    <th className="px-3 py-2">Ticker</th>
                    <th className="px-3 py-2">Taxonomy</th>
                    <th className="px-3 py-2">Family</th>
                    <th className="px-3 py-2">Severity</th>
                    <th className="px-3 py-2">Conf.</th>
                    <th className="px-3 py-2">Dataset</th>
                    <th className="px-3 py-2">Key Metrics</th>
                    <th className="px-3 py-2">Artifact</th>
                    <th className="px-3 py-2">Graph</th>
                    <th className="px-3 py-2">A/I</th>
                    <th className="px-3 py-2">Created</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {signals.map((s) => {
                    const fam = dsmFamily(s.signal_type);
                    const artifact = getArtifactId(s);
                    const counts = graphTargetCounts(s);
                    return (
                      <tr
                        key={s.signal_id}
                        onClick={() => setSelectedId(s.signal_id)}
                        className={`cursor-pointer align-top hover:bg-gray-50 ${selectedId === s.signal_id ? 'bg-brand-50' : ''}`}
                      >
                        <td className="px-3 py-2 font-mono text-xs">{getTicker(s)}</td>
                        <td className="px-3 py-2 text-xs">{dsmShortType(s.signal_type)}</td>
                        <td className={`px-3 py-2 text-xs ${FAMILY_STYLES[fam]}`}>{fam}</td>
                        <td className="px-3 py-2"><SeverityLabel severity={s.severity} /></td>
                        <td className="px-3 py-2 text-xs">{s.confidence.toFixed(2)}</td>
                        <td className="px-3 py-2 text-xs">{s.dataset || '—'}</td>
                        <td className="max-w-[16rem] px-3 py-2 text-xs text-gray-600">
                          <span className="block truncate" title={keyMetricsSummary(s, fam)}>
                            {keyMetricsSummary(s, fam) || '—'}
                          </span>
                        </td>
                        <td className="max-w-[8rem] px-3 py-2">
                          {artifact ? (
                            <span className="block truncate font-mono text-xs text-gray-600" title={artifact}>
                              {artifact}
                            </span>
                          ) : (
                            <span className="text-xs text-gray-400">—</span>
                          )}
                        </td>
                        <td className="px-3 py-2 text-xs">{counts.nodes + counts.relationships}</td>
                        <td className="px-3 py-2 text-xs">
                          <span className="text-orange-700">{hasLifecycleMatch(s, alertSignalIds) ? 'A' : ''}</span>
                          <span className="text-blue-700">{hasLifecycleMatch(s, insightSignalIds) ? 'I' : ''}</span>
                          {!hasLifecycleMatch(s, alertSignalIds) && !hasLifecycleMatch(s, insightSignalIds) ? (
                            <span className="text-gray-400">—</span>
                          ) : null}
                        </td>
                        <td className="px-3 py-2 text-xs text-gray-600">{formatUtc(s.created_at)}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          ) : (
            <EmptyState message="No DSM taxonomy signals for the current filters." />
          )}
        </div>

        <div className="rounded border border-gray-200 bg-white p-3">
          {!selected ? (
            <EmptyState message="Select a DSM signal to inspect its taxonomy detail." />
          ) : detailQ.isLoading && selectedId ? (
            <LoadingState />
          ) : (
            <DsmDetailBody signal={selected} alert={matchedAlert} insight={matchedInsight} />
          )}
        </div>
      </div>
    </div>
  );
}

function DsmDetailBody({
  signal,
  alert,
  insight,
}: {
  signal: SignalRecord;
  alert: AlertRecord | null;
  insight: InsightRecord | null;
}) {
  const fam = dsmFamily(signal.signal_type);
  const proposal = getArtifactProposal(signal);
  const artifactId = getArtifactId(signal);
  const counts = graphTargetCounts(signal);

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        <span className="font-mono text-sm font-semibold text-gray-900">{getTicker(signal)}</span>
        <span className="text-xs text-gray-600">{dsmShortType(signal.signal_type)}</span>
        <SeverityLabel severity={signal.severity} />
        <span className="text-xs text-gray-500">conf {signal.confidence.toFixed(2)}</span>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <code className="break-all text-xs text-gray-700">{signal.signal_id}</code>
        <CopyButton value={signal.signal_id} />
      </div>

      <div className="grid grid-cols-2 gap-2 text-sm">
        <div><div className="text-xs text-gray-500">Detector</div><div className="text-xs font-mono">{signal.detector_id} v{signal.detector_version}</div></div>
        <div><div className="text-xs text-gray-500">Model</div><div className="text-xs font-mono">{signal.model_version}</div></div>
        <div><div className="text-xs text-gray-500">Source</div><div className="text-xs font-mono">{signal.source_id}</div></div>
        <div><div className="text-xs text-gray-500">Dataset</div><div className="text-xs">{signal.dataset || '—'}</div></div>
        <div><div className="text-xs text-gray-500">Broker</div><div className="text-xs font-mono">{signal.broker_partition}/{signal.broker_offset}</div></div>
        <div><div className="text-xs text-gray-500">Created</div><div className="text-xs">{formatUtc(signal.created_at)}</div></div>
      </div>

      <div className="rounded border border-gray-200 bg-gray-50 p-2 text-xs">
        <div className="mb-1 text-gray-600">Lifecycle</div>
        {alert ? (
          <div>Open alert <Link to="/marketops/alerts" className="break-all font-mono text-brand-700 hover:underline">{alert.alert_id}</Link></div>
        ) : (
          <div className="text-gray-400">No matching open alert.</div>
        )}
        {insight ? (
          <div>Active insight <Link to="/marketops/insights" className="break-all font-mono text-brand-700 hover:underline">{insight.insight_id}</Link></div>
        ) : (
          <div className="text-gray-400">No matching active insight.</div>
        )}
      </div>

      <div className="rounded border border-gray-200 bg-gray-50 p-2">
        <div className="mb-1 text-xs font-medium text-gray-600">DSM Artifact Proposal</div>
        {proposal ? (
          <div className="grid grid-cols-2 gap-2 text-xs">
            <div className="col-span-2"><div className="text-gray-500">Artifact ID</div><div className="break-all font-mono">{proposal.artifact_id ?? '—'}</div></div>
            <div><div className="text-gray-500">Type</div><div className="font-mono">{proposal.artifact_type ?? '—'}</div></div>
            <div><div className="text-gray-500">Symbol</div><div className="font-mono">{proposal.subject?.symbol ?? '—'}</div></div>
            <div><div className="text-gray-500">Severity</div><div>{proposal.severity ?? '—'}</div></div>
            <div><div className="text-gray-500">Confidence</div><div>{proposal.confidence != null ? proposal.confidence.toFixed(2) : '—'}</div></div>
            <div className="col-span-2"><div className="text-gray-500">Quality Issues</div><div>{proposal.quality_issues?.length ? proposal.quality_issues.join(', ') : 'none'}</div></div>
            {proposal.summary ? <div className="col-span-2 text-gray-700">{proposal.summary}</div> : null}
          </div>
        ) : (
          <p className="text-xs text-gray-400">{artifactId ? `Artifact id ${artifactId} (proposal body unavailable).` : 'No artifact proposal.'}</p>
        )}
      </div>

      <div className="space-y-2">
        <div>
          <div className="mb-1 text-xs font-medium text-gray-600">Price Metrics</div>
          <MetricList signal={signal} spec={PRICE_METRICS} />
        </div>
        <div>
          <div className="mb-1 text-xs font-medium text-gray-600">Option-Interest Metrics</div>
          <MetricList signal={signal} spec={OPTION_METRICS} />
        </div>
        <div>
          <div className="mb-1 text-xs font-medium text-gray-600">Quality / Scoring</div>
          <MetricList signal={signal} spec={QUALITY_METRICS} />
        </div>
      </div>

      <div className="rounded border border-gray-200 bg-gray-50 p-2 text-xs">
        <div className="mb-1 text-gray-600">Graph Proposal</div>
        <div className="flex gap-4">
          <div><span className="text-gray-500">Nodes: </span>{counts.nodes}</div>
          <div><span className="text-gray-500">Relationships: </span>{counts.relationships}</div>
        </div>
      </div>

      <div>
        <div className="mb-1 text-xs font-medium text-gray-600">Evidence</div>
        <div className="text-xs text-gray-600">
          <span className="text-gray-500">Events: </span>
          {signal.event_ids.length ? (
            signal.event_ids.map((id) => (
              <Link key={id} to="/marketops/normalized" className="mr-2 break-all font-mono text-brand-700 hover:underline">{id}</Link>
            ))
          ) : (
            <span className="text-gray-400">—</span>
          )}
        </div>
        <div className="text-xs">
          <span className="text-gray-500">Signal: </span>
          <Link to="/marketops/signals" className="break-all font-mono text-brand-700 hover:underline">{signal.signal_id}</Link>
        </div>
      </div>

      <JsonViewer label="Supporting Metrics" value={signal.supporting_metrics} />
      <JsonViewer label="DSM Artifact Proposal" value={proposal ?? signal.semantic_evidence} />
      <JsonViewer label="Graph Targets" value={signal.graph_targets} />
      <JsonViewer label="Semantic Evidence" value={signal.semantic_evidence} />
      <JsonViewer label="Evidence" value={signal.evidence} />
      <JsonViewer label="Recommendation" value={signal.recommendation} />
      <JsonViewer label="Full Signal Event" value={signal.event} />
    </div>
  );
}
