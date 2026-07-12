// Pure parsing helpers for the MarketOps back-test workspace (G081 frontend).
//
// Back-test run metrics/filters/parameters arrive as already-parsed JSON from
// the gateway (typed `unknown` on MarketOpsBacktestRun). Narrow with type
// guards only; never JSON.parse. Missing/malformed values collapse to 0 / ''
// and must never throw. Recommendation values mirror the four backend policy
// constants in internal/marketops/dsm/policy.go.
import type { MarketOpsBacktestPolicyResult, MarketOpsBacktestRun } from '../types';

export const MARKETOPS_BACKTEST_DETECTOR_ID = 'marketops.dsm.taxonomy_v1';

export const MARKETOPS_BACKTEST_RECOMMENDATIONS = [
  'auto_accept_candidate',
  'auto_reject_candidate',
  'manual_review_required',
  'supersede_candidate',
] as const;

// Restrained token colors for recommendation chips/labels. auto_accept -> green,
// manual_review -> amber, auto_reject -> red, supersede -> violet (matches the
// spec's token guidance and the existing DSM severity palette).
export const BACKTEST_RECOMMENDATION_STYLES: Record<string, string> = {
  auto_accept_candidate: 'text-emerald-700 bg-emerald-50 border-emerald-200',
  auto_reject_candidate: 'text-red-700 bg-red-50 border-red-200',
  manual_review_required: 'text-amber-700 bg-amber-50 border-amber-200',
  supersede_candidate: 'text-violet-700 bg-violet-50 border-violet-200',
};

export function recommendationStyle(key: string): string {
  return BACKTEST_RECOMMENDATION_STYLES[key] ?? 'text-gray-700 bg-gray-50 border-gray-200';
}

// Humanize a recommendation key: "manual_review_required" -> "manual review required".
export function recommendationLabel(key: string): string {
  return typeof key === 'string' ? key.replace(/_/g, ' ') : '';
}

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null && !Array.isArray(v);
}

// Coerce a metric value to a finite number. The gateway emits numbers, but be
// defensive: numeric strings parse, everything else (null/missing/objects) is 0.
function asNumber(v: unknown): number {
  if (typeof v === 'number' && Number.isFinite(v)) return v;
  if (typeof v === 'string' && v !== '') {
    const n = Number(v);
    if (Number.isFinite(n)) return n;
  }
  return 0;
}

// Coerce recommendation_counts into Record<string, number>, skipping
// non-numeric values so a malformed map never breaks the chips.
function asCountMap(v: unknown): Record<string, number> {
  if (!isRecord(v)) return {};
  const out: Record<string, number> = {};
  for (const [k, val] of Object.entries(v)) {
    if (typeof val === 'number' && Number.isFinite(val)) out[k] = val;
    else if (typeof val === 'string' && val !== '') {
      const n = Number(val);
      if (Number.isFinite(n)) out[k] = n;
    }
  }
  return out;
}

export interface BacktestMetricsSummary {
  scanned: number;
  signals: number;
  artifacts: number;
  graphProposals: number;
  policyResults: number;
  recommendationCounts: Record<string, number>;
  batches: number;
  maxRecords: number;
  batchSize: number;
  startedAt: string;
  completedAt: string;
}

const EMPTY_SUMMARY: BacktestMetricsSummary = {
  scanned: 0,
  signals: 0,
  artifacts: 0,
  graphProposals: 0,
  policyResults: 0,
  recommendationCounts: {},
  batches: 0,
  maxRecords: 0,
  batchSize: 0,
  startedAt: '',
  completedAt: '',
};

