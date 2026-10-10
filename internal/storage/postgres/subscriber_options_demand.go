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

func (r *Repository) ListSubscriberOptionsDemandAggregates(ctx context.Context) ([]storage.SubscriberOptionsDemandAggregate, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT global_asset_id,highest_tier_rank,eligible_tenant_count,eligible_watcher_count,deferred_sessions FROM subscriber_options_demand_aggregate()`)
	if err != nil {
		return nil, fmt.Errorf("list subscriber options demand aggregates: %w", err)
	}
	defer rows.Close()
	values := []storage.SubscriberOptionsDemandAggregate{}
	for rows.Next() {
		var value storage.SubscriberOptionsDemandAggregate
		if err := rows.Scan(&value.GlobalAssetID, &value.HighestTierRank, &value.EligibleTenantCount, &value.EligibleWatcherCount, &value.DeferredSessions); err != nil {
			return nil, fmt.Errorf("scan subscriber options demand aggregate: %w", err)
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (r *Repository) RecordSubscriberOptionsDemandShadowSnapshot(ctx context.Context, snapshot storage.SubscriberOptionsDemandSnapshot) (storage.SubscriberOptionsDemandSnapshot, error) {
	snapshot.SnapshotRunID, snapshot.PlannerVersion, snapshot.PlannedBy, snapshot.CorrelationID = strings.TrimSpace(snapshot.SnapshotRunID), strings.TrimSpace(snapshot.PlannerVersion), strings.TrimSpace(snapshot.PlannedBy), strings.TrimSpace(snapshot.CorrelationID)
	if snapshot.SnapshotRunID == "" {
		snapshot.SnapshotRunID = newSubscriberID("suboptdemand")
	}
	if snapshot.PlannerVersion == "" {
		snapshot.PlannerVersion = storage.SubscriberOptionsDemandShadowVersion
	}
	if snapshot.PlannedAt.IsZero() {
		snapshot.PlannedAt = time.Now().UTC()
	}
	if snapshot.SessionDate.IsZero() || snapshot.MaxSymbols < 1 || snapshot.MaxSymbols > 1000 || snapshot.SourceDemandCount < 0 || snapshot.PlannedBy != "subscriber-options-demand-planner" {
		return snapshot, errors.New("invalid subscriber options demand shadow snapshot")
	}
	selected, deferred := 0, 0
	for _, member := range snapshot.Members {
		if strings.TrimSpace(member.GlobalAssetID) == "" || member.Priority < 1 || member.HighestTierRank < 0 || member.EligibleTenantCount < 1 || member.EligibleWatcherCount < 1 || member.DeferredSessions < 0 {
			return snapshot, errors.New("invalid subscriber options demand snapshot member")
		}
		switch member.SelectionState {
		case "selected":
			selected++
		case "deferred":
			deferred++
		default:
			return snapshot, errors.New("invalid subscriber options demand selection state")
		}
	}
	if selected > snapshot.MaxSymbols || len(snapshot.Members) > 1000 {
		return snapshot, errors.New("invalid subscriber options demand snapshot capacity")
	}
	report, err := json.Marshal(map[string]any{"schema_version": storage.SubscriberOptionsDemandShadowVersion, "provider_execution_enabled": false, "scheduled_execution_enabled": false, "capture_execution_enabled": false})
	if err != nil {
		return snapshot, fmt.Errorf("encode options demand snapshot report: %w", err)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return snapshot, fmt.Errorf("begin options demand snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, `INSERT INTO subscriber_options_demand_snapshot_runs (snapshot_run_id,planner_version,session_date,execution_mode,max_symbols,source_demand_count,candidate_count,selected_count,deferred_count,report,planned_by,correlation_id,planned_at) VALUES ($1,$2,$3,'shadow',$4,$5,$6,$7,$8,$9::jsonb,$10,$11,$12)`, snapshot.SnapshotRunID, snapshot.PlannerVersion, snapshot.SessionDate.UTC(), snapshot.MaxSymbols, snapshot.SourceDemandCount, len(snapshot.Members), selected, deferred, string(report), snapshot.PlannedBy, snapshot.CorrelationID, snapshot.PlannedAt.UTC())
	if err != nil {
		return snapshot, fmt.Errorf("insert options demand snapshot: %w", err)
	}
	for _, member := range snapshot.Members {
		_, err = tx.ExecContext(ctx, `INSERT INTO subscriber_options_demand_snapshot_members (snapshot_run_id,global_asset_id,priority,selection_state,highest_tier_rank,eligible_tenant_count,eligible_watcher_count,deferred_sessions,selection_reason) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'entitled_watchlist_demand_union')`, snapshot.SnapshotRunID, member.GlobalAssetID, member.Priority, member.SelectionState, member.HighestTierRank, member.EligibleTenantCount, member.EligibleWatcherCount, member.DeferredSessions)
		if err != nil {
			return snapshot, fmt.Errorf("insert options demand snapshot member: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return snapshot, fmt.Errorf("commit options demand snapshot: %w", err)
	}
	return snapshot, nil
}
