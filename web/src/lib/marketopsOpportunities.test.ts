import { describe, expect, it } from 'vitest';
import {
  summarizeMarketOpsOpportunity,
  summarizeMarketOpsHypothesisEvaluation,
  parseOpportunityContributions,
  aggregateOpportunityRejectionReasons,
  reasonCodeLabel,
  formatScore,
  opportunityLifecycleStyle,
  directionLabel,
  directionTone,
} from './marketopsOpportunities';

describe('summarizeMarketOpsOpportunity (G139)', () => {
  it('reads scalars and parses contributions / overlap / families from the payload', () => {
    const v = summarizeMarketOpsOpportunity({
      opportunity_id: 'mopp-1',
      tenant_id: 'tenant-1',
      symbol: 'AAPL',
      direction: 'downside',
      horizon: '5_to_20_sessions',
      lifecycle_status: 'active',
      opportunity_score: 0.8,
      confidence_score: 0.75,
      domain_diversity_score: 0.67,
      conflict_score: 0.1,
      hypothesis_evaluation_ids: ['eval-1', 'eval-2'],
      conflicting_evaluation_ids: ['eval-9'],
      invalidating_evidence_ids: ['ev-9'],
      research_only: true,
      version: 1,
      opportunity_payload: {
        scoring_version: 'marketops.opportunity_score.v1',
        contributions: [
          { evaluation_id: 'eval-1', hypothesis_key: 'h1', hypothesis_version: '1', domain: 'options', trigger_score: 0.9, confidence_score: 0.8, quality_score: 0.7 },
          { evaluation_id: 'eval-2', hypothesis_key: 'h2', hypothesis_version: '1', domain: 'flow', trigger_score: 0.4 },
        ],
        overlap_suppressed_evaluation_ids: ['eval-3'],
        hypothesis_families: ['options', 'flow'],
      },
    });
    expect(v.opportunityId).toBe('mopp-1');
    expect(v.symbol).toBe('AAPL');
    expect(v.researchOnly).toBe(true);
    expect(v.scoringVersion).toBe('marketops.opportunity_score.v1');
    expect(v.overlapSuppressedEvaluationIds).toEqual(['eval-3']);
    expect(v.hypothesisFamilies).toEqual(['options', 'flow']);
    // Contributions ordered by trigger score desc.
    expect(v.contributions.map((c) => c.evaluationId)).toEqual(['eval-1', 'eval-2']);
    expect(v.contributions[0].triggerScore).toBeCloseTo(0.9);
    expect(v.contributions[1].qualityScore).toBeNull();
  });

  it('collapses non-object payloads to empty values', () => {
    const v = summarizeMarketOpsOpportunity(null);
    expect(v.opportunityId).toBe('');
    expect(v.contributions).toEqual([]);
    expect(v.overlapSuppressedEvaluationIds).toEqual([]);
  });
});

describe('parseOpportunityContributions (G139)', () => {
  it('orders by trigger score desc with absent scores last and tolerates bad input', () => {
    const cs = parseOpportunityContributions({
      contributions: [
        { evaluation_id: 'a', trigger_score: 0.2 },
        { evaluation_id: 'b', trigger_score: 0.9 },
        { evaluation_id: 'c' },
      ],
    });
    expect(cs.map((c) => c.evaluationId)).toEqual(['b', 'a', 'c']);
    expect(cs[2].triggerScore).toBeNull();
    expect(parseOpportunityContributions(null)).toEqual([]);
    expect(parseOpportunityContributions({ contributions: 'nope' })).toEqual([]);
  });
});

