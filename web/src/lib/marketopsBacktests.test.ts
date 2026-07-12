import { describe, expect, it } from 'vitest';
import {
  summarizeBacktestMetrics,
  isZeroInputBacktest,
  compareBacktestRuns,
  dominantRecommendation,
  parseBacktestSymbols,
  policyResultsByProposal,
  recommendationLabel,
  recommendationStyle,
  comparisonRecommendationLabel,
  comparisonRecommendationStyle,
  COMPARISON_DELTA_FIELDS,
  summarizeComparisonDeltas,
  comparisonMetrics,
  summarizeBacktestEvaluation,
  isNeedsMoreDataEvaluation,
  formatEvaluationPercent,
  MARKETOPS_BACKTEST_EVALUATION_RECOMMENDATIONS,
  promotionReadinessLabel,
  promotionReadinessStyle,
  promotionCandidateStatusLabel,
  promotionCandidateStatusStyle,
  summarizePromotionEvidence,
  MARKETOPS_BACKTEST_PROMOTION_READINESS_STATUSES,
  MARKETOPS_BACKTEST_PROMOTION_DECISION_STATUSES,
} from './marketopsBacktests';

describe('summarizeBacktestMetrics (G081)', () => {
  it('reads the canonical metric fields', () => {
    const s = summarizeBacktestMetrics({
      scanned: 5,
      signals: 3,
      artifacts: 3,
      graph_proposals: 4,
      policy_results: 4,
      recommendation_counts: { auto_accept_candidate: 2, manual_review_required: 2 },
      batches: 1,
      max_records: 5,
      batch_size: 5,
      started_at: '2026-07-12T00:00:00Z',
      completed_at: '2026-07-12T00:00:01Z',
    });
    expect(s.scanned).toBe(5);
    expect(s.signals).toBe(3);
    expect(s.graphProposals).toBe(4);
    expect(s.policyResults).toBe(4);
    expect(s.recommendationCounts).toEqual({ auto_accept_candidate: 2, manual_review_required: 2 });
    expect(s.batches).toBe(1);
    expect(s.startedAt).toBe('2026-07-12T00:00:00Z');
    expect(s.completedAt).toBe('2026-07-12T00:00:01Z');
  });

  it('tolerates a non-object metrics payload', () => {
    expect(summarizeBacktestMetrics(undefined)).toEqual({
      scanned: 0, signals: 0, artifacts: 0, graphProposals: 0, policyResults: 0,
      recommendationCounts: {}, batches: 0, maxRecords: 0, batchSize: 0,
      startedAt: '', completedAt: '',
    });
    expect(summarizeBacktestMetrics(null).signals).toBe(0);
    expect(summarizeBacktestMetrics('not-an-object').scanned).toBe(0);
    expect(summarizeBacktestMetrics([1, 2, 3]).policyResults).toBe(0);
  });

  it('coerces numeric strings and ignores malformed values', () => {
    const s = summarizeBacktestMetrics({
      scanned: '7',
      signals: null,
      graph_proposals: { oops: true },
      recommendation_counts: { auto_accept_candidate: '3', bad: 'NaN', manual_review_required: 1 },
      started_at: 12345,
    });
    expect(s.scanned).toBe(7);
    expect(s.signals).toBe(0);
    expect(s.graphProposals).toBe(0);
    expect(s.recommendationCounts).toEqual({ auto_accept_candidate: 3, manual_review_required: 1 });
    expect(s.startedAt).toBe('');
  });

  it('identifies successful runs with no matching normalized events', () => {
    expect(isZeroInputBacktest('succeeded', { scanned: 0 })).toBe(true);
    expect(isZeroInputBacktest('succeeded', { scanned: '0' })).toBe(true);
    expect(isZeroInputBacktest('succeeded', { scanned: 1 })).toBe(false);
    expect(isZeroInputBacktest('failed', { scanned: 0 })).toBe(false);
  });
});

describe('dominantRecommendation (G081)', () => {
  it('returns the highest-count recommendation', () => {
    expect(dominantRecommendation({ auto_accept_candidate: 1, manual_review_required: 3 })).toEqual({
      key: 'manual_review_required',
      count: 3,
    });
  });

  it('returns null when all counts are zero or empty', () => {
    expect(dominantRecommendation({})).toBeNull();
    expect(dominantRecommendation({ auto_accept_candidate: 0 })).toBeNull();
  });
});

