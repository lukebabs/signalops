import { afterEach, describe, expect, it, vi } from 'vitest';
import { QueryClient } from '@tanstack/react-query';

// Hoisted mutable auth state so the mocked auth modules read live values.
const state = vi.hoisted(() => ({ token: 'jwt-abc' as string | null, authEnabled: true }));

vi.mock('../auth/config', () => ({
  authConfig: {
    get authEnabled() {
      return state.authEnabled;
    },
    issuer: 'https://auth.syncratic.co/realms/syncratic',
    clientId: 'signalops-web',
    audience: 'signalops-api',
    realm: 'syncratic',
  },
}));
vi.mock('../auth/session', () => ({
  getAccessToken: () => state.token,
}));

// Import the client AFTER the mocks are registered.
const { api } = await import('./client');
const {
  queryKeys,
  applyBacktestCreateResult,
  applyBacktestCalibrationSummaryCreateResult,
  applyBacktestCalibrationBaselineCreateResult,
  applyBacktestCalibrationComparisonCreateResult,
  applyBacktestEvaluationCreateResult,
  applyBacktestPromotionCandidateCreateResult,
  applyBacktestPromotionCandidateDecisionResult,
} = await import('./queries');

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  state.token = 'jwt-abc';
  state.authEnabled = true;
});

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

