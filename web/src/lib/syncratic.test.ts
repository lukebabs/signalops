import { describe, expect, it } from 'vitest';
import {
  summarizeSyncraticInsight,
  summarizeSyncraticContextWindow,
  summarizeSyncraticMaterialization,
  summarizeSyncraticAsk,
  summarizeSyncraticAskRouteResult,
  detectSyncraticDataQualityWarning,
  classifySyncraticInsightBadge,
  classifySyncraticAskError,
  messageForSyncraticAskError,
  SYNCRATIC_ASK_BADGE_LABELS,
  syncraticSeverityStyle,
  syncraticInsightStatusStyle,
  shortSyncraticId,
} from './syncratic';

describe('summarizeSyncraticInsight (G088)', () => {
  it('reads fields and distinguishes alert vs signal evidence counts', () => {
    const s = summarizeSyncraticInsight({
      syncratic_insight_id: 'synins-1',
      context_window_id: 'synctx-1',
      insight_type: 'marketops.syncratic.multi_event_context',
      subject_symbol: 'AAPL',
      subject_type: 'ticker',
      subject_id: 'AAPL',
      status: 'active',
      severity: 'medium',
      confidence: 0.75,
      title: 'AAPL Syncratic context',
      summary: '2 supporting signals and 1 supporting alert',
      explanation: 'deterministic window',
      builder_version: 'syncratic.context_builder.v1',
      supporting_alert_ids: ['alert-1'],
      supporting_signal_ids: ['sig-1', 'sig-2'],
      supporting_event_ids: ['evt-1'],
      supporting_artifact_ids: [],
      related_graph_proposal_ids: ['graphprop-1'],
      related_label_ids: [],
      created_at: '2026-07-13T00:00:00Z',
      updated_at: '2026-07-13T00:00:00Z',
    });
    expect(s.insightId).toBe('synins-1');
    expect(s.contextWindowId).toBe('synctx-1');
    expect(s.subjectSymbol).toBe('AAPL');
    expect(s.alertCount).toBe(1);
    expect(s.signalCount).toBe(2);
    expect(s.graphProposalCount).toBe(1);
    // Alert and signal counts are distinct, never merged.
    expect(s.alertCount).not.toBe(s.signalCount);
    expect(s.supportingSignalIds).toEqual(['sig-1', 'sig-2']);
  });

  it('tolerates a non-object / partial payload without throwing', () => {
    expect(summarizeSyncraticInsight(undefined).signalCount).toBe(0);
    expect(summarizeSyncraticInsight(null).insightId).toBe('');
    expect(summarizeSyncraticInsight('nope').alertCount).toBe(0);
    const s = summarizeSyncraticInsight({ supporting_signal_ids: 'oops', supporting_alert_ids: [1, 2, 'alert-1'] });
    expect(s.signalCount).toBe(0);
    expect(s.alertCount).toBe(1);
  });
});

describe('summarizeSyncraticContextWindow (G088)', () => {
  it('surfaces digest, builder version, strategy, window, and evidence counts', () => {
    const w = summarizeSyncraticContextWindow({
      context_window_id: 'synctx-1',
      context_strategy: 'symbol_signal_cluster_5d',
      context_builder_version: 'syncratic.context_builder.v1',
      window_start: '2026-07-01T00:00:00Z',
      window_end: '2026-07-14T00:00:00Z',
      evidence_digest: 'sha256:abc',
      idempotency_key: 'tenant-local|daily_market_surveillance|AAPL',
      signal_types: ['marketops.dsm.volatility_expansion'],
      detector_ids: ['marketops.dsm.taxonomy_v1'],
      event_ids: ['evt-1'],
      signal_ids: ['sig-1', 'sig-2'],
      alert_ids: ['alert-1'],
      artifact_ids: [],
      graph_proposal_ids: ['graphprop-1'],
      label_ids: [],
    });
    expect(w.contextStrategy).toBe('symbol_signal_cluster_5d');
    expect(w.contextBuilderVersion).toBe('syncratic.context_builder.v1');
    expect(w.windowStart).toBe('2026-07-01T00:00:00Z');
    expect(w.windowEnd).toBe('2026-07-14T00:00:00Z');
    expect(w.evidenceDigest).toBe('sha256:abc');
    expect(w.idempotencyKey).toContain('AAPL');
    expect(w.signalCount).toBe(2);
    expect(w.alertCount).toBe(1);
  });

  it('tolerates missing/unknown JSON fields', () => {
    const w = summarizeSyncraticContextWindow({ summary_metrics: { yield: 0.3 }, baseline_refs: [] });
    expect(w.contextWindowId).toBe('');
    expect(w.signalCount).toBe(0);
  });
});