// Tolerantly summarize a back-test run's metrics JSON. Never throws: a missing
// or non-object metrics value yields the empty summary.
export function summarizeBacktestMetrics(metrics: unknown): BacktestMetricsSummary {
  if (!isRecord(metrics)) return { ...EMPTY_SUMMARY };
  const m = metrics;
  return {
    scanned: asNumber(m.scanned),
    signals: asNumber(m.signals),
    artifacts: asNumber(m.artifacts),
    graphProposals: asNumber(m.graph_proposals),
    policyResults: asNumber(m.policy_results),
    recommendationCounts: asCountMap(m.recommendation_counts),
    batches: asNumber(m.batches),
    maxRecords: asNumber(m.max_records),
    batchSize: asNumber(m.batch_size),
    startedAt: typeof m.started_at === 'string' ? m.started_at : '',
    completedAt: typeof m.completed_at === 'string' ? m.completed_at : '',
  };
}

// A succeeded run with scanned=0 is not an execution failure; it means the
// selected source/dataset/window/symbol filters matched no normalized events.
export function isZeroInputBacktest(status: string, metrics: unknown): boolean {
  return status === 'succeeded' && summarizeBacktestMetrics(metrics).scanned === 0;
}

// The recommendation with the highest count (>0). Ties keep the first insertion
// order encountered. Returns null when no recommendation has a positive count.
export function dominantRecommendation(counts: Record<string, number>): { key: string; count: number } | null {
  let best: { key: string; count: number } | null = null;
  for (const [k, c] of Object.entries(counts)) {
    if (c > 0 && (!best || c > best.count)) best = { key: k, count: c };
  }
  return best;
}

// Parse a comma- or whitespace-separated symbols input into an uppercase,
// trimmed, de-duplicated array (empty values dropped). Mirrors the backend's
// cleanSymbols: input "spy, aapl aapl" -> ["SPY", "AAPL"].
export function parseBacktestSymbols(input: string): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const raw of String(input ?? '').split(/[\s,]+/)) {
    const v = raw.trim().toUpperCase();
    if (!v || seen.has(v)) continue;
    seen.add(v);
    out.push(v);
  }
  return out;
}

export interface BacktestComparisonSummary {
  runs: number;
  succeeded: number;
  failed: number;
  zeroInput: number;
  scanned: number;
  signals: number;
  artifacts: number;
  graphProposals: number;
  policyResults: number;
  signalYieldPct: number;
  policyResultsPerSignal: number;
  recommendationCounts: Record<string, number>;
  recommendationShares: Record<string, number>;
  dominantRecommendation: { key: string; count: number; share: number } | null;
  datasets: string[];
  detectorIds: string[];
}

// Aggregate the current run list into a lightweight calibration/comparison
// summary. This is intentionally run-list scoped: no new backend aggregate API
// is required, and operators can change list filters to compare different sets.
export function compareBacktestRuns(runs: MarketOpsBacktestRun[] | unknown): BacktestComparisonSummary {
  const list = Array.isArray(runs) ? runs : [];
  const recommendationCounts: Record<string, number> = {};
  const datasets = new Set<string>();
  const detectorIds = new Set<string>();
  let succeeded = 0;
  let failed = 0;
  let zeroInput = 0;
  let scanned = 0;
  let signals = 0;
  let artifacts = 0;
  let graphProposals = 0;
  let policyResults = 0;

  for (const run of list) {
    if (!isRecord(run)) continue;
    const status = typeof run.status === 'string' ? run.status : '';
    if (status === 'succeeded') succeeded++;
    if (status === 'failed') failed++;
    const metrics = summarizeBacktestMetrics(run.metrics);
    if (isZeroInputBacktest(status, run.metrics)) zeroInput++;
    scanned += metrics.scanned;
    signals += metrics.signals;
    artifacts += metrics.artifacts;
    graphProposals += metrics.graphProposals;
    policyResults += metrics.policyResults;
    for (const [key, count] of Object.entries(metrics.recommendationCounts)) {
      recommendationCounts[key] = (recommendationCounts[key] ?? 0) + count;
    }
    if (typeof run.dataset === 'string' && run.dataset) datasets.add(run.dataset);
    if (typeof run.detector_id === 'string' && run.detector_id) detectorIds.add(run.detector_id);
  }

  const recommendationShares: Record<string, number> = {};
  for (const [key, count] of Object.entries(recommendationCounts)) {
    recommendationShares[key] = policyResults > 0 ? count / policyResults : 0;
  }
  const dominant = dominantRecommendation(recommendationCounts);
  return {
    runs: list.filter(isRecord).length,
    succeeded,
    failed,
    zeroInput,
    scanned,
    signals,
    artifacts,
    graphProposals,
    policyResults,
    signalYieldPct: scanned > 0 ? (signals / scanned) * 100 : 0,
    policyResultsPerSignal: signals > 0 ? policyResults / signals : 0,
    recommendationCounts,
    recommendationShares,
    dominantRecommendation: dominant ? { ...dominant, share: policyResults > 0 ? dominant.count / policyResults : 0 } : null,
    datasets: Array.from(datasets).sort(),
    detectorIds: Array.from(detectorIds).sort(),
  };
}

