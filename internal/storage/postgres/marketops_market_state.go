package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) UpsertMarketOpsFeatureDefinition(ctx context.Context, record storage.MarketOpsFeatureDefinitionRecord) error {
	if err := validateMarketOpsFeatureDefinition(record); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO marketops_feature_definitions (
 tenant_id, feature_key, feature_version, domain, title, description, value_type, unit,
 calculation_spec, required_inputs, quality_policy, status
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
ON CONFLICT (tenant_id, feature_key, feature_version) DO UPDATE SET
 title=EXCLUDED.title, description=EXCLUDED.description, status=EXCLUDED.status, updated_at=now()`,
		strings.TrimSpace(record.TenantID), strings.TrimSpace(record.FeatureKey), strings.TrimSpace(record.FeatureVersion),
		strings.TrimSpace(record.Domain), strings.TrimSpace(record.Title), strings.TrimSpace(record.Description),
		strings.TrimSpace(record.ValueType), nullString(record.Unit), jsonOrEmpty(record.CalculationSpec),
		jsonArrayOrEmpty(record.RequiredInputs), jsonOrEmpty(record.QualityPolicy), strings.TrimSpace(record.Status))
	if err != nil {
		return fmt.Errorf("upsert marketops feature definition: %w", err)
	}
	return nil
}

func (r *Repository) ListMarketOpsFeatureDefinitions(ctx context.Context, filter storage.MarketOpsFeatureDefinitionFilter) ([]storage.MarketOpsFeatureDefinitionRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsFeatureDefinitionSelect+`
WHERE ($1='' OR tenant_id=$1) AND ($2='' OR feature_key=$2) AND ($3='' OR feature_version=$3)
 AND ($4='' OR domain=$4) AND ($5='' OR status=$5)
ORDER BY feature_key, feature_version DESC LIMIT $6`,
		strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.FeatureKey), strings.TrimSpace(filter.FeatureVersion),
		strings.TrimSpace(filter.Domain), strings.TrimSpace(filter.Status), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops feature definitions: %w", err)
	}
	defer rows.Close()
	records := []storage.MarketOpsFeatureDefinitionRecord{}
	for rows.Next() {
		record, err := scanMarketOpsFeatureDefinition(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list marketops feature definitions rows: %w", err)
	}
	return records, nil
}

const marketOpsFeatureDefinitionSelect = `
SELECT tenant_id, feature_key, feature_version, domain, title, description, value_type,
 COALESCE(unit, ''), calculation_spec, required_inputs, quality_policy, status, created_at, updated_at
FROM marketops_feature_definitions `

func scanMarketOpsFeatureDefinition(scanner marketOpsMarketStateScanner) (storage.MarketOpsFeatureDefinitionRecord, error) {
	var record storage.MarketOpsFeatureDefinitionRecord
	if err := scanner.Scan(&record.TenantID, &record.FeatureKey, &record.FeatureVersion, &record.Domain,
		&record.Title, &record.Description, &record.ValueType, &record.Unit, &record.CalculationSpec,
		&record.RequiredInputs, &record.QualityPolicy, &record.Status, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return storage.MarketOpsFeatureDefinitionRecord{}, mapScanError("scan marketops feature definition", err)
	}
	return record, nil
}

func (r *Repository) UpsertMarketOpsFeatureObservation(ctx context.Context, record storage.MarketOpsFeatureObservationRecord) error {
	if err := validateMarketOpsFeatureObservation(record); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO marketops_feature_observations (
 feature_observation_id, tenant_id, app_id, asset_id, symbol, session_date, as_of_time,
 feature_key, feature_version, dimensions, numeric_value, text_value, boolean_value,
 quality_state, quality_score, quality_details, source_event_ids, source_artifact_ids,
 calculation_run_id, deterministic_key
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
ON CONFLICT (tenant_id, deterministic_key) DO UPDATE SET
 app_id=EXCLUDED.app_id, asset_id=EXCLUDED.asset_id, symbol=EXCLUDED.symbol,
 session_date=EXCLUDED.session_date, as_of_time=EXCLUDED.as_of_time,
 feature_key=EXCLUDED.feature_key, feature_version=EXCLUDED.feature_version,
 dimensions=EXCLUDED.dimensions, numeric_value=EXCLUDED.numeric_value, text_value=EXCLUDED.text_value,
 boolean_value=EXCLUDED.boolean_value, quality_state=EXCLUDED.quality_state,
 quality_score=EXCLUDED.quality_score, quality_details=EXCLUDED.quality_details,
 source_event_ids=EXCLUDED.source_event_ids, source_artifact_ids=EXCLUDED.source_artifact_ids,
 calculation_run_id=EXCLUDED.calculation_run_id`,
		strings.TrimSpace(record.FeatureObservationID), strings.TrimSpace(record.TenantID), recordAppID(record.AppID),
		strings.TrimSpace(record.AssetID), strings.ToUpper(strings.TrimSpace(record.Symbol)), record.SessionDate.UTC(), record.AsOfTime.UTC(),
		strings.TrimSpace(record.FeatureKey), strings.TrimSpace(record.FeatureVersion), jsonOrEmpty(record.DimensionsJSON),
		record.NumericValue, record.TextValue, record.BooleanValue, strings.TrimSpace(record.QualityState), record.QualityScore,
		jsonOrEmpty(record.QualityDetailsJSON), pqArray(record.SourceEventIDs), pqArray(record.SourceArtifactIDs),
		strings.TrimSpace(record.CalculationRunID), strings.TrimSpace(record.DeterministicKey))
	if err != nil {
		return fmt.Errorf("upsert marketops feature observation: %w", err)
	}
	return nil
}

