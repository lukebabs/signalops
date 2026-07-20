package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) UpsertMarketOpsOpportunity(ctx context.Context, record storage.MarketOpsOpportunityRecord) error {
	if err := validateMarketOpsOpportunity(record); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO marketops_opportunities (
 opportunity_id, tenant_id, app_id, asset_id, symbol, opened_session_date, last_evaluated_date,
 direction, horizon, lifecycle_status, opportunity_score, confidence_score, domain_diversity_score,
 conflict_score, hypothesis_evaluation_ids, conflicting_evaluation_ids, signal_ids,
 supporting_evidence_ids, invalidating_evidence_ids, summary, opportunity_payload, version,
 research_only, build_run_id, deterministic_key
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)
ON CONFLICT (tenant_id, deterministic_key, version) DO UPDATE SET
 app_id=EXCLUDED.app_id, last_evaluated_date=EXCLUDED.last_evaluated_date,
 lifecycle_status=EXCLUDED.lifecycle_status, opportunity_score=EXCLUDED.opportunity_score,
 confidence_score=EXCLUDED.confidence_score, domain_diversity_score=EXCLUDED.domain_diversity_score,
 conflict_score=EXCLUDED.conflict_score, hypothesis_evaluation_ids=EXCLUDED.hypothesis_evaluation_ids,
 conflicting_evaluation_ids=EXCLUDED.conflicting_evaluation_ids, signal_ids=EXCLUDED.signal_ids,
 supporting_evidence_ids=EXCLUDED.supporting_evidence_ids,
 invalidating_evidence_ids=EXCLUDED.invalidating_evidence_ids, summary=EXCLUDED.summary,
 opportunity_payload=EXCLUDED.opportunity_payload, research_only=EXCLUDED.research_only,
 build_run_id=EXCLUDED.build_run_id, updated_at=now()`,
		record.OpportunityID, strings.TrimSpace(record.TenantID), recordAppID(record.AppID),
		strings.TrimSpace(record.AssetID), strings.ToUpper(strings.TrimSpace(record.Symbol)),
		record.OpenedSessionDate.UTC(), record.LastEvaluatedDate.UTC(), strings.TrimSpace(record.Direction),
		strings.TrimSpace(record.Horizon), strings.TrimSpace(record.LifecycleStatus), record.OpportunityScore,
		record.ConfidenceScore, record.DomainDiversityScore, record.ConflictScore,
		pqArray(record.HypothesisEvaluationIDs), pqArray(record.ConflictingEvaluationIDs), pqArray(record.SignalIDs),
		pqArray(record.SupportingEvidenceIDs), pqArray(record.InvalidatingEvidenceIDs), strings.TrimSpace(record.Summary),
		jsonOrEmpty(record.OpportunityPayloadJSON), record.Version, record.ResearchOnly,
		strings.TrimSpace(record.BuildRunID), strings.TrimSpace(record.DeterministicKey))
	if err != nil {
		return fmt.Errorf("upsert marketops opportunity: %w", err)
	}
	return nil
}

func (r *Repository) ListMarketOpsOpportunities(ctx context.Context, filter storage.MarketOpsOpportunityFilter) ([]storage.MarketOpsOpportunityRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsOpportunitySelect+`
WHERE ($1='' OR tenant_id=$1) AND ($2='' OR app_id=$2) AND ($3='' OR opportunity_id=$3)
 AND ($4='' OR asset_id=$4) AND ($5='' OR symbol=$5) AND ($6='' OR direction=$6)
 AND ($7='' OR horizon=$7) AND ($8='' OR lifecycle_status=$8)
 AND ($9::boolean IS NULL OR research_only=$9)
 AND ($10::timestamptz IS NULL OR last_evaluated_date >= $10::date)
 AND ($11::timestamptz IS NULL OR last_evaluated_date <= $11::date)
ORDER BY opportunity_score DESC, last_evaluated_date DESC, opportunity_id LIMIT $12`,
		strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AppID), strings.TrimSpace(filter.OpportunityID),
		strings.TrimSpace(filter.AssetID), strings.ToUpper(strings.TrimSpace(filter.Symbol)), strings.TrimSpace(filter.Direction),
		strings.TrimSpace(filter.Horizon), strings.TrimSpace(filter.LifecycleStatus), nullableBool(filter.ResearchOnly),
		nullTime(filter.SessionStart), nullTime(filter.SessionEnd), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops opportunities: %w", err)
	}
	defer rows.Close()
	out := []storage.MarketOpsOpportunityRecord{}
	for rows.Next() {
		record, err := scanMarketOpsOpportunity(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

func (r *Repository) GetMarketOpsOpportunity(ctx context.Context, tenantID, opportunityID string) (storage.MarketOpsOpportunityRecord, error) {
	return scanMarketOpsOpportunity(r.db.QueryRowContext(ctx, marketOpsOpportunitySelect+` WHERE tenant_id=$1 AND opportunity_id=$2`, strings.TrimSpace(tenantID), strings.TrimSpace(opportunityID)))
}

const marketOpsOpportunitySelect = `
SELECT opportunity_id, tenant_id, app_id, asset_id, symbol, opened_session_date, last_evaluated_date,
 direction, horizon, lifecycle_status, opportunity_score, confidence_score, domain_diversity_score,
 conflict_score, COALESCE(array_to_json(hypothesis_evaluation_ids),'[]'::json)::text,
 COALESCE(array_to_json(conflicting_evaluation_ids),'[]'::json)::text,
 COALESCE(array_to_json(signal_ids),'[]'::json)::text,
 COALESCE(array_to_json(supporting_evidence_ids),'[]'::json)::text,
 COALESCE(array_to_json(invalidating_evidence_ids),'[]'::json)::text,
 summary, opportunity_payload, version, research_only, build_run_id, deterministic_key, created_at, updated_at
FROM marketops_opportunities`

func scanMarketOpsOpportunity(scanner interface{ Scan(...any) error }) (storage.MarketOpsOpportunityRecord, error) {
	var record storage.MarketOpsOpportunityRecord
	var evaluationsJSON, conflictsJSON, signalsJSON, evidenceJSON, invalidatingJSON string
	err := scanner.Scan(&record.OpportunityID, &record.TenantID, &record.AppID, &record.AssetID, &record.Symbol,
		&record.OpenedSessionDate, &record.LastEvaluatedDate, &record.Direction, &record.Horizon,
		&record.LifecycleStatus, &record.OpportunityScore, &record.ConfidenceScore,
		&record.DomainDiversityScore, &record.ConflictScore, &evaluationsJSON, &conflictsJSON,
		&signalsJSON, &evidenceJSON, &invalidatingJSON, &record.Summary, &record.OpportunityPayloadJSON,
		&record.Version, &record.ResearchOnly, &record.BuildRunID, &record.DeterministicKey,
		&record.CreatedAt, &record.UpdatedAt)
	if err != nil {
		return storage.MarketOpsOpportunityRecord{}, mapScanError("scan marketops opportunity", err)
	}
	for _, item := range []struct {
		raw         string
		destination *[]string
	}{
		{evaluationsJSON, &record.HypothesisEvaluationIDs},
		{conflictsJSON, &record.ConflictingEvaluationIDs},
		{signalsJSON, &record.SignalIDs},
		{evidenceJSON, &record.SupportingEvidenceIDs},
		{invalidatingJSON, &record.InvalidatingEvidenceIDs},
	} {
		if err := json.Unmarshal([]byte(item.raw), item.destination); err != nil {
			return storage.MarketOpsOpportunityRecord{}, err
		}
	}
	return record, nil
}

func validateMarketOpsOpportunity(record storage.MarketOpsOpportunityRecord) error {
	if err := requireMarketOpsFields("opportunity", map[string]string{
		"opportunity_id": record.OpportunityID, "tenant_id": record.TenantID, "asset_id": record.AssetID,
		"symbol": record.Symbol, "direction": record.Direction, "horizon": record.Horizon,
		"lifecycle_status": record.LifecycleStatus, "summary": record.Summary,
		"build_run_id": record.BuildRunID, "deterministic_key": record.DeterministicKey,
	}); err != nil {
		return err
	}
	if record.OpenedSessionDate.IsZero() || record.LastEvaluatedDate.IsZero() || record.LastEvaluatedDate.Before(record.OpenedSessionDate) {
		return fmt.Errorf("marketops opportunity session dates are invalid")
	}
	if !oneOf(record.Direction, "upside", "downside", "non_directional") {
		return fmt.Errorf("marketops opportunity direction is invalid")
	}
	if !oneOf(record.LifecycleStatus, storage.MarketOpsOpportunityEmerging, storage.MarketOpsOpportunityActive,
		storage.MarketOpsOpportunityStrengthening, storage.MarketOpsOpportunityWeakening,
		storage.MarketOpsOpportunityInvalidated, storage.MarketOpsOpportunityResolved, storage.MarketOpsOpportunityExpired) {
		return fmt.Errorf("marketops opportunity lifecycle_status is invalid")
	}
	if record.Version <= 0 || len(record.HypothesisEvaluationIDs) == 0 {
		return fmt.Errorf("marketops opportunity version and hypothesis evaluations are required")
	}
	for name, value := range map[string]float64{
		"opportunity_score": record.OpportunityScore, "confidence_score": record.ConfidenceScore,
		"domain_diversity_score": record.DomainDiversityScore, "conflict_score": record.ConflictScore,
	} {
		if value < 0 || value > 1 {
			return fmt.Errorf("marketops opportunity %s must be between 0 and 1", name)
		}
	}
	return validateJSONObject("marketops opportunity payload", jsonOrEmpty(record.OpportunityPayloadJSON))
}
