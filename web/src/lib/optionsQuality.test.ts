import { describe, expect, it } from 'vitest';
import {
  getOptionsQualityFields,
  normalizeRatioQuality,
  formatZeroRate,
  isOptionsRatioAlgorithmPayload,
  isOptionsRatioQualityUsable,
  summarizeAlgorithmResultQuality,
  summarizeProposalQualityGate,
  optionsRatioQualityStyle,
} from './optionsQuality';

describe('getOptionsQualityFields (G132)', () => {
  it('extracts the six quality fields when present', () => {
    const f = getOptionsQualityFields({
      open_interest_quality: 'partial_zero',
      open_interest_zero_count: 21,
      open_interest_positive_count: 12,
      open_interest_zero_rate: 0.636364,
      call_put_oi_denominator_is_zero: false,
      call_put_oi_ratio_quality: 'usable',
      unrelated_key: 'ignored',
    });
    expect(f).toEqual({
      openInterestQuality: 'partial_zero',
      openInterestZeroCount: 21,
      openInterestPositiveCount: 12,
      openInterestZeroRate: 0.636364,
      callPutOiDenominatorIsZero: false,
      callPutOiRatioQuality: 'usable',
    });
  });

  it('returns an empty object for non-records and omits absent fields', () => {
    expect(getOptionsQualityFields(null)).toEqual({});
    expect(getOptionsQualityFields('nope')).toEqual({});
    expect(getOptionsQualityFields({ open_interest_quality: 'usable' })).toEqual({ openInterestQuality: 'usable' });
  });
});

describe('normalizeRatioQuality (G132)', () => {
  it('passes known tokens through and maps unknown/absent to unknown', () => {
    expect(normalizeRatioQuality('usable')).toBe('usable');
    expect(normalizeRatioQuality('partial_zero')).toBe('partial_zero');
    expect(normalizeRatioQuality('all_zero')).toBe('all_zero');
    expect(normalizeRatioQuality('denominator_zero')).toBe('denominator_zero');
    expect(normalizeRatioQuality('empty')).toBe('empty');
    expect(normalizeRatioQuality('missing')).toBe('missing');
    expect(normalizeRatioQuality('garbage')).toBe('unknown');
    expect(normalizeRatioQuality(undefined)).toBe('unknown');
    expect(normalizeRatioQuality(42)).toBe('unknown');
  });
});

describe('formatZeroRate (G132)', () => {
  it('formats a 0..1 rate as a percentage and em-dashes when absent', () => {
    expect(formatZeroRate(0.636364)).toBe('63.6%');
    expect(formatZeroRate(0)).toBe('0.0%');
    expect(formatZeroRate(undefined)).toBe('—');
    expect(formatZeroRate('nope')).toBe('—');
  });
});

describe('isOptionsRatioAlgorithmPayload (G132)', () => {
  it('is true only for options_distribution_daily call/put OI ratio payloads', () => {
    expect(isOptionsRatioAlgorithmPayload({ dataset: 'options_distribution_daily', feature: 'call_put_open_interest_ratio' })).toBe(true);
    expect(isOptionsRatioAlgorithmPayload({ dataset: 'options_distribution_daily', feature: 'other' })).toBe(false);
    expect(isOptionsRatioAlgorithmPayload({ dataset: 'other', feature: 'call_put_open_interest_ratio' })).toBe(false);
    expect(isOptionsRatioAlgorithmPayload(null)).toBe(false);
    expect(isOptionsRatioAlgorithmPayload({})).toBe(false);
  });
});

describe('isOptionsRatioQualityUsable (G132)', () => {
  it('is true only for usable', () => {
    expect(isOptionsRatioQualityUsable('usable')).toBe(true);
    expect(isOptionsRatioQualityUsable('partial_zero')).toBe(false);
    expect(isOptionsRatioQualityUsable('unknown')).toBe(false);
  });
});

describe('summarizeAlgorithmResultQuality (G132)', () => {
  it('extracts quality from an options-ratio result_payload', () => {
    const q = summarizeAlgorithmResultQuality({
      dataset: 'options_distribution_daily',
      feature: 'call_put_open_interest_ratio',
      call_put_oi_ratio_quality: 'denominator_zero',
      open_interest_quality: 'partial_zero',
      open_interest_zero_rate: 0.5,
    });
    expect(q.isOptionsRatio).toBe(true);
    expect(q.ratioQuality).toBe('denominator_zero');
    expect(q.quality.openInterestQuality).toBe('partial_zero');
  });

  it('returns isOptionsRatio=false for non-options payloads', () => {
    const q = summarizeAlgorithmResultQuality({ dataset: 'other', feature: 'x' });
    expect(q.isOptionsRatio).toBe(false);
    expect(q.ratioQuality).toBe('unknown');
    expect(q.quality).toEqual({});
  });
});

describe('summarizeProposalQualityGate (G132)', () => {
  it('parses quality_gate and nested algorithm-result ratio quality', () => {
    const g = summarizeProposalQualityGate({
      schema_version: 'algorithm_signal_proposal.v1',
      quality_gate: { passed: true, policy: 'g131.options_distribution_quality.v1' },
      algorithm_result: {
        algorithm_result_id: 'algres_1',
        payload: {
          dataset: 'options_distribution_daily',
          feature: 'call_put_open_interest_ratio',
          call_put_oi_ratio_quality: 'usable',
          open_interest_quality: 'usable',
          open_interest_zero_rate: 0,
        },
      },
    });
    expect(g.present).toBe(true);
    expect(g.passed).toBe(true);
    expect(g.policy).toBe('g131.options_distribution_quality.v1');
    expect(g.isOptionsRatio).toBe(true);
    expect(g.ratioQuality).toBe('usable');
    expect(g.openInterestQuality).toBe('usable');
    expect(g.quality.openInterestZeroRate).toBe(0);
  });

  it('handles a proposal without a quality_gate (older proposals)', () => {
    const g = summarizeProposalQualityGate({ schema_version: 'algorithm_signal_proposal.v1' });
    expect(g.present).toBe(false);
    expect(g.passed).toBeNull();
    expect(g.isOptionsRatio).toBe(false);
  });

  it('reports not-passed and a non-usable ratio when the gate blocked the result', () => {
    const g = summarizeProposalQualityGate({
      quality_gate: { passed: false, policy: 'g131.options_distribution_quality.v1' },
      algorithm_result: { payload: { dataset: 'options_distribution_daily', feature: 'call_put_open_interest_ratio', call_put_oi_ratio_quality: 'denominator_zero' } },
    });
    expect(g.present).toBe(true);
    expect(g.passed).toBe(false);
    expect(g.ratioQuality).toBe('denominator_zero');
  });

  it('tolerates a non-object proposal payload', () => {
    expect(summarizeProposalQualityGate(null).present).toBe(false);
    expect(summarizeProposalQualityGate('nope').present).toBe(false);
  });
});

describe('optionsRatioQualityStyle (G132)', () => {
  it('tones usable/partial/blocked/unknown correctly', () => {
    expect(optionsRatioQualityStyle('usable')).toContain('emerald');
    expect(optionsRatioQualityStyle('partial_zero')).toContain('amber');
    expect(optionsRatioQualityStyle('all_zero')).toContain('red');
    expect(optionsRatioQualityStyle('denominator_zero')).toContain('red');
    expect(optionsRatioQualityStyle('empty')).toContain('red');
    expect(optionsRatioQualityStyle('missing')).toContain('red');
    expect(optionsRatioQualityStyle('unknown')).toContain('gray');
    expect(optionsRatioQualityStyle('future')).toContain('gray');
  });
});
