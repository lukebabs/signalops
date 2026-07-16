import { describe, expect, it } from 'vitest';
import {
  summarizeAlgorithmDefinition,
  summarizeAlgorithmExecutionRequest,
  summarizeAlgorithmResult,
  summarizeAlgorithmExecutionSummary,
  summarizeAlgorithmSignalProposal,
  summarizeAlgorithmSignalProposalSummary,
  summarizeAlgorithmSignalMaterializationPreflight,
  summarizeAlgorithmSignalMaterialization,
  algorithmSeverityCountEntries,
  algorithmCountEntries,
  algorithmDefinitionStatusStyle,
  algorithmExecutionStatusStyle,
  algorithmSeverityStyle,
  algorithmProposalStatusStyle,
  algorithmPreflightStatusStyle,
  algorithmMaterializationStatusStyle,
  findPreflightItemByProposalId,
  describeMaterializationEligibility,
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

describe('algorithmPreflightStatusStyle (G119)', () => {
  it('tones eligible neutrally, warns on duplicate_risk/blocked, errors on invalid', () => {
    // eligible is deliberately neutral (slate) — never a deploy/accept green.
    expect(algorithmPreflightStatusStyle('eligible')).toContain('slate');
    expect(algorithmPreflightStatusStyle('eligible')).not.toContain('emerald');
    expect(algorithmPreflightStatusStyle('duplicate_risk')).toContain('amber');
    expect(algorithmPreflightStatusStyle('blocked')).toContain('orange');
    expect(algorithmPreflightStatusStyle('invalid')).toContain('red');
    // Unknown future tokens fall back to neutral gray.
    expect(algorithmPreflightStatusStyle('future')).toContain('gray-600');
  });
});

describe('summarizeAlgorithmSignalMaterializationPreflight (G119)', () => {
  it('normalizes scalars/bools/arrays and orders reason maps by count desc then token asc', () => {
    const v = summarizeAlgorithmSignalMaterializationPreflight({
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
      global_blocking_reasons: { high_critical_unreviewed_proposals: 1, review_coverage_below_threshold: 1 },
      item_reason_counts: { missing_source_events: 2, unreviewed_proposal: 2, duplicate_signal_event_overlap: 1 },
      items: [
        {
          proposal_id: 'algsigprop-1',
          preflight_status: 'blocked',
          reasons: ['unreviewed_proposal'],
          duplicate_signal_ids: [],
          source_event_ids: ['evt-1'],
          would_write: false,
          confidence: 0.9,
        },
        {
          proposal_id: 'algsigprop-2',
          preflight_status: 'duplicate_risk',
          reasons: ['duplicate_signal_event_overlap'],
          duplicate_signal_ids: ['sig-1', 'sig-2'],
          source_event_ids: [],
          would_write: true,
          confidence: 0.8,
        },
      ],
    });
    expect(v.totalProposals).toBe(4);
    expect(v.blockedCount).toBe(2);
    expect(v.duplicateRiskCount).toBe(1);
    expect(v.reviewCoverageSatisfied).toBe(false);
    expect(v.minReviewedRatio).toBeCloseTo(1);
    expect(v.highCriticalUnreviewedCount).toBe(1);
    // Reason maps ordered count desc then token asc. Two reasons tie at 2 -> token asc.
    expect(v.itemReasonCounts.map((e) => e.key)).toEqual(['missing_source_events', 'unreviewed_proposal', 'duplicate_signal_event_overlap']);
    expect(v.itemReasonCounts[0]).toEqual({ key: 'missing_source_events', count: 2 });
    expect(v.globalBlockingReasons.map((e) => e.key)).toEqual([
      'high_critical_unreviewed_proposals',
      'review_coverage_below_threshold',
    ]);
    expect(v.items).toHaveLength(2);
    expect(v.items[0].preflightStatus).toBe('blocked');
    expect(v.items[0].reasons).toEqual(['unreviewed_proposal']);
    expect(v.items[1].duplicateSignalIds).toEqual(['sig-1', 'sig-2']);
    expect(v.items[1].wouldWrite).toBe(true);
  });

  it('collapses non-object payloads and empty reason maps to empty values', () => {
    const v = summarizeAlgorithmSignalMaterializationPreflight(null);
    expect(v.totalProposals).toBe(0);
    expect(v.reviewCoverageSatisfied).toBe(false);
    expect(v.itemReasonCounts).toEqual([]);
    expect(v.globalBlockingReasons).toEqual([]);
    expect(v.items).toEqual([]);

    const v2 = summarizeAlgorithmSignalMaterializationPreflight({ total_proposals: 0, items: 'nope' });
    expect(v2.totalProposals).toBe(0);
    expect(v2.items).toEqual([]);
  });
});

describe('algorithmMaterializationStatusStyle (G123)', () => {
  it('tones succeeded/duplicate/blocked/failed/in-progress/superseded and falls back for unknown', () => {
    expect(algorithmMaterializationStatusStyle('succeeded')).toContain('emerald');
    expect(algorithmMaterializationStatusStyle('duplicate')).toContain('amber');
    expect(algorithmMaterializationStatusStyle('blocked')).toContain('orange');
    expect(algorithmMaterializationStatusStyle('failed')).toContain('red');
    expect(algorithmMaterializationStatusStyle('requested')).toContain('blue');
    expect(algorithmMaterializationStatusStyle('running')).toContain('blue');
    expect(algorithmMaterializationStatusStyle('superseded')).toContain('gray');
    expect(algorithmMaterializationStatusStyle('future')).toContain('gray-600');
  });
});

describe('summarizeAlgorithmSignalMaterialization (G123)', () => {
  it('normalizes scalars, omitempty timestamps, and passes JSON through', () => {
    const v = summarizeAlgorithmSignalMaterialization({
      materialization_id: 'algmat_1',
      tenant_id: 'tenant-local',
      proposal_id: 'algsigprop_1',
      algorithm_id: 'zscore_anomaly_v1',
      algorithm_version: '0.1.0',
      proposed_signal_type: 'signalops.algorithm.anomaly_candidate',
      signal_id: 'sig_alg_1',
      materialization_status: 'succeeded',
      materialization_policy_version: 'algorithm_materialization.v1',
      idempotency_key: 'algmat_idem_abc',
      duplicate_of_signal_id: '',
      requested_by: 'operator-1',
      requested_at: '2026-07-16T00:00:00Z',
      completed_at: '2026-07-16T00:00:05Z',
      error_code: '',
      error_message: '',
      request_metadata: { note: 'ok' },
      preflight_snapshot: { eligible: true },
      signal_payload_preview: { signal_type: 'signalops.algorithm.anomaly_candidate' },
      created_at: '2026-07-16T00:00:00Z',
      updated_at: '2026-07-16T00:00:05Z',
    });
    expect(v.materializationId).toBe('algmat_1');
    expect(v.materializationStatus).toBe('succeeded');
    expect(v.signalId).toBe('sig_alg_1');
    expect(v.completedAt).toBe('2026-07-16T00:00:05Z');
    expect(v.failedAt).toBe('');
    expect(v.requestMetadata).toEqual({ note: 'ok' });
    expect(v.preflightSnapshot).toEqual({ eligible: true });
  });

  it('collapses non-object payloads to the empty view', () => {
    const v = summarizeAlgorithmSignalMaterialization(null);
    expect(v.materializationId).toBe('');
    expect(v.materializationStatus).toBe('');
    expect(v.requestMetadata).toEqual({});
    expect(v.signalPayloadPreview).toEqual({});
  });
});

describe('findPreflightItemByProposalId (G123)', () => {
  const view = summarizeAlgorithmSignalMaterializationPreflight({
    total_proposals: 2,
    items: [
      { proposal_id: 'algsigprop-1', preflight_status: 'eligible', would_write: true },
      { proposal_id: 'algsigprop-2', preflight_status: 'blocked', would_write: false },
    ],
  });

  it('returns the matching item and null when absent', () => {
    expect(findPreflightItemByProposalId(view, 'algsigprop-1')?.preflightStatus).toBe('eligible');
    expect(findPreflightItemByProposalId(view, 'missing')).toBeNull();
    expect(findPreflightItemByProposalId(null, 'algsigprop-1')).toBeNull();
    expect(findPreflightItemByProposalId(view, '')).toBeNull();
  });
});

describe('describeMaterializationEligibility (G123)', () => {
  const eligibleItem = summarizeAlgorithmSignalMaterializationPreflight({
    items: [{ proposal_id: 'algsigprop-1', preflight_status: 'eligible', would_write: true }],
  }).items[0];

  const baseInput = {
    proposalStatus: 'reviewed',
    preflightItem: eligibleItem,
    preflightLoading: false,
    preflightFailed: false,
    globalBlockingActive: false,
    canMutate: true,
    mutationPending: false,
    hasRecordedMaterialization: false,
  };

  it('allows a reviewed eligible would-write proposal for an operator', () => {
    expect(describeMaterializationEligibility(baseInput)).toEqual({ canMaterialize: true, reason: '' });
  });

  it('disables without operator/admin role', () => {
    const r = describeMaterializationEligibility({ ...baseInput, canMutate: false });
    expect(r.canMaterialize).toBe(false);
    expect(r.reason).toContain('operator');
  });

  it('disables while a mutation is pending', () => {
    expect(describeMaterializationEligibility({ ...baseInput, mutationPending: true }).canMaterialize).toBe(false);
  });

  it('disables when preflight is loading or failed', () => {
    expect(describeMaterializationEligibility({ ...baseInput, preflightLoading: true }).canMaterialize).toBe(false);
    expect(describeMaterializationEligibility({ ...baseInput, preflightFailed: true }).canMaterialize).toBe(false);
  });

  it('disables when there is no matching preflight item', () => {
    expect(describeMaterializationEligibility({ ...baseInput, preflightItem: null }).canMaterialize).toBe(false);
  });

  it('disables when the proposal is not reviewed', () => {
    const r = describeMaterializationEligibility({ ...baseInput, proposalStatus: 'proposed' });
    expect(r.canMaterialize).toBe(false);
    expect(r.reason).toContain('reviewed');
  });

  it('disables when a materialization is already recorded', () => {
    const r = describeMaterializationEligibility({ ...baseInput, hasRecordedMaterialization: true });
    expect(r.canMaterialize).toBe(false);
    expect(r.reason).toContain('Already materialized');
  });

  it('disables on global blockers', () => {
    expect(describeMaterializationEligibility({ ...baseInput, globalBlockingActive: true }).canMaterialize).toBe(false);
  });

  it('disables when the preflight item is not eligible or would not write', () => {
    const blockedItem = { ...eligibleItem, preflightStatus: 'blocked', wouldWrite: true };
    expect(describeMaterializationEligibility({ ...baseInput, preflightItem: blockedItem }).canMaterialize).toBe(false);
    const noWriteItem = { ...eligibleItem, preflightStatus: 'eligible', wouldWrite: false };
    expect(describeMaterializationEligibility({ ...baseInput, preflightItem: noWriteItem }).canMaterialize).toBe(false);
  });
});
