import { useState } from 'react';
import { FlaskConical, Plus } from 'lucide-react';
import {
  useMarketOpsBacktests,
  useMarketOpsBacktest,
  useMarketOpsBacktestSignals,
  useMarketOpsBacktestGraphProposals,
  useCreateMarketOpsBacktest,
} from '../api/queries';
import { isApiError } from '../api/client';
import { LoadingState, ErrorState, EmptyState } from '../components/States';
import { MetricTile } from '../components/MetricTile';
import { StatusBadge } from '../components/StatusBadge';
import { CopyButton } from '../components/CopyButton';
import { JsonViewer } from '../components/JsonViewer';
import { RefreshButton } from '../components/RefreshButton';
import { formatUtc, duration, orDash, toRfc3339Utc, toDatetimeLocal } from '../lib/format';
import { dsmShortType, getTicker, getMetric } from '../lib/marketopsDsm';
import {
  MARKETOPS_BACKTEST_DETECTOR_ID,
  MARKETOPS_BACKTEST_RECOMMENDATIONS,
  summarizeBacktestMetrics,
  isZeroInputBacktest,
  compareBacktestRuns,
  dominantRecommendation,
  parseBacktestSymbols,
  policyResultsByProposal,
  recommendationLabel,
  recommendationStyle,
} from '../lib/marketopsBacktests';
import { useTenant } from '../auth/session';
import type {
  MarketOpsBacktestRun,
  MarketOpsBacktestSignal,
  MarketOpsBacktestGraphProposal,
  MarketOpsBacktestPolicyResult,
  MarketOpsBacktestRunStatus,
  SignalRecord,
} from '../types';

const STATUSES: MarketOpsBacktestRunStatus[] = ['started', 'succeeded', 'failed'];
const LIMITS = [25, 50, 100, 200];
const DATASETS = ['equity_eod_prices', 'options_contracts_daily'] as const;

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null && !Array.isArray(v);
}

// Pull the symbols array out of the run's filters JSON (persisted by the runner
// as {symbols, max_records}). Tolerates any shape.
function filterSymbols(filters: unknown): string[] {
  if (isRecord(filters) && Array.isArray(filters.symbols)) {
    return filters.symbols.filter((s): s is string => typeof s === 'string' && s.length > 0);
  }
  return [];
}

