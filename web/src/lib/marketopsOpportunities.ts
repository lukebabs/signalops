// Pure display helpers for the G139 MarketOps Opportunities workbench. Opportunity
// + hypothesis-evaluation JSON arrives already-parsed from the gateway (typed
// `unknown` on flexible fields). Narrow with type guards only; never JSON.parse.
// Missing/malformed values collapse to '' / 0 / null and must never throw. These
// power a read-only inspection surface — no review, trade, materialization, or
// build mutation.

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

function asStringArray(v: unknown): string[] {
  return Array.isArray(v) ? v.filter((x): x is string => typeof x === 'string') : [];
}

// Server-nullable numerics (omitempty pointers) -> number | null so the UI can
// render absent scores as unavailable rather than 0.
function asNullableNumber(v: unknown): number | null {
  if (typeof v === 'number' && Number.isFinite(v)) return v;
  if (typeof v === 'string' && v !== '') {
    const n = Number(v);
    if (Number.isFinite(n)) return n;
  }
  return null;
}

export interface MarketOpsOpportunityContribution {
  evaluationId: string;
  hypothesisKey: string;
  hypothesisVersion: string;
  domain: string;
  triggerScore: number | null;
  confidenceScore: number | null;
  qualityScore: number | null;
}

// Parse the embedded `contributions` array from opportunity_payload, ordered by
// trigger score descending (absent scores sort last). Unknown shapes collapse to
// an empty list — missing data renders as unavailable, not zero evidence.
export function parseOpportunityContributions(payload: unknown): MarketOpsOpportunityContribution[] {
  if (!isRecord(payload) || !Array.isArray(payload.contributions)) return [];
  return payload.contributions
    .filter(isRecord)
    .map((c) => ({
      evaluationId: asString(c.evaluation_id),
      hypothesisKey: asString(c.hypothesis_key),
      hypothesisVersion: asString(c.hypothesis_version),
      domain: asString(c.domain),
      triggerScore: asNullableNumber(c.trigger_score),
      confidenceScore: asNullableNumber(c.confidence_score),
      qualityScore: asNullableNumber(c.quality_score),
    }))
    .sort((a, b) => (b.triggerScore ?? -Infinity) - (a.triggerScore ?? -Infinity));
}

export function parseOverlapSuppressedIds(payload: unknown): string[] {
  if (!isRecord(payload)) return [];
  return asStringArray(payload.overlap_suppressed_evaluation_ids);
}

export function parseHypothesisFamilies(payload: unknown): string[] {
  if (!isRecord(payload)) return [];
  return asStringArray(payload.hypothesis_families);
}

export interface MarketOpsOpportunityView {
  opportunityId: string;
  tenantId: string;
  appId: string;
  assetId: string;
  symbol: string;
  openedSessionDate: string;
  lastEvaluatedDate: string;
  direction: string;
  horizon: string;
  lifecycleStatus: string;
  opportunityScore: number;
  confidenceScore: number;
  domainDiversityScore: number;
  conflictScore: number;
  hypothesisEvaluationIds: string[];
  conflictingEvaluationIds: string[];
  signalIds: string[];
  supportingEvidenceIds: string[];
  invalidatingEvidenceIds: string[];
  summary: string;
  version: number;
  researchOnly: boolean;
  buildRunId: string;
  deterministicKey: string;
  createdAt: string;
  updatedAt: string;
  // Parsed from opportunity_payload.
  contributions: MarketOpsOpportunityContribution[];
  overlapSuppressedEvaluationIds: string[];
  hypothesisFamilies: string[];
  scoringVersion: string;
  opportunityPayload: unknown;
}

const EMPTY_OPPORTUNITY: MarketOpsOpportunityView = {
  opportunityId: '',
  tenantId: '',
  appId: '',
  assetId: '',
  symbol: '',
  openedSessionDate: '',
  lastEvaluatedDate: '',
  direction: '',
  horizon: '',
  lifecycleStatus: '',
  opportunityScore: 0,
  confidenceScore: 0,
  domainDiversityScore: 0,
  conflictScore: 0,
  hypothesisEvaluationIds: [],
  conflictingEvaluationIds: [],
  signalIds: [],
  supportingEvidenceIds: [],
  invalidatingEvidenceIds: [],
  summary: '',
  version: 0,
  researchOnly: false,
  buildRunId: '',
  deterministicKey: '',
  createdAt: '',
  updatedAt: '',
  contributions: [],
  overlapSuppressedEvaluationIds: [],
  hypothesisFamilies: [],
  scoringVersion: '',
  opportunityPayload: {},
};

