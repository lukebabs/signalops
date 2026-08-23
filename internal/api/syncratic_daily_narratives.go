package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const (
	dailyNarrativeBatchStrategy       = "marketops_daily_narrative_batch_v1"
	dailyNarrativeBuilderVersion      = "marketops.syncratic.daily_context_builder.v1"
	dailyNarrativePayloadVersion      = "marketops.syncratic.daily_context_payload.v1"
	dailyNarrativeAskPromptVersion    = "marketops.syncratic.daily_narrative_prompt.v1"
	dailyNarrativeInsightType         = "marketops.syncratic.daily_narrative.v1"
	dailyNarrativeSubjectSymbol       = "MARKETOPS"
	dailyNarrativeStrategyOverview    = "marketops_daily_overview_v1"
	dailyNarrativeStrategySRI         = "marketops_sri_daily_v1"
	dailyNarrativeStrategyRiskReward  = "marketops_risk_reward_daily_v1"
	dailyNarrativeStrategyReviewQueue = "marketops_review_queue_daily_v1"
)

type syncraticDailyNarrativeMaterializeRequest struct {
	TenantID      string   `json:"tenant_id"`
	SessionDate   string   `json:"session_date"`
	Strategies    []string `json:"strategies"`
	EnqueueBriefs bool     `json:"enqueue_briefs"`
	DryRun        bool     `json:"dry_run"`
}

type syncraticDailyNarrativeMaterializeResponse struct {
	TenantID                   string                         `json:"tenant_id"`
	SessionDate                string                         `json:"session_date"`
	ContextBuilderVersion      string                         `json:"context_builder_version"`
	DryRun                     bool                           `json:"dry_run"`
	MaterializedContextWindows int                            `json:"materialized_context_windows"`
	MaterializedInsights       int                            `json:"materialized_insights"`
	SkippedUnchanged           int                            `json:"skipped_unchanged"`
	ContextWindowIDs           []string                       `json:"context_window_ids"`
	SyncraticInsightIDs        []string                       `json:"syncratic_insight_ids"`
	QueuedJobIDs               []string                       `json:"queued_job_ids"`
	Decisions                  []syncraticMaterializeDecision `json:"decisions"`
}

func syncraticDailyNarrativeStrategies(requested []string) []string {
	allowed := map[string]bool{
		dailyNarrativeStrategyOverview:    true,
		dailyNarrativeStrategySRI:         true,
		dailyNarrativeStrategyRiskReward:  true,
		dailyNarrativeStrategyReviewQueue: true,
	}
	if len(requested) == 0 {
		return []string{dailyNarrativeStrategyOverview, dailyNarrativeStrategySRI, dailyNarrativeStrategyRiskReward, dailyNarrativeStrategyReviewQueue}
	}
	out := []string{}
	seen := map[string]bool{}
	for _, item := range requested {
		item = strings.TrimSpace(item)
		if allowed[item] && !seen[item] {
			out = append(out, item)
			seen[item] = true
		}
	}
	return out
}

func isDailyNarrativeContextStrategy(strategy string) bool {
	switch strings.TrimSpace(strategy) {
	case dailyNarrativeStrategyOverview, dailyNarrativeStrategySRI, dailyNarrativeStrategyRiskReward, dailyNarrativeStrategyReviewQueue:
		return true
	default:
		return false
	}
}

