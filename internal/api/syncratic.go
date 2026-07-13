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
	defaultSyncraticBuilderVersion = "syncratic.context_builder.v1"
	defaultSyncraticInsightType    = "marketops.syncratic.multi_event_context"
)

type syncraticContextWindowCreateRequest struct {
	TenantID              string   `json:"tenant_id"`
	SubjectSymbol         string   `json:"subject_symbol"`
	ContextStrategy       string   `json:"context_strategy"`
	ContextBuilderVersion string   `json:"context_builder_version"`
	WindowStart           string   `json:"window_start"`
	WindowEnd             string   `json:"window_end"`
	SignalTypes           []string `json:"signal_types"`
}

type syncraticInsightCreateRequest struct {
	TenantID        string `json:"tenant_id"`
	ContextWindowID string `json:"context_window_id"`
	InsightType     string `json:"insight_type"`
	BuilderVersion  string `json:"builder_version"`
}

type syncraticMaterializeRequest struct {
	TenantID              string `json:"tenant_id"`
	UniverseGroup         string `json:"universe_group"`
	ContextStrategy       string `json:"context_strategy"`
	ContextBuilderVersion string `json:"context_builder_version"`
	WindowStart           string `json:"window_start"`
	WindowEnd             string `json:"window_end"`
	MinEvidenceCount      int    `json:"min_evidence_count"`
	MaxAssets             int    `json:"max_assets"`
	MaxCandidateWindows   int    `json:"max_candidate_windows"`
	MaxContextWindows     int    `json:"max_context_windows"`
	MaxInsights           int    `json:"max_insights"`
	SignalLimit           int    `json:"signal_limit"`
	AlertLimit            int    `json:"alert_limit"`
}

type syncraticMaterializeResponse struct {
	TenantID                   string    `json:"tenant_id"`
	UniverseGroup              string    `json:"universe_group"`
	ContextStrategy            string    `json:"context_strategy"`
	ContextBuilderVersion      string    `json:"context_builder_version"`
	WindowStart                time.Time `json:"window_start"`
	WindowEnd                  time.Time `json:"window_end"`
	ScannedAssets              int       `json:"scanned_assets"`
	CandidateWindows           int       `json:"candidate_windows"`
	MaterializedContextWindows int       `json:"materialized_context_windows"`
	MaterializedInsights       int       `json:"materialized_insights"`
	SkippedBelowThreshold      int       `json:"skipped_below_threshold"`
	SkippedUnchanged           int       `json:"skipped_unchanged"`
	SkippedBudgetCap           int       `json:"skipped_budget_cap"`
	ContextWindowIDs           []string  `json:"context_window_ids"`
	SyncraticInsightIDs        []string  `json:"syncratic_insight_ids"`
}

type syncraticContextWindowDTO struct {
	ContextWindowID        string          `json:"context_window_id"`
	TenantID               string          `json:"tenant_id"`
	AppID                  string          `json:"app_id"`
	Domain                 string          `json:"domain"`
	UseCase                string          `json:"use_case"`
	SubjectType            string          `json:"subject_type"`
	SubjectID              string          `json:"subject_id"`
	SubjectSymbol          string          `json:"subject_symbol"`
	WindowStart            time.Time       `json:"window_start"`
	WindowEnd              time.Time       `json:"window_end"`
	ContextStrategy        string          `json:"context_strategy"`
	ContextBuilderVersion  string          `json:"context_builder_version"`
	SignalTypes            []string        `json:"signal_types"`
	DetectorIDs            []string        `json:"detector_ids"`
	EventIDs               []string        `json:"event_ids"`
	SignalIDs              []string        `json:"signal_ids"`
	AlertIDs               []string        `json:"alert_ids"`
	ArtifactIDs            []string        `json:"artifact_ids"`
	GraphProposalIDs       []string        `json:"graph_proposal_ids"`
	LabelIDs               []string        `json:"label_ids"`
	BaselineRefs           json.RawMessage `json:"baseline_refs"`
	EvaluationRefs         json.RawMessage `json:"evaluation_refs"`
	PromotionCandidateRefs json.RawMessage `json:"promotion_candidate_refs"`
	SummaryMetrics         json.RawMessage `json:"summary_metrics"`
	EvidenceDigest         string          `json:"evidence_digest"`
	IdempotencyKey         string          `json:"idempotency_key"`
	Status                 string          `json:"status"`
	CreatedAt              time.Time       `json:"created_at"`
	UpdatedAt              time.Time       `json:"updated_at"`
}

