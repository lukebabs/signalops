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
const { queryKeys, applySyncraticMaterializeResult, applySyncraticAskResult } = await import('./queries');

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

describe('Syncratic API client (G088)', () => {
  it('builds the insights list path with filters + bearer + default limit', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ syncratic_insights: [] }));
    vi.stubGlobal('fetch', fetchMock);
    state.authEnabled = true;
    state.token = 'jwt-abc';

    await api.listSyncraticInsights({
      status: 'active',
      subject_symbol: 'AAPL',
      insight_type: 'marketops.syncratic.multi_event_context',
    });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/syncratic/insights');
    expect(url).toContain('tenant_id=tenant-local');
    expect(url).toContain('status=active');
    expect(url).toContain('subject_symbol=AAPL');
    expect(url).toContain('insight_type=marketops.syncratic.multi_event_context');
    expect(url).toContain('limit=50');
    expect(fetchMock.mock.calls[0][1].headers['Authorization']).toBe('Bearer jwt-abc');
  });

  it('omits unset insight filters from the query string', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ syncratic_insights: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listSyncraticInsights({});

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('tenant_id=tenant-local');
    expect(url).not.toContain('status=');
    expect(url).not.toContain('subject_symbol=');
    expect(url).not.toContain('insight_type=');
  });

  it('URL-encodes the insight id on the detail path', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ syncratic_insight: {} }));
    vi.stubGlobal('fetch', fetchMock);

    await api.getSyncraticInsight('synins/a b');

    expect(String(fetchMock.mock.calls[0][0])).toContain('/v1/syncratic/insights/synins%2Fa%20b');
  });

  it('builds the context-windows list path with strategy + status + default limit', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ context_windows: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listSyncraticContextWindows({
      subject_symbol: 'AAPL',
      context_strategy: 'symbol_signal_cluster_5d',
      status: 'active',
    });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/syncratic/context-windows');
    expect(url).toContain('tenant_id=tenant-local');
    expect(url).toContain('context_strategy=symbol_signal_cluster_5d');
    expect(url).toContain('status=active');
    expect(url).toContain('limit=50');
  });

  it('URL-encodes the context window id on the detail path', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ context_window: {} }));
    vi.stubGlobal('fetch', fetchMock);

    await api.getSyncraticContextWindow('synctx/a b');

    expect(String(fetchMock.mock.calls[0][0])).toContain('/v1/syncratic/context-windows/synctx%2Fa%20b');
  });

  it('posts the materialize request body to /v1/syncratic/materialize with content type', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ materialization: { materialized_insights: 1 } }, 201),
    );
    vi.stubGlobal('fetch', fetchMock);
    state.authEnabled = true;
    state.token = 'jwt-abc';

    await api.materializeSyncraticContexts({
      tenant_id: 'tenant-local',
      window_start: '2026-07-01T00:00:00Z',
      window_end: '2026-07-14T00:00:00Z',
      min_evidence_count: 2,
    });

    expect(String(fetchMock.mock.calls[0][0])).toContain('/v1/syncratic/materialize');
    expect(fetchMock.mock.calls[0][1].method).toBe('POST');
    expect(fetchMock.mock.calls[0][1].headers['Authorization']).toBe('Bearer jwt-abc');
    expect(fetchMock.mock.calls[0][1].headers['Content-Type']).toBe('application/json');
    expect(JSON.parse(fetchMock.mock.calls[0][1].body)).toEqual({
      tenant_id: 'tenant-local',
      window_start: '2026-07-01T00:00:00Z',
      window_end: '2026-07-14T00:00:00Z',
      min_evidence_count: 2,
    });
  });

  it('parses list, detail, context-window, and materialization envelopes', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        jsonResponse({
          syncratic_insights: [
            {
              syncratic_insight_id: 'synins-1',
              subject_symbol: 'AAPL',
              supporting_signal_ids: ['sig-1', 'sig-2'],
              supporting_alert_ids: ['alert-1'],
            },
          ],
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({ syncratic_insight: { syncratic_insight_id: 'synins-1', subject_symbol: 'AAPL' } }),
      )
      .mockResolvedValueOnce(jsonResponse({ context_windows: [{ context_window_id: 'synctx-1' }] }))
      .mockResolvedValueOnce(
        jsonResponse({ context_window: { context_window_id: 'synctx-1', context_strategy: 'symbol_signal_cluster_5d' } }),
      )
      .mockResolvedValueOnce(
        jsonResponse(
          { materialization: { scanned_assets: 5, materialized_context_windows: 1, materialized_insights: 1, skipped_below_threshold: 4 } },
          201,
        ),
      );
    vi.stubGlobal('fetch', fetchMock);

    const list = await api.listSyncraticInsights({});
    const detail = await api.getSyncraticInsight('synins-1');
    const cwList = await api.listSyncraticContextWindows({});
    const cw = await api.getSyncraticContextWindow('synctx-1');
    const mat = await api.materializeSyncraticContexts({
      tenant_id: 'tenant-local',
      window_start: '2026-07-01T00:00:00Z',
      window_end: '2026-07-14T00:00:00Z',
    });

    expect(list.syncratic_insights[0].syncratic_insight_id).toBe('synins-1');
    expect(list.syncratic_insights[0].supporting_signal_ids).toHaveLength(2);
    expect(detail.syncratic_insight.subject_symbol).toBe('AAPL');
    expect(cwList.context_windows[0].context_window_id).toBe('synctx-1');
    expect(cw.context_window.context_strategy).toBe('symbol_signal_cluster_5d');
    expect(mat.materialization.materialized_insights).toBe(1);
    expect(mat.materialization.skipped_below_threshold).toBe(4);
  });
});