describe('summarizeSyncraticMaterialization (G088)', () => {
  it('classifies below-threshold + unchanged + budget-cap skips as non-error outcomes', () => {
    const counters = summarizeSyncraticMaterialization({
      scanned_assets: 5,
      candidate_windows: 1,
      materialized_context_windows: 1,
      materialized_insights: 1,
      skipped_below_threshold: 4,
      skipped_unchanged: 0,
      skipped_budget_cap: 0,
    });
    const byKey = Object.fromEntries(counters.map((c) => [c.key, c]));
    expect(byKey.scanned_assets.kind).toBe('scanned');
    expect(byKey.materialized_insights.kind).toBe('materialized');
    // Skips are normal outcomes — kind 'skipped', never an error flag.
    expect(byKey.skipped_below_threshold.kind).toBe('skipped');
    expect(byKey.skipped_below_threshold.value).toBe(4);
    expect(byKey.skipped_unchanged.kind).toBe('skipped');
    expect(byKey.skipped_budget_cap.kind).toBe('skipped');
    // The full set of kinds is exactly the three non-error classifications.
    expect(new Set(counters.map((c) => c.kind))).toEqual(new Set(['scanned', 'materialized', 'skipped']));
  });

  it('tolerates a non-object result with an empty counter list', () => {
    expect(summarizeSyncraticMaterialization(undefined)).toEqual([]);
    expect(summarizeSyncraticMaterialization(null)).toEqual([]);
    expect(summarizeSyncraticMaterialization('nope')).toEqual([]);
  });
});

describe('status/severity display helpers (G088)', () => {
  it('resolves styles for known severities and falls back for unknown ones', () => {
    expect(syncraticSeverityStyle('critical')).toContain('red');
    expect(syncraticSeverityStyle('high')).toContain('orange');
    expect(syncraticSeverityStyle('medium')).toContain('amber');
    expect(syncraticSeverityStyle('unknown_future')).toContain('gray');
  });

  it('resolves styles for known insight statuses and falls back for unknown ones', () => {
    expect(syncraticInsightStatusStyle('active')).toContain('blue');
    expect(syncraticInsightStatusStyle('reviewed')).toContain('green');
    expect(syncraticInsightStatusStyle('superseded')).toContain('violet');
    expect(syncraticInsightStatusStyle('unknown_future')).toContain('gray');
  });
});

describe('shortSyncraticId (G088)', () => {
  it('returns the last underscore segment for compact table cells', () => {
    expect(shortSyncraticId('synctx_abc123def')).toBe('abc123def');
    expect(shortSyncraticId('synins_9dd57597915529ef')).toBe('9dd57597915529ef');
    expect(shortSyncraticId('noseparator')).toBe('noseparator');
  });
});

// --- G090 Syncratic Ask enrichment helpers ---------------------------------

const ASK_METRICS = {
  metrics: {
    syncratic_ask: {
      enabled: true,
      ask_query_id: 'ask-1',
      ask_status: 'completed',
      prompt_builder_version: 'marketops.syncratic.ask_prompt.v1',
      prompt_digest: 'sha256:abc',
      context_window_id: 'synctx_1',
      context_evidence_digest: 'digest',
      request_scope: 'tenant',
      request_k: 1,
      direct_reasoning: true,
      graph_enabled: false,
      kee_enabled: false,
      prompt_bytes: 9709,
      caps: {},
      response: { confidence: 0.82, evidence_count: 3, citation_count: 2 },
      latency_ms: 1234,
    },
  },
};

describe('summarizeSyncraticAsk (G090)', () => {
  it('reads the persisted Ask metadata scalars without throwing', () => {
    const a = summarizeSyncraticAsk(ASK_METRICS);
    expect(a.present).toBe(true);
    expect(a.askQueryId).toBe('ask-1');
    expect(a.askStatus).toBe('completed');
    expect(a.promptBuilderVersion).toBe('marketops.syncratic.ask_prompt.v1');
    expect(a.directReasoning).toBe(true);
    expect(a.graphEnabled).toBe(false);
    expect(a.keeEnabled).toBe(false);
    expect(a.promptBytes).toBe(9709);
    expect(a.latencyMs).toBe(1234);
    expect(a.responseConfidence).toBeCloseTo(0.82);
    expect(a.responseEvidenceCount).toBe(3);
    expect(a.responseCitationCount).toBe(2);
  });

  it('reports present:false for a deterministic insight (no Ask metadata)', () => {
    const a = summarizeSyncraticAsk({ metrics: {} });
    expect(a.present).toBe(false);
    expect(a.askStatus).toBe('');
  });

  it('tolerates missing/malformed payloads', () => {
    expect(summarizeSyncraticAsk(undefined).present).toBe(false);
    expect(summarizeSyncraticAsk(null).present).toBe(false);
    expect(summarizeSyncraticAsk('nope').present).toBe(false);
    // response nested under a non-object collapses to 0s, not a throw.
    const a = summarizeSyncraticAsk({ metrics: { syncratic_ask: { ask_status: 'completed', response: 'oops' } } });
    expect(a.present).toBe(true);
    expect(a.responseConfidence).toBe(0);
  });
});

