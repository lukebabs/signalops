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

describe('alert/insight API client (G048)', () => {
  it('listAlerts maps filters to query params and defaults limit to 50', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ alerts: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listAlerts({ tenant_id: 'tenant-local', status: 'open' });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url.startsWith('http://localhost:5173/v1/alerts?')).toBe(true);
    expect(url).toContain('tenant_id=tenant-local');
    expect(url).toContain('status=open');
    expect(url).toContain('limit=50'); // default fallback
    // omitted optional filters are not serialized
    expect(url).not.toContain('severity=');
    expect(url).not.toContain('source_id=');
  });

  it('listAlerts forwards severity/source/dataset when provided', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ alerts: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listAlerts({
      tenant_id: 'tenant-local',
      source_id: 'src-1',
      dataset: 'sensor_observations',
      severity: 'high',
      limit: 25,
    });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('source_id=src-1');
    expect(url).toContain('dataset=sensor_observations');
    expect(url).toContain('severity=high');
    expect(url).toContain('limit=25');
  });

  it('listInsights maps insight_type and status filters', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ insights: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listInsights({ tenant_id: 'tenant-local', insight_type: 'temperature.anomaly', status: 'active' });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url.startsWith('http://localhost:5173/v1/insights?')).toBe(true);
    expect(url).toContain('insight_type=temperature.anomaly');
    expect(url).toContain('status=active');
    expect(url).toContain('limit=50');
  });

  it('getAlert encodes the alert id path segment', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi
      .fn()
      .mockResolvedValue(jsonResponse({ alert: { alert_id: 'alert:signal-1' } }));
    vi.stubGlobal('fetch', fetchMock);

    await api.getAlert('alert:signal-1');

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/alerts/alert%3Asignal-1');
  });

  it('getInsight encodes the insight id path segment', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi
      .fn()
      .mockResolvedValue(jsonResponse({ insight: { insight_id: 'insight:signal-1' } }));
    vi.stubGlobal('fetch', fetchMock);

    await api.getInsight('insight:signal-1');

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/insights/insight%3Asignal-1');
  });
});
