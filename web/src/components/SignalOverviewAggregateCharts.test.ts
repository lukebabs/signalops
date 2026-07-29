import { describe, expect, it } from 'vitest';

import { riskRewardRegimePoints } from './SignalOverviewAggregateCharts';

describe('riskRewardRegimePoints', () => {
  it('calculates median and interquartile range from scored assets', () => {
    const regime = riskRewardRegimePoints([{
      trade_date: "2026-07-28",
      categories: [{ key: "bullish", count: 2, members: [{ ticker: "A", label: "bullish", score: 60, as_of: "2026-07-28" }, { ticker: "B", label: "bullish", score: 20, as_of: "2026-07-28" }] }, { key: "neutral", count: 0, members: [] }, { key: "bearish", count: 2, members: [{ ticker: "C", label: "bearish", score: -20, as_of: "2026-07-28" }, { ticker: "D", label: "bearish", score: -60, as_of: "2026-07-28" }] }],
    }]);

    expect(regime).toEqual([expect.objectContaining({ tradeDate: "2026-07-28", median: 0, lowerQuartile: -30, upperQuartile: 30 })]);
  });

  it("skips unscored dates and makes a single score its full range", () => {
    const regime = riskRewardRegimePoints([{ trade_date: "2026-07-27", categories: [{ key: "neutral", count: 1, members: [{ ticker: "A", label: "neutral", as_of: "2026-07-27" }] }] }, { trade_date: "2026-07-28", categories: [{ key: "bullish", count: 1, members: [{ ticker: "A", label: "bullish", score: 42, as_of: "2026-07-28" }] }] }]);

    expect(regime).toEqual([expect.objectContaining({ tradeDate: "2026-07-28", median: 42, lowerQuartile: 42, upperQuartile: 42 })]);
  });
});