export function MarketOpsBacktestsRoute() {
  const TENANT_ID = useTenant();

  // List filters (spec §1: status, detector id, limit).
  const [status, setStatus] = useState<MarketOpsBacktestRunStatus | ''>('');
  const [detectorId, setDetectorId] = useState(MARKETOPS_BACKTEST_DETECTOR_ID);
  const [limit, setLimit] = useState(50);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const list = useMarketOpsBacktests({
    tenant_id: TENANT_ID,
    detector_id: detectorId || undefined,
    status: status || undefined,
    limit,
  });
  const detail = useMarketOpsBacktest(selectedId, TENANT_ID);
  const create = useCreateMarketOpsBacktest();

  const runs = list.data?.backtest_runs ?? [];
  const succeeded = runs.filter((r) => r.status === 'succeeded').length;
  const failed = runs.filter((r) => r.status === 'failed').length;
  const totalSignals = runs.reduce((n, r) => n + summarizeBacktestMetrics(r.metrics).signals, 0);
  const totalProposals = runs.reduce((n, r) => n + summarizeBacktestMetrics(r.metrics).graphProposals, 0);
  const comparison = compareBacktestRuns(runs);

  const selected: MarketOpsBacktestRun | null = detail.data?.backtest_run ?? runs.find((r) => r.run_id === selectedId) ?? null;

  function refresh() {
    list.refetch();
    if (selectedId) detail.refetch();
  }

  const inputCls = 'rounded border border-gray-300 px-2 py-1 text-sm';
  const labelCls = 'text-xs text-gray-500';

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <FlaskConical size={18} className="text-brand-700" />
        <div>
          <h1 className="text-lg font-semibold">Back-Tests</h1>
          <p className="text-xs text-gray-500">
            Isolated experimental DSM runs · not production signals, graph state, or replay jobs
          </p>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-2 md:grid-cols-5">
        <MetricTile label="Runs" value={runs.length} />
        <MetricTile label="Succeeded" value={succeeded} />
        <MetricTile label="Failed" value={failed} />
        <MetricTile label="Signals" value={totalSignals} hint={list.isError ? 'unreachable' : undefined} />
        <MetricTile label="Graph Proposals" value={totalProposals} hint={list.isError ? 'unreachable' : undefined} />
      </div>

      <BacktestComparisonPanel comparison={comparison} />

      <div className="flex flex-wrap items-center gap-2">
        <select
          value={status}
          onChange={(e) => setStatus(e.target.value as MarketOpsBacktestRunStatus | '')}
          className={inputCls}
          aria-label="Filter by status"
        >
          <option value="">any status</option>
          {STATUSES.map((s) => (
            <option key={s} value={s}>{s}</option>
          ))}
        </select>
        <input
          value={detectorId}
          onChange={(e) => setDetectorId(e.target.value)}
          className={inputCls}
          aria-label="Filter by detector id"
          placeholder="detector id"
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
        <RefreshButton onClick={refresh} loading={list.isFetching} />
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        {/* Run list — left/span-2 on desktop. */}
        <div className="lg:col-span-2">
          {list.isLoading ? (
            <LoadingState />
          ) : list.isError ? (
            <ErrorState error={list.error} />
          ) : runs.length ? (
            <div className="overflow-x-auto rounded border border-gray-200 bg-white">
              <table className="min-w-full divide-y divide-gray-200 text-sm">
                <thead className="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500">
                  <tr>
                    <th className="px-3 py-2">Run</th>
                    <th className="px-3 py-2">Status</th>
                    <th className="px-3 py-2">Dataset</th>
                    <th className="px-3 py-2">Detector</th>
                    <th className="px-3 py-2">Window</th>
                    <th className="px-3 py-2">Scanned</th>
                    <th className="px-3 py-2">Signals</th>
                    <th className="px-3 py-2">Graph</th>
                    <th className="px-3 py-2">Top Rec.</th>
                    <th className="px-3 py-2">Updated</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {runs.map((r) => {
                    const m = summarizeBacktestMetrics(r.metrics);
                    const top = dominantRecommendation(m.recommendationCounts);
                    return (
                      <tr
                        key={r.run_id}
                        onClick={() => setSelectedId(r.run_id)}
                        className={`cursor-pointer align-top hover:bg-gray-50 ${selectedId === r.run_id ? 'bg-brand-50' : ''}`}
                      >
                        <td className="px-3 py-2">
                          <div className="font-mono text-xs text-gray-800">{r.run_id}</div>
                          <div className="text-xs text-gray-500">{r.requested_by || '—'}</div>
                        </td>
                        <td className="px-3 py-2"><StatusBadge status={r.status} /></td>
                        <td className="px-3 py-2 text-xs">{r.dataset || '—'}</td>
                        <td className="px-3 py-2 text-xs font-mono">{r.detector_id}</td>
                        <td className="px-3 py-2 text-xs text-gray-600">
                          <div>{formatUtc(r.window_start)}</div>
                          <div>{formatUtc(r.window_end)}</div>
                        </td>
                        <td className="px-3 py-2 text-xs">{m.scanned}</td>
                        <td className="px-3 py-2 text-xs">{m.signals}</td>
                        <td className="px-3 py-2 text-xs">{m.graphProposals}</td>
                        <td className="px-3 py-2 text-xs">
                          {top ? (
                            <span title={top.key}>{recommendationLabel(top.key)} ({top.count})</span>
                          ) : (
                            <span className="text-gray-400">—</span>
                          )}
                        </td>
                        <td className="px-3 py-2 text-xs text-gray-600">{formatUtc(r.updated_at)}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          ) : (
            <EmptyState message="No MarketOps back-test runs for the current filters." />
          )}
        </div>

        {/* Create form — right column. */}
        <BacktestCreateForm
          tenantId={TENANT_ID}
          detectorId={detectorId}
          create={create}
          onCreated={(runId) => setSelectedId(runId)}
        />
      </div>

      {selected && (
        <BacktestRunDetail
          run={selected}
          tenantId={TENANT_ID}
          loading={detail.isLoading && !!selectedId}
          error={detail.isError}
        />
      )}
    </div>
  );
}

// Bounded synchronous create form (spec §2). The runner completes before 201
// returns, so "Queuing…" really means "Running…". max_records is bounded and
// visible to keep runs small.
function BacktestCreateForm({
  tenantId,
  detectorId,
  create,
  onCreated,
}: {
  tenantId: string;
  detectorId: string;
  create: ReturnType<typeof useCreateMarketOpsBacktest>;
  onCreated: (runId: string) => void;
}) {
  const [fRunId, setFRunId] = useState('');
  const [fSourceId, setFSourceId] = useState('src-massive');
  const [fDataset, setFDataset] = useState<string>('equity_eod_prices');
  const [fSymbols, setFSymbols] = useState('SPY');
  const [fStart, setFStart] = useState(() => {
    const d = new Date();
    d.setDate(d.getDate() - 1);
    return toDatetimeLocal(d.toISOString());
  });
  const [fEnd, setFEnd] = useState(() => toDatetimeLocal(new Date().toISOString()));
  const [fMax, setFMax] = useState(5);
  const [fBatch, setFBatch] = useState(5);
  const [fDetectorId, setFDetectorId] = useState(detectorId);
  const [fDetectorVersion, setFDetectorVersion] = useState('v1');
  const [fAutoAccept, setFAutoAccept] = useState(0.75);
  const [errors, setErrors] = useState<Record<string, string>>({});

  // Keep the detector field in sync with the list filter when it changes upstream.
  // (Avoids clobbering an operator edit: only sync when the external value differs
  // and the field hasn't been touched away from the previous external value.)
  const [lastSyncedDetector, setLastSyncedDetector] = useState(detectorId);
  if (detectorId !== lastSyncedDetector && fDetectorId === lastSyncedDetector) {
    setFDetectorId(detectorId);
    setLastSyncedDetector(detectorId);
  } else if (detectorId !== lastSyncedDetector) {
    setLastSyncedDetector(detectorId);
  }

  function touch(key?: string) {
    create.reset();
    setErrors((e) => (key ? { ...e, [key]: '' } : {}));
  }

  function validate(): Record<string, string> {
    const e: Record<string, string> = {};
    if (!fStart.trim()) e.start = 'Required';
    if (!fEnd.trim()) e.end = 'Required';
    if (!e.start && !e.end) {
      const st = new Date(toRfc3339Utc(fStart)).getTime();
      const et = new Date(toRfc3339Utc(fEnd)).getTime();
      if (isNaN(st) || isNaN(et)) e.end = 'Invalid datetime';
      else if (et <= st) e.end = 'window_end must be after window_start';
    }
    if (!Number.isFinite(fMax) || fMax < 1 || fMax > 1000) e.max = '1–1000';
    if (!Number.isFinite(fBatch) || fBatch < 1 || fBatch > 1000) e.batch = '1–1000';
    if (!Number.isFinite(fAutoAccept) || fAutoAccept < 0 || fAutoAccept > 1) e.auto = '0–1';
    return e;
  }

  function onSubmit(ev: React.FormEvent) {
    ev.preventDefault();
    const e = validate();
    setErrors(e);
    if (Object.values(e).some(Boolean)) return;
    create.mutate(
      {
        tenant_id: tenantId,
        run_id: fRunId.trim() || undefined,
        source_id: fSourceId.trim() || undefined,
        dataset: fDataset,
        detector_id: fDetectorId.trim() || undefined,
        detector_version: fDetectorVersion.trim() || undefined,
        window_start: toRfc3339Utc(fStart),
        window_end: toRfc3339Utc(fEnd),
        symbols: parseBacktestSymbols(fSymbols),
        max_records: fMax,
        batch_size: fBatch,
        auto_accept_confidence: fAutoAccept,
      },
      { onSuccess: (d) => onCreated(d.backtest_run.run_id) },
    );
  }

  const inputCls = 'rounded border border-gray-300 px-2 py-1 text-sm';
  const labelCls = 'text-xs text-gray-500';

  return (
    <form
      onSubmit={onSubmit}
      className="space-y-2 rounded border border-gray-200 bg-white p-3"
      aria-label="Create back-test run"
    >
      <div className="flex items-center gap-1 text-sm font-semibold text-gray-900">
        <Plus size={14} /> New Back-Test Run
      </div>

      <label className="block">
        <span className={labelCls}>Run id <span className="text-gray-400">(optional · backend generates)</span></span>
        <input
          value={fRunId}
          onChange={(e) => { touch(); setFRunId(e.target.value); }}
          placeholder="bt-…"
          className={`${inputCls} mt-0.5 w-full`}
        />
      </label>

      <div className="grid grid-cols-2 gap-2">
        <label className="block">
          <span className={labelCls}>Source id</span>
          <input
            value={fSourceId}
            onChange={(e) => { touch(); setFSourceId(e.target.value); }}
            className={`${inputCls} mt-0.5 w-full`}
          />
        </label>
        <label className="block">
          <span className={labelCls}>Dataset</span>
          <select
            value={fDataset}
            onChange={(e) => { touch(); setFDataset(e.target.value); }}
            className={`${inputCls} mt-0.5 w-full`}
          >
            {DATASETS.map((d) => (
              <option key={d} value={d} disabled={d === 'options_contracts_daily'}>
                {d}{d === 'options_contracts_daily' ? ' (disabled — no option source context)' : ''}
              </option>
            ))}
          </select>
        </label>
      </div>

      <label className="block">
        <span className={labelCls}>Symbols <span className="text-gray-400">(comma-separated, uppercased)</span></span>
        <input
          value={fSymbols}
          onChange={(e) => { touch(); setFSymbols(e.target.value); }}
          placeholder="SPY, AAPL"
          className={`${inputCls} mt-0.5 w-full`}
        />
      </label>

      <div className="grid grid-cols-2 gap-2">
        <label className="block">
          <span className={labelCls}>Window start <span className="text-gray-400">(UTC)</span></span>
          <input
            type="datetime-local"
            value={fStart}
            onChange={(e) => { touch('start'); setFStart(e.target.value); }}
            className={`${inputCls} mt-0.5 w-full`}
            aria-invalid={Boolean(errors.start)}
          />
          {errors.start && <span className="text-xs text-red-700">{errors.start}</span>}
        </label>
        <label className="block">
          <span className={labelCls}>Window end <span className="text-gray-400">(UTC)</span></span>
          <input
            type="datetime-local"
            value={fEnd}
            onChange={(e) => { touch('end'); setFEnd(e.target.value); }}
            className={`${inputCls} mt-0.5 w-full`}
            aria-invalid={Boolean(errors.end)}
          />
          {errors.end && <span className="text-xs text-red-700">{errors.end}</span>}
        </label>
      </div>

      <div className="grid grid-cols-3 gap-2">
        <label className="block">
          <span className={labelCls}>Max records (1–1000)</span>
          <input
            type="number"
            min={1}
            max={1000}
            value={fMax}
            onChange={(e) => { touch('max'); setFMax(Number(e.target.value)); }}
            className={`${inputCls} mt-0.5 w-full`}
            aria-invalid={Boolean(errors.max)}
          />
          {errors.max && <span className="text-xs text-red-700">{errors.max}</span>}
        </label>
        <label className="block">
          <span className={labelCls}>Batch size (1–1000)</span>
          <input
            type="number"
            min={1}
            max={1000}
            value={fBatch}
            onChange={(e) => { touch('batch'); setFBatch(Number(e.target.value)); }}
            className={`${inputCls} mt-0.5 w-full`}
            aria-invalid={Boolean(errors.batch)}
          />
          {errors.batch && <span className="text-xs text-red-700">{errors.batch}</span>}
        </label>
        <label className="block">
          <span className={labelCls}>Auto-accept conf (0–1)</span>
          <input
            type="number"
            min={0}
            max={1}
            step={0.05}
            value={fAutoAccept}
            onChange={(e) => { touch('auto'); setFAutoAccept(Number(e.target.value)); }}
            className={`${inputCls} mt-0.5 w-full`}
            aria-invalid={Boolean(errors.auto)}
          />
          {errors.auto && <span className="text-xs text-red-700">{errors.auto}</span>}
        </label>
      </div>

      <div className="grid grid-cols-2 gap-2">
        <label className="block">
          <span className={labelCls}>Detector id</span>
          <input
            value={fDetectorId}
            onChange={(e) => { touch(); setFDetectorId(e.target.value); }}
            className={`${inputCls} mt-0.5 w-full`}
          />
        </label>
        <label className="block">
          <span className={labelCls}>Detector version</span>
          <input
            value={fDetectorVersion}
            onChange={(e) => { touch(); setFDetectorVersion(e.target.value); }}
            className={`${inputCls} mt-0.5 w-full`}
          />
        </label>
      </div>

      <button
        type="submit"
        disabled={create.isPending}
        className="inline-flex w-full items-center justify-center gap-1 rounded bg-brand-500 px-3 py-1.5 text-sm text-white hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-50"
      >
        <Plus size={14} /> {create.isPending ? 'Running…' : 'Run back-test'}
      </button>

      {create.isSuccess && create.data && (
        <div className="rounded border border-green-200 bg-green-50 p-2 text-xs text-green-800">
          <div className="flex flex-wrap items-center gap-2">
            <StatusBadge status={create.data.backtest_run.status} />
            <span>Run complete</span>
            <code className="font-mono">{create.data.backtest_run.run_id}</code>
          </div>
        </div>
      )}
      {create.isError && (
        <p className="text-xs text-red-700" role="alert">
          Create failed: {isApiError(create.error) ? create.error.message : 'unknown error'}.
        </p>
      )}
    </form>
  );
}


function BacktestComparisonPanel({ comparison }: { comparison: ReturnType<typeof compareBacktestRuns> }) {
  const recEntries = Object.entries(comparison.recommendationCounts).filter(([, count]) => count > 0);
  const dominant = comparison.dominantRecommendation;
  return (
    <div className="rounded border border-gray-200 bg-white p-3">
      <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
        <div>
          <div className="text-sm font-semibold text-gray-900">Calibration Summary</div>
          <div className="text-xs text-gray-500">Compared over the currently listed back-test runs.</div>
        </div>
        <div className="text-xs text-gray-500">
          {comparison.datasets.length ? comparison.datasets.join(', ') : 'all datasets'} · {comparison.detectorIds.length ? comparison.detectorIds.length : 0} detector{comparison.detectorIds.length === 1 ? '' : 's'}
        </div>
      </div>
      <div className="grid grid-cols-2 gap-2 md:grid-cols-5">
        <MetricTile label="Compared Runs" value={comparison.runs} />
        <MetricTile label="Zero Input" value={comparison.zeroInput} hint={comparison.runs ? `${((comparison.zeroInput / comparison.runs) * 100).toFixed(0)}%` : undefined} />
        <MetricTile label="Signal Yield" value={`${comparison.signalYieldPct.toFixed(1)}%`} hint={`${comparison.signals}/${comparison.scanned} scanned`} />
        <MetricTile label="Policy / Signal" value={comparison.policyResultsPerSignal.toFixed(1)} hint={`${comparison.policyResults} policy results`} />
        <MetricTile label="Dominant Rec." value={dominant ? recommendationLabel(dominant.key) : '—'} hint={dominant ? `${dominant.count} · ${(dominant.share * 100).toFixed(0)}%` : undefined} />
      </div>
      <div className="mt-2">
        <div className="mb-1 text-xs font-medium text-gray-600">Recommendation Mix</div>
        {recEntries.length ? (
          <div className="flex flex-wrap gap-2">
            {recEntries.map(([key, count]) => (
              <RecommendationChip key={key} recommendation={key} count={count} />
            ))}
          </div>
        ) : (
          <span className="text-xs text-gray-400">No policy recommendations in the current comparison set.</span>
        )}
      </div>
    </div>
  );
}

function RecommendationChip({ recommendation, count }: { recommendation: string; count?: number }) {
  return (
    <span
      className={`inline-flex items-center gap-1 rounded border px-2 py-0.5 text-xs font-medium ${recommendationStyle(recommendation)}`}
      title={recommendation}
    >
      {recommendationLabel(recommendation)}{count !== undefined ? ` ${count}` : ''}
    </span>
  );
}

function BacktestRunDetail({
  run,
  tenantId,
  loading,
  error,
}: {
  run: MarketOpsBacktestRun;
  tenantId: string;
  loading: boolean;
  error: boolean;
}) {
  const m = summarizeBacktestMetrics(run.metrics);
  const symbols = filterSymbols(run.filters);
  const zeroInput = isZeroInputBacktest(run.status, run.metrics);
  const recEntries = Object.entries(m.recommendationCounts).filter(([, c]) => c > 0);

  return (
    <div className="space-y-3 rounded border border-gray-200 bg-white p-3">
      <div className="flex flex-wrap items-center gap-2">
        <StatusBadge status={run.status} />
        <span className="text-xs text-gray-500">{duration(run.started_at, run.completed_at) || orDash(run.started_at)}</span>
        <code className="break-all text-xs text-gray-700">{run.run_id}</code>
        <CopyButton value={run.run_id} />
      </div>

      {run.error_message && (
        <p className="rounded border border-red-200 bg-red-50 p-2 text-xs text-red-800" role="alert">
          {run.error_message}
        </p>
      )}

      <div className="grid grid-cols-2 gap-2 text-sm md:grid-cols-4">
        <div><div className="text-xs text-gray-500">Detector</div><div className="text-xs font-mono">{run.detector_id} v{run.detector_version || '—'}</div></div>
        <div><div className="text-xs text-gray-500">Dataset</div><div className="text-xs">{run.dataset || '—'}</div></div>
        <div><div className="text-xs text-gray-500">Source</div><div className="text-xs font-mono">{run.source_id || '—'}</div></div>
        <div><div className="text-xs text-gray-500">Requested by</div><div className="text-xs">{run.requested_by || '—'}</div></div>
        <div><div className="text-xs text-gray-500">Window</div><div className="text-xs text-gray-600">{formatUtc(run.window_start)} → {formatUtc(run.window_end)}</div></div>
        <div><div className="text-xs text-gray-500">Symbols</div><div className="text-xs font-mono">{symbols.length ? symbols.join(', ') : '—'}</div></div>
        <div><div className="text-xs text-gray-500">Started</div><div className="text-xs text-gray-600">{formatUtc(run.started_at)}</div></div>
        <div><div className="text-xs text-gray-500">Completed</div><div className="text-xs text-gray-600">{formatUtc(run.completed_at)}</div></div>
      </div>

      <div className="grid grid-cols-2 gap-2 md:grid-cols-5">
        <MetricTile label="Scanned" value={m.scanned} />
        <MetricTile label="Signals" value={m.signals} />
        <MetricTile label="Artifacts" value={m.artifacts} />
        <MetricTile label="Graph Proposals" value={m.graphProposals} />
        <MetricTile label="Policy Results" value={m.policyResults} />
      </div>

      {zeroInput && (
        <div className="rounded border border-amber-200 bg-amber-50 p-3 text-xs text-amber-900">
          <div className="font-semibold">No matching normalized events found.</div>
          <div className="mt-1 text-amber-800">
            This back-test completed successfully, but no normalized rows matched the selected filters. Broaden the symbols or window, or use a known populated window such as SPY on 2026-07-09.
          </div>
          <div className="mt-2 grid gap-1 font-mono text-[11px] text-amber-950 sm:grid-cols-2">
            <div>symbols: {symbols.length ? symbols.join(', ') : 'any'}</div>
            <div>source: {run.source_id || 'any'}</div>
            <div>dataset: {run.dataset || 'any'}</div>
            <div>window: {formatUtc(run.window_start)} - {formatUtc(run.window_end)}</div>
          </div>
        </div>
      )}

      <div>
        <div className="mb-1 text-xs font-medium text-gray-600">Recommendation Counts</div>
        {recEntries.length ? (
          <div className="flex flex-wrap gap-2">
            {recEntries.map(([key, count]) => (
              <RecommendationChip key={key} recommendation={key} count={count} />
            ))}
          </div>
        ) : (
          <span className="text-xs text-gray-400">No policy recommendations recorded.</span>
        )}
      </div>

      {error ? (
        <p className="text-xs text-amber-700">Run detail unavailable.</p>
      ) : (
        <>
          <BacktestSignalsSection runId={run.run_id} tenantId={tenantId} loading={loading} />
          <BacktestGraphProposalsSection runId={run.run_id} tenantId={tenantId} />
        </>
      )}

      <div className="space-y-2">
        <JsonViewer label="Metrics (raw)" value={run.metrics} />
        <JsonViewer label="Filters (raw)" value={run.filters} />
        <JsonViewer label="Parameters (raw)" value={run.parameters} />
      </div>
    </div>
  );
}

function BacktestSignalsSection({
  runId,
  tenantId,
  loading,
}: {
  runId: string;
  tenantId: string;
  loading: boolean;
}) {
  const q = useMarketOpsBacktestSignals(runId, { tenant_id: tenantId, limit: 50 });
  const signals = q.data?.backtest_signals ?? [];

  return (
    <div className="rounded border border-gray-200 bg-gray-50 p-2">
      <div className="mb-1 text-xs font-semibold text-gray-700">Back-Test Signals</div>
      <p className="mb-2 text-xs text-gray-500">
        Isolated to this run · not production signal ledger rows.
      </p>
      {loading || q.isLoading ? (
        <p className="text-xs text-gray-500">Loading signals…</p>
      ) : q.isError ? (
        <p className="text-xs text-amber-700">Signals unavailable.</p>
      ) : signals.length ? (
        <div className="overflow-x-auto rounded border border-gray-200 bg-white">
          <table className="min-w-full divide-y divide-gray-200 text-sm">
            <thead className="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500">
              <tr>
                <th className="px-2 py-1">Signal</th>
                <th className="px-2 py-1">Type</th>
                <th className="px-2 py-1">Severity</th>
                <th className="px-2 py-1">Conf.</th>
                <th className="px-2 py-1">Ticker</th>
                <th className="px-2 py-1">Events</th>
                <th className="px-2 py-1">Metrics</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {signals.map((row) => (
                <BacktestSignalRow key={row.signal.signal_id} row={row} />
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <p className="text-xs text-gray-400">No back-test signals generated for this run.</p>
      )}
    </div>
  );
}

function BacktestSignalRow({ row }: { row: MarketOpsBacktestSignal }) {
  const s = row.signal;
  const metrics = s.supporting_metrics;
  const summary =
    isRecord(metrics) &&
    Object.keys(metrics)
      .slice(0, 3)
      .map((k) => `${k}=${getMetric(s, k)}`)
      .join(' · ');
  const events = s.event_ids ?? [];
  return (
    <tr className="align-top">
      <td className="px-2 py-1">
        <code className="break-all text-xs text-gray-700">{s.signal_id}</code>
      </td>
      <td className="px-2 py-1 text-xs">{dsmShortType(s.signal_type)}</td>
      <td className="px-2 py-1 text-xs">{s.severity}</td>
      <td className="px-2 py-1 text-xs">{s.confidence.toFixed(2)}</td>
      <td className="px-2 py-1 font-mono text-xs">{getTicker(s)}</td>
      <td className="px-2 py-1 text-xs text-gray-600">{events.length ? `${events.length}` : '—'}</td>
      <td className="max-w-[14rem] px-2 py-1 text-xs text-gray-600">
        <span className="block truncate" title={summary || ''}>{summary || '—'}</span>
      </td>
    </tr>
  );
}

function BacktestGraphProposalsSection({
  runId,
  tenantId,
}: {
  runId: string;
  tenantId: string;
}) {
  const [recommendation, setRecommendation] = useState<string>('');
  const [candidateType, setCandidateType] = useState<string>('');
  const [subjectSymbol, setSubjectSymbol] = useState<string>('');
  const [gpLimit, setGpLimit] = useState(50);

  const q = useMarketOpsBacktestGraphProposals(runId, {
    tenant_id: tenantId,
    recommendation: (recommendation || undefined) as never,
    candidate_type: candidateType || undefined,
    subject_symbol: subjectSymbol || undefined,
    limit: gpLimit,
  });
  const proposals = q.data?.backtest_graph_proposals ?? [];
  const policyMap = policyResultsByProposal(q.data?.policy_results);

  // recommendation filters only policy_results server-side; hide proposals whose
  // joined recommendation does not match so the table stays consistent.
  const visible = recommendation
    ? proposals.filter((p) => policyMap.get(p.graph_proposal.proposal_id)?.recommendation === recommendation)
    : proposals;

  const inputCls = 'rounded border border-gray-300 px-2 py-1 text-sm';

  return (
    <div className="rounded border border-gray-200 bg-gray-50 p-2">
      <div className="mb-1 text-xs font-semibold text-gray-700">Back-Test Graph Proposals</div>
      <p className="mb-2 text-xs text-gray-500">
        Isolated policy outputs · not production graph state. No decision controls.
      </p>

      <div className="mb-2 flex flex-wrap items-center gap-2">
        <select
          value={recommendation}
          onChange={(e) => setRecommendation(e.target.value)}
          className={inputCls}
          aria-label="Filter by recommendation"
        >
          <option value="">any recommendation</option>
          {MARKETOPS_BACKTEST_RECOMMENDATIONS.map((r) => (
            <option key={r} value={r}>{recommendationLabel(r)}</option>
          ))}
        </select>
        <select
          value={candidateType}
          onChange={(e) => setCandidateType(e.target.value)}
          className={inputCls}
          aria-label="Filter by candidate type"
        >
          <option value="">any type</option>
          <option value="node_candidate">node_candidate</option>
          <option value="relationship_candidate">relationship_candidate</option>
        </select>
        <input
          value={subjectSymbol}
          onChange={(e) => setSubjectSymbol(e.target.value)}
          placeholder="subject symbol"
          className={inputCls}
          aria-label="Filter by subject symbol"
        />
        <select
          value={gpLimit}
          onChange={(e) => setGpLimit(Number(e.target.value))}
          className={inputCls}
          aria-label="Page limit"
        >
          {LIMITS.map((n) => (
            <option key={n} value={n}>{n}</option>
          ))}
        </select>
      </div>

      {q.isLoading ? (
        <p className="text-xs text-gray-500">Loading graph proposals…</p>
      ) : q.isError ? (
        <p className="text-xs text-amber-700">Graph proposals unavailable.</p>
      ) : visible.length ? (
        <div className="overflow-x-auto rounded border border-gray-200 bg-white">
          <table className="min-w-full divide-y divide-gray-200 text-sm">
            <thead className="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500">
              <tr>
                <th className="px-2 py-1">Proposal</th>
                <th className="px-2 py-1">Recommendation</th>
                <th className="px-2 py-1">Status</th>
                <th className="px-2 py-1">Type</th>
                <th className="px-2 py-1">Symbol</th>
                <th className="px-2 py-1">Subject</th>
                <th className="px-2 py-1">Conf.</th>
                <th className="px-2 py-1">Policy Reason</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {visible.map((row) => (
                <BacktestGraphProposalRow key={row.graph_proposal.proposal_id} row={row} policy={policyMap.get(row.graph_proposal.proposal_id)} />
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <p className="text-xs text-gray-400">No back-test graph proposals matched the current filters.</p>
      )}
    </div>
  );
}

function BacktestGraphProposalRow({
  row,
  policy,
}: {
  row: MarketOpsBacktestGraphProposal;
  policy: MarketOpsBacktestPolicyResult | undefined;
}) {
  const p = row.graph_proposal;
  const subject =
    p.candidate_type === 'relationship_candidate'
      ? [p.from_node, p.relationship, p.to_node].filter(Boolean).join(' → ')
      : p.node_id;
  const rec = policy?.recommendation;
  return (
    <tr className="align-top">
      <td className="px-2 py-1">
        <code className="break-all text-xs text-gray-700">{p.proposal_id}</code>
      </td>
      <td className="px-2 py-1 text-xs">
        {rec ? <RecommendationChip recommendation={rec} /> : <span className="text-gray-400">—</span>}
      </td>
      <td className="px-2 py-1 text-xs">{p.status || '—'}</td>
      <td className="px-2 py-1 text-xs">
        {p.candidate_type === 'node_candidate' ? 'node' : p.candidate_type === 'relationship_candidate' ? 'rel' : p.candidate_type}
      </td>
      <td className="px-2 py-1 font-mono text-xs">{p.subject_symbol || '—'}</td>
      <td className="max-w-[16rem] px-2 py-1 text-xs text-gray-600">
        <span className="block truncate font-mono" title={subject}>{subject || '—'}</span>
      </td>
      <td className="px-2 py-1 text-xs">{(policy?.confidence ?? p.confidence).toFixed(2)}</td>
      <td className="max-w-[18rem] px-2 py-1 text-xs text-gray-600">
        <span className="block truncate" title={policy?.reason || ''}>{policy?.reason || '—'}</span>
      </td>
    </tr>
  );
}
