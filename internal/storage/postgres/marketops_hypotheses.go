package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) UpsertMarketOpsHypothesisDefinition(ctx context.Context, record storage.MarketOpsHypothesisDefinitionRecord) error {
	if err := validateMarketOpsHypothesisDefinition(record); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO marketops_hypothesis_definitions (
 tenant_id, hypothesis_key, hypothesis_version, title, domain, direction, description, rationale,
 required_features, required_transitions, quality_policy, eligibility_expression, trigger_expression,
 persistence_rule, corroboration_rule, invalidation_rule, expected_outcomes, scoring_config,
 calibration_policy, lifecycle_status, owner, approved_by, approved_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)
ON CONFLICT (tenant_id, hypothesis_key, hypothesis_version) DO UPDATE SET
 title=EXCLUDED.title, domain=EXCLUDED.domain, direction=EXCLUDED.direction,
 description=EXCLUDED.description, rationale=EXCLUDED.rationale,
 required_features=EXCLUDED.required_features, required_transitions=EXCLUDED.required_transitions,
 quality_policy=EXCLUDED.quality_policy, eligibility_expression=EXCLUDED.eligibility_expression,
 trigger_expression=EXCLUDED.trigger_expression, persistence_rule=EXCLUDED.persistence_rule,
 corroboration_rule=EXCLUDED.corroboration_rule, invalidation_rule=EXCLUDED.invalidation_rule,
 expected_outcomes=EXCLUDED.expected_outcomes, scoring_config=EXCLUDED.scoring_config,
 calibration_policy=EXCLUDED.calibration_policy, lifecycle_status=EXCLUDED.lifecycle_status,
 owner=EXCLUDED.owner, approved_by=EXCLUDED.approved_by, approved_at=EXCLUDED.approved_at, updated_at=now()`,
		strings.TrimSpace(record.TenantID), strings.TrimSpace(record.HypothesisKey), strings.TrimSpace(record.HypothesisVersion),
		strings.TrimSpace(record.Title), strings.TrimSpace(record.Domain), strings.TrimSpace(record.Direction),
		strings.TrimSpace(record.Description), strings.TrimSpace(record.Rationale), jsonArrayOrEmpty(record.RequiredFeaturesJSON),
		jsonArrayOrEmpty(record.RequiredTransitionsJSON), jsonOrEmpty(record.QualityPolicyJSON), jsonOrEmpty(record.EligibilityExpressionJSON),
		jsonOrEmpty(record.TriggerExpressionJSON), jsonOrEmpty(record.PersistenceRuleJSON), jsonOrEmpty(record.CorroborationRuleJSON),
		jsonOrEmpty(record.InvalidationRuleJSON), jsonArrayOrEmpty(record.ExpectedOutcomesJSON), jsonOrEmpty(record.ScoringConfigJSON),
		jsonOrEmpty(record.CalibrationPolicyJSON), strings.TrimSpace(record.LifecycleStatus), nullString(record.Owner),
		nullString(record.ApprovedBy), record.ApprovedAt)
	if err != nil {
		return fmt.Errorf("upsert marketops hypothesis definition: %w", err)
	}
	return nil
}

func (r *Repository) ListMarketOpsHypothesisDefinitions(ctx context.Context, filter storage.MarketOpsHypothesisDefinitionFilter) ([]storage.MarketOpsHypothesisDefinitionRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsHypothesisDefinitionSelect+`
WHERE ($1='' OR tenant_id=$1) AND ($2='' OR hypothesis_key=$2) AND ($3='' OR hypothesis_version=$3)
 AND ($4='' OR domain=$4) AND ($5='' OR lifecycle_status=$5)
ORDER BY hypothesis_key, hypothesis_version DESC LIMIT $6`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.HypothesisKey),
		strings.TrimSpace(filter.HypothesisVersion), strings.TrimSpace(filter.Domain), strings.TrimSpace(filter.LifecycleStatus), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops hypothesis definitions: %w", err)
	}
	defer rows.Close()
	out := []storage.MarketOpsHypothesisDefinitionRecord{}
	for rows.Next() {
		record, err := scanMarketOpsHypothesisDefinition(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

func (r *Repository) GetMarketOpsHypothesisDefinition(ctx context.Context, tenantID, hypothesisKey, hypothesisVersion string) (storage.MarketOpsHypothesisDefinitionRecord, error) {
	return scanMarketOpsHypothesisDefinition(r.db.QueryRowContext(ctx, marketOpsHypothesisDefinitionSelect+` WHERE tenant_id=$1 AND hypothesis_key=$2 AND hypothesis_version=$3`, strings.TrimSpace(tenantID), strings.TrimSpace(hypothesisKey), strings.TrimSpace(hypothesisVersion)))
}

const marketOpsHypothesisDefinitionSelect = `
SELECT tenant_id, hypothesis_key, hypothesis_version, title, domain, direction, description, rationale,
 required_features, required_transitions, quality_policy, eligibility_expression, trigger_expression,
 persistence_rule, corroboration_rule, invalidation_rule, expected_outcomes, scoring_config,
 calibration_policy, lifecycle_status, COALESCE(owner,''), COALESCE(approved_by,''), approved_at, created_at, updated_at
FROM marketops_hypothesis_definitions`

func scanMarketOpsHypothesisDefinition(scanner interface{ Scan(...any) error }) (storage.MarketOpsHypothesisDefinitionRecord, error) {
	var record storage.MarketOpsHypothesisDefinitionRecord
	var approvedAt sql.NullTime
	err := scanner.Scan(&record.TenantID, &record.HypothesisKey, &record.HypothesisVersion, &record.Title, &record.Domain,
		&record.Direction, &record.Description, &record.Rationale, &record.RequiredFeaturesJSON, &record.RequiredTransitionsJSON,
		&record.QualityPolicyJSON, &record.EligibilityExpressionJSON, &record.TriggerExpressionJSON, &record.PersistenceRuleJSON,
		&record.CorroborationRuleJSON, &record.InvalidationRuleJSON, &record.ExpectedOutcomesJSON, &record.ScoringConfigJSON,
		&record.CalibrationPolicyJSON, &record.LifecycleStatus, &record.Owner, &record.ApprovedBy, &approvedAt, &record.CreatedAt, &record.UpdatedAt)
	if err != nil {
		return storage.MarketOpsHypothesisDefinitionRecord{}, mapScanError("scan marketops hypothesis definition", err)
	}
	if approvedAt.Valid {
		record.ApprovedAt = &approvedAt.Time
	}
	return record, nil
}

func (r *Repository) UpsertMarketOpsHypothesisEvaluation(ctx context.Context, record storage.MarketOpsHypothesisEvaluationRecord) error {
	if err := validateMarketOpsHypothesisEvaluation(record); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO marketops_hypothesis_evaluations (
 evaluation_id, tenant_id, app_id, hypothesis_key, hypothesis_version, market_state_id, asset_id, symbol,
 session_date, as_of_time, eligible, triggered, trigger_score, confidence_score, magnitude_score,
 rarity_score, persistence_score, corroboration_score, quality_score, invalidated, evidence_ids,
 reason_codes, evaluation_payload, evaluation_run_id, deterministic_key
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)
ON CONFLICT (tenant_id, deterministic_key) DO UPDATE SET
 app_id=EXCLUDED.app_id, eligible=EXCLUDED.eligible, triggered=EXCLUDED.triggered,
 trigger_score=EXCLUDED.trigger_score, confidence_score=EXCLUDED.confidence_score,
 magnitude_score=EXCLUDED.magnitude_score, rarity_score=EXCLUDED.rarity_score,
 persistence_score=EXCLUDED.persistence_score, corroboration_score=EXCLUDED.corroboration_score,
 quality_score=EXCLUDED.quality_score, invalidated=EXCLUDED.invalidated,
 evidence_ids=EXCLUDED.evidence_ids, reason_codes=EXCLUDED.reason_codes,
 evaluation_payload=EXCLUDED.evaluation_payload, evaluation_run_id=EXCLUDED.evaluation_run_id`,
		strings.TrimSpace(record.EvaluationID), strings.TrimSpace(record.TenantID), recordAppID(record.AppID),
		strings.TrimSpace(record.HypothesisKey), strings.TrimSpace(record.HypothesisVersion), strings.TrimSpace(record.MarketStateID),
		strings.TrimSpace(record.AssetID), strings.ToUpper(strings.TrimSpace(record.Symbol)), record.SessionDate.UTC(), record.AsOfTime.UTC(),
		record.Eligible, record.Triggered, record.TriggerScore, record.ConfidenceScore, record.MagnitudeScore, record.RarityScore,
		record.PersistenceScore, record.CorroborationScore, record.QualityScore, record.Invalidated, pqArray(record.EvidenceIDs),
		pqArray(record.ReasonCodes), jsonOrEmpty(record.EvaluationPayloadJSON), strings.TrimSpace(record.EvaluationRunID), strings.TrimSpace(record.DeterministicKey))
	if err != nil {
		return fmt.Errorf("upsert marketops hypothesis evaluation: %w", err)
	}
	return nil
}

