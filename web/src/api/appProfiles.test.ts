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

const PROFILES_PAYLOAD = {
  app_profiles: [
    {
      app_id: 'console',
      label: 'SignalOps Console',
      default_route: '/dashboard',
      domains: ['market_data', 'crm', 'security', 'operations', 'iot', 'procurement', 'custom'],
      enabled_modules: ['dashboard', 'event_explorer', 'timeline'],
      dashboard_profile: 'console.default',
    },
    {
      app_id: 'marketops',
      label: 'MarketOps',
      default_route: '/marketops/dashboard',
      domains: ['market_data'],
      enabled_modules: ['dashboard', 'symbols', 'signals', 'alerts'],
      dashboard_profile: 'marketdata.default',
    },
  ],
};

describe('app profiles API client (G066/G067)', () => {
  it('getAppProfiles attaches the Bearer token and parses app_profiles', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(PROFILES_PAYLOAD));
    vi.stubGlobal('fetch', fetchMock);
    state.authEnabled = true;
    state.token = 'jwt-abc';

    const data = await api.getAppProfiles();

    const [url, options] = fetchMock.mock.calls[0];
    expect(String(url).endsWith('/v1/app-profiles')).toBe(true);
    expect(String(url)).not.toContain('?'); // no query params
    expect(options.headers['Authorization']).toBe('Bearer jwt-abc');
    expect(data.app_profiles).toHaveLength(2);
    expect(data.app_profiles.map((p) => p.app_id)).toEqual(['console', 'marketops']);
  });

  it('getAppProfiles omits Authorization when auth is disabled', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(PROFILES_PAYLOAD));
    vi.stubGlobal('fetch', fetchMock);
    state.authEnabled = false;
    state.token = 'jwt-abc';

    await api.getAppProfiles();

    const [, options] = fetchMock.mock.calls[0];
    expect(options.headers['Authorization']).toBeUndefined();
  });

  it('uses a stable app-profiles query key', () => {
    expect(queryKeys.appProfiles).toEqual(['app-profiles']);
  });
});

describe('G066 metadata filter params on list APIs', () => {
  // All five G066-aware list filters accept optional app_id/domain/use_case.
  it.each([
    ['listRawEvents', (f: never) => api.listRawEvents(f)],
    ['listNormalizedEvents', (f: never) => api.listNormalizedEvents(f)],
    ['listSignals', (f: never) => api.listSignals(f)],
    ['listAlerts', (f: never) => api.listAlerts(f)],
    ['listInsights', (f: never) => api.listInsights(f)],
  ])('%s forwards app_id, domain, and use_case when present', async (_name, call) => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({}));
    vi.stubGlobal('fetch', fetchMock);

    await call({
      app_id: 'marketops',
      domain: 'market_data',
      use_case: 'daily_market_surveillance',
    } as never);

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('app_id=marketops');
    expect(url).toContain('domain=market_data');
    expect(url).toContain('use_case=daily_market_surveillance');
  });

  it('omits app_id/domain/use_case when not provided', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ raw_events: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listRawEvents({});

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).not.toContain('app_id=');
    expect(url).not.toContain('domain=');
    expect(url).not.toContain('use_case=');
  });
});