export function summarizeMarketOpsOpportunity(o: unknown): MarketOpsOpportunityView {
  if (!isRecord(o)) return { ...EMPTY_OPPORTUNITY };
  return {
    opportunityId: asString(o.opportunity_id),
    tenantId: asString(o.tenant_id),
    appId: asString(o.app_id),
    assetId: asString(o.asset_id),
    symbol: asString(o.symbol),
    openedSessionDate: asString(o.opened_session_date),
    lastEvaluatedDate: asString(o.last_evaluated_date),
    direction: asString(o.direction),
    horizon: asString(o.horizon),
    lifecycleStatus: asString(o.lifecycle_status),
    opportunityScore: asNumber(o.opportunity_score),
    confidenceScore: asNumber(o.confidence_score),
    domainDiversityScore: asNumber(o.domain_diversity_score),
    conflictScore: asNumber(o.conflict_score),
    hypothesisEvaluationIds: asStringArray(o.hypothesis_evaluation_ids),
    conflictingEvaluationIds: asStringArray(o.conflicting_evaluation_ids),
    signalIds: asStringArray(o.signal_ids),
    supportingEvidenceIds: asStringArray(o.supporting_evidence_ids),
    invalidatingEvidenceIds: asStringArray(o.invalidating_evidence_ids),
    summary: asString(o.summary),
    version: asNumber(o.version),
    researchOnly: asBool(o.research_only),
    buildRunId: asString(o.build_run_id),
    deterministicKey: asString(o.deterministic_key),
    createdAt: asString(o.created_at),
    updatedAt: asString(o.updated_at),
    contributions: parseOpportunityContributions(o.opportunity_payload),
    overlapSuppressedEvaluationIds: parseOverlapSuppressedIds(o.opportunity_payload),
    hypothesisFamilies: parseHypothesisFamilies(o.opportunity_payload),
    scoringVersion: isRecord(o.opportunity_payload) ? asString((o.opportunity_payload as Record<string, unknown>).scoring_version) : '',
    opportunityPayload: o.opportunity_payload ?? {},
  };
}

export interface MarketOpsHypothesisEvaluationView {
  evaluationId: string;
  hypothesisKey: string;
  hypothesisVersion: string;
  marketStateId: string;
  assetId: string;
  symbol: string;
  sessionDate: string;
  eligible: boolean;
  triggered: boolean;
  invalidated: boolean;
  triggerScore: number | null;
  confidenceScore: number | null;
  qualityScore: number | null;
  reasonCodes: string[];
  evidenceIds: string[];
}

const EMPTY_EVALUATION: MarketOpsHypothesisEvaluationView = {
  evaluationId: '',
  hypothesisKey: '',
  hypothesisVersion: '',
  marketStateId: '',
  assetId: '',
  symbol: '',
  sessionDate: '',
  eligible: false,
  triggered: false,
  invalidated: false,
  triggerScore: null,
  confidenceScore: null,
  qualityScore: null,
  reasonCodes: [],
  evidenceIds: [],
};

export function summarizeMarketOpsHypothesisEvaluation(e: unknown): MarketOpsHypothesisEvaluationView {
  if (!isRecord(e)) return { ...EMPTY_EVALUATION };
  return {
    evaluationId: asString(e.evaluation_id),
    hypothesisKey: asString(e.hypothesis_key),
    hypothesisVersion: asString(e.hypothesis_version),
    marketStateId: asString(e.market_state_id),
    assetId: asString(e.asset_id),
    symbol: asString(e.symbol),
    sessionDate: asString(e.session_date),
    eligible: asBool(e.eligible),
    triggered: asBool(e.triggered),
    invalidated: asBool(e.invalidated),
    triggerScore: asNullableNumber(e.trigger_score),
    confidenceScore: asNullableNumber(e.confidence_score),
    qualityScore: asNullableNumber(e.quality_score),
    reasonCodes: asStringArray(e.reason_codes),
    evidenceIds: asStringArray(e.evidence_ids),
  };
}

