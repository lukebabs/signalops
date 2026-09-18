import { describe, expect, it } from 'vitest';

import {
  isMarketOpsMarketHoliday,
  isMarketOpsTradingDay,
  marketOpsTrailingTradingDays,
} from './marketopsTradingCalendar';

describe('MarketOps trading calendar', () => {
  it('classifies regular sessions, weekends, and market holidays', () => {
    expect(isMarketOpsTradingDay('2026-08-21')).toBe(true);
    expect(isMarketOpsTradingDay('2026-08-22')).toBe(false);
    expect(isMarketOpsMarketHoliday('2026-09-07')).toBe(true);
    expect(isMarketOpsTradingDay('2026-09-07')).toBe(false);
    expect(isMarketOpsMarketHoliday('2027-07-05')).toBe(true);
    expect(isMarketOpsTradingDay('2027-07-05')).toBe(false);
  });

  it('builds trailing trading-day windows without weekend or holiday dates', () => {
    expect(marketOpsTrailingTradingDays('2026-09-08', 3)).toEqual([
      '2026-09-08',
      '2026-09-04',
      '2026-09-03',
    ]);
  });
});
