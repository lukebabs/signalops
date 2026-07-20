import { describe, expect, it } from 'vitest';
import {
  summarizeMarketOpsState,
  summarizeMarketOpsFeatureObservation,
  summarizeMarketOpsStateTransition,
  summarizeMarketOpsOutcome,
  summarizeMarketOpsOpportunityDisposition,
  summarizeMarketOpsHypothesisEvaluation,
  observationSurfaceCellId,
  groupObservationsBySurfaceCell,
  parseHypothesisRequirements,
  requirementMatches,
  qualityReason,
  selectPriorState,
  isMaterialTransition,
  partitionMaterialTransitions,
  parseHypothesisCalibrationReport,
  qualityStateStyle,
  dispositionStyle,
  formatNullableNumber,
  formatNullablePercent,
} from './marketopsState';

describe('summarizers (G147)', () => {
  it('state keeps nullable qualityScore and reads counts', () => {
    const s = summarizeMarketOpsState({ market_state_id: 'ms-1', symbol: 'AAPL', feature_count: 9, required_feature_count: 10, completeness_ratio: 0.9, quality_state: 'usable', quality_score: 0.96 });
    expect(s.marketStateId).toBe('ms-1');
    expect(s.qualityScore).toBeCloseTo(0.96);
    expect(s.completenessRatio).toBeCloseTo(0.9);
    const s2 = summarizeMarketOpsState({ market_state_id: 'ms-2', quality_state: 'degraded' });
    expect(s2.qualityScore).toBeNull();
  });

  it('feature observation keeps nullable value slots distinct', () => {
    const o = summarizeMarketOpsFeatureObservation({ feature_observation_id: 'o-1', feature_key: 'iv', dimensions: { target_dte: 30 }, numeric_value: 0.45, quality_state: 'usable' });
    expect(o.numericValue).toBeCloseTo(0.45);
    expect(o.textValue).toBeNull();
    expect(o.booleanValue).toBeNull();
    const o2 = summarizeMarketOpsFeatureObservation({ feature_observation_id: 'o-2', feature_key: 'flag', boolean_value: false, quality_state: 'usable' });
    expect(o2.booleanValue).toBe(false);
    expect(o2.numericValue).toBeNull();
  });

  it('transition keeps nullable metrics', () => {
    const t = summarizeMarketOpsStateTransition({ transition_id: 't-1', transition_type: 'zscore', zscore: 2.4, lookback_sessions: 20 });
    expect(t.zscore).toBeCloseTo(2.4);
    expect(t.lookbackSessions).toBe(20);
    expect(t.percentile).toBeNull();
  });

  it('outcome keeps nullable returns and status', () => {
    const o = summarizeMarketOpsOutcome({ outcome_id: 'o-1', outcome_status: 'pending', horizon_sessions: 5 });
    expect(o.outcomeStatus).toBe('pending');
    expect(o.forwardReturn).toBeNull();
    expect(o.horizonSessions).toBe(5);
  });

  it('disposition reads fields', () => {
    const d = summarizeMarketOpsOpportunityDisposition({ disposition_id: 'd-1', disposition: 'watch', actor: 'analyst-1', note: 'n' });
    expect(d.disposition).toBe('watch');
    expect(d.actor).toBe('analyst-1');
  });

  it('hypothesis evaluation surfaces all nullable scores', () => {
    const e = summarizeMarketOpsHypothesisEvaluation({ evaluation_id: 'e-1', hypothesis_key: 'H1', eligible: true, triggered: false, trigger_score: 0.3, rarity_score: 0.9, evaluation_payload: { threshold: 2 }, evaluation_run_id: 'run-1', deterministic_key: 'eval-key', created_at: '2026-07-20T01:00:00Z' });
    expect(e.eligible).toBe(true);
    expect(e.triggerScore).toBeCloseTo(0.3);
    expect(e.rarityScore).toBeCloseTo(0.9);
    expect(e.confidenceScore).toBeNull();
    expect(e.evaluationPayload).toEqual({ threshold: 2 });
    expect(e.evaluationRunId).toBe('run-1');
    expect(e.deterministicKey).toBe('eval-key');
  });
});

