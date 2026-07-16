package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) UpsertAlgorithmDefinition(ctx context.Context, record storage.AlgorithmDefinitionRecord) error {
	if err := validateAlgorithmDefinition(record); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO algorithm_definitions (
  tenant_id, algorithm_id, name, description, algorithm_type, runtime_type,
  input_features, input_event_types, output_schema, config_schema, default_config,
  version, status, metadata, updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6,
  $7, $8, $9, $10, $11,
  $12, $13, $14, now()
)
ON CONFLICT (tenant_id, algorithm_id) DO UPDATE SET
  name=EXCLUDED.name,
  description=EXCLUDED.description,
  algorithm_type=EXCLUDED.algorithm_type,
  runtime_type=EXCLUDED.runtime_type,
  input_features=EXCLUDED.input_features,
  input_event_types=EXCLUDED.input_event_types,
  output_schema=EXCLUDED.output_schema,
  config_schema=EXCLUDED.config_schema,
  default_config=EXCLUDED.default_config,
  version=EXCLUDED.version,
  status=EXCLUDED.status,
  metadata=EXCLUDED.metadata,
  updated_at=now()`, strings.TrimSpace(record.TenantID), strings.TrimSpace(record.AlgorithmID), strings.TrimSpace(record.Name), strings.TrimSpace(record.Description), strings.TrimSpace(record.AlgorithmType), strings.TrimSpace(record.RuntimeType), pqArray(record.InputFeatures), pqArray(record.InputEventTypes), jsonOrEmpty(record.OutputSchema), jsonOrEmpty(record.ConfigSchema), jsonOrEmpty(record.DefaultConfig), strings.TrimSpace(record.Version), strings.TrimSpace(record.Status), jsonOrEmpty(record.MetadataJSON))
	if err != nil {
		return fmt.Errorf("upsert algorithm definition: %w", err)
	}
	return nil
}

func (r *Repository) ListAlgorithmDefinitions(ctx context.Context, filter storage.AlgorithmDefinitionFilter) ([]storage.AlgorithmDefinitionRecord, error) {
	rows, err := r.db.QueryContext(ctx, algorithmDefinitionSelect+`
WHERE tenant_id=$1 AND ($2='' OR algorithm_type=$2) AND ($3='' OR runtime_type=$3) AND ($4='' OR status=$4)
ORDER BY algorithm_id ASC LIMIT $5`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AlgorithmType), strings.TrimSpace(filter.RuntimeType), strings.TrimSpace(filter.Status), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list algorithm definitions: %w", err)
	}
	defer rows.Close()
	records := []storage.AlgorithmDefinitionRecord{}
	for rows.Next() {
		record, err := scanAlgorithmDefinition(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list algorithm definitions rows: %w", err)
	}
	return records, nil
}

func (r *Repository) GetAlgorithmDefinition(ctx context.Context, tenantID string, algorithmID string) (storage.AlgorithmDefinitionRecord, error) {
	record, err := scanAlgorithmDefinition(r.db.QueryRowContext(ctx, algorithmDefinitionSelect+` WHERE tenant_id=$1 AND algorithm_id=$2`, strings.TrimSpace(tenantID), strings.TrimSpace(algorithmID)))
	if err != nil {
		return storage.AlgorithmDefinitionRecord{}, err
	}
	return record, nil
}

func (r *Repository) UpsertAlgorithmExecutionRequest(ctx context.Context, record storage.AlgorithmExecutionRequestRecord) error {
	if err := validateAlgorithmExecutionRequest(record); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO algorithm_execution_requests (
  tenant_id, execution_request_id, algorithm_id, algorithm_version, event_ids, feature_refs,
  entity_refs, window_ref, config, correlation_id, status, requested_by, result, error_message, updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6,
  $7, $8, $9, $10, $11, $12, $13, $14, now()
)
ON CONFLICT (tenant_id, execution_request_id) DO UPDATE SET
  algorithm_id=EXCLUDED.algorithm_id,
  algorithm_version=EXCLUDED.algorithm_version,
  event_ids=EXCLUDED.event_ids,
  feature_refs=EXCLUDED.feature_refs,
  entity_refs=EXCLUDED.entity_refs,
  window_ref=EXCLUDED.window_ref,
  config=EXCLUDED.config,
  correlation_id=EXCLUDED.correlation_id,
  status=EXCLUDED.status,
  requested_by=EXCLUDED.requested_by,
  result=EXCLUDED.result,
  error_message=EXCLUDED.error_message,
  updated_at=now()`, strings.TrimSpace(record.TenantID), strings.TrimSpace(record.ExecutionRequestID), strings.TrimSpace(record.AlgorithmID), strings.TrimSpace(record.AlgorithmVersion), pqArray(record.EventIDs), pqArray(record.FeatureRefs), pqArray(record.EntityRefs), strings.TrimSpace(record.WindowRef), jsonOrEmpty(record.ConfigJSON), strings.TrimSpace(record.CorrelationID), strings.TrimSpace(record.Status), firstNonEmptyString(record.RequestedBy, "operator-local"), jsonOrEmpty(record.ResultJSON), strings.TrimSpace(record.ErrorMessage))
	if err != nil {
		return fmt.Errorf("upsert algorithm execution request: %w", err)
	}
	return nil
}