describe('MarketOps back-tests API client (G081)', () => {
  it('builds the list path with defaults (tenant-local, taxonomy detector, limit 50) + bearer token', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ backtest_runs: [] }));
    vi.stubGlobal('fetch', fetchMock);
    state.authEnabled = true;
    state.token = 'jwt-abc';

    await api.listMarketOpsBacktests({});

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/marketops/backtests');
    expect(url).toContain('tenant_id=tenant-local');
    expect(url).toContain('detector_id=marketops.dsm.taxonomy_v1');
    expect(url).toContain('limit=50');
    expect(fetchMock.mock.calls[0][1].headers['Authorization']).toBe('Bearer jwt-abc');
  });

  it('applies status + detector + limit list filters', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ backtest_runs: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listMarketOpsBacktests({
      tenant_id: 'tenant-local',
      detector_id: 'marketops.dsm.taxonomy_v1',
      status: 'succeeded',
      limit: 25,
    });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('status=succeeded');
    expect(url).toContain('detector_id=marketops.dsm.taxonomy_v1');
    expect(url).toContain('limit=25');
  });

  it('encodes the run id and sends tenant_id on the detail path', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ backtest_run: {} }));
    vi.stubGlobal('fetch', fetchMock);

    await api.getMarketOpsBacktest('bt:run/1', 'tenant-local');

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/marketops/backtests/bt%3Arun%2F1');
    expect(url).toContain('tenant_id=tenant-local');
  });

  it('posts the create body to /v1/marketops/backtests with bearer token + content type', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ backtest_run: { run_id: 'bt-1' }, metrics: { scanned: 5 } }, 201),
    );
    vi.stubGlobal('fetch', fetchMock);
    state.authEnabled = true;
    state.token = 'jwt-abc';

    await api.createMarketOpsBacktest({
      tenant_id: 'tenant-local',
      source_id: 'src-massive',
      dataset: 'equity_eod_prices',
      detector_id: 'marketops.dsm.taxonomy_v1',
      detector_version: 'v1',
      window_start: '2026-07-09T00:00:00Z',
      window_end: '2026-07-10T00:00:00Z',
      symbols: ['SPY'],
      max_records: 5,
      batch_size: 5,
      auto_accept_confidence: 0.75,
    });

    expect(String(fetchMock.mock.calls[0][0])).toContain('/v1/marketops/backtests');
    expect(fetchMock.mock.calls[0][1].method).toBe('POST');
    expect(fetchMock.mock.calls[0][1].headers['Authorization']).toBe('Bearer jwt-abc');
    expect(fetchMock.mock.calls[0][1].headers['Content-Type']).toBe('application/json');
    expect(JSON.parse(fetchMock.mock.calls[0][1].body)).toEqual({
      tenant_id: 'tenant-local',
      source_id: 'src-massive',
      dataset: 'equity_eod_prices',
      detector_id: 'marketops.dsm.taxonomy_v1',
      detector_version: 'v1',
      window_start: '2026-07-09T00:00:00Z',
      window_end: '2026-07-10T00:00:00Z',
      symbols: ['SPY'],
      max_records: 5,
      batch_size: 5,
      auto_accept_confidence: 0.75,
    });
  });

  it('omits an unset run_id from the create body', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ backtest_run: { run_id: 'bt-auto' } }, 201));
    vi.stubGlobal('fetch', fetchMock);

    await api.createMarketOpsBacktest({
      tenant_id: 'tenant-local',
      window_start: '2026-07-09T00:00:00Z',
      window_end: '2026-07-10T00:00:00Z',
    });

    const body = JSON.parse(fetchMock.mock.calls[0][1].body);
    expect(body.run_id).toBeUndefined();
    expect(body.tenant_id).toBe('tenant-local');
  });

  it('builds the signals path with run id, signal_type, and limit', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ backtest_signals: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listMarketOpsBacktestSignals('bt-1', { tenant_id: 'tenant-local', signal_type: 'marketops.dsm.pinning_risk', limit: 50 });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/marketops/backtests/bt-1/signals');
    expect(url).toContain('signal_type=marketops.dsm.pinning_risk');
    expect(url).toContain('limit=50');
  });

  it('builds the graph-proposals path with recommendation + candidate_type + subject_symbol', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ backtest_graph_proposals: [], policy_results: [] }),
    );
    vi.stubGlobal('fetch', fetchMock);

    await api.listMarketOpsBacktestGraphProposals('bt-1', {
      tenant_id: 'tenant-local',
      recommendation: 'manual_review_required',
      candidate_type: 'relationship_candidate',
      subject_symbol: 'AAPL',
      limit: 50,
    });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/marketops/backtests/bt-1/graph-proposals');
    expect(url).toContain('recommendation=manual_review_required');
    expect(url).toContain('candidate_type=relationship_candidate');
    expect(url).toContain('subject_symbol=AAPL');
    expect(url).toContain('limit=50');
  });

  it('creates and lists persisted calibration summaries via the G082 endpoint', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ calibration_summary: { summary_id: 'btcal-1', run_count: 2 } }, 201))
      .mockResolvedValueOnce(jsonResponse({ calibration_summaries: [{ summary_id: 'btcal-1', run_count: 2 }] }))
      .mockResolvedValueOnce(jsonResponse({ calibration_summary: { summary_id: 'btcal-1', run_count: 2 } }));
    vi.stubGlobal('fetch', fetchMock);

    const created = await api.createMarketOpsBacktestCalibrationSummary({
      tenant_id: 'tenant-local',
      detector_id: 'marketops.dsm.taxonomy_v1',
      status: 'succeeded',
      limit: 50,
    });
    const list = await api.listMarketOpsBacktestCalibrationSummaries({ tenant_id: 'tenant-local', detector_id: 'marketops.dsm.taxonomy_v1', limit: 10 });
    const detail = await api.getMarketOpsBacktestCalibrationSummary('btcal-1');

    expect(String(fetchMock.mock.calls[0][0])).toContain('/v1/marketops/backtest-calibration-summaries');
    expect(fetchMock.mock.calls[0][1].method).toBe('POST');
    expect(JSON.parse(fetchMock.mock.calls[0][1].body)).toEqual({
      tenant_id: 'tenant-local',
      detector_id: 'marketops.dsm.taxonomy_v1',
      status: 'succeeded',
      limit: 50,
    });
    expect(String(fetchMock.mock.calls[1][0])).toContain('detector_id=marketops.dsm.taxonomy_v1');
    expect(String(fetchMock.mock.calls[2][0])).toContain('/v1/marketops/backtest-calibration-summaries/btcal-1');
    expect(created.calibration_summary.summary_id).toBe('btcal-1');
    expect(list.calibration_summaries[0].run_count).toBe(2);
    expect(detail.calibration_summary.summary_id).toBe('btcal-1');
  });

  it('parses the list, detail, create, signals, and graph-proposals envelopes', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ backtest_runs: [{ run_id: 'bt-1', status: 'succeeded' }] }))
      .mockResolvedValueOnce(jsonResponse({ backtest_run: { run_id: 'bt-1', status: 'succeeded' } }))
      .mockResolvedValueOnce(jsonResponse({ backtest_run: { run_id: 'bt-1' }, metrics: { scanned: 5 } }, 201))
      .mockResolvedValueOnce(jsonResponse({ backtest_signals: [{ run_id: 'bt-1', signal: { signal_id: 'sig-1' } }] }))
      .mockResolvedValueOnce(
        jsonResponse({
          backtest_graph_proposals: [{ run_id: 'bt-1', graph_proposal: { proposal_id: 'p1' } }],
          policy_results: [{ proposal_id: 'p1', recommendation: 'auto_accept_candidate' }],
        }),
      );
    vi.stubGlobal('fetch', fetchMock);

    const list = await api.listMarketOpsBacktests({});
    const detail = await api.getMarketOpsBacktest('bt-1');
    const created = await api.createMarketOpsBacktest({ tenant_id: 'tenant-local', window_start: '2026-07-09T00:00:00Z', window_end: '2026-07-10T00:00:00Z' });
    const signals = await api.listMarketOpsBacktestSignals('bt-1', {});
    const gp = await api.listMarketOpsBacktestGraphProposals('bt-1', {});

    expect(list.backtest_runs[0].run_id).toBe('bt-1');
    expect(detail.backtest_run.status).toBe('succeeded');
    expect(created.metrics).toEqual({ scanned: 5 });
    expect(signals.backtest_signals[0].signal.signal_id).toBe('sig-1');
    expect(gp.backtest_graph_proposals[0].graph_proposal.proposal_id).toBe('p1');
    expect(gp.policy_results[0].recommendation).toBe('auto_accept_candidate');
  });
});

