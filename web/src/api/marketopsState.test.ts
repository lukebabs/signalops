import { afterEach, describe, expect, it, vi } from 'vitest';
import { QueryClient } from '@tanstack/react-query';

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
const { queryKeys, applyOpportunityDispositionResult } = await import('./queries');

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  state.token = 'jwt-abc';
  state.authEnabled = true;
});

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
}

describe('MarketOps state API client (G147)', () => {
  it('builds the states list path with filters + bearer', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ market_states: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listMarketOpsStates({
      tenant_id: 'tenant-1',
      symbol: 'AAPL',
      state_schema_version: 'marketops.state.v1',
      quality_state: 'usable',
      session_start: '2026-07-01',
      session_end: '2026-07-31',
      limit: 25,
    });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/marketops/states');
    expect(url).toContain('tenant_id=tenant-1');
    expect(url).toContain('symbol=AAPL');
    expect(url).toContain('state_schema_version=marketops.state.v1');
    expect(url).toContain('quality_state=usable');
    expect(url).toContain('session_start=2026-07-01');
    expect(url).toContain('session_end=2026-07-31');
    expect(url).toContain('limit=25');
    expect(fetchMock.mock.calls[0][1].headers['Authorization']).toBe('Bearer jwt-abc');
  });

  it('fetches state detail by path with no tenant query', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ market_state: {} }));
    vi.stubGlobal('fetch', fetchMock);

    await api.getMarketOpsState('mstate/a');

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/marketops/states/mstate%2Fa');
    expect(url).not.toContain('tenant_id=');
  });

  it('builds the feature observations path with a JSON dimensions filter', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ feature_observations: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listMarketOpsFeatureObservations({ tenant_id: 'tenant-1', symbol: 'AAPL', dimensions: '{"target_dte":30}', limit: 10 });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/marketops/features/observations');
    expect(url).toContain('dimensions=%7B%22target_dte%22%3A30%7D');
    expect(url).toContain('limit=10');
  });

  it('builds the transitions path with current_state_id + transition_type', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ transitions: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listMarketOpsStateTransitions({ tenant_id: 'tenant-1', symbol: 'AAPL', current_state_id: 'mstate-1', transition_type: 'zscore' });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/marketops/transitions');
    expect(url).toContain('current_state_id=mstate-1');
    expect(url).toContain('transition_type=zscore');
  });

  it('builds the outcomes path with outcome_status + horizon_sessions', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ outcomes: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listMarketOpsOutcomes({ tenant_id: 'tenant-1', source_type: 'hypothesis_evaluation', outcome_status: 'matured', horizon_sessions: 5 });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/marketops/outcomes');
    expect(url).toContain('source_type=hypothesis_evaluation');
    expect(url).toContain('outcome_status=matured');
    expect(url).toContain('horizon_sessions=5');
    expect(url).not.toContain('&status=');
    expect(url).not.toContain('?status=');
  });

  it('parses states, transitions, and outcomes envelopes', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ market_states: [{ market_state_id: 'ms-1', symbol: 'AAPL', quality_score: 0.9 }] }))
      .mockResolvedValueOnce(jsonResponse({ transitions: [{ transition_id: 't-1', transition_type: 'zscore' }] }))
      .mockResolvedValueOnce(jsonResponse({ outcomes: [{ outcome_id: 'o-1', outcome_status: 'matured', horizon_sessions: 5 }] }));
    vi.stubGlobal('fetch', fetchMock);

    const states = await api.listMarketOpsStates({ tenant_id: 'tenant-1' });
    const transitions = await api.listMarketOpsStateTransitions({ tenant_id: 'tenant-1' });
    const outcomes = await api.listMarketOpsOutcomes({ tenant_id: 'tenant-1' });

    expect(states.market_states[0].market_state_id).toBe('ms-1');
    expect(states.market_states[0].quality_score).toBeCloseTo(0.9);
    expect(transitions.transitions[0].transition_type).toBe('zscore');
    expect(outcomes.outcomes[0].horizon_sessions).toBe(5);
  });
});

describe('MarketOps opportunity disposition API client (G146/G147)', () => {
  it('lists dispositions with tenant + limit', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ opportunity_dispositions: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listMarketOpsOpportunityDispositions('mopp-1', { tenant_id: 'tenant-1', limit: 25 });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/marketops/opportunities/mopp-1/dispositions');
    expect(url).toContain('tenant_id=tenant-1');
    expect(url).toContain('limit=25');
  });

  it('posts a disposition without an actor field or actor header', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ opportunity_disposition: { disposition_id: 'd-1' } }));
    vi.stubGlobal('fetch', fetchMock);

    await api.createMarketOpsOpportunityDisposition('mopp-1', {
      tenant_id: 'tenant-1',
      disposition: 'needs_more_evidence',
      note: 'await another session',
      metadata: {},
    });

    const call = fetchMock.mock.calls[0];
    expect(String(call[0])).toContain('/v1/marketops/opportunities/mopp-1/dispositions');
    expect(call[1].method).toBe('POST');
    expect(call[1].headers['Authorization']).toBe('Bearer jwt-abc');
    expect(call[1].headers['X-SignalOps-Actor']).toBeUndefined();
    const body = JSON.parse(call[1].body as string);
    expect(body).toEqual({ tenant_id: 'tenant-1', disposition: 'needs_more_evidence', note: 'await another session', metadata: {} });
    expect(body).not.toHaveProperty('actor');
  });
});

describe('MarketOps state query keys (G147)', () => {
  it('invalidates only the selected opportunity disposition ledger', () => {
    const queryClient = new QueryClient();
    const invalidate = vi.spyOn(queryClient, 'invalidateQueries');
    applyOpportunityDispositionResult(queryClient, 'mopp-1');
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['marketops-opportunity-dispositions', 'mopp-1'] });
  });

  it('uses stable state/feature/transition/outcome/disposition keys', () => {
    expect(queryKeys.marketOpsStates({ tenant_id: 't', symbol: 'AAPL' })).toEqual(['marketops-states', { tenant_id: 't', symbol: 'AAPL' }]);
    expect(queryKeys.marketOpsState('ms-1')).toEqual(['marketops-state', 'ms-1']);
    expect(queryKeys.marketOpsFeatureObservations({ tenant_id: 't' })).toEqual(['marketops-feature-observations', { tenant_id: 't' }]);
    expect(queryKeys.marketOpsStateTransitions({ tenant_id: 't' })).toEqual(['marketops-state-transitions', { tenant_id: 't' }]);
    expect(queryKeys.marketOpsOutcomes({ tenant_id: 't' })).toEqual(['marketops-outcomes', { tenant_id: 't' }]);
    expect(queryKeys.marketOpsOpportunityDispositions('mopp-1', { tenant_id: 't' })).toEqual([
      'marketops-opportunity-dispositions',
      'mopp-1',
      { tenant_id: 't' },
    ]);
  });
});
