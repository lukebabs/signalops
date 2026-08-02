import type { MarketOpsAssetQuote, MarketOpsIntradayConditionSnapshot, MarketOpsRiskRewardSummary } from '../types';

export type MarketOpsAssetQuickFilter = 'large_gainers' | 'large_decliners' | 'active_intraday_conditions' | 'bullish_risk_reward' | 'bearish_risk_reward' | 'high_atr_risk' | 'call_volume_extremes' | 'put_volume_extremes';

type QuickFilterGroup = 'mover' | 'intraday' | 'posture' | 'risk' | 'options_flow';

export const MARKETOPS_ASSET_QUICK_FILTERS: Array<{ key: MarketOpsAssetQuickFilter; group: QuickFilterGroup; label: string; hint: string }> = [
  { key: 'large_gainers', group: 'mover', label: 'Large gainers', hint: 'Latest session ≥ +2.0%' },
  { key: 'large_decliners', group: 'mover', label: 'Large decliners', hint: 'Latest session ≤ −2.0%' },
  { key: 'active_intraday_conditions', group: 'intraday', label: 'Active intraday conditions', hint: 'Current 15-minute monitor' },
  { key: 'bullish_risk_reward', group: 'posture', label: 'Bullish Risk/Reward', hint: 'Eligible post-close posture' },
  { key: 'bearish_risk_reward', group: 'posture', label: 'Bearish Risk/Reward', hint: 'Eligible post-close posture' },
  { key: 'high_atr_risk', group: 'risk', label: 'High ATR risk', hint: 'Eligible elevated volatility' },
  { key: 'call_volume_extremes', group: 'options_flow', label: 'Call-volume extremes', hint: 'Put/call volume < 0.30 · ≥1k contracts' },
  { key: 'put_volume_extremes', group: 'options_flow', label: 'Put-volume extremes', hint: 'Put/call volume > 1.20 · ≥1k contracts' },
];

export type MarketOpsAssetQuickFilterInput = {
  ticker: string;
  quote?: MarketOpsAssetQuote;
  intraday?: MarketOpsIntradayConditionSnapshot;
  riskReward?: MarketOpsRiskRewardSummary;
  optionsFlowExtreme?: "call_volume_extreme" | "put_volume_extreme";
};

const eligibleRiskReward = (riskReward?: MarketOpsRiskRewardSummary) => (riskReward?.confidence ?? 0) >= 0.625;

// These filters intentionally use the last persisted return. Cache freshness
// tells the UI whether a live update is expected; it does not invalidate the
// latest completed-session observation used for analyst triage.
const usableMarketMove = (quote?: MarketOpsAssetQuote) => Boolean(
  quote && Number.isFinite(quote.change_percent),
);

export function matchesMarketOpsAssetQuickFilter(asset: MarketOpsAssetQuickFilterInput, filter: MarketOpsAssetQuickFilter): boolean {
  switch (filter) {
    case 'large_gainers': return usableMarketMove(asset.quote) && (asset.quote?.change_percent ?? Number.NEGATIVE_INFINITY) >= 2;
    case 'large_decliners': return usableMarketMove(asset.quote) && (asset.quote?.change_percent ?? Number.POSITIVE_INFINITY) <= -2;
    case 'active_intraday_conditions': return Boolean(asset.intraday && !asset.intraday.stale && asset.intraday.conditions.length);
    case 'bullish_risk_reward': return eligibleRiskReward(asset.riskReward) && asset.riskReward?.direction === 'bullish';
    case 'bearish_risk_reward': return eligibleRiskReward(asset.riskReward) && asset.riskReward?.direction === 'bearish';
    case 'high_atr_risk': return eligibleRiskReward(asset.riskReward) && asset.riskReward?.risk_level === 'high';
    case 'call_volume_extremes': return asset.optionsFlowExtreme === 'call_volume_extreme';
    case 'put_volume_extremes': return asset.optionsFlowExtreme === 'put_volume_extreme';
  }
}

export function matchesAllMarketOpsAssetQuickFilters(asset: MarketOpsAssetQuickFilterInput, filters: MarketOpsAssetQuickFilter[]): boolean {
  return filters.every((filter) => matchesMarketOpsAssetQuickFilter(asset, filter));
}

export function toggleMarketOpsAssetQuickFilter(filters: MarketOpsAssetQuickFilter[], filter: MarketOpsAssetQuickFilter): MarketOpsAssetQuickFilter[] {
  if (filters.includes(filter)) return filters.filter((item) => item !== filter);
  const group = MARKETOPS_ASSET_QUICK_FILTERS.find((item) => item.key === filter)?.group;
  return [...filters.filter((item) => MARKETOPS_ASSET_QUICK_FILTERS.find((candidate) => candidate.key === item)?.group !== group), filter];
}