describe('MarketOps back-test calibration baselines + comparisons API client (G083)', () => {
  it('creates, lists, and fetches a baseline via the G083 endpoints with bearer + content type', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        jsonResponse({ calibration_baseline: { baseline_id: 'btbase-1', status: 'active', summary_id: 'btcal-1' } }, 201),
      )
      .mockResolvedValueOnce(jsonResponse({ calibration_baselines: [{ baseline_id: 'btbase-1' }] }))
      .mockResolvedValueOnce(jsonResponse({ calibration_baseline: { baseline_id: 'btbase-1' } }));
    vi.stubGlobal('fetch', fetchMock);
    state.authEnabled = true;
    state.token = 'jwt-abc';

    const created = await api.createMarketOpsBacktestCalibrationBaseline({
      tenant_id: 'tenant-local',
      name: 'Taxonomy July baseline',
      summary_id: 'btcal-1',
      description: 'note',
      status: 'active',
    });
    const list = await api.listMarketOpsBacktestCalibrationBaselines({
      tenant_id: 'tenant-local',
      detector_id: 'marketops.dsm.taxonomy_v1',
      status: 'active',
      limit: 50,
    });
    const detail = await api.getMarketOpsBacktestCalibrationBaseline('btbase-1');

    // Create: POST to the baselines endpoint with bearer + JSON body.
    expect(String(fetchMock.mock.calls[0][0])).toContain('/v1/marketops/backtest-calibration-baselines');
    expect(fetchMock.mock.calls[0][1].method).toBe('POST');
    expect(fetchMock.mock.calls[0][1].headers['Authorization']).toBe('Bearer jwt-abc');
    expect(fetchMock.mock.calls[0][1].headers['Content-Type']).toBe('application/json');
    expect(JSON.parse(fetchMock.mock.calls[0][1].body)).toEqual({
      tenant_id: 'tenant-local',
      name: 'Taxonomy July baseline',
      summary_id: 'btcal-1',
      description: 'note',
      status: 'active',
    });
    // List: GET with defaults (tenant-local, taxonomy detector, limit 50) + status filter.
    const listUrl = String(fetchMock.mock.calls[1][0]);
    expect(listUrl).toContain('tenant_id=tenant-local');
    expect(listUrl).toContain('detector_id=marketops.dsm.taxonomy_v1');
    expect(listUrl).toContain('status=active');
    expect(listUrl).toContain('limit=50');
    // Detail: GET with encoded baseline id.
    expect(String(fetchMock.mock.calls[2][0])).toContain('/v1/marketops/backtest-calibration-baselines/btbase-1');
    // Envelopes parse.
    expect(created.calibration_baseline.baseline_id).toBe('btbase-1');
    expect(list.calibration_baselines[0].baseline_id).toBe('btbase-1');
    expect(detail.calibration_baseline.baseline_id).toBe('btbase-1');
  });

  it('omits unset optional baseline fields and still sends required ones', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ calibration_baseline: { baseline_id: 'btbase-2' } }, 201),
    );
    vi.stubGlobal('fetch', fetchMock);

    await api.createMarketOpsBacktestCalibrationBaseline({
      tenant_id: 'tenant-local',
      name: 'bare',
      summary_id: 'btcal-1',
    });

    const body = JSON.parse(fetchMock.mock.calls[0][1].body);
    expect(body.tenant_id).toBe('tenant-local');
    expect(body.name).toBe('bare');
    expect(body.summary_id).toBe('btcal-1');
    expect(body.description).toBeUndefined();
    expect(body.status).toBeUndefined();
  });

  it('creates, lists, and fetches a comparison via the G083 endpoints', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        jsonResponse({
          calibration_comparison: {
            comparison_id: 'btcmp-1',
            baseline_id: 'btbase-1',
            baseline_summary_id: 'btcal-base',
            candidate_summary_id: 'btcal-cand',
            recommendation: 'neutral_candidate',
            recommendation_reason: 'within tolerance',
            comparison_metrics: { baseline: {}, candidate: {}, deltas: { run_count_delta: 0 } },
          },
        }, 201),
      )
      .mockResolvedValueOnce(jsonResponse({ calibration_comparisons: [{ comparison_id: 'btcmp-1' }] }))
      .mockResolvedValueOnce(jsonResponse({ calibration_comparison: { comparison_id: 'btcmp-1' } }));
    vi.stubGlobal('fetch', fetchMock);

    const created = await api.createMarketOpsBacktestCalibrationComparison({
      tenant_id: 'tenant-local',
      baseline_id: 'btbase-1',
      candidate_summary_id: 'btcal-cand',
    });
    const list = await api.listMarketOpsBacktestCalibrationComparisons({
      tenant_id: 'tenant-local',
      baseline_id: 'btbase-1',
      recommendation: 'neutral_candidate',
      limit: 50,
    });
    const detail = await api.getMarketOpsBacktestCalibrationComparison('btcmp-1');

    // Create body carries only the comparison request fields.
    expect(String(fetchMock.mock.calls[0][0])).toContain('/v1/marketops/backtest-calibration-comparisons');
    expect(fetchMock.mock.calls[0][1].method).toBe('POST');
    expect(JSON.parse(fetchMock.mock.calls[0][1].body)).toEqual({
      tenant_id: 'tenant-local',
      baseline_id: 'btbase-1',
      candidate_summary_id: 'btcal-cand',
    });
    // List: baseline-scoped + recommendation filter.
    const listUrl = String(fetchMock.mock.calls[1][0]);
    expect(listUrl).toContain('baseline_id=btbase-1');
    expect(listUrl).toContain('recommendation=neutral_candidate');
    expect(listUrl).toContain('limit=50');
    // Detail: encoded comparison id.
    expect(String(fetchMock.mock.calls[2][0])).toContain('/v1/marketops/backtest-calibration-comparisons/btcmp-1');
    // Envelopes parse, including nested comparison_metrics.
    expect(created.calibration_comparison.recommendation).toBe('neutral_candidate');
    expect(created.calibration_comparison.comparison_metrics.deltas?.run_count_delta).toBe(0);
    expect(list.calibration_comparisons[0].comparison_id).toBe('btcmp-1');
    expect(detail.calibration_comparison.comparison_id).toBe('btcmp-1');
  });

  it('encodes special characters in baseline + comparison ids on the detail paths', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ calibration_baseline: {} }))
      .mockResolvedValueOnce(jsonResponse({ calibration_comparison: {} }));
    vi.stubGlobal('fetch', fetchMock);

    await api.getMarketOpsBacktestCalibrationBaseline('btbase/a b');
    await api.getMarketOpsBacktestCalibrationComparison('btcmp/a b');

    expect(String(fetchMock.mock.calls[0][0])).toContain('/v1/marketops/backtest-calibration-baselines/btbase%2Fa%20b');
    expect(String(fetchMock.mock.calls[1][0])).toContain('/v1/marketops/backtest-calibration-comparisons/btcmp%2Fa%20b');
  });
});

