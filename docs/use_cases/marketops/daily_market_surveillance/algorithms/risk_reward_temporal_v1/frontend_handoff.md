# Risk/Reward panel — frontend implementation handoff

## Purpose and placement

Add a **Risk/Reward** panel to the expanded selected-asset content in the MarketOps Assets view. Place it above Quantitative Corroboration, after the asset overview. It describes a persisted EOD technical posture; it is not a trade recommendation, does not modify hypotheses, and does not make a provider request.

Read `GET /v1/tenants/{tenant_id}/marketops/assets/{symbol}/algorithm-observations`. Reuse the existing query key, authentication, retry, stale-data retention, and loading/error conventions used by Algorithm Evidence.

## API contract

The response adds a `risk_reward` object:

```ts
type RiskRewardPoint = {
  algorithm_result_id: string;
  trade_date: string; // YYYY-MM-DD, newest first
  score: number; // -100..100; negative=bearish, positive=bullish
  direction: 'bullish' | 'bearish' | 'neutral';
  risk_level: 'low' | 'medium' | 'high' | 'unavailable';
  confidence: number; // 0..1
  severity: 'info' | 'low' | 'medium' | 'high';
  technical_factors: Array<{
    key: string;
    direction: 'bullish' | 'bearish' | 'neutral';
    weight: number;
    value: number;
    detail: string;
  }>;
  speculative_corroboration: {
    direction: 'bullish' | 'bearish' | 'neutral' | 'unavailable';
    agreement: 'aligned' | 'bullish' | 'bearish' | 'neutral';
    put_call_volume_ratio?: number;
    deviation_pct?: number;
  };
  research_only: true;
};

type RiskRewardResponse = {
  latest?: RiskRewardPoint;
  history: RiskRewardPoint[]; // newest 60 at most
};
```

Treat absent `latest` and empty `history` as **No persisted risk/reward analysis yet**. Do not derive a score client-side, substitute zeros, or infer unavailable factors.

The asset-list risk/reward summary additionally exposes optional `previous_trade_date`, `previous_score`, and `score_change` fields. They compare the latest score with the preceding persisted trading session (not the previous calendar day). A positive change is an improving technical posture; a negative change is regressing. When no prior persisted session exists, show an unavailable/awaiting state rather than a zero change.

## Visual and interaction specification

1. Header: `Risk/Reward` with subtext: `Persisted EOD technical posture · research-only, not a trading recommendation.`
2. Current-summary row, when `latest` exists:
   - Direction badge: bullish green/up, bearish red/down, neutral gray.
   - Signed technical score, for example `−60 / 100` or `+45 / 100`.
   - One-session evolution: `↑ Improving · +12`, `↓ Regressing · −12`, or `→ Unchanged · 0`, compared with the prior persisted trading session. Do not present it as a forecast or recommendation.
   - Confidence as a whole percent.
   - Risk badge: low neutral, medium amber, high red; `unavailable` gray.
   - EOD date, never a live/intraday timestamp.
3. Technical factors:
   - One compact row per supplied factor: human-readable title, directional icon/tone, formatted value, weight, and `detail` explanation.
   - Preserve API order. This makes current evidence explicit without presenting it as a recommendation.
4. **Speculative options corroboration**:
   - Visually separate it from Technical factors.
   - Show put/call volume ratio and 10-session deviation when present.
   - Explain: `Put/call below 1.0 is call-heavy / bullish positioning; above 1.0 is put-heavy / bearish positioning.`
   - Show `Aligned`, `Divergent`, `Neutral`, or `Unavailable`; never blend it into the technical score in UI copy.
5. History chart:
   - Compact 60-trading-day line chart, trade date on x-axis and fixed `-100..100` y-axis with a zero baseline.
   - Green above-zero points/segments, red below-zero, and neutral gray at zero.
   - Tooltip shows date, direction, score, confidence, risk, and speculative corroboration state.
   - Missing dates remain gaps; do not interpolate.

Use existing asset-detail card styling and ECharts conventions. The panel must not move or replace Hypotheses, Algorithm Evidence, Price/Sentiment/Corroboration, or the options card.

## States and accessibility

- Loading: skeleton matching summary and factor rows.
- Error: compact retryable inline error; retain prior query data if available.
- Empty: explain that a persisted Market State session with usable inputs is required.
- Partial: display available technical factors and the low-confidence state; do not hide missing inputs.
- Use text labels in addition to color/icons, expose chart summary through accessible text, and ensure tooltips are supplemental rather than the only source of meaning.

## Acceptance criteria

- A selected asset with a persisted result displays latest EOD score, direction, confidence, risk, technical factors, and separate put/call corroboration.
- An asset without results displays the empty state and makes no direct Massive/provider call.
- The chart renders up to 60 persisted points in chronological display order with a zero baseline and no interpolation.
- Bearish/bullish/neutral colors and labels match the response; put/call corroboration never changes the technical score.
- Existing overview, hypotheses, quantitative corroboration, dashboard, and asset-list behavior remain unchanged.