func (r *Repository) ListAlgorithmExecutionRequests(ctx context.Context, filter storage.AlgorithmExecutionRequestFilter) ([]storage.AlgorithmExecutionRequestRecord, error) {
	rows, err := r.db.QueryContext(ctx, algorithmExecutionRequestSelect+`
WHERE tenant_id=$1 AND ($2='' OR algorithm_id=$2) AND ($3='' OR status=$3) AND ($4='' OR correlation_id=$4)
ORDER BY updated_at DESC LIMIT $5`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AlgorithmID), strings.TrimSpace(filter.Status), strings.TrimSpace(filter.CorrelationID), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list algorithm execution requests: %w", err)
	}
	defer rows.Close()
	records := []storage.AlgorithmExecutionRequestRecord{}
	for rows.Next() {
		record, err := scanAlgorithmExecutionRequest(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list algorithm execution requests rows: %w", err)
	}
	return records, nil
}

func (r *Repository) GetAlgorithmExecutionRequest(ctx context.Context, tenantID string, executionRequestID string) (storage.AlgorithmExecutionRequestRecord, error) {
	return scanAlgorithmExecutionRequest(r.db.QueryRowContext(ctx, algorithmExecutionRequestSelect+` WHERE tenant_id=$1 AND execution_request_id=$2`, strings.TrimSpace(tenantID), strings.TrimSpace(executionRequestID)))
}

