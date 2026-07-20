// Pure helpers for the G148-C MarketOps intelligence readiness view. The
// aggregate + per-symbol JSON arrives already-parsed from the gateway (typed
// unknown on flexible fields). Narrow with type guards only; never JSON.parse.
// Missing values collapse to '' / 0 / null and never throw. This is a read-only
// research-rollout view: it never implies production readiness and exposes no
// execution controls. A symbol with an empty latest_market_state_id is
// unobserved — its state columns render as "Not observed," never zero coverage.

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

function asNullableString(v: unknown): string | null {
  return typeof v === 'string' && v !== '' ? v : null;
}

export interface MarketOpsIntelligenceReadinessSymbolView {
  resultId: string;
  runId: string;
  universeGroup: string;
  symbol: string;
  assetId: string;
  latestMarketStateId: string;
  latestStateDate: string | null;
  latestStateSchemaVersion: string;
  latestStateQuality: string;
  latestStateCompleteness: number;
  requiredFeatureCoverage: number;
  surfaceCoverage: number;
  evaluationCount: number;
  eligibleCount: number;
  triggeredCount: number;
  evaluationRejectionReasons: string[];
  opportunityCount: number;
  pendingOutcomeCount: number;
  maturedOutcomeCount: number;
  exactCalibrationCount: number;
  calibrationBelowMinimum: boolean;
  coverageState: string;
  evaluationState: string;
  governanceState: string;
  calibrationState: string;
  outcomeState: string;
  rolloutStatus: string;
  readinessReasons: string[];
  stageStatus: unknown;
  stageErrors: unknown;
  inputCoverage: unknown;
  proposalStatusCounts: unknown;
  createdAt: string;
  updatedAt: string;
  observed: boolean;
}

const EMPTY_SYMBOL: MarketOpsIntelligenceReadinessSymbolView = {
  resultId: '', runId: '', universeGroup: '', symbol: '', assetId: '',
  latestMarketStateId: '', latestStateDate: null, latestStateSchemaVersion: '',
  latestStateQuality: '', latestStateCompleteness: 0, requiredFeatureCoverage: 0,
  surfaceCoverage: 0, evaluationCount: 0, eligibleCount: 0, triggeredCount: 0,
  evaluationRejectionReasons: [], opportunityCount: 0, pendingOutcomeCount: 0,
  maturedOutcomeCount: 0, exactCalibrationCount: 0, calibrationBelowMinimum: false,
  coverageState: '', evaluationState: '', governanceState: '', calibrationState: '',
  outcomeState: '', rolloutStatus: '', readinessReasons: [], stageStatus: {},
  stageErrors: {}, inputCoverage: {}, proposalStatusCounts: {}, createdAt: '', updatedAt: '',
  observed: false,
};

export function summarizeMarketOpsIntelligenceReadinessSymbol(s: unknown): MarketOpsIntelligenceReadinessSymbolView {
  if (!isRecord(s)) return { ...EMPTY_SYMBOL };
  const latestMarketStateId = asString(s.latest_market_state_id);
  return {
    resultId: asString(s.result_id),
    runId: asString(s.run_id),
    universeGroup: asString(s.universe_group),
    symbol: asString(s.symbol),
    assetId: asString(s.asset_id),
    latestMarketStateId,
    latestStateDate: asNullableString(s.latest_state_date),
    latestStateSchemaVersion: asString(s.latest_state_schema_version),
    latestStateQuality: asString(s.latest_state_quality),
    latestStateCompleteness: asNumber(s.latest_state_completeness),
    requiredFeatureCoverage: asNumber(s.required_feature_coverage),
    surfaceCoverage: asNumber(s.surface_coverage),
    evaluationCount: asNumber(s.evaluation_count),
    eligibleCount: asNumber(s.eligible_count),
    triggeredCount: asNumber(s.triggered_count),
    evaluationRejectionReasons: asStringArray(s.evaluation_rejection_reasons),
    opportunityCount: asNumber(s.opportunity_count),
    pendingOutcomeCount: asNumber(s.pending_outcome_count),
    maturedOutcomeCount: asNumber(s.matured_outcome_count),
    exactCalibrationCount: asNumber(s.exact_calibration_count),
    calibrationBelowMinimum: asBool(s.calibration_below_minimum),
    coverageState: asString(s.coverage_state),
    evaluationState: asString(s.evaluation_state),
    governanceState: asString(s.governance_state),
    calibrationState: asString(s.calibration_state),
    outcomeState: asString(s.outcome_state),
    rolloutStatus: asString(s.rollout_status),
    readinessReasons: asStringArray(s.readiness_reasons),
    stageStatus: s.stage_status ?? {},
    stageErrors: s.stage_errors ?? {},
    inputCoverage: s.input_coverage ?? {},
    proposalStatusCounts: s.proposal_status_counts ?? {},
    createdAt: asString(s.created_at),
    updatedAt: asString(s.updated_at),
    observed: latestMarketStateId !== '',
  };
}