func materializeSyncraticDailyNarratives(ctx context.Context, repo storage.QueryRepository, req syncraticDailyNarrativeMaterializeRequest) (syncraticDailyNarrativeMaterializeResponse, error) {
	tenantID := strings.TrimSpace(req.TenantID)
	if tenantID == "" {
		return syncraticDailyNarrativeMaterializeResponse{}, fmt.Errorf("tenant_id is required")
	}
	session, err := parseSyncraticDailyNarrativeSession(req.SessionDate)
	if err != nil {
		return syncraticDailyNarrativeMaterializeResponse{}, err
	}
	strategies := syncraticDailyNarrativeStrategies(req.Strategies)
	if len(strategies) == 0 {
		return syncraticDailyNarrativeMaterializeResponse{}, fmt.Errorf("at least one supported daily narrative strategy is required")
	}
	resp := syncraticDailyNarrativeMaterializeResponse{TenantID: tenantID, SessionDate: session.Format("2006-01-02"), ContextBuilderVersion: dailyNarrativeBuilderVersion, DryRun: req.DryRun}
	jobs, _ := repo.(storage.SyncraticIntelligenceJobRepository)
	for _, strategy := range strategies {
		contextWindow, err := buildSyncraticDailyNarrativeContext(ctx, repo, tenantID, session, strategy)
		if err != nil {
			return resp, err
		}
		decision := syncraticMaterializeDecision{SubjectSymbol: contextWindow.SubjectSymbol, EvidenceDigest: contextWindow.EvidenceDigest, ContextWindowID: contextWindow.ContextWindowID, EvidenceCount: dailyNarrativeEvidenceCount(contextWindow)}
		existing, err := repo.ListSyncraticContextWindows(ctx, storage.SyncraticContextWindowFilter{TenantID: tenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SubjectSymbol: dailyNarrativeSubjectSymbol, ContextStrategy: strategy, Limit: 10})
		if err != nil {
			return resp, err
		}
		unchanged := false
		for _, item := range existing {
			if item.IdempotencyKey == contextWindow.IdempotencyKey && item.EvidenceDigest == contextWindow.EvidenceDigest {
				unchanged = true
				break
			}
		}
		if unchanged {
			resp.SkippedUnchanged++
			decision.Action = "skipped"
			decision.Reason = "unchanged_evidence_digest"
			resp.Decisions = append(resp.Decisions, decision)
			continue
		}
		if req.DryRun {
			decision.Action = "would_materialize"
			decision.Reason = "eligible"
			resp.Decisions = append(resp.Decisions, decision)
			continue
		}
		if err := repo.UpsertSyncraticContextWindow(ctx, contextWindow); err != nil {
			return resp, err
		}
		insight := buildSyncraticDailyNarrativeInsight(contextWindow)
		if err := repo.UpsertSyncraticInsight(ctx, insight); err != nil {
			return resp, err
		}
		if req.EnqueueBriefs && jobs != nil {
			job := storage.SyncraticIntelligenceJobRecord{JobID: stableSyncraticID("synjob", tenantID, strategy, contextWindow.ContextWindowID, contextWindow.EvidenceDigest), TenantID: tenantID, AppID: "marketops", UseCase: "daily_market_surveillance", SubjectSymbol: contextWindow.SubjectSymbol, SessionDate: session, ContextWindowID: contextWindow.ContextWindowID, EvidenceDigest: contextWindow.EvidenceDigest, MaxAttempts: 3}
			if err := jobs.UpsertSyncraticIntelligenceJob(ctx, job); err != nil {
				return resp, err
			}
			resp.QueuedJobIDs = append(resp.QueuedJobIDs, job.JobID)
		}
		resp.MaterializedContextWindows++
		resp.MaterializedInsights++
		resp.ContextWindowIDs = append(resp.ContextWindowIDs, contextWindow.ContextWindowID)
		resp.SyncraticInsightIDs = append(resp.SyncraticInsightIDs, insight.SyncraticInsightID)
		decision.Action = "materialized"
		decision.Reason = "eligible"
		resp.Decisions = append(resp.Decisions, decision)
	}
	return resp, nil
}

func parseSyncraticDailyNarrativeSession(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return dateUTC(time.Now().UTC()), nil
	}
	session, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, fmt.Errorf("session_date must be YYYY-MM-DD")
	}
	return dateUTC(session), nil
}

