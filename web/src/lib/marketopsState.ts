// Pure helpers for the G147 Market State analyst experience. Market-state,
// feature definition/observation, transition, outcome, and disposition JSON
// arrives already-parsed from the gateway (typed `unknown` on flexible fields).
// Narrow with type guards only; never JSON.parse. Missing values collapse to
// null/'' and must never throw. Pointer/optional numerics+booleans stay nullable
// — absent vs zero vs false is material. Read-only inspection surface; the only
// mutation is the append-only opportunity disposition, which lives in the route.

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

function asNullableNumber(v: unknown): number | null {
  if (typeof v === 'number' && Number.isFinite(v)) return v;
  return null;
}

function asNullableInt(v: unknown): number | null {
  const n = asNullableNumber(v);
  return n === null ? null : Math.trunc(n);
}

function asNullableBool(v: unknown): boolean | null {
  return typeof v === 'boolean' ? v : null;
}

function asNullableString(v: unknown): string | null {
  return typeof v === 'string' ? v : null;
}

function dimNumber(d: unknown, key: string): number | undefined {
  if (!isRecord(d)) return undefined;
  const v = d[key];
  return typeof v === 'number' && Number.isFinite(v) ? v : undefined;
}

function dimString(d: unknown, key: string): string | undefined {
  if (!isRecord(d)) return undefined;
  const v = d[key];
  return typeof v === 'string' ? v : undefined;
}

export interface MarketOpsMarketStateView {
  marketStateId: string;
  tenantId: string;
  appId: string;
  assetId: string;
  symbol: string;
  sessionDate: string;
  asOfTime: string;
  stateSchemaVersion: string;
  featureCount: number;
  requiredFeatureCount: number;
  completenessRatio: number;
  qualityState: string;
  qualityScore: number | null;
  eligibleHypotheses: string[];
  buildRunId: string;
  deterministicKey: string;
  createdAt: string;
  statePayload: unknown;
  qualitySummary: unknown;
}

export function summarizeMarketOpsState(s: unknown): MarketOpsMarketStateView {
  if (!isRecord(s)) {
    return {
      marketStateId: '', tenantId: '', appId: '', assetId: '', symbol: '', sessionDate: '', asOfTime: '',
      stateSchemaVersion: '', featureCount: 0, requiredFeatureCount: 0, completenessRatio: 0,
      qualityState: '', qualityScore: null, eligibleHypotheses: [], buildRunId: '', deterministicKey: '',
      createdAt: '', statePayload: {}, qualitySummary: {},
    };
  }
  return {
    marketStateId: asString(s.market_state_id),
    tenantId: asString(s.tenant_id),
    appId: asString(s.app_id),
    assetId: asString(s.asset_id),
    symbol: asString(s.symbol),
    sessionDate: asString(s.session_date),
    asOfTime: asString(s.as_of_time),
    stateSchemaVersion: asString(s.state_schema_version),
    featureCount: asNumber(s.feature_count),
    requiredFeatureCount: asNumber(s.required_feature_count),
    completenessRatio: asNumber(s.completeness_ratio),
    qualityState: asString(s.quality_state),
    qualityScore: asNullableNumber(s.quality_score),
    eligibleHypotheses: asStringArray(s.eligible_hypotheses),
    buildRunId: asString(s.build_run_id),
    deterministicKey: asString(s.deterministic_key),
    createdAt: asString(s.created_at),
    statePayload: s.state_payload ?? {},
    qualitySummary: s.quality_summary ?? {},
  };
}

export interface MarketOpsFeatureDefinitionView {
  featureKey: string;
  featureVersion: string;
  domain: string;
  title: string;
  description: string;
  valueType: string;
  unit: string | null;
  status: string;
}

export function summarizeMarketOpsFeatureDefinition(d: unknown): MarketOpsFeatureDefinitionView {
  if (!isRecord(d)) {
    return { featureKey: '', featureVersion: '', domain: '', title: '', description: '', valueType: '', unit: null, status: '' };
  }
  return {
    featureKey: asString(d.feature_key),
    featureVersion: asString(d.feature_version),
    domain: asString(d.domain),
    title: asString(d.title),
    description: asString(d.description),
    valueType: asString(d.value_type),
    unit: asNullableString(d.unit),
    status: asString(d.status),
  };
}

