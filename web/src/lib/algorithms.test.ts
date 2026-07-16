import { describe, expect, it } from 'vitest';
import {
  summarizeAlgorithmDefinition,
  summarizeAlgorithmExecutionRequest,
  summarizeAlgorithmResult,
  summarizeAlgorithmExecutionSummary,
  summarizeAlgorithmSignalProposal,
  summarizeAlgorithmSignalProposalSummary,
  algorithmSeverityCountEntries,
  algorithmCountEntries,
  algorithmDefinitionStatusStyle,
  algorithmExecutionStatusStyle,
  algorithmSeverityStyle,
  algorithmProposalStatusStyle,
} from './algorithms';

describe('summarizeAlgorithmDefinition (G109)', () => {
  it('reads scalar + array fields without throwing', () => {
    const s = summarizeAlgorithmDefinition({
      algorithm_id: 'zscore_v1',
      name: 'Z-Score anomaly',
      algorithm_type: 'anomaly',
      runtime_type: 'python_plugin',
      input_features: ['ret', 'vol'],
      input_event_types: ['price'],
      version: '1.0.0',
      status: 'active',
      created_at: '2026-07-14T00:00:00Z',
      updated_at: '2026-07-14T00:00:00Z',
    });
    expect(s.algorithmId).toBe('zscore_v1');
    expect(s.algorithmType).toBe('anomaly');
    expect(s.runtimeType).toBe('python_plugin');
    expect(s.inputFeatures).toEqual(['ret', 'vol']);
    expect(s.inputEventTypes).toEqual(['price']);
    expect(s.status).toBe('active');
  });

  it('collapses non-object payloads to the empty summary', () => {
    expect(summarizeAlgorithmDefinition(null).algorithmId).toBe('');
    expect(summarizeAlgorithmDefinition('nope').inputFeatures).toEqual([]);
  });
});

describe('summarizeAlgorithmExecutionRequest (G109)', () => {
  it('reads request fields and id arrays', () => {
    const s = summarizeAlgorithmExecutionRequest({
      execution_request_id: 'algexec_1',
      algorithm_id: 'zscore_v1',
      algorithm_version: '1.0.0',
      feature_refs: ['f1'],
      entity_refs: ['AAPL'],
      window_ref: 'win_1',
      correlation_id: 'corr_1',
      status: 'succeeded',
      requested_by: 'operator-local',
    });
    expect(s.executionRequestId).toBe('algexec_1');
    expect(s.status).toBe('succeeded');
    expect(s.featureRefs).toEqual(['f1']);
    expect(s.windowRef).toBe('win_1');
  });
});

describe('summarizeAlgorithmResult (G109)', () => {
  it('reads score/confidence/severity and lineage id arrays', () => {
    const s = summarizeAlgorithmResult({
      algorithm_result_id: 'algres_1',
      result_type: 'anomaly',
      score: 2.5,
      confidence: 0.9,
      severity: 'high',
      source_event_ids: ['evt_1', 'evt_2'],
      feature_value_ids: ['fv_1'],
      evidence_refs: ['ev_1'],
    });
    expect(s.algorithmResultId).toBe('algres_1');
    expect(s.score).toBeCloseTo(2.5);
    expect(s.severity).toBe('high');
    expect(s.sourceEventIds).toHaveLength(2);
  });
});

describe('algorithmSeverityCountEntries (G109)', () => {
  it('orders severities critical→info with unknown last', () => {
    const entries = algorithmSeverityCountEntries({ info: 2, critical: 1, medium: 3, weird: 5 });
    expect(entries.map((e) => e.severity)).toEqual(['critical', 'medium', 'info', 'weird']);
    expect(entries[0].count).toBe(1);
  });

  it('tolerates a non-object map', () => {
    expect(algorithmSeverityCountEntries(null)).toEqual([]);
    expect(algorithmSeverityCountEntries('nope')).toEqual([]);
  });
});

