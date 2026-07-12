package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) UpsertMarketOpsBacktestEvaluation(ctx context.Context, record storage.MarketOpsBacktestEvaluationRecord) error {
	if strings.TrimSpace(record.EvaluationID) == "" || strings.TrimSpace(record.TenantID) == "" || strings.TrimSpace(record.RunID) == "" {
		return fmt.Errorf("marketops backtest evaluation_id, tenant_id, and run_id are required")
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO marketops_backtest_evaluations (
 evaluation_id, tenant_id, app_id, domain, use_case, run_id, detector_id, dataset, label_source, label_version,
 scoring_version, requested_by, candidate_count, labeled_count, positive_count, negative_count, superseded_count,
 unresolved_count, true_positive, false_positive, true_negative, false_negative, manual_review_count, unscored_count,
 precision, recall, specificity, accuracy, label_coverage, recommendation, recommendation_note, metrics
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32)
ON CONFLICT (evaluation_id) DO UPDATE SET
 tenant_id=EXCLUDED.tenant_id, app_id=EXCLUDED.app_id, domain=EXCLUDED.domain, use_case=EXCLUDED.use_case,
 run_id=EXCLUDED.run_id, detector_id=EXCLUDED.detector_id, dataset=EXCLUDED.dataset, label_source=EXCLUDED.label_source,
 label_version=EXCLUDED.label_version, scoring_version=EXCLUDED.scoring_version, requested_by=EXCLUDED.requested_by,
 candidate_count=EXCLUDED.candidate_count, labeled_count=EXCLUDED.labeled_count, positive_count=EXCLUDED.positive_count,
 negative_count=EXCLUDED.negative_count, superseded_count=EXCLUDED.superseded_count, unresolved_count=EXCLUDED.unresolved_count,
 true_positive=EXCLUDED.true_positive, false_positive=EXCLUDED.false_positive, true_negative=EXCLUDED.true_negative,
 false_negative=EXCLUDED.false_negative, manual_review_count=EXCLUDED.manual_review_count, unscored_count=EXCLUDED.unscored_count,
 precision=EXCLUDED.precision, recall=EXCLUDED.recall, specificity=EXCLUDED.specificity, accuracy=EXCLUDED.accuracy,
 label_coverage=EXCLUDED.label_coverage, recommendation=EXCLUDED.recommendation, recommendation_note=EXCLUDED.recommendation_note,
 metrics=EXCLUDED.metrics`,
		record.EvaluationID, strings.TrimSpace(record.TenantID), recordAppID(record.AppID), recordDomain(record.Domain), recordUseCase(record.UseCase), strings.TrimSpace(record.RunID), strings.TrimSpace(record.DetectorID), strings.TrimSpace(record.Dataset), firstNonEmptyString(record.LabelSource, "g080_graph_proposal_decision"), firstNonEmptyString(record.LabelVersion, "marketops.eval_label.v1"), firstNonEmptyString(record.ScoringVersion, "marketops.eval_scoring.v1"), firstNonEmptyString(record.RequestedBy, "operator-local"), record.CandidateCount, record.LabeledCount, record.PositiveCount, record.NegativeCount, record.SupersededCount, record.UnresolvedCount, record.TruePositive, record.FalsePositive, record.TrueNegative, record.FalseNegative, record.ManualReviewCount, record.UnscoredCount, record.Precision, record.Recall, record.Specificity, record.Accuracy, record.LabelCoverage, strings.TrimSpace(record.Recommendation), strings.TrimSpace(record.RecommendationNote), jsonOrEmpty(record.MetricsJSON))
	if err != nil {
		return fmt.Errorf("upsert marketops backtest evaluation: %w", err)
	}
	return nil
}

func (r *Repository) ListMarketOpsBacktestEvaluations(ctx context.Context, filter storage.MarketOpsBacktestEvaluationFilter) ([]storage.MarketOpsBacktestEvaluationRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsBacktestEvaluationSelect+`
WHERE ($1='' OR tenant_id=$1) AND ($2='' OR app_id=$2) AND ($3='' OR domain=$3) AND ($4='' OR use_case=$4)
 AND ($5='' OR run_id=$5) AND ($6='' OR detector_id=$6) AND ($7='' OR dataset=$7) AND ($8='' OR recommendation=$8)
ORDER BY created_at DESC LIMIT $9`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AppID), strings.TrimSpace(filter.Domain), strings.TrimSpace(filter.UseCase), strings.TrimSpace(filter.RunID), strings.TrimSpace(filter.DetectorID), strings.TrimSpace(filter.Dataset), strings.TrimSpace(filter.Recommendation), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops backtest evaluations: %w", err)
	}
	defer rows.Close()
	records := []storage.MarketOpsBacktestEvaluationRecord{}
	for rows.Next() {
		record, err := scanMarketOpsBacktestEvaluation(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list marketops backtest evaluations rows: %w", err)
	}
	return records, nil
}

func (r *Repository) GetMarketOpsBacktestEvaluation(ctx context.Context, evaluationID string) (storage.MarketOpsBacktestEvaluationRecord, error) {
	record, err := scanMarketOpsBacktestEvaluation(r.db.QueryRowContext(ctx, marketOpsBacktestEvaluationSelect+` WHERE evaluation_id=$1`, strings.TrimSpace(evaluationID)))
	if err != nil {
		return storage.MarketOpsBacktestEvaluationRecord{}, err
	}
	return record, nil
}

const marketOpsBacktestEvaluationSelect = `SELECT evaluation_id, tenant_id, app_id, domain, use_case, run_id, detector_id, dataset, label_source, label_version,
 scoring_version, requested_by, candidate_count, labeled_count, positive_count, negative_count, superseded_count, unresolved_count,
 true_positive, false_positive, true_negative, false_negative, manual_review_count, unscored_count, precision, recall, specificity,
 accuracy, label_coverage, recommendation, recommendation_note, metrics, created_at FROM marketops_backtest_evaluations`

type marketOpsBacktestEvaluationScanner interface{ Scan(dest ...any) error }

func scanMarketOpsBacktestEvaluation(scanner marketOpsBacktestEvaluationScanner) (storage.MarketOpsBacktestEvaluationRecord, error) {
	var record storage.MarketOpsBacktestEvaluationRecord
	if err := scanner.Scan(&record.EvaluationID, &record.TenantID, &record.AppID, &record.Domain, &record.UseCase, &record.RunID, &record.DetectorID, &record.Dataset, &record.LabelSource, &record.LabelVersion, &record.ScoringVersion, &record.RequestedBy, &record.CandidateCount, &record.LabeledCount, &record.PositiveCount, &record.NegativeCount, &record.SupersededCount, &record.UnresolvedCount, &record.TruePositive, &record.FalsePositive, &record.TrueNegative, &record.FalseNegative, &record.ManualReviewCount, &record.UnscoredCount, &record.Precision, &record.Recall, &record.Specificity, &record.Accuracy, &record.LabelCoverage, &record.Recommendation, &record.RecommendationNote, &record.MetricsJSON, &record.CreatedAt); err != nil {
		return storage.MarketOpsBacktestEvaluationRecord{}, mapScanError("scan marketops backtest evaluation", err)
	}
	return record, nil
}