// Translate a known rejection reason token/prefix into concise operator language.
// The raw token is preserved by the caller for tooltips/secondary text.
export function reasonCodeLabel(token: string): string {
  if (token === 'eligible_not_triggered') return 'Eligible, not triggered';
  if (token === 'triggered_research_only') return 'Triggered, research-only';
  if (token.startsWith('threshold_not_met:')) return `Threshold not met (${token.slice('threshold_not_met:'.length)})`;
  if (token.startsWith('threshold_not_met')) return 'Threshold not met';
  return token || 'unknown';
}

export interface ReasonAggregationEntry {
  token: string;
  label: string;
  count: number;
}

export interface ReasonAggregation {
  evaluated: number;
  eligible: number;
  triggered: number;
  entries: ReasonAggregationEntry[]; // top N by count desc, then token asc
}

// Aggregate rejection reason codes across a scoped set of hypothesis evaluations
// for empty-queue diagnostics. Counts evaluated/eligible/triggered and ranks the
// most frequent reason tokens. Never throws.
export function aggregateOpportunityRejectionReasons(
  evaluations: MarketOpsHypothesisEvaluationView[],
  topN = 5,
): ReasonAggregation {
  const counts = new Map<string, number>();
  let eligible = 0;
  let triggered = 0;
  for (const e of evaluations) {
    if (e.eligible) eligible++;
    if (e.triggered) triggered++;
    for (const token of e.reasonCodes) {
      const key = token || 'unknown';
      counts.set(key, (counts.get(key) ?? 0) + 1);
    }
  }
  const entries = Array.from(counts.entries())
    .map(([token, count]) => ({ token, label: reasonCodeLabel(token), count }))
    .sort((a, b) => b.count - a.count || a.token.localeCompare(b.token))
    .slice(0, topN);
  return { evaluated: evaluations.length, eligible, triggered, entries };
}

// Format a (0..1) score to a fixed precision, or `—` when absent.
export function formatScore(value: number | null | undefined, digits = 2): string {
  if (value === null || value === undefined) return '—';
  if (typeof value === 'number' && Number.isFinite(value)) return value.toFixed(digits);
  return '—';
}

// Restrained opportunity lifecycle badge tones. Color is secondary — the route
// always pairs tone with the lifecycle text. active/strengthening -> positive;
// weakening -> warning; invalidated -> error; resolved/expired -> muted.
const LIFECYCLE_STYLES: Record<string, string> = {
  emerging: 'border-blue-200 bg-blue-50 text-blue-700',
  active: 'border-emerald-200 bg-emerald-50 text-emerald-700',
  strengthening: 'border-emerald-200 bg-emerald-50 text-emerald-700',
  weakening: 'border-amber-200 bg-amber-50 text-amber-700',
  invalidated: 'border-red-200 bg-red-50 text-red-700',
  resolved: 'border-gray-200 bg-gray-100 text-gray-500',
  expired: 'border-gray-200 bg-gray-100 text-gray-400',
};

export function opportunityLifecycleStyle(status: string): string {
  return LIFECYCLE_STYLES[status] ?? 'border-gray-200 bg-gray-50 text-gray-600';
}

// Direction display label + secondary tone. The route renders an icon/text so
// direction is never communicated by color alone.
export function directionLabel(direction: string): string {
  switch (direction) {
    case 'upside':
      return 'Upside';
    case 'downside':
      return 'Downside';
    case 'non_directional':
      return 'Non-directional';
    default:
      return direction || '—';
  }
}

export function directionTone(direction: string): string {
  switch (direction) {
    case 'upside':
      return 'text-emerald-700';
    case 'downside':
      return 'text-red-700';
    case 'non_directional':
      return 'text-gray-600';
    default:
      return 'text-gray-600';
  }
}