export interface MarketOpsFeatureObservationView {
  featureObservationId: string;
  sessionDate: string;
  featureKey: string;
  featureVersion: string;
  domain: string;
  dimensions: unknown;
  numericValue: number | null;
  textValue: string | null;
  booleanValue: boolean | null;
  qualityState: string;
  qualityScore: number | null;
  qualityDetails: unknown;
  sourceEventIds: string[];
  sourceArtifactIds: string[];
  calculationRunId: string;
  deterministicKey: string;
}

export function summarizeMarketOpsFeatureObservation(o: unknown): MarketOpsFeatureObservationView {
  if (!isRecord(o)) {
    return {
      featureObservationId: '', sessionDate: '', featureKey: '', featureVersion: '', domain: '',
      dimensions: {}, numericValue: null, textValue: null, booleanValue: null, qualityState: '',
      qualityScore: null, qualityDetails: {}, sourceEventIds: [], sourceArtifactIds: [],
      calculationRunId: '', deterministicKey: '',
    };
  }
  return {
    featureObservationId: asString(o.feature_observation_id),
    sessionDate: asString(o.session_date),
    featureKey: asString(o.feature_key),
    featureVersion: asString(o.feature_version),
    domain: asString(o.domain),
    dimensions: o.dimensions ?? {},
    numericValue: asNullableNumber(o.numeric_value),
    textValue: asNullableString(o.text_value),
    booleanValue: asNullableBool(o.boolean_value),
    qualityState: asString(o.quality_state),
    qualityScore: asNullableNumber(o.quality_score),
    qualityDetails: o.quality_details ?? {},
    sourceEventIds: asStringArray(o.source_event_ids),
    sourceArtifactIds: asStringArray(o.source_artifact_ids),
    calculationRunId: asString(o.calculation_run_id),
    deterministicKey: asString(o.deterministic_key),
  };
}

export interface MarketOpsStateTransitionView {
  transitionId: string;
  sessionDate: string;
  asOfTime: string;
  currentStateId: string;
  baselineStateId: string | null;
  featureKey: string;
  featureVersion: string;
  dimensions: unknown;
  transitionType: string;
  lookbackSessions: number | null;
  currentValue: number | null;
  baselineValue: number | null;
  transitionValue: number | null;
  zscore: number | null;
  percentile: number | null;
  persistenceSessions: number | null;
  direction: string | null;
  qualityState: string;
  transitionPayload: unknown;
  calculationRunId: string;
  deterministicKey: string;
}

export function summarizeMarketOpsStateTransition(t: unknown): MarketOpsStateTransitionView {
  if (!isRecord(t)) {
    return {
      transitionId: '', sessionDate: '', asOfTime: '', currentStateId: '', baselineStateId: null, featureKey: '',
      featureVersion: '', dimensions: {}, transitionType: '', lookbackSessions: null, currentValue: null,
      baselineValue: null, transitionValue: null, zscore: null, percentile: null, persistenceSessions: null,
      direction: null, qualityState: '', transitionPayload: {}, calculationRunId: '', deterministicKey: '',
    };
  }
  return {
    transitionId: asString(t.transition_id),
    sessionDate: asString(t.session_date),
    asOfTime: asString(t.as_of_time),
    currentStateId: asString(t.current_state_id),
    baselineStateId: asNullableString(t.baseline_state_id),
    featureKey: asString(t.feature_key),
    featureVersion: asString(t.feature_version),
    dimensions: t.dimensions ?? {},
    transitionType: asString(t.transition_type),
    lookbackSessions: asNullableInt(t.lookback_sessions),
    currentValue: asNullableNumber(t.current_value),
    baselineValue: asNullableNumber(t.baseline_value),
    transitionValue: asNullableNumber(t.transition_value),
    zscore: asNullableNumber(t.zscore),
    percentile: asNullableNumber(t.percentile),
    persistenceSessions: asNullableInt(t.persistence_sessions),
    direction: asNullableString(t.direction),
    qualityState: asString(t.quality_state),
    transitionPayload: t.transition_payload ?? {},
    calculationRunId: asString(t.calculation_run_id),
    deterministicKey: asString(t.deterministic_key),
  };
}

