package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) UpsertMarketOpsBacktestPromotionCandidate(ctx context.Context, record storage.MarketOpsBacktestPromotionCandidateRecord) error {
	if strings.TrimSpace(record.CandidateID) == "" || strings.TrimSpace(record.TenantID) == "" || strings.TrimSpace(record.BaselineID) == "" || strings.TrimSpace(record.ComparisonID) == "" {
		return fmt.Errorf("marketops backtest promotion candidate_id, tenant_id, baseline_id, and comparison_id are required")
	}
	status := firstNonEmptyString(record.Status, storage.MarketOpsBacktestPromotionCandidateStatusProposed)
	_, err := r.db.ExecContext(ctx, `
INSERT INTO marketops_backtest_promotion_candidates (
 candidate_id, tenant_id, app_id, domain, use_case, baseline_id, comparison_id, evaluation_id, run_id, detector_id,
 detector_version, dataset, policy_version, candidate_version, readiness_status, readiness_reasons, evidence, status,
 requested_by, reviewed_by, reviewed_at, decision_note
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)
ON CONFLICT (candidate_id) DO UPDATE SET
 tenant_id=EXCLUDED.tenant_id, app_id=EXCLUDED.app_id, domain=EXCLUDED.domain, use_case=EXCLUDED.use_case,
 baseline_id=EXCLUDED.baseline_id, comparison_id=EXCLUDED.comparison_id, evaluation_id=EXCLUDED.evaluation_id,
 run_id=EXCLUDED.run_id, detector_id=EXCLUDED.detector_id, detector_version=EXCLUDED.detector_version,
 dataset=EXCLUDED.dataset, policy_version=EXCLUDED.policy_version, candidate_version=EXCLUDED.candidate_version,
 readiness_status=EXCLUDED.readiness_status, readiness_reasons=EXCLUDED.readiness_reasons, evidence=EXCLUDED.evidence,
 status=EXCLUDED.status, requested_by=EXCLUDED.requested_by, reviewed_by=EXCLUDED.reviewed_by,
 reviewed_at=EXCLUDED.reviewed_at, decision_note=EXCLUDED.decision_note, updated_at=now()`,
		record.CandidateID, strings.TrimSpace(record.TenantID), recordAppID(record.AppID), recordDomain(record.Domain), recordUseCase(record.UseCase), strings.TrimSpace(record.BaselineID), strings.TrimSpace(record.ComparisonID), strings.TrimSpace(record.EvaluationID), strings.TrimSpace(record.RunID), strings.TrimSpace(record.DetectorID), strings.TrimSpace(record.DetectorVersion), strings.TrimSpace(record.Dataset), strings.TrimSpace(record.PolicyVersion), strings.TrimSpace(record.CandidateVersion), strings.TrimSpace(record.ReadinessStatus), pqArray(record.ReadinessReasons), jsonOrEmpty(record.EvidenceJSON), status, firstNonEmptyString(record.RequestedBy, "operator-local"), strings.TrimSpace(record.ReviewedBy), record.ReviewedAt, strings.TrimSpace(record.DecisionNote))
	if err != nil {
		return fmt.Errorf("upsert marketops backtest promotion candidate: %w", err)
	}
	return nil
}

func (r *Repository) ListMarketOpsBacktestPromotionCandidates(ctx context.Context, filter storage.MarketOpsBacktestPromotionCandidateFilter) ([]storage.MarketOpsBacktestPromotionCandidateRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsBacktestPromotionCandidateSelect+`
WHERE ($1='' OR tenant_id=$1) AND ($2='' OR app_id=$2) AND ($3='' OR domain=$3) AND ($4='' OR use_case=$4)
 AND ($5='' OR baseline_id=$5) AND ($6='' OR comparison_id=$6) AND ($7='' OR evaluation_id=$7) AND ($8='' OR run_id=$8)
 AND ($9='' OR detector_id=$9) AND ($10='' OR dataset=$10) AND ($11='' OR readiness_status=$11) AND ($12='' OR status=$12)
ORDER BY created_at DESC LIMIT $13`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AppID), strings.TrimSpace(filter.Domain), strings.TrimSpace(filter.UseCase), strings.TrimSpace(filter.BaselineID), strings.TrimSpace(filter.ComparisonID), strings.TrimSpace(filter.EvaluationID), strings.TrimSpace(filter.RunID), strings.TrimSpace(filter.DetectorID), strings.TrimSpace(filter.Dataset), strings.TrimSpace(filter.ReadinessStatus), strings.TrimSpace(filter.Status), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops backtest promotion candidates: %w", err)
	}
	defer rows.Close()
	records := []storage.MarketOpsBacktestPromotionCandidateRecord{}
	for rows.Next() {
		record, err := scanMarketOpsBacktestPromotionCandidate(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list marketops backtest promotion candidates rows: %w", err)
	}
	return records, nil
}

