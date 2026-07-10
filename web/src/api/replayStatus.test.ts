import { afterEach, describe, expect, it, vi } from 'vitest';
import { api } from './client';

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

describe('replay status API client (G065)', () => {
  it('getReplayStatus builds GET /v1/replay/status with tenant_id and limit', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ replay_status: { workers: [] } }));
    vi.stubGlobal('fetch', fetchMock);

    await api.getReplayStatus({ tenant_id: 'tenant-local', limit: 5 });

    const [url, options] = fetchMock.mock.calls[0];
    const s = String(url);
    expect(s.startsWith('http://localhost:5173/v1/replay/status?')).toBe(true);
    expect(s).toContain('tenant_id=tenant-local');
    expect(s).toContain('limit=5');
    // The get() helper uses fetch's default GET (no explicit method).
    expect(options.method ?? 'GET').toBe('GET');
  });

  it('getReplayStatus omits empty params', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ replay_status: { workers: [] } }));
    vi.stubGlobal('fetch', fetchMock);

    await api.getReplayStatus();

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/replay/status');
    expect(url).not.toContain('tenant_id=');
    expect(url).not.toContain('limit=');
  });
});