describe('summarizeSyncraticAskRouteResult (G090)', () => {
  it('reads the ask_result envelope and flags a skip', () => {
    const r = summarizeSyncraticAskRouteResult({
      context_window_id: 'synctx_1',
      syncratic_insight_id: 'synins_1',
      ask_query_id: '',
      ask_status: 'skipped',
      prompt_digest: 'sha256:abc',
      updated: false,
      skipped_reason: 'unchanged_prompt_and_evidence',
      prompt_builder_version: 'marketops.syncratic.ask_prompt.v1',
    });
    expect(r.askStatus).toBe('skipped');
    expect(r.updated).toBe(false);
    expect(r.skippedReason).toBe('unchanged_prompt_and_evidence');
  });

  it('tolerates a non-object result', () => {
    const r = summarizeSyncraticAskRouteResult(null);
    expect(r.updated).toBe(false);
    expect(r.askStatus).toBe('');
  });
});

describe('detectSyncraticDataQualityWarning (G090)', () => {
  it('matches "data quality warning" case-insensitively', () => {
    expect(
      detectSyncraticDataQualityWarning({ explanation: 'DATA QUALITY WARNING: evidence mismatch' }),
    ).toBe(true);
    expect(detectSyncraticDataQualityWarning({ title: 'Data Quality warning' })).toBe(true);
  });

  it('matches "subject mismatch" and "does not support"', () => {
    expect(detectSyncraticDataQualityWarning({ summary: 'Subject mismatch detected' })).toBe(true);
    expect(
      detectSyncraticDataQualityWarning({ explanation: 'evidence does not support the context subject' }),
    ).toBe(true);
  });

  it('does not flag a clean market thesis and does not infer from symbol alone', () => {
    expect(
      detectSyncraticDataQualityWarning({ subject_symbol: 'AAPL', explanation: 'AAPL shows bounded volatility worth review.' }),
    ).toBe(false);
    // Symbol present but no warning language → not a data-quality block.
    expect(detectSyncraticDataQualityWarning({ subject_symbol: 'MSFT' })).toBe(false);
  });
});

describe('classifySyncraticInsightBadge (G090)', () => {
  it('returns deterministic when no Ask metadata is present', () => {
    expect(classifySyncraticInsightBadge({ metrics: {} })).toBe('deterministic');
  });

  it('returns ask_completed when metrics.syncratic_ask.ask_status is completed', () => {
    expect(classifySyncraticInsightBadge(ASK_METRICS)).toBe('ask_completed');
  });

  it('returns ask_skipped from the latest transient route result', () => {
    // Persisted state is ask_completed, but the latest action skipped.
    expect(classifySyncraticInsightBadge(ASK_METRICS, 'skipped')).toBe('ask_skipped');
  });

  it('overrides to data_quality when warning language is present, even if Ask completed', () => {
    const blocked = { ...ASK_METRICS, explanation: 'DATA QUALITY WARNING' };
    expect(classifySyncraticInsightBadge(blocked)).toBe('data_quality');
  });

  it('labels every badge kind for the chip renderer', () => {
    expect(SYNCRATIC_ASK_BADGE_LABELS.deterministic).toBe('Deterministic');
    expect(SYNCRATIC_ASK_BADGE_LABELS.ask_completed).toBe('Ask completed');
    expect(SYNCRATIC_ASK_BADGE_LABELS.ask_skipped).toBe('Ask skipped');
    expect(SYNCRATIC_ASK_BADGE_LABELS.data_quality).toBe('Data Quality Warning');
  });
});

describe('classifySyncraticAskError / messageForSyncraticAskError (G090)', () => {
  it('maps each backend code to a sanitized kind', () => {
    expect(classifySyncraticAskError(0, 'network_error')).toBe('network');
    expect(classifySyncraticAskError(401, 'unauthorized')).toBe('auth');
    expect(classifySyncraticAskError(400, 'empty_context_window')).toBe('empty');
    expect(classifySyncraticAskError(400, 'syncratic_ask_invalid')).toBe('invalid');
    expect(classifySyncraticAskError(404, 'context_window_not_found')).toBe('not_found');
    expect(classifySyncraticAskError(503, 'syncratic_ask_unavailable')).toBe('unavailable');
    expect(classifySyncraticAskError(502, 'syncratic_ask_failed')).toBe('failed');
    expect(classifySyncraticAskError(500, 'syncratic_ask_failed')).toBe('failed');
    expect(classifySyncraticAskError(418, 'im_a_teapot')).toBe('unknown');
  });

  it('renders the verbatim empty_context_window operator copy', () => {
    expect(messageForSyncraticAskError({ status: 400, code: 'empty_context_window' })).toBe(
      'No pure supporting evidence exists for this context subject. Review signal/entity mapping or rematerialize after evidence is corrected.',
    );
  });

  it('sanitizes 502 failures — never surfaces upstream bodies', () => {
    const msg = messageForSyncraticAskError({ status: 502, code: 'syncratic_ask_failed' });
    expect(msg).toBe('Syncratic Ask failed. Upstream details are not exposed; retry or review gateway logs.');
    expect(msg).not.toContain('portal.syncratic');
    expect(msg).not.toContain('upstream');
  });

  it('falls back to the unknown message for non-ApiError shapes', () => {
    expect(messageForSyncraticAskError(new Error('boom'))).toContain('unexpectedly');
  });
});