// Index policy results by proposal_id so the graph-proposals table can render
// each proposal's joined recommendation + reason. Entries with no proposal_id
// are skipped.
export function policyResultsByProposal(results: MarketOpsBacktestPolicyResult[] | unknown): Map<string, MarketOpsBacktestPolicyResult> {
  const map = new Map<string, MarketOpsBacktestPolicyResult>();
  if (!Array.isArray(results)) return map;
  for (const r of results) {
    if (isRecord(r) && typeof r.proposal_id === 'string' && r.proposal_id) {
      map.set(r.proposal_id, r as unknown as MarketOpsBacktestPolicyResult);
    }
  }
  return map;
}

// G083 calibration comparison display helpers. These power the stored
// baseline-to-candidate comparison panel: recommendation chips + the ordered
// key deltas from comparison_metrics.deltas. Recommendation values are advisory
// labels only — the UI never renders them as deploy/promote decisions.

// Advisory comparison recommendation values. Mirror the backend constants in
// internal/storage/storage.go (MarketOpsBacktestCalibrationRecommendation*).
export const MARKETOPS_BACKTEST_COMPARISON_RECOMMENDATIONS = [
  'needs_more_data',
  'regression_candidate',
  'improvement_candidate',
  'neutral_candidate',
  'manual_review_required',
] as const;

// Restrained token colors for comparison recommendation chips. regression -> red,
// improvement -> green, manual_review -> amber, needs_more_data -> gray,
// neutral -> slate. Unknown future values fall back to gray.
export const COMPARISON_RECOMMENDATION_STYLES: Record<string, string> = {
  regression_candidate: 'text-red-700 bg-red-50 border-red-200',
  improvement_candidate: 'text-emerald-700 bg-emerald-50 border-emerald-200',
  manual_review_required: 'text-amber-700 bg-amber-50 border-amber-200',
  needs_more_data: 'text-gray-700 bg-gray-50 border-gray-200',
  neutral_candidate: 'text-slate-700 bg-slate-50 border-slate-200',
};

export function comparisonRecommendationStyle(key: string): string {
  return COMPARISON_RECOMMENDATION_STYLES[key] ?? 'text-gray-700 bg-gray-50 border-gray-200';
}

// Humanize a comparison recommendation key: "neutral_candidate" -> "neutral candidate".
export function comparisonRecommendationLabel(key: string): string {
  return typeof key === 'string' ? key.replace(/_/g, ' ') : '';
}

export type ComparisonDeltaKind = 'count' | 'pct' | 'signed' | 'flag';

export interface ComparisonDeltaField {
  key: string;
  label: string;
  kind: ComparisonDeltaKind;
}

