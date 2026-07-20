import { describe, expect, it } from 'vitest';
import {
  summarizeMarketOpsIntelligenceReadinessSymbol,
  summarizeMarketOpsIntelligenceReadinessAggregate,
  rolloutStatusStyle,
  dimensionStateStyle,
  formatCoverageRatio,
} from './marketopsReadiness';

describe('summarizeMarketOpsIntelligenceReadinessSymbol (G148-C)', () => {
  it('reads fields and marks a symbol observed when latest_market_state_id is present', () => {
    const s = summarizeMarketOpsIntelligenceReadinessSymbol({
      result_id: 'r1',
      symbol: 'AAPL',
      universe_group: 'top50_megacap',
      latest_market_state_id: 'ms-1',
      latest_state_date: '2026-07-18T00:00:00Z',
      latest_state_completeness: 0.2,
      required_feature_coverage: 0.4,
      rollout_status: 'blocked',
      calibration_below_minimum: true,
      coverage_state: 'incomplete',
      readiness_reasons: ['state quality blocks evaluation'],
    });
    expect(s.symbol).toBe('AAPL');
    expect(s.observed).toBe(true);
    expect(s.rolloutStatus).toBe('blocked');
    expect(s.calibrationBelowMinimum).toBe(true);
    expect(s.latestStateCompleteness).toBeCloseTo(0.2);
    expect(s.readinessReasons).toEqual(['state quality blocks evaluation']);
  });

  it('marks a symbol not observed when latest_market_state_id is empty', () => {
    const s = summarizeMarketOpsIntelligenceReadinessSymbol({
      result_id: 'r2',
      symbol: 'MSFT',
      latest_market_state_id: '',
      rollout_status: 'not_observed',
    });
    expect(s.observed).toBe(false);
    expect(s.calibrationBelowMinimum).toBe(false);
  });

  it('collapses non-object payloads to the empty view', () => {
    const s = summarizeMarketOpsIntelligenceReadinessSymbol(null);
    expect(s.symbol).toBe('');
    expect(s.observed).toBe(false);
    expect(s.readinessReasons).toEqual([]);
  });
});

describe('summarizeMarketOpsIntelligenceReadinessAggregate (G148-C)', () => {
  it('orders dimension counts by count desc and reads production_ready_supported', () => {
    const a = summarizeMarketOpsIntelligenceReadinessAggregate({
      symbol_count: 2,
      production_ready_supported: false,
      latest_session_date: '2026-07-18',
      dimension_counts: {
        rollout_status: { not_observed: 1, blocked: 1 },
        coverage_state: { incomplete: 1, unavailable: 1 },
      },
    });
    expect(a.symbolCount).toBe(2);
    expect(a.productionReadySupported).toBe(false);
    expect(a.latestSessionDate).toBe('2026-07-18');
    // Ties broken by key asc.
    expect(a.rolloutStatus.map((e) => e.key)).toEqual(['blocked', 'not_observed']);
    expect(a.coverageState.map((e) => e.key)).toEqual(['incomplete', 'unavailable']);
  });

  it('collapses a non-object aggregate to empty values', () => {
    const a = summarizeMarketOpsIntelligenceReadinessAggregate(null);
    expect(a.symbolCount).toBe(0);
    expect(a.rolloutStatus).toEqual([]);
  });
});

describe('formatCoverageRatio (G148-C)', () => {
  it('renders "Not observed" for unobserved symbols, never zero', () => {
    expect(formatCoverageRatio(0, false)).toBe('Not observed');
    expect(formatCoverageRatio(0.2, true)).toBe('20%');
    expect(formatCoverageRatio(1, true)).toBe('100%');
  });
});

describe('rollout + dimension styles (G148-C)', () => {
  it('tones rollout statuses with text-paired color', () => {
    expect(rolloutStatusStyle('blocked')).toContain('red');
    expect(rolloutStatusStyle('review_ready')).toContain('emerald');
    expect(rolloutStatusStyle('research_evaluation_ready')).toContain('amber');
    expect(rolloutStatusStyle('inspection_ready')).toContain('blue');
    expect(rolloutStatusStyle('not_observed')).toContain('gray');
    expect(rolloutStatusStyle('future')).toContain('gray-600');
  });

  it('tones dimension states and warns on below_minimum / partial', () => {
    expect(dimensionStateStyle('available')).toContain('emerald');
    expect(dimensionStateStyle('below_minimum')).toContain('amber');
    expect(dimensionStateStyle('incomplete')).toContain('amber');
    expect(dimensionStateStyle('blocked')).toContain('red');
    expect(dimensionStateStyle('research_only')).toContain('blue');
    expect(dimensionStateStyle('unavailable')).toContain('gray');
    expect(dimensionStateStyle('future')).toContain('gray');
  });
});