func (r *Repository) GetMarketOpsBacktestPromotionCandidate(ctx context.Context, candidateID string) (storage.MarketOpsBacktestPromotionCandidateRecord, error) {
	record, err := scanMarketOpsBacktestPromotionCandidate(r.db.QueryRowContext(ctx, marketOpsBacktestPromotionCandidateSelect+` WHERE candidate_id=$1`, strings.TrimSpace(candidateID)))
	if err != nil {
		return storage.MarketOpsBacktestPromotionCandidateRecord{}, err
	}
	return record, nil
}

func (r *Repository) MutateMarketOpsBacktestPromotionCandidateDecision(ctx context.Context, mutation storage.MarketOpsBacktestPromotionCandidateDecisionMutation) (storage.MarketOpsBacktestPromotionCandidateRecord, error) {
	if strings.TrimSpace(mutation.CandidateID) == "" || strings.TrimSpace(mutation.Status) == "" {
		return storage.MarketOpsBacktestPromotionCandidateRecord{}, fmt.Errorf("marketops backtest promotion candidate_id and status are required")
	}
	if !marketOpsBacktestPromotionCandidateStatusAllowed(mutation.Status) {
		return storage.MarketOpsBacktestPromotionCandidateRecord{}, fmt.Errorf("marketops backtest promotion candidate status is invalid")
	}
	decidedAt := mutation.ReviewedAt.UTC()
	_, err := r.db.ExecContext(ctx, `UPDATE marketops_backtest_promotion_candidates SET status=$2, reviewed_by=$3, reviewed_at=$4, decision_note=$5, updated_at=now() WHERE candidate_id=$1`, strings.TrimSpace(mutation.CandidateID), strings.TrimSpace(mutation.Status), firstNonEmptyString(mutation.ReviewedBy, "operator-local"), decidedAt, strings.TrimSpace(mutation.DecisionNote))
	if err != nil {
		return storage.MarketOpsBacktestPromotionCandidateRecord{}, fmt.Errorf("mutate marketops backtest promotion candidate decision: %w", err)
	}
	return r.GetMarketOpsBacktestPromotionCandidate(ctx, mutation.CandidateID)
}

const marketOpsBacktestPromotionCandidateSelect = `SELECT candidate_id, tenant_id, app_id, domain, use_case, baseline_id, comparison_id, evaluation_id,
 run_id, detector_id, detector_version, dataset, policy_version, candidate_version, readiness_status,
 COALESCE(array_to_json(readiness_reasons), '[]'::json)::text, evidence, status, requested_by, reviewed_by, reviewed_at,
 decision_note, created_at, updated_at FROM marketops_backtest_promotion_candidates`

type marketOpsBacktestPromotionCandidateScanner interface{ Scan(dest ...any) error }

func scanMarketOpsBacktestPromotionCandidate(scanner marketOpsBacktestPromotionCandidateScanner) (storage.MarketOpsBacktestPromotionCandidateRecord, error) {
	var record storage.MarketOpsBacktestPromotionCandidateRecord
	var reasonsJSON string
	if err := scanner.Scan(&record.CandidateID, &record.TenantID, &record.AppID, &record.Domain, &record.UseCase, &record.BaselineID, &record.ComparisonID, &record.EvaluationID, &record.RunID, &record.DetectorID, &record.DetectorVersion, &record.Dataset, &record.PolicyVersion, &record.CandidateVersion, &record.ReadinessStatus, &reasonsJSON, &record.EvidenceJSON, &record.Status, &record.RequestedBy, &record.ReviewedBy, &record.ReviewedAt, &record.DecisionNote, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return storage.MarketOpsBacktestPromotionCandidateRecord{}, mapScanError("scan marketops backtest promotion candidate", err)
	}
	if err := json.Unmarshal([]byte(reasonsJSON), &record.ReadinessReasons); err != nil {
		return storage.MarketOpsBacktestPromotionCandidateRecord{}, err
	}
	return record, nil
}

func marketOpsBacktestPromotionCandidateStatusAllowed(status string) bool {
	switch strings.TrimSpace(status) {
	case storage.MarketOpsBacktestPromotionCandidateStatusProposed, storage.MarketOpsBacktestPromotionCandidateStatusApprovedForPromotion, storage.MarketOpsBacktestPromotionCandidateStatusRejected, storage.MarketOpsBacktestPromotionCandidateStatusDeferred, storage.MarketOpsBacktestPromotionCandidateStatusSuperseded:
		return true
	default:
		return false
	}
}
