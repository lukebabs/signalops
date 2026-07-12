import { describe, expect, it } from 'vitest';
import {
  summarizeBacktestMetrics,
  isZeroInputBacktest,
  compareBacktestRuns,
  dominantRecommendation,
  parseBacktestSymbols,
  policyResultsByProposal,
  recommendationLabel,
  recommendationStyle,
} from './marketopsBacktests';

describe('summarizeBacktestMetrics (G081)', () => {
  it('reads the canonical metric fields', () => {
    const s = summarizeBacktestMetrics({
      scanned: 5,
      signals: 3,
      artifacts: 3,
      graph_proposals: 4,
      policy_results: 4,
      recommendation_counts: { auto_accept_candidate: 2, manual_review_required: 2 },
      batches: 1,
      max_records: 5,
      batch_size: 5,
      started_at: '2026-07-12T00:00:00Z',
      completed_at: '2026-07-12T00:00:01Z',
    });
    expect(s.scanned).toBe(5);
    expect(s.signals).toBe(3);
    expect(s.graphProposals).toBe(4);
    expect(s.policyResults).toBe(4);
    expect(s.recommendationCounts).toEqual({ auto_accept_candidate: 2, manual_review_required: 2 });
    expect(s.batches).toBe(1);
    expect(s.startedAt).toBe('2026-07-12T00:00:00Z');
    expect(s.completedAt).toBe('2026-07-12T00:00:01Z');
  });

  it('tolerates a non-object metrics payload', () => {
    expect(summarizeBacktestMetrics(undefined)).toEqual({
      scanned: 0, signals: 0, artifacts: 0, graphProposals: 0, policyResults: 0,
      recommendationCounts: {}, batches: 0, maxRecords: 0, batchSize: 0,
      startedAt: '', completedAt: '',
    });
    expect(summarizeBacktestMetrics(null).signals).toBe(0);
    expect(summarizeBacktestMetrics('not-an-object').scanned).toBe(0);
    expect(summarizeBacktestMetrics([1, 2, 3]).policyResults).toBe(0);
  });

  it('coerces numeric strings and ignores malformed values', () => {
    const s = summarizeBacktestMetrics({
      scanned: '7',
      signals: null,
      graph_proposals: { oops: true },
      recommendation_counts: { auto_accept_candidate: '3', bad: 'NaN', manual_review_required: 1 },
      started_at: 12345,
    });
    expect(s.scanned).toBe(7);
    expect(s.signals).toBe(0);
    expect(s.graphProposals).toBe(0);
    expect(s.recommendationCounts).toEqual({ auto_accept_candidate: 3, manual_review_required: 1 });
    expect(s.startedAt).toBe('');
  });

  it('identifies successful runs with no matching normalized events', () => {
    expect(isZeroInputBacktest('succeeded', { scanned: 0 })).toBe(true);
    expect(isZeroInputBacktest('succeeded', { scanned: '0' })).toBe(true);
    expect(isZeroInputBacktest('succeeded', { scanned: 1 })).toBe(false);
    expect(isZeroInputBacktest('failed', { scanned: 0 })).toBe(false);
  });
});

describe('dominantRecommendation (G081)', () => {
  it('returns the highest-count recommendation', () => {
    expect(dominantRecommendation({ auto_accept_candidate: 1, manual_review_required: 3 })).toEqual({
      key: 'manual_review_required',
      count: 3,
    });
  });

  it('returns null when all counts are zero or empty', () => {
    expect(dominantRecommendation({})).toBeNull();
    expect(dominantRecommendation({ auto_accept_candidate: 0 })).toBeNull();
  });
});

describe('parseBacktestSymbols (G081)', () => {
  it('uppercases, trims, and de-duplicates comma/space input', () => {
    expect(parseBacktestSymbols('spy, aapl aapl,MSFT')).toEqual(['SPY', 'AAPL', 'MSFT']);
    expect(parseBacktestSymbols('  spy ,  spy ')).toEqual(['SPY']);
    expect(parseBacktestSymbols('SPY')).toEqual(['SPY']);
  });

  it('drops empty values', () => {
    expect(parseBacktestSymbols(',,, ,')).toEqual([]);
    expect(parseBacktestSymbols('')).toEqual([]);
    expect(parseBacktestSymbols('SPY,,AAPL,')).toEqual(['SPY', 'AAPL']);
  });
});