func (r *Repository) ListMarketOpsFeatureObservations(ctx context.Context, filter storage.MarketOpsFeatureObservationFilter) ([]storage.MarketOpsFeatureObservationRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsFeatureObservationSelect+`
JOIN marketops_feature_definitions d
 ON d.tenant_id=o.tenant_id AND d.feature_key=o.feature_key AND d.feature_version=o.feature_version
WHERE ($1='' OR o.tenant_id=$1) AND ($2='' OR o.app_id=$2) AND ($3='' OR o.asset_id=$3)
 AND ($4='' OR o.symbol=$4) AND ($5='' OR o.feature_key=$5) AND ($6='' OR o.feature_version=$6)
 AND ($7='' OR d.domain=$7) AND ($8='' OR o.quality_state=$8)
 AND ($9::jsonb = '{}'::jsonb OR o.dimensions @> $9::jsonb)
 AND ($10::timestamptz IS NULL OR o.session_date >= $10::date)
 AND ($11::timestamptz IS NULL OR o.session_date <= $11::date)
 AND (cardinality($12::text[]) = 0 OR o.feature_observation_id = ANY($12::text[]))
ORDER BY o.session_date DESC, o.as_of_time DESC, o.feature_key LIMIT $13`,
		strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AppID), strings.TrimSpace(filter.AssetID),
		strings.ToUpper(strings.TrimSpace(filter.Symbol)), strings.TrimSpace(filter.FeatureKey), strings.TrimSpace(filter.FeatureVersion),
		strings.TrimSpace(filter.Domain), strings.TrimSpace(filter.QualityState), jsonOrEmpty(filter.DimensionsJSON),
		nullTime(filter.SessionStart), nullTime(filter.SessionEnd), pqArray(filter.FeatureObservationIDs), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops feature observations: %w", err)
	}
	defer rows.Close()
	records := []storage.MarketOpsFeatureObservationRecord{}
	for rows.Next() {
		record, err := scanMarketOpsFeatureObservation(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list marketops feature observations rows: %w", err)
	}
	return records, nil
}

const marketOpsFeatureObservationSelect = `
SELECT o.feature_observation_id, o.tenant_id, o.app_id, o.asset_id, o.symbol, o.session_date,
 o.as_of_time, o.feature_key, o.feature_version, o.dimensions, o.numeric_value, o.text_value,
 o.boolean_value, o.quality_state, o.quality_score, o.quality_details,
 COALESCE(array_to_json(o.source_event_ids), '[]'::json)::text,
 COALESCE(array_to_json(o.source_artifact_ids), '[]'::json)::text,
 o.calculation_run_id, o.deterministic_key, o.created_at
FROM marketops_feature_observations o `