func buildSyncraticDailyNarrativeContext(ctx context.Context, repo storage.QueryRepository, tenantID string, session time.Time, strategy string) (storage.SyncraticContextWindowRecord, error) {
	start := dateUTC(session)
	end := start.AddDate(0, 0, 1)
	priorStart := start.AddDate(0, 0, -30)
	metrics := map[string]any{"strategy": strategy, "session_date": start.Format("2006-01-02"), "sections": map[string]any{}}
	lineage := map[string]any{"strategy": strategy, "session_date": start.Format("2006-01-02"), "artifacts": map[string]any{}}
	warnings := []map[string]any{}
	record := storage.SyncraticContextWindowRecord{TenantID: tenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SubjectType: "market_scope", SubjectID: strategy, SubjectSymbol: dailyNarrativeSubjectSymbol, WindowStart: start, WindowEnd: end, ContextStrategy: strategy, ContextBuilderVersion: dailyNarrativeBuilderVersion, ContextPayloadVersion: dailyNarrativePayloadVersion, BaselineRefsJSON: []byte(`[]`), EvaluationRefsJSON: []byte(`[]`), PromotionCandidateRefsJSON: []byte(`[]`), Status: "active"}

	riskSummary, riskRefs, riskWarnings, err := dailyNarrativeRiskReward(ctx, repo, tenantID, priorStart, end)
	if err != nil {
		return storage.SyncraticContextWindowRecord{}, err
	}
	sriSummary, sriRefs, sriWarnings, err := dailyNarrativeSRI(ctx, repo, tenantID, priorStart, end)
	if err != nil {
		return storage.SyncraticContextWindowRecord{}, err
	}
	reviewSummary, reviewRefs, reviewWarnings, err := dailyNarrativeReviewQueue(ctx, repo, tenantID, priorStart, end)
	if err != nil {
		return storage.SyncraticContextWindowRecord{}, err
	}
	warnings = append(warnings, riskWarnings...)
	warnings = append(warnings, sriWarnings...)
	warnings = append(warnings, reviewWarnings...)

	sections := metrics["sections"].(map[string]any)
	artifacts := lineage["artifacts"].(map[string]any)
	switch strategy {
	case dailyNarrativeStrategyRiskReward:
		sections["risk_reward"] = riskSummary
		artifacts["risk_reward_snapshots"] = riskRefs
	case dailyNarrativeStrategySRI:
		sections["sri"] = sriSummary
		artifacts["sri_snapshots"] = sriRefs
	case dailyNarrativeStrategyReviewQueue:
		sections["review_queue"] = reviewSummary
		artifacts["opportunities"] = reviewRefs
		record.OpportunityIDs = stringSliceFromAny(reviewRefs)
	default:
		sections["risk_reward"] = riskSummary
		sections["sri"] = sriSummary
		sections["review_queue"] = reviewSummary
		artifacts["risk_reward_snapshots"] = riskRefs
		artifacts["sri_snapshots"] = sriRefs
		artifacts["opportunities"] = reviewRefs
		record.OpportunityIDs = stringSliceFromAny(reviewRefs)
	}
	record.EvaluationRefsJSON = mustJSON(lineage["artifacts"])
	record.SummaryMetricsJSON = mustJSON(metrics)
	record.QualityWarningsJSON = mustJSON(warnings)
	record.LineageRefsJSON = mustJSON(lineage)
	record.IdempotencyKey = syncraticMaterializationKey(tenantID, "daily_market_surveillance", strategy, dailyNarrativeSubjectSymbol, start, end, dailyNarrativeBuilderVersion)
	record.EvidenceDigest = syncraticDailyNarrativeEvidenceDigest(record)
	record.ContextWindowID = stableSyncraticID("synctx", record.IdempotencyKey)
	return record, nil
}