func (r *Repository) InsertAlgorithmResult(ctx context.Context, record storage.AlgorithmResultRecord) error {
	if err := validateAlgorithmResult(record); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO algorithm_results (
  tenant_id, algorithm_result_id, algorithm_id, algorithm_version, execution_request_id,
  result_type, score, confidence, severity, result_payload, source_event_ids,
  feature_value_ids, evidence_refs, correlation_id
) VALUES (
  $1, $2, $3, $4, $5,
  $6, $7, $8, $9, $10, $11,
  $12, $13, $14
)
ON CONFLICT (tenant_id, algorithm_result_id) DO NOTHING`, strings.TrimSpace(record.TenantID), strings.TrimSpace(record.AlgorithmResultID), strings.TrimSpace(record.AlgorithmID), strings.TrimSpace(record.AlgorithmVersion), strings.TrimSpace(record.ExecutionRequestID), strings.TrimSpace(record.ResultType), record.Score, record.Confidence, strings.TrimSpace(record.Severity), jsonOrEmpty(record.ResultPayloadJSON), pqArray(record.SourceEventIDs), pqArray(record.FeatureValueIDs), pqArray(record.EvidenceRefs), strings.TrimSpace(record.CorrelationID))
	if err != nil {
		return fmt.Errorf("insert algorithm result: %w", err)
	}
	return nil
}

func (r *Repository) ListAlgorithmResults(ctx context.Context, filter storage.AlgorithmResultFilter) ([]storage.AlgorithmResultRecord, error) {
	rows, err := r.db.QueryContext(ctx, algorithmResultSelect+`
WHERE tenant_id=$1 AND ($2='' OR algorithm_id=$2) AND ($3='' OR execution_request_id=$3)
  AND ($4='' OR result_type=$4) AND ($5='' OR severity=$5) AND ($6='' OR correlation_id=$6)
ORDER BY created_at DESC LIMIT $7`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AlgorithmID), strings.TrimSpace(filter.ExecutionRequestID), strings.TrimSpace(filter.ResultType), strings.TrimSpace(filter.Severity), strings.TrimSpace(filter.CorrelationID), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list algorithm results: %w", err)
	}
	defer rows.Close()
	records := []storage.AlgorithmResultRecord{}
	for rows.Next() {
		record, err := scanAlgorithmResult(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list algorithm results rows: %w", err)
	}
	return records, nil
}

func (r *Repository) GetAlgorithmResult(ctx context.Context, tenantID string, algorithmResultID string) (storage.AlgorithmResultRecord, error) {
	return scanAlgorithmResult(r.db.QueryRowContext(ctx, algorithmResultSelect+` WHERE tenant_id=$1 AND algorithm_result_id=$2`, strings.TrimSpace(tenantID), strings.TrimSpace(algorithmResultID)))
}

func (r *Repository) InsertAlgorithmSignalProposal(ctx context.Context, record storage.AlgorithmSignalProposalRecord) (bool, error) {
	if err := validateAlgorithmSignalProposal(record); err != nil {
		return false, err
	}
	result, err := r.db.ExecContext(ctx, `
INSERT INTO algorithm_signal_proposals (
  tenant_id, proposal_id, algorithm_result_id, algorithm_id, algorithm_version, execution_request_id,
  proposed_signal_type, status, score, confidence, severity, proposal_payload, rationale,
  source_event_ids, evidence_refs, correlation_id, created_by, updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6,
  $7, $8, $9, $10, $11, $12, $13,
  $14, $15, $16, $17, now()
)
ON CONFLICT (tenant_id, algorithm_result_id, proposed_signal_type) DO NOTHING`, strings.TrimSpace(record.TenantID), strings.TrimSpace(record.ProposalID), strings.TrimSpace(record.AlgorithmResultID), strings.TrimSpace(record.AlgorithmID), strings.TrimSpace(record.AlgorithmVersion), strings.TrimSpace(record.ExecutionRequestID), strings.TrimSpace(record.ProposedSignalType), strings.TrimSpace(record.Status), record.Score, record.Confidence, strings.TrimSpace(record.Severity), jsonOrEmpty(record.ProposalPayloadJSON), jsonOrEmpty(record.RationaleJSON), pqArray(record.SourceEventIDs), pqArray(record.EvidenceRefs), strings.TrimSpace(record.CorrelationID), firstNonEmptyString(record.CreatedBy, "operator-local"))
	if err != nil {
		return false, fmt.Errorf("insert algorithm signal proposal: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("insert algorithm signal proposal rows affected: %w", err)
	}
	return rows > 0, nil
}

func (r *Repository) ListAlgorithmSignalProposals(ctx context.Context, filter storage.AlgorithmSignalProposalFilter) ([]storage.AlgorithmSignalProposalRecord, error) {
	rows, err := r.db.QueryContext(ctx, algorithmSignalProposalSelect+`
WHERE tenant_id=$1 AND ($2='' OR algorithm_id=$2) AND ($3='' OR execution_request_id=$3)
  AND ($4='' OR algorithm_result_id=$4) AND ($5='' OR status=$5) AND ($6='' OR severity=$6)
  AND ($7='' OR correlation_id=$7)
ORDER BY created_at DESC LIMIT $8`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AlgorithmID), strings.TrimSpace(filter.ExecutionRequestID), strings.TrimSpace(filter.AlgorithmResultID), strings.TrimSpace(filter.Status), strings.TrimSpace(filter.Severity), strings.TrimSpace(filter.CorrelationID), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list algorithm signal proposals: %w", err)
	}
	defer rows.Close()
	records := []storage.AlgorithmSignalProposalRecord{}
	for rows.Next() {
		record, err := scanAlgorithmSignalProposal(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list algorithm signal proposals rows: %w", err)
	}
	return records, nil
}

func (r *Repository) GetAlgorithmSignalProposal(ctx context.Context, tenantID string, proposalID string) (storage.AlgorithmSignalProposalRecord, error) {
	return scanAlgorithmSignalProposal(r.db.QueryRowContext(ctx, algorithmSignalProposalSelect+` WHERE tenant_id=$1 AND proposal_id=$2`, strings.TrimSpace(tenantID), strings.TrimSpace(proposalID)))
}

func (r *Repository) SummarizeAlgorithmSignalProposals(ctx context.Context, filter storage.AlgorithmSignalProposalFilter) (storage.AlgorithmSignalProposalSummaryRecord, error) {
	tenantID := strings.TrimSpace(filter.TenantID)
	if tenantID == "" {
		return storage.AlgorithmSignalProposalSummaryRecord{}, fmt.Errorf("algorithm signal proposal summary tenant id is required")
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT status, severity, proposed_signal_type, algorithm_id, reviewed_by, COUNT(*)
FROM algorithm_signal_proposals
WHERE tenant_id=$1 AND ($2='' OR algorithm_id=$2) AND ($3='' OR execution_request_id=$3)
  AND ($4='' OR algorithm_result_id=$4) AND ($5='' OR status=$5) AND ($6='' OR severity=$6)
  AND ($7='' OR correlation_id=$7)
GROUP BY status, severity, proposed_signal_type, algorithm_id, reviewed_by`, tenantID, strings.TrimSpace(filter.AlgorithmID), strings.TrimSpace(filter.ExecutionRequestID), strings.TrimSpace(filter.AlgorithmResultID), strings.TrimSpace(filter.Status), strings.TrimSpace(filter.Severity), strings.TrimSpace(filter.CorrelationID))
	if err != nil {
		return storage.AlgorithmSignalProposalSummaryRecord{}, fmt.Errorf("summarize algorithm signal proposals: %w", err)
	}
	defer rows.Close()
	summary := storage.AlgorithmSignalProposalSummaryRecord{TenantID: tenantID, StatusCounts: map[string]int{}, SeverityCounts: map[string]int{}, ProposedSignalTypeCounts: map[string]int{}, AlgorithmIDCounts: map[string]int{}, ReviewerCounts: map[string]int{}}
	for rows.Next() {
		var status, severity, signalType, algorithmID, reviewer string
		var count int
		if err := rows.Scan(&status, &severity, &signalType, &algorithmID, &reviewer, &count); err != nil {
			return storage.AlgorithmSignalProposalSummaryRecord{}, mapScanError("scan algorithm signal proposal summary", err)
		}
		summary.TotalProposals += count
		summary.StatusCounts[status] += count
		summary.SeverityCounts[severity] += count
		summary.ProposedSignalTypeCounts[signalType] += count
		summary.AlgorithmIDCounts[algorithmID] += count
		if strings.TrimSpace(reviewer) != "" {
			summary.ReviewerCounts[reviewer] += count
		}
		switch status {
		case storage.AlgorithmSignalProposalStatusProposed:
			summary.ProposedCount += count
		case storage.AlgorithmSignalProposalStatusReviewed:
			summary.ReviewedCount += count
		case storage.AlgorithmSignalProposalStatusRejected:
			summary.RejectedCount += count
		case storage.AlgorithmSignalProposalStatusSuperseded:
			summary.SupersededCount += count
		}
		if status == storage.AlgorithmSignalProposalStatusProposed && (severity == "high" || severity == "critical") {
			summary.HighCriticalUnreviewedCount += count
		}
	}
	if err := rows.Err(); err != nil {
		return storage.AlgorithmSignalProposalSummaryRecord{}, fmt.Errorf("summarize algorithm signal proposals rows: %w", err)
	}
	if summary.TotalProposals > 0 {
		summary.ReviewedRatio = float64(summary.ReviewedCount+summary.RejectedCount+summary.SupersededCount) / float64(summary.TotalProposals)
	}
	return summary, nil
}

