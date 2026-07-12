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
const { queryKeys, applyBacktestCreateResult } = await import('./queries');

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
