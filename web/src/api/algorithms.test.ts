import { afterEach, describe, expect, it, vi } from 'vitest';
import { QueryClient } from '@tanstack/react-query';
import type { AlgorithmSignalProposal, AlgorithmSignalProposalResponse } from '../types';

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
const { applyAlgorithmSignalProposalDecisionResult, applyMaterializeAlgorithmSignalProposalResult } = await import('./queries');

function sampleProposal(overrides: Partial<AlgorithmSignalProposal> = {}): AlgorithmSignalProposal {
  return {
    proposal_id: 'algsigprop_c6c2acad697176d0f438b66e',
    tenant_id: 'tenant-local',
    algorithm_result_id: 'algres_1',
    algorithm_id: 'ruptures_change_point_v1',
    algorithm_version: '0.1.0',
    execution_request_id: 'algexec-g110-ruptures-aapl-openclose',
    proposed_signal_type: 'signalops.algorithm.change_point_candidate',
    status: 'proposed',
    score: 2.5,
    confidence: 0.9,
    severity: 'critical',
    proposal_payload: { window: {} },
    rationale: { detector: 'ruptures' },
    source_event_ids: ['evt_1'],
    evidence_refs: ['ev_1'],
    correlation_id: 'corr_1',
    created_by: 'operator-local',
    reviewed_by: '',
    decision_note: '',
    created_at: '2026-07-16T00:00:00Z',
    updated_at: '2026-07-16T00:00:00Z',
    ...overrides,
  };
}

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

describe('algorithm signal proposal API client (G114)', () => {
  it('builds the proposals list path with filters + tenant + bearer + default limit', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ algorithm_signal_proposals: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listAlgorithmSignalProposals({
      tenant_id: 'tenant-local',
      status: 'proposed',
      severity: 'critical',
      algorithm_id: 'ruptures_change_point_v1',
      execution_request_id: 'algexec_1',
      algorithm_result_id: 'algres_1',
      correlation_id: 'corr_1',
      proposal_source: 'hypothesis_evaluation',
      hypothesis_evaluation_id: 'heval_1',
      hypothesis_key: 'H001',
    });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/algorithms/signal-proposals');
    expect(url).toContain('tenant_id=tenant-local');
    expect(url).toContain('status=proposed');
    expect(url).toContain('severity=critical');
    expect(url).toContain('algorithm_id=ruptures_change_point_v1');
    expect(url).toContain('execution_request_id=algexec_1');
    expect(url).toContain('algorithm_result_id=algres_1');
    expect(url).toContain('correlation_id=corr_1');
    expect(url).toContain('proposal_source=hypothesis_evaluation');
    expect(url).toContain('hypothesis_evaluation_id=heval_1');
    expect(url).toContain('hypothesis_key=H001');
    expect(url).toContain('limit=50');
    expect(fetchMock.mock.calls[0][1].headers['Authorization']).toBe('Bearer jwt-abc');
  });

  it('omits unset proposal filters, defaults tenant, and applies status=proposed from the route', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ algorithm_signal_proposals: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listAlgorithmSignalProposals({ tenant_id: 'tenant-local', status: 'proposed' });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('tenant_id=tenant-local');
    expect(url).toContain('status=proposed');
    expect(url).not.toContain('severity=');
    expect(url).not.toContain('correlation_id=');
  });

  it('URL-encodes the proposal id on the detail path and sends the tenant + bearer', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ algorithm_signal_proposal: {} }));
    vi.stubGlobal('fetch', fetchMock);

    await api.getAlgorithmSignalProposal('algsigprop/a b', 'tenant-local');

    const call = fetchMock.mock.calls[0];
    expect(String(call[0])).toContain('/v1/algorithms/signal-proposals/algsigprop%2Fa%20b');
    expect(String(call[0])).toContain('tenant_id=tenant-local');
    expect(call[1].headers['Authorization']).toBe('Bearer jwt-abc');
  });

  it('posts the decision body to the decision path with bearer and no actor header', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi
      .fn()
      .mockResolvedValue(jsonResponse({ algorithm_signal_proposal: sampleProposal({ status: 'reviewed' }) }));
    vi.stubGlobal('fetch', fetchMock);

    await api.decideAlgorithmSignalProposal('algsigprop_1', {
      tenant_id: 'tenant-local',
      status: 'reviewed',
      note: 'Useful evidence; no production signal materialized.',
    });

    const call = fetchMock.mock.calls[0];
    expect(String(call[0])).toContain('/v1/algorithms/signal-proposals/algsigprop_1/decision');
    expect(call[1].method).toBe('POST');
    expect(call[1].headers['Authorization']).toBe('Bearer jwt-abc');
    expect(call[1].headers['X-SignalOps-Actor']).toBeUndefined();
    const body = JSON.parse(call[1].body as string);
    expect(body).toEqual({
      tenant_id: 'tenant-local',
      status: 'reviewed',
      note: 'Useful evidence; no production signal materialized.',
    });
  });

  it('parses the list and detail envelopes', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ algorithm_signal_proposals: [sampleProposal()] }))
      .mockResolvedValueOnce(
        jsonResponse({ algorithm_signal_proposal: sampleProposal({ status: 'rejected', reviewed_by: 'a1' }) }),
      );
    vi.stubGlobal('fetch', fetchMock);

    const list = await api.listAlgorithmSignalProposals({ tenant_id: 'tenant-local' });
    const detail = await api.getAlgorithmSignalProposal('algsigprop_c6c2acad697176d0f438b66e');

    expect(list.algorithm_signal_proposals[0].proposal_id).toBe('algsigprop_c6c2acad697176d0f438b66e');
    expect(list.algorithm_signal_proposals[0].proposed_signal_type).toBe('signalops.algorithm.change_point_candidate');
    expect(detail.algorithm_signal_proposal.status).toBe('rejected');
    expect(detail.algorithm_signal_proposal.reviewed_by).toBe('a1');
  });
});