func (r *Repository) MutateAlgorithmSignalProposal(ctx context.Context, mutation storage.AlgorithmSignalProposalMutation) (storage.AlgorithmSignalProposalRecord, error) {
	if strings.TrimSpace(mutation.TenantID) == "" {
		return storage.AlgorithmSignalProposalRecord{}, fmt.Errorf("algorithm signal proposal tenant id is required")
	}
	if strings.TrimSpace(mutation.ProposalID) == "" {
		return storage.AlgorithmSignalProposalRecord{}, fmt.Errorf("algorithm signal proposal id is required")
	}
	if !validAlgorithmSignalProposalStatus(mutation.Status) {
		return storage.AlgorithmSignalProposalRecord{}, fmt.Errorf("algorithm signal proposal status is invalid")
	}
	decidedAt := mutation.DecidedAt.UTC()
	if decidedAt.IsZero() {
		decidedAt = nowUTC()
	}
	return scanAlgorithmSignalProposal(r.db.QueryRowContext(ctx, `
UPDATE algorithm_signal_proposals
SET status=$3, reviewed_by=$4, decision_note=$5, decided_at=$6, decision_metadata=$7, updated_at=now()
WHERE tenant_id=$1 AND proposal_id=$2 `+algorithmSignalProposalReturning, strings.TrimSpace(mutation.TenantID), strings.TrimSpace(mutation.ProposalID), strings.TrimSpace(mutation.Status), firstNonEmptyString(mutation.ReviewedBy, "operator-local"), strings.TrimSpace(mutation.DecisionNote), decidedAt, jsonOrEmpty(mutation.MetadataJSON)))
}