describe('MarketOps back-test query keys (G081)', () => {
  it('uses stable list, detail, signals, and graph-proposals keys', () => {
    const filter = { tenant_id: 'tenant-local', limit: 50 };
    const sigFilter = { tenant_id: 'tenant-local', limit: 50 };
    const gpFilter = { tenant_id: 'tenant-local', limit: 50 };

    expect(queryKeys.marketOpsBacktests(filter)).toEqual(['marketops-backtests', filter]);
    expect(queryKeys.marketOpsBacktests({ ...filter })).toEqual(['marketops-backtests', filter]);
    expect(queryKeys.marketOpsBacktest('bt-1', 'tenant-local')).toEqual(['marketops-backtest', 'bt-1', 'tenant-local']);
    expect(queryKeys.marketOpsBacktestSignals('bt-1', sigFilter)).toEqual(['marketops-backtest-signals', 'bt-1', sigFilter]);
    expect(queryKeys.marketOpsBacktestGraphProposals('bt-1', gpFilter)).toEqual([
      'marketops-backtest-graph-proposals',
      'bt-1',
      gpFilter,
    ]);
    expect(queryKeys.marketOpsBacktestCalibrationSummaries({ tenant_id: 'tenant-local', limit: 10 })[0]).toBe('marketops-backtest-calibration-summaries');
    expect(queryKeys.marketOpsBacktestCalibrationSummary('btcal-1')).toEqual(['marketops-backtest-calibration-summary', 'btcal-1']);
  });

  it('uses stable baseline + comparison list/detail keys (G083)', () => {
    const baselineFilter = { tenant_id: 'tenant-local', status: 'active' as const, limit: 50 };
    const comparisonFilter = { tenant_id: 'tenant-local', baseline_id: 'btbase-1', limit: 50 };

    expect(queryKeys.marketOpsBacktestCalibrationBaselines(baselineFilter)).toEqual([
      'marketops-backtest-calibration-baselines',
      baselineFilter,
    ]);
    expect(queryKeys.marketOpsBacktestCalibrationBaselines({ ...baselineFilter })).toEqual([
      'marketops-backtest-calibration-baselines',
      baselineFilter,
    ]);
    expect(queryKeys.marketOpsBacktestCalibrationBaseline('btbase-1')).toEqual([
      'marketops-backtest-calibration-baseline',
      'btbase-1',
    ]);
    expect(queryKeys.marketOpsBacktestCalibrationComparisons(comparisonFilter)).toEqual([
      'marketops-backtest-calibration-comparisons',
      comparisonFilter,
    ]);
    expect(queryKeys.marketOpsBacktestCalibrationComparisons({ ...comparisonFilter })).toEqual([
      'marketops-backtest-calibration-comparisons',
      comparisonFilter,
    ]);
    expect(queryKeys.marketOpsBacktestCalibrationComparison('btcmp-1')).toEqual([
      'marketops-backtest-calibration-comparison',
      'btcmp-1',
    ]);
  });

  it('list and detail keys share the prefixes the create mutation invalidates', () => {
    // useCreateMarketOpsBacktest invalidates ['marketops-backtests'] (list) and
    // ['marketops-backtest'] (detail). Assert the hook keys sit under those prefixes.
    const listKey = queryKeys.marketOpsBacktests({ tenant_id: 'tenant-local', limit: 50 });
    const detailKey = queryKeys.marketOpsBacktest('bt-1', 'tenant-local');
    expect(listKey[0]).toBe('marketops-backtests');
    expect(detailKey[0]).toBe('marketops-backtest');
  });
});

describe('applyBacktestCreateResult (G081 mutation invalidation)', () => {
  it('seeds the detail cache and invalidates the list + detail prefixes', () => {
    const queryClient = new QueryClient();
    const setSpy = vi.spyOn(queryClient, 'setQueryData');
    const invSpy = vi.spyOn(queryClient, 'invalidateQueries');

    const created = {
      backtest_run: { run_id: 'bt-new', tenant_id: 'tenant-local', status: 'succeeded' },
      metrics: { scanned: 5 },
    } as never;

    applyBacktestCreateResult(queryClient, created);

    // Detail cache seeded with the returned run (instant selectability).
    const detailKey = queryKeys.marketOpsBacktest('bt-new', 'tenant-local');
    expect(setSpy).toHaveBeenCalledWith(
      detailKey,
      { backtest_run: expect.objectContaining({ run_id: 'bt-new' }) },
    );
    expect(queryClient.getQueryData<{ backtest_run: { run_id: string } }>(detailKey)?.backtest_run.run_id).toBe('bt-new');
    // List + detail prefixes invalidated so the run table + detail refetch.
    expect(invSpy).toHaveBeenCalledWith({ queryKey: ['marketops-backtests'] });
    expect(invSpy).toHaveBeenCalledWith({ queryKey: ['marketops-backtest'] });
  });
});