describe('compareBacktestRuns (G082)', () => {
  it('aggregates run metrics, zero-input runs, and recommendation shares', () => {
    const summary = compareBacktestRuns([
      {
        run_id: 'bt-1', tenant_id: 'tenant-local', app_id: 'marketops', domain: 'market_data', use_case: 'daily_market_surveillance',
        source_id: 'src-massive', source_adapter: 'market_data.massive', dataset: 'equity_eod_prices', detector_id: 'marketops.dsm.taxonomy_v1', detector_version: 'v1', status: 'succeeded', requested_by: 'operator', window_start: '', window_end: '', started_at: '', filters: {}, parameters: {}, created_at: '', updated_at: '',
        metrics: { scanned: 2, signals: 1, artifacts: 1, graph_proposals: 5, policy_results: 5, recommendation_counts: { auto_accept_candidate: 5 } },
      },
      {
        run_id: 'bt-2', tenant_id: 'tenant-local', app_id: 'marketops', domain: 'market_data', use_case: 'daily_market_surveillance',
        source_id: 'src-massive', source_adapter: 'market_data.massive', dataset: 'equity_eod_prices', detector_id: 'marketops.dsm.taxonomy_v1', detector_version: 'v1', status: 'succeeded', requested_by: 'operator', window_start: '', window_end: '', started_at: '', filters: {}, parameters: {}, created_at: '', updated_at: '',
        metrics: { scanned: 0, signals: 0, artifacts: 0, graph_proposals: 0, policy_results: 0, recommendation_counts: {} },
      },
      {
        run_id: 'bt-3', tenant_id: 'tenant-local', app_id: 'marketops', domain: 'market_data', use_case: 'daily_market_surveillance',
        source_id: 'src-massive', source_adapter: 'market_data.massive', dataset: 'options_contracts_daily', detector_id: 'marketops.dsm.taxonomy_v1', detector_version: 'v1', status: 'failed', requested_by: 'operator', window_start: '', window_end: '', started_at: '', filters: {}, parameters: {}, created_at: '', updated_at: '',
        metrics: { scanned: 3, signals: 2, artifacts: 2, graph_proposals: 4, policy_results: 4, recommendation_counts: { manual_review_required: 3, auto_accept_candidate: 1 } },
      },
    ]);

    expect(summary.runs).toBe(3);
    expect(summary.succeeded).toBe(2);
    expect(summary.failed).toBe(1);
    expect(summary.zeroInput).toBe(1);
    expect(summary.scanned).toBe(5);
    expect(summary.signals).toBe(3);
    expect(summary.signalYieldPct).toBe(60);
    expect(summary.policyResultsPerSignal).toBe(3);
    expect(summary.recommendationCounts).toEqual({ auto_accept_candidate: 6, manual_review_required: 3 });
    expect(summary.dominantRecommendation).toEqual({ key: 'auto_accept_candidate', count: 6, share: 6 / 9 });
    expect(summary.datasets).toEqual(['equity_eod_prices', 'options_contracts_daily']);
  });

  it('returns an empty summary for non-array input', () => {
    const summary = compareBacktestRuns(null);
    expect(summary.runs).toBe(0);
    expect(summary.signalYieldPct).toBe(0);
    expect(summary.dominantRecommendation).toBeNull();
  });
});

describe('policyResultsByProposal (G081)', () => {
  it('indexes results by proposal_id', () => {
    const map = policyResultsByProposal([
      { proposal_id: 'p1', recommendation: 'auto_accept_candidate', reason: 'ok' },
      { proposal_id: 'p2', recommendation: 'manual_review_required', reason: 'low conf' },
    ] as never);
    expect(map.get('p1')?.recommendation).toBe('auto_accept_candidate');
    expect(map.get('p2')?.reason).toBe('low conf');
    expect(map.has('p3')).toBe(false);
  });

  it('skips entries without a proposal_id and tolerates non-arrays', () => {
    const map = policyResultsByProposal([
      { recommendation: 'auto_accept_candidate' },
      { proposal_id: '', recommendation: 'auto_accept_candidate' },
      { proposal_id: 'p9', recommendation: 'auto_reject_candidate' },
    ] as never);
    expect(map.size).toBe(1);
    expect(map.get('p9')?.recommendation).toBe('auto_reject_candidate');
    expect(policyResultsByProposal(undefined).size).toBe(0);
    expect(policyResultsByProposal(null).size).toBe(0);
  });
});

describe('recommendation presentation (G081)', () => {
  it('humanizes the recommendation key', () => {
    expect(recommendationLabel('manual_review_required')).toBe('manual review required');
    expect(recommendationLabel('auto_accept_candidate')).toBe('auto accept candidate');
  });

  it('resolves a style for known keys and falls back for unknown ones', () => {
    expect(recommendationStyle('auto_accept_candidate')).toContain('emerald');
    expect(recommendationStyle('manual_review_required')).toContain('amber');
    expect(recommendationStyle('supersede_candidate')).toContain('violet');
    expect(recommendationStyle('unknown_future_value')).toContain('gray');
  });
});
