// Read-only, asset-specific indicators for the Market State view.  These are
// derived exclusively from the selected state's persisted observations and
// transitions; absent or non-usable inputs never become a zero or a signal.
import type { MarketOpsFeatureObservationView, MarketOpsStateTransitionView } from './marketopsState';

export type MarketOpsActiveIndicator = {
  key: string;
  title: string;
  tone: 'positive' | 'negative' | 'neutral';
  detail: string;
};

const usable = (quality: string) => quality === 'usable' || quality === 'usable_with_warning';
const number = (rows: MarketOpsFeatureObservationView[], key: string): number | null => {
  const row = rows.find((item) => item.featureKey === key && usable(item.qualityState) && item.numericValue !== null);
  return row?.numericValue ?? null;
};
const text = (rows: MarketOpsFeatureObservationView[], key: string): string | null => {
  const row = rows.find((item) => item.featureKey === key && usable(item.qualityState) && item.textValue !== null);
  return row?.textValue ?? null;
};
const fixed = (value: number, digits = 1) => value.toFixed(digits);

/**
 * Produces only conditions that are presently true for one persisted asset
 * state. Thresholds are display rules, not research-hypothesis triggers.
 */
export function activeMarketOpsIndicators(
  observations: MarketOpsFeatureObservationView[],
  transitions: MarketOpsStateTransitionView[],
): MarketOpsActiveIndicator[] {
  const out: MarketOpsActiveIndicator[] = [];
  const rsi = number(observations, 'rsi_14');
  if (rsi !== null && rsi >= 70) out.push({ key: 'overbought', title: 'Overbought', tone: 'negative', detail: `RSI ${fixed(rsi)} ≥ 70` });
  if (rsi !== null && rsi <= 30) out.push({ key: 'oversold', title: 'Oversold', tone: 'positive', detail: `RSI ${fixed(rsi)} ≤ 30` });

  const smaDistance = number(observations, 'distance_sma_20_pct');
  if (smaDistance !== null && smaDistance >= 5) out.push({ key: 'extended_above_sma', title: 'Extended above 20-day SMA', tone: 'negative', detail: `${fixed(smaDistance)}% above 20-day SMA` });
  if (smaDistance !== null && smaDistance <= -5) out.push({ key: 'extended_below_sma', title: 'Extended below 20-day SMA', tone: 'positive', detail: `${fixed(Math.abs(smaDistance))}% below 20-day SMA` });

  const volume = number(observations, 'volume_ratio_20d');
  if (volume !== null && volume >= 1.5) out.push({ key: 'unusual_volume', title: 'Unusual volume', tone: 'neutral', detail: `${fixed(volume, 2)}× the 20-day average` });

  const gap = number(observations, 'gap_pct');
  if (gap !== null && gap >= 1) out.push({ key: 'gap_up', title: 'Gap up', tone: 'positive', detail: `Opened +${fixed(gap)}% from prior close` });
  if (gap !== null && gap <= -1) out.push({ key: 'gap_down', title: 'Gap down', tone: 'negative', detail: `Opened −${fixed(Math.abs(gap))}% from prior close` });

  const atr = number(observations, 'atr_14_pct');
  if (atr !== null && atr >= 3) out.push({ key: 'elevated_range', title: 'Elevated trading range', tone: 'neutral', detail: `14-day ATR is ${fixed(atr)}% of close` });

  const ivRvRatio = number(observations, 'iv_rv_ratio_20d');
  if (ivRvRatio !== null && ivRvRatio >= 1.25) out.push({ key: 'iv_premium', title: 'Implied-volatility premium', tone: 'neutral', detail: `30-day ATM IV is ${fixed(ivRvRatio, 2)}× 20-day realized volatility` });

  const term = text(observations, 'term_structure_state');
  if (term === 'backwardation') out.push({ key: 'iv_backwardation', title: 'IV term structure in backwardation', tone: 'negative', detail: 'Near-term ATM IV exceeds longer-dated IV' });

  const putCallOI = number(observations, 'put_call_oi_ratio');
  if (putCallOI !== null && putCallOI >= 1.25) out.push({ key: 'put_oi_skew', title: 'Put open-interest skew', tone: 'negative', detail: `Put/call OI ratio ${fixed(putCallOI, 2)}` });
  if (putCallOI !== null && putCallOI <= 0.8) out.push({ key: 'call_oi_skew', title: 'Call open-interest skew', tone: 'positive', detail: `Put/call OI ratio ${fixed(putCallOI, 2)}` });

  const putCallVolume = number(observations, 'put_call_volume_ratio');
  if (putCallVolume !== null && putCallVolume >= 1.25) out.push({ key: 'put_volume_skew', title: 'Put-volume skew', tone: 'negative', detail: `Put/call volume ratio ${fixed(putCallVolume, 2)}` });
  if (putCallVolume !== null && putCallVolume <= 0.8) out.push({ key: 'call_volume_skew', title: 'Call-volume skew', tone: 'positive', detail: `Put/call volume ratio ${fixed(putCallVolume, 2)}` });

  for (const transition of transitions) {
    if (transition.featureKey !== 'oi_change_1d' || !usable(transition.qualityState) || transition.transitionValue === null) continue;
    const rare = (transition.zscore !== null && transition.zscore >= 2) || (transition.percentile !== null && transition.percentile >= 0.95);
    if (!rare) continue;
    const direction = transition.transitionValue >= 0 ? 'increase' : 'decrease';
    const rarity = transition.zscore !== null ? `${fixed(transition.zscore, 1)}σ` : `${fixed((transition.percentile ?? 0) * 100, 0)}th percentile`;
    out.push({ key: `unusual_oi:${transition.transitionId}`, title: `Unusual OI ${direction}`, tone: transition.transitionValue >= 0 ? 'neutral' : 'negative', detail: `${fixed(Math.abs(transition.transitionValue), 0)} contracts; ${rarity}` });
  }
  return out;
}