export interface MarketOpsOutcomeView {
  outcomeId: string;
  sourceType: string;
  sourceId: string;
  hypothesisKey: string;
  hypothesisVersion: string;
  symbol: string;
  direction: string;
  originSessionDate: string;
  horizonSessions: number;
  maturedSessionDate: string | null;
  outcomeStatus: string;
  forwardReturn: number | null;
  maxFavorableExcursion: number | null;
  maxAdverseExcursion: number | null;
  maximumDrawdown: number | null;
  realizedVolChange: number | null;
  directionalHit: boolean | null;
  thresholdHit: boolean | null;
  daysToThreshold: number | null;
  calculationVersion: string;
  calculationRunId: string;
}

export function summarizeMarketOpsOutcome(o: unknown): MarketOpsOutcomeView {
  if (!isRecord(o)) {
    return {
      outcomeId: '', sourceType: '', sourceId: '', hypothesisKey: '', hypothesisVersion: '', symbol: '',
      direction: '', originSessionDate: '', horizonSessions: 0, maturedSessionDate: null, outcomeStatus: '',
      forwardReturn: null, maxFavorableExcursion: null, maxAdverseExcursion: null, maximumDrawdown: null,
      realizedVolChange: null, directionalHit: null, thresholdHit: null, daysToThreshold: null,
      calculationVersion: '', calculationRunId: '',
    };
  }
  return {
    outcomeId: asString(o.outcome_id),
    sourceType: asString(o.source_type),
    sourceId: asString(o.source_id),
    hypothesisKey: asString(o.hypothesis_key),
    hypothesisVersion: asString(o.hypothesis_version),
    symbol: asString(o.symbol),
    direction: asString(o.direction),
    originSessionDate: asString(o.origin_session_date),
    horizonSessions: asNumber(o.horizon_sessions),
    maturedSessionDate: asNullableString(o.matured_session_date),
    outcomeStatus: asString(o.outcome_status),
    forwardReturn: asNullableNumber(o.forward_return),
    maxFavorableExcursion: asNullableNumber(o.max_favorable_excursion),
    maxAdverseExcursion: asNullableNumber(o.max_adverse_excursion),
    maximumDrawdown: asNullableNumber(o.maximum_drawdown),
    realizedVolChange: asNullableNumber(o.realized_vol_change),
    directionalHit: asNullableBool(o.directional_hit),
    thresholdHit: asNullableBool(o.threshold_hit),
    daysToThreshold: asNullableInt(o.days_to_threshold),
    calculationVersion: asString(o.calculation_version),
    calculationRunId: asString(o.calculation_run_id),
  };
}

export interface MarketOpsOpportunityDispositionView {
  dispositionId: string;
  opportunityId: string;
  disposition: string;
  actor: string;
  note: string;
  metadata: unknown;
  createdAt: string;
}

export function summarizeMarketOpsOpportunityDisposition(d: unknown): MarketOpsOpportunityDispositionView {
  if (!isRecord(d)) {
    return { dispositionId: '', opportunityId: '', disposition: '', actor: '', note: '', metadata: {}, createdAt: '' };
  }
  return {
    dispositionId: asString(d.disposition_id),
    opportunityId: asString(d.opportunity_id),
    disposition: asString(d.disposition),
    actor: asString(d.actor),
    note: asString(d.note),
    metadata: d.metadata ?? {},
    createdAt: asString(d.created_at),
  };
}

export interface MarketOpsHypothesisEvaluationStateView {
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
  magnitudeScore: number | null;
  rarityScore: number | null;
  persistenceScore: number | null;
  corroborationScore: number | null;
  qualityScore: number | null;
  reasonCodes: string[];
  evidenceIds: string[];
  evaluationPayload: unknown;
  evaluationRunId: string;
  deterministicKey: string;
  createdAt: string;
}