func dailyNarrativeRiskReward(ctx context.Context, repo storage.QueryRepository, tenantID string, start, end time.Time) (map[string]any, []string, []map[string]any, error) {
	reader, ok := repo.(storage.MarketOpsRiskRewardSnapshotRepository)
	if !ok {
		return map[string]any{"available": false}, nil, []map[string]any{{"code": "risk_reward_reader_unavailable", "blocking": false, "message": "Risk/Reward snapshot reader is unavailable."}}, nil
	}
	items, err := reader.ListMarketOpsRiskRewardSnapshots(ctx, storage.MarketOpsRiskRewardSnapshotFilter{TenantID: tenantID, SessionStart: start, SessionEnd: end, Limit: 5000})
	if err != nil {
		return nil, nil, nil, err
	}
	if len(items) == 0 {
		return map[string]any{"available": false, "snapshot_count": 0}, nil, []map[string]any{{"code": "risk_reward_evidence_missing", "blocking": false, "message": "No Risk/Reward snapshots were available for the narrative window."}}, nil
	}
	byDate := map[string]map[string]storage.MarketOpsRiskRewardSnapshotRecord{}
	refs := []string{}
	for _, item := range items {
		refs = append(refs, item.SnapshotID)
		date := item.SessionDate.UTC().Format("2006-01-02")
		if byDate[date] == nil {
			byDate[date] = map[string]storage.MarketOpsRiskRewardSnapshotRecord{}
		}
		current, exists := byDate[date][strings.ToUpper(item.Symbol)]
		if !exists || item.Eligible && !current.Eligible || item.UsableInputCount > current.UsableInputCount {
			byDate[date][strings.ToUpper(item.Symbol)] = item
		}
	}
	dateSet := map[string]struct{}{}
	for date := range byDate {
		dateSet[date] = struct{}{}
	}
	dates := mapKeys(dateSet)
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))
	latest := ""
	if len(dates) > 0 {
		latest = dates[0]
	}
	counts := map[string]int{"bullish": 0, "bearish": 0, "neutral": 0, "unavailable": 0}
	leaders := []map[string]any{}
	if latest != "" {
		for _, item := range byDate[latest] {
			dir := strings.ToLower(strings.TrimSpace(item.TechnicalDirection))
			if _, ok := counts[dir]; !ok {
				dir = "unavailable"
			}
			counts[dir]++
			leaders = append(leaders, map[string]any{"snapshot_id": item.SnapshotID, "symbol": item.Symbol, "direction": item.TechnicalDirection, "score": item.TechnicalScore, "risk_level": item.RiskLevel, "confidence": item.Confidence, "eligible": item.Eligible})
		}
	}
	sort.Slice(leaders, func(i, j int) bool {
		leftScore, rightScore := asFloat(leaders[i]["score"]), asFloat(leaders[j]["score"])
		if leftScore != rightScore {
			return leftScore > rightScore
		}
		return strings.ToUpper(fmt.Sprint(leaders[i]["symbol"])) < strings.ToUpper(fmt.Sprint(leaders[j]["symbol"]))
	})
	if len(leaders) > 12 {
		leaders = leaders[:12]
	}
	return map[string]any{"available": true, "latest_session_date": latest, "session_count": len(dates), "snapshot_count": len(items), "breadth": counts, "top_examples": leaders}, limitStrings(uniqueSorted(refs), 120), nil, nil
}

func dailyNarrativeSRI(ctx context.Context, repo storage.QueryRepository, tenantID string, start, end time.Time) (map[string]any, []string, []map[string]any, error) {
	var items []storage.MarketOpsSRISnapshotRecord
	var err error
	if global, ok := repo.(storage.SubscriberGlobalSRIRepository); ok {
		items, err = global.ListSubscriberGlobalSRISnapshots(ctx, storage.MarketOpsSRISnapshotFilter{TenantID: tenantID, SessionStart: start, SessionEnd: end, Limit: 1000})
	} else if local, ok := repo.(storage.MarketOpsSRIRepository); ok {
		items, err = local.ListMarketOpsSRISnapshots(ctx, storage.MarketOpsSRISnapshotFilter{TenantID: tenantID, SessionStart: start, SessionEnd: end, Limit: 1000})
	} else {
		return map[string]any{"available": false}, nil, []map[string]any{{"code": "sri_reader_unavailable", "blocking": false, "message": "SRI reader is unavailable."}}, nil
	}
	if err != nil {
		return nil, nil, nil, err
	}
	if len(items) == 0 {
		return map[string]any{"available": false, "snapshot_count": 0}, nil, []map[string]any{{"code": "sri_evidence_missing", "blocking": false, "message": "No SRI snapshots were available for the narrative window."}}, nil
	}
	latestBySegment := map[string]storage.MarketOpsSRISnapshotRecord{}
	refs := []string{}
	for _, item := range items {
		refs = append(refs, item.SnapshotID)
		current, exists := latestBySegment[item.SegmentID]
		if !exists || item.SessionDate.After(current.SessionDate) {
			latestBySegment[item.SegmentID] = item
		}
	}
	latest := []storage.MarketOpsSRISnapshotRecord{}
	for _, item := range latestBySegment {
		latest = append(latest, item)
	}
	sort.Slice(latest, func(i, j int) bool {
		ri, rj := 9999, 9999
		if latest[i].Rank != nil {
			ri = *latest[i].Rank
		}
		if latest[j].Rank != nil {
			rj = *latest[j].Rank
		}
		return ri < rj
	})
	leaders := []map[string]any{}
	for _, item := range latest {
		if len(leaders) >= 16 {
			break
		}
		leaders = append(leaders, map[string]any{"snapshot_id": item.SnapshotID, "segment_id": item.SegmentID, "session_date": item.SessionDate.UTC().Format("2006-01-02"), "state": item.State, "rank": nullableIntValue(item.Rank), "rank_change_5d": nullableIntValue(item.RankChange5D), "composite_score": nullableFloatValue(item.CompositeScore), "momentum_acceleration": nullableFloatValue(item.MomentumAcceleration), "quality_state": item.QualityState})
	}
	return map[string]any{"available": true, "snapshot_count": len(items), "latest_segments": len(latest), "leaders": leaders}, limitStrings(uniqueSorted(refs), 160), nil, nil
}

