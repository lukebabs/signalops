import { describe, expect, it } from 'vitest';
import {
  summarizeMarketOpsOptionsCoverage,
  summarizeMarketOpsOptionsDistribution,
  summarizeMarketOpsOptionsChainRow,
  optionsBucketEntries,
  marketOpsOptionsDateOnly,
  MONEYNESS_BUCKET_ORDER,
  EXPIRATION_BUCKET_ORDER,
} from './marketopsOptions';

describe('summarizeMarketOpsOptionsCoverage (G128)', () => {
  it('reads scalar coverage fields without throwing', () => {
    const c = summarizeMarketOpsOptionsCoverage({
      tenant_id: 'tenant-local',
      symbol: 'NVDA',
      trade_day_count: 27,
      contract_count: 250,
      first_trade_date: '2025-12-02T00:00:00Z',
      last_trade_date: '2026-07-17T00:00:00Z',
      last_updated_at: '2026-07-17T03:33:03Z',
    });
    expect(c.symbol).toBe('NVDA');
    expect(c.tradeDayCount).toBe(27);
    expect(c.contractCount).toBe(250);
  });

  it('collapses non-object payloads to empty values', () => {
    expect(summarizeMarketOpsOptionsCoverage(null).symbol).toBe('');
    expect(summarizeMarketOpsOptionsCoverage('nope').contractCount).toBe(0);
  });
});

describe('optionsBucketEntries (G128)', () => {
  it('orders known keys and appends unknown ones', () => {
    const entries = optionsBucketEntries(
      {
        '105-110%': { call_open_interest: 1, put_open_interest: 2 },
        '<90%': { call_open_interest: 5, put_open_interest: 0 },
        'weird-bucket': { call_open_interest: 9, put_open_interest: 9 },
      },
      MONEYNESS_BUCKET_ORDER,
    );
    // Known keys render in canonical order; unknown key sorts after.
    expect(entries.map((e) => e.key)).toEqual(['<90%', '105-110%', 'weird-bucket']);
    expect(entries[0]).toEqual({
      key: '<90%',
      callOpenInterest: 5,
      putOpenInterest: 0,
      callVolume: 0,
      putVolume: 0,
      contractCount: 0,
    });
  });

  it('expiration buckets use the expiration order', () => {
    const entries = optionsBucketEntries(
      { '61d+': { call_open_interest: 4 }, '0-7d': { put_open_interest: 7 } },
      EXPIRATION_BUCKET_ORDER,
    );
    expect(entries.map((e) => e.key)).toEqual(['0-7d', '61d+']);
  });

  it('tolerates a non-object map and missing per-bucket fields', () => {
    expect(optionsBucketEntries(null, MONEYNESS_BUCKET_ORDER)).toEqual([]);
    expect(optionsBucketEntries('nope', MONEYNESS_BUCKET_ORDER)).toEqual([]);
    const entries = optionsBucketEntries({ '0-7d': {} }, EXPIRATION_BUCKET_ORDER);
    expect(entries[0].callOpenInterest).toBe(0);
  });
});

describe('summarizeMarketOpsOptionsDistribution (G128)', () => {
  it('normalizes scalars, orders buckets, passes metrics through, and reads source trade dates', () => {
    const v = summarizeMarketOpsOptionsDistribution({
      trade_date: '2026-07-17T00:00:00Z',
      window_name: '10_trade_days',
      provider: 'massive',
      source_id: 'src-massive',
      trade_days: 5,
      contract_count: 33,
      call_contract_count: 3,
      put_contract_count: 30,
      total_call_open_interest: 10,
      total_put_open_interest: 100,
      missing_open_interest_count: 33,
      call_put_open_interest_ratio: 0.1,
      call_put_volume_ratio: 0.2,
      ratio_zscore: 1.5,
      confidence: 0.8,
      moneyness_distribution: { '95-100%': { call_open_interest: 1, put_open_interest: 2 } },
      expiration_distribution: { '8-30d': { call_open_interest: 0, put_open_interest: 5 } },
      metrics: { z: 1.5 },
      source_trade_dates: ['2026-07-13T00:00:00Z', '2026-07-17T00:00:00Z'],
      updated_at: '2026-07-17T03:33:03Z',
    });
    expect(v.tradeDate).toBe('2026-07-17T00:00:00Z');
    expect(v.windowName).toBe('10_trade_days');
    expect(v.provider).toBe('massive');
    expect(v.missingOpenInterestCount).toBe(33);
    expect(v.callPutOpenInterestRatio).toBeCloseTo(0.1);
    expect(v.moneynessBuckets.map((e) => e.key)).toEqual(['95-100%']);
    expect(v.moneynessBuckets[0].putOpenInterest).toBe(2);
    expect(v.expirationBuckets.map((e) => e.key)).toEqual(['8-30d']);
    expect(v.metrics).toEqual({ z: 1.5 });
    expect(v.sourceTradeDates).toHaveLength(2);
  });

  it('collapses non-object payloads to empty values', () => {
    const v = summarizeMarketOpsOptionsDistribution(null);
    expect(v.callPutOpenInterestRatio).toBe(0);
    expect(v.moneynessBuckets).toEqual([]);
    expect(v.metrics).toEqual({});
  });
});

describe('summarizeMarketOpsOptionsChainRow (G128)', () => {
  it('keeps present numerics and nulls absent (omitempty) fields', () => {
    const r = summarizeMarketOpsOptionsChainRow({
      option_ticker: 'O:NVDA260717C00170000',
      contract_type: 'call',
      expiration_date: '2026-07-17T00:00:00Z',
      strike_price: 170,
      underlying_close: 172.5,
      moneyness: 0.9855,
      open_interest: 1543,
      volume: 123,
      implied_volatility: 0.45,
      delta: 0.51,
      provider: 'massive',
      source_id: 'src-massive',
      ingestion_run_id: 'optchain_1',
      payload_hash: 'hash',
      updated_at: '2026-07-17T03:33:03Z',
    });
    expect(r.optionTicker).toBe('O:NVDA260717C00170000');
    expect(r.contractType).toBe('call');
    expect(r.strikePrice).toBeCloseTo(170);
    expect(r.underlyingClose).toBeCloseTo(172.5);
    expect(r.openInterest).toBe(1543);
    // Absent omitempty greeks collapse to null, not 0.
    expect(r.gamma).toBeNull();
    expect(r.theta).toBeNull();
    expect(r.vega).toBeNull();
  });

  it('nulls open interest / volume when the server omits them', () => {
    const r = summarizeMarketOpsOptionsChainRow({ option_ticker: 'O:X', strike_price: 100 });
    expect(r.openInterest).toBeNull();
    expect(r.volume).toBeNull();
    expect(r.moneyness).toBeNull();
  });

  it('collapses non-object rows to the empty view', () => {
    const r = summarizeMarketOpsOptionsChainRow(null);
    expect(r.optionTicker).toBe('');
    expect(r.openInterest).toBeNull();
    expect(r.rawPayload).toEqual({});
  });
});

describe('marketOpsOptionsDateOnly (G128)', () => {
  it('extracts the YYYY-MM-DD portion of an RFC3339 timestamp', () => {
    expect(marketOpsOptionsDateOnly('2026-07-17T00:00:00Z')).toBe('2026-07-17');
    expect(marketOpsOptionsDateOnly('2026-07-17')).toBe('2026-07-17');
    expect(marketOpsOptionsDateOnly('')).toBe('');
    expect(marketOpsOptionsDateOnly(null as unknown as string)).toBe('');
  });
});