describe('observationSurfaceCellId (G147)', () => {
  it('maps ATM and 25-delta wing cells by dimensions', () => {
    expect(observationSurfaceCellId('atm_iv_30d', {})).toBe('atm-30');
    expect(observationSurfaceCellId('atm_iv_90d', {})).toBe('atm-90');
    expect(observationSurfaceCellId('iv', { option_type: 'put', target_dte: 30, target_delta: 0.25 })).toBe('put-30-25d');
    expect(observationSurfaceCellId('iv', { option_type: 'call', target_dte: 60, target_delta: 0.25 })).toBe('call-60-25d');
    expect(observationSurfaceCellId('iv_change_1d', { surface_cell: 'atm', target_dte: 60, target_delta: 0.5 })).toBe('atm-60');
  });

  it('returns null for non-surface dimensions', () => {
    expect(observationSurfaceCellId('iv', {})).toBeNull();
    expect(observationSurfaceCellId('iv', { target_dte: 45 })).toBeNull();
    expect(observationSurfaceCellId('iv', { option_type: 'put', target_dte: 30, target_delta: 25 })).toBeNull();
  });

  it('groups observations by canonical cell', () => {
    const obs = [
      summarizeMarketOpsFeatureObservation({ feature_observation_id: 'a', feature_key: 'atm_iv_30d', dimensions: {}, numeric_value: 1 }),
      summarizeMarketOpsFeatureObservation({ feature_observation_id: 'b', feature_key: 'iv', dimensions: { option_type: 'put', target_dte: 30, target_delta: 0.25 }, numeric_value: 2 }),
      summarizeMarketOpsFeatureObservation({ feature_observation_id: 'c', feature_key: 'iv', dimensions: {}, numeric_value: 3 }),
    ];
    const groups = groupObservationsBySurfaceCell(obs);
    expect(groups.get('atm-30')).toHaveLength(1);
    expect(groups.get('put-30-25d')).toHaveLength(1);
    expect(groups.has('atm-60')).toBe(false);
  });
});

describe('hypothesis requirements (G147)', () => {
  it('parses string and dimension-aware requirements', () => {
    const requirements = parseHypothesisRequirements([
      'atm_iv_30d',
      { feature_key: 'iv', dimensions: { option_type: 'put_or_call', target_dte: 30, target_delta: 0.25 } },
      { ignored: true },
    ]);
    expect(requirements).toHaveLength(2);
    expect(requirementMatches(requirements[0], 'atm_iv_30d', {})).toBe(true);
    expect(requirementMatches(requirements[1], 'iv', { option_type: 'put', target_dte: 30, target_delta: 0.25 })).toBe(true);
    expect(requirementMatches(requirements[1], 'iv', { option_type: 'call', target_dte: 30, target_delta: 0.25 })).toBe(true);
    expect(requirementMatches(requirements[1], 'iv', { option_type: 'put', target_dte: 30, target_delta: 25 })).toBe(false);
  });

  it('surfaces persisted quality reasons only', () => {
    expect(qualityReason({ missing_reason: 'stale input' })).toBe('stale input');
    expect(qualityReason({})).toBe('');
  });
});

describe('selectPriorState (G147)', () => {
  const mk = (id: string, session: string, asOf = '2026-01-01T00:00:00Z') =>
    summarizeMarketOpsState({ market_state_id: id, session_date: session, as_of_time: asOf, deterministic_key: id });

  it('chooses the nearest earlier persisted session', () => {
    const states = [mk('a', '2026-07-10'), mk('b', '2026-07-15'), mk('c', '2026-07-20')];
    expect(selectPriorState(states, mk('sel', '2026-07-20'))?.marketStateId).toBe('b');
  });

  it('returns null when there is no earlier session', () => {
    const states = [mk('a', '2026-07-20')];
    expect(selectPriorState(states, mk('sel', '2026-07-20'))).toBeNull();
  });
});

describe('material transition filter (G147)', () => {
  const mkT = (over: Record<string, unknown> = {}) =>
    summarizeMarketOpsStateTransition({ transition_id: 't', transition_type: 'level', ...over });

  it('flags acceleration, rare z-score, rare percentile, and multi-session non-zero change', () => {
    expect(isMaterialTransition(mkT({ transition_type: 'acceleration' }))).toBe(true);
    expect(isMaterialTransition(mkT({ zscore: 2.5 }))).toBe(true);
    expect(isMaterialTransition(mkT({ percentile: 0.97 }))).toBe(true);
    expect(isMaterialTransition(mkT({ lookback_sessions: 5, transition_value: 0.4 }))).toBe(true);
    expect(isMaterialTransition(mkT({ lookback_sessions: 1, transition_value: 0 }))).toBe(false);
  });

  it('partitions material vs all', () => {
    const all = [mkT({ transition_id: 'a', transition_type: 'acceleration' }), mkT({ transition_id: 'b', transition_value: 0 })];
    const { material } = partitionMaterialTransitions(all);
    expect(material.map((t) => t.transitionId)).toEqual(['a']);
  });
});

