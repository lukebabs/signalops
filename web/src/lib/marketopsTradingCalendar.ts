// MarketOps UI trading-day helper.
//
// This intentionally mirrors scripts/lib/marketops_trading_calendar.sh so
// operator-facing 10/20 trading-day filters stay aligned with scheduler
// eligibility. Update both files together when the exchange calendar rolls.
const MARKETOPS_US_MARKET_HOLIDAYS = new Set([
  '2026-01-01',
  '2026-01-19',
  '2026-02-16',
  '2026-04-03',
  '2026-05-25',
  '2026-06-19',
  '2026-07-03',
  '2026-09-07',
  '2026-11-26',
  '2026-12-25',
  '2027-01-01',
  '2027-01-18',
  '2027-02-15',
  '2027-03-26',
  '2027-05-31',
  '2027-06-18',
  '2027-07-05',
  '2027-09-06',
  '2027-11-25',
  '2027-12-24',
]);

export function isMarketOpsMarketHoliday(date: string): boolean {
  return MARKETOPS_US_MARKET_HOLIDAYS.has(date.slice(0, 10));
}

export function isMarketOpsTradingDay(date: string): boolean {
  const day = date.slice(0, 10);
  if (!/^\d{4}-\d{2}-\d{2}$/.test(day)) return false;
  const parsed = new Date(`${day}T00:00:00Z`);
  if (Number.isNaN(parsed.getTime())) return false;
  const weekday = parsed.getUTCDay();
  return weekday !== 0 && weekday !== 6 && !isMarketOpsMarketHoliday(day);
}

export function marketOpsTrailingTradingDays(latestDate: string, days: number): string[] {
  const latest = latestDate.slice(0, 10);
  if (!Number.isFinite(days) || days <= 0 || !/^\d{4}-\d{2}-\d{2}$/.test(latest)) {
    return [];
  }
  const tradingDays: string[] = [];
  const cursor = new Date(`${latest}T00:00:00Z`);
  if (Number.isNaN(cursor.getTime())) return [];
  while (tradingDays.length < days) {
    const day = cursor.toISOString().slice(0, 10);
    if (isMarketOpsTradingDay(day)) tradingDays.push(day);
    cursor.setUTCDate(cursor.getUTCDate() - 1);
  }
  return tradingDays;
}