describe('Syncratic query keys (G088)', () => {
  it('uses stable insight + context-window list/detail keys', () => {
    const insightFilter = { tenant_id: 'tenant-local', limit: 50 };
    const cwFilter = { tenant_id: 'tenant-local', limit: 50 };

    expect(queryKeys.syncraticInsights(insightFilter)).toEqual(['syncratic-insights', insightFilter]);
    expect(queryKeys.syncraticInsights({ ...insightFilter })).toEqual(['syncratic-insights', insightFilter]);
    expect(queryKeys.syncraticInsight('synins-1')).toEqual(['syncratic-insight', 'synins-1']);
    expect(queryKeys.syncraticContextWindows(cwFilter)).toEqual(['syncratic-context-windows', cwFilter]);
    expect(queryKeys.syncraticContextWindows({ ...cwFilter })).toEqual(['syncratic-context-windows', cwFilter]);
    expect(queryKeys.syncraticContextWindow('synctx-1')).toEqual(['syncratic-context-window', 'synctx-1']);
  });
});

describe('applySyncraticMaterializeResult (G088 invalidation)', () => {
  it('invalidates only Syncratic insight + context-window prefixes', () => {
    const queryClient = new QueryClient();
    const invSpy = vi.spyOn(queryClient, 'invalidateQueries');

    applySyncraticMaterializeResult(queryClient, {
      materialization: { materialized_insights: 1 },
    } as never);

    expect(invSpy).toHaveBeenCalledWith({ queryKey: ['syncratic-insights'] });
    expect(invSpy).toHaveBeenCalledWith({ queryKey: ['syncratic-insight'] });
    expect(invSpy).toHaveBeenCalledWith({ queryKey: ['syncratic-context-windows'] });
    expect(invSpy).toHaveBeenCalledWith({ queryKey: ['syncratic-context-window'] });
    // Production + sibling evidence queries are never invalidated by a materialize.
    expect(invSpy).not.toHaveBeenCalledWith({ queryKey: ['alerts'] });
    expect(invSpy).not.toHaveBeenCalledWith({ queryKey: ['signals'] });
    expect(invSpy).not.toHaveBeenCalledWith({ queryKey: ['insights'] });
    expect(invSpy).not.toHaveBeenCalledWith({ queryKey: ['marketops-backtests'] });
    expect(invSpy).not.toHaveBeenCalledWith({ queryKey: ['marketops-backtest-promotion-candidates'] });
    expect(invSpy).not.toHaveBeenCalledWith({ queryKey: ['marketops-dsm-graph-proposals'] });
  });
});