describe('parseBacktestSymbols (G081)', () => {
  it('uppercases, trims, and de-duplicates comma/space input', () => {
    expect(parseBacktestSymbols('spy, aapl aapl,MSFT')).toEqual(['SPY', 'AAPL', 'MSFT']);
    expect(parseBacktestSymbols('  spy ,  spy ')).toEqual(['SPY']);
    expect(parseBacktestSymbols('SPY')).toEqual(['SPY']);
  });

  it('drops empty values', () => {
    expect(parseBacktestSymbols(',,, ,')).toEqual([]);
    expect(parseBacktestSymbols('')).toEqual([]);
    expect(parseBacktestSymbols('SPY,,AAPL,')).toEqual(['SPY', 'AAPL']);
  });
});


describe('compareBacktestRuns (G082)', () => {
  it('aggregates run metrics, zero-input runs, and recommendation shares', () => {
    const summary = compareBacktestRuns([
      {
        run_id: 'bt-1', tenant_id: 'tenant-local', app_id: 'marketops', domain: 'market_data', use_case: 'daily_market_surveillance',
        source_id: 'src-massive', source_adapter: 'market_data.massive', dataset: 'equity_eod_prices', detector_id: 'marketops.dsm.taxonomy_v1', detector_version: 'v1', status: 'succeeded', requested_by: 'operator', window_start: '', window_end: '', started_at: '', filters: {}, parameters: {}, created_at: '', updated_at: '',
        metrics: { scanned: 2, signals: 1, artifacts: 1, graph_proposals: 5, policy_results: 5, recommendation_counts: { auto_accept_candidate: 5 } },
      },
      {
        run_id: 'bt-2', tenant_id: 'tenant-local', app_id: 'marketops', domain: 'market_data', use_case: 'daily_market_surveillance',
        source_id: 'src-massive', source_adapter: 'market_data.massive', dataset: 'equity_eod_prices', detector_id: 'marketops.dsm.taxonomy_v1', detector_version: 'v1', status: 'succeeded', requested_by: 'operator', window_start: '', window_end: '', started_at: '', filters: {}, parameters: {}, created_at: '', updated_at: '',
        metrics: { scanned: 0, signals: 0, artifacts: 0, graph_proposals: 0, policy_results: 0, recommendation_counts: {} },
      },
      {
        run_id: 'bt-3', tenant_id: 'tenant-local', app_id: 'marketops', domain: 'market_data', use_case: 'daily_market_surveillance',
        source_id: 'src-massive', source_adapter: 'market_data.massive', dataset: 'options_contracts_daily', detector_id: 'marketops.dsm.taxonomy_v1', detector_version: 'v1', status: 'failed', requested_by: 'operator', window_start: '', window_end: '', started_at: '', filters: {}, parameters: {}, created_at: '', updated_at: '',
        metrics: { scanned: 3, signals: 2, artifacts: 2, graph_proposals: 4, policy_results: 4, recommendation_counts: { manual_review_required: 3, auto_accept_candidate: 1 } },
      },
    ]);

    expect(summary.runs).toBe(3);
    expect(summary.succeeded).toBe(2);
    expect(summary.failed).toBe(1);
    expect(summary.zeroInput).toBe(1);
    expect(summary.scanned).toBe(5);
    expect(summary.signals).toBe(3);
    expect(summary.signalYieldPct).toBe(60);
    expect(summary.policyResultsPerSignal).toBe(3);
    expect(summary.recommendationCounts).toEqual({ auto_accept_candidate: 6, manual_review_required: 3 });
    expect(summary.dominantRecommendation).toEqual({ key: 'auto_accept_candidate', count: 6, share: 6 / 9 });
    expect(summary.datasets).toEqual(['equity_eod_prices', 'options_contracts_daily']);
  });

  it('returns an empty summary for non-array input', () => {
    const summary = compareBacktestRuns(null);
    expect(summary.runs).toBe(0);
    expect(summary.signalYieldPct).toBe(0);
    expect(summary.dominantRecommendation).toBeNull();
  });
});

