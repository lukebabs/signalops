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

describe('MarketOps DSM graph proposals API client (G079)', () => {
  it('builds the graph-proposals list path with all filters', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ graph_proposals: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listMarketOpsDSMGraphProposals({
      tenant_id: 'tenant-local',
      app_id: 'marketops',
      domain: 'market_data',
      use_case: 'daily_market_surveillance',
      artifact_id: 'artifact_marketops_dsm_v1_g079',
      signal_id: 'sig_marketops_dsm_taxonomy_v1_g079',
      signal_type: 'marketops.dsm.pinning_risk',
      subject_symbol: 'AAPL',
      candidate_type: 'relationship_candidate',
      status: 'proposed',
      limit: 25,
    });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/marketops/dsm/graph-proposals');
    expect(url).toContain('tenant_id=tenant-local');
    expect(url).toContain('app_id=marketops');
    expect(url).toContain('domain=market_data');
    expect(url).toContain('use_case=daily_market_surveillance');
    expect(url).toContain('artifact_id=artifact_marketops_dsm_v1_g079');
    expect(url).toContain('signal_id=sig_marketops_dsm_taxonomy_v1_g079');
    expect(url).toContain('signal_type=marketops.dsm.pinning_risk');
    expect(url).toContain('subject_symbol=AAPL');
    expect(url).toContain('candidate_type=relationship_candidate');
    expect(url).toContain('status=proposed');
    expect(url).toContain('limit=25');
  });

  it('defaults the list limit to 50 and attaches the bearer token', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ graph_proposals: [] }));
    vi.stubGlobal('fetch', fetchMock);
    state.authEnabled = true;
    state.token = 'jwt-abc';

    await api.listMarketOpsDSMGraphProposals({ tenant_id: 'tenant-local', signal_id: 'sig-1' });

    expect(String(fetchMock.mock.calls[0][0])).toContain('limit=50');
    expect(fetchMock.mock.calls[0][1].headers['Authorization']).toBe('Bearer jwt-abc');
  });

  it('omits undefined filter values from the query string', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ graph_proposals: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listMarketOpsDSMGraphProposals({ signal_id: 'sig-1' });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('signal_id=sig-1');
    expect(url).not.toContain('artifact_id=');
    expect(url).not.toContain('candidate_type=');
    expect(url).not.toContain('status=');
  });

  it('encodes proposal ids for detail fetches', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ graph_proposal: { proposal_id: 'graphprop:1' } }));
    vi.stubGlobal('fetch', fetchMock);

    await api.getMarketOpsDSMGraphProposal('graphprop:1');

    expect(String(fetchMock.mock.calls[0][0])).toContain('/v1/marketops/dsm/graph-proposals/graphprop%3A1');
  });

  it('parses the list and detail envelopes', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        jsonResponse({ graph_proposals: [{ proposal_id: 'graphprop-1', candidate_type: 'node_candidate' }] }),
      )
      .mockResolvedValueOnce(
        jsonResponse({ graph_proposal: { proposal_id: 'graphprop-1', status: 'proposed' } }),
      );
    vi.stubGlobal('fetch', fetchMock);

    const list = await api.listMarketOpsDSMGraphProposals({});
    const detail = await api.getMarketOpsDSMGraphProposal('graphprop-1');

    expect(list.graph_proposals[0].proposal_id).toBe('graphprop-1');
    expect(list.graph_proposals[0].candidate_type).toBe('node_candidate');
    expect(detail.graph_proposal.status).toBe('proposed');
  });
});

describe('MarketOps DSM graph proposal query keys (G079)', () => {
  it('uses stable list and detail keys', () => {
    const filter = { tenant_id: 'tenant-local', signal_id: 'sig-1', limit: 50 };
    const a = queryKeys.marketOpsDSMGraphProposals(filter);
    const b = queryKeys.marketOpsDSMGraphProposals({ ...filter });
    expect(a).toEqual(['marketops-dsm-graph-proposals', filter]);
    expect(a).toEqual(b);
    expect(queryKeys.marketOpsDSMGraphProposal('graphprop-1')).toEqual([
      'marketops-dsm-graph-proposal',
      'graphprop-1',
    ]);
  });
});
