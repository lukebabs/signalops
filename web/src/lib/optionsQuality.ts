// Pure helpers for the G132 MarketOps options quality visibility. G130 persists
// options-distribution quality metadata; G131 uses it to block low-quality
// call/put OI ratio results from the proposal queue. These helpers extract the
// quality fields from the three places they live (distribution `metrics`,
// algorithm `result_payload`, and proposal `proposal_payload`) with type guards
// only — never JSON.parse. Missing/malformed values collapse to `unknown` and
// must never throw. These power read-only visibility; they perform no gating,
// ingestion, or mutation.

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null && !Array.isArray(v);
}

function asString(v: unknown): string {
  return typeof v === 'string' ? v : '';
}

function asNumber(v: unknown): number {
  if (typeof v === 'number' && Number.isFinite(v)) return v;
  if (typeof v === 'string' && v !== '') {
    const n = Number(v);
    if (Number.isFinite(n)) return n;
  }
  return 0;
}

function asBool(v: unknown): boolean {
  return v === true;
}

// Permissive union: the backend emits usable / partial_zero / all_zero /
// denominator_zero / empty / missing; any other or absent value renders as
// `unknown`. Kept open (| string) so future tokens don't break the UI.
export type MarketOpsOptionsRatioQuality =
  | 'usable'
  | 'partial_zero'
  | 'all_zero'
  | 'denominator_zero'
  | 'empty'
  | 'missing'
  | 'unknown'
  | string;

export interface MarketOpsOptionsQualityFields {
  openInterestQuality?: string;
  openInterestZeroCount?: number;
  openInterestPositiveCount?: number;
  openInterestZeroRate?: number;
  callPutOiDenominatorIsZero?: boolean;
  callPutOiRatioQuality?: string;
}

const KNOWN_QUALITY = new Set<string>([
  'usable',
  'partial_zero',
  'all_zero',
  'denominator_zero',
  'empty',
  'missing',
]);

// Map a raw ratio-quality token to the known set, defaulting to `unknown` for
// anything absent or unrecognized (so missing fields never read as usable).
export function normalizeRatioQuality(value: unknown): MarketOpsOptionsRatioQuality {
  const s = asString(value);
  return KNOWN_QUALITY.has(s) ? s : 'unknown';
}

// Extract the six quality fields from a backend object (distribution `metrics`
// or algorithm `result_payload`). A field is set only when the key is present,
// so callers can distinguish "0" from "absent" (absent zero-rate => unknown).
export function getOptionsQualityFields(value: unknown): MarketOpsOptionsQualityFields {
  if (!isRecord(value)) return {};
  const f: MarketOpsOptionsQualityFields = {};
  if ('open_interest_quality' in value) f.openInterestQuality = asString(value.open_interest_quality);
  if ('open_interest_zero_count' in value) f.openInterestZeroCount = asNumber(value.open_interest_zero_count);
  if ('open_interest_positive_count' in value) f.openInterestPositiveCount = asNumber(value.open_interest_positive_count);
  if ('open_interest_zero_rate' in value) f.openInterestZeroRate = asNumber(value.open_interest_zero_rate);
  if ('call_put_oi_denominator_is_zero' in value) f.callPutOiDenominatorIsZero = asBool(value.call_put_oi_denominator_is_zero);
  if ('call_put_oi_ratio_quality' in value) f.callPutOiRatioQuality = asString(value.call_put_oi_ratio_quality);
  return f;
}

// Format an open-interest zero-rate (0..1) as a percentage, or `—` when absent.
export function formatZeroRate(value: unknown): string {
  if (typeof value === 'number' && Number.isFinite(value)) return `${(value * 100).toFixed(1)}%`;
  return '—';
}

// True only for an options-distribution call/put OI ratio payload — used to avoid
// showing options-specific quality warnings on unrelated algorithm results.
export function isOptionsRatioAlgorithmPayload(value: unknown): boolean {
  if (!isRecord(value)) return false;
  return value.dataset === 'options_distribution_daily' && value.feature === 'call_put_open_interest_ratio';
}