describe('policyResultsByProposal (G081)', () => {
  it('indexes results by proposal_id', () => {
    const map = policyResultsByProposal([
      { proposal_id: 'p1', recommendation: 'auto_accept_candidate', reason: 'ok' },
      { proposal_id: 'p2', recommendation: 'manual_review_required', reason: 'low conf' },
    ] as never);
    expect(map.get('p1')?.recommendation).toBe('auto_accept_candidate');
    expect(map.get('p2')?.reason).toBe('low conf');
    expect(map.has('p3')).toBe(false);
  });

  it('skips entries without a proposal_id and tolerates non-arrays', () => {
    const map = policyResultsByProposal([
      { recommendation: 'auto_accept_candidate' },
      { proposal_id: '', recommendation: 'auto_accept_candidate' },
      { proposal_id: 'p9', recommendation: 'auto_reject_candidate' },
    ] as never);
    expect(map.size).toBe(1);
    expect(map.get('p9')?.recommendation).toBe('auto_reject_candidate');
    expect(policyResultsByProposal(undefined).size).toBe(0);
    expect(policyResultsByProposal(null).size).toBe(0);
  });
});

describe('recommendation presentation (G081)', () => {
  it('humanizes the recommendation key', () => {
    expect(recommendationLabel('manual_review_required')).toBe('manual review required');
    expect(recommendationLabel('auto_accept_candidate')).toBe('auto accept candidate');
  });

  it('resolves a style for known keys and falls back for unknown ones', () => {
    expect(recommendationStyle('auto_accept_candidate')).toContain('emerald');
    expect(recommendationStyle('manual_review_required')).toContain('amber');
    expect(recommendationStyle('supersede_candidate')).toContain('violet');
    expect(recommendationStyle('unknown_future_value')).toContain('gray');
  });
});

describe('comparison recommendation presentation (G083)', () => {
  it('humanizes the advisory comparison recommendation key', () => {
    expect(comparisonRecommendationLabel('neutral_candidate')).toBe('neutral candidate');
    expect(comparisonRecommendationLabel('needs_more_data')).toBe('needs more data');
    expect(comparisonRecommendationLabel('regression_candidate')).toBe('regression candidate');
  });

  it('resolves advisory styles: regression red, improvement green, manual review amber', () => {
    expect(comparisonRecommendationStyle('regression_candidate')).toContain('red');
    expect(comparisonRecommendationStyle('improvement_candidate')).toContain('emerald');
    expect(comparisonRecommendationStyle('manual_review_required')).toContain('amber');
    expect(comparisonRecommendationStyle('unknown_future_value')).toContain('gray');
  });
});

describe('summarizeComparisonDeltas (G083)', () => {
  it('surfaces the spec key deltas in order', () => {
    expect(COMPARISON_DELTA_FIELDS.map((f) => f.key)).toEqual([
      'run_count_delta',
      'zero_input_rate_delta',
      'scanned_delta',
      'signal_yield_delta',
      'policy_results_per_signal_delta',
      'auto_accept_candidate_share_delta',
      'auto_reject_candidate_share_delta',
      'manual_review_required_share_delta',
      'supersede_existing_candidate_share_delta',
      'dominant_recommendation_changed',
    ]);
  });

  it('formats counts, percentage-point fractions, rates, and the boolean flag', () => {
    const entries = summarizeComparisonDeltas({
      run_count_delta: 3,
      zero_input_rate_delta: 0.05,
      scanned_delta: -20,
      signal_yield_delta: -0.125,
      policy_results_per_signal_delta: 0.5,
      auto_accept_candidate_share_delta: 0,
      auto_reject_candidate_share_delta: 0.1,
      manual_review_required_share_delta: -0.033,
      supersede_existing_candidate_share_delta: 0,
      dominant_recommendation_changed: true,
    });

    const byKey = Object.fromEntries(entries.map((e) => [e.key, e]));
    expect(byKey.run_count_delta.display).toBe('+3');
    expect(byKey.run_count_delta.changed).toBe(true);
    expect(byKey.zero_input_rate_delta.display).toBe('+5.0%');
    expect(byKey.scanned_delta.display).toBe('-20');
    expect(byKey.signal_yield_delta.display).toBe('-12.5%');
    // policy_results_per_signal is a rate, not a 0–1 fraction -> plain signed decimal.
    expect(byKey.policy_results_per_signal_delta.display).toBe('+0.50');
    // Zero numeric deltas are not "changed" and render as 0.
    expect(byKey.auto_accept_candidate_share_delta.display).toBe('0.0%');
    expect(byKey.auto_accept_candidate_share_delta.changed).toBe(false);
    expect(byKey.auto_reject_candidate_share_delta.display).toBe('+10.0%');
    expect(byKey.manual_review_required_share_delta.display).toBe('-3.3%');
    expect(byKey.dominant_recommendation_changed.display).toBe('changed');
    expect(byKey.dominant_recommendation_changed.changed).toBe(true);
  });

  it('marks missing/malformed deltas as unchanged and renders em-dash', () => {
    const entries = summarizeComparisonDeltas({
      run_count_delta: 'oops',
      scanned_delta: null,
      dominant_recommendation_changed: false,
    });
    const byKey = Object.fromEntries(entries.map((e) => [e.key, e]));
    expect(byKey.run_count_delta.display).toBe('—');
    expect(byKey.run_count_delta.changed).toBe(false);
    expect(byKey.scanned_delta.display).toBe('—');
    expect(byKey.scanned_delta.changed).toBe(false);
    expect(byKey.signal_yield_delta.display).toBe('—');
    expect(byKey.dominant_recommendation_changed.display).toBe('unchanged');
    expect(byKey.dominant_recommendation_changed.changed).toBe(false);
  });

  it('tolerates non-object deltas without throwing', () => {
    expect(summarizeComparisonDeltas(undefined).length).toBe(COMPARISON_DELTA_FIELDS.length);
    expect(summarizeComparisonDeltas(null).every((e) => e.display === '—' || e.kind === 'flag')).toBe(true);
    expect(summarizeComparisonDeltas('nope')[0].display).toBe('—');
  });
});

