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

describe('replay cancel API client (G062)', () => {
  it('cancelReplayJob POSTs to /v1/replay/jobs/{id}/cancel with placeholder actor header', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi
      .fn()
      .mockResolvedValue(jsonResponse({ replay_job: { replay_job_id: 'replay-123', status: 'canceled' } }));
    vi.stubGlobal('fetch', fetchMock);

    await api.cancelReplayJob('replay-123', { reason: 'operator canceled from Replay UI' });

    const [url, options] = fetchMock.mock.calls[0];
    expect(String(url)).toContain('/v1/replay/jobs/replay-123/cancel');
    expect(options.method).toBe('POST');
    expect(options.headers['Content-Type']).toBe('application/json');
    expect(options.headers['X-SignalOps-Actor']).toBe('operator-local');
    const body = JSON.parse(options.body);
    expect(body.reason).toBe('operator canceled from Replay UI');
  });

  it('cancelReplayJob URL-encodes the replay job id', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi
      .fn()
      .mockResolvedValue(jsonResponse({ replay_job: { replay_job_id: 'replay:job-1', status: 'canceled' } }));
    vi.stubGlobal('fetch', fetchMock);

    await api.cancelReplayJob('replay:job-1');

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/replay/jobs/replay%3Ajob-1/cancel');
  });

  it('cancelReplayJob tolerates an empty body', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi
      .fn()
      .mockResolvedValue(jsonResponse({ replay_job: { replay_job_id: 'replay-123', status: 'canceled' } }));
    vi.stubGlobal('fetch', fetchMock);

    await api.cancelReplayJob('replay-123');

    const options = fetchMock.mock.calls[0][1];
    expect(options.method).toBe('POST');
    expect(JSON.parse(options.body)).toEqual({});
  });

  it('parses cancel error envelopes into ApiError (404)', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi
      .fn()
      .mockResolvedValue(jsonResponse({ error: 'replay_job_not_found', message: 'replay job not found' }, 404));
    vi.stubGlobal('fetch', fetchMock);

    await expect(api.cancelReplayJob('replay-missing')).rejects.toMatchObject({
      status: 404,
      code: 'replay_job_not_found',
      message: 'replay job not found',
    });
  });
});