func scanMarketOpsFeatureObservation(scanner marketOpsMarketStateScanner) (storage.MarketOpsFeatureObservationRecord, error) {
	var record storage.MarketOpsFeatureObservationRecord
	var eventIDsJSON, artifactIDsJSON string
	if err := scanner.Scan(&record.FeatureObservationID, &record.TenantID, &record.AppID, &record.AssetID,
		&record.Symbol, &record.SessionDate, &record.AsOfTime, &record.FeatureKey, &record.FeatureVersion,
		&record.DimensionsJSON, &record.NumericValue, &record.TextValue, &record.BooleanValue,
		&record.QualityState, &record.QualityScore, &record.QualityDetailsJSON, &eventIDsJSON,
		&artifactIDsJSON, &record.CalculationRunID, &record.DeterministicKey, &record.CreatedAt); err != nil {
		return storage.MarketOpsFeatureObservationRecord{}, mapScanError("scan marketops feature observation", err)
	}
	if err := json.Unmarshal([]byte(eventIDsJSON), &record.SourceEventIDs); err != nil {
		return storage.MarketOpsFeatureObservationRecord{}, fmt.Errorf("scan marketops feature observation event ids: %w", err)
	}
	if err := json.Unmarshal([]byte(artifactIDsJSON), &record.SourceArtifactIDs); err != nil {
		return storage.MarketOpsFeatureObservationRecord{}, fmt.Errorf("scan marketops feature observation artifact ids: %w", err)
	}
	return record, nil
}

