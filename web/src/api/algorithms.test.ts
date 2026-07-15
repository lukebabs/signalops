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

const { api } = await import('./client');

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

describe('algorithm API client (G109)', () => {
  it('builds the definitions list path with filters + tenant + bearer + default limit', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ algorithm_definitions: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listAlgorithmDefinitions({
      tenant_id: 'tenant-local',
      algorithm_type: 'anomaly',
      runtime_type: 'python_plugin',
      status: 'active',
    });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/algorithms/definitions');
    expect(url).toContain('tenant_id=tenant-local');
    expect(url).toContain('algorithm_type=anomaly');
    expect(url).toContain('runtime_type=python_plugin');
    expect(url).toContain('status=active');
    expect(url).toContain('limit=50');
    expect(fetchMock.mock.calls[0][1].headers['Authorization']).toBe('Bearer jwt-abc');
  });

  it('omits unset definition filters and defaults tenant', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ algorithm_definitions: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listAlgorithmDefinitions({});

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('tenant_id=tenant-local');
    expect(url).not.toContain('algorithm_type=');
    expect(url).not.toContain('status=');
  });

  it('builds the execution-requests list path scoped by algorithm_id', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ algorithm_execution_requests: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listAlgorithmExecutionRequests({
      tenant_id: 'tenant-local',
      algorithm_id: 'zscore_v1',
      status: 'succeeded',
      correlation_id: 'corr_1',
    });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/algorithms/execution-requests');
    expect(url).toContain('algorithm_id=zscore_v1');
    expect(url).toContain('status=succeeded');
    expect(url).toContain('correlation_id=corr_1');
  });

  it('builds the execution summary path with tenant + limit=10', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ algorithm_execution_summary: {} }));
    vi.stubGlobal('fetch', fetchMock);

    await api.getAlgorithmExecutionSummary('algexec_1', 'tenant-local');

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/algorithms/execution-requests/algexec_1/summary');
    expect(url).toContain('tenant_id=tenant-local');
    expect(url).toContain('limit=10');
  });

  it('URL-encodes the result id on the result detail path', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ algorithm_result: {} }));
    vi.stubGlobal('fetch', fetchMock);

    await api.getAlgorithmResult('algres/a b');

    expect(String(fetchMock.mock.calls[0][0])).toContain('/v1/algorithms/results/algres%2Fa%20b');
  });

  it('parses definitions list and execution summary envelopes', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        jsonResponse({
          algorithm_definitions: [{ algorithm_id: 'zscore_v1', status: 'active', input_features: ['ret'] }],
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          algorithm_execution_summary: {
            execution_request: { execution_request_id: 'algexec_1', status: 'succeeded' },
            result_count: 2,
            severity_counts: { high: 2 },
            max_score: 2.5,
            max_confidence: 0.8,
            top_results: [{ algorithm_result_id: 'algres_1', score: 2.5, severity: 'high' }],
          },
        }),
      );
    vi.stubGlobal('fetch', fetchMock);

    const defs = await api.listAlgorithmDefinitions({});
    const summary = await api.getAlgorithmExecutionSummary('algexec_1');

    expect(defs.algorithm_definitions[0].algorithm_id).toBe('zscore_v1');
    expect(summary.algorithm_execution_summary.result_count).toBe(2);
    expect(summary.algorithm_execution_summary.top_results[0].algorithm_result_id).toBe('algres_1');
    expect(summary.algorithm_execution_summary.severity_counts.high).toBe(2);
  });
});