func (r *Repository) ListMarketOpsHypothesisEvaluations(ctx context.Context, filter storage.MarketOpsHypothesisEvaluationFilter) ([]storage.MarketOpsHypothesisEvaluationRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsHypothesisEvaluationSelect+`
WHERE ($1='' OR tenant_id=$1) AND ($2='' OR app_id=$2) AND ($3='' OR hypothesis_key=$3)
 AND ($4='' OR hypothesis_version=$4) AND ($5='' OR market_state_id=$5) AND ($6='' OR asset_id=$6)
 AND ($7='' OR symbol=$7) AND ($8::boolean IS NULL OR eligible=$8) AND ($9::boolean IS NULL OR triggered=$9)
 AND ($10::boolean IS NULL OR invalidated=$10) AND ($11::timestamptz IS NULL OR session_date >= $11::date)
 AND ($12::timestamptz IS NULL OR session_date <= $12::date)
ORDER BY session_date DESC, hypothesis_key LIMIT $13`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AppID),
		strings.TrimSpace(filter.HypothesisKey), strings.TrimSpace(filter.HypothesisVersion), strings.TrimSpace(filter.MarketStateID),
		strings.TrimSpace(filter.AssetID), strings.ToUpper(strings.TrimSpace(filter.Symbol)), nullableBool(filter.Eligible),
		nullableBool(filter.Triggered), nullableBool(filter.Invalidated), nullTime(filter.SessionStart), nullTime(filter.SessionEnd), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops hypothesis evaluations: %w", err)
	}
	defer rows.Close()
	out := []storage.MarketOpsHypothesisEvaluationRecord{}
	for rows.Next() {
		record, err := scanMarketOpsHypothesisEvaluation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

const marketOpsHypothesisEvaluationSelect = `
SELECT evaluation_id, tenant_id, app_id, hypothesis_key, hypothesis_version, market_state_id, asset_id, symbol,
 session_date, as_of_time, eligible, triggered, trigger_score, confidence_score, magnitude_score, rarity_score,
 persistence_score, corroboration_score, quality_score, invalidated,
 COALESCE(array_to_json(evidence_ids),'[]'::json)::text, COALESCE(array_to_json(reason_codes),'[]'::json)::text,
 evaluation_payload, evaluation_run_id, deterministic_key, created_at
FROM marketops_hypothesis_evaluations`

func scanMarketOpsHypothesisEvaluation(scanner interface{ Scan(...any) error }) (storage.MarketOpsHypothesisEvaluationRecord, error) {
	var record storage.MarketOpsHypothesisEvaluationRecord
	var evidenceJSON, reasonsJSON string
	err := scanner.Scan(&record.EvaluationID, &record.TenantID, &record.AppID, &record.HypothesisKey, &record.HypothesisVersion,
		&record.MarketStateID, &record.AssetID, &record.Symbol, &record.SessionDate, &record.AsOfTime, &record.Eligible,
		&record.Triggered, &record.TriggerScore, &record.ConfidenceScore, &record.MagnitudeScore, &record.RarityScore,
		&record.PersistenceScore, &record.CorroborationScore, &record.QualityScore, &record.Invalidated,
		&evidenceJSON, &reasonsJSON, &record.EvaluationPayloadJSON, &record.EvaluationRunID, &record.DeterministicKey, &record.CreatedAt)
	if err != nil {
		return storage.MarketOpsHypothesisEvaluationRecord{}, mapScanError("scan marketops hypothesis evaluation", err)
	}
	if err := json.Unmarshal([]byte(evidenceJSON), &record.EvidenceIDs); err != nil {
		return storage.MarketOpsHypothesisEvaluationRecord{}, err
	}
	if err := json.Unmarshal([]byte(reasonsJSON), &record.ReasonCodes); err != nil {
		return storage.MarketOpsHypothesisEvaluationRecord{}, err
	}
	return record, nil
}

func validateMarketOpsHypothesisDefinition(record storage.MarketOpsHypothesisDefinitionRecord) error {
	if err := requireMarketOpsFields("hypothesis definition", map[string]string{"tenant_id": record.TenantID, "hypothesis_key": record.HypothesisKey, "hypothesis_version": record.HypothesisVersion, "title": record.Title, "domain": record.Domain, "direction": record.Direction, "lifecycle_status": record.LifecycleStatus}); err != nil {
		return err
	}
	if !oneOf(record.LifecycleStatus, storage.MarketOpsHypothesisLifecycleDraft, storage.MarketOpsHypothesisLifecycleResearch,
		storage.MarketOpsHypothesisLifecycleBacktestReady, storage.MarketOpsHypothesisLifecycleCalibration,
		storage.MarketOpsHypothesisLifecycleCandidate, storage.MarketOpsHypothesisLifecycleApproved,
		storage.MarketOpsHypothesisLifecyclePaused, storage.MarketOpsHypothesisLifecycleRetired) {
		return fmt.Errorf("marketops hypothesis lifecycle_status is invalid")
	}
	for name, value := range map[string][]byte{"required_features": record.RequiredFeaturesJSON, "required_transitions": record.RequiredTransitionsJSON, "expected_outcomes": record.ExpectedOutcomesJSON} {
		if err := validateJSONArray("marketops hypothesis "+name, jsonArrayOrEmpty(value)); err != nil {
			return err
		}
	}
	for name, value := range map[string][]byte{"quality_policy": record.QualityPolicyJSON, "eligibility_expression": record.EligibilityExpressionJSON, "trigger_expression": record.TriggerExpressionJSON, "persistence_rule": record.PersistenceRuleJSON, "corroboration_rule": record.CorroborationRuleJSON, "invalidation_rule": record.InvalidationRuleJSON, "scoring_config": record.ScoringConfigJSON, "calibration_policy": record.CalibrationPolicyJSON} {
		if err := validateJSONObject("marketops hypothesis "+name, jsonOrEmpty(value)); err != nil {
			return err
		}
	}
	return nil
}

func validateMarketOpsHypothesisEvaluation(record storage.MarketOpsHypothesisEvaluationRecord) error {
	if err := requireMarketOpsFields("hypothesis evaluation", map[string]string{"evaluation_id": record.EvaluationID, "tenant_id": record.TenantID, "hypothesis_key": record.HypothesisKey, "hypothesis_version": record.HypothesisVersion, "market_state_id": record.MarketStateID, "asset_id": record.AssetID, "symbol": record.Symbol, "evaluation_run_id": record.EvaluationRunID, "deterministic_key": record.DeterministicKey}); err != nil {
		return err
	}
	if record.SessionDate.IsZero() || record.AsOfTime.IsZero() {
		return fmt.Errorf("marketops hypothesis evaluation session_date and as_of_time are required")
	}
	if record.Triggered && !record.Eligible {
		return fmt.Errorf("marketops hypothesis evaluation triggered requires eligible")
	}
	for name, value := range map[string]*float64{"trigger_score": record.TriggerScore, "confidence_score": record.ConfidenceScore, "magnitude_score": record.MagnitudeScore, "rarity_score": record.RarityScore, "persistence_score": record.PersistenceScore, "corroboration_score": record.CorroborationScore, "quality_score": record.QualityScore} {
		if err := validateOptionalScore("marketops hypothesis evaluation "+name, value); err != nil {
			return err
		}
	}
	return validateJSONObject("marketops hypothesis evaluation payload", jsonOrEmpty(record.EvaluationPayloadJSON))
}

func nullableBool(value *bool) any {
	if value == nil {
		return nil
	}
	return *value
}
