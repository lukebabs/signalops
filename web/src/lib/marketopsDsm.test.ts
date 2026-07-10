import { describe, expect, it } from 'vitest';
import type { SignalRecord } from '../types';
import {
  MARKETOPS_DSM_DETECTOR_ID,
  MARKETOPS_DSM_USE_CASE,
  MARKETOPS_DSM_SIGNAL_TYPES,
  dsmShortType,
  dsmFamily,
  getTicker,
  getMetric,
  getArtifactProposal,
  getArtifactId,
  graphTargetCounts,
  countGraphTargets,
  hasLifecycleMatch,
} from './marketopsDsm';

// Fixture mirrors the verified marketops.dsm.taxonomy_v1 payload shape.
function baseSignal(over: Partial<SignalRecord> = {}): SignalRecord {
  return {
    signal_id: 'sig-dsm-1',
    tenant_id: 'tenant-local',
    source_id: 'src-massive',
    source_adapter: 'market_data.massive',
    dataset: 'equity_eod_prices',
    detector_id: MARKETOPS_DSM_DETECTOR_ID,
    detector_version: '1',
    model_version: 'marketops-dsm-v1',
    signal_type: 'marketops.dsm.volatility_expansion',
    severity: 'high',
    confidence: 0.89,
    event_ids: ['evt-aapl-20260709'],
    window_start: '2026-07-09T13:30:00Z',
    window_end: '2026-07-09T20:00:00Z',
    entities: [{ type: 'ticker', id: 'ticker:AAPL', external_id: 'AAPL', confidence: 1 }],
    supporting_metrics: {
      open_close_move_pct: 6.0,
      intraday_range_pct: 9.0,
      vwap_distance_pct: 2.9126,
      daily_return_pct: 4.9505,
      volume: 1200000,
      open_interest: 2000,
      volume_open_interest_ratio: 0.75,
      days_to_expiration: 27,
      moneyness_pct: 5.0,
      contract_type: 'call',
      quality_issue_count: 0,
      detector_score: 0.89,
    },
    graph_targets: [
      { type: 'node_candidate', node_id: 'ticker:AAPL', labels: ['MarketAsset'], confidence: 1 },
      { type: 'node_candidate', node_id: 'signal_type:marketops.dsm.volatility_expansion', confidence: 1 },
      { type: 'relationship_candidate', from: 'ticker:AAPL', relationship: 'EXHIBITS_SIGNAL', to: 'signal_type:marketops.dsm.volatility_expansion', confidence: 0.89 },
    ],
    semantic_evidence: [
      {
        type: 'dsm_artifact_proposal',
        artifact_id: 'artifact_marketops_dsm_v1_a1b2',
        summary: 'volatility expansion threshold crossed for AAPL',
        quality_issues: [],
        artifact: {
          artifact_id: 'artifact_marketops_dsm_v1_a1b2',
          artifact_type: 'marketops.dsm.signal_artifact.v1',
          signal_type: 'marketops.dsm.volatility_expansion',
          source_event_id: 'evt-aapl-20260709',
          subject: { type: 'ticker', id: 'ticker:AAPL', symbol: 'AAPL' },
          severity: 'high',
          confidence: 0.89,
          summary: 'volatility expansion threshold crossed for AAPL',
          features: { open_close_move_pct: 6.0 },
          quality_issues: ['missing_close'],
        },
      },
    ],
    evidence: null,
    recommendation: { action: 'review_marketops_signal', artifact_ids: ['artifact_marketops_dsm_v1_a1b2'], graph_target_count: 3 },
    event: null,
    broker_topic: 'signalops.local.signal.v1',
    broker_partition: 2,
    broker_offset: 2,
    created_at: '2026-07-09T20:01:00Z',
    updated_at: '2026-07-09T20:01:00Z',
    ...over,
  };
}

describe('DSM constants (G076)', () => {
  it('exposes the detector id, use case, and the eight taxonomy types', () => {
    expect(MARKETOPS_DSM_DETECTOR_ID).toBe('marketops.dsm.taxonomy_v1');
    expect(MARKETOPS_DSM_USE_CASE).toBe('daily_market_surveillance');
    expect(MARKETOPS_DSM_SIGNAL_TYPES).toHaveLength(8);
    expect(MARKETOPS_DSM_SIGNAL_TYPES).toContain('marketops.dsm.pinning_risk');
  });
});