export function isOptionsRatioQualityUsable(quality: MarketOpsOptionsRatioQuality): boolean {
  return quality === 'usable';
}

export interface AlgorithmResultQualitySummary {
  isOptionsRatio: boolean;
  ratioQuality: MarketOpsOptionsRatioQuality;
  quality: MarketOpsOptionsQualityFields;
}

// Extract options-ratio quality from an algorithm result_payload. Returns
// isOptionsRatio=false (and unknown quality) for any non-options payload so the
// UI shows no options-specific warning on unrelated results.
export function summarizeAlgorithmResultQuality(resultPayload: unknown): AlgorithmResultQualitySummary {
  if (!isOptionsRatioAlgorithmPayload(resultPayload)) {
    return { isOptionsRatio: false, ratioQuality: 'unknown', quality: {} };
  }
  const quality = getOptionsQualityFields(resultPayload);
  return {
    isOptionsRatio: true,
    ratioQuality: normalizeRatioQuality(quality.callPutOiRatioQuality),
    quality,
  };
}

export interface ProposalQualityGateSummary {
  present: boolean;
  passed: boolean | null;
  policy: string;
  isOptionsRatio: boolean;
  ratioQuality: MarketOpsOptionsRatioQuality;
  openInterestQuality: string;
  quality: MarketOpsOptionsQualityFields;
}

const EMPTY_GATE: ProposalQualityGateSummary = {
  present: false,
  passed: null,
  policy: '',
  isOptionsRatio: false,
  ratioQuality: 'unknown',
  openInterestQuality: '',
  quality: {},
};

// Extract the G131 quality gate + nested algorithm-result ratio quality from a
// proposal_payload. Reads proposal_payload.quality_gate.{passed,policy} and the
// nested proposal_payload.algorithm_result.payload quality fields. Never throws;
// older proposals without a gate return present=false.
export function summarizeProposalQualityGate(proposalPayload: unknown): ProposalQualityGateSummary {
  if (!isRecord(proposalPayload)) return { ...EMPTY_GATE };
  const payload = isRecord(proposalPayload.algorithm_result)
    ? (proposalPayload.algorithm_result as Record<string, unknown>).payload
    : undefined;
  const isOptionsRatio = isOptionsRatioAlgorithmPayload(payload);
  const quality = getOptionsQualityFields(payload);
  const ratioQuality = isOptionsRatio ? normalizeRatioQuality(quality.callPutOiRatioQuality) : 'unknown';
  const gate = proposalPayload.quality_gate;
  if (!isRecord(gate)) {
    return { ...EMPTY_GATE, isOptionsRatio, ratioQuality, openInterestQuality: quality.openInterestQuality ?? '', quality };
  }
  return {
    present: true,
    passed: typeof gate.passed === 'boolean' ? gate.passed : null,
    policy: asString(gate.policy),
    isOptionsRatio,
    ratioQuality,
    openInterestQuality: quality.openInterestQuality ?? '',
    quality,
  };
}

// Restrained options ratio-quality badge tones. usable -> success; partial_zero
// -> warning; all_zero / denominator_zero / empty / missing -> blocked/error;
// unknown -> muted. Unknown future tokens fall back to muted gray.
const QUALITY_STYLES: Record<string, string> = {
  usable: 'border-emerald-200 bg-emerald-50 text-emerald-700',
  partial_zero: 'border-amber-200 bg-amber-50 text-amber-700',
  all_zero: 'border-red-200 bg-red-50 text-red-700',
  denominator_zero: 'border-red-200 bg-red-50 text-red-700',
  empty: 'border-red-200 bg-red-50 text-red-700',
  missing: 'border-red-200 bg-red-50 text-red-700',
  unknown: 'border-gray-200 bg-gray-100 text-gray-500',
};

export function optionsRatioQualityStyle(quality: string): string {
  return QUALITY_STYLES[quality] ?? QUALITY_STYLES.unknown;
}