describe('applyBacktestCalibrationSummaryCreateResult (G082 mutation invalidation)', () => {
  it('seeds the summary detail cache and invalidates summary lists', () => {
    const queryClient = new QueryClient();
    const setSpy = vi.spyOn(queryClient, 'setQueryData');
    const invSpy = vi.spyOn(queryClient, 'invalidateQueries');

    const created = {
      calibration_summary: { summary_id: 'btcal-new', run_count: 2 },
    } as never;

    applyBacktestCalibrationSummaryCreateResult(queryClient, created);

    const detailKey = queryKeys.marketOpsBacktestCalibrationSummary('btcal-new');
    expect(setSpy).toHaveBeenCalledWith(detailKey, created);
    expect(queryClient.getQueryData<{ calibration_summary: { summary_id: string } }>(detailKey)?.calibration_summary.summary_id).toBe('btcal-new');
    expect(invSpy).toHaveBeenCalledWith({ queryKey: ['marketops-backtest-calibration-summaries'] });
  });
});

describe('applyBacktestCalibrationBaselineCreateResult (G083 mutation invalidation)', () => {
  it('seeds the baseline detail cache and invalidates only baseline prefixes', () => {
    const queryClient = new QueryClient();
    const setSpy = vi.spyOn(queryClient, 'setQueryData');
    const invSpy = vi.spyOn(queryClient, 'invalidateQueries');

    const created = {
      calibration_baseline: { baseline_id: 'btbase-new', tenant_id: 'tenant-local', status: 'active' },
    } as never;

    applyBacktestCalibrationBaselineCreateResult(queryClient, created);

    const detailKey = queryKeys.marketOpsBacktestCalibrationBaseline('btbase-new');
    expect(setSpy).toHaveBeenCalledWith(detailKey, created);
    expect(queryClient.getQueryData<{ calibration_baseline: { baseline_id: string } }>(detailKey)?.calibration_baseline.baseline_id).toBe('btbase-new');
    // List + detail prefixes invalidated so the baseline table refetches.
    expect(invSpy).toHaveBeenCalledWith({ queryKey: ['marketops-backtest-calibration-baselines'] });
    expect(invSpy).toHaveBeenCalledWith({ queryKey: ['marketops-backtest-calibration-baseline'] });
    // Production signal / DSM / graph proposal queries are never invalidated.
    expect(invSpy).not.toHaveBeenCalledWith({ queryKey: ['signals'] });
    expect(invSpy).not.toHaveBeenCalledWith({ queryKey: ['marketops-dsm-graph-proposals'] });
    expect(invSpy).not.toHaveBeenCalledWith({ queryKey: ['marketops-backtest-graph-proposals'] });
  });
});

describe('applyBacktestCalibrationComparisonCreateResult (G083 mutation invalidation)', () => {
  it('seeds the comparison detail cache and invalidates only comparison prefixes', () => {
    const queryClient = new QueryClient();
    const setSpy = vi.spyOn(queryClient, 'setQueryData');
    const invSpy = vi.spyOn(queryClient, 'invalidateQueries');

    const created = {
      calibration_comparison: {
        comparison_id: 'btcmp-new',
        baseline_id: 'btbase-1',
        recommendation: 'neutral_candidate',
      },
    } as never;

    applyBacktestCalibrationComparisonCreateResult(queryClient, created);

    const detailKey = queryKeys.marketOpsBacktestCalibrationComparison('btcmp-new');
    expect(setSpy).toHaveBeenCalledWith(detailKey, created);
    expect(queryClient.getQueryData<{ calibration_comparison: { comparison_id: string } }>(detailKey)?.calibration_comparison.comparison_id).toBe('btcmp-new');
    expect(invSpy).toHaveBeenCalledWith({ queryKey: ['marketops-backtest-calibration-comparisons'] });
    expect(invSpy).toHaveBeenCalledWith({ queryKey: ['marketops-backtest-calibration-comparison'] });
    // No production or baseline-list invalidation leaks from a comparison create.
    expect(invSpy).not.toHaveBeenCalledWith({ queryKey: ['marketops-backtest-calibration-baselines'] });
    expect(invSpy).not.toHaveBeenCalledWith({ queryKey: ['marketops-dsm-graph-proposals'] });
  });
});