describe('comparisonMetrics (G083)', () => {
  it('extracts comparison_metrics tolerantly', () => {
    const metrics = comparisonMetrics({
      comparison_metrics: { baseline: { run_count: 2 }, deltas: { run_count_delta: 1 } },
    });
    expect(metrics.deltas).toEqual({ run_count_delta: 1 });
    expect(comparisonMetrics({})).toEqual({});
    expect(comparisonMetrics(null)).toEqual({});
    expect(comparisonMetrics({ comparison_metrics: 'bad' })).toEqual({});
  });
});

describe('summarizeBacktestEvaluation (G085)', () => {
  it('reads the canonical evaluation fields', () => {
    const s = summarizeBacktestEvaluation({
      evaluation_id: 'bteval-1',
      run_id: 'bt-1',
      recommendation: 'improvement_candidate',
      recommendation_note: 'automatic recommendations align with available labels',
      candidate_count: 5,
      labeled_count: 5,
      positive_count: 5,
      negative_count: 0,
      true_positive: 5,
      false_positive: 0,
      true_negative: 0,
      false_negative: 0,
      manual_review_count: 0,
      precision: 1,
      recall: 1,
      specificity: 0,
      accuracy: 1,
      label_coverage: 1,
      created_at: '2026-07-12T20:50:00Z',
    });
    expect(s.evaluationId).toBe('bteval-1');
    expect(s.runId).toBe('bt-1');
    expect(s.recommendation).toBe('improvement_candidate');
    expect(s.recommendationNote).toBe('automatic recommendations align with available labels');
    expect(s.candidateCount).toBe(5);
    expect(s.labeledCount).toBe(5);
    expect(s.truePositive).toBe(5);
    expect(s.falsePositive).toBe(0);
    expect(s.manualReviewCount).toBe(0);
    expect(s.precision).toBe(1);
    expect(s.recall).toBe(1);
    expect(s.specificity).toBe(0);
    expect(s.accuracy).toBe(1);
    expect(s.labelCoverage).toBe(1);
    expect(s.createdAt).toBe('2026-07-12T20:50:00Z');
  });

  it('tolerates a non-object payload', () => {
    expect(summarizeBacktestEvaluation(undefined).labeledCount).toBe(0);
    expect(summarizeBacktestEvaluation(null).evaluationId).toBe('');
    expect(summarizeBacktestEvaluation('not-an-object').candidateCount).toBe(0);
    expect(summarizeBacktestEvaluation([1, 2, 3]).precision).toBe(0);
  });

  it('coerces numeric strings and ignores malformed values', () => {
    const s = summarizeBacktestEvaluation({
      candidate_count: '5',
      labeled_count: { oops: 1 },
      precision: null,
      true_positive: '3',
    });
    expect(s.candidateCount).toBe(5);
    expect(s.labeledCount).toBe(0);
    expect(s.precision).toBe(0);
    expect(s.truePositive).toBe(3);
  });
});