describe('parseHypothesisCalibrationReport (G147)', () => {
  const validParams = (key: string, version: string) => ({
    summary_version: 'marketops.hypothesis_calibration.v1',
    mode: 'single',
    hypothesis_key: key,
    hypothesis_versions: [version],
    symbols: ['AAPL'],
    window_start: '2024-01-01',
    window_end: '2026-07-01',
    as_of: '2026-07-01',
    minimum_sample_size: 100,
    warnings: ['short window'],
    promotion_allowed: false,
    versions: {
      [version]: {
        hypothesis_version: version,
        overall: {
          evaluations: 250, eligible_states: 240, triggers: 60, trigger_rate: 0.25, independent_samples: 120,
          matured_outcome_samples: 110, directional_hit_rate: 0.55, mean_forward_return: 0.01, below_minimum_sample_size: false,
        },
        by_horizon: {
          '5': {
            evaluations: 60, eligible_states: 55, triggers: 12, independent_samples: 30,
            matured_outcome_samples: 25, directional_hit_rate: null, mean_forward_return: null,
          },
        },
      },
    },
  });

  it('accepts a matching schema/key/version and surfaces metrics', () => {
    const r = parseHypothesisCalibrationReport(validParams('H001', 'v1'), 'H001', 'v1');
    expect(r.valid).toBe(true);
    expect(r.promotionAllowed).toBe(false);
    expect(r.selectedVersion?.overall.independentSamples).toBe(120);
    expect(r.selectedVersion?.overall.directionalHitRate).toBeCloseTo(0.55);
    expect(r.selectedVersion?.byHorizon['5'].maturedOutcomeSamples).toBe(25);
    expect(r.selectedVersion?.byHorizon['5'].directionalHitRate).toBeNull();
    expect(r.warnings).toEqual(['short window']);
  });

  it('rejects wrong summary_version', () => {
    const r = parseHypothesisCalibrationReport({ ...validParams('H001', 'v1'), summary_version: 'other' }, 'H001', 'v1');
    expect(r.valid).toBe(false);
    expect(r.incompatibleReason).toContain('not compatible');
  });

  it('rejects hypothesis_key mismatch', () => {
    const r = parseHypothesisCalibrationReport(validParams('H001', 'v1'), 'H002', 'v1');
    expect(r.valid).toBe(false);
    expect(r.incompatibleReason).toContain('hypothesis_key');
  });

  it('rejects when the selected version is not in hypothesis_versions', () => {
    const r = parseHypothesisCalibrationReport(validParams('H001', 'v1'), 'H001', 'v2');
    expect(r.valid).toBe(false);
    expect(r.incompatibleReason).toContain('version');
  });

  it('rejects when the versions entry is missing for the selected version', () => {
    const params = validParams('H001', 'v1');
    params.versions = {};
    const r = parseHypothesisCalibrationReport(params, 'H001', 'v1');
    expect(r.valid).toBe(false);
    expect(r.incompatibleReason).toContain('version entry');
  });

  it('flags below-minimum-sample without claiming performance', () => {
    const params = validParams('H001', 'v1');
    (params.versions as Record<string, { overall: { below_minimum_sample_size: boolean } }>).v1.overall.below_minimum_sample_size = true;
    const r = parseHypothesisCalibrationReport(params, 'H001', 'v1');
    expect(r.valid).toBe(true);
    expect(r.selectedVersion?.overall.belowMinimumSampleSize).toBe(true);
  });

  it('handles non-object payloads', () => {
    expect(parseHypothesisCalibrationReport(null, 'H001', 'v1').valid).toBe(false);
    expect(parseHypothesisCalibrationReport('nope', 'H001', 'v1').valid).toBe(false);
  });
});

describe('styles + formatting (G147)', () => {
  it('tones quality + disposition states', () => {
    expect(qualityStateStyle('usable')).toContain('emerald');
    expect(qualityStateStyle('unusable')).toContain('red');
    expect(qualityStateStyle('missing')).toContain('gray');
    expect(dispositionStyle('advance')).toContain('emerald');
    expect(dispositionStyle('dismiss')).toContain('gray');
    expect(dispositionStyle('future')).toContain('gray-600');
  });

  it('formats nullable numbers and em-dashes nulls', () => {
    expect(formatNullableNumber(0.5, 3)).toBe('0.500');
    expect(formatNullableNumber(null)).toBe('—');
    expect(formatNullablePercent(0.125)).toBe('12.5%');
    expect(formatNullablePercent(null)).toBe('—');
  });
});