func (r *Repository) UpsertMarketOpsMarketState(ctx context.Context, record storage.MarketOpsMarketStateRecord) error {
	if err := validateMarketOpsMarketState(record); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO marketops_market_states (
 market_state_id, tenant_id, app_id, asset_id, symbol, session_date, as_of_time,
 state_schema_version, state_payload, feature_observation_ids, feature_count,
 required_feature_count, completeness_ratio, quality_state, quality_score, quality_summary,
 eligible_hypotheses, build_run_id, deterministic_key
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
ON CONFLICT (tenant_id, deterministic_key) DO UPDATE SET
 app_id=EXCLUDED.app_id, asset_id=EXCLUDED.asset_id, symbol=EXCLUDED.symbol,
 session_date=EXCLUDED.session_date, as_of_time=EXCLUDED.as_of_time,
 state_schema_version=EXCLUDED.state_schema_version, state_payload=EXCLUDED.state_payload,
 feature_observation_ids=EXCLUDED.feature_observation_ids, feature_count=EXCLUDED.feature_count,
 required_feature_count=EXCLUDED.required_feature_count, completeness_ratio=EXCLUDED.completeness_ratio,
 quality_state=EXCLUDED.quality_state, quality_score=EXCLUDED.quality_score,
 quality_summary=EXCLUDED.quality_summary, eligible_hypotheses=EXCLUDED.eligible_hypotheses,
 build_run_id=EXCLUDED.build_run_id`,
		strings.TrimSpace(record.MarketStateID), strings.TrimSpace(record.TenantID), recordAppID(record.AppID),
		strings.TrimSpace(record.AssetID), strings.ToUpper(strings.TrimSpace(record.Symbol)), record.SessionDate.UTC(),
		record.AsOfTime.UTC(), strings.TrimSpace(record.StateSchemaVersion), jsonOrEmpty(record.StatePayloadJSON),
		pqArray(record.FeatureObservationIDs), record.FeatureCount, record.RequiredFeatureCount, record.CompletenessRatio,
		strings.TrimSpace(record.QualityState), record.QualityScore, jsonOrEmpty(record.QualitySummaryJSON),
		pqArray(record.EligibleHypotheses), strings.TrimSpace(record.BuildRunID), strings.TrimSpace(record.DeterministicKey))
	if err != nil {
		return fmt.Errorf("upsert marketops market state: %w", err)
	}
	return nil
}

func (r *Repository) ListMarketOpsMarketStates(ctx context.Context, filter storage.MarketOpsMarketStateFilter) ([]storage.MarketOpsMarketStateRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsMarketStateSelect+`
WHERE ($1='' OR tenant_id=$1) AND ($2='' OR app_id=$2) AND ($3='' OR asset_id=$3)
 AND ($4='' OR symbol=$4) AND ($5='' OR state_schema_version=$5) AND ($6='' OR quality_state=$6)
 AND ($7::timestamptz IS NULL OR session_date >= $7::date)
 AND ($8::timestamptz IS NULL OR session_date <= $8::date)
ORDER BY session_date DESC, as_of_time DESC LIMIT $9`,
		strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AppID), strings.TrimSpace(filter.AssetID),
		strings.ToUpper(strings.TrimSpace(filter.Symbol)), strings.TrimSpace(filter.StateSchemaVersion),
		strings.TrimSpace(filter.QualityState), nullTime(filter.SessionStart), nullTime(filter.SessionEnd), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops market states: %w", err)
	}
	defer rows.Close()
	records := []storage.MarketOpsMarketStateRecord{}
	for rows.Next() {
		record, err := scanMarketOpsMarketState(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list marketops market states rows: %w", err)
	}
	return records, nil
}

func (r *Repository) GetMarketOpsMarketState(ctx context.Context, marketStateID string) (storage.MarketOpsMarketStateRecord, error) {
	return scanMarketOpsMarketState(r.db.QueryRowContext(ctx, marketOpsMarketStateSelect+`WHERE market_state_id=$1`, strings.TrimSpace(marketStateID)))
}

const marketOpsMarketStateSelect = `
SELECT market_state_id, tenant_id, app_id, asset_id, symbol, session_date, as_of_time,
 state_schema_version, state_payload, COALESCE(array_to_json(feature_observation_ids), '[]'::json)::text,
 feature_count, required_feature_count, completeness_ratio, quality_state, quality_score, quality_summary,
 COALESCE(array_to_json(eligible_hypotheses), '[]'::json)::text, build_run_id, deterministic_key, created_at
FROM marketops_market_states `

func scanMarketOpsMarketState(scanner marketOpsMarketStateScanner) (storage.MarketOpsMarketStateRecord, error) {
	var record storage.MarketOpsMarketStateRecord
	var observationIDsJSON, hypothesesJSON string
	if err := scanner.Scan(&record.MarketStateID, &record.TenantID, &record.AppID, &record.AssetID,
		&record.Symbol, &record.SessionDate, &record.AsOfTime, &record.StateSchemaVersion,
		&record.StatePayloadJSON, &observationIDsJSON, &record.FeatureCount, &record.RequiredFeatureCount,
		&record.CompletenessRatio, &record.QualityState, &record.QualityScore, &record.QualitySummaryJSON,
		&hypothesesJSON, &record.BuildRunID, &record.DeterministicKey, &record.CreatedAt); err != nil {
		return storage.MarketOpsMarketStateRecord{}, mapScanError("scan marketops market state", err)
	}
	if err := json.Unmarshal([]byte(observationIDsJSON), &record.FeatureObservationIDs); err != nil {
		return storage.MarketOpsMarketStateRecord{}, fmt.Errorf("scan marketops market state feature ids: %w", err)
	}
	if err := json.Unmarshal([]byte(hypothesesJSON), &record.EligibleHypotheses); err != nil {
		return storage.MarketOpsMarketStateRecord{}, fmt.Errorf("scan marketops market state hypotheses: %w", err)
	}
	return record, nil
}

func (r *Repository) UpsertMarketOpsStateTransition(ctx context.Context, record storage.MarketOpsStateTransitionRecord) error {
	if err := validateMarketOpsStateTransition(record); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO marketops_state_transitions (
 transition_id, tenant_id, app_id, asset_id, symbol, session_date, as_of_time,
 current_state_id, baseline_state_id, feature_key, feature_version, dimensions,
 transition_type, lookback_sessions, current_value, baseline_value, transition_value,
 zscore, percentile, persistence_sessions, direction, quality_state, transition_payload,
 calculation_run_id, deterministic_key
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)
ON CONFLICT (tenant_id, deterministic_key) DO UPDATE SET
 app_id=EXCLUDED.app_id, asset_id=EXCLUDED.asset_id, symbol=EXCLUDED.symbol,
 session_date=EXCLUDED.session_date, as_of_time=EXCLUDED.as_of_time,
 current_state_id=EXCLUDED.current_state_id, baseline_state_id=EXCLUDED.baseline_state_id,
 feature_key=EXCLUDED.feature_key, feature_version=EXCLUDED.feature_version,
 dimensions=EXCLUDED.dimensions, transition_type=EXCLUDED.transition_type,
 lookback_sessions=EXCLUDED.lookback_sessions, current_value=EXCLUDED.current_value,
 baseline_value=EXCLUDED.baseline_value, transition_value=EXCLUDED.transition_value,
 zscore=EXCLUDED.zscore, percentile=EXCLUDED.percentile,
 persistence_sessions=EXCLUDED.persistence_sessions, direction=EXCLUDED.direction,
 quality_state=EXCLUDED.quality_state, transition_payload=EXCLUDED.transition_payload,
 calculation_run_id=EXCLUDED.calculation_run_id`,
		strings.TrimSpace(record.TransitionID), strings.TrimSpace(record.TenantID), recordAppID(record.AppID),
		strings.TrimSpace(record.AssetID), strings.ToUpper(strings.TrimSpace(record.Symbol)), record.SessionDate.UTC(),
		record.AsOfTime.UTC(), strings.TrimSpace(record.CurrentStateID), nullString(record.BaselineStateID),
		strings.TrimSpace(record.FeatureKey), strings.TrimSpace(record.FeatureVersion), jsonOrEmpty(record.DimensionsJSON),
		strings.TrimSpace(record.TransitionType), record.LookbackSessions, record.CurrentValue, record.BaselineValue,
		record.TransitionValue, record.ZScore, record.Percentile, record.PersistenceSessions, nullString(record.Direction),
		strings.TrimSpace(record.QualityState), jsonOrEmpty(record.TransitionPayloadJSON),
		strings.TrimSpace(record.CalculationRunID), strings.TrimSpace(record.DeterministicKey))
	if err != nil {
		return fmt.Errorf("upsert marketops state transition: %w", err)
	}
	return nil
}

