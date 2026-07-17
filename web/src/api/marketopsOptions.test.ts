import { afterEach, describe, expect, it, vi } from 'vitest';

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
const { queryKeys } = await import('./queries');

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

describe('MarketOps asset options API client (G128)', () => {
  it('builds the coverage path with tenant + symbol and bearer', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ options_coverage: {} }));
    vi.stubGlobal('fetch', fetchMock);

    await api.getMarketOpsOptionsCoverage('tenant-local', 'NVDA');

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/tenants/tenant-local/marketops/assets/NVDA/options/coverage');
    expect(fetchMock.mock.calls[0][1].headers['Authorization']).toBe('Bearer jwt-abc');
  });

  it('builds the distribution path with default window + limit', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ options_distributions: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listMarketOpsOptionsDistributions('tenant-local', 'NVDA', {});

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/tenants/tenant-local/marketops/assets/NVDA/options/distribution');
    expect(url).toContain('window=10_trade_days');
    expect(url).toContain('limit=10');
    expect(fetchMock.mock.calls[0][1].headers['Authorization']).toBe('Bearer jwt-abc');
  });

  it('forwards a custom distribution window + limit', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ options_distributions: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listMarketOpsOptionsDistributions('tenant-local', 'NVDA', { window: '20_trade_days', limit: 5 });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('window=20_trade_days');
    expect(url).toContain('limit=5');
  });

  it('builds the chain path with trade_date + contract_type + default limit', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ options_chain: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listMarketOpsOptionsChain('tenant-local', 'NVDA', { trade_date: '2026-07-17', contract_type: 'call' });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/tenants/tenant-local/marketops/assets/NVDA/options/chain');
    expect(url).toContain('trade_date=2026-07-17');
    expect(url).toContain('contract_type=call');
    expect(url).toContain('limit=500');
  });

  it('omits unset chain filters but still sends the default limit', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ options_chain: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listMarketOpsOptionsChain('tenant-local', 'NVDA', {});

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('limit=500');
    expect(url).not.toContain('trade_date=');
    expect(url).not.toContain('contract_type=');
  });

  it('encodes the symbol path segment', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ options_coverage: {} }));
    vi.stubGlobal('fetch', fetchMock);

    await api.getMarketOpsOptionsCoverage('tenant-local', 'a/b');

    expect(String(fetchMock.mock.calls[0][0])).toContain('/marketops/assets/a%2Fb/options/coverage');
  });

  it('parses coverage, distribution, and chain envelopes', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ options_coverage: { symbol: 'NVDA', trade_day_count: 10, contract_count: 250 } }))
      .mockResolvedValueOnce(
        jsonResponse({
          options_distributions: [
            { trade_date: '2026-07-17T00:00:00Z', window_name: '10_trade_days', call_put_open_interest_ratio: 0.1 },
          ],
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          options_chain: [{ option_ticker: 'O:NVDA260717C00170000', contract_type: 'call', open_interest: 1543 }],
        }),
      );
    vi.stubGlobal('fetch', fetchMock);

    const cov = await api.getMarketOpsOptionsCoverage('tenant-local', 'NVDA');
    const dist = await api.listMarketOpsOptionsDistributions('tenant-local', 'NVDA', {});
    const chain = await api.listMarketOpsOptionsChain('tenant-local', 'NVDA', { trade_date: '2026-07-17' });

    expect(cov.options_coverage.symbol).toBe('NVDA');
    expect(cov.options_coverage.contract_count).toBe(250);
    expect(dist.options_distributions[0].call_put_open_interest_ratio).toBeCloseTo(0.1);
    expect(chain.options_chain[0].option_ticker).toBe('O:NVDA260717C00170000');
    expect(chain.options_chain[0].open_interest).toBe(1543);
  });
});

describe('MarketOps asset options query keys (G128)', () => {
  it('uses stable coverage/distribution/chain keys', () => {
    expect(queryKeys.marketOpsOptionsCoverage('tenant-local', 'NVDA')).toEqual([
      'marketops-options-coverage',
      'tenant-local',
      'NVDA',
    ]);
    expect(queryKeys.marketOpsOptionsDistributions('tenant-local', 'NVDA', { window: '10_trade_days', limit: 10 })).toEqual([
      'marketops-options-distributions',
      'tenant-local',
      'NVDA',
      { window: '10_trade_days', limit: 10 },
    ]);
    expect(queryKeys.marketOpsOptionsChain('tenant-local', 'NVDA', { trade_date: '2026-07-17' })).toEqual([
      'marketops-options-chain',
      'tenant-local',
      'NVDA',
      { trade_date: '2026-07-17' },
    ]);
  });
});
