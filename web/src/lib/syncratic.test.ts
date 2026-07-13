import { describe, expect, it } from 'vitest';
import {
  summarizeSyncraticInsight,
  summarizeSyncraticContextWindow,
  summarizeSyncraticMaterialization,
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
