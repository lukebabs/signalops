// Pure parsing helpers for the MarketOps back-test workspace (G081 frontend).
//
// Back-test run metrics/filters/parameters arrive as already-parsed JSON from
// the gateway (typed `unknown` on MarketOpsBacktestRun). Narrow with type
// guards only; never JSON.parse. Missing/malformed values collapse to 0 / ''
// and must never throw. Recommendation values mirror the four backend policy
// constants in internal/marketops/dsm/policy.go.
import type { MarketOpsBacktestPolicyResult } from '../types';

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