// Full hypothesis-evaluation view with all nullable scores surfaced (the G139
// contribution summarizer only exposes trigger/confidence/quality). Used by the
// G147 Hypotheses tab to show every score the backend persisted, or unavailable.
export function summarizeMarketOpsHypothesisEvaluation(e: unknown): MarketOpsHypothesisEvaluationStateView {
  if (!isRecord(e)) {
    return {
      evaluationId: '', hypothesisKey: '', hypothesisVersion: '', marketStateId: '', assetId: '', symbol: '',
      sessionDate: '', eligible: false, triggered: false, invalidated: false, triggerScore: null,
      confidenceScore: null, magnitudeScore: null, rarityScore: null, persistenceScore: null,
      corroborationScore: null, qualityScore: null, reasonCodes: [], evidenceIds: [], evaluationPayload: {},
      evaluationRunId: '', deterministicKey: '', createdAt: '',
    };
  }
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
    magnitudeScore: asNullableNumber(e.magnitude_score),
    rarityScore: asNullableNumber(e.rarity_score),
    persistenceScore: asNullableNumber(e.persistence_score),
    corroborationScore: asNullableNumber(e.corroboration_score),
    qualityScore: asNullableNumber(e.quality_score),
    reasonCodes: asStringArray(e.reason_codes),
    evidenceIds: asStringArray(e.evidence_ids),
    evaluationPayload: e.evaluation_payload ?? {},
    evaluationRunId: asString(e.evaluation_run_id),
    deterministicKey: asString(e.deterministic_key),
    createdAt: asString(e.created_at),
  };
}

// Canonical seven-cell surface. Core ATM observations use explicit feature keys
// with {} dimensions; normalized ATM changes use surface_cell=atm, delta=.50;
// wing observations use option_type + target_delta=.25 + target_dte.
export interface SurfaceCellSpec {
  id: string;
  label: string;
  optionType?: string;
  targetDte: number;
  targetDelta: number;
}

export const SURFACE_CELLS: SurfaceCellSpec[] = [
  { id: 'atm-30', label: '30-DTE ATM', targetDte: 30, targetDelta: 0.5 },
  { id: 'atm-60', label: '60-DTE ATM', targetDte: 60, targetDelta: 0.5 },
  { id: 'atm-90', label: '90-DTE ATM', targetDte: 90, targetDelta: 0.5 },
  { id: 'put-30-25d', label: '30-DTE 25Δ Put', optionType: 'put', targetDte: 30, targetDelta: 0.25 },
  { id: 'put-60-25d', label: '60-DTE 25Δ Put', optionType: 'put', targetDte: 60, targetDelta: 0.25 },
  { id: 'call-30-25d', label: '30-DTE 25Δ Call', optionType: 'call', targetDte: 30, targetDelta: 0.25 },
  { id: 'call-60-25d', label: '60-DTE 25Δ Call', optionType: 'call', targetDte: 60, targetDelta: 0.25 },
];

export function observationSurfaceCellId(featureKey: string, dimensions: unknown): string | null {
  const explicitAtm = /^atm_iv_(30|60|90)d$/.exec(featureKey);
  if (explicitAtm) return `atm-${explicitAtm[1]}`;

  const dte = dimNumber(dimensions, 'target_dte');
  const optionType = dimString(dimensions, 'option_type');
  const surfaceCell = dimString(dimensions, 'surface_cell');
  const delta = dimNumber(dimensions, 'target_delta');
  if (dte === undefined) return null;

  if (
    (dte === 30 || dte === 60 || dte === 90) &&
    (surfaceCell === 'atm' || (optionType === undefined && delta === 0.5))
  ) {
    return `atm-${dte}`;
  }
  if (
    (optionType === 'put' || optionType === 'call') &&
    delta === 0.25 &&
    (dte === 30 || dte === 60)
  ) {
    return `${optionType}-${dte}-25d`;
  }
  return null;
}

export function groupObservationsBySurfaceCell(
  observations: MarketOpsFeatureObservationView[],
): Map<string, MarketOpsFeatureObservationView[]> {
  const groups = new Map<string, MarketOpsFeatureObservationView[]>();
  for (const observation of observations) {
    const cellId = observationSurfaceCellId(observation.featureKey, observation.dimensions);
    if (!cellId) continue;
    const rows = groups.get(cellId) ?? [];
    rows.push(observation);
    groups.set(cellId, rows);
  }
  return groups;
}

export interface HypothesisRequirementView {
  featureKey: string;
  dimensions: Record<string, unknown>;
}