describe('MarketOps back-test label-aware evaluations API client (G085)', () => {
  it('creates, lists, and fetches an evaluation via the G085 endpoints with bearer + content type', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        jsonResponse(
          {
            backtest_evaluation: {
              evaluation_id: 'bteval-1',
              tenant_id: 'tenant-local',
              run_id: 'bt-1',
              recommendation: 'improvement_candidate',
              recommendation_note: 'automatic recommendations align with available labels',
              candidate_count: 5,
              labeled_count: 5,
              precision: 1,
              recall: 1,
              label_coverage: 1,
              metrics: { matched_samples: [], scoring_notes: ['manual_review_required and supersede_candidate recommendations are not counted as automatic true/false outcomes'] },
            },
          },
          201,
        ),
      )
      .mockResolvedValueOnce(jsonResponse({ backtest_evaluations: [{ evaluation_id: 'bteval-1', run_id: 'bt-1' }] }))
      .mockResolvedValueOnce(jsonResponse({ backtest_evaluation: { evaluation_id: 'bteval-1', recommendation: 'improvement_candidate' } }));
    vi.stubGlobal('fetch', fetchMock);
    state.authEnabled = true;
    state.token = 'jwt-abc';

    const created = await api.createMarketOpsBacktestEvaluation({ tenant_id: 'tenant-local', run_id: 'bt-1' });
    const list = await api.listMarketOpsBacktestEvaluations({
      tenant_id: 'tenant-local',
      run_id: 'bt-1',
      recommendation: 'improvement_candidate',
      limit: 50,
    });
    const detail = await api.getMarketOpsBacktestEvaluation('bteval-1');

    // Create: POST to the evaluations endpoint with bearer + JSON body; only
    // tenant_id + run_id are sent (label_source is derived server-side).
    expect(String(fetchMock.mock.calls[0][0])).toContain('/v1/marketops/backtest-evaluations');
    expect(fetchMock.mock.calls[0][1].method).toBe('POST');
    expect(fetchMock.mock.calls[0][1].headers['Authorization']).toBe('Bearer jwt-abc');
    expect(fetchMock.mock.calls[0][1].headers['Content-Type']).toBe('application/json');
    expect(JSON.parse(fetchMock.mock.calls[0][1].body)).toEqual({ tenant_id: 'tenant-local', run_id: 'bt-1' });
    // List: run-scoped + recommendation filter + default limit.
    const listUrl = String(fetchMock.mock.calls[1][0]);
    expect(listUrl).toContain('tenant_id=tenant-local');
    expect(listUrl).toContain('run_id=bt-1');
    expect(listUrl).toContain('recommendation=improvement_candidate');
    expect(listUrl).toContain('limit=50');
    // Detail: encoded evaluation id.
    expect(String(fetchMock.mock.calls[2][0])).toContain('/v1/marketops/backtest-evaluations/bteval-1');
    // Envelopes parse, including nested metrics.matched_samples + scoring_notes.
    expect(created.backtest_evaluation.recommendation).toBe('improvement_candidate');
    expect(created.backtest_evaluation.metrics.matched_samples).toEqual([]);
    expect(created.backtest_evaluation.metrics.scoring_notes).toHaveLength(1);
    expect(list.backtest_evaluations[0].run_id).toBe('bt-1');
    expect(detail.backtest_evaluation.evaluation_id).toBe('bteval-1');
  });

  it('omits label_source + requested_by + evaluation_id from the create body', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ backtest_evaluation: { evaluation_id: 'bteval-2' } }, 201));
    vi.stubGlobal('fetch', fetchMock);

    await api.createMarketOpsBacktestEvaluation({ tenant_id: 'tenant-local', run_id: 'bt-1' });

    const body = JSON.parse(fetchMock.mock.calls[0][1].body);
    expect(body.tenant_id).toBe('tenant-local');
    expect(body.run_id).toBe('bt-1');
    expect(body.label_source).toBeUndefined();
    expect(body.requested_by).toBeUndefined();
    expect(body.evaluation_id).toBeUndefined();
  });

  it('encodes special characters in the evaluation id on the detail path', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ backtest_evaluation: {} }));
    vi.stubGlobal('fetch', fetchMock);

    await api.getMarketOpsBacktestEvaluation('bteval/a b');

    expect(String(fetchMock.mock.calls[0][0])).toContain('/v1/marketops/backtest-evaluations/bteval%2Fa%20b');
  });
});

describe('MarketOps back-test evaluation query keys (G085)', () => {
  it('uses stable evaluation list/detail keys', () => {
    const filter = { tenant_id: 'tenant-local', run_id: 'bt-1', limit: 50 };

    expect(queryKeys.marketOpsBacktestEvaluations(filter)).toEqual(['marketops-backtest-evaluations', filter]);
    expect(queryKeys.marketOpsBacktestEvaluations({ ...filter })).toEqual(['marketops-backtest-evaluations', filter]);
    expect(queryKeys.marketOpsBacktestEvaluation('bteval-1')).toEqual(['marketops-backtest-evaluation', 'bteval-1']);
  });

  it('list and detail keys share the prefixes the create mutation invalidates', () => {
    const listKey = queryKeys.marketOpsBacktestEvaluations({ tenant_id: 'tenant-local', run_id: 'bt-1', limit: 50 });
    const detailKey = queryKeys.marketOpsBacktestEvaluation('bteval-1');
    expect(listKey[0]).toBe('marketops-backtest-evaluations');
    expect(detailKey[0]).toBe('marketops-backtest-evaluation');
  });
});

describe('applyBacktestEvaluationCreateResult (G085 mutation invalidation)', () => {
  it('seeds the evaluation detail cache and invalidates only evaluation prefixes', () => {
    const queryClient = new QueryClient();
    const setSpy = vi.spyOn(queryClient, 'setQueryData');
    const invSpy = vi.spyOn(queryClient, 'invalidateQueries');

    const created = {
      backtest_evaluation: {
        evaluation_id: 'bteval-new',
        tenant_id: 'tenant-local',
        run_id: 'bt-1',
        recommendation: 'improvement_candidate',
      },
    } as never;

    applyBacktestEvaluationCreateResult(queryClient, created);

    const detailKey = queryKeys.marketOpsBacktestEvaluation('bteval-new');
    expect(setSpy).toHaveBeenCalledWith(detailKey, created);
    expect(queryClient.getQueryData<{ backtest_evaluation: { evaluation_id: string } }>(detailKey)?.backtest_evaluation.evaluation_id).toBe('bteval-new');
    // List + detail prefixes invalidated so the evaluations table refetches.
    expect(invSpy).toHaveBeenCalledWith({ queryKey: ['marketops-backtest-evaluations'] });
    expect(invSpy).toHaveBeenCalledWith({ queryKey: ['marketops-backtest-evaluation'] });
    // Production signal / DSM graph proposal / back-test run + graph-proposal
    // queries are never invalidated by an evaluation create.
    expect(invSpy).not.toHaveBeenCalledWith({ queryKey: ['signals'] });
    expect(invSpy).not.toHaveBeenCalledWith({ queryKey: ['marketops-dsm-graph-proposals'] });
    expect(invSpy).not.toHaveBeenCalledWith({ queryKey: ['marketops-backtests'] });
    expect(invSpy).not.toHaveBeenCalledWith({ queryKey: ['marketops-backtest-graph-proposals'] });
  });
});

