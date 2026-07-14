package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) UpsertMarketOpsBacktestCalibrationReadiness(ctx context.Context, record storage.MarketOpsBacktestCalibrationReadinessRecord) error {
	if strings.TrimSpace(record.ReadinessID) == "" || strings.TrimSpace(record.TenantID) == "" || strings.TrimSpace(record.ReadinessStatus) == "" {
		return fmt.Errorf("marketops backtest readiness_id, tenant_id, and readiness_status are required")
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO marketops_backtest_calibration_readiness (
 readiness_id, tenant_id, app_id, domain, use_case, baseline_id, comparison_id, evaluation_id, candidate_id,
 detector_id, dataset_scope, universe_group, window_start, window_end, readiness_status, readiness_reasons,
 coverage_metrics, label_metrics, evaluation_metrics, thresholds, evidence, requested_by
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)
ON CONFLICT (readiness_id) DO UPDATE SET
 tenant_id=EXCLUDED.tenant_id, app_id=EXCLUDED.app_id, domain=EXCLUDED.domain, use_case=EXCLUDED.use_case,
 baseline_id=EXCLUDED.baseline_id, comparison_id=EXCLUDED.comparison_id, evaluation_id=EXCLUDED.evaluation_id,
 candidate_id=EXCLUDED.candidate_id, detector_id=EXCLUDED.detector_id, dataset_scope=EXCLUDED.dataset_scope,
 universe_group=EXCLUDED.universe_group, window_start=EXCLUDED.window_start, window_end=EXCLUDED.window_end,
 readiness_status=EXCLUDED.readiness_status, readiness_reasons=EXCLUDED.readiness_reasons,
 coverage_metrics=EXCLUDED.coverage_metrics, label_metrics=EXCLUDED.label_metrics, evaluation_metrics=EXCLUDED.evaluation_metrics,
 thresholds=EXCLUDED.thresholds, evidence=EXCLUDED.evidence, requested_by=EXCLUDED.requested_by`,
		record.ReadinessID, strings.TrimSpace(record.TenantID), recordAppID(record.AppID), recordDomain(record.Domain), recordUseCase(record.UseCase),
		strings.TrimSpace(record.BaselineID), strings.TrimSpace(record.ComparisonID), strings.TrimSpace(record.EvaluationID), strings.TrimSpace(record.CandidateID),
		strings.TrimSpace(record.DetectorID), pqArray(record.DatasetScope), strings.TrimSpace(record.UniverseGroup), record.WindowStart, record.WindowEnd,
		strings.TrimSpace(record.ReadinessStatus), pqArray(record.ReadinessReasons), jsonOrEmpty(record.CoverageMetricsJSON), jsonOrEmpty(record.LabelMetricsJSON),
		jsonOrEmpty(record.EvaluationMetricsJSON), jsonOrEmpty(record.ThresholdsJSON), jsonOrEmpty(record.EvidenceJSON), firstNonEmptyString(record.RequestedBy, "operator-local"))
	if err != nil {
		return fmt.Errorf("upsert marketops backtest calibration readiness: %w", err)
	}
	return nil
}

func (r *Repository) ListMarketOpsBacktestCalibrationReadiness(ctx context.Context, filter storage.MarketOpsBacktestCalibrationReadinessFilter) ([]storage.MarketOpsBacktestCalibrationReadinessRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsBacktestCalibrationReadinessSelect+`
WHERE ($1='' OR tenant_id=$1) AND ($2='' OR app_id=$2) AND ($3='' OR domain=$3) AND ($4='' OR use_case=$4)
 AND ($5='' OR baseline_id=$5) AND ($6='' OR comparison_id=$6) AND ($7='' OR evaluation_id=$7) AND ($8='' OR candidate_id=$8)
 AND ($9='' OR detector_id=$9) AND ($10='' OR readiness_status=$10)
ORDER BY created_at DESC LIMIT $11`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AppID), strings.TrimSpace(filter.Domain), strings.TrimSpace(filter.UseCase), strings.TrimSpace(filter.BaselineID), strings.TrimSpace(filter.ComparisonID), strings.TrimSpace(filter.EvaluationID), strings.TrimSpace(filter.CandidateID), strings.TrimSpace(filter.DetectorID), strings.TrimSpace(filter.ReadinessStatus), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops backtest calibration readiness: %w", err)
	}
	defer rows.Close()
	records := []storage.MarketOpsBacktestCalibrationReadinessRecord{}
	for rows.Next() {
		record, err := scanMarketOpsBacktestCalibrationReadiness(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list marketops backtest calibration readiness rows: %w", err)
	}
	return records, nil
}

func (r *Repository) GetMarketOpsBacktestCalibrationReadiness(ctx context.Context, readinessID string) (storage.MarketOpsBacktestCalibrationReadinessRecord, error) {
	record, err := scanMarketOpsBacktestCalibrationReadiness(r.db.QueryRowContext(ctx, marketOpsBacktestCalibrationReadinessSelect+` WHERE readiness_id=$1`, strings.TrimSpace(readinessID)))
	if err != nil {
		return storage.MarketOpsBacktestCalibrationReadinessRecord{}, err
	}
	return record, nil
}

const marketOpsBacktestCalibrationReadinessSelect = `SELECT readiness_id, tenant_id, app_id, domain, use_case, baseline_id, comparison_id, evaluation_id,
 candidate_id, detector_id, COALESCE(array_to_json(dataset_scope), '[]'::json)::text, universe_group, window_start, window_end,
 readiness_status, COALESCE(array_to_json(readiness_reasons), '[]'::json)::text, coverage_metrics, label_metrics,
 evaluation_metrics, thresholds, evidence, requested_by, created_at FROM marketops_backtest_calibration_readiness`

type marketOpsBacktestCalibrationReadinessScanner interface{ Scan(dest ...any) error }

func scanMarketOpsBacktestCalibrationReadiness(scanner marketOpsBacktestCalibrationReadinessScanner) (storage.MarketOpsBacktestCalibrationReadinessRecord, error) {
	var record storage.MarketOpsBacktestCalibrationReadinessRecord
	var datasetScopeJSON, reasonsJSON string
	if err := scanner.Scan(&record.ReadinessID, &record.TenantID, &record.AppID, &record.Domain, &record.UseCase, &record.BaselineID, &record.ComparisonID, &record.EvaluationID, &record.CandidateID, &record.DetectorID, &datasetScopeJSON, &record.UniverseGroup, &record.WindowStart, &record.WindowEnd, &record.ReadinessStatus, &reasonsJSON, &record.CoverageMetricsJSON, &record.LabelMetricsJSON, &record.EvaluationMetricsJSON, &record.ThresholdsJSON, &record.EvidenceJSON, &record.RequestedBy, &record.CreatedAt); err != nil {
		return storage.MarketOpsBacktestCalibrationReadinessRecord{}, mapScanError("scan marketops backtest calibration readiness", err)
	}
	if err := json.Unmarshal([]byte(datasetScopeJSON), &record.DatasetScope); err != nil {
		return storage.MarketOpsBacktestCalibrationReadinessRecord{}, err
	}
	if err := json.Unmarshal([]byte(reasonsJSON), &record.ReadinessReasons); err != nil {
		return storage.MarketOpsBacktestCalibrationReadinessRecord{}, err
	}
	return record, nil
}