func (r *Repository) ListAlgorithmSignalMaterializations(ctx context.Context, filter storage.AlgorithmSignalMaterializationFilter) ([]storage.AlgorithmSignalMaterializationRecord, error) {
	tenantID := strings.TrimSpace(filter.TenantID)
	if tenantID == "" {
		return nil, fmt.Errorf("algorithm signal materialization tenant id is required")
	}
	rows, err := r.db.QueryContext(ctx, algorithmSignalMaterializationSelect+`
WHERE tenant_id=$1 AND ($2='' OR proposal_id=$2) AND ($3='' OR algorithm_result_id=$3)
  AND ($4='' OR execution_request_id=$4) AND ($5='' OR algorithm_id=$5)
  AND ($6='' OR materialization_status=$6) AND ($7='' OR signal_id=$7)
ORDER BY updated_at DESC LIMIT $8`, tenantID, strings.TrimSpace(filter.ProposalID), strings.TrimSpace(filter.AlgorithmResultID), strings.TrimSpace(filter.ExecutionRequestID), strings.TrimSpace(filter.AlgorithmID), strings.TrimSpace(filter.MaterializationStatus), strings.TrimSpace(filter.SignalID), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list algorithm signal materializations: %w", err)
	}
	defer rows.Close()
	records := []storage.AlgorithmSignalMaterializationRecord{}
	for rows.Next() {
		record, err := scanAlgorithmSignalMaterialization(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list algorithm signal materializations rows: %w", err)
	}
	return records, nil
}

func (r *Repository) GetAlgorithmSignalMaterialization(ctx context.Context, tenantID string, materializationID string) (storage.AlgorithmSignalMaterializationRecord, error) {
	return scanAlgorithmSignalMaterialization(r.db.QueryRowContext(ctx, algorithmSignalMaterializationSelect+` WHERE tenant_id=$1 AND materialization_id=$2`, strings.TrimSpace(tenantID), strings.TrimSpace(materializationID)))
}

const algorithmDefinitionSelect = `SELECT algorithm_id, tenant_id, name, description, algorithm_type, runtime_type,
 COALESCE(array_to_json(input_features), '[]'::json)::text, COALESCE(array_to_json(input_event_types), '[]'::json)::text,
 output_schema, config_schema, default_config, version, status, metadata, created_at, updated_at FROM algorithm_definitions`

const algorithmExecutionRequestSelect = `SELECT execution_request_id, tenant_id, algorithm_id, algorithm_version,
 COALESCE(array_to_json(event_ids), '[]'::json)::text, COALESCE(array_to_json(feature_refs), '[]'::json)::text,
 COALESCE(array_to_json(entity_refs), '[]'::json)::text, window_ref, config, correlation_id, status, requested_by,
 result, error_message, created_at, updated_at FROM algorithm_execution_requests`

const algorithmResultSelect = `SELECT algorithm_result_id, tenant_id, algorithm_id, algorithm_version, execution_request_id,
 result_type, score, confidence, severity, result_payload, COALESCE(array_to_json(source_event_ids), '[]'::json)::text,
 COALESCE(array_to_json(feature_value_ids), '[]'::json)::text, COALESCE(array_to_json(evidence_refs), '[]'::json)::text,
 correlation_id, created_at FROM algorithm_results`

const algorithmSignalProposalSelect = `SELECT proposal_id, tenant_id, algorithm_result_id, algorithm_id, algorithm_version, execution_request_id,
 proposed_signal_type, status, score, confidence, severity, proposal_payload, rationale,
 COALESCE(array_to_json(source_event_ids), '[]'::json)::text, COALESCE(array_to_json(evidence_refs), '[]'::json)::text,
 correlation_id, created_by, reviewed_by, decision_note, decided_at, created_at, updated_at FROM algorithm_signal_proposals`

const algorithmSignalProposalReturning = `RETURNING proposal_id, tenant_id, algorithm_result_id, algorithm_id, algorithm_version, execution_request_id,
 proposed_signal_type, status, score, confidence, severity, proposal_payload, rationale,
 COALESCE(array_to_json(source_event_ids), '[]'::json)::text, COALESCE(array_to_json(evidence_refs), '[]'::json)::text,
 correlation_id, created_by, reviewed_by, decision_note, decided_at, created_at, updated_at`

const algorithmSignalMaterializationSelect = `SELECT materialization_id, tenant_id, proposal_id, algorithm_result_id,
 execution_request_id, algorithm_id, algorithm_version, proposed_signal_type, COALESCE(signal_id, ''),
 materialization_status, materialization_policy_version, idempotency_key, COALESCE(duplicate_of_signal_id, ''),
 requested_by, requested_at, started_at, completed_at, failed_at, COALESCE(error_code, ''), COALESCE(error_message, ''),
 request_metadata, preflight_snapshot, signal_payload_preview, created_at, updated_at FROM algorithm_signal_materializations`

type algorithmDefinitionScanner interface{ Scan(dest ...any) error }
type algorithmExecutionRequestScanner interface{ Scan(dest ...any) error }
type algorithmResultScanner interface{ Scan(dest ...any) error }
type algorithmSignalProposalScanner interface{ Scan(dest ...any) error }
type algorithmSignalMaterializationScanner interface{ Scan(dest ...any) error }

func scanAlgorithmDefinition(scanner algorithmDefinitionScanner) (storage.AlgorithmDefinitionRecord, error) {
	var record storage.AlgorithmDefinitionRecord
	var inputFeaturesJSON, inputEventTypesJSON string
	if err := scanner.Scan(&record.AlgorithmID, &record.TenantID, &record.Name, &record.Description, &record.AlgorithmType, &record.RuntimeType, &inputFeaturesJSON, &inputEventTypesJSON, &record.OutputSchema, &record.ConfigSchema, &record.DefaultConfig, &record.Version, &record.Status, &record.MetadataJSON, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return storage.AlgorithmDefinitionRecord{}, mapScanError("scan algorithm definition", err)
	}
	if err := json.Unmarshal([]byte(inputFeaturesJSON), &record.InputFeatures); err != nil {
		return storage.AlgorithmDefinitionRecord{}, fmt.Errorf("scan algorithm definition input features: %w", err)
	}
	if err := json.Unmarshal([]byte(inputEventTypesJSON), &record.InputEventTypes); err != nil {
		return storage.AlgorithmDefinitionRecord{}, fmt.Errorf("scan algorithm definition input event types: %w", err)
	}
	return record, nil
}

func scanAlgorithmExecutionRequest(scanner algorithmExecutionRequestScanner) (storage.AlgorithmExecutionRequestRecord, error) {
	var record storage.AlgorithmExecutionRequestRecord
	var eventIDsJSON, featureRefsJSON, entityRefsJSON string
	if err := scanner.Scan(&record.ExecutionRequestID, &record.TenantID, &record.AlgorithmID, &record.AlgorithmVersion, &eventIDsJSON, &featureRefsJSON, &entityRefsJSON, &record.WindowRef, &record.ConfigJSON, &record.CorrelationID, &record.Status, &record.RequestedBy, &record.ResultJSON, &record.ErrorMessage, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return storage.AlgorithmExecutionRequestRecord{}, mapScanError("scan algorithm execution request", err)
	}
	for _, item := range []struct {
		raw  string
		dest *[]string
		name string
	}{{eventIDsJSON, &record.EventIDs, "event ids"}, {featureRefsJSON, &record.FeatureRefs, "feature refs"}, {entityRefsJSON, &record.EntityRefs, "entity refs"}} {
		if err := json.Unmarshal([]byte(item.raw), item.dest); err != nil {
			return storage.AlgorithmExecutionRequestRecord{}, fmt.Errorf("scan algorithm execution request %s: %w", item.name, err)
		}
	}
	return record, nil
}

func scanAlgorithmResult(scanner algorithmResultScanner) (storage.AlgorithmResultRecord, error) {
	var record storage.AlgorithmResultRecord
	var sourceEventIDsJSON, featureValueIDsJSON, evidenceRefsJSON string
	if err := scanner.Scan(&record.AlgorithmResultID, &record.TenantID, &record.AlgorithmID, &record.AlgorithmVersion, &record.ExecutionRequestID, &record.ResultType, &record.Score, &record.Confidence, &record.Severity, &record.ResultPayloadJSON, &sourceEventIDsJSON, &featureValueIDsJSON, &evidenceRefsJSON, &record.CorrelationID, &record.CreatedAt); err != nil {
		return storage.AlgorithmResultRecord{}, mapScanError("scan algorithm result", err)
	}
	for _, item := range []struct {
		raw  string
		dest *[]string
		name string
	}{{sourceEventIDsJSON, &record.SourceEventIDs, "source event ids"}, {featureValueIDsJSON, &record.FeatureValueIDs, "feature value ids"}, {evidenceRefsJSON, &record.EvidenceRefs, "evidence refs"}} {
		if err := json.Unmarshal([]byte(item.raw), item.dest); err != nil {
			return storage.AlgorithmResultRecord{}, fmt.Errorf("scan algorithm result %s: %w", item.name, err)
		}
	}
	return record, nil
}

func scanAlgorithmSignalProposal(scanner algorithmSignalProposalScanner) (storage.AlgorithmSignalProposalRecord, error) {
	var record storage.AlgorithmSignalProposalRecord
	var sourceEventIDsJSON, evidenceRefsJSON string
	if err := scanner.Scan(&record.ProposalID, &record.TenantID, &record.AlgorithmResultID, &record.AlgorithmID, &record.AlgorithmVersion, &record.ExecutionRequestID, &record.ProposedSignalType, &record.Status, &record.Score, &record.Confidence, &record.Severity, &record.ProposalPayloadJSON, &record.RationaleJSON, &sourceEventIDsJSON, &evidenceRefsJSON, &record.CorrelationID, &record.CreatedBy, &record.ReviewedBy, &record.DecisionNote, &record.DecidedAt, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return storage.AlgorithmSignalProposalRecord{}, mapScanError("scan algorithm signal proposal", err)
	}
	for _, item := range []struct {
		raw  string
		dest *[]string
		name string
	}{{sourceEventIDsJSON, &record.SourceEventIDs, "source event ids"}, {evidenceRefsJSON, &record.EvidenceRefs, "evidence refs"}} {
		if err := json.Unmarshal([]byte(item.raw), item.dest); err != nil {
			return storage.AlgorithmSignalProposalRecord{}, fmt.Errorf("scan algorithm signal proposal %s: %w", item.name, err)
		}
	}
	return record, nil
}

func scanAlgorithmSignalMaterialization(scanner algorithmSignalMaterializationScanner) (storage.AlgorithmSignalMaterializationRecord, error) {
	var record storage.AlgorithmSignalMaterializationRecord
	if err := scanner.Scan(&record.MaterializationID, &record.TenantID, &record.ProposalID, &record.AlgorithmResultID, &record.ExecutionRequestID, &record.AlgorithmID, &record.AlgorithmVersion, &record.ProposedSignalType, &record.SignalID, &record.MaterializationStatus, &record.MaterializationPolicyVersion, &record.IdempotencyKey, &record.DuplicateOfSignalID, &record.RequestedBy, &record.RequestedAt, &record.StartedAt, &record.CompletedAt, &record.FailedAt, &record.ErrorCode, &record.ErrorMessage, &record.RequestMetadataJSON, &record.PreflightSnapshotJSON, &record.SignalPayloadPreviewJSON, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return storage.AlgorithmSignalMaterializationRecord{}, mapScanError("scan algorithm signal materialization", err)
	}
	return record, nil
}

func validateAlgorithmDefinition(record storage.AlgorithmDefinitionRecord) error {
	for name, value := range map[string]string{"tenant id": record.TenantID, "algorithm id": record.AlgorithmID, "name": record.Name, "algorithm type": record.AlgorithmType, "runtime type": record.RuntimeType, "version": record.Version, "status": record.Status} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("algorithm definition %s is required", name)
		}
	}
	return validateAlgorithmJSON(record.OutputSchema, record.ConfigSchema, record.DefaultConfig, record.MetadataJSON)
}

