package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

// StartSyncraticIntelligenceWorker runs the durable post-close brief queue.
// It is intentionally small and single-item: PostgreSQL leases make multiple
// gateway replicas safe without relying on in-memory ownership.
func StartSyncraticIntelligenceWorker(ctx context.Context, repo storage.QueryRepository, askClient syncraticAskClient) {
	jobs, ok := repo.(storage.SyncraticIntelligenceJobRepository)
	if !ok || askClient == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			processSyncraticIntelligenceJob(ctx, repo, jobs, askClient)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func processSyncraticIntelligenceJob(ctx context.Context, repo storage.QueryRepository, jobs storage.SyncraticIntelligenceJobRepository, askClient syncraticAskClient) {
	job, err := jobs.ClaimSyncraticIntelligenceJob(ctx, time.Now().UTC(), 2*time.Minute)
	if err != nil {
		return // an empty queue and a transient DB error are both retried next tick
	}
	contextWindow, err := repo.GetSyncraticContextWindow(ctx, job.ContextWindowID)
	if err != nil {
		_ = jobs.FailSyncraticIntelligenceJob(ctx, job.JobID, "context_window_not_found", strings.TrimSpace(err.Error()), time.Now().UTC())
		return
	}
	if !syncraticContextHasAnalystEvidence(contextWindow) {
		insight, err := syncraticInsightForContextType(ctx, repo, contextWindow, defaultSyncraticEODInsightType)
		if err != nil {
			_ = jobs.FailSyncraticIntelligenceJob(ctx, job.JobID, "insight_persistence_failed", strings.TrimSpace(err.Error()), time.Now().UTC())
			return
		}
		insight.Title = fmt.Sprintf("%s evidence coverage incomplete", contextWindow.SubjectSymbol)
		insight.Summary = "No market interpretation was generated because the completed session has no persisted supporting evidence for this asset."
		insight.Explanation = "Await persisted market-state, transition, evidence, signal, alert, or research-artifact rows before requesting an analyst explanation. This is a data-coverage status, not a market signal."
		insight.Severity = "low"
		insight.Confidence = 0
		insight.MetricsJSON = mustJSON(map[string]any{"syncratic_ask": map[string]any{"enabled": false, "ask_status": "skipped", "skipped_reason": "insufficient_persisted_evidence"}, "evidence_coverage": "missing"})
		insight.RecommendationJSON = mustJSON(map[string]any{"action": "await_evidence", "source": "deterministic_coverage_gate", "reason": "No persisted supporting evidence exists for this context window"})
		if err := repo.UpsertSyncraticInsight(ctx, insight); err != nil {
			_ = jobs.FailSyncraticIntelligenceJob(ctx, job.JobID, "insight_persistence_failed", strings.TrimSpace(err.Error()), time.Now().UTC())
			return
		}
		_ = jobs.CompleteSyncraticIntelligenceJob(ctx, job.JobID, insight.SyncraticInsightID, "", time.Now().UTC())
		return
	}
	insight, result, err := enrichSyncraticInsightWithAsk(ctx, repo, askClient, job.ContextWindowID, syncraticAskRequest{TenantID: job.TenantID, PromptBuilderVersion: "marketops.syncratic.eod_overview_prompt.v1", IncludeRecordDetails: true, InsightType: defaultSyncraticEODInsightType})
	if err != nil {
		code := "syncratic_ask_failed"
		if errors.Is(err, storage.ErrNotFound) { code = "context_window_not_found" }
		_ = jobs.FailSyncraticIntelligenceJob(ctx, job.JobID, code, strings.TrimSpace(err.Error()), time.Now().UTC())
		return
	}
	_ = jobs.CompleteSyncraticIntelligenceJob(ctx, job.JobID, insight.SyncraticInsightID, result.AskQueryID, time.Now().UTC())
}
func syncraticContextHasAnalystEvidence(contextWindow storage.SyncraticContextWindowRecord) bool {
	return len(contextWindow.MarketOpsEvidenceIDs)+len(contextWindow.SignalIDs)+len(contextWindow.AlertIDs)+len(contextWindow.ArtifactIDs)+
		len(contextWindow.GraphProposalIDs)+len(contextWindow.LabelIDs) > 0
}