export function parseHypothesisRequirements(value: unknown): HypothesisRequirementView[] {
  if (!Array.isArray(value)) return [];
  return value.flatMap((entry): HypothesisRequirementView[] => {
    if (typeof entry === 'string') return [{ featureKey: entry, dimensions: {} }];
    if (!isRecord(entry)) return [];
    const featureKey = asString(entry.feature_key);
    if (!featureKey) return [];
    return [{ featureKey, dimensions: isRecord(entry.dimensions) ? entry.dimensions : {} }];
  });
}

export function requirementMatches(
  requirement: HypothesisRequirementView,
  featureKey: string,
  dimensions: unknown,
): boolean {
  if (requirement.featureKey !== featureKey || !isRecord(dimensions)) {
    return requirement.featureKey === featureKey && Object.keys(requirement.dimensions).length === 0;
  }
  return Object.entries(requirement.dimensions).every(([key, expected]) => {
    const actual = dimensions[key];
    if (key === 'option_type' && expected === 'put_or_call') return actual === 'put' || actual === 'call';
    return actual === expected;
  });
}

export function qualityReason(details: unknown): string {
  if (!isRecord(details)) return '';
  return asString(details.reason) || asString(details.missing_reason) || asString(details.error);
}

// Select the nearest earlier persisted session for prior comparison. Among
// states with a strictly earlier session_date, pick the latest session_date;
// tie-break by newest as_of_time then deterministic id. Never compares across
// different schemas/dates implicitly.
export function selectPriorState(
  states: MarketOpsMarketStateView[],
  selected: MarketOpsMarketStateView,
): MarketOpsMarketStateView | null {
  const prior = states
    .filter((s) => s.sessionDate && s.sessionDate < selected.sessionDate)
    .sort(
      (a, b) =>
        b.sessionDate.localeCompare(a.sessionDate) ||
        b.asOfTime.localeCompare(a.asOfTime) ||
        b.deterministicKey.localeCompare(a.deterministicKey),
    );
  return prior[0] ?? null;
}

// Material-transition presentation filter. Prioritizes acceleration/regime
// transitions, non-zero multi-session changes, z-score/percentile rarity, and
// quality degradation/recovery. This is a presentation filter over persisted
// rows, not a claim that hidden rows do not exist.
export function isMaterialTransition(t: MarketOpsStateTransitionView): boolean {
  if (t.transitionType === 'acceleration' || t.transitionType === 'regime') return true;
  if (t.lookbackSessions !== null && t.lookbackSessions > 1 && t.transitionValue !== null && t.transitionValue !== 0) return true;
  if (t.zscore !== null && Math.abs(t.zscore) >= 2) return true;
  if (t.percentile !== null && (t.percentile >= 0.95 || t.percentile <= 0.05)) return true;
  if (t.qualityState === 'degraded' || t.qualityState === 'recovered') return true;
  return false;
}

export function partitionMaterialTransitions(transitions: MarketOpsStateTransitionView[]): {
  material: MarketOpsStateTransitionView[];
  all: MarketOpsStateTransitionView[];
} {
  return { material: transitions.filter(isMaterialTransition), all: transitions };
}

// --- G145 hypothesis calibration runtime parser ---------------------------------

export interface CalibrationMetricsView {
  evaluations: number;
  eligibleStates: number;
  triggers: number;
  triggerRate: number | null;
  independentSamples: number;
  maturedOutcomeSamples: number;
  directionalHitRate: number | null;
  meanForwardReturn: number | null;
  medianForwardReturn: number | null;
  meanFavorableExcursion: number | null;
  medianFavorableExcursion: number | null;
  meanAdverseExcursion: number | null;
  medianAdverseExcursion: number | null;
  drawdownIncidence: number | null;
  meanRealizedVolChange: number | null;
  calibrationError: number | null;
  confidenceBands: unknown;
  belowMinimumSampleSize: boolean;
}

export type CalibrationSegmentMap = Record<string, CalibrationMetricsView>;

export interface CalibrationVersionView {
  hypothesisVersion: string;
  overall: CalibrationMetricsView;
  byHorizon: CalibrationSegmentMap;
  byAsset: CalibrationSegmentMap;
  byYear: CalibrationSegmentMap;
  byVolatilityRegime: CalibrationSegmentMap;
  byEarningsWindow: CalibrationSegmentMap;
}

