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

describe('MarketOps assets API client (G071)', () => {
  it('builds the tenant-scoped path with default query params', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ assets: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listMarketOpsAssets({ tenant_id: 'tenant-local' });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/tenants/tenant-local/marketops/assets');
    expect(url).toContain('universe_group=top50_megacap');
    expect(url).toContain('active_only=true');
    expect(url).toContain('limit=50');
  });

  it('defaults tenant_id to tenant-local when omitted', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ assets: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listMarketOpsAssets({});

    expect(String(fetchMock.mock.calls[0][0])).toContain('/v1/tenants/tenant-local/marketops/assets');
  });

  it('sends active_only=false only when explicitly disabled', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ assets: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listMarketOpsAssets({ active_only: false });

    expect(String(fetchMock.mock.calls[0][0])).toContain('active_only=false');
  });

  it('encodes the tenant path segment', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ assets: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listMarketOpsAssets({ tenant_id: 'acme:ops' });

    expect(String(fetchMock.mock.calls[0][0])).toContain('/v1/tenants/acme%3Aops/marketops/assets');
  });

  it('attaches the bearer token', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ assets: [] }));
    vi.stubGlobal('fetch', fetchMock);
    state.authEnabled = true;
    state.token = 'jwt-abc';

    await api.listMarketOpsAssets({ tenant_id: 'tenant-local' });

    expect(fetchMock.mock.calls[0][1].headers['Authorization']).toBe('Bearer jwt-abc');
  });

  it('parses the assets envelope', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi
      .fn()
      .mockResolvedValue(jsonResponse({ assets: [{ ticker: 'NVDA', rank: 1, is_active: true }] }));
    vi.stubGlobal('fetch', fetchMock);

    const data = await api.listMarketOpsAssets({});

    expect(data.assets).toHaveLength(1);
    expect(data.assets[0].ticker).toBe('NVDA');
  });
});

describe('MarketOps assets query key (G071)', () => {
  it('uses a stable marketops-assets key derived from the filter', () => {
    const filter = { tenant_id: 'tenant-local', universe_group: 'top50_megacap', active_only: true, limit: 50 };
    const a = queryKeys.marketOpsAssets(filter);
    const b = queryKeys.marketOpsAssets({ ...filter });
    expect(a).toEqual(['marketops-assets', filter]);
    expect(a).toEqual(b);
  });
});