describe('applyAlgorithmSignalProposalDecisionResult (G114 mutation invalidation)', () => {
  it('seeds the detail cache and invalidates list/detail prefixes only', () => {
    const queryClient = new QueryClient();
    const setSpy = vi.spyOn(queryClient, 'setQueryData');
    const invSpy = vi.spyOn(queryClient, 'invalidateQueries');
    const decided: AlgorithmSignalProposalResponse = {
      algorithm_signal_proposal: sampleProposal({ status: 'reviewed', reviewed_by: 'analyst-1' }),
    };

    applyAlgorithmSignalProposalDecisionResult(queryClient, decided, 'algsigprop_c6c2acad697176d0f438b66e', 'tenant-local');

    // Detail cache seeded under the proposal detail key with the returned row.
    expect(setSpy).toHaveBeenCalledOnce();
    expect(setSpy.mock.calls[0][0]).toEqual([
      'algorithm-signal-proposal',
      'algsigprop_c6c2acad697176d0f438b66e',
      'tenant-local',
    ]);
    expect(setSpy.mock.calls[0][1]).toEqual(decided);

    // List + detail prefixes invalidated; nothing else (no production signal/exec queries).
    const invalidated = invSpy.mock.calls.map((c) => c[0]);
    expect(invalidated).toContainEqual({ queryKey: ['algorithm-signal-proposals'] });
    expect(invalidated).toContainEqual({
      queryKey: ['algorithm-signal-proposal', 'algsigprop_c6c2acad697176d0f438b66e', 'tenant-local'],
    });
    // G116: the coverage summary prefix is also invalidated so it refreshes after a decision.
    expect(invalidated).toContainEqual({ queryKey: ['algorithm-signal-proposal-summary'] });
    // G119: the materialization preflight prefix is invalidated too, because a
    // review decision flips a proposal's preflight_status (e.g. unreviewed -> eligible).
    expect(invalidated).toContainEqual({ queryKey: ['algorithm-signal-materialization-preflight'] });
  });
});