func (r *Repository) ListMarketOpsStateTransitions(ctx context.Context, filter storage.MarketOpsStateTransitionFilter) ([]storage.MarketOpsStateTransitionRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsStateTransitionSelect+`
WHERE ($1='' OR tenant_id=$1) AND ($2='' OR app_id=$2) AND ($3='' OR asset_id=$3)
 AND ($4='' OR symbol=$4) AND ($5='' OR current_state_id=$5) AND ($6='' OR feature_key=$6)
 AND ($7='' OR feature_version=$7) AND ($8='' OR transition_type=$8) AND ($9='' OR quality_state=$9)
 AND ($10::timestamptz IS NULL OR session_date >= $10::date)
 AND ($11::timestamptz IS NULL OR session_date <= $11::date)
ORDER BY session_date DESC, as_of_time DESC LIMIT $12`,
		strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AppID), strings.TrimSpace(filter.AssetID),
		strings.ToUpper(strings.TrimSpace(filter.Symbol)), strings.TrimSpace(filter.CurrentStateID),
		strings.TrimSpace(filter.FeatureKey), strings.TrimSpace(filter.FeatureVersion), strings.TrimSpace(filter.TransitionType),
		strings.TrimSpace(filter.QualityState), nullTime(filter.SessionStart), nullTime(filter.SessionEnd), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops state transitions: %w", err)
	}
	defer rows.Close()
	records := []storage.MarketOpsStateTransitionRecord{}
	for rows.Next() {
		record, err := scanMarketOpsStateTransition(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list marketops state transitions rows: %w", err)
	}
	return records, nil
}

const marketOpsStateTransitionSelect = `
SELECT transition_id, tenant_id, app_id, asset_id, symbol, session_date, as_of_time,
 current_state_id, COALESCE(baseline_state_id, ''), feature_key, feature_version, dimensions,
 transition_type, lookback_sessions, current_value, baseline_value, transition_value,
 zscore, percentile, persistence_sessions, COALESCE(direction, ''), quality_state,
 transition_payload, calculation_run_id, deterministic_key, created_at
FROM marketops_state_transitions `

func scanMarketOpsStateTransition(scanner marketOpsMarketStateScanner) (storage.MarketOpsStateTransitionRecord, error) {
	var record storage.MarketOpsStateTransitionRecord
	if err := scanner.Scan(&record.TransitionID, &record.TenantID, &record.AppID, &record.AssetID,
		&record.Symbol, &record.SessionDate, &record.AsOfTime, &record.CurrentStateID, &record.BaselineStateID,
		&record.FeatureKey, &record.FeatureVersion, &record.DimensionsJSON, &record.TransitionType,
		&record.LookbackSessions, &record.CurrentValue, &record.BaselineValue, &record.TransitionValue,
		&record.ZScore, &record.Percentile, &record.PersistenceSessions, &record.Direction,
		&record.QualityState, &record.TransitionPayloadJSON, &record.CalculationRunID,
		&record.DeterministicKey, &record.CreatedAt); err != nil {
		return storage.MarketOpsStateTransitionRecord{}, mapScanError("scan marketops state transition", err)
	}
	return record, nil
}