func dailyNarrativeReviewQueue(ctx context.Context, repo storage.QueryRepository, tenantID string, start, end time.Time) (map[string]any, []string, []map[string]any, error) {
	reader, ok := repo.(storage.MarketOpsOpportunityQueryRepository)
	if !ok {
		return map[string]any{"available": false}, nil, []map[string]any{{"code": "review_queue_reader_unavailable", "blocking": false, "message": "Review Queue opportunity reader is unavailable."}}, nil
	}
	items, err := reader.ListMarketOpsOpportunities(ctx, storage.MarketOpsOpportunityFilter{TenantID: tenantID, SessionStart: start, SessionEnd: end, Limit: 300})
	if err != nil {
		return nil, nil, nil, err
	}
	refs := []string{}
	counts := map[string]int{}
	active := []map[string]any{}
	expired := 0
	for _, item := range items {
		refs = append(refs, item.OpportunityID)
		counts[item.LifecycleStatus]++
		if item.LifecycleStatus == storage.MarketOpsOpportunityExpired {
			expired++
			continue
		}
		if len(active) < 20 {
			active = append(active, map[string]any{"opportunity_id": item.OpportunityID, "symbol": item.Symbol, "direction": item.Direction, "status": item.LifecycleStatus, "score": item.OpportunityScore, "confidence": item.ConfidenceScore, "summary": truncateText(item.Summary, 220), "last_evaluated_date": item.LastEvaluatedDate.UTC().Format("2006-01-02")})
		}
	}
	warnings := []map[string]any{}
	if len(items) == 0 {
		warnings = append(warnings, map[string]any{"code": "review_queue_empty", "blocking": false, "message": "No Review Queue opportunities were available for the narrative window."})
	}
	return map[string]any{"available": len(items) > 0, "opportunity_count": len(items), "status_counts": counts, "expired_count": expired, "active_examples": active}, limitStrings(uniqueSorted(refs), 120), warnings, nil
}

func buildSyncraticDailyNarrativeInsight(contextWindow storage.SyncraticContextWindowRecord) storage.SyncraticInsightRecord {
	title, summary, explanation := dailyNarrativeCopy(contextWindow)
	return storage.SyncraticInsightRecord{SyncraticInsightID: stableSyncraticID("synins", contextWindow.ContextWindowID, dailyNarrativeInsightType, dailyNarrativeBuilderVersion), TenantID: contextWindow.TenantID, AppID: contextWindow.AppID, Domain: contextWindow.Domain, UseCase: contextWindow.UseCase, ContextWindowID: contextWindow.ContextWindowID, InsightType: dailyNarrativeInsightType, SubjectType: contextWindow.SubjectType, SubjectID: contextWindow.SubjectID, SubjectSymbol: contextWindow.SubjectSymbol, Status: storage.SyncraticInsightStatusActive, Severity: "medium", Confidence: 0.7, Title: title, Summary: summary, Explanation: explanation, SupportingAlertIDs: contextWindow.AlertIDs, SupportingSignalIDs: contextWindow.SignalIDs, SupportingEventIDs: contextWindow.EventIDs, SupportingArtifactIDs: contextWindow.ArtifactIDs, RelatedGraphProposalIDs: contextWindow.GraphProposalIDs, RelatedLabelIDs: contextWindow.LabelIDs, MetricsJSON: json.RawMessage(jsonOrDefault(contextWindow.SummaryMetricsJSON, `{}`)), RecommendationJSON: mustJSON(map[string]any{"action": "review_context", "source": "deterministic_daily_narrative", "reason": "Review the cited MarketOps artifacts before relying on the generated narrative."}), BuilderVersion: dailyNarrativeBuilderVersion}
}