export interface HypothesisCalibrationReport {
  valid: boolean;
  incompatibleReason: string;
  summaryVersion: string;
  mode: string;
  hypothesisKey: string;
  hypothesisVersions: string[];
  symbols: string[];
  windowStart: string;
  windowEnd: string;
  asOf: string;
  minimumSampleSize: number;
  warnings: string[];
  promotionAllowed: boolean;
  selectedVersion: CalibrationVersionView | null;
  comparison: unknown;
  walkForward: unknown[];
}

const EMPTY_REPORT: HypothesisCalibrationReport = {
  valid: false,
  incompatibleReason: '',
  summaryVersion: '',
  mode: '',
  hypothesisKey: '',
  hypothesisVersions: [],
  symbols: [],
  windowStart: '',
  windowEnd: '',
  asOf: '',
  minimumSampleSize: 0,
  warnings: [],
  promotionAllowed: false,
  selectedVersion: null,
  comparison: null,
  walkForward: [],
};

function parseCalibrationMetrics(record: unknown): CalibrationMetricsView {
  if (!isRecord(record)) {
    return {
      evaluations: 0, eligibleStates: 0, triggers: 0, triggerRate: null, independentSamples: 0,
      maturedOutcomeSamples: 0, directionalHitRate: null, meanForwardReturn: null, medianForwardReturn: null,
      meanFavorableExcursion: null, medianFavorableExcursion: null, meanAdverseExcursion: null,
      medianAdverseExcursion: null, drawdownIncidence: null, meanRealizedVolChange: null,
      calibrationError: null, confidenceBands: {}, belowMinimumSampleSize: false,
    };
  }
  return {
    evaluations: asNumber(record.evaluations),
    eligibleStates: asNumber(record.eligible_states),
    triggers: asNumber(record.triggers),
    triggerRate: asNullableNumber(record.trigger_rate),
    independentSamples: asNumber(record.independent_samples),
    maturedOutcomeSamples: asNumber(record.matured_outcome_samples),
    directionalHitRate: asNullableNumber(record.directional_hit_rate),
    meanForwardReturn: asNullableNumber(record.mean_forward_return),
    medianForwardReturn: asNullableNumber(record.median_forward_return),
    meanFavorableExcursion: asNullableNumber(record.mean_favorable_excursion),
    medianFavorableExcursion: asNullableNumber(record.median_favorable_excursion),
    meanAdverseExcursion: asNullableNumber(record.mean_adverse_excursion),
    medianAdverseExcursion: asNullableNumber(record.median_adverse_excursion),
    drawdownIncidence: asNullableNumber(record.drawdown_incidence),
    meanRealizedVolChange: asNullableNumber(record.mean_realized_volatility_change),
    calibrationError: asNullableNumber(record.calibration_error),
    confidenceBands: record.confidence_bands ?? {},
    belowMinimumSampleSize: asBool(record.below_minimum_sample_size),
  };
}

function parseCalibrationSegmentMap(value: unknown): CalibrationSegmentMap {
  if (!isRecord(value)) return {};
  return Object.fromEntries(Object.entries(value).map(([key, metrics]) => [key, parseCalibrationMetrics(metrics)]));
}

