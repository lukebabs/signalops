package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) UpsertMarketOpsBacktestEvaluationLabel(ctx context.Context, record storage.MarketOpsBacktestEvaluationLabelRecord) error {
	if strings.TrimSpace(record.LabelID) == "" || strings.TrimSpace(record.TenantID) == "" || strings.TrimSpace(record.SourceProposalID) == "" || strings.TrimSpace(record.Label) == "" || strings.TrimSpace(record.LabelVersion) == "" {
		return fmt.Errorf("marketops backtest evaluation label_id, tenant_id, source_proposal_id, label, and label_version are required")
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO marketops_backtest_evaluation_labels (
 label_id, tenant_id, app_id, domain, use_case, source_proposal_id, artifact_id, signal_id, subject_symbol,
 candidate_type, graph_fact_key, decision_status, label, label_source, labeled_by, labeled_at, label_version, metadata
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
ON CONFLICT (source_proposal_id, label_version) DO UPDATE SET
 label_id=EXCLUDED.label_id, tenant_id=EXCLUDED.tenant_id, app_id=EXCLUDED.app_id, domain=EXCLUDED.domain, use_case=EXCLUDED.use_case,
 artifact_id=EXCLUDED.artifact_id, signal_id=EXCLUDED.signal_id, subject_symbol=EXCLUDED.subject_symbol,
 candidate_type=EXCLUDED.candidate_type, graph_fact_key=EXCLUDED.graph_fact_key, decision_status=EXCLUDED.decision_status,
 label=EXCLUDED.label, label_source=EXCLUDED.label_source, labeled_by=EXCLUDED.labeled_by, labeled_at=EXCLUDED.labeled_at,
 metadata=EXCLUDED.metadata, updated_at=now()`,
		record.LabelID, strings.TrimSpace(record.TenantID), recordAppID(record.AppID), recordDomain(record.Domain), recordUseCase(record.UseCase),
		strings.TrimSpace(record.SourceProposalID), strings.TrimSpace(record.ArtifactID), strings.TrimSpace(record.SignalID), strings.TrimSpace(record.SubjectSymbol),
		strings.TrimSpace(record.CandidateType), strings.TrimSpace(record.GraphFactKey), strings.TrimSpace(record.DecisionStatus), strings.TrimSpace(record.Label),
		firstNonEmptyString(record.LabelSource, "g080_graph_proposal_decision"), firstNonEmptyString(record.LabeledBy, "operator-local"), record.LabeledAt.UTC(), strings.TrimSpace(record.LabelVersion), jsonOrEmpty(record.MetadataJSON))
	if err != nil {
		return fmt.Errorf("upsert marketops backtest evaluation label: %w", err)
	}
	return nil
}

func (r *Repository) ListMarketOpsBacktestEvaluationLabels(ctx context.Context, filter storage.MarketOpsBacktestEvaluationLabelFilter) ([]storage.MarketOpsBacktestEvaluationLabelRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsBacktestEvaluationLabelSelect+`
WHERE ($1='' OR tenant_id=$1) AND ($2='' OR app_id=$2) AND ($3='' OR domain=$3) AND ($4='' OR use_case=$4)
 AND ($5='' OR source_proposal_id=$5) AND ($6='' OR artifact_id=$6) AND ($7='' OR signal_id=$7)
 AND ($8='' OR subject_symbol=$8) AND ($9='' OR candidate_type=$9) AND ($10='' OR decision_status=$10)
 AND ($11='' OR label=$11) AND ($12='' OR label_source=$12)
ORDER BY labeled_at DESC, created_at DESC LIMIT $13`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AppID), strings.TrimSpace(filter.Domain), strings.TrimSpace(filter.UseCase), strings.TrimSpace(filter.SourceProposalID), strings.TrimSpace(filter.ArtifactID), strings.TrimSpace(filter.SignalID), strings.TrimSpace(filter.SubjectSymbol), strings.TrimSpace(filter.CandidateType), strings.TrimSpace(filter.DecisionStatus), strings.TrimSpace(filter.Label), strings.TrimSpace(filter.LabelSource), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops backtest evaluation labels: %w", err)
	}
	defer rows.Close()
	records := []storage.MarketOpsBacktestEvaluationLabelRecord{}
	for rows.Next() {
		record, err := scanMarketOpsBacktestEvaluationLabel(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list marketops backtest evaluation labels rows: %w", err)
	}
	return records, nil
}

func (r *Repository) GetMarketOpsBacktestEvaluationLabel(ctx context.Context, labelID string) (storage.MarketOpsBacktestEvaluationLabelRecord, error) {
	record, err := scanMarketOpsBacktestEvaluationLabel(r.db.QueryRowContext(ctx, marketOpsBacktestEvaluationLabelSelect+` WHERE label_id=$1`, strings.TrimSpace(labelID)))
	if err != nil {
		return storage.MarketOpsBacktestEvaluationLabelRecord{}, err
	}
	return record, nil
}

const marketOpsBacktestEvaluationLabelSelect = `SELECT label_id, tenant_id, app_id, domain, use_case, source_proposal_id, artifact_id, signal_id,
 subject_symbol, candidate_type, graph_fact_key, decision_status, label, label_source, labeled_by, labeled_at, label_version,
 metadata, created_at, updated_at FROM marketops_backtest_evaluation_labels`

type marketOpsBacktestEvaluationLabelScanner interface{ Scan(dest ...any) error }

func scanMarketOpsBacktestEvaluationLabel(scanner marketOpsBacktestEvaluationLabelScanner) (storage.MarketOpsBacktestEvaluationLabelRecord, error) {
	var record storage.MarketOpsBacktestEvaluationLabelRecord
	if err := scanner.Scan(&record.LabelID, &record.TenantID, &record.AppID, &record.Domain, &record.UseCase, &record.SourceProposalID, &record.ArtifactID, &record.SignalID, &record.SubjectSymbol, &record.CandidateType, &record.GraphFactKey, &record.DecisionStatus, &record.Label, &record.LabelSource, &record.LabeledBy, &record.LabeledAt, &record.LabelVersion, &record.MetadataJSON, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return storage.MarketOpsBacktestEvaluationLabelRecord{}, mapScanError("scan marketops backtest evaluation label", err)
	}
	return record, nil
}