describe('dsmShortType / dsmFamily (G076)', () => {
  it('strips the marketops.dsm. prefix', () => {
    expect(dsmShortType('marketops.dsm.volatility_expansion')).toBe('volatility_expansion');
    expect(dsmShortType('other')).toBe('other');
  });

  it('classifies families per the taxonomy', () => {
    expect(dsmFamily('marketops.dsm.price_quality_exception')).toBe('quality');
    expect(dsmFamily('marketops.dsm.hedging_pressure')).toBe('option');
    expect(dsmFamily('marketops.dsm.speculative_put_pressure')).toBe('option');
    expect(dsmFamily('marketops.dsm.pinning_risk')).toBe('option');
    expect(dsmFamily('marketops.dsm.volatility_expansion')).toBe('equity');
    expect(dsmFamily('marketops.dsm.accumulation')).toBe('equity');
    expect(dsmFamily('marketops.dsm.divergence')).toBe('equity');
    expect(dsmFamily('marketops.dsm.unknown_thing')).toBe('unknown');
  });
});

describe('getTicker (G076)', () => {
  it('extracts the ticker from entities[0].external_id', () => {
    expect(getTicker(baseSignal())).toBe('AAPL');
  });

  it('falls back to the artifact subject symbol when entities are absent', () => {
    expect(getTicker(baseSignal({ entities: [] }))).toBe('AAPL');
  });

  it('falls back to artifact symbol when entities are malformed', () => {
    expect(getTicker(baseSignal({ entities: 'nope' as unknown }))).toBe('AAPL');
  });

  it('returns "-" when no ticker is recoverable', () => {
    expect(getTicker(baseSignal({ entities: [], semantic_evidence: [] }))).toBe('-');
  });
});

describe('getMetric (G076)', () => {
  it('reads numeric and string metrics', () => {
    const s = baseSignal();
    expect(getMetric(s, 'open_close_move_pct')).toBe(6.0);
    expect(getMetric(s, 'contract_type')).toBe('call');
  });

  it('returns null for missing/unsupported metric maps', () => {
    const s = baseSignal();
    expect(getMetric(s, 'does_not_exist')).toBeNull();
    expect(getMetric(baseSignal({ supporting_metrics: null }), 'volume')).toBeNull();
    expect(getMetric(baseSignal({ supporting_metrics: 'nope' as unknown }), 'volume')).toBeNull();
  });
});

describe('artifact proposal extraction (G076)', () => {
  it('extracts the nested artifact proposal from semantic_evidence[0]', () => {
    const p = getArtifactProposal(baseSignal());
    expect(p).not.toBeNull();
    expect(p?.artifact_id).toBe('artifact_marketops_dsm_v1_a1b2');
    expect(p?.artifact_type).toBe('marketops.dsm.signal_artifact.v1');
    expect(p?.subject?.symbol).toBe('AAPL');
    expect(p?.quality_issues).toEqual(['missing_close']);
    expect(p?.confidence).toBe(0.89);
  });

  it('getArtifactId prefers the nested artifact, then the evidence element, then recommendation', () => {
    expect(getArtifactId(baseSignal())).toBe('artifact_marketops_dsm_v1_a1b2');
    expect(getArtifactId(baseSignal({ semantic_evidence: [{ artifact_id: 'flat-id' }] as unknown }))).toBe('flat-id');
    expect(getArtifactId(baseSignal({ semantic_evidence: [] }))).toBe('artifact_marketops_dsm_v1_a1b2');
  });

  it('returns null when no artifact id is recoverable', () => {
    expect(getArtifactId(baseSignal({ semantic_evidence: [], recommendation: null }))).toBeNull();
  });
});

describe('graph target counting (G076)', () => {
  it('counts node and relationship candidates separately', () => {
    const counts = graphTargetCounts(baseSignal());
    expect(counts.nodes).toBe(2);
    expect(counts.relationships).toBe(1);
    expect(countGraphTargets(baseSignal())).toBe(3);
  });

  it('is defensive about malformed graph_targets', () => {
    expect(graphTargetCounts(baseSignal({ graph_targets: null }))).toEqual({ nodes: 0, relationships: 0 });
    expect(graphTargetCounts(baseSignal({ graph_targets: 'nope' as unknown }))).toEqual({ nodes: 0, relationships: 0 });
    expect(countGraphTargets(baseSignal({ graph_targets: [null, 5, { type: 'other' }] as unknown }))).toBe(0);
  });
});

describe('malformed payloads never throw (G076)', () => {
  it('handles every DSM field being absent/null without throwing', () => {
    const broken = baseSignal({
      entities: null,
      supporting_metrics: null,
      semantic_evidence: null,
      graph_targets: null,
      recommendation: null,
    });
    expect(() => {
      getTicker(broken);
      getMetric(broken, 'volume');
      getArtifactProposal(broken);
      getArtifactId(broken);
      graphTargetCounts(broken);
    }).not.toThrow();
    expect(getTicker(broken)).toBe('-');
    expect(getArtifactProposal(broken)).toBeNull();
  });
});

describe('hasLifecycleMatch (G076)', () => {
  it('matches by signal id against a coverage set', () => {
    const s = baseSignal();
    expect(hasLifecycleMatch(s, new Set(['sig-dsm-1']))).toBe(true);
    expect(hasLifecycleMatch(s, new Set(['sig-other']))).toBe(false);
  });
});