describe('algorithm signal proposal summary API client (G116)', () => {
  it('builds the summary path with coupled filters, tenant, bearer, and no limit', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ algorithm_signal_proposal_summary: {} }));
    vi.stubGlobal('fetch', fetchMock);

    await api.getAlgorithmSignalProposalSummary({
      tenant_id: 'tenant-local',
      algorithm_id: 'ruptures_change_point_v1',
      execution_request_id: 'algexec_1',
      status: 'proposed',
      severity: 'critical',
      correlation_id: 'corr_1',
      // limit must be ignored by the summary endpoint — it is never sent.
      limit: 50,
    });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/algorithms/signal-proposals/summary');
    expect(url).toContain('tenant_id=tenant-local');
    expect(url).toContain('algorithm_id=ruptures_change_point_v1');
    expect(url).toContain('execution_request_id=algexec_1');
    expect(url).toContain('status=proposed');
    expect(url).toContain('severity=critical');
    expect(url).toContain('correlation_id=corr_1');
    expect(url).not.toContain('limit=');
    expect(fetchMock.mock.calls[0][1].headers['Authorization']).toBe('Bearer jwt-abc');
  });

  it('defaults tenant and omits unset filters on the summary path', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ algorithm_signal_proposal_summary: {} }));
    vi.stubGlobal('fetch', fetchMock);

    await api.getAlgorithmSignalProposalSummary({});

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/algorithms/signal-proposals/summary');
    expect(url).toContain('tenant_id=tenant-local');
    expect(url).not.toContain('status=');
    expect(url).not.toContain('severity=');
    expect(url).not.toContain('limit=');
  });

  it('parses the summary envelope', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({
        algorithm_signal_proposal_summary: {
          tenant_id: 'tenant-local',
          total_proposals: 1,
          proposed_count: 0,
          reviewed_count: 1,
          rejected_count: 0,
          superseded_count: 0,
          reviewed_ratio: 1,
          high_critical_unreviewed_count: 0,
          status_counts: { reviewed: 1 },
          severity_counts: { critical: 1 },
          proposed_signal_type_counts: { 'signalops.algorithm.change_point_candidate': 1 },
          algorithm_id_counts: { 'signalops.algorithms.ruptures_change_point_v1': 1 },
          reviewer_counts: { 'operator-local': 1 },
        },
      }),
    );
    vi.stubGlobal('fetch', fetchMock);

    const s = await api.getAlgorithmSignalProposalSummary({ tenant_id: 'tenant-local' });
    expect(s.algorithm_signal_proposal_summary.total_proposals).toBe(1);
    expect(s.algorithm_signal_proposal_summary.reviewed_ratio).toBe(1);
    expect(s.algorithm_signal_proposal_summary.high_critical_unreviewed_count).toBe(0);
    expect(s.algorithm_signal_proposal_summary.status_counts.reviewed).toBe(1);
    expect(s.algorithm_signal_proposal_summary.proposed_signal_type_counts['signalops.algorithm.change_point_candidate']).toBe(1);
  });
});