func dailyNarrativeCopy(contextWindow storage.SyncraticContextWindowRecord) (string, string, string) {
	strategy := strings.TrimSpace(contextWindow.ContextStrategy)
	session := contextWindow.WindowStart.UTC().Format("2006-01-02")
	names := map[string]string{dailyNarrativeStrategyOverview: "MarketOps daily overview", dailyNarrativeStrategySRI: "Sector Rotation daily overview", dailyNarrativeStrategyRiskReward: "Risk/Reward daily evolution", dailyNarrativeStrategyReviewQueue: "Review Queue daily brief"}
	name := firstNonEmpty(names[strategy], "MarketOps daily narrative")
	title := fmt.Sprintf("%s · %s", name, session)
	summary := "Deterministic context is ready for Syncratic narrative synthesis over cited MarketOps artifacts."
	explanation := "This daily narrative context was assembled from persisted MarketOps evidence only. Use the artifact references and quality warnings to verify the narrative; generated Syncratic Ask prose is an explainability layer, not a signal or recommendation."
	return title, summary, explanation
}

func buildSyncraticDailyNarrativeAskPrompt(contextWindow storage.SyncraticContextWindowRecord, req syncraticAskRequest) (string, syncraticAskPromptMeta, error) {
	version := firstNonEmpty(strings.TrimSpace(req.PromptBuilderVersion), dailyNarrativeAskPromptVersion)
	if version != dailyNarrativeAskPromptVersion {
		return "", syncraticAskPromptMeta{}, fmt.Errorf("daily narrative context requires prompt_builder_version %s", dailyNarrativeAskPromptVersion)
	}
	const (
		// Syncratic AI Gateway currently governs Ask at 4k input tokens / 1k output tokens.
		// The gateway tokenizer is not available in SignalOps, so daily narrative prompts
		// use a conservative serialized-byte proxy and compact/chunk before crossing it.
		dailyNarrativeInputTokenBudget       = 4000
		dailyNarrativeDefaultPromptByteProxy = 10000
		dailyNarrativeMaxPromptByteProxy     = 10000
	)
	maxPromptBytes := req.MaxPromptBytes
	if maxPromptBytes <= 0 {
		maxPromptBytes = dailyNarrativeDefaultPromptByteProxy
	}
	if maxPromptBytes < 1000 {
		return "", syncraticAskPromptMeta{}, fmt.Errorf("max_prompt_bytes must be at least 1000")
	}
	if maxPromptBytes > dailyNarrativeMaxPromptByteProxy {
		maxPromptBytes = dailyNarrativeMaxPromptByteProxy
	}
	profiles := []dailyNarrativePromptProfile{
		{LeaderLimit: 8, ExampleLimit: 8, RefSampleLimit: 12},
		{LeaderLimit: 5, ExampleLimit: 5, RefSampleLimit: 6},
		{LeaderLimit: 3, ExampleLimit: 3, RefSampleLimit: 3},
	}
	var lastPrompt string
	var lastCaps map[string]int
	for _, profile := range profiles {
		caps := map[string]int{"max_prompt_bytes": maxPromptBytes, "max_section_leaders": profile.LeaderLimit, "max_section_examples": profile.ExampleLimit, "max_artifact_ref_samples_per_kind": profile.RefSampleLimit}
		payload := map[string]any{
			"prompt_builder_version": version,
			"role":                   "MarketOps daily explainability layer over deterministic SignalOps evidence.",
			"prompt_mode":            dailyNarrativePromptMode(contextWindow.ContextStrategy),
			"compaction_policy": map[string]any{
				"full_provenance_location":   "syncratic_context_windows.lineage_refs",
				"prompt_contains":            "bounded summaries plus capped citation samples and total artifact counts",
				"reason":                     "chunked/map-reduce-ready Ask context; avoid cramming full session lineage into one prompt",
				"gateway_input_token_budget": dailyNarrativeInputTokenBudget,
				"local_prompt_byte_proxy":    maxPromptBytes,
			},
			"instructions": []string{
				"Use only the supplied JSON context; do not retrieve documents or use external knowledge.",
				"Start with what changed or what matters in the completed session.",
				"Separate observed facts, calculated metrics, inferred hypotheses, historical associations, governance state, and unknown future outcomes.",
				"Cite supplied artifact sample IDs for key claims and state when only counts are available because lineage was compacted.",
				"Call out missing or stale evidence as data quality, not as neutral market evidence.",
				"Keep expired Review Queue items separate from active items.",
				"Do not provide trading instructions, price targets, portfolio allocation, or certainty claims.",
			},
			"context_metadata":      map[string]any{"tenant_id": contextWindow.TenantID, "context_window_id": contextWindow.ContextWindowID, "subject": contextWindow.SubjectSymbol, "strategy": contextWindow.ContextStrategy, "session_date": contextWindow.WindowStart.UTC().Format("2006-01-02"), "evidence_digest": contextWindow.EvidenceDigest},
			"summary_metrics":       compactDailyNarrativeSummary(contextWindow.ContextStrategy, contextWindow.SummaryMetricsJSON, profile),
			"lineage_ref_summary":   compactDailyNarrativeLineage(contextWindow.LineageRefsJSON, profile.RefSampleLimit),
			"quality_warnings":      json.RawMessage(jsonOrDefault(contextWindow.QualityWarningsJSON, `[]`)),
			"recommended_next_step": dailyNarrativeNextStep(contextWindow.ContextStrategy),
			"output_contract": []string{
				"title",
				"executive_summary",
				"what_changed",
				"top_drivers",
				"contradictions_or_weak_evidence",
				"analyst_followups",
				"cited_artifacts",
				"data_quality_warnings",
			},
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			return "", syncraticAskPromptMeta{}, err
		}
		prompt := "Produce an evidence-grounded MarketOps daily narrative. Generated synthesis is explainability only and not deterministic evidence.\nCONTEXT_JSON:\n" + string(raw)
		lastPrompt = prompt
		lastCaps = caps
		if len(prompt) <= maxPromptBytes {
			sum := sha256.Sum256([]byte(prompt))
			return prompt, syncraticAskPromptMeta{PromptBuilderVersion: version, PromptDigest: "sha256:" + hex.EncodeToString(sum[:]), ContextEvidenceDigest: contextWindow.EvidenceDigest, MaxPromptBytes: maxPromptBytes, IncludedRecordDetails: true, Caps: caps, PromptBytes: len(prompt)}, nil
		}
	}
	return "", syncraticAskPromptMeta{PromptBuilderVersion: version, ContextEvidenceDigest: contextWindow.EvidenceDigest, MaxPromptBytes: maxPromptBytes, IncludedRecordDetails: true, Caps: lastCaps, PromptBytes: len(lastPrompt)}, fmt.Errorf("context_requires_chunking: compacted prompt bytes %d exceed max_prompt_bytes %d", len(lastPrompt), maxPromptBytes)
}