// Runtime-validate a calibration summary `parameters` payload against the
// selected hypothesis key + exact version. Rejects wrong schema/key/version or
// missing version entries — never reinterprets generic counters as performance.
export function parseHypothesisCalibrationReport(
  parameters: unknown,
  selectedKey: string,
  selectedVersion: string,
): HypothesisCalibrationReport {
  const base: HypothesisCalibrationReport = {
    ...EMPTY_REPORT,
    hypothesisKey: selectedKey,
    hypothesisVersions: selectedVersion ? [selectedVersion] : [],
  };
  if (!isRecord(parameters)) {
    return { ...base, incompatibleReason: 'Calibration report is not compatible with marketops.hypothesis_calibration.v1' };
  }
  const summaryVersion = asString(parameters.summary_version);
  if (summaryVersion !== 'marketops.hypothesis_calibration.v1') {
    return { ...base, summaryVersion, incompatibleReason: 'Calibration report is not compatible with marketops.hypothesis_calibration.v1' };
  }
  const hypothesisKey = asString(parameters.hypothesis_key);
  if (selectedKey && hypothesisKey !== selectedKey) {
    return { ...base, summaryVersion, hypothesisKey, incompatibleReason: 'Calibration hypothesis_key does not match the selected definition' };
  }
  const hypothesisVersions = asStringArray(parameters.hypothesis_versions);
  if (selectedVersion && !hypothesisVersions.includes(selectedVersion)) {
    return {
      ...base, summaryVersion, hypothesisKey, hypothesisVersions,
      incompatibleReason: 'Selected hypothesis version is not present in the calibration report',
    };
  }
  const versions = parameters.versions;
  const versionEntry = isRecord(versions) ? versions[selectedVersion] : undefined;
  if (selectedVersion && (!isRecord(versionEntry) || !isRecord(versionEntry.overall))) {
    return {
      ...base, summaryVersion, hypothesisKey, hypothesisVersions,
      incompatibleReason: 'Calibration report is missing the selected version entry',
    };
  }
  const selectedView: CalibrationVersionView | null =
    selectedVersion && isRecord(versionEntry)
      ? {
          hypothesisVersion: asString(versionEntry.hypothesis_version) || selectedVersion,
          overall: parseCalibrationMetrics(versionEntry.overall),
          byHorizon: parseCalibrationSegmentMap(versionEntry.by_horizon),
          byAsset: parseCalibrationSegmentMap(versionEntry.by_asset),
          byYear: parseCalibrationSegmentMap(versionEntry.by_year),
          byVolatilityRegime: parseCalibrationSegmentMap(versionEntry.by_volatility_regime),
          byEarningsWindow: parseCalibrationSegmentMap(versionEntry.by_earnings_window),
        }
      : null;
  return {
    valid: true,
    incompatibleReason: '',
    summaryVersion,
    mode: asString(parameters.mode),
    hypothesisKey,
    hypothesisVersions,
    symbols: asStringArray(parameters.symbols),
    windowStart: asString(parameters.window_start),
    windowEnd: asString(parameters.window_end),
    asOf: asString(parameters.as_of),
    minimumSampleSize: asNumber(parameters.minimum_sample_size),
    warnings: asStringArray(parameters.warnings),
    promotionAllowed: asBool(parameters.promotion_allowed),
    selectedVersion: selectedView,
    comparison: parameters.comparison ?? null,
    walkForward: Array.isArray(parameters.walk_forward) ? parameters.walk_forward : [],
  };
}

// Restrained quality-state tones. Color supplements text only.
const QUALITY_STATE_STYLES: Record<string, string> = {
  usable: 'border-emerald-200 bg-emerald-50 text-emerald-700',
  degraded: 'border-amber-200 bg-amber-50 text-amber-700',
  recovered: 'border-emerald-200 bg-emerald-50 text-emerald-700',
  unusable: 'border-red-200 bg-red-50 text-red-700',
  missing: 'border-gray-200 bg-gray-100 text-gray-500',
};

export function qualityStateStyle(state: string): string {
  return QUALITY_STATE_STYLES[state] ?? 'border-gray-200 bg-gray-50 text-gray-600';
}

// Disposition tones. Append-only analyst judgment — distinct from lifecycle.
const DISPOSITION_STYLES: Record<string, string> = {
  watch: 'border-blue-200 bg-blue-50 text-blue-700',
  advance: 'border-emerald-200 bg-emerald-50 text-emerald-700',
  needs_more_evidence: 'border-amber-200 bg-amber-50 text-amber-700',
  dismiss: 'border-gray-200 bg-gray-100 text-gray-500',
  resolved: 'border-emerald-200 bg-emerald-50 text-emerald-700',
};

export function dispositionStyle(disposition: string): string {
  return DISPOSITION_STYLES[disposition] ?? 'border-gray-200 bg-gray-50 text-gray-600';
}

export function formatNullableNumber(value: number | null | undefined, digits = 2): string {
  if (value === null || value === undefined) return '—';
  return value.toFixed(digits);
}

export function formatNullablePercent(value: number | null | undefined, digits = 1): string {
  if (value === null || value === undefined) return '—';
  return `${(value * 100).toFixed(digits)}%`;
}
