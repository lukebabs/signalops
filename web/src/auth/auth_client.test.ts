import { afterEach, describe, expect, it, vi } from 'vitest';

// Hoisted mutable auth state so the mocked modules can read the live values.
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
const { api } = await import('./../api/client');

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

describe('api client auth behavior (G053)', () => {
  it('attaches Authorization: Bearer to /v1/* when auth enabled + token present', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ alerts: [] }));
    vi.stubGlobal('fetch', fetchMock);
    state.authEnabled = true;
    state.token = 'jwt-abc';

    await api.listAlerts({ tenant_id: 'tenant-local' });

    const [, options] = fetchMock.mock.calls[0];
    expect(options.headers['Authorization']).toBe('Bearer jwt-abc');
  });

  it('omits Authorization when auth enabled but no token (health stays usable)', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ status: 'ok' }));
    vi.stubGlobal('fetch', fetchMock);
    state.token = null;

    await api.healthz();

    const [, options] = fetchMock.mock.calls[0];
    expect(options.headers['Authorization']).toBeUndefined();
  });

  it('omits Authorization when auth is disabled', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ alerts: [] }));
    vi.stubGlobal('fetch', fetchMock);
    state.authEnabled = false;
    state.token = 'jwt-abc';

    await api.listAlerts({ tenant_id: 'tenant-local' });

    const [, options] = fetchMock.mock.calls[0];
    expect(options.headers['Authorization']).toBeUndefined();
  });

  it('drops X-SignalOps-Actor and sends Authorization on lifecycle mutation when auth enabled', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ alert: { alert_id: 'a' } }));
    vi.stubGlobal('fetch', fetchMock);
    state.authEnabled = true;
    state.token = 'jwt-abc';

    await api.mutateAlertLifecycle({ alertId: 'alert:1', action: 'acknowledge' });

    const [url, options] = fetchMock.mock.calls[0];
    expect(String(url)).toContain('/v1/alerts/alert%3A1/acknowledge');
    expect(options.headers['Authorization']).toBe('Bearer jwt-abc');
    expect(options.headers['X-SignalOps-Actor']).toBeUndefined();
  });

  it('keeps the operator-local placeholder header when auth is disabled', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ alert: { alert_id: 'a' } }));
    vi.stubGlobal('fetch', fetchMock);
    state.authEnabled = false;

    await api.mutateAlertLifecycle({ alertId: 'alert:1', action: 'acknowledge' });

    const [, options] = fetchMock.mock.calls[0];
    expect(options.headers['X-SignalOps-Actor']).toBe('operator-local');
    expect(options.headers['Authorization']).toBeUndefined();
  });

  it('maps a 401 gateway error envelope to an ApiError with status/code/message', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi
      .fn()
      .mockResolvedValue(jsonResponse({ error: 'unauthorized', message: 'missing or invalid token' }, 401));
    vi.stubGlobal('fetch', fetchMock);

    await expect(api.listAlerts({ tenant_id: 'tenant-local' })).rejects.toMatchObject({
      status: 401,
      code: 'unauthorized',
      message: 'missing or invalid token',
    });
  });
});