describe('label-aware evaluation states (G085)', () => {
  it('flags zero-label evaluations as needs-more-data (the authoritative signal)', () => {
    expect(isNeedsMoreDataEvaluation({ labeled_count: 0, candidate_count: 5 })).toBe(true);
    expect(isNeedsMoreDataEvaluation({ labeled_count: '0' })).toBe(true);
    expect(isNeedsMoreDataEvaluation({ labeled_count: 3 })).toBe(false);
    // A missing/empty record has zero labels and is treated as needs-more-data.
    expect(isNeedsMoreDataEvaluation(undefined)).toBe(true);
    expect(isNeedsMoreDataEvaluation({})).toBe(true);
  });

  it('surfaces the advisory recommendation token set (never deploy/promote commands)', () => {
    expect([...MARKETOPS_BACKTEST_EVALUATION_RECOMMENDATIONS]).toEqual([
      'needs_more_data',
      'manual_review_required',
      'improvement_candidate',
      'neutral_candidate',
      'regression_candidate',
    ]);
    // Evaluation tokens reuse the G083 advisory comparison palette.
    for (const token of MARKETOPS_BACKTEST_EVALUATION_RECOMMENDATIONS) {
      expect(comparisonRecommendationStyle(token)).not.toContain('brand');
      expect(comparisonRecommendationLabel(token)).toBe(token.replace(/_/g, ' '));
    }
  });
});

describe('formatEvaluationPercent (G085)', () => {
  it('renders 0–1 fractions as compact percentages', () => {
    expect(formatEvaluationPercent(1)).toBe('100%');
    expect(formatEvaluationPercent(0)).toBe('0%');
    expect(formatEvaluationPercent(0.8)).toBe('80%');
    expect(formatEvaluationPercent(0.3333333)).toBe('33.3%');
    expect(formatEvaluationPercent(0.667)).toBe('66.7%');
  });

  it('coerces numeric strings and renders em-dash for malformed values', () => {
    expect(formatEvaluationPercent('0.5')).toBe('50%');
    expect(formatEvaluationPercent(undefined)).toBe('—');
    expect(formatEvaluationPercent(null)).toBe('—');
    expect(formatEvaluationPercent('oops')).toBe('—');
    expect(formatEvaluationPercent(Infinity)).toBe('—');
  });
});

describe('promotion readiness + status presentation (G086)', () => {
  it('surfaces the spec readiness + decision status sets', () => {
    expect([...MARKETOPS_BACKTEST_PROMOTION_READINESS_STATUSES]).toEqual([
      'ready_for_review',
      'needs_more_data',
      'manual_review_required',
      'regression_detected',
      'blocked',
    ]);
    expect([...MARKETOPS_BACKTEST_PROMOTION_DECISION_STATUSES]).toEqual([
      'approved_for_promotion',
      'rejected',
      'deferred',
      'superseded',
    ]);
  });

  it('humanizes readiness + candidate status keys', () => {
    expect(promotionReadinessLabel('needs_more_data')).toBe('needs more data');
    expect(promotionReadinessLabel('regression_detected')).toBe('regression detected');
    expect(promotionCandidateStatusLabel('approved_for_promotion')).toBe('approved for promotion');
    expect(promotionCandidateStatusLabel('unknown_future_value')).toBe('unknown future value');
  });

  it('resolves restrained readiness + status styles', () => {
    expect(promotionReadinessStyle('ready_for_review')).toContain('emerald');
    expect(promotionReadinessStyle('manual_review_required')).toContain('amber');
    expect(promotionReadinessStyle('regression_detected')).toContain('red');
    expect(promotionReadinessStyle('blocked')).toContain('red');
    expect(promotionReadinessStyle('needs_more_data')).toContain('gray');
    expect(promotionCandidateStatusStyle('approved_for_promotion')).toContain('emerald');
    expect(promotionCandidateStatusStyle('rejected')).toContain('red');
    expect(promotionCandidateStatusStyle('deferred')).toContain('amber');
    expect(promotionCandidateStatusStyle('superseded')).toContain('violet');
    expect(promotionCandidateStatusStyle('proposed')).toContain('gray');
    // Unknown future values fall back to a neutral gray chip.
    expect(promotionReadinessStyle('unknown_future_value')).toContain('gray');
    expect(promotionCandidateStatusStyle('unknown_future_value')).toContain('gray');
  });
});

