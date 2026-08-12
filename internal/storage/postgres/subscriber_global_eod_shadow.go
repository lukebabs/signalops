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

func (r *Repository) ListSubscriberGlobalEODHotSetCandidates(ctx context.Context, limit int) ([]storage.SubscriberGlobalEODHotSetCandidate, error) {
	rows, err := r.db.QueryContext(ctx, `WITH canonical_sources AS (
  SELECT resolution.canonical_global_asset_id, source.global_asset_id, source.eligibility_status
  FROM subscriber_global_asset_identity_resolutions resolution
  JOIN subscriber_global_assets source ON source.global_asset_id=resolution.source_global_asset_id
), candidate_evidence AS (
  SELECT canonical_global_asset_id,
    bool_or(eligibility_status='eligible') AS has_eligible_source,
    count(link.*) FILTER (WHERE link.source_is_active)::int AS active_source_rows,
    COALESCE(min(link.source_rank) FILTER (WHERE link.source_is_active), 0)::int AS best_source_rank
  FROM canonical_sources source
  LEFT JOIN subscriber_global_asset_source_links link ON link.global_asset_id=source.global_asset_id
  GROUP BY canonical_global_asset_id
)
SELECT canonical_global_asset_id,
  CASE WHEN has_eligible_source THEN 'eligible' ELSE 'ineligible' END,
  active_source_rows, best_source_rank
FROM candidate_evidence
ORDER BY canonical_global_asset_id
LIMIT $1`, globalEODPlannerCandidateLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list global EOD hot-set candidates: %w", err)
	}
	defer rows.Close()
	records := []storage.SubscriberGlobalEODHotSetCandidate{}
	for rows.Next() {
		var record storage.SubscriberGlobalEODHotSetCandidate
		if err := rows.Scan(&record.GlobalAssetID, &record.EligibilityStatus, &record.ActiveSourceRows, &record.BestSourceRank); err != nil {
			return nil, fmt.Errorf("scan global EOD hot-set candidate: %w", err)
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

// RecordSubscriberGlobalEODHotSetShadowPlan persists a planner result only. Its
// execution_mode is always shadow and it does not modify coverage or schedule work.
func (r *Repository) RecordSubscriberGlobalEODHotSetShadowPlan(ctx context.Context, plan storage.SubscriberGlobalEODHotSetPlan) (storage.SubscriberGlobalEODHotSetPlan, error) {
	plan.PlanRunID = strings.TrimSpace(plan.PlanRunID)
	plan.PlannerVersion = strings.TrimSpace(plan.PlannerVersion)
	plan.PlannedBy = strings.TrimSpace(plan.PlannedBy)
	plan.CorrelationID = strings.TrimSpace(plan.CorrelationID)
	if plan.PlanRunID == "" {
		plan.PlanRunID = newSubscriberID("subeodplan")
	}
	if plan.PlannerVersion == "" {
		plan.PlannerVersion = storage.SubscriberGlobalEODPlannerShadowVersion
	}
	if plan.PlannedBy == "" || plan.Capacity <= 0 || plan.Capacity > 1000 || plan.CandidateCount < 0 || plan.EligibleCount < 0 || plan.ExcludedCount < 0 || len(plan.Members) > plan.Capacity {
		return plan, errors.New("invalid global EOD shadow plan")
	}
	if plan.PlannedAt.IsZero() {
		plan.PlannedAt = time.Now().UTC()
	}
	report, err := json.Marshal(map[string]any{
		"excluded_by_reason": plan.ExcludedByReason,
		"schema_version":     storage.SubscriberGlobalEODPlannerShadowVersion,
	})
	if err != nil {
		return plan, fmt.Errorf("encode global EOD shadow plan report: %w", err)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return plan, fmt.Errorf("begin global EOD shadow plan: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `INSERT INTO subscriber_global_eod_hot_set_plan_runs
  (plan_run_id, planner_version, execution_mode, capacity, candidate_count, eligible_count, selected_count, excluded_count, report, planned_by, correlation_id, planned_at)
VALUES ($1,$2,'shadow',$3,$4,$5,$6,$7,$8::jsonb,$9,$10,$11)`,
		plan.PlanRunID, plan.PlannerVersion, plan.Capacity, plan.CandidateCount, plan.EligibleCount, len(plan.Members), plan.ExcludedCount, string(report), plan.PlannedBy, plan.CorrelationID, plan.PlannedAt); err != nil {
		return plan, fmt.Errorf("insert global EOD shadow plan: %w", err)
	}
	for _, member := range plan.Members {
		if strings.TrimSpace(member.GlobalAssetID) == "" || member.Priority <= 0 {
			return plan, errors.New("invalid global EOD shadow plan member")
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO subscriber_global_eod_hot_set_plan_members
  (plan_run_id, global_asset_id, priority, source_rank, selection_reason)
VALUES ($1,$2,$3,NULLIF($4,0),'eligible_active_ranked')`, plan.PlanRunID, member.GlobalAssetID, member.Priority, member.SourceRank); err != nil {
			return plan, fmt.Errorf("insert global EOD shadow plan member: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return plan, fmt.Errorf("commit global EOD shadow plan: %w", err)
	}
	return plan, nil
}

func globalEODPlannerCandidateLimit(limit int) int {
	if limit <= 0 || limit > 10000 {
		return 10000
	}
	return limit
}
