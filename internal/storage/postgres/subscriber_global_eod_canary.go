package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
	"github.com/lukebabs/signalops/internal/subscriber/eodcanary"
)

// PrepareSubscriberGlobalEODCanary freezes a bounded cohort from an existing
// shadow plan. The inserted records are intentionally unable to enable a
// provider pull or scheduler execution.
func (r *Repository) PrepareSubscriberGlobalEODCanary(ctx context.Context, request storage.SubscriberGlobalEODCanaryPreparation) (storage.SubscriberGlobalEODCanaryPreparation, error) {
	request.CanaryRunID = strings.TrimSpace(request.CanaryRunID)
	request.PlanRunID = strings.TrimSpace(request.PlanRunID)
	request.PreparedBy = strings.TrimSpace(request.PreparedBy)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	if request.CanaryRunID == "" {
		request.CanaryRunID = newSubscriberID("subeodcanary")
	}
	if request.StartPriority == 0 {
		request.StartPriority = 1
	}
	if request.PlanRunID == "" || request.PreparedBy == "" || request.MaxSymbols <= 0 || request.MaxSymbols > eodcanary.MaximumCanarySize || request.SessionDate.IsZero() {
		return request, errors.New("invalid global EOD canary preparation")
	}
	if request.PreparedAt.IsZero() {
		request.PreparedAt = time.Now().UTC()
	}
	request.SessionDate = request.SessionDate.UTC()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return request, fmt.Errorf("begin global EOD canary preparation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var executionMode string
	if err := tx.QueryRowContext(ctx, `SELECT execution_mode FROM subscriber_global_eod_hot_set_plan_runs WHERE plan_run_id=$1`, request.PlanRunID).Scan(&executionMode); err != nil {
		return request, fmt.Errorf("load shadow plan for global EOD canary: %w", err)
	}
	if request.StartPriority <= 0 {
		return request, errors.New("global EOD canary start priority must be positive")
	}
	if executionMode != "shadow" {
		return request, errors.New("global EOD canary must derive from a shadow plan")
	}
	rows, err := tx.QueryContext(ctx, `SELECT global_asset_id,priority,COALESCE(source_rank,0) FROM subscriber_global_eod_hot_set_plan_members WHERE plan_run_id=$1 AND priority >= $2 ORDER BY priority,global_asset_id`, request.PlanRunID, request.StartPriority)
	if err != nil {
		return request, fmt.Errorf("list shadow plan members for global EOD canary: %w", err)
	}
	defer rows.Close()
	candidates := []eodcanary.Member{}
	for rows.Next() {
		var member eodcanary.Member
		if err := rows.Scan(&member.GlobalAssetID, &member.Priority, &member.SourceRank); err != nil {
			return request, fmt.Errorf("scan shadow plan member for global EOD canary: %w", err)
		}
		candidates = append(candidates, member)
	}
	if err := rows.Err(); err != nil {
		return request, fmt.Errorf("iterate shadow plan members for global EOD canary: %w", err)
	}
	selected, err := eodcanary.Select(candidates, request.MaxSymbols)
	if err != nil {
		return request, err
	}
	report, err := json.Marshal(map[string]any{
		"schema_version":              storage.SubscriberGlobalEODCanaryVersion,
		"source_plan_execution_mode":  executionMode,
		"provider_execution_enabled":  false,
		"scheduled_execution_enabled": false,
		"parity_required":             true,
		"start_priority":              request.StartPriority,
	})
	if err != nil {
		return request, fmt.Errorf("encode global EOD canary report: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO subscriber_global_eod_canary_runs
  (canary_run_id,plan_run_id,canary_version,session_date,execution_state,max_symbols,selected_count,parity_required,provider_execution_enabled,scheduled_execution_enabled,report,prepared_by,correlation_id,prepared_at)
VALUES ($1,$2,$3,$4,'prepared',$5,$6,true,false,false,$7::jsonb,$8,$9,$10)`, request.CanaryRunID, request.PlanRunID, storage.SubscriberGlobalEODCanaryVersion, request.SessionDate, request.MaxSymbols, len(selected), string(report), request.PreparedBy, request.CorrelationID, request.PreparedAt); err != nil {
		return request, fmt.Errorf("insert global EOD canary run: %w", err)
	}
	request.Members = make([]storage.SubscriberGlobalEODCanaryMember, 0, len(selected))
	for _, member := range selected {
		if _, err := tx.ExecContext(ctx, `INSERT INTO subscriber_global_eod_canary_members (canary_run_id,global_asset_id,priority,source_rank,selection_reason) VALUES ($1,$2,$3,NULLIF($4,0),'bounded_shadow_plan_priority')`, request.CanaryRunID, member.GlobalAssetID, member.Priority, member.SourceRank); err != nil {
			return request, fmt.Errorf("insert global EOD canary member: %w", err)
		}
		request.Members = append(request.Members, storage.SubscriberGlobalEODCanaryMember{GlobalAssetID: member.GlobalAssetID, Priority: member.Priority, SourceRank: member.SourceRank})
	}
	if err := tx.Commit(); err != nil {
		return request, fmt.Errorf("commit global EOD canary preparation: %w", err)
	}
	return request, nil
}