describe('algorithm signal materialization preflight API client (G119)', () => {
  it('builds the preflight path with coupled filters + tenant + bearer + limit + defaults', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ algorithm_signal_materialization_preflight: {} }));
    vi.stubGlobal('fetch', fetchMock);

    await api.getAlgorithmSignalMaterializationPreflight({
      tenant_id: 'tenant-local',
      algorithm_id: 'ruptures_change_point_v1',
      execution_request_id: 'algexec_1',
      algorithm_result_id: 'algres_1',
      status: 'reviewed',
      severity: 'medium',
      correlation_id: 'corr_1',
      limit: 50,
    });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/algorithms/signal-proposals/materialization-preflight');
    expect(url).toContain('tenant_id=tenant-local');
    expect(url).toContain('algorithm_id=ruptures_change_point_v1');
    expect(url).toContain('execution_request_id=algexec_1');
    expect(url).toContain('algorithm_result_id=algres_1');
    expect(url).toContain('status=reviewed');
    expect(url).toContain('severity=medium');
    expect(url).toContain('correlation_id=corr_1');
    // limit is coupled to the proposal list (sent), not the endpoint's 200 default.
    expect(url).toContain('limit=50');
    // Preflight-only defaults are sent explicitly so the request is self-describing.
    expect(url).toContain('min_reviewed_ratio=1');
    expect(url).toContain('policy_version=materialization_preflight.v1');
    expect(fetchMock.mock.calls[0][1].headers['Authorization']).toBe('Bearer jwt-abc');
  });

  it('omits unset filters, defaults tenant, and still couples limit + preflight defaults', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ algorithm_signal_materialization_preflight: {} }));
    vi.stubGlobal('fetch', fetchMock);

    await api.getAlgorithmSignalMaterializationPreflight({});

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('tenant_id=tenant-local');
    expect(url).not.toContain('status=');
    expect(url).not.toContain('severity=');
    // limit couples to the proposal-list default of 50 (not the endpoint's 200).
    expect(url).toContain('limit=50');
    expect(url).toContain('min_reviewed_ratio=1');
    expect(url).toContain('policy_version=materialization_preflight.v1');
  });

  it('forwards explicit min_reviewed_ratio and policy_version overrides', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ algorithm_signal_materialization_preflight: {} }));
    vi.stubGlobal('fetch', fetchMock);

    await api.getAlgorithmSignalMaterializationPreflight({
      tenant_id: 'tenant-local',
      min_reviewed_ratio: 0.5,
      policy_version: 'materialization_preflight.v2',
    });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('min_reviewed_ratio=0.5');
    expect(url).toContain('policy_version=materialization_preflight.v2');
  });

  it('parses the preflight envelope (counts, items, statuses, reason maps, coverage)', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({
        algorithm_signal_materialization_preflight: {
          tenant_id: 'tenant-local',
          policy_version: 'materialization_preflight.v1',
          total_proposals: 4,
          eligible_count: 0,
          duplicate_risk_count: 1,
          blocked_count: 2,
          invalid_count: 1,
          would_write_count: 0,
          reviewed_ratio: 0.75,
          min_reviewed_ratio: 1,
          review_coverage_satisfied: false,
          high_critical_unreviewed_count: 1,
          global_blocking_reasons: { review_coverage_below_threshold: 1, high_critical_unreviewed_proposals: 1 },
          item_reason_counts: { unreviewed_proposal: 1, duplicate_signal_event_overlap: 1, missing_source_events: 1 },
          items: [
            {
              proposal_id: 'algsigprop-reviewed',
              algorithm_result_id: 'algres-reviewed',
              algorithm_id: 'signalops.algorithms.zscore_anomaly_v1',
              execution_request_id: 'algexec-1',
              proposed_signal_type: 'signalops.algorithm.anomaly_candidate',
              status: 'reviewed',
              severity: 'medium',
              confidence: 0.9,
              preflight_status: 'blocked',
              reasons: [],
              duplicate_signal_ids: [],
              source_event_ids: ['evt-1'],
              would_write: false,
              materialization_policy: 'materialization_preflight.v1',
            },
          ],
        },
      }),
    );
    vi.stubGlobal('fetch', fetchMock);

    const p = await api.getAlgorithmSignalMaterializationPreflight({ tenant_id: 'tenant-local' });
    const env = p.algorithm_signal_materialization_preflight;
    expect(env.total_proposals).toBe(4);
    expect(env.blocked_count).toBe(2);
    expect(env.review_coverage_satisfied).toBe(false);
    expect(env.high_critical_unreviewed_count).toBe(1);
    expect(env.global_blocking_reasons.review_coverage_below_threshold).toBe(1);
    expect(env.item_reason_counts.missing_source_events).toBe(1);
    expect(env.items[0].proposal_id).toBe('algsigprop-reviewed');
    expect(env.items[0].preflight_status).toBe('blocked');
    expect(env.items[0].source_event_ids).toEqual(['evt-1']);
  });
});