// Ordered key deltas surfaced from comparison_metrics.deltas, matching the spec
// display list. count -> signed integer; pct -> fraction rendered as signed
// percentage points (shares / signal yield / zero-input rate are 0–1
// fractions, like the existing calibration summary UI); signed -> plain signed
// decimal (policy_results_per_signal is a rate, not a 0–1 fraction); flag ->
// boolean dominant-recommendation change.
export const COMPARISON_DELTA_FIELDS: ComparisonDeltaField[] = [
  { key: 'run_count_delta', label: 'Run count', kind: 'count' },
  { key: 'zero_input_rate_delta', label: 'Zero-input rate', kind: 'pct' },
  { key: 'scanned_delta', label: 'Scanned', kind: 'count' },
  { key: 'signal_yield_delta', label: 'Signal yield', kind: 'pct' },
  { key: 'policy_results_per_signal_delta', label: 'Policy / signal', kind: 'signed' },
  { key: 'auto_accept_candidate_share_delta', label: 'Auto-accept', kind: 'pct' },
  { key: 'auto_reject_candidate_share_delta', label: 'Auto-reject', kind: 'pct' },
  { key: 'manual_review_required_share_delta', label: 'Manual review', kind: 'pct' },
  { key: 'supersede_existing_candidate_share_delta', label: 'Supersede', kind: 'pct' },
  { key: 'dominant_recommendation_changed', label: 'Dominant rec. changed', kind: 'flag' },
];

export interface ComparisonDeltaEntry {
  key: string;
  label: string;
  kind: ComparisonDeltaKind;
  raw: unknown;
  display: string;
  // True when the delta represents a material change (non-zero / flag set).
  changed: boolean;
}

// Strict finite-number extraction for single-value display. Unlike asNumber
// (which coerces to 0 for aggregation), this returns undefined for anything that
// is not a finite number or numeric string, so unparseable deltas render as '—'.
function asFiniteNumber(v: unknown): number | undefined {
  if (typeof v === 'number') return Number.isFinite(v) ? v : undefined;
  if (typeof v === 'string' && v.trim() !== '') {
    const n = Number(v);
    return Number.isFinite(n) ? n : undefined;
  }
  return undefined;
}

// Render a single delta value per its kind. Never throws: missing/malformed
// values collapse to '—'. pct deltas multiply the 0–1 fraction by 100 and
// render as signed percentage points with one decimal.
function formatDeltaValue(value: unknown, kind: ComparisonDeltaKind): string {
  if (kind === 'flag') {
    return value === true ? 'changed' : value === false ? 'unchanged' : '—';
  }
  const n = asFiniteNumber(value);
  if (n === undefined) return '—';
  if (kind === 'count') {
    return n > 0 ? `+${Math.round(n)}` : `${Math.round(n)}`;
  }
  if (kind === 'pct') {
    const pts = n * 100;
    return `${pts > 0 ? '+' : ''}${pts.toFixed(1)}%`;
  }
  // signed: plain signed decimal (rate deltas, not a 0–1 fraction).
  return `${n > 0 ? '+' : ''}${n.toFixed(2)}`;
}

// A delta is "changed" when it is material: non-zero for numeric kinds, true for
// the flag kind. Missing/malformed values are not changed.
function deltaIsChanged(value: unknown, kind: ComparisonDeltaKind): boolean {
  if (kind === 'flag') return value === true;
  const n = asFiniteNumber(value);
  return n !== undefined && n !== 0;
}

// Build the ordered, display-formatted delta rows for a comparison. Tolerates
// any deltas shape (missing/non-object) and never throws.
export function summarizeComparisonDeltas(deltas: unknown): ComparisonDeltaEntry[] {
  const record = isRecord(deltas) ? deltas : {};
  return COMPARISON_DELTA_FIELDS.map(({ key, label, kind }) => {
    const raw = record[key];
    return {
      key,
      label,
      kind,
      raw,
      display: formatDeltaValue(raw, kind),
      changed: deltaIsChanged(raw, kind),
    };
  });
}

// Extract comparison_metrics from a comparison record tolerantly. The gateway
// emits a parsed object; never JSON.parse.
export function comparisonMetrics(record: unknown): Record<string, unknown> {
  if (!isRecord(record)) return {};
  const metrics = (record as { comparison_metrics?: unknown }).comparison_metrics;
  return isRecord(metrics) ? metrics : {};
}