describe('summarizeAlgorithmExecutionSummary (G109)', () => {
  it('reads counts, max score/confidence, ordered severity counts, and top results', () => {
    const view = summarizeAlgorithmExecutionSummary({
      execution_request: { execution_request_id: 'algexec_1', status: 'succeeded' },
      result_count: 3,
      severity_counts: { high: 2, critical: 1 },
      max_score: 3.2,
      max_confidence: 0.95,
      top_results: [
        { algorithm_result_id: 'algres_1', score: 3.2, severity: 'critical' },
        { algorithm_result_id: 'algres_2', score: 1.1, severity: 'high' },
      ],
    });
    expect(view.resultCount).toBe(3);
    expect(view.maxScore).toBeCloseTo(3.2);
    expect(view.maxConfidence).toBeCloseTo(0.95);
    expect(view.severityCounts.map((c) => c.severity)).toEqual(['critical', 'high']);
    expect(view.topResults).toHaveLength(2);
    expect(view.topResults[0].algorithmResultId).toBe('algres_1');
    expect(view.executionRequest.status).toBe('succeeded');
  });

  it('collapses a non-object summary to empty values', () => {
    const view = summarizeAlgorithmExecutionSummary(undefined);
    expect(view.resultCount).toBe(0);
    expect(view.topResults).toEqual([]);
    expect(view.severityCounts).toEqual([]);
  });
});

describe('algorithm style helpers (G109)', () => {
  it('maps known definition statuses and falls back for unknown', () => {
    expect(algorithmDefinitionStatusStyle('active')).toContain('emerald');
    expect(algorithmDefinitionStatusStyle('draft')).toContain('amber');
    expect(algorithmDefinitionStatusStyle('future')).toContain('gray-600');
  });

  it('maps known execution statuses and falls back for unknown', () => {
    expect(algorithmExecutionStatusStyle('succeeded')).toContain('emerald');
    expect(algorithmExecutionStatusStyle('failed')).toContain('red');
    expect(algorithmExecutionStatusStyle('future')).toContain('gray-600');
  });

  it('maps known severities and falls back for unknown', () => {
    expect(algorithmSeverityStyle('critical')).toContain('red');
    expect(algorithmSeverityStyle('high')).toContain('orange');
    expect(algorithmSeverityStyle('future')).toContain('gray-600');
  });
});

describe('summarizeAlgorithmSignalProposal (G114)', () => {
  it('reads scalar, lineage, and JSON fields without throwing', () => {
    const s = summarizeAlgorithmSignalProposal({
      proposal_id: 'algsigprop_1',
      tenant_id: 'tenant-local',
      algorithm_result_id: 'algres_1',
      algorithm_id: 'ruptures_change_point_v1',
      algorithm_version: '0.1.0',
      execution_request_id: 'algexec_1',
      proposed_signal_type: 'signalops.algorithm.change_point_candidate',
      status: 'proposed',
      score: 2.5,
      confidence: 0.9,
      severity: 'critical',
      proposal_payload: { window: { start: 't0', end: 't1' } },
      rationale: { detector: 'ruptures' },
      source_event_ids: ['evt_1', 'evt_2'],
      evidence_refs: ['ev_1'],
      correlation_id: 'corr_1',
      created_by: 'operator-local',
      created_at: '2026-07-16T00:00:00Z',
      updated_at: '2026-07-16T00:00:00Z',
    });
    expect(s.proposalId).toBe('algsigprop_1');
    expect(s.proposedSignalType).toBe('signalops.algorithm.change_point_candidate');
    expect(s.status).toBe('proposed');
    expect(s.score).toBeCloseTo(2.5);
    expect(s.confidence).toBeCloseTo(0.9);
    expect(s.sourceEventIds).toEqual(['evt_1', 'evt_2']);
    expect(s.evidenceRefs).toEqual(['ev_1']);
    expect(s.proposalPayload).toEqual({ window: { start: 't0', end: 't1' } });
    expect(s.rationale).toEqual({ detector: 'ruptures' });
    // decided_at omitted on the backend until a decision is recorded.
    expect(s.decidedAt).toBe('');
    expect(s.reviewedBy).toBe('');
  });

  it('reads review metadata when a decision has been recorded', () => {
    const s = summarizeAlgorithmSignalProposal({
      proposal_id: 'algsigprop_1',
      status: 'reviewed',
      reviewed_by: 'analyst-1',
      decision_note: 'Useful evidence; no production signal materialized.',
      decided_at: '2026-07-16T01:00:00Z',
    });
    expect(s.status).toBe('reviewed');
    expect(s.reviewedBy).toBe('analyst-1');
    expect(s.decisionNote).toBe('Useful evidence; no production signal materialized.');
    expect(s.decidedAt).toBe('2026-07-16T01:00:00Z');
  });

  it('collapses non-object payloads to the empty summary', () => {
    expect(summarizeAlgorithmSignalProposal(null).proposalId).toBe('');
    expect(summarizeAlgorithmSignalProposal('nope').sourceEventIds).toEqual([]);
    expect(summarizeAlgorithmSignalProposal(42).proposalPayload).toEqual({});
  });
});