describe('summarizeMarketOpsHypothesisEvaluation (G139)', () => {
  it('reads nullable scores + reason codes', () => {
    const e = summarizeMarketOpsHypothesisEvaluation({
      evaluation_id: 'eval-1',
      hypothesis_key: 'h1',
      eligible: true,
      triggered: false,
      trigger_score: 0.3,
      quality_score: 0.5,
      reason_codes: ['eligible_not_triggered', 'threshold_not_met:oi_change_below_minimum'],
      evidence_ids: ['ev-1'],
    });
    expect(e.evaluationId).toBe('eval-1');
    expect(e.eligible).toBe(true);
    expect(e.triggerScore).toBeCloseTo(0.3);
    expect(e.confidenceScore).toBeNull();
    expect(e.reasonCodes).toHaveLength(2);
  });
});

describe('reasonCodeLabel (G139)', () => {
  it('translates known tokens and passes unknown through', () => {
    expect(reasonCodeLabel('eligible_not_triggered')).toBe('Eligible, not triggered');
    expect(reasonCodeLabel('triggered_research_only')).toBe('Triggered, research-only');
    expect(reasonCodeLabel('threshold_not_met:oi_change_below_minimum')).toBe('Threshold not met (oi_change_below_minimum)');
    expect(reasonCodeLabel('threshold_not_met')).toBe('Threshold not met');
    expect(reasonCodeLabel('something_new')).toBe('something_new');
  });
});

describe('aggregateOpportunityRejectionReasons (G139)', () => {
  it('counts evaluated/eligible/triggered and ranks top reason codes', () => {
    const evals = [
      { eligible: true, triggered: false, reasonCodes: ['eligible_not_triggered', 'threshold_not_met:x'] },
      { eligible: true, triggered: false, reasonCodes: ['eligible_not_triggered'] },
      { eligible: false, triggered: false, reasonCodes: ['threshold_not_met:y'] },
    ].map((e) => ({ ...e, evaluationId: '', hypothesisKey: '', hypothesisVersion: '', marketStateId: '', assetId: '', symbol: '', sessionDate: '', invalidated: false, triggerScore: null, confidenceScore: null, qualityScore: null, evidenceIds: [] as string[] }));
    const agg = aggregateOpportunityRejectionReasons(evals, 5);
    expect(agg.evaluated).toBe(3);
    expect(agg.eligible).toBe(2);
    expect(agg.triggered).toBe(0);
    expect(agg.entries[0]).toEqual({ token: 'eligible_not_triggered', label: 'Eligible, not triggered', count: 2 });
    expect(agg.entries.map((e) => e.count)).toEqual([2, 1, 1]);
  });

  it('respects the topN cap', () => {
    const evals = Array.from({ length: 7 }, (_, i) => ({
      eligible: false, triggered: false, reasonCodes: [`reason_${i}`],
      evaluationId: '', hypothesisKey: '', hypothesisVersion: '', marketStateId: '', assetId: '', symbol: '', sessionDate: '', invalidated: false, triggerScore: null, confidenceScore: null, qualityScore: null, evidenceIds: [] as string[],
    }));
    expect(aggregateOpportunityRejectionReasons(evals, 5).entries).toHaveLength(5);
  });
});

describe('formatScore + styles (G139)', () => {
  it('formats scores and em-dashes nulls', () => {
    expect(formatScore(0.75)).toBe('0.75');
    expect(formatScore(null)).toBe('—');
    expect(formatScore(undefined, 3)).toBe('—');
  });

  it('tones lifecycle values', () => {
    expect(opportunityLifecycleStyle('active')).toContain('emerald');
    expect(opportunityLifecycleStyle('invalidated')).toContain('red');
    expect(opportunityLifecycleStyle('weakening')).toContain('amber');
    expect(opportunityLifecycleStyle('expired')).toContain('gray');
    expect(opportunityLifecycleStyle('future')).toContain('gray-600');
  });

  it('labels and tones directions', () => {
    expect(directionLabel('upside')).toBe('Upside');
    expect(directionLabel('downside')).toBe('Downside');
    expect(directionLabel('non_directional')).toBe('Non-directional');
    expect(directionTone('upside')).toContain('emerald');
    expect(directionTone('downside')).toContain('red');
  });
});
