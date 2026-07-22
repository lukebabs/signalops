package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) UpsertMarketOpsAlgorithmAdjudication(ctx context.Context, record storage.MarketOpsAlgorithmAdjudicationRecord) error {
	if strings.TrimSpace(record.TenantID) == "" || strings.TrimSpace(record.AdjudicationID) == "" || strings.TrimSpace(record.HypothesisEvaluationID) == "" || strings.TrimSpace(record.AlgorithmResultID) == "" {
		return fmt.Errorf("adjudication identity is required")
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO marketops_algorithm_adjudications (tenant_id,adjudication_id,hypothesis_evaluation_id,algorithm_result_id,hypothesis_key,hypothesis_version,symbol,session_date,verdict,confidence,explanation,correlation_id,adjudicator_version)
 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
 ON CONFLICT (tenant_id,hypothesis_evaluation_id,algorithm_result_id,adjudicator_version) DO NOTHING`,
		strings.TrimSpace(record.TenantID), strings.TrimSpace(record.AdjudicationID), strings.TrimSpace(record.HypothesisEvaluationID), strings.TrimSpace(record.AlgorithmResultID), strings.TrimSpace(record.HypothesisKey), strings.TrimSpace(record.HypothesisVersion), strings.ToUpper(strings.TrimSpace(record.Symbol)), record.SessionDate.UTC(), strings.TrimSpace(record.Verdict), record.Confidence, jsonOrEmpty(record.ExplanationJSON), strings.TrimSpace(record.CorrelationID), strings.TrimSpace(record.AdjudicatorVersion))
	return err
}

func (r *Repository) ListMarketOpsAlgorithmAdjudications(ctx context.Context, filter storage.MarketOpsAlgorithmAdjudicationFilter) ([]storage.MarketOpsAlgorithmAdjudicationRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT adjudication_id,tenant_id,hypothesis_evaluation_id,algorithm_result_id,hypothesis_key,hypothesis_version,symbol,session_date,verdict,confidence,explanation,correlation_id,adjudicator_version,created_at FROM marketops_algorithm_adjudications WHERE ($1='' OR tenant_id=$1) AND ($2='' OR symbol=$2) AND ($3='' OR hypothesis_evaluation_id=$3) AND ($4='' OR correlation_id=$4) ORDER BY session_date DESC,created_at DESC LIMIT $5`, strings.TrimSpace(filter.TenantID), strings.ToUpper(strings.TrimSpace(filter.Symbol)), strings.TrimSpace(filter.HypothesisEvaluationID), strings.TrimSpace(filter.CorrelationID), clampLimit(filter.Limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []storage.MarketOpsAlgorithmAdjudicationRecord{}
	for rows.Next() {
		var item storage.MarketOpsAlgorithmAdjudicationRecord
		if err := rows.Scan(&item.AdjudicationID, &item.TenantID, &item.HypothesisEvaluationID, &item.AlgorithmResultID, &item.HypothesisKey, &item.HypothesisVersion, &item.Symbol, &item.SessionDate, &item.Verdict, &item.Confidence, &item.ExplanationJSON, &item.CorrelationID, &item.AdjudicatorVersion, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