type syncraticInsightDTO struct {
	SyncraticInsightID      string          `json:"syncratic_insight_id"`
	TenantID                string          `json:"tenant_id"`
	AppID                   string          `json:"app_id"`
	Domain                  string          `json:"domain"`
	UseCase                 string          `json:"use_case"`
	ContextWindowID         string          `json:"context_window_id"`
	InsightType             string          `json:"insight_type"`
	SubjectType             string          `json:"subject_type"`
	SubjectID               string          `json:"subject_id"`
	SubjectSymbol           string          `json:"subject_symbol"`
	Status                  string          `json:"status"`
	Severity                string          `json:"severity"`
	Confidence              float64         `json:"confidence"`
	Title                   string          `json:"title"`
	Summary                 string          `json:"summary"`
	Explanation             string          `json:"explanation"`
	SupportingAlertIDs      []string        `json:"supporting_alert_ids"`
	SupportingSignalIDs     []string        `json:"supporting_signal_ids"`
	SupportingEventIDs      []string        `json:"supporting_event_ids"`
	SupportingArtifactIDs   []string        `json:"supporting_artifact_ids"`
	RelatedGraphProposalIDs []string        `json:"related_graph_proposal_ids"`
	RelatedLabelIDs         []string        `json:"related_label_ids"`
	Metrics                 json.RawMessage `json:"metrics"`
	Recommendation          json.RawMessage `json:"recommendation"`
	BuilderVersion          string          `json:"builder_version"`
	CreatedAt               time.Time       `json:"created_at"`
	UpdatedAt               time.Time       `json:"updated_at"`
}