describe('MarketOps back-test promotion candidates API client (G086)', () => {
  it('creates, lists, fetches, and decides a candidate via the G086 endpoints', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        jsonResponse(
          {
            promotion_candidate: {
              candidate_id: 'btpromo-1',
              tenant_id: 'tenant-local',
              baseline_id: 'btbase-1',
              comparison_id: 'btcmp-1',
              evaluation_id: 'bteval-1',
              run_id: 'bt-1',
              readiness_status: 'ready_for_review',
              readiness_reasons: ['comparison and evaluation evidence meet review thresholds'],
              status: 'proposed',
              evidence: { comparison: { recommendation: 'neutral_candidate' }, readiness: { status: 'ready_for_review' } },
            },
          },
          201,
        ),
      )
      .mockResolvedValueOnce(jsonResponse({ promotion_candidates: [{ candidate_id: 'btpromo-1' }] }))
      .mockResolvedValueOnce(jsonResponse({ promotion_candidate: { candidate_id: 'btpromo-1' } }))
      .mockResolvedValueOnce(jsonResponse({ promotion_candidate: { candidate_id: 'btpromo-1', status: 'deferred', decision_note: 'hold' } }));
    vi.stubGlobal('fetch', fetchMock);
    state.authEnabled = true;
    state.token = 'jwt-abc';

    const created = await api.createMarketOpsBacktestPromotionCandidate({
      tenant_id: 'tenant-local',
      baseline_id: 'btbase-1',
      comparison_id: 'btcmp-1',
      evaluation_id: 'bteval-1',
      candidate_version: 'taxonomy-v1-policy-v1-review-20260712',
    });
    const list = await api.listMarketOpsBacktestPromotionCandidates({
      tenant_id: 'tenant-local',
      baseline_id: 'btbase-1',
      status: 'proposed',
      readiness_status: 'ready_for_review',
      limit: 50,
    });
    const detail = await api.getMarketOpsBacktestPromotionCandidate('btpromo-1');
    const decided = await api.decideMarketOpsBacktestPromotionCandidate('btpromo-1', {
      status: 'deferred',
      decision_note: 'hold',
    });

    // Create: POST with bearer + JSON body; only the request fields are sent.
    expect(String(fetchMock.mock.calls[0][0])).toContain('/v1/marketops/backtest-promotion-candidates');
    expect(fetchMock.mock.calls[0][1].method).toBe('POST');
    expect(fetchMock.mock.calls[0][1].headers['Authorization']).toBe('Bearer jwt-abc');
    expect(fetchMock.mock.calls[0][1].headers['Content-Type']).toBe('application/json');
    expect(JSON.parse(fetchMock.mock.calls[0][1].body)).toEqual({
      tenant_id: 'tenant-local',
      baseline_id: 'btbase-1',
      comparison_id: 'btcmp-1',
      evaluation_id: 'bteval-1',
      candidate_version: 'taxonomy-v1-policy-v1-review-20260712',
    });
    // List: baseline + status + readiness filters + default limit.
    const listUrl = String(fetchMock.mock.calls[1][0]);
    expect(listUrl).toContain('tenant_id=tenant-local');
    expect(listUrl).toContain('baseline_id=btbase-1');
    expect(listUrl).toContain('status=proposed');
    expect(listUrl).toContain('readiness_status=ready_for_review');
    expect(listUrl).toContain('limit=50');
    // Detail: encoded candidate id.
    expect(String(fetchMock.mock.calls[2][0])).toContain('/v1/marketops/backtest-promotion-candidates/btpromo-1');
    // Decision: POST to /{candidate_id}/decision with status + note.
    expect(String(fetchMock.mock.calls[3][0])).toContain('/v1/marketops/backtest-promotion-candidates/btpromo-1/decision');
    expect(fetchMock.mock.calls[3][1].method).toBe('POST');
    expect(JSON.parse(fetchMock.mock.calls[3][1].body)).toEqual({ status: 'deferred', decision_note: 'hold' });
    // Envelopes parse, including nested evidence + readiness_reasons.
    expect(created.promotion_candidate.readiness_status).toBe('ready_for_review');
    expect(created.promotion_candidate.readiness_reasons).toHaveLength(1);
    expect(created.promotion_candidate.evidence.comparison?.recommendation).toBe('neutral_candidate');
    expect(list.promotion_candidates[0].candidate_id).toBe('btpromo-1');
    expect(detail.promotion_candidate.candidate_id).toBe('btpromo-1');
    expect(decided.promotion_candidate.status).toBe('deferred');
  });

  it('omits optional fields from the create body and the decision body', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ promotion_candidate: { candidate_id: 'btpromo-2' } }, 201))
      .mockResolvedValueOnce(jsonResponse({ promotion_candidate: { candidate_id: 'btpromo-2', status: 'rejected' } }));
    vi.stubGlobal('fetch', fetchMock);

    await api.createMarketOpsBacktestPromotionCandidate({
      tenant_id: 'tenant-local',
      baseline_id: 'btbase-1',
      comparison_id: 'btcmp-1',
    });
    await api.decideMarketOpsBacktestPromotionCandidate('btpromo-2', { status: 'rejected' });

    const createBody = JSON.parse(fetchMock.mock.calls[0][1].body);
    expect(createBody).toEqual({ tenant_id: 'tenant-local', baseline_id: 'btbase-1', comparison_id: 'btcmp-1' });
    expect(createBody.evaluation_id).toBeUndefined();
    expect(createBody.candidate_version).toBeUndefined();
    const decisionBody = JSON.parse(fetchMock.mock.calls[1][1].body);
    expect(decisionBody).toEqual({ status: 'rejected' });
    expect(decisionBody.decision_note).toBeUndefined();
  });

  it('encodes special characters in the candidate id on the detail + decision paths', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ promotion_candidate: {} }))
      .mockResolvedValueOnce(jsonResponse({ promotion_candidate: {} }));
    vi.stubGlobal('fetch', fetchMock);

    await api.getMarketOpsBacktestPromotionCandidate('btpromo/a b');
    await api.decideMarketOpsBacktestPromotionCandidate('btpromo/a b', { status: 'deferred' });

    expect(String(fetchMock.mock.calls[0][0])).toContain('/v1/marketops/backtest-promotion-candidates/btpromo%2Fa%20b');
    expect(String(fetchMock.mock.calls[1][0])).toContain('/v1/marketops/backtest-promotion-candidates/btpromo%2Fa%20b/decision');
  });
});

