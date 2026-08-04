import { describe, expect, it } from 'vitest';
import { matchesAllMarketOpsAssetQuickFilters, matchesMarketOpsAssetQuickFilter, toggleMarketOpsAssetQuickFilter, type MarketOpsAssetQuickFilterInput } from './marketopsAssetQuickFilters';

const asset = {
  ticker: 'AMAT',
  quote: { stale: false, market_status: 'end_of_day', change_percent: 3.5 },
  intraday: { stale: false, conditions: [{ key: 'momentum', score: 0.8 }] },
  riskReward: { direction: 'bullish', risk_level: 'high', confidence: 0.625 },
} as unknown as MarketOpsAssetQuickFilterInput;

describe('MarketOps asset quick filters', () => {
  it('matches analyst triage predicates without treating degraded Risk/Reward as eligible', () => {
    expect(matchesMarketOpsAssetQuickFilter(asset, 'large_gainers')).toBe(true);
    expect(matchesMarketOpsAssetQuickFilter({ ...asset, quote: { stale: true, market_status: 'end_of_day', change_percent: 3.5 } } as MarketOpsAssetQuickFilterInput, 'large_gainers')).toBe(true);
    expect(matchesMarketOpsAssetQuickFilter({ ...asset, quote: { stale: true, market_status: 'regular', change_percent: 3.5 } } as MarketOpsAssetQuickFilterInput, 'large_gainers')).toBe(false);
    expect(matchesMarketOpsAssetQuickFilter(asset, 'active_intraday_conditions')).toBe(true);
    expect(matchesMarketOpsAssetQuickFilter(asset, 'bullish_risk_reward')).toBe(true);
    expect(matchesMarketOpsAssetQuickFilter({ ...asset, riskReward: { ...asset.riskReward!, confidence: 0.25 } }, 'bullish_risk_reward')).toBe(false);
  });

  it('intersects filters and replaces opposing mover/posture choices', () => {
    expect(matchesAllMarketOpsAssetQuickFilters(asset, ['large_gainers', 'high_atr_risk'])).toBe(true);
    expect(toggleMarketOpsAssetQuickFilter(['large_gainers', 'bullish_risk_reward'], 'large_decliners')).toEqual(['bullish_risk_reward', 'large_decliners']);
    expect(toggleMarketOpsAssetQuickFilter(['large_gainers', 'bullish_risk_reward'], 'bearish_risk_reward')).toEqual(['large_gainers', 'bearish_risk_reward']);
  });
});