export interface ReadinessCountEntry {
  key: string;
  count: number;
}

export interface MarketOpsIntelligenceReadinessAggregateView {
  symbolCount: number;
  productionReadySupported: boolean;
  latestSessionDate: string | null;
  coverageState: ReadinessCountEntry[];
  evaluationState: ReadinessCountEntry[];
  governanceState: ReadinessCountEntry[];
  calibrationState: ReadinessCountEntry[];
  outcomeState: ReadinessCountEntry[];
  rolloutStatus: ReadinessCountEntry[];
}

const EMPTY_AGGREGATE: MarketOpsIntelligenceReadinessAggregateView = {
  symbolCount: 0,
  productionReadySupported: false,
  latestSessionDate: null,
  coverageState: [],
  evaluationState: [],
  governanceState: [],
  calibrationState: [],
  outcomeState: [],
  rolloutStatus: [],
};

function countEntries(map: unknown): ReadinessCountEntry[] {
  if (!isRecord(map)) return [];
  return Object.entries(map)
    .filter(([, c]) => typeof c === 'number')
    .map(([key, count]) => ({ key, count: count as number }))
    .sort((a, b) => b.count - a.count || a.key.localeCompare(b.key));
}

export function summarizeMarketOpsIntelligenceReadinessAggregate(a: unknown): MarketOpsIntelligenceReadinessAggregateView {
  if (!isRecord(a)) return { ...EMPTY_AGGREGATE };
  const dc = isRecord(a.dimension_counts) ? a.dimension_counts : {};
  return {
    symbolCount: asNumber(a.symbol_count),
    productionReadySupported: asBool(a.production_ready_supported),
    latestSessionDate: asNullableString(a.latest_session_date),
    coverageState: countEntries(dc.coverage_state),
    evaluationState: countEntries(dc.evaluation_state),
    governanceState: countEntries(dc.governance_state),
    calibrationState: countEntries(dc.calibration_state),
    outcomeState: countEntries(dc.outcome_state),
    rolloutStatus: countEntries(dc.rollout_status),
  };
}

// Restrained rollout-status tones. Increasing research readiness shifts toward
// emerald, but review_ready is still research-only (never production). Color is
// always paired with the status text.
const ROLLOUT_STYLES: Record<string, string> = {
  not_observed: 'border-gray-200 bg-gray-100 text-gray-500',
  inspection_ready: 'border-blue-200 bg-blue-50 text-blue-700',
  research_evaluation_ready: 'border-amber-200 bg-amber-50 text-amber-700',
  review_ready: 'border-emerald-200 bg-emerald-50 text-emerald-700',
  blocked: 'border-red-200 bg-red-50 text-red-700',
};

export function rolloutStatusStyle(status: string): string {
  return ROLLOUT_STYLES[status] ?? 'border-gray-200 bg-gray-50 text-gray-600';
}

// Generic per-state tone for the five readiness dimensions. Color supplements
// the state text only — never the sole signal.
export function dimensionStateStyle(state: string): string {
  const s = (state || '').toLowerCase();
  if (s === 'available' || s === 'usable' || s === 'ready' || s === 'complete') {
    return 'border-emerald-200 bg-emerald-50 text-emerald-700';
  }
  if (s === 'blocked' || s === 'invalidated' || s === 'rejected') {
    return 'border-red-200 bg-red-50 text-red-700';
  }
  if (s.includes('below_minimum') || s === 'partial' || s === 'incomplete' || s === 'pending' || s === 'proposal_pending' || s === 'evaluated_no_trigger') {
    return 'border-amber-200 bg-amber-50 text-amber-700';
  }
  if (s === 'research_only') return 'border-blue-200 bg-blue-50 text-blue-700';
  return 'border-gray-200 bg-gray-100 text-gray-500';
}

// Format a 0..1 coverage ratio as a percentage, or `—` when absent/zero-derived
// from an unobserved symbol. Callers pass the observed flag to avoid rendering
// "0%" for missing state.
export function formatCoverageRatio(value: number, observed: boolean): string {
  if (!observed) return 'Not observed';
  if (!Number.isFinite(value)) return '—';
  return `${(value * 100).toFixed(0)}%`;
}

export function isSymbolObserved(s: MarketOpsIntelligenceReadinessSymbolView): boolean {
  return s.observed;
}