func validateAlgorithmExecutionRequest(record storage.AlgorithmExecutionRequestRecord) error {
	for name, value := range map[string]string{"tenant id": record.TenantID, "execution request id": record.ExecutionRequestID, "algorithm id": record.AlgorithmID, "algorithm version": record.AlgorithmVersion, "correlation id": record.CorrelationID, "status": record.Status} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("algorithm execution request %s is required", name)
		}
	}
	return validateAlgorithmJSON(record.ConfigJSON, record.ResultJSON)
}

func validateAlgorithmResult(record storage.AlgorithmResultRecord) error {
	for name, value := range map[string]string{"tenant id": record.TenantID, "algorithm result id": record.AlgorithmResultID, "algorithm id": record.AlgorithmID, "algorithm version": record.AlgorithmVersion, "execution request id": record.ExecutionRequestID, "result type": record.ResultType, "severity": record.Severity, "correlation id": record.CorrelationID} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("algorithm result %s is required", name)
		}
	}
	if record.Score < 0 {
		return errors.New("algorithm result score must be non-negative")
	}
	if record.Confidence < 0 || record.Confidence > 1 {
		return errors.New("algorithm result confidence must be between 0 and 1")
	}
	return validateAlgorithmJSON(record.ResultPayloadJSON)
}

func validateAlgorithmSignalProposal(record storage.AlgorithmSignalProposalRecord) error {
	for name, value := range map[string]string{"tenant id": record.TenantID, "proposal id": record.ProposalID, "algorithm result id": record.AlgorithmResultID, "algorithm id": record.AlgorithmID, "algorithm version": record.AlgorithmVersion, "execution request id": record.ExecutionRequestID, "proposed signal type": record.ProposedSignalType, "status": record.Status, "severity": record.Severity, "correlation id": record.CorrelationID} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("algorithm signal proposal %s is required", name)
		}
	}
	if record.Score < 0 {
		return errors.New("algorithm signal proposal score must be non-negative")
	}
	if record.Confidence < 0 || record.Confidence > 1 {
		return errors.New("algorithm signal proposal confidence must be between 0 and 1")
	}
	if !validAlgorithmSignalProposalStatus(record.Status) {
		return errors.New("algorithm signal proposal status is invalid")
	}
	return validateAlgorithmJSON(record.ProposalPayloadJSON, record.RationaleJSON)
}

func validAlgorithmSignalProposalStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case storage.AlgorithmSignalProposalStatusProposed, storage.AlgorithmSignalProposalStatusReviewed, storage.AlgorithmSignalProposalStatusRejected, storage.AlgorithmSignalProposalStatusSuperseded:
		return true
	default:
		return false
	}
}

func nowUTC() time.Time {
	return time.Now().UTC()
}

func validateAlgorithmJSON(values ...[]byte) error {
	for _, value := range values {
		if len(value) == 0 {
			continue
		}
		if !json.Valid(value) {
			return errors.New("algorithm json fields must be valid json")
		}
	}
	return nil
}
