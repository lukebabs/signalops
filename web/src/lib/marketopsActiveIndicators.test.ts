import { describe, expect, it } from 'vitest';
import { activeMarketOpsIndicators } from './marketopsActiveIndicators';
import { summarizeMarketOpsFeatureObservation, summarizeMarketOpsStateTransition } from './marketopsState';

const observation = (feature_key: string, numeric_value: number, quality_state = 'usable') => summarizeMarketOpsFeatureObservation({ feature_key, numeric_value, quality_state });
const textObservation = (feature_key: string, text_value: string, quality_state = 'usable') => summarizeMarketOpsFeatureObservation({ feature_key, text_value, quality_state });

describe('activeMarketOpsIndicators', () => {
  it('shows only active, asset-backed underlying and options conditions', () => {
    const result = activeMarketOpsIndicators([
      observation('rsi_14', 72.4), observation('volume_ratio_20d', 1.7), observation('gap_pct', -1.3),
      observation('iv_rv_ratio_20d', 1.4), observation('put_call_oi_ratio', 1.3), textObservation('term_structure_state', 'backwardation'),
    ], []);
    expect(result.map((item) => item.key)).toEqual(['overbought', 'unusual_volume', 'gap_down', 'iv_premium', 'iv_backwardation', 'put_oi_skew']);
    expect(result.find((item) => item.key === 'overbought')?.detail).toBe('RSI 72.4 ≥ 70');
  });

  it('supports oversold and bullish call-skew states', () => {
    const result = activeMarketOpsIndicators([observation('rsi_14', 28), observation('distance_sma_20_pct', -6), observation('put_call_volume_ratio', .7)], []);
    expect(result.map((item) => item.key)).toEqual(['oversold', 'extended_below_sma', 'call_volume_skew']);
  });

  it('does not create indicators from missing or unusable values', () => {
    const result = activeMarketOpsIndicators([observation('rsi_14', 80, 'missing'), observation('volume_ratio_20d', 0), observation('gap_pct', 0)], []);
    expect(result).toEqual([]);
  });

  it('shows statistically unusual OI only when persisted transition statistics qualify', () => {
    const result = activeMarketOpsIndicators([], [
      summarizeMarketOpsStateTransition({ transition_id: 'rare', feature_key: 'oi_change_1d', transition_value: 250, zscore: 2.4, quality_state: 'usable' }),
      summarizeMarketOpsStateTransition({ transition_id: 'ordinary', feature_key: 'oi_change_1d', transition_value: 250, zscore: 1.9, quality_state: 'usable' }),
    ]);
    expect(result).toEqual([{ key: 'unusual_oi:rare', title: 'Unusual OI increase', tone: 'neutral', detail: '250 contracts; 2.4σ' }]);
  });
});
