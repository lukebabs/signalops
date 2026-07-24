package api

import (
	"context"
	"fmt"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

// MaterializeSyncraticPostClose creates one bounded evidence context and one
// queued Ask brief per active asset. It is called only after the deterministic
// post-close cohort has completed; Ask never changes the underlying evidence.
func MaterializeSyncraticPostClose(ctx context.Context, repo storage.QueryRepository, tenantID, sessionDate string, dryRun bool) (syncraticMaterializeResponse, error) {
	date, err := time.Parse("2006-01-02", sessionDate)
	if err != nil { return syncraticMaterializeResponse{}, fmt.Errorf("session_date must be YYYY-MM-DD") }
	return materializeSyncraticContexts(ctx, repo, syncraticMaterializeRequest{
		TenantID: tenantID, UniverseGroup: "top50_megacap", ContextStrategy: "market_state_session_v2",
		ContextBuilderVersion: "syncratic.context_builder.v2", WindowStart: date.UTC().Format(time.RFC3339),
		WindowEnd: date.Add(24*time.Hour).UTC().Format(time.RFC3339), IncludeAllAssets: true, EnqueueBriefs: true,
		SessionDate: sessionDate, MaxAssets: 500, InsightType: defaultSyncraticEODInsightType, DryRun: dryRun,
	})
}