describe('algorithmProposalStatusStyle (G114)', () => {
  it('maps the four reviewable statuses and falls back for unknown', () => {
    expect(algorithmProposalStatusStyle('proposed')).toContain('blue');
    // reviewed is positive/complete tone but NOT a deploy/accept green token.
    expect(algorithmProposalStatusStyle('reviewed')).toContain('emerald');
    expect(algorithmProposalStatusStyle('rejected')).toContain('red');
    expect(algorithmProposalStatusStyle('superseded')).toContain('gray');
    // accepted is intentionally not a status; render it as neutral, not positive.
    expect(algorithmProposalStatusStyle('accepted')).toContain('gray-600');
    expect(algorithmProposalStatusStyle('future')).toContain('gray-600');
  });
});

describe('algorithmCountEntries (G116)', () => {
  it('orders by count desc then key asc and drops non-numeric values', () => {
    expect(algorithmCountEntries({ b: 2, a: 2, c: 5, d: 'x', e: null })).toEqual([
      { key: 'c', count: 5 },
      { key: 'a', count: 2 },
      { key: 'b', count: 2 },
    ]);
  });

  it('tolerates a non-object map', () => {
    expect(algorithmCountEntries(null)).toEqual([]);
    expect(algorithmCountEntries('nope')).toEqual([]);
    expect(algorithmCountEntries({})).toEqual([]);
  });
});

describe('summarizeAlgorithmSignalProposalSummary (G116)', () => {
  it('reads scalar metrics and orders breakdown counts', () => {
    const v = summarizeAlgorithmSignalProposalSummary({
      tenant_id: 'tenant-local',
      total_proposals: 5,
      proposed_count: 2,
      reviewed_count: 2,
      rejected_count: 1,
      superseded_count: 0,
      reviewed_ratio: 0.4,
      high_critical_unreviewed_count: 1,
      status_counts: { reviewed: 2, proposed: 2, rejected: 1 },
      severity_counts: { info: 3, critical: 2, high: 1 },
      proposed_signal_type_counts: { 'signalops.algorithm.change_point_candidate': 4, 'signalops.algorithm.other': 1 },
      algorithm_id_counts: { algo_a: 3, algo_b: 2 },
      reviewer_counts: { 'analyst-1': 2 },
    });
    expect(v.tenantId).toBe('tenant-local');
    expect(v.totalProposals).toBe(5);
    expect(v.reviewedRatio).toBeCloseTo(0.4);
    expect(v.highCriticalUnreviewedCount).toBe(1);
    // Generic counts: count desc, tie broken by key asc.
    expect(v.statusCounts.map((e) => e.key)).toEqual(['proposed', 'reviewed', 'rejected']);
    expect(v.statusCounts[0]).toEqual({ key: 'proposed', count: 2 });
    expect(v.proposedSignalTypeCounts[0]).toEqual({ key: 'signalops.algorithm.change_point_candidate', count: 4 });
    // Severity keeps rank ordering (critical, high, info), normalized to {key,count}.
    expect(v.severityCounts).toEqual([
      { key: 'critical', count: 2 },
      { key: 'high', count: 1 },
      { key: 'info', count: 3 },
    ]);
    expect(v.reviewerCounts).toEqual([{ key: 'analyst-1', count: 2 }]);
  });

  it('collapses non-object summaries and empty maps to empty values', () => {
    const v = summarizeAlgorithmSignalProposalSummary(null);
    expect(v.totalProposals).toBe(0);
    expect(v.reviewedRatio).toBe(0);
    expect(v.statusCounts).toEqual([]);
    expect(v.severityCounts).toEqual([]);
    expect(v.reviewerCounts).toEqual([]);

    const v2 = summarizeAlgorithmSignalProposalSummary({ total_proposals: 0, status_counts: {} });
    expect(v2.totalProposals).toBe(0);
    expect(v2.statusCounts).toEqual([]);
  });
});