describe('algorithm signal materialization API client (G123)', () => {
  it('posts the materialization body to the materializations path with bearer and no actor header', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ algorithm_signal_materialization: { materialization_id: 'algmat_1', materialization_status: 'succeeded' } }),
    );
    vi.stubGlobal('fetch', fetchMock);

    await api.materializeAlgorithmSignalProposal('algsigprop_1', {
      tenant_id: 'tenant-local',
      policy_version: 'algorithm_materialization.v1',
      metadata: { note: 'ship it' },
    });

    const call = fetchMock.mock.calls[0];
    expect(String(call[0])).toContain('/v1/algorithms/signal-proposals/algsigprop_1/materializations');
    expect(call[1].method).toBe('POST');
    expect(call[1].headers['Authorization']).toBe('Bearer jwt-abc');
    // The gateway derives the actor from the JWT (no operator-local header).
    expect(call[1].headers['X-SignalOps-Actor']).toBeUndefined();
    const body = JSON.parse(call[1].body as string);
    expect(body).toEqual({
      tenant_id: 'tenant-local',
      policy_version: 'algorithm_materialization.v1',
      metadata: { note: 'ship it' },
    });
  });

  it('builds the materializations list path with filters + tenant + limit + bearer', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ algorithm_signal_materializations: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.listAlgorithmSignalMaterializations({
      tenant_id: 'tenant-local',
      proposal_id: 'algsigprop_1',
      status: 'succeeded',
      signal_id: 'sig_alg_1',
      limit: 25,
    });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain('/v1/algorithms/signal-materializations');
    expect(url).toContain('tenant_id=tenant-local');
    expect(url).toContain('proposal_id=algsigprop_1');
    expect(url).toContain('status=succeeded');
    expect(url).toContain('signal_id=sig_alg_1');
    expect(url).toContain('limit=25');
    expect(fetchMock.mock.calls[0][1].headers['Authorization']).toBe('Bearer jwt-abc');
  });

  it('parses the materialization + list envelopes', async () => {
    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } });
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        jsonResponse({
          algorithm_signal_materialization: {
            materialization_id: 'algmat_1',
            materialization_status: 'succeeded',
            signal_id: 'sig_alg_1',
          },
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          algorithm_signal_materializations: [
            { materialization_id: 'algmat_1', materialization_status: 'duplicate', duplicate_of_signal_id: 'sig_existing' },
          ],
        }),
      );
    vi.stubGlobal('fetch', fetchMock);

    const one = await api.materializeAlgorithmSignalProposal('algsigprop_1', { tenant_id: 'tenant-local' });
    const list = await api.listAlgorithmSignalMaterializations({ tenant_id: 'tenant-local', proposal_id: 'algsigprop_1' });

    expect(one.algorithm_signal_materialization.materialization_status).toBe('succeeded');
    expect(one.algorithm_signal_materialization.signal_id).toBe('sig_alg_1');
    expect(list.algorithm_signal_materializations[0].materialization_status).toBe('duplicate');
    expect(list.algorithm_signal_materializations[0].duplicate_of_signal_id).toBe('sig_existing');
  });
});

describe('applyMaterializeAlgorithmSignalProposalResult (G123 mutation invalidation)', () => {
  it('invalidates materialization ledger, preflight, and proposal prefixes only', () => {
    const queryClient = new QueryClient();
    const invSpy = vi.spyOn(queryClient, 'invalidateQueries');
    applyMaterializeAlgorithmSignalProposalResult(queryClient);

    const invalidated = invSpy.mock.calls.map((c) => c[0]);
    expect(invalidated).toContainEqual({ queryKey: ['algorithm-signal-materializations'] });
    expect(invalidated).toContainEqual({ queryKey: ['algorithm-signal-materialization-preflight'] });
    expect(invalidated).toContainEqual({ queryKey: ['algorithm-signal-proposals'] });
    expect(invalidated).toContainEqual({ queryKey: ['algorithm-signal-proposal-summary'] });
    // No production signal/alert/insight/execution queries are touched.
    expect(invalidated).not.toContainEqual({ queryKey: ['signals'] });
    expect(invalidated).not.toContainEqual({ queryKey: ['alerts'] });
  });
});
