package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
	"github.com/lukebabs/signalops/internal/subscriber/eodcanaryexecution"
)

// PrepareSubscriberGlobalEODCanaryExecutionGate records the only execution
// control available in this release. Its database schema permanently holds the
// plan in disabled mode with the kill switch engaged; no provider client is
// constructed or called here.
func (r *Repository) PrepareSubscriberGlobalEODCanaryExecutionGate(ctx context.Context, gate storage.SubscriberGlobalEODCanaryExecutionGate) (storage.SubscriberGlobalEODCanaryExecutionGate, error) {
	gate.ExecutionPlanID = strings.TrimSpace(gate.ExecutionPlanID)
	gate.CanaryRunID = strings.TrimSpace(gate.CanaryRunID)
	gate.ExpectedWorkerIdentity = strings.TrimSpace(gate.ExpectedWorkerIdentity)
	gate.PlannedBy = strings.TrimSpace(gate.PlannedBy)
	gate.CorrelationID = strings.TrimSpace(gate.CorrelationID)
	if gate.ExecutionPlanID == "" {
		gate.ExecutionPlanID = newSubscriberID("subeodgate")
	}
	if gate.CanaryRunID == "" || gate.ExpectedWorkerIdentity != eodcanaryexecution.ExpectedWorkerIdentity || gate.PlannedBy == "" || gate.MaxProviderRequests <= 0 || gate.MaxProviderRequests > eodcanaryexecution.MaximumProviderRequests {
		return gate, errors.New("invalid global EOD canary execution gate")
	}
	if gate.PlannedAt.IsZero() {
		gate.PlannedAt = time.Now().UTC()
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return gate, fmt.Errorf("begin global EOD canary execution gate: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var sessionDate time.Time
	var selectedCount int
	if err := tx.QueryRowContext(ctx, `SELECT session_date, selected_count FROM subscriber_global_eod_canary_runs WHERE canary_run_id=$1 AND execution_state='prepared' AND provider_execution_enabled=false AND scheduled_execution_enabled=false AND parity_required=true`, gate.CanaryRunID).Scan(&sessionDate, &selectedCount); err != nil {
		return gate, fmt.Errorf("load disabled global EOD canary: %w", err)
	}
	if selectedCount != gate.MaxProviderRequests {
		return gate, errors.New("canary selected count must exactly equal the two-request execution budget")
	}
	rows, err := tx.QueryContext(ctx, `SELECT member.global_asset_id, asset.canonical_symbol, member.priority
FROM subscriber_global_eod_canary_members member
JOIN subscriber_global_assets asset ON asset.global_asset_id=member.global_asset_id
WHERE member.canary_run_id=$1
ORDER BY member.priority, member.global_asset_id`, gate.CanaryRunID)
	if err != nil {
		return gate, fmt.Errorf("list frozen global EOD canary members: %w", err)
	}
	defer rows.Close()
	candidates := []eodcanaryexecution.Member{}
	for rows.Next() {
		var member eodcanaryexecution.Member
		if err := rows.Scan(&member.GlobalAssetID, &member.Ticker, &member.Priority); err != nil {
			return gate, fmt.Errorf("scan frozen global EOD canary member: %w", err)
		}
		candidates = append(candidates, member)
	}
	if err := rows.Err(); err != nil {
		return gate, fmt.Errorf("iterate frozen global EOD canary members: %w", err)
	}
	members, err := eodcanaryexecution.Freeze(gate.ExpectedWorkerIdentity, candidates, gate.MaxProviderRequests)
	if err != nil {
		return gate, err
	}
	controlProvenance, err := json.Marshal(map[string]any{
		"schema_version":              storage.SubscriberGlobalEODCanaryExecutionGateVersion,
		"provider_execution_enabled":  false,
		"scheduled_execution_enabled": false,
		"kill_switch_engaged":         true,
		"parity_status":               "not_started",
		"canary_session_date":         sessionDate.Format("2006-01-02"),
	})
	if err != nil {
		return gate, fmt.Errorf("encode canary execution control provenance: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO subscriber_global_eod_canary_execution_plans
  (execution_plan_id,canary_run_id,execution_version,expected_worker_identity,session_date,max_provider_requests,provider_execution_enabled,scheduled_execution_enabled,kill_switch_engaged,execution_state,correlation_id,control_provenance,planned_by,planned_at)
VALUES ($1,$2,$3,$4,$5,$6,false,false,true,'disabled',$7,$8::jsonb,$9,$10)`, gate.ExecutionPlanID, gate.CanaryRunID, storage.SubscriberGlobalEODCanaryExecutionGateVersion, gate.ExpectedWorkerIdentity, sessionDate, gate.MaxProviderRequests, gate.CorrelationID, string(controlProvenance), gate.PlannedBy, gate.PlannedAt); err != nil {
		return gate, fmt.Errorf("insert disabled global EOD canary execution plan: %w", err)
	}
	gate.Members = make([]storage.SubscriberGlobalEODCanaryExecutionMember, 0, len(members))
	for _, member := range members {
		baselineProvenance, err := json.Marshal(map[string]any{
			"canary_run_id":     gate.CanaryRunID,
			"execution_plan_id": gate.ExecutionPlanID,
			"request_ordinal":   member.RequestOrdinal,
			"frozen_at":         gate.PlannedAt.UTC().Format(time.RFC3339Nano),
		})
		if err != nil {
			return gate, fmt.Errorf("encode frozen member provenance: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO subscriber_global_eod_canary_execution_members
  (execution_plan_id,global_asset_id,request_ordinal,expected_symbol,expected_algorithm_version,expected_validation_contract_ref,expected_baseline_provenance)
VALUES ($1,$2,$3,$4,'subscriber-global-eod-baseline-v1','docs/projects/subscriber_project/s4_canary_execution_gate.md',$5::jsonb)`, gate.ExecutionPlanID, member.GlobalAssetID, member.RequestOrdinal, member.Ticker, string(baselineProvenance)); err != nil {
			return gate, fmt.Errorf("insert frozen global EOD canary execution member: %w", err)
		}
		plannedEvidence, err := json.Marshal(map[string]any{
			"provider_request_allowed": false,
			"parity_status":            "not_started",
			"expected_symbol":          member.Ticker,
			"request_ordinal":          member.RequestOrdinal,
		})
		if err != nil {
			return gate, fmt.Errorf("encode planned canary evidence: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO subscriber_global_eod_canary_evidence_events
  (evidence_event_id,execution_plan_id,global_asset_id,evidence_kind,event_ordinal,payload,provenance,recorded_by,recorded_at)
VALUES ($1,$2,$3,'execution_planned',1,$4::jsonb,$5::jsonb,$6,$7)`, newSubscriberID("subeodevidence"), gate.ExecutionPlanID, member.GlobalAssetID, string(plannedEvidence), string(baselineProvenance), gate.PlannedBy, gate.PlannedAt); err != nil {
			return gate, fmt.Errorf("insert planned global EOD canary evidence: %w", err)
		}
		gate.Members = append(gate.Members, storage.SubscriberGlobalEODCanaryExecutionMember{GlobalAssetID: member.GlobalAssetID, Ticker: member.Ticker, RequestOrdinal: member.RequestOrdinal})
	}
	if err := tx.Commit(); err != nil {
		return gate, fmt.Errorf("commit disabled global EOD canary execution plan: %w", err)
	}
	return gate, nil
}
