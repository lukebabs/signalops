import { afterEach, describe, expect, it, vi } from 'vitest';

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
vi.mock('../auth/session', () => ({ getAccessToken: () => state.token }));

const { api } = await import('./client');
const { queryKeys } = await import('./queries');

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  state.token = 'jwt-abc';
  state.authEnabled = true;
});

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
}

describe('MarketOps intelligence readiness API client (G148-C)', () => {
  it('issues ONE aggregate request with encoded filters + bearer (no per-symbol fan-out)', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ readiness: { aggregate: {}, symbols: [] } }));
    vi.stubGlobal('fetch', fetchMock);

    await api.getMarketOpsIntelligenceReadiness({
      tenant_id: 'tenant-1',
      universe_group: 'top50_megacap',
      symbols: 'AAPL,MSFT',
      latest_session_date: '2026-07-18',
      rollout_status: 'blocked',
      limit: 25,
    });

    // A single call makes exactly one request — proving no per-symbol queries.
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/marketops/intelligence/readiness');
    expect(url).toContain('tenant_id=tenant-1');
    expect(url).toContain('universe_group=top50_megacap');
    expect(url).toContain('symbols=AAPL%2CMSFT');
    expect(url).toContain('latest_session_date=2026-07-18');
    expect(url).toContain('rollout_status=blocked');
    expect(url).toContain('limit=25');
    expect(fetchMock.mock.calls[0][1].headers['Authorization']).toBe('Bearer jwt-abc');
  });

  it('defaults tenant + limit and omits unset filters', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ readiness: { aggregate: {}, symbols: [] } }));
    vi.stubGlobal('fetch', fetchMock);

    await api.getMarketOpsIntelligenceReadiness({});

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('tenant_id=tenant-local');
    expect(url).toContain('limit=50');
    expect(url).not.toContain('symbols=');
    expect(url).not.toContain('rollout_status=');
    expect(url).not.toContain('latest_session_date=');
  });

  it('parses the readiness envelope', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({
        readiness: {
          aggregate: {
            symbol_count: 2,
            production_ready_supported: false,
            latest_session_date: '2026-07-18',
            dimension_counts: { rollout_status: { blocked: 1, not_observed: 1 } },
          },
          symbols: [
            { result_id: 'r1', symbol: 'AAPL', latest_market_state_id: 'ms-1', rollout_status: 'blocked', calibration_below_minimum: true },
            { result_id: 'r2', symbol: 'MSFT', latest_market_state_id: '', rollout_status: 'not_observed' },
          ],
        },
      }),
    );
    vi.stubGlobal('fetch', fetchMock);

    const r = await api.getMarketOpsIntelligenceReadiness({ tenant_id: 'tenant-1' });
    expect(r.readiness.aggregate.symbol_count).toBe(2);
    expect(r.readiness.aggregate.production_ready_supported).toBe(false);
    expect(r.readiness.symbols).toHaveLength(2);
    expect(r.readiness.symbols[0].rollout_status).toBe('blocked');
    expect(r.readiness.symbols[1].latest_market_state_id).toBe('');
  });
});

describe('MarketOps intelligence readiness query key (G148-C)', () => {
  it('uses a stable readiness key derived from the filter', () => {
    expect(queryKeys.marketOpsIntelligenceReadiness({ tenant_id: 't', limit: 50 })).toEqual([
      'marketops-intelligence-readiness',
      { tenant_id: 't', limit: 50 },
    ]);
  });
});
