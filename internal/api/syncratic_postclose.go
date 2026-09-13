package api

import (
	"context"
	"fmt"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

// MaterializeSyncraticPostClose creates one bounded evidence context per active
// asset and the aggregate MarketOps daily narrative contexts used by the
// Dashboard/Syncratic Intelligence views. It is called only after the
// deterministic post-close cohort has completed; Ask never changes the
// underlying evidence.
func MaterializeSyncraticPostClose(ctx context.Context, repo storage.QueryRepository, tenantID, sessionDate string, dryRun bool) (syncraticMaterializeResponse, error) {
	date, err := time.Parse("2006-01-02", sessionDate)
	if err != nil {
		return syncraticMaterializeResponse{}, fmt.Errorf("session_date must be YYYY-MM-DD")
	}
	result, err := materializeSyncraticContexts(ctx, repo, syncraticMaterializeRequest{
		TenantID: tenantID, UniverseGroup: "top50_megacap", ContextStrategy: "market_state_session_v2",
		ContextBuilderVersion: "syncratic.context_builder.v2", WindowStart: date.UTC().Format(time.RFC3339),
		WindowEnd: date.Add(24 * time.Hour).UTC().Format(time.RFC3339), IncludeAllAssets: true, EnqueueBriefs: true,
		SessionDate: sessionDate, MaxAssets: 500, InsightType: defaultSyncraticEODInsightType, DryRun: dryRun,
	})
	if err != nil {
		return result, err
	}
	daily, err := materializeSyncraticDailyNarratives(ctx, repo, syncraticDailyNarrativeMaterializeRequest{
		TenantID:      tenantID,
		SessionDate:   sessionDate,
		EnqueueBriefs: true,
		DryRun:        dryRun,
	})
	if err != nil {
		return result, fmt.Errorf("materialize daily Syncratic narratives: %w", err)
	}
	result.MaterializedContextWindows += daily.MaterializedContextWindows
	result.MaterializedInsights += daily.MaterializedInsights
	result.SkippedUnchanged += daily.SkippedUnchanged
	result.ContextWindowIDs = append(result.ContextWindowIDs, daily.ContextWindowIDs...)
	result.SyncraticInsightIDs = append(result.SyncraticInsightIDs, daily.SyncraticInsightIDs...)
	result.QueuedJobIDs = append(result.QueuedJobIDs, daily.QueuedJobIDs...)
	result.Decisions = append(result.Decisions, daily.Decisions...)
	return result, nil
}