func (r *Repository) UpsertMarketOpsEvidence(ctx context.Context, record storage.MarketOpsEvidenceRecord) error {
	if err := validateMarketOpsEvidence(record); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO marketops_evidence (
 evidence_id, tenant_id, app_id, asset_id, symbol, session_date, as_of_time,
 evidence_type, evidence_version, domain, direction, magnitude, rarity_score,
 persistence_score, quality_score, statement, evidence_payload, source_feature_ids,
 source_transition_ids, deterministic_key
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
ON CONFLICT (tenant_id, deterministic_key) DO UPDATE SET
 app_id=EXCLUDED.app_id, asset_id=EXCLUDED.asset_id, symbol=EXCLUDED.symbol,
 session_date=EXCLUDED.session_date, as_of_time=EXCLUDED.as_of_time,
 evidence_type=EXCLUDED.evidence_type, evidence_version=EXCLUDED.evidence_version,
 domain=EXCLUDED.domain, direction=EXCLUDED.direction, magnitude=EXCLUDED.magnitude,
 rarity_score=EXCLUDED.rarity_score, persistence_score=EXCLUDED.persistence_score,
 quality_score=EXCLUDED.quality_score, statement=EXCLUDED.statement,
 evidence_payload=EXCLUDED.evidence_payload, source_feature_ids=EXCLUDED.source_feature_ids,
 source_transition_ids=EXCLUDED.source_transition_ids`,
		strings.TrimSpace(record.EvidenceID), strings.TrimSpace(record.TenantID), recordAppID(record.AppID),
		strings.TrimSpace(record.AssetID), strings.ToUpper(strings.TrimSpace(record.Symbol)), record.SessionDate.UTC(),
		record.AsOfTime.UTC(), strings.TrimSpace(record.EvidenceType), strings.TrimSpace(record.EvidenceVersion),
		strings.TrimSpace(record.Domain), nullString(record.Direction), record.Magnitude, record.RarityScore,
		record.PersistenceScore, record.QualityScore, strings.TrimSpace(record.Statement), jsonOrEmpty(record.EvidencePayloadJSON),
		pqArray(record.SourceFeatureIDs), pqArray(record.SourceTransitionIDs), strings.TrimSpace(record.DeterministicKey))
	if err != nil {
		return fmt.Errorf("upsert marketops evidence: %w", err)
	}
	return nil
}

func (r *Repository) ListMarketOpsEvidence(ctx context.Context, filter storage.MarketOpsEvidenceFilter) ([]storage.MarketOpsEvidenceRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsEvidenceSelect+`
WHERE ($1='' OR tenant_id=$1) AND ($2='' OR app_id=$2) AND ($3='' OR asset_id=$3)
 AND ($4='' OR symbol=$4) AND ($5='' OR evidence_type=$5) AND ($6='' OR evidence_version=$6)
 AND ($7='' OR domain=$7) AND ($8='' OR direction=$8)
 AND ($9::timestamptz IS NULL OR session_date >= $9::date)
 AND ($10::timestamptz IS NULL OR session_date <= $10::date)
ORDER BY session_date DESC, as_of_time DESC LIMIT $11`,
		strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AppID), strings.TrimSpace(filter.AssetID),
		strings.ToUpper(strings.TrimSpace(filter.Symbol)), strings.TrimSpace(filter.EvidenceType),
		strings.TrimSpace(filter.EvidenceVersion), strings.TrimSpace(filter.Domain), strings.TrimSpace(filter.Direction),
		nullTime(filter.SessionStart), nullTime(filter.SessionEnd), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops evidence: %w", err)
	}
	defer rows.Close()
	records := []storage.MarketOpsEvidenceRecord{}
	for rows.Next() {
		record, err := scanMarketOpsEvidence(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list marketops evidence rows: %w", err)
	}
	return records, nil
}

func (r *Repository) GetMarketOpsEvidence(ctx context.Context, evidenceID string) (storage.MarketOpsEvidenceRecord, error) {
	return scanMarketOpsEvidence(r.db.QueryRowContext(ctx, marketOpsEvidenceSelect+`WHERE evidence_id=$1`, strings.TrimSpace(evidenceID)))
}

const marketOpsEvidenceSelect = `
SELECT evidence_id, tenant_id, app_id, asset_id, symbol, session_date, as_of_time,
 evidence_type, evidence_version, domain, COALESCE(direction, ''), magnitude, rarity_score,
 persistence_score, quality_score, statement, evidence_payload,
 COALESCE(array_to_json(source_feature_ids), '[]'::json)::text,
 COALESCE(array_to_json(source_transition_ids), '[]'::json)::text,
 deterministic_key, created_at
FROM marketops_evidence `

func scanMarketOpsEvidence(scanner marketOpsMarketStateScanner) (storage.MarketOpsEvidenceRecord, error) {
	var record storage.MarketOpsEvidenceRecord
	var featureIDsJSON, transitionIDsJSON string
	if err := scanner.Scan(&record.EvidenceID, &record.TenantID, &record.AppID, &record.AssetID,
		&record.Symbol, &record.SessionDate, &record.AsOfTime, &record.EvidenceType, &record.EvidenceVersion,
		&record.Domain, &record.Direction, &record.Magnitude, &record.RarityScore, &record.PersistenceScore,
		&record.QualityScore, &record.Statement, &record.EvidencePayloadJSON, &featureIDsJSON,
		&transitionIDsJSON, &record.DeterministicKey, &record.CreatedAt); err != nil {
		return storage.MarketOpsEvidenceRecord{}, mapScanError("scan marketops evidence", err)
	}
	if err := json.Unmarshal([]byte(featureIDsJSON), &record.SourceFeatureIDs); err != nil {
		return storage.MarketOpsEvidenceRecord{}, fmt.Errorf("scan marketops evidence feature ids: %w", err)
	}
	if err := json.Unmarshal([]byte(transitionIDsJSON), &record.SourceTransitionIDs); err != nil {
		return storage.MarketOpsEvidenceRecord{}, fmt.Errorf("scan marketops evidence transition ids: %w", err)
	}
	return record, nil
}

type marketOpsMarketStateScanner interface {
	Scan(dest ...any) error
}

func validateMarketOpsFeatureDefinition(record storage.MarketOpsFeatureDefinitionRecord) error {
	if err := requireMarketOpsFields("feature definition", map[string]string{
		"tenant_id": record.TenantID, "feature_key": record.FeatureKey, "feature_version": record.FeatureVersion,
		"domain": record.Domain, "title": record.Title, "value_type": record.ValueType, "status": record.Status,
	}); err != nil {
		return err
	}
	if !oneOf(record.ValueType, "numeric", "text", "boolean") {
		return fmt.Errorf("marketops feature definition value_type is invalid")
	}
	if !oneOf(record.Status, storage.MarketOpsFeatureDefinitionStatusDraft, storage.MarketOpsFeatureDefinitionStatusActive,
		storage.MarketOpsFeatureDefinitionStatusDisabled, storage.MarketOpsFeatureDefinitionStatusDeprecated) {
		return fmt.Errorf("marketops feature definition status is invalid")
	}
	if err := validateJSONObject("marketops feature calculation spec", jsonOrEmpty(record.CalculationSpec)); err != nil {
		return err
	}
	if err := validateJSONArray("marketops feature required inputs", jsonArrayOrEmpty(record.RequiredInputs)); err != nil {
		return err
	}
	return validateJSONObject("marketops feature quality policy", jsonOrEmpty(record.QualityPolicy))
}

func validateMarketOpsFeatureObservation(record storage.MarketOpsFeatureObservationRecord) error {
	if err := requireMarketOpsFields("feature observation", map[string]string{
		"feature_observation_id": record.FeatureObservationID, "tenant_id": record.TenantID,
		"asset_id": record.AssetID, "symbol": record.Symbol, "feature_key": record.FeatureKey,
		"feature_version": record.FeatureVersion, "quality_state": record.QualityState,
		"calculation_run_id": record.CalculationRunID, "deterministic_key": record.DeterministicKey,
	}); err != nil {
		return err
	}
	if record.SessionDate.IsZero() || record.AsOfTime.IsZero() {
		return fmt.Errorf("marketops feature observation session_date and as_of_time are required")
	}
	values := 0
	if record.NumericValue != nil {
		values++
	}
	if record.TextValue != nil {
		values++
	}
	if record.BooleanValue != nil {
		values++
	}
	if values > 1 {
		return fmt.Errorf("marketops feature observation may set at most one typed value")
	}
	if values == 0 && oneOf(record.QualityState, storage.MarketOpsQualityUsable, storage.MarketOpsQualityUsableWithWarning) {
		return fmt.Errorf("marketops feature observation usable quality requires a typed value")
	}
	if !validMarketOpsQualityState(record.QualityState) {
		return fmt.Errorf("marketops feature observation quality_state is invalid")
	}
	if err := validateOptionalScore("marketops feature observation quality_score", record.QualityScore); err != nil {
		return err
	}
	if err := validateJSONObject("marketops feature observation dimensions", jsonOrEmpty(record.DimensionsJSON)); err != nil {
		return err
	}
	return validateJSONObject("marketops feature observation quality details", jsonOrEmpty(record.QualityDetailsJSON))
}

func validateMarketOpsMarketState(record storage.MarketOpsMarketStateRecord) error {
	if err := requireMarketOpsFields("market state", map[string]string{
		"market_state_id": record.MarketStateID, "tenant_id": record.TenantID, "asset_id": record.AssetID,
		"symbol": record.Symbol, "state_schema_version": record.StateSchemaVersion,
		"quality_state": record.QualityState, "build_run_id": record.BuildRunID,
		"deterministic_key": record.DeterministicKey,
	}); err != nil {
		return err
	}
	if record.SessionDate.IsZero() || record.AsOfTime.IsZero() {
		return fmt.Errorf("marketops market state session_date and as_of_time are required")
	}
	if record.FeatureCount < 0 || record.RequiredFeatureCount < 0 {
		return fmt.Errorf("marketops market state feature counts cannot be negative")
	}
	if record.CompletenessRatio < 0 || record.CompletenessRatio > 1 {
		return fmt.Errorf("marketops market state completeness_ratio must be between 0 and 1")
	}
	if !validMarketOpsQualityState(record.QualityState) {
		return fmt.Errorf("marketops market state quality_state is invalid")
	}
	if err := validateOptionalScore("marketops market state quality_score", record.QualityScore); err != nil {
		return err
	}
	if err := validateJSONObject("marketops market state payload", jsonOrEmpty(record.StatePayloadJSON)); err != nil {
		return err
	}
	return validateJSONObject("marketops market state quality summary", jsonOrEmpty(record.QualitySummaryJSON))
}

func validateMarketOpsStateTransition(record storage.MarketOpsStateTransitionRecord) error {
	if err := requireMarketOpsFields("state transition", map[string]string{
		"transition_id": record.TransitionID, "tenant_id": record.TenantID, "asset_id": record.AssetID,
		"symbol": record.Symbol, "current_state_id": record.CurrentStateID, "feature_key": record.FeatureKey,
		"feature_version": record.FeatureVersion, "transition_type": record.TransitionType,
		"quality_state": record.QualityState, "calculation_run_id": record.CalculationRunID,
		"deterministic_key": record.DeterministicKey,
	}); err != nil {
		return err
	}
	if record.SessionDate.IsZero() || record.AsOfTime.IsZero() {
		return fmt.Errorf("marketops state transition session_date and as_of_time are required")
	}
	if record.LookbackSessions != nil && *record.LookbackSessions < 0 {
		return fmt.Errorf("marketops state transition lookback_sessions cannot be negative")
	}
	if record.PersistenceSessions != nil && *record.PersistenceSessions < 0 {
		return fmt.Errorf("marketops state transition persistence_sessions cannot be negative")
	}
	if err := validateOptionalScore("marketops state transition percentile", record.Percentile); err != nil {
		return err
	}
	if !validMarketOpsQualityState(record.QualityState) {
		return fmt.Errorf("marketops state transition quality_state is invalid")
	}
	if err := validateJSONObject("marketops state transition dimensions", jsonOrEmpty(record.DimensionsJSON)); err != nil {
		return err
	}
	return validateJSONObject("marketops state transition payload", jsonOrEmpty(record.TransitionPayloadJSON))
}

func validateMarketOpsEvidence(record storage.MarketOpsEvidenceRecord) error {
	if err := requireMarketOpsFields("evidence", map[string]string{
		"evidence_id": record.EvidenceID, "tenant_id": record.TenantID, "asset_id": record.AssetID,
		"symbol": record.Symbol, "evidence_type": record.EvidenceType,
		"evidence_version": record.EvidenceVersion, "domain": record.Domain,
		"statement": record.Statement, "deterministic_key": record.DeterministicKey,
	}); err != nil {
		return err
	}
	if record.SessionDate.IsZero() || record.AsOfTime.IsZero() {
		return fmt.Errorf("marketops evidence session_date and as_of_time are required")
	}
	for name, score := range map[string]*float64{
		"rarity_score": record.RarityScore, "persistence_score": record.PersistenceScore,
		"quality_score": record.QualityScore,
	} {
		if err := validateOptionalScore("marketops evidence "+name, score); err != nil {
			return err
		}
	}
	return validateJSONObject("marketops evidence payload", jsonOrEmpty(record.EvidencePayloadJSON))
}

func validMarketOpsQualityState(value string) bool {
	return oneOf(value, storage.MarketOpsQualityUsable, storage.MarketOpsQualityUsableWithWarning,
		storage.MarketOpsQualityPartial, storage.MarketOpsQualitySparse, storage.MarketOpsQualityStale,
		storage.MarketOpsQualityInvalid, storage.MarketOpsQualityMissing, storage.MarketOpsQualityNotApplicable)
}

func validateOptionalScore(name string, value *float64) error {
	if value != nil && (*value < 0 || *value > 1) {
		return fmt.Errorf("%s must be between 0 and 1", name)
	}
	return nil
}

func requireMarketOpsFields(recordName string, fields map[string]string) error {
	for name, value := range fields {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("marketops %s %s is required", recordName, name)
		}
	}
	return nil
}

func oneOf(value string, allowed ...string) bool {
	value = strings.TrimSpace(value)
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