describe('summarizePromotionEvidence (G086)', () => {
  it('reads the comparison + evaluation + version evidence blocks', () => {
    const s = summarizePromotionEvidence({
      baseline: { baseline_id: 'btbase-1', summary_id: 'btcal-1', name: 'July' },
      comparison: { comparison_id: 'btcmp-1', recommendation: 'neutral_candidate', recommendation_reason: 'within tolerance', metrics: {} },
      evaluation: {
        evaluation_id: 'bteval-1',
        recommendation: 'improvement_candidate',
        recommendation_note: 'labels align',
        precision: 0.9,
        recall: 0.8,
        accuracy: 0.85,
        label_coverage: 0.667,
      },
      detector: { detector_id: 'marketops.dsm.taxonomy_v1', detector_version: 'v1' },
      run: { run_id: 'bt-1' },
      policy_version: 'marketops.backtest.policy_v1',
      readiness: { status: 'ready_for_review', reasons: ['comparison and evaluation evidence meet review thresholds', 'recall is below review threshold'] },
    });
    expect(s.comparisonRecommendation).toBe('neutral_candidate');
    expect(s.comparisonRecommendationReason).toBe('within tolerance');
    expect(s.hasEvaluation).toBe(true);
    expect(s.evaluationId).toBe('bteval-1');
    expect(s.evaluationRecommendation).toBe('improvement_candidate');
    expect(s.evaluationRecommendationNote).toBe('labels align');
    // Evaluation rates are preserved as 0–1 fractions (the UI formats them).
    expect(s.precision).toBe(0.9);
    expect(s.recall).toBe(0.8);
    expect(s.accuracy).toBe(0.85);
    expect(s.labelCoverage).toBeCloseTo(0.667, 3);
    expect(s.policyVersion).toBe('marketops.backtest.policy_v1');
    expect(s.detectorVersion).toBe('v1');
    expect(s.readinessReasons).toEqual([
      'comparison and evaluation evidence meet review thresholds',
      'recall is below review threshold',
    ]);
  });

  it('renders evidence metrics as compact percentages via formatEvaluationPercent', () => {
    const s = summarizePromotionEvidence({ evaluation: { evaluation_id: 'bteval-1', precision: 0.9, recall: 0.333, accuracy: 1, label_coverage: 0 } });
    expect(formatEvaluationPercent(s.precision)).toBe('90%');
    expect(formatEvaluationPercent(s.recall)).toBe('33.3%');
    expect(formatEvaluationPercent(s.accuracy)).toBe('100%');
    expect(formatEvaluationPercent(s.labelCoverage)).toBe('0%');
  });

  it('flags a missing/empty evaluation block as no evaluation', () => {
    expect(summarizePromotionEvidence({ comparison: { recommendation: 'neutral_candidate' } }).hasEvaluation).toBe(false);
    expect(summarizePromotionEvidence({ evaluation: {} }).hasEvaluation).toBe(false);
    expect(summarizePromotionEvidence({ evaluation: { evaluation_id: '' } }).hasEvaluation).toBe(false);
  });

  it('tolerates a non-object / partial evidence payload without throwing', () => {
    const empty = summarizePromotionEvidence(undefined);
    expect(empty.comparisonRecommendation).toBe('');
    expect(empty.hasEvaluation).toBe(false);
    expect(empty.readinessReasons).toEqual([]);
    expect(summarizePromotionEvidence(null).policyVersion).toBe('');
    expect(summarizePromotionEvidence('nope').precision).toBe(0);
    // readiness.reasons drops non-string entries.
    const mixed = summarizePromotionEvidence({ readiness: { reasons: ['ok', 5, null, 'also ok'] } });
    expect(mixed.readinessReasons).toEqual(['ok', 'also ok']);
  });
});
