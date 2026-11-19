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

// AssumeSubscriberOptionsCaptureRole permits only the fixed capture-worker
// group after authentication by its NOINHERIT login.
func (r *Repository) AssumeSubscriberOptionsCaptureRole(ctx context.Context) error {
	if _, err := r.db.ExecContext(ctx, "SET ROLE signalops_subscriber_options_capture"); err != nil {
		return fmt.Errorf("assume subscriber options capture role: %w", err)
	}
	return nil
}

// PrepareSubscriberOptionsCaptureCanaryGate freezes only priority-one selected
// shadow demand in a database-disabled, one-request plan. This method imports
// no market-data client and cannot make a provider request.
func (r *Repository) PrepareSubscriberOptionsCaptureCanaryGate(ctx context.Context, gate storage.SubscriberOptionsCaptureCanaryGate) (storage.SubscriberOptionsCaptureCanaryGate, error) {
	gate.CapturePlanID, gate.SnapshotRunID, gate.PlannedBy, gate.CorrelationID = strings.TrimSpace(gate.CapturePlanID), strings.TrimSpace(gate.SnapshotRunID), strings.TrimSpace(gate.PlannedBy), strings.TrimSpace(gate.CorrelationID)
	if gate.CapturePlanID == "" {
		gate.CapturePlanID = newSubscriberID("suboptcapturegate")
	}
	if gate.SnapshotRunID == "" || gate.PlannedBy != "subscriber-options-capture" || gate.CorrelationID == "" {
		return gate, errors.New("invalid Options capture canary gate")
	}
	if gate.PlannedAt.IsZero() {
		gate.PlannedAt = time.Now().UTC()
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return gate, fmt.Errorf("begin Options capture canary gate: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var sessionDate time.Time
	if err := tx.QueryRowContext(ctx, `SELECT session_date FROM subscriber_options_demand_snapshot_runs WHERE snapshot_run_id=$1 AND execution_mode='shadow' AND selected_count>0`, gate.SnapshotRunID).Scan(&sessionDate); err != nil {
		return gate, fmt.Errorf("load selected Options demand snapshot: %w", err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT member.global_asset_id,asset.canonical_symbol FROM subscriber_options_demand_snapshot_members member JOIN subscriber_global_assets asset ON asset.global_asset_id=member.global_asset_id WHERE member.snapshot_run_id=$1 AND member.selection_state='selected' AND member.priority=1`, gate.SnapshotRunID).Scan(&gate.GlobalAssetID, &gate.Ticker); err != nil {
		return gate, fmt.Errorf("load priority-one Options capture candidate: %w", err)
	}
	provenance, err := json.Marshal(map[string]any{"schema_version": storage.SubscriberOptionsCaptureCanaryGateVersion, "provider_execution_enabled": false, "scheduled_execution_enabled": false, "kill_switch_engaged": true, "provider_request_budget": 1, "source_snapshot_run_id": gate.SnapshotRunID})
	if err != nil {
		return gate, fmt.Errorf("encode Options capture gate provenance: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO subscriber_options_capture_canary_plans (capture_plan_id,snapshot_run_id,capture_version,session_date,max_provider_requests,provider_execution_enabled,scheduled_execution_enabled,kill_switch_engaged,execution_state,expected_worker_identity,expected_provider,control_provenance,correlation_id,planned_by,planned_at) VALUES ($1,$2,$3,$4,1,false,false,true,'disabled','subscriber-options-capture','massive',$5::jsonb,$6,$7,$8)`, gate.CapturePlanID, gate.SnapshotRunID, storage.SubscriberOptionsCaptureCanaryGateVersion, sessionDate, string(provenance), gate.CorrelationID, gate.PlannedBy, gate.PlannedAt.UTC()); err != nil {
		return gate, fmt.Errorf("insert disabled Options capture plan: %w", err)
	}
	memberProvenance, err := json.Marshal(map[string]any{"snapshot_run_id": gate.SnapshotRunID, "capture_plan_id": gate.CapturePlanID, "selection_state": "selected", "selection_priority": 1, "frozen_at": gate.PlannedAt.UTC().Format(time.RFC3339Nano)})
	if err != nil {
		return gate, fmt.Errorf("encode Options capture member provenance: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO subscriber_options_capture_canary_members (capture_plan_id,global_asset_id,expected_symbol,request_ordinal,expected_readiness_policy,expected_baseline_provenance) VALUES ($1,$2,$3,1,'subscriber-options-prospective-readiness-v1',$4::jsonb)`, gate.CapturePlanID, gate.GlobalAssetID, gate.Ticker, string(memberProvenance)); err != nil {
		return gate, fmt.Errorf("insert frozen Options capture member: %w", err)
	}
	evidence, err := json.Marshal(map[string]any{"provider_request_allowed": false, "expected_symbol": gate.Ticker, "request_ordinal": 1, "readiness_policy": "subscriber-options-prospective-readiness-v1"})
	if err != nil {
		return gate, fmt.Errorf("encode Options capture planned evidence: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO subscriber_options_capture_canary_evidence_events (evidence_event_id,capture_plan_id,global_asset_id,evidence_kind,event_ordinal,payload,provenance,recorded_by,recorded_at) VALUES ($1,$2,$3,'capture_planned',1,$4::jsonb,$5::jsonb,$6,$7)`, newSubscriberID("suboptcaptureevidence"), gate.CapturePlanID, gate.GlobalAssetID, string(evidence), string(memberProvenance), gate.PlannedBy, gate.PlannedAt.UTC()); err != nil {
		return gate, fmt.Errorf("insert Options capture planned evidence: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return gate, fmt.Errorf("commit disabled Options capture plan: %w", err)
	}
	return gate, nil
}
