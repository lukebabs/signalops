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

describe('MarketOps opportunities API client (G139)', () => {
  it('builds the opportunity list path with every filter and bearer', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ opportunities: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listMarketOpsOpportunities({
      tenant_id: 'tenant-1',
      app_id: 'marketops',
      opportunity_id: 'mopp-1',
      asset_id: 'ticker:AAPL',
      symbol: 'AAPL',
      direction: 'downside',
      horizon: '5_to_20_sessions',
      lifecycle_status: 'active',
      research_only: true,
      session_start: '2026-07-01',
      session_end: '2026-07-31',
      limit: 25,
    });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/marketops/opportunities');
    expect(url).toContain('tenant_id=tenant-1');
    expect(url).toContain('app_id=marketops');
    expect(url).toContain('opportunity_id=mopp-1');
    expect(url).toContain('asset_id=ticker%3AAAPL');
    expect(url).toContain('symbol=AAPL');
    expect(url).toContain('direction=downside');
    expect(url).toContain('horizon=5_to_20_sessions');
    expect(url).toContain('lifecycle_status=active');
    expect(url).toContain('research_only=true');
    expect(url).toContain('session_start=2026-07-01');
    expect(url).toContain('session_end=2026-07-31');
    expect(url).toContain('limit=25');
    expect(fetchMock.mock.calls[0][1].headers['Authorization']).toBe('Bearer jwt-abc');
  });

  it('defaults limit to 50 and serializes research_only=false only when explicitly set', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ opportunities: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listMarketOpsOpportunities({ tenant_id: 'tenant-1', research_only: false });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('limit=50');
    expect(url).toContain('research_only=false');
  });

  it('omits research_only when unset and builds the detail path with tenant + encoded id', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ opportunity: {} }));
    vi.stubGlobal('fetch', fetchMock);

    await api.getMarketOpsOpportunity('mopp/a b', 'tenant-1');

    const call = fetchMock.mock.calls[0];
    expect(String(call[0])).toContain('/v1/marketops/opportunities/mopp%2Fa%20b');
    expect(String(call[0])).toContain('tenant_id=tenant-1');
    expect(call[1].headers['Authorization']).toBe('Bearer jwt-abc');
  });

  it('builds the hypothesis-evaluations path with boolean filters', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ hypothesis_evaluations: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listMarketOpsHypothesisEvaluations({
      tenant_id: 'tenant-1',
      symbol: 'AAPL',
      eligible: true,
      triggered: false,
      limit: 200,
    });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/marketops/hypothesis-evaluations');
    expect(url).toContain('symbol=AAPL');
    expect(url).toContain('eligible=true');
    expect(url).toContain('triggered=false');
    expect(url).toContain('limit=200');
  });

  it('builds hypothesis, evidence, and lineage paths', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ hypothesis: { hypothesis_key: 'h' } }))
      .mockResolvedValueOnce(jsonResponse({ evidence: { evidence_id: 'ev' } }))
      .mockResolvedValueOnce(jsonResponse({ lineage: { source_event_ids: [] } }));
    vi.stubGlobal('fetch', fetchMock);

    await api.getMarketOpsHypothesis('hkey', '1.0', 'tenant-1');
    await api.getMarketOpsEvidence('ev/1');
    await api.getMarketOpsMarketStateLineage('ms/1');

    const calls = fetchMock.mock.calls.map((c) => String(c[0]));
    expect(calls[0]).toContain('/v1/marketops/hypotheses/hkey/1.0');
    expect(calls[0]).toContain('tenant_id=tenant-1');
    expect(calls[1]).toContain('/v1/marketops/evidence/ev%2F1');
    expect(calls[2]).toContain('/v1/marketops/states/ms%2F1/lineage');
  });

  it('parses opportunity + hypothesis-evaluation envelopes', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ opportunities: [{ opportunity_id: 'mopp-1', symbol: 'AAPL', opportunity_score: 0.8 }] }))
      .mockResolvedValueOnce(
        jsonResponse({
          hypothesis_evaluations: [{ evaluation_id: 'eval-1', eligible: true, triggered: true, reason_codes: ['eligible_not_triggered'] }],
        }),
      );
    vi.stubGlobal('fetch', fetchMock);

    const list = await api.listMarketOpsOpportunities({ tenant_id: 'tenant-1' });
    const evals = await api.listMarketOpsHypothesisEvaluations({ tenant_id: 'tenant-1' });

    expect(list.opportunities[0].opportunity_id).toBe('mopp-1');
    expect(list.opportunities[0].opportunity_score).toBeCloseTo(0.8);
    expect(evals.hypothesis_evaluations[0].evaluation_id).toBe('eval-1');
    expect(evals.hypothesis_evaluations[0].reason_codes).toEqual(['eligible_not_triggered']);
  });
});

describe('MarketOps opportunities query keys (G139)', () => {
  it('uses stable list/detail/supporting keys', () => {
    const filter = { tenant_id: 'tenant-1', symbol: 'AAPL', limit: 50 };
    expect(queryKeys.marketOpsOpportunities(filter)).toEqual(['marketops-opportunities', filter]);
    expect(queryKeys.marketOpsOpportunity('mopp-1', 'tenant-1')).toEqual(['marketops-opportunity', 'mopp-1', 'tenant-1']);
    expect(queryKeys.marketOpsHypothesisEvaluations(filter)).toEqual(['marketops-hypothesis-evaluations', filter]);
    expect(queryKeys.marketOpsHypothesis('h', '1', 't')).toEqual(['marketops-hypothesis', 'h', '1', 't']);
    expect(queryKeys.marketOpsEvidence('ev-1')).toEqual(['marketops-evidence', 'ev-1']);
    expect(queryKeys.marketOpsMarketStateLineage('ms-1')).toEqual(['marketops-market-state-lineage', 'ms-1']);
  });
});