describe('askSyncraticContextWindow (G090)', () => {
  it('posts the ask request with force=false to the context-window ask path', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({
        ask_result: { ask_status: 'completed', updated: true },
        syncratic_insight: { syncratic_insight_id: 'synins-1' },
      }),
    );
    vi.stubGlobal('fetch', fetchMock);
    state.authEnabled = true;
    state.token = 'jwt-abc';

    await api.askSyncraticContextWindow('synctx-1', {
      tenant_id: 'tenant-local',
      max_prompt_bytes: 12000,
      force: false,
    });

    expect(String(fetchMock.mock.calls[0][0])).toContain('/v1/syncratic/context-windows/synctx-1/ask');
    expect(fetchMock.mock.calls[0][1].method).toBe('POST');
    expect(fetchMock.mock.calls[0][1].headers['Authorization']).toBe('Bearer jwt-abc');
    expect(fetchMock.mock.calls[0][1].headers['Content-Type']).toBe('application/json');
    expect(JSON.parse(fetchMock.mock.calls[0][1].body)).toEqual({
      tenant_id: 'tenant-local',
      max_prompt_bytes: 12000,
      force: false,
    });
  });

  it('sends force=true for an explicit regenerate action', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ ask_result: {}, syncratic_insight: {} }));
    vi.stubGlobal('fetch', fetchMock);

    await api.askSyncraticContextWindow('synctx/a b', {
      tenant_id: 'tenant-local',
      force: true,
    });

    // URL-encodes the context window id on the ask path.
    expect(String(fetchMock.mock.calls[0][0])).toContain('/v1/syncratic/context-windows/synctx%2Fa%20b/ask');
    expect(JSON.parse(fetchMock.mock.calls[0][1].body).force).toBe(true);
  });
});

describe('applySyncraticAskResult (G090 cache effect)', () => {
  it('seeds the refreshed insight into the detail cache and invalidates only Syncratic prefixes', () => {
    const queryClient = new QueryClient();
    const invSpy = vi.spyOn(queryClient, 'invalidateQueries');
    const setSpy = vi.spyOn(queryClient, 'setQueryData');

    applySyncraticAskResult(queryClient, {
      ask_result: { ask_status: 'completed', updated: true } as never,
      syncratic_insight: { syncratic_insight_id: 'synins-1' } as never,
    });

    // Instant detail update from the full returned insight.
    expect(setSpy).toHaveBeenCalledWith(['syncratic-insight', 'synins-1'], {
      syncratic_insight: { syncratic_insight_id: 'synins-1' },
    });
    expect(invSpy).toHaveBeenCalledWith({ queryKey: ['syncratic-insights'] });
    expect(invSpy).toHaveBeenCalledWith({ queryKey: ['syncratic-insight'] });
    expect(invSpy).toHaveBeenCalledWith({ queryKey: ['syncratic-context-windows'] });
    expect(invSpy).toHaveBeenCalledWith({ queryKey: ['syncratic-context-window'] });
    // Production + sibling queries are untouched by an Ask action.
    expect(invSpy).not.toHaveBeenCalledWith({ queryKey: ['alerts'] });
    expect(invSpy).not.toHaveBeenCalledWith({ queryKey: ['signals'] });
    expect(invSpy).not.toHaveBeenCalledWith({ queryKey: ['insights'] });
    expect(invSpy).not.toHaveBeenCalledWith({ queryKey: ['marketops-backtests'] });
  });

  it('skips the cache seed when the returned insight has no id (defensive)', () => {
    const queryClient = new QueryClient();
    const setSpy = vi.spyOn(queryClient, 'setQueryData');
    const invSpy = vi.spyOn(queryClient, 'invalidateQueries');

    applySyncraticAskResult(queryClient, {
      ask_result: { ask_status: 'skipped', updated: false } as never,
      syncratic_insight: {} as never,
    });

    // No id to seed under, but the list/detail prefixes still invalidate.
    expect(setSpy).not.toHaveBeenCalled();
    expect(invSpy).toHaveBeenCalledWith({ queryKey: ['syncratic-insights'] });
  });
});