type dailyNarrativePromptProfile struct {
	LeaderLimit    int
	ExampleLimit   int
	RefSampleLimit int
}

func dailyNarrativePromptMode(strategy string) string {
	switch strings.TrimSpace(strategy) {
	case dailyNarrativeStrategyOverview:
		return "overview_synthesis_compact"
	case dailyNarrativeStrategySRI, dailyNarrativeStrategyRiskReward, dailyNarrativeStrategyReviewQueue:
		return "focused_narrative_compact"
	default:
		return "daily_narrative_compact"
	}
}

func dailyNarrativeNextStep(strategy string) string {
	switch strings.TrimSpace(strategy) {
	case dailyNarrativeStrategyOverview:
		return "Synthesize the focused section summaries; if more detail is needed, inspect the source section narrative and full stored lineage."
	case dailyNarrativeStrategySRI:
		return "Explain sector rotation movement and quality gaps using the bounded SRI leader set."
	case dailyNarrativeStrategyRiskReward:
		return "Explain Risk/Reward breadth and top directional examples using the bounded snapshot sample."
	case dailyNarrativeStrategyReviewQueue:
		return "Separate active from expired opportunities and summarize only the bounded active examples."
	default:
		return "Explain the bounded MarketOps daily context and cite supplied artifact samples."
	}
}