describe('MarketOps back-test promotion candidate query keys (G086)', () => {
  it('uses stable promotion candidate list/detail keys', () => {
    const filter = { tenant_id: 'tenant-local', baseline_id: 'btbase-1', status: 'proposed' as const, limit: 50 };

    expect(queryKeys.marketOpsBacktestPromotionCandidates(filter)).toEqual([
      'marketops-backtest-promotion-candidates',
      filter,
    ]);
    expect(queryKeys.marketOpsBacktestPromotionCandidates({ ...filter })).toEqual([
      'marketops-backtest-promotion-candidates',
      filter,
    ]);
    expect(queryKeys.marketOpsBacktestPromotionCandidate('btpromo-1')).toEqual([
      'marketops-backtest-promotion-candidate',
      'btpromo-1',
    ]);
  });

  it('list and detail keys share the prefixes the create + decision mutations invalidate', () => {
    const listKey = queryKeys.marketOpsBacktestPromotionCandidates({ tenant_id: 'tenant-local', limit: 50 });
    const detailKey = queryKeys.marketOpsBacktestPromotionCandidate('btpromo-1');
    expect(listKey[0]).toBe('marketops-backtest-promotion-candidates');
    expect(detailKey[0]).toBe('marketops-backtest-promotion-candidate');
  });
});

describe('applyBacktestPromotionCandidateCreateResult (G086 mutation invalidation)', () => {
  it('seeds the candidate detail cache and invalidates only promotion prefixes', () => {
    const queryClient = new QueryClient();
    const setSpy = vi.spyOn(queryClient, 'setQueryData');
    const invSpy = vi.spyOn(queryClient, 'invalidateQueries');

    const created = {
      promotion_candidate: { candidate_id: 'btpromo-new', status: 'proposed', readiness_status: 'ready_for_review' },
    } as never;

    applyBacktestPromotionCandidateCreateResult(queryClient, created);

    const detailKey = queryKeys.marketOpsBacktestPromotionCandidate('btpromo-new');
    expect(setSpy).toHaveBeenCalledWith(detailKey, created);
    expect(queryClient.getQueryData<{ promotion_candidate: { candidate_id: string } }>(detailKey)?.promotion_candidate.candidate_id).toBe('btpromo-new');
    expect(invSpy).toHaveBeenCalledWith({ queryKey: ['marketops-backtest-promotion-candidates'] });
    expect(invSpy).toHaveBeenCalledWith({ queryKey: ['marketops-backtest-promotion-candidate'] });
    // No production or sibling evidence queries are invalidated by a create.
    expect(invSpy).not.toHaveBeenCalledWith({ queryKey: ['signals'] });
    expect(invSpy).not.toHaveBeenCalledWith({ queryKey: ['marketops-dsm-graph-proposals'] });
    expect(invSpy).not.toHaveBeenCalledWith({ queryKey: ['marketops-backtest-evaluations'] });
    expect(invSpy).not.toHaveBeenCalledWith({ queryKey: ['marketops-backtest-calibration-baselines'] });
  });
});

describe('applyBacktestPromotionCandidateDecisionResult (G086 mutation invalidation)', () => {
  it('seeds the decided candidate detail cache and invalidates only promotion prefixes', () => {
    const queryClient = new QueryClient();
    const setSpy = vi.spyOn(queryClient, 'setQueryData');
    const invSpy = vi.spyOn(queryClient, 'invalidateQueries');

    const decided = {
      promotion_candidate: { candidate_id: 'btpromo-1', status: 'approved_for_promotion', decision_note: 'planning only' },
    } as never;

    applyBacktestPromotionCandidateDecisionResult(queryClient, decided);

    const detailKey = queryKeys.marketOpsBacktestPromotionCandidate('btpromo-1');
    expect(setSpy).toHaveBeenCalledWith(detailKey, decided);
    expect(queryClient.getQueryData<{ promotion_candidate: { status: string } }>(detailKey)?.promotion_candidate.status).toBe('approved_for_promotion');
    expect(invSpy).toHaveBeenCalledWith({ queryKey: ['marketops-backtest-promotion-candidates'] });
    expect(invSpy).toHaveBeenCalledWith({ queryKey: ['marketops-backtest-promotion-candidate'] });
    // A decision never touches production deploy/graph/signal queries.
    expect(invSpy).not.toHaveBeenCalledWith({ queryKey: ['signals'] });
    expect(invSpy).not.toHaveBeenCalledWith({ queryKey: ['marketops-dsm-graph-proposals'] });
    expect(invSpy).not.toHaveBeenCalledWith({ queryKey: ['marketops-backtest-graph-proposals'] });
  });
});