describe('materializeSyncraticContexts dry_run (G092)', () => {
  it('sends dry_run=true for a preview and parses decisions[]', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({
        materialization: {
          dry_run: true,
          scanned_assets: 10,
          materialized_context_windows: 0,
          materialized_insights: 0,
          skipped_below_threshold: 9,
          decisions: [
            {
              subject_symbol: 'AAPL',
              action: 'would_materialize',
              reason: 'eligible',
              evidence_count: 9,
              signal_count: 9,
              alert_count: 0,
              artifact_count: 0,
              graph_proposal_count: 0,
              label_count: 0,
              critical_alert: false,
              related_evidence: false,
              context_window_id: 'synctx_aapl',
            },
          ],
        },
      }),
    );
    vi.stubGlobal('fetch', fetchMock);

    const res = await api.materializeSyncraticContexts({
      tenant_id: 'tenant-local',
      window_start: '2026-07-12T00:00:00Z',
      window_end: '2026-07-14T00:00:00Z',
      max_assets: 10,
      max_context_windows: 1,
      max_insights: 1,
      dry_run: true,
    });

    expect(String(fetchMock.mock.calls[0][0])).toContain('/v1/syncratic/materialize');
    expect(JSON.parse(fetchMock.mock.calls[0][1].body).dry_run).toBe(true);
    // Dry-run returns 200, not 201 — the client treats any 2xx as success.
    expect(res.materialization.dry_run).toBe(true);
    expect(res.materialization.decisions).toHaveLength(1);
    expect(res.materialization.decisions[0].action).toBe('would_materialize');
    expect(res.materialization.decisions[0].subject_symbol).toBe('AAPL');
  });

  it('sends dry_run=false for a confirmed write', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ materialization: { dry_run: false, materialized_insights: 1, decisions: [] } }, 201),
    );
    vi.stubGlobal('fetch', fetchMock);

    await api.materializeSyncraticContexts({
      tenant_id: 'tenant-local',
      window_start: '2026-07-12T00:00:00Z',
      window_end: '2026-07-14T00:00:00Z',
      dry_run: false,
    });

    expect(JSON.parse(fetchMock.mock.calls[0][1].body).dry_run).toBe(false);
  });
});

describe('applySyncraticMaterializeResult dry_run guard (G092)', () => {
  it('does not invalidate caches on a dry-run preview', () => {
    const queryClient = new QueryClient();
    const invSpy = vi.spyOn(queryClient, 'invalidateQueries');

    applySyncraticMaterializeResult(queryClient, {
      materialization: { dry_run: true, decisions: [] },
    } as never);

    expect(invSpy).not.toHaveBeenCalled();
  });

  it('invalidates Syncratic prefixes on a confirmed write', () => {
    const queryClient = new QueryClient();
    const invSpy = vi.spyOn(queryClient, 'invalidateQueries');

    applySyncraticMaterializeResult(queryClient, {
      materialization: { dry_run: false, materialized_insights: 1, decisions: [] },
    } as never);

    expect(invSpy).toHaveBeenCalledWith({ queryKey: ['syncratic-insights'] });
    expect(invSpy).toHaveBeenCalledWith({ queryKey: ['syncratic-context-window'] });
    expect(invSpy).not.toHaveBeenCalledWith({ queryKey: ['alerts'] });
  });
});