func compactDailyNarrativeSummary(strategy string, raw []byte, profile dailyNarrativePromptProfile) any {
	root := jsonObjectOrEmpty(raw)
	if len(root) == 0 {
		return map[string]any{}
	}
	root = copyMapAny(root)
	sections, ok := root["sections"].(map[string]any)
	if !ok {
		return compactDailyNarrativeValue(root, profile)
	}
	selected := map[string]any{}
	for key, value := range sections {
		selected[key] = compactDailyNarrativeSection(key, value, profile)
	}
	if strategy == dailyNarrativeStrategyOverview {
		root["sections"] = selected
		root["synthesis_note"] = "Daily Overview is a compact synthesis over section summaries. Full source lineage remains persisted on the context window."
		return compactDailyNarrativeValue(root, profile)
	}
	root["sections"] = selected
	return compactDailyNarrativeValue(root, profile)
}

func compactDailyNarrativeSection(section string, value any, profile dailyNarrativePromptProfile) any {
	sectionMap, ok := value.(map[string]any)
	if !ok {
		return compactDailyNarrativeValue(value, profile)
	}
	out := copyMapAny(sectionMap)
	for key, item := range out {
		switch key {
		case "leaders":
			out[key] = compactDailyNarrativeArray(item, profile.LeaderLimit, profile)
		case "top_examples", "active_examples":
			out[key] = compactDailyNarrativeArray(item, profile.ExampleLimit, profile)
		default:
			out[key] = compactDailyNarrativeValue(item, profile)
		}
	}
	return out
}

func compactDailyNarrativeLineage(raw []byte, sampleLimit int) map[string]any {
	root := jsonObjectOrEmpty(raw)
	out := map[string]any{}
	if strategy := asString(root["strategy"]); strategy != "" {
		out["strategy"] = strategy
	}
	artifacts, ok := root["artifacts"].(map[string]any)
	if !ok {
		out["artifacts"] = map[string]any{}
		return out
	}
	artifactOut := map[string]any{}
	keys := make([]string, 0, len(artifacts))
	for key := range artifacts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		refs := stringSliceFromAnyJSON(artifacts[key])
		refs = uniqueSorted(refs)
		artifactOut[key] = map[string]any{"total": len(refs), "sample": limitStrings(refs, sampleLimit)}
	}
	out["artifacts"] = artifactOut
	return out
}

func compactDailyNarrativeValue(value any, profile dailyNarrativePromptProfile) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			switch key {
			case "leaders":
				out[key] = compactDailyNarrativeArray(item, profile.LeaderLimit, profile)
			case "top_examples", "active_examples":
				out[key] = compactDailyNarrativeArray(item, profile.ExampleLimit, profile)
			case "summary":
				out[key] = truncateText(asString(item), 180)
			default:
				out[key] = compactDailyNarrativeValue(item, profile)
			}
		}
		return out
	case []any:
		return compactDailyNarrativeArray(typed, profile.ExampleLimit, profile)
	case string:
		return truncateText(typed, 220)
	default:
		return value
	}
}

func compactDailyNarrativeArray(value any, limit int, profile dailyNarrativePromptProfile) []any {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		out = append(out, compactDailyNarrativeValue(item, profile))
	}
	return out
}

func copyMapAny(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func stringSliceFromAnyJSON(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		out := []string{}
		for _, item := range typed {
			if text := asString(item); text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func syncraticDailyNarrativeEvidenceDigest(record storage.SyncraticContextWindowRecord) string {
	raw := mustJSON(map[string]any{"strategy": record.ContextStrategy, "metrics": json.RawMessage(jsonOrDefault(record.SummaryMetricsJSON, `{}`)), "lineage": json.RawMessage(jsonOrDefault(record.LineageRefsJSON, `{}`)), "warnings": json.RawMessage(jsonOrDefault(record.QualityWarningsJSON, `[]`))})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func dailyNarrativeEvidenceCount(record storage.SyncraticContextWindowRecord) int {
	var refs map[string]json.RawMessage
	if err := json.Unmarshal(record.EvaluationRefsJSON, &refs); err == nil {
		count := 0
		for _, raw := range refs {
			var items []string
			if err := json.Unmarshal(raw, &items); err == nil {
				count += len(items)
			}
		}
		if count > 0 {
			return count
		}
	}
	return len(record.SignalIDs) + len(record.AlertIDs) + len(record.MarketOpsEvidenceIDs) + len(record.OpportunityIDs) + len(record.OutcomeIDs) + len(record.CalibrationSummaryIDs)
}

func nullableFloatValue(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableIntValue(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func stringSliceFromAny(value any) []string {
	raw, ok := value.([]string)
	if ok {
		return raw
	}
	return nil
}