func syncraticContextWindowResponse(record storage.SyncraticContextWindowRecord) syncraticContextWindowDTO {
	return syncraticContextWindowDTO{ContextWindowID: record.ContextWindowID, TenantID: record.TenantID, AppID: record.AppID, Domain: record.Domain, UseCase: record.UseCase, SubjectType: record.SubjectType, SubjectID: record.SubjectID, SubjectSymbol: record.SubjectSymbol, WindowStart: record.WindowStart, WindowEnd: record.WindowEnd, ContextStrategy: record.ContextStrategy, ContextBuilderVersion: record.ContextBuilderVersion, SignalTypes: record.SignalTypes, DetectorIDs: record.DetectorIDs, EventIDs: record.EventIDs, SignalIDs: record.SignalIDs, AlertIDs: record.AlertIDs, ArtifactIDs: record.ArtifactIDs, GraphProposalIDs: record.GraphProposalIDs, LabelIDs: record.LabelIDs, BaselineRefs: json.RawMessage(jsonOrDefault(record.BaselineRefsJSON, `[]`)), EvaluationRefs: json.RawMessage(jsonOrDefault(record.EvaluationRefsJSON, `[]`)), PromotionCandidateRefs: json.RawMessage(jsonOrDefault(record.PromotionCandidateRefsJSON, `[]`)), SummaryMetrics: json.RawMessage(jsonOrDefault(record.SummaryMetricsJSON, `{}`)), EvidenceDigest: record.EvidenceDigest, IdempotencyKey: record.IdempotencyKey, Status: record.Status, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}

func syncraticContextWindowResponses(records []storage.SyncraticContextWindowRecord) []syncraticContextWindowDTO {
	responses := make([]syncraticContextWindowDTO, 0, len(records))
	for _, record := range records {
		responses = append(responses, syncraticContextWindowResponse(record))
	}
	return responses
}

func syncraticInsightResponse(record storage.SyncraticInsightRecord) syncraticInsightDTO {
	return syncraticInsightDTO{SyncraticInsightID: record.SyncraticInsightID, TenantID: record.TenantID, AppID: record.AppID, Domain: record.Domain, UseCase: record.UseCase, ContextWindowID: record.ContextWindowID, InsightType: record.InsightType, SubjectType: record.SubjectType, SubjectID: record.SubjectID, SubjectSymbol: record.SubjectSymbol, Status: record.Status, Severity: record.Severity, Confidence: record.Confidence, Title: record.Title, Summary: record.Summary, Explanation: record.Explanation, SupportingAlertIDs: record.SupportingAlertIDs, SupportingSignalIDs: record.SupportingSignalIDs, SupportingEventIDs: record.SupportingEventIDs, SupportingArtifactIDs: record.SupportingArtifactIDs, RelatedGraphProposalIDs: record.RelatedGraphProposalIDs, RelatedLabelIDs: record.RelatedLabelIDs, Metrics: json.RawMessage(jsonOrDefault(record.MetricsJSON, `{}`)), Recommendation: json.RawMessage(jsonOrDefault(record.RecommendationJSON, `{}`)), BuilderVersion: record.BuilderVersion, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}

func syncraticInsightResponses(records []storage.SyncraticInsightRecord) []syncraticInsightDTO {
	responses := make([]syncraticInsightDTO, 0, len(records))
	for _, record := range records {
		responses = append(responses, syncraticInsightResponse(record))
	}
	return responses
}

func buildSyncraticContextWindow(ctx context.Context, repo storage.QueryRepository, tenantID, subjectSymbol, strategy string, windowStart, windowEnd time.Time, builderVersion string, signalTypes []string, signalLimit, alertLimit int) (storage.SyncraticContextWindowRecord, error) {
	tenantID = strings.TrimSpace(tenantID)
	subjectSymbol = strings.ToUpper(strings.TrimSpace(subjectSymbol))
	strategy = firstNonEmpty(strings.TrimSpace(strategy), "symbol_signal_cluster_5d")
	builderVersion = firstNonEmpty(strings.TrimSpace(builderVersion), defaultSyncraticBuilderVersion)
	if tenantID == "" || subjectSymbol == "" || windowStart.IsZero() || windowEnd.IsZero() || !windowEnd.After(windowStart) {
		return storage.SyncraticContextWindowRecord{}, fmt.Errorf("tenant_id, subject_symbol, valid window_start, and valid window_end are required")
	}
	signals, err := repo.ListSignalLedger(ctx, storage.SignalLedgerFilter{TenantID: tenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", Limit: signalLimitOrDefault(signalLimit)})
	if err != nil {
		return storage.SyncraticContextWindowRecord{}, err
	}
	alerts, err := repo.ListAlertLedger(ctx, storage.AlertLedgerFilter{TenantID: tenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", Limit: alertLimitOrDefault(alertLimit)})
	if err != nil {
		return storage.SyncraticContextWindowRecord{}, err
	}
	artifacts, err := repo.ListMarketOpsDSMArtifacts(ctx, storage.MarketOpsDSMArtifactFilter{TenantID: tenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SubjectSymbol: subjectSymbol, Limit: 500})
	if err != nil {
		return storage.SyncraticContextWindowRecord{}, err
	}
	proposals, err := repo.ListMarketOpsDSMGraphProposals(ctx, storage.MarketOpsDSMGraphProposalFilter{TenantID: tenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SubjectSymbol: subjectSymbol, Limit: 500})
	if err != nil {
		return storage.SyncraticContextWindowRecord{}, err
	}
	labels, err := repo.ListMarketOpsBacktestEvaluationLabels(ctx, storage.MarketOpsBacktestEvaluationLabelFilter{TenantID: tenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SubjectSymbol: subjectSymbol, Limit: 200})
	if err != nil {
		return storage.SyncraticContextWindowRecord{}, err
	}
	promotions, err := repo.ListMarketOpsBacktestPromotionCandidates(ctx, storage.MarketOpsBacktestPromotionCandidateFilter{TenantID: tenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", Limit: 100})
	if err != nil {
		return storage.SyncraticContextWindowRecord{}, err
	}

	allowedTypes := stringSet(signalTypes)
	record := storage.SyncraticContextWindowRecord{TenantID: tenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SubjectType: "ticker", SubjectID: subjectSymbol, SubjectSymbol: subjectSymbol, WindowStart: windowStart.UTC(), WindowEnd: windowEnd.UTC(), ContextStrategy: strategy, ContextBuilderVersion: builderVersion, Status: "active"}
	detectors := map[string]struct{}{}
	signalTypeSet := map[string]struct{}{}
	for _, signal := range signals {
		if !timeInWindow(signal.SignalTime, windowStart, windowEnd) || !recordMatchesSymbol(signal.EntitiesJSON, signal.EventJSON, subjectSymbol) {
			continue
		}
		if len(allowedTypes) > 0 {
			if _, ok := allowedTypes[signal.SignalType]; !ok {
				continue
			}
		}
		record.SignalIDs = append(record.SignalIDs, signal.SignalID)
		record.EventIDs = append(record.EventIDs, signal.EventIDs...)
		signalTypeSet[signal.SignalType] = struct{}{}
		if strings.TrimSpace(signal.DetectorID) != "" {
			detectors[signal.DetectorID] = struct{}{}
		}
	}
	for _, alert := range alerts {
		if !timeInWindow(alert.LastObservedAt, windowStart, windowEnd) || !recordMatchesSymbol(alert.EntitiesJSON, alert.EvidenceJSON, subjectSymbol) {
			continue
		}
		record.AlertIDs = append(record.AlertIDs, alert.AlertID)
		record.EventIDs = append(record.EventIDs, alert.EventIDs...)
		if strings.TrimSpace(alert.DetectorID) != "" {
			detectors[alert.DetectorID] = struct{}{}
		}
	}
	for _, artifact := range artifacts {
		if !timeInWindow(artifact.UpdatedAt, windowStart, windowEnd) {
			continue
		}
		record.ArtifactIDs = append(record.ArtifactIDs, artifact.ArtifactID)
		record.EventIDs = append(record.EventIDs, artifact.EventIDs...)
		if strings.TrimSpace(artifact.SignalID) != "" {
			record.SignalIDs = append(record.SignalIDs, artifact.SignalID)
		}
		if strings.TrimSpace(artifact.SignalType) != "" {
			signalTypeSet[artifact.SignalType] = struct{}{}
		}
		if strings.TrimSpace(artifact.DetectorID) != "" {
			detectors[artifact.DetectorID] = struct{}{}
		}
	}
	for _, proposal := range proposals {
		if !timeInWindow(proposal.UpdatedAt, windowStart, windowEnd) {
			continue
		}
		record.GraphProposalIDs = append(record.GraphProposalIDs, proposal.ProposalID)
		record.EventIDs = append(record.EventIDs, proposal.EventIDs...)
	}
	for _, label := range labels {
		if !timeInWindow(label.LabeledAt, windowStart, windowEnd) {
			continue
		}
		record.LabelIDs = append(record.LabelIDs, label.LabelID)
	}
	promotionRefs := []map[string]string{}
	for _, promotion := range promotions {
		promotionRefs = append(promotionRefs, map[string]string{"candidate_id": promotion.CandidateID, "status": promotion.Status, "readiness_status": promotion.ReadinessStatus})
	}
	record.SignalIDs = uniqueSorted(record.SignalIDs)
	record.AlertIDs = uniqueSorted(record.AlertIDs)
	record.EventIDs = uniqueSorted(record.EventIDs)
	record.ArtifactIDs = uniqueSorted(record.ArtifactIDs)
	record.GraphProposalIDs = uniqueSorted(record.GraphProposalIDs)
	record.LabelIDs = uniqueSorted(record.LabelIDs)
	record.SignalTypes = setKeys(signalTypeSet)
	record.DetectorIDs = setKeys(detectors)
	metrics := map[string]any{"signal_count": len(record.SignalIDs), "alert_count": len(record.AlertIDs), "event_count": len(record.EventIDs), "artifact_count": len(record.ArtifactIDs), "graph_proposal_count": len(record.GraphProposalIDs), "label_count": len(record.LabelIDs), "subject_symbol": subjectSymbol, "context_strategy": strategy}
	record.SummaryMetricsJSON = mustJSON(metrics)
	record.BaselineRefsJSON = []byte(`[]`)
	record.EvaluationRefsJSON = []byte(`[]`)
	record.PromotionCandidateRefsJSON = mustJSON(promotionRefs)
	record.IdempotencyKey = syncraticMaterializationKey(tenantID, record.UseCase, strategy, subjectSymbol, record.WindowStart, record.WindowEnd, builderVersion)
	record.EvidenceDigest = syncraticEvidenceDigest(record)
	record.ContextWindowID = stableSyncraticID("synctx", record.IdempotencyKey)
	return record, nil
}

func buildSyncraticInsight(contextWindow storage.SyncraticContextWindowRecord, insightType, builderVersion string) storage.SyncraticInsightRecord {
	insightType = firstNonEmpty(strings.TrimSpace(insightType), defaultSyncraticInsightType)
	builderVersion = firstNonEmpty(strings.TrimSpace(builderVersion), contextWindow.ContextBuilderVersion, defaultSyncraticBuilderVersion)
	severity := "medium"
	if len(contextWindow.AlertIDs) == 0 {
		severity = "low"
	}
	confidence := 0.65
	if len(contextWindow.AlertIDs)+len(contextWindow.SignalIDs) >= 3 {
		confidence = 0.75
	}
	metrics := json.RawMessage(jsonOrDefault(contextWindow.SummaryMetricsJSON, `{}`))
	recommendation := mustJSON(map[string]any{"action": "review_context", "reason": "Syncratic insight is derived from deterministic multi-record evidence"})
	return storage.SyncraticInsightRecord{SyncraticInsightID: stableSyncraticID("synins", contextWindow.ContextWindowID, insightType, builderVersion), TenantID: contextWindow.TenantID, AppID: contextWindow.AppID, Domain: contextWindow.Domain, UseCase: contextWindow.UseCase, ContextWindowID: contextWindow.ContextWindowID, InsightType: insightType, SubjectType: contextWindow.SubjectType, SubjectID: contextWindow.SubjectID, SubjectSymbol: contextWindow.SubjectSymbol, Status: storage.SyncraticInsightStatusActive, Severity: severity, Confidence: confidence, Title: fmt.Sprintf("%s Syncratic context", contextWindow.SubjectSymbol), Summary: fmt.Sprintf("%s has %d supporting signals and %d supporting alerts in the %s window.", contextWindow.SubjectSymbol, len(contextWindow.SignalIDs), len(contextWindow.AlertIDs), contextWindow.ContextStrategy), Explanation: "This insight was synthesized from a deterministic Syncratic context window over persisted SignalOps and MarketOps evidence.", SupportingAlertIDs: contextWindow.AlertIDs, SupportingSignalIDs: contextWindow.SignalIDs, SupportingEventIDs: contextWindow.EventIDs, SupportingArtifactIDs: contextWindow.ArtifactIDs, RelatedGraphProposalIDs: contextWindow.GraphProposalIDs, RelatedLabelIDs: contextWindow.LabelIDs, MetricsJSON: metrics, RecommendationJSON: recommendation, BuilderVersion: builderVersion}
}

func materializeSyncraticContexts(ctx context.Context, repo storage.QueryRepository, req syncraticMaterializeRequest) (syncraticMaterializeResponse, error) {
	tenantID := strings.TrimSpace(req.TenantID)
	if tenantID == "" {
		return syncraticMaterializeResponse{}, fmt.Errorf("tenant_id is required")
	}
	windowStart, err := parseRFC3339(req.WindowStart)
	if err != nil {
		return syncraticMaterializeResponse{}, fmt.Errorf("window_start is required")
	}
	windowEnd, err := parseRFC3339(req.WindowEnd)
	if err != nil || !windowEnd.After(windowStart) {
		return syncraticMaterializeResponse{}, fmt.Errorf("valid window_end is required")
	}
	universeGroup := firstNonEmpty(req.UniverseGroup, "top50_megacap")
	strategy := firstNonEmpty(req.ContextStrategy, "symbol_signal_cluster_5d")
	builderVersion := firstNonEmpty(req.ContextBuilderVersion, defaultSyncraticBuilderVersion)
	minEvidence := req.MinEvidenceCount
	if minEvidence <= 0 {
		minEvidence = 2
	}
	maxAssets := bounded(req.MaxAssets, 50)
	maxCandidates := bounded(req.MaxCandidateWindows, 50)
	maxContexts := bounded(req.MaxContextWindows, 10)
	maxInsights := bounded(req.MaxInsights, 10)
	assets, err := repo.ListMarketOpsAssets(ctx, tenantID, universeGroup, true, maxAssets)
	if err != nil {
		return syncraticMaterializeResponse{}, err
	}
	resp := syncraticMaterializeResponse{TenantID: tenantID, UniverseGroup: universeGroup, ContextStrategy: strategy, ContextBuilderVersion: builderVersion, WindowStart: windowStart.UTC(), WindowEnd: windowEnd.UTC()}
	for _, asset := range assets {
		resp.ScannedAssets++
		if resp.CandidateWindows >= maxCandidates {
			resp.SkippedBudgetCap++
			continue
		}
		contextWindow, err := buildSyncraticContextWindow(ctx, repo, tenantID, asset.Ticker, strategy, windowStart, windowEnd, builderVersion, nil, req.SignalLimit, req.AlertLimit)
		if err != nil {
			return resp, err
		}
		evidenceCount := len(contextWindow.SignalIDs) + len(contextWindow.AlertIDs)
		critical := false
		alerts, err := repo.ListAlertLedger(ctx, storage.AlertLedgerFilter{TenantID: tenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", Severity: "critical", Limit: alertLimitOrDefault(req.AlertLimit)})
		if err != nil {
			return resp, err
		}
		for _, alert := range alerts {
			if timeInWindow(alert.LastObservedAt, windowStart, windowEnd) && recordMatchesSymbol(alert.EntitiesJSON, alert.EvidenceJSON, asset.Ticker) {
				critical = true
				break
			}
		}
		related := len(contextWindow.GraphProposalIDs) > 0 || len(contextWindow.LabelIDs) > 0
		if evidenceCount < minEvidence && !critical && !related {
			resp.SkippedBelowThreshold++
			continue
		}
		resp.CandidateWindows++
		existing, err := repo.ListSyncraticContextWindows(ctx, storage.SyncraticContextWindowFilter{TenantID: tenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SubjectSymbol: contextWindow.SubjectSymbol, ContextStrategy: strategy, Limit: 20})
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
			continue
		}
		if resp.MaterializedContextWindows >= maxContexts || resp.MaterializedInsights >= maxInsights {
			resp.SkippedBudgetCap++
			continue
		}
		if err := repo.UpsertSyncraticContextWindow(ctx, contextWindow); err != nil {
			return resp, err
		}
		insight := buildSyncraticInsight(contextWindow, defaultSyncraticInsightType, builderVersion)
		if err := repo.UpsertSyncraticInsight(ctx, insight); err != nil {
			return resp, err
		}
		resp.MaterializedContextWindows++
		resp.MaterializedInsights++
		resp.ContextWindowIDs = append(resp.ContextWindowIDs, contextWindow.ContextWindowID)
		resp.SyncraticInsightIDs = append(resp.SyncraticInsightIDs, insight.SyncraticInsightID)
	}
	return resp, nil
}

func signalLimitOrDefault(limit int) int { return bounded(limit, 1000) }
func alertLimitOrDefault(limit int) int  { return bounded(limit, 1000) }
func bounded(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	if value > 5000 {
		return 5000
	}
	return value
}

func parseRFC3339(value string) (time.Time, error) {
	return time.Parse(time.RFC3339, strings.TrimSpace(value))
}
func timeInWindow(value, start, end time.Time) bool {
	return !value.IsZero() && (value.Equal(start) || value.After(start)) && value.Before(end)
}
func stringSet(values []string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			set[value] = struct{}{}
		}
	}
	return set
}
func setKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		if k != "" {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}
func uniqueSorted(values []string) []string {
	set := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			set[value] = struct{}{}
		}
	}
	return setKeys(set)
}
func mustJSON(value any) []byte {
	raw, err := json.Marshal(value)
	if err != nil {
		return []byte(`{}`)
	}
	return raw
}
func stableSyncraticID(prefix string, parts ...string) string {
	h := sha256.New()
	h.Write([]byte(prefix))
	h.Write([]byte{0})
	for _, part := range parts {
		h.Write([]byte(strings.TrimSpace(part)))
		h.Write([]byte{0})
	}
	return prefix + "_" + hex.EncodeToString(h.Sum(nil))[:24]
}
func syncraticMaterializationKey(tenantID, useCase, strategy, symbol string, start, end time.Time, builderVersion string) string {
	return strings.Join([]string{tenantID, useCase, strategy, symbol, start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339), builderVersion}, "|")
}
func syncraticEvidenceDigest(record storage.SyncraticContextWindowRecord) string {
	raw := mustJSON(map[string]any{"events": record.EventIDs, "signals": record.SignalIDs, "alerts": record.AlertIDs, "artifacts": record.ArtifactIDs, "graph_proposals": record.GraphProposalIDs, "labels": record.LabelIDs, "metrics": json.RawMessage(jsonOrDefault(record.SummaryMetricsJSON, `{}`))})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func recordMatchesSymbolValue(raw []byte, symbol string) bool {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" || len(raw) == 0 {
		return false
	}
	var value any
	if err := json.Unmarshal(raw, &value); err == nil && jsonValueContainsSymbol(value, symbol) {
		return true
	}
	upperRaw := strings.ToUpper(string(raw))
	return strings.Contains(upperRaw, symbol)
}

func jsonValueContainsSymbol(value any, symbol string) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			lowerKey := strings.ToLower(strings.TrimSpace(key))
			if text, ok := item.(string); ok {
				candidate := strings.ToUpper(strings.TrimSpace(text))
				switch lowerKey {
				case "symbol", "ticker", "subject_symbol", "entity_id", "id", "value":
					if candidate == symbol {
						return true
					}
				}
			}
			if jsonValueContainsSymbol(item, symbol) {
				return true
			}
		}
	case []any:
		for _, item := range typed {
			if jsonValueContainsSymbol(item, symbol) {
				return true
			}
		}
	}
	return false
}

func recordMatchesSymbol(rawA []byte, rawB []byte, symbol string) bool {
	return recordMatchesSymbolValue(rawA, symbol) || recordMatchesSymbolValue(rawB, symbol)
}
