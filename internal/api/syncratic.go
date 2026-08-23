package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
	"github.com/lukebabs/signalops/internal/syncratic/userapi"
)

const (
	defaultSyncraticBuilderVersion   = "syncratic.context_builder.v3"
	defaultSyncraticInsightType      = "marketops.syncratic.multi_event_context"
	defaultSyncraticEODInsightType   = "marketops.syncratic.eod_overview.v1"
	defaultSyncraticAskDrilldownType = "marketops.syncratic.ask_drilldown.v1"
	defaultSyncraticAskPromptVersion = "marketops.syncratic.ask_prompt.v2"
	defaultSyncraticAskScope         = "tenant"
)

type syncraticAskClient interface {
	Ask(context.Context, userapi.AskRequest) (userapi.AskResponse, error)
}

type syncraticContextWindowCreateRequest struct {
	TenantID              string   `json:"tenant_id"`
	MarketStateID         string   `json:"market_state_id"`
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
	IncludeAllAssets      bool   `json:"include_all_assets"`
	EnqueueBriefs         bool   `json:"enqueue_briefs"`
	SessionDate           string `json:"session_date"`
	InsightType           string `json:"insight_type"`
	DryRun                bool   `json:"dry_run"`
}

type syncraticAskRequest struct {
	TenantID             string `json:"tenant_id"`
	PromptBuilderVersion string `json:"prompt_builder_version"`
	MaxPromptBytes       int    `json:"max_prompt_bytes"`
	IncludeRecordDetails bool   `json:"include_record_details"`
	Force                bool   `json:"force"`
	InsightType          string `json:"insight_type"`
}

type syncraticAskResult struct {
	ContextWindowID      string `json:"context_window_id"`
	SyncraticInsightID   string `json:"syncratic_insight_id"`
	AskQueryID           string `json:"ask_query_id"`
	AskStatus            string `json:"ask_status"`
	PromptDigest         string `json:"prompt_digest"`
	Updated              bool   `json:"updated"`
	SkippedReason        string `json:"skipped_reason"`
	PromptBuilderVersion string `json:"prompt_builder_version"`
}

type syncraticMaterializeResponse struct {
	TenantID                   string                         `json:"tenant_id"`
	UniverseGroup              string                         `json:"universe_group"`
	ContextStrategy            string                         `json:"context_strategy"`
	ContextBuilderVersion      string                         `json:"context_builder_version"`
	WindowStart                time.Time                      `json:"window_start"`
	WindowEnd                  time.Time                      `json:"window_end"`
	DryRun                     bool                           `json:"dry_run"`
	ScannedAssets              int                            `json:"scanned_assets"`
	CandidateWindows           int                            `json:"candidate_windows"`
	MaterializedContextWindows int                            `json:"materialized_context_windows"`
	MaterializedInsights       int                            `json:"materialized_insights"`
	SkippedBelowThreshold      int                            `json:"skipped_below_threshold"`
	SkippedUnchanged           int                            `json:"skipped_unchanged"`
	SkippedBudgetCap           int                            `json:"skipped_budget_cap"`
	ContextWindowIDs           []string                       `json:"context_window_ids"`
	SyncraticInsightIDs        []string                       `json:"syncratic_insight_ids"`
	QueuedJobIDs               []string                       `json:"queued_job_ids"`
	Decisions                  []syncraticMaterializeDecision `json:"decisions"`
}

type syncraticMaterializeDecision struct {
	SubjectSymbol      string `json:"subject_symbol"`
	Action             string `json:"action"`
	Reason             string `json:"reason"`
	EvidenceCount      int    `json:"evidence_count"`
	SignalCount        int    `json:"signal_count"`
	AlertCount         int    `json:"alert_count"`
	ArtifactCount      int    `json:"artifact_count"`
	GraphProposalCount int    `json:"graph_proposal_count"`
	LabelCount         int    `json:"label_count"`
	CriticalAlert      bool   `json:"critical_alert"`
	RelatedEvidence    bool   `json:"related_evidence"`
	EvidenceDigest     string `json:"evidence_digest,omitempty"`
	ContextWindowID    string `json:"context_window_id,omitempty"`
}

type syncraticContextWindowDTO struct {
	ContextWindowID         string          `json:"context_window_id"`
	TenantID                string          `json:"tenant_id"`
	AppID                   string          `json:"app_id"`
	Domain                  string          `json:"domain"`
	UseCase                 string          `json:"use_case"`
	SubjectType             string          `json:"subject_type"`
	SubjectID               string          `json:"subject_id"`
	SubjectSymbol           string          `json:"subject_symbol"`
	WindowStart             time.Time       `json:"window_start"`
	WindowEnd               time.Time       `json:"window_end"`
	ContextStrategy         string          `json:"context_strategy"`
	ContextBuilderVersion   string          `json:"context_builder_version"`
	ContextPayloadVersion   string          `json:"context_payload_version"`
	SignalTypes             []string        `json:"signal_types"`
	DetectorIDs             []string        `json:"detector_ids"`
	EventIDs                []string        `json:"event_ids"`
	SignalIDs               []string        `json:"signal_ids"`
	AlertIDs                []string        `json:"alert_ids"`
	ArtifactIDs             []string        `json:"artifact_ids"`
	GraphProposalIDs        []string        `json:"graph_proposal_ids"`
	LabelIDs                []string        `json:"label_ids"`
	MarketStateIDs          []string        `json:"market_state_ids"`
	StateTransitionIDs      []string        `json:"state_transition_ids"`
	MarketOpsEvidenceIDs    []string        `json:"marketops_evidence_ids"`
	HypothesisEvaluationIDs []string        `json:"hypothesis_evaluation_ids"`
	OpportunityIDs          []string        `json:"opportunity_ids"`
	OutcomeIDs              []string        `json:"outcome_ids"`
	CalibrationSummaryIDs   []string        `json:"calibration_summary_ids"`
	BaselineRefs            json.RawMessage `json:"baseline_refs"`
	EvaluationRefs          json.RawMessage `json:"evaluation_refs"`
	PromotionCandidateRefs  json.RawMessage `json:"promotion_candidate_refs"`
	SummaryMetrics          json.RawMessage `json:"summary_metrics"`
	QualityWarnings         json.RawMessage `json:"quality_warnings"`
	LineageRefs             json.RawMessage `json:"lineage_refs"`
	EvidenceDigest          string          `json:"evidence_digest"`
	IdempotencyKey          string          `json:"idempotency_key"`
	Status                  string          `json:"status"`
	CreatedAt               time.Time       `json:"created_at"`
	UpdatedAt               time.Time       `json:"updated_at"`
}

type syncraticInsightCurrentnessDTO struct {
	IsCurrent                      bool   `json:"is_current"`
	CurrentnessKey                 string `json:"currentness_key"`
	SupersededByContextWindowID    string `json:"superseded_by_context_window_id"`
	SupersededBySyncraticInsightID string `json:"superseded_by_syncratic_insight_id"`
	Reason                         string `json:"reason"`
}

type syncraticInsightDTO struct {
	SyncraticInsightID      string                         `json:"syncratic_insight_id"`
	TenantID                string                         `json:"tenant_id"`
	AppID                   string                         `json:"app_id"`
	Domain                  string                         `json:"domain"`
	UseCase                 string                         `json:"use_case"`
	ContextWindowID         string                         `json:"context_window_id"`
	InsightType             string                         `json:"insight_type"`
	SubjectType             string                         `json:"subject_type"`
	SubjectID               string                         `json:"subject_id"`
	SubjectSymbol           string                         `json:"subject_symbol"`
	Status                  string                         `json:"status"`
	Severity                string                         `json:"severity"`
	Confidence              float64                        `json:"confidence"`
	Title                   string                         `json:"title"`
	Summary                 string                         `json:"summary"`
	Explanation             string                         `json:"explanation"`
	SupportingAlertIDs      []string                       `json:"supporting_alert_ids"`
	SupportingSignalIDs     []string                       `json:"supporting_signal_ids"`
	SupportingEventIDs      []string                       `json:"supporting_event_ids"`
	SupportingArtifactIDs   []string                       `json:"supporting_artifact_ids"`
	RelatedGraphProposalIDs []string                       `json:"related_graph_proposal_ids"`
	RelatedLabelIDs         []string                       `json:"related_label_ids"`
	Metrics                 json.RawMessage                `json:"metrics"`
	Recommendation          json.RawMessage                `json:"recommendation"`
	Currentness             syncraticInsightCurrentnessDTO `json:"currentness"`
	BuilderVersion          string                         `json:"builder_version"`
	CreatedAt               time.Time                      `json:"created_at"`
	UpdatedAt               time.Time                      `json:"updated_at"`
}

func syncraticContextWindowResponse(record storage.SyncraticContextWindowRecord) syncraticContextWindowDTO {
	return syncraticContextWindowDTO{ContextWindowID: record.ContextWindowID, TenantID: record.TenantID, AppID: record.AppID, Domain: record.Domain, UseCase: record.UseCase, SubjectType: record.SubjectType, SubjectID: record.SubjectID, SubjectSymbol: record.SubjectSymbol, WindowStart: record.WindowStart, WindowEnd: record.WindowEnd, ContextStrategy: record.ContextStrategy, ContextBuilderVersion: record.ContextBuilderVersion, ContextPayloadVersion: record.ContextPayloadVersion, SignalTypes: record.SignalTypes, DetectorIDs: record.DetectorIDs, EventIDs: record.EventIDs, SignalIDs: record.SignalIDs, AlertIDs: record.AlertIDs, ArtifactIDs: record.ArtifactIDs, GraphProposalIDs: record.GraphProposalIDs, LabelIDs: record.LabelIDs, MarketStateIDs: record.MarketStateIDs, StateTransitionIDs: record.StateTransitionIDs, MarketOpsEvidenceIDs: record.MarketOpsEvidenceIDs, HypothesisEvaluationIDs: record.HypothesisEvaluationIDs, OpportunityIDs: record.OpportunityIDs, OutcomeIDs: record.OutcomeIDs, CalibrationSummaryIDs: record.CalibrationSummaryIDs, BaselineRefs: json.RawMessage(jsonOrDefault(record.BaselineRefsJSON, `[]`)), EvaluationRefs: json.RawMessage(jsonOrDefault(record.EvaluationRefsJSON, `[]`)), PromotionCandidateRefs: json.RawMessage(jsonOrDefault(record.PromotionCandidateRefsJSON, `[]`)), SummaryMetrics: json.RawMessage(jsonOrDefault(record.SummaryMetricsJSON, `{}`)), QualityWarnings: json.RawMessage(jsonOrDefault(record.QualityWarningsJSON, `[]`)), LineageRefs: json.RawMessage(jsonOrDefault(record.LineageRefsJSON, `{}`)), EvidenceDigest: record.EvidenceDigest, IdempotencyKey: record.IdempotencyKey, Status: record.Status, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}

func syncraticContextWindowResponses(records []storage.SyncraticContextWindowRecord) []syncraticContextWindowDTO {
	responses := make([]syncraticContextWindowDTO, 0, len(records))
	for _, record := range records {
		responses = append(responses, syncraticContextWindowResponse(record))
	}
	return responses
}

func syncraticInsightResponse(record storage.SyncraticInsightRecord) syncraticInsightDTO {
	return syncraticInsightResponseWithCurrentness(record, syncraticInsightCurrentnessDTO{IsCurrent: true, CurrentnessKey: syncraticCurrentnessKey(record, storage.SyncraticContextWindowRecord{}), Reason: "only_row"})
}

func syncraticInsightResponseWithCurrentness(record storage.SyncraticInsightRecord, currentness syncraticInsightCurrentnessDTO) syncraticInsightDTO {
	return syncraticInsightDTO{SyncraticInsightID: record.SyncraticInsightID, TenantID: record.TenantID, AppID: record.AppID, Domain: record.Domain, UseCase: record.UseCase, ContextWindowID: record.ContextWindowID, InsightType: record.InsightType, SubjectType: record.SubjectType, SubjectID: record.SubjectID, SubjectSymbol: record.SubjectSymbol, Status: record.Status, Severity: record.Severity, Confidence: record.Confidence, Title: record.Title, Summary: record.Summary, Explanation: record.Explanation, SupportingAlertIDs: record.SupportingAlertIDs, SupportingSignalIDs: record.SupportingSignalIDs, SupportingEventIDs: record.SupportingEventIDs, SupportingArtifactIDs: record.SupportingArtifactIDs, RelatedGraphProposalIDs: record.RelatedGraphProposalIDs, RelatedLabelIDs: record.RelatedLabelIDs, Metrics: json.RawMessage(jsonOrDefault(record.MetricsJSON, `{}`)), Recommendation: json.RawMessage(jsonOrDefault(record.RecommendationJSON, `{}`)), Currentness: currentness, BuilderVersion: record.BuilderVersion, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}

func syncraticInsightResponses(records []storage.SyncraticInsightRecord) []syncraticInsightDTO {
	return syncraticInsightResponsesWithContexts(records, nil)
}

func syncraticInsightResponsesWithContexts(records []storage.SyncraticInsightRecord, contexts map[string]storage.SyncraticContextWindowRecord) []syncraticInsightDTO {
	currentness := syncraticInsightCurrentness(records, contexts)
	responses := make([]syncraticInsightDTO, 0, len(records))
	for _, record := range records {
		responses = append(responses, syncraticInsightResponseWithCurrentness(record, currentness[record.SyncraticInsightID]))
	}
	return responses
}

func syncraticContextWindowMap(records []storage.SyncraticContextWindowRecord) map[string]storage.SyncraticContextWindowRecord {
	out := map[string]storage.SyncraticContextWindowRecord{}
	for _, record := range records {
		out[record.ContextWindowID] = record
	}
	return out
}

func syncraticCurrentnessKey(record storage.SyncraticInsightRecord, contextWindow storage.SyncraticContextWindowRecord) string {
	strategy := strings.TrimSpace(contextWindow.ContextStrategy)
	builderVersion := firstNonEmpty(strings.TrimSpace(contextWindow.ContextBuilderVersion), strings.TrimSpace(record.BuilderVersion))
	parts := []string{record.TenantID, record.AppID, record.Domain, record.UseCase, record.SubjectSymbol, strategy, builderVersion}
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return strings.Join(parts, "|")
}

func syncraticInsightCurrentness(records []storage.SyncraticInsightRecord, contexts map[string]storage.SyncraticContextWindowRecord) map[string]syncraticInsightCurrentnessDTO {
	if contexts == nil {
		contexts = map[string]storage.SyncraticContextWindowRecord{}
	}
	type candidate struct {
		record storage.SyncraticInsightRecord
		ctx    storage.SyncraticContextWindowRecord
	}
	best := map[string]candidate{}
	for _, record := range records {
		ctx := contexts[record.ContextWindowID]
		key := syncraticCurrentnessKey(record, ctx)
		if strings.TrimSpace(record.Status) != storage.SyncraticInsightStatusActive {
			continue
		}
		current, ok := best[key]
		if !ok || syncraticInsightCurrentnessAfter(record, ctx, current.record, current.ctx) {
			best[key] = candidate{record: record, ctx: ctx}
		}
	}
	out := map[string]syncraticInsightCurrentnessDTO{}
	for _, record := range records {
		ctx := contexts[record.ContextWindowID]
		key := syncraticCurrentnessKey(record, ctx)
		bestCandidate, ok := best[key]
		if !ok || bestCandidate.record.SyncraticInsightID == record.SyncraticInsightID {
			reason := "latest_window_end"
			if !ok {
				reason = "non_active_status"
			}
			out[record.SyncraticInsightID] = syncraticInsightCurrentnessDTO{IsCurrent: ok, CurrentnessKey: key, Reason: reason}
			continue
		}
		out[record.SyncraticInsightID] = syncraticInsightCurrentnessDTO{IsCurrent: false, CurrentnessKey: key, SupersededByContextWindowID: bestCandidate.record.ContextWindowID, SupersededBySyncraticInsightID: bestCandidate.record.SyncraticInsightID, Reason: "newer_context_window"}
	}
	return out
}

func syncraticInsightCurrentnessAfter(a storage.SyncraticInsightRecord, aCtx storage.SyncraticContextWindowRecord, b storage.SyncraticInsightRecord, bCtx storage.SyncraticContextWindowRecord) bool {
	if !aCtx.WindowEnd.Equal(bCtx.WindowEnd) {
		return aCtx.WindowEnd.After(bCtx.WindowEnd)
	}
	if !a.UpdatedAt.Equal(b.UpdatedAt) {
		return a.UpdatedAt.After(b.UpdatedAt)
	}
	return a.SyncraticInsightID > b.SyncraticInsightID
}

func buildSyncraticContextWindow(ctx context.Context, repo storage.QueryRepository, tenantID, subjectSymbol, strategy string, windowStart, windowEnd time.Time, builderVersion string, signalTypes []string, signalLimit, alertLimit int) (storage.SyncraticContextWindowRecord, error) {
	if strings.TrimSpace(tenantID) == "" || strings.TrimSpace(subjectSymbol) == "" || windowStart.IsZero() || windowEnd.IsZero() || !windowEnd.After(windowStart) {
		return storage.SyncraticContextWindowRecord{}, fmt.Errorf("tenant_id, subject_symbol, valid window_start, and valid window_end are required")
	}
	ledger, err := loadSyncraticContextLedger(ctx, repo, strings.TrimSpace(tenantID), windowStart, windowEnd, signalLimit, alertLimit)
	if err != nil {
		return storage.SyncraticContextWindowRecord{}, err
	}
	return buildSyncraticContextWindowWithLedger(ctx, repo, tenantID, subjectSymbol, strategy, windowStart, windowEnd, builderVersion, signalTypes, ledger)
}

func buildSyncraticContextWindowWithLedger(ctx context.Context, repo storage.QueryRepository, tenantID, subjectSymbol, strategy string, windowStart, windowEnd time.Time, builderVersion string, signalTypes []string, ledger syncraticContextLedger) (storage.SyncraticContextWindowRecord, error) {
	tenantID = strings.TrimSpace(tenantID)
	subjectSymbol = strings.ToUpper(strings.TrimSpace(subjectSymbol))
	strategy = firstNonEmpty(strings.TrimSpace(strategy), "symbol_signal_cluster_5d")
	builderVersion = firstNonEmpty(strings.TrimSpace(builderVersion), defaultSyncraticBuilderVersion)
	if tenantID == "" || subjectSymbol == "" || windowStart.IsZero() || windowEnd.IsZero() || !windowEnd.After(windowStart) {
		return storage.SyncraticContextWindowRecord{}, fmt.Errorf("tenant_id, subject_symbol, valid window_start, and valid window_end are required")
	}
	artifacts, err := repo.ListMarketOpsDSMArtifacts(ctx, storage.MarketOpsDSMArtifactFilter{TenantID: tenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SubjectSymbol: subjectSymbol, Limit: maxSyncraticArtifacts})
	if err != nil {
		return storage.SyncraticContextWindowRecord{}, err
	}
	proposals, err := repo.ListMarketOpsDSMGraphProposals(ctx, storage.MarketOpsDSMGraphProposalFilter{TenantID: tenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SubjectSymbol: subjectSymbol, Limit: maxSyncraticProposals})
	if err != nil {
		return storage.SyncraticContextWindowRecord{}, err
	}
	labels, err := repo.ListMarketOpsBacktestEvaluationLabels(ctx, storage.MarketOpsBacktestEvaluationLabelFilter{TenantID: tenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SubjectSymbol: subjectSymbol, Limit: maxSyncraticLabels})
	if err != nil {
		return storage.SyncraticContextWindowRecord{}, err
	}
	promotions, err := repo.ListMarketOpsBacktestPromotionCandidates(ctx, storage.MarketOpsBacktestPromotionCandidateFilter{TenantID: tenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", Limit: maxSyncraticArtifacts})
	if err != nil {
		return storage.SyncraticContextWindowRecord{}, err
	}
	states, err := repo.ListMarketOpsMarketStates(ctx, storage.MarketOpsMarketStateFilter{TenantID: tenantID, AppID: "marketops", Symbol: subjectSymbol, SessionStart: windowStart, SessionEnd: windowEnd, Limit: 1})
	if err != nil {
		return storage.SyncraticContextWindowRecord{}, err
	}
	transitions, err := repo.ListMarketOpsStateTransitions(ctx, storage.MarketOpsStateTransitionFilter{TenantID: tenantID, AppID: "marketops", Symbol: subjectSymbol, SessionStart: windowStart, SessionEnd: windowEnd, Limit: maxSyncraticTransitions})
	if err != nil {
		return storage.SyncraticContextWindowRecord{}, err
	}
	evidence, err := repo.ListMarketOpsEvidence(ctx, storage.MarketOpsEvidenceFilter{TenantID: tenantID, AppID: "marketops", Symbol: subjectSymbol, SessionStart: windowStart, SessionEnd: windowEnd, Limit: maxSyncraticMarketEvidence})
	if err != nil {
		return storage.SyncraticContextWindowRecord{}, err
	}
	if err != nil {
		return storage.SyncraticContextWindowRecord{}, err
	}

	allowedTypes := stringSet(signalTypes)
	signals, availableSignals := compactSyncraticSignals(subjectSymbol, ledger.Signals, windowStart, windowEnd, allowedTypes)
	alerts, availableAlerts := compactSyncraticAlerts(subjectSymbol, ledger.Alerts, windowStart, windowEnd)
	record := storage.SyncraticContextWindowRecord{TenantID: tenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SubjectType: "ticker", SubjectID: subjectSymbol, SubjectSymbol: subjectSymbol, WindowStart: windowStart.UTC(), WindowEnd: windowEnd.UTC(), ContextStrategy: strategy, ContextBuilderVersion: builderVersion, Status: "active", ContextPayloadVersion: "signalops.syncratic.market_state_session.v3"}
	detectors := map[string]struct{}{}
	signalTypeSet := map[string]struct{}{}
	eligibleSignalIDs := map[string]struct{}{}
	for _, signal := range signals {
		if !timeInWindow(signal.SignalTime, windowStart, windowEnd) || !recordEvidenceMatchesSymbol(subjectSymbol, signal.EntitiesJSON, signal.EventJSON, signal.SemanticEvidenceJSON, signal.EvidenceJSON) {
			continue
		}
		if len(allowedTypes) > 0 {
			if _, ok := allowedTypes[signal.SignalType]; !ok {
				continue
			}
		}
		record.SignalIDs = append(record.SignalIDs, signal.SignalID)
		eligibleSignalIDs[signal.SignalID] = struct{}{}
		record.EventIDs = append(record.EventIDs, signal.EventIDs...)
		signalTypeSet[signal.SignalType] = struct{}{}
		if strings.TrimSpace(signal.DetectorID) != "" {
			detectors[signal.DetectorID] = struct{}{}
		}
	}
	for _, alert := range alerts {
		if !timeInWindow(alert.LastObservedAt, windowStart, windowEnd) || !recordEvidenceMatchesSymbol(subjectSymbol, alert.EntitiesJSON, alert.EvidenceJSON) {
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
		if _, ok := eligibleSignalIDs[artifact.SignalID]; ok {
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
	record.EventIDs = limitStrings(record.EventIDs, maxSyncraticEventIDs)
	record.ArtifactIDs = uniqueSorted(record.ArtifactIDs)
	record.GraphProposalIDs = uniqueSorted(record.GraphProposalIDs)
	record.LabelIDs = uniqueSorted(record.LabelIDs)
	for _, state := range states {
		record.MarketStateIDs = append(record.MarketStateIDs, state.MarketStateID)
		if state.QualityState != "" && state.QualityState != "complete" {
			record.QualityWarningsJSON = mustJSON([]map[string]string{{"kind": "market_state_quality", "value": state.QualityState}})
		}
	}
	for _, transition := range transitions {
		record.StateTransitionIDs = append(record.StateTransitionIDs, transition.TransitionID)
	}
	for _, item := range evidence {
		record.MarketOpsEvidenceIDs = append(record.MarketOpsEvidenceIDs, item.EvidenceID)
	}
	record.MarketStateIDs = uniqueSorted(record.MarketStateIDs)
	record.StateTransitionIDs = uniqueSorted(record.StateTransitionIDs)
	record.MarketOpsEvidenceIDs = uniqueSorted(record.MarketOpsEvidenceIDs)
	stateSummary := make([]map[string]any, 0, len(states))
	for _, state := range states {
		stateSummary = append(stateSummary, map[string]any{"market_state_id": state.MarketStateID, "quality_state": state.QualityState, "completeness_ratio": state.CompletenessRatio, "eligible_hypotheses": state.EligibleHypotheses, "state_payload": json.RawMessage(jsonOrDefault(state.StatePayloadJSON, `{}`))})
	}
	eodEvidence := make([]map[string]any, 0, len(evidence))
	for _, item := range evidence {
		eodEvidence = append(eodEvidence, map[string]any{"evidence_id": item.EvidenceID, "type": item.EvidenceType, "domain": item.Domain, "direction": item.Direction, "statement": item.Statement, "magnitude": item.Magnitude, "quality_score": item.QualityScore, "payload": json.RawMessage(jsonOrDefault(item.EvidencePayloadJSON, `{}`))})
	}
	record.SignalTypes = setKeys(signalTypeSet)
	record.DetectorIDs = setKeys(detectors)
	metrics := map[string]any{"signal_count": len(record.SignalIDs), "alert_count": len(record.AlertIDs), "evidence_retention": map[string]any{"signals": map[string]int{"available": availableSignals, "included": len(record.SignalIDs), "omitted": max(0, availableSignals-len(record.SignalIDs))}, "alerts": map[string]int{"available": availableAlerts, "included": len(record.AlertIDs), "omitted": max(0, availableAlerts-len(record.AlertIDs))}}, "event_count": len(record.EventIDs), "artifact_count": len(record.ArtifactIDs), "graph_proposal_count": len(record.GraphProposalIDs), "label_count": len(record.LabelIDs), "market_state": stateSummary, "market_evidence": eodEvidence, "subject_symbol": subjectSymbol, "context_strategy": strategy}
	record.SummaryMetricsJSON = mustJSON(metrics)
	record.BaselineRefsJSON = []byte(`[]`)
	record.EvaluationRefsJSON = []byte(`[]`)
	record.PromotionCandidateRefsJSON = mustJSON(promotionRefs)
	if len(record.QualityWarningsJSON) == 0 {
		record.QualityWarningsJSON = []byte(`[]`)
	}
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

func enrichSyncraticInsightWithAsk(ctx context.Context, repo storage.QueryRepository, askClient syncraticAskClient, contextWindowID string, req syncraticAskRequest) (storage.SyncraticInsightRecord, syncraticAskResult, error) {
	if askClient == nil {
		return storage.SyncraticInsightRecord{}, syncraticAskResult{}, fmt.Errorf("syncratic ask client is not configured")
	}
	contextWindow, err := repo.GetSyncraticContextWindow(ctx, contextWindowID)
	if err != nil {
		return storage.SyncraticInsightRecord{}, syncraticAskResult{}, err
	}
	tenantID := strings.TrimSpace(req.TenantID)
	if tenantID != "" && tenantID != contextWindow.TenantID {
		return storage.SyncraticInsightRecord{}, syncraticAskResult{}, fmt.Errorf("tenant_id does not match context window")
	}
	var prompt string
	var promptMeta syncraticAskPromptMeta
	if contextWindow.ContextStrategy == marketStateContextStrategy {
		prompt, promptMeta, err = buildMarketStateAskPrompt(contextWindow, req)
	} else if isDailyNarrativeContextStrategy(contextWindow.ContextStrategy) {
		prompt, promptMeta, err = buildSyncraticDailyNarrativeAskPrompt(contextWindow, req)
	} else {
		signalDetails, missingSignalDetails, detailErr := syncraticAskSignalDetails(ctx, repo, contextWindow, 5)
		if detailErr != nil {
			return storage.SyncraticInsightRecord{}, syncraticAskResult{}, detailErr
		}
		prompt, promptMeta, err = buildSyncraticAskPrompt(contextWindow, req, signalDetails, missingSignalDetails)
	}
	if err != nil {
		return storage.SyncraticInsightRecord{}, syncraticAskResult{}, err
	}
	insightType := strings.TrimSpace(req.InsightType)
	if insightType == "" && contextWindow.ContextStrategy == "market_state_session_v2" {
		insightType = defaultSyncraticAskDrilldownType
	}
	insight, err := syncraticInsightForContextType(ctx, repo, contextWindow, insightType)
	if err != nil {
		return storage.SyncraticInsightRecord{}, syncraticAskResult{}, err
	}
	if !req.Force && syncraticAskAlreadyApplied(insight, promptMeta) {
		return insight, syncraticAskResult{ContextWindowID: contextWindow.ContextWindowID, SyncraticInsightID: insight.SyncraticInsightID, AskStatus: "skipped", PromptDigest: promptMeta.PromptDigest, Updated: false, SkippedReason: "unchanged_prompt_and_evidence", PromptBuilderVersion: promptMeta.PromptBuilderVersion}, nil
	}
	started := time.Now().UTC()
	question := "Interpret this deterministic SignalOps MarketOps context window for an operator. Rank the strongest drivers, explain why the cluster matters now, call out contradictions or weak evidence, and recommend next checks using only the caller-supplied external context."
	if contextWindow.ContextStrategy == marketStateContextStrategy {
		question = marketStateAskQuestion(contextWindow)
	} else if isDailyNarrativeContextStrategy(contextWindow.ContextStrategy) {
		question = "Return only the requested JSON daily narrative for this bounded deterministic MarketOps context. Interpret the supplied evidence for an analyst; do not describe the prompt, JSON, or instructions."
	}
	includeRefs := false
	directReasoning := false
	graphEnabled := false
	keeEnabled := false
	askResp, err := askClient.Ask(ctx, userapi.AskRequest{
		Question:        question,
		Scope:           defaultSyncraticAskScope,
		K:               1,
		ThreadMode:      "off",
		IncludeRefs:     &includeRefs,
		DirectReasoning: &directReasoning,
		GraphEnabled:    &graphEnabled,
		KEEEnabled:      &keeEnabled,
		IdempotencyKey:  syncraticAskIdempotencyKey(contextWindow, promptMeta, req.Force, started),
		ExternalContext: &userapi.AskExternalContext{Items: []userapi.AskExternalContextItem{{Title: "SignalOps MarketOps context window " + contextWindow.ContextWindowID, SourceID: contextWindow.ContextWindowID, Text: prompt}}},
	})
	if err != nil {
		return storage.SyncraticInsightRecord{}, syncraticAskResult{}, fmt.Errorf("syncratic ask failed: %w", err)
	}
	completed := time.Now().UTC()
	updated := applySyncraticAskResponse(insight, contextWindow, promptMeta, askResp, started, completed)
	if err := repo.UpsertSyncraticInsight(ctx, updated); err != nil {
		return storage.SyncraticInsightRecord{}, syncraticAskResult{}, err
	}
	stored, err := repo.GetSyncraticInsight(ctx, updated.SyncraticInsightID)
	if err != nil {
		return storage.SyncraticInsightRecord{}, syncraticAskResult{}, err
	}
	return stored, syncraticAskResult{ContextWindowID: contextWindow.ContextWindowID, SyncraticInsightID: stored.SyncraticInsightID, AskQueryID: askResp.QueryID, AskStatus: "completed", PromptDigest: promptMeta.PromptDigest, Updated: true, PromptBuilderVersion: promptMeta.PromptBuilderVersion}, nil
}

func syncraticAskIdempotencyKey(contextWindow storage.SyncraticContextWindowRecord, meta syncraticAskPromptMeta, force bool, started time.Time) string {
	parts := []string{"signalops", "syncratic-ask", contextWindow.ContextWindowID, strings.TrimPrefix(meta.PromptDigest, "sha256:")}
	if force {
		parts = append(parts, "force", started.Format("20060102T150405.000000000Z"))
	}
	return strings.Join(parts, "-")
}

type syncraticAskPromptMeta struct {
	PromptBuilderVersion  string
	PromptDigest          string
	ContextEvidenceDigest string
	MaxPromptBytes        int
	IncludedRecordDetails bool
	Caps                  map[string]int
	PromptBytes           int
}

type syncraticAskSignalDetail struct {
	SignalID             string          `json:"signal_id"`
	SignalType           string          `json:"signal_type"`
	DetectorID           string          `json:"detector_id"`
	DetectorVersion      string          `json:"detector_version,omitempty"`
	SignalTime           string          `json:"signal_time"`
	WindowStart          string          `json:"window_start,omitempty"`
	WindowEnd            string          `json:"window_end,omitempty"`
	Severity             string          `json:"severity"`
	Confidence           float64         `json:"confidence"`
	EventIDs             []string        `json:"event_ids,omitempty"`
	ArtifactIDs          []string        `json:"artifact_ids,omitempty"`
	Entities             json.RawMessage `json:"entities,omitempty"`
	SupportingMetrics    json.RawMessage `json:"supporting_metrics,omitempty"`
	EvidenceSummaries    []string        `json:"evidence_summaries,omitempty"`
	SubjectMismatchHints []string        `json:"subject_mismatch_hints,omitempty"`
}

func syncraticAskSignalDetails(ctx context.Context, repo storage.QueryRepository, contextWindow storage.SyncraticContextWindowRecord, maxDetails int) ([]syncraticAskSignalDetail, []string, error) {
	if maxDetails <= 0 || len(contextWindow.SignalIDs) == 0 {
		return nil, nil, nil
	}
	details := []syncraticAskSignalDetail{}
	missing := []string{}
	for _, signalID := range limitStrings(contextWindow.SignalIDs, maxDetails) {
		record, err := repo.GetSignalLedger(ctx, signalID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				missing = append(missing, signalID)
				continue
			}
			return nil, nil, err
		}
		details = append(details, syncraticAskSignalDetail{
			SignalID:             record.SignalID,
			SignalType:           record.SignalType,
			DetectorID:           record.DetectorID,
			DetectorVersion:      record.DetectorVersion,
			SignalTime:           record.SignalTime.UTC().Format(time.RFC3339),
			WindowStart:          optionalTime(record.WindowStart),
			WindowEnd:            optionalTime(record.WindowEnd),
			Severity:             record.Severity,
			Confidence:           record.Confidence,
			EventIDs:             limitStrings(record.EventIDs, 5),
			ArtifactIDs:          limitStrings(record.ArtifactIDs, 5),
			Entities:             validJSONRaw(record.EntitiesJSON),
			SupportingMetrics:    validJSONRaw(record.SupportingMetrics),
			EvidenceSummaries:    compactEvidenceSummaries(record.EvidenceJSON, 2),
			SubjectMismatchHints: signalSubjectMismatchHints(contextWindow.SubjectSymbol, record),
		})
	}
	return details, missing, nil
}

func optionalTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func validJSONRaw(raw []byte) json.RawMessage {
	if len(raw) == 0 || !json.Valid(raw) {
		return nil
	}
	return json.RawMessage(raw)
}

func syncraticAskAnalysisMode(details []syncraticAskSignalDetail) string {
	for _, detail := range details {
		if len(detail.SubjectMismatchHints) > 0 {
			return "data_quality_blocked"
		}
	}
	return "market_interpretation_allowed"
}

func syncraticAskDataQualityChecks(details []syncraticAskSignalDetail, missing []string) map[string]any {
	mismatchSignals := []map[string]any{}
	for _, detail := range details {
		if len(detail.SubjectMismatchHints) > 0 {
			mismatchSignals = append(mismatchSignals, map[string]any{"signal_id": detail.SignalID, "hints": detail.SubjectMismatchHints})
		}
	}
	return map[string]any{"subject_mismatch_count": len(mismatchSignals), "subject_mismatch_signals": mismatchSignals, "missing_signal_detail_count": len(missing)}
}

func signalSubjectMismatchHints(subject string, record storage.SignalLedgerRecord) []string {
	subject = strings.ToUpper(strings.TrimSpace(subject))
	if subject == "" {
		return nil
	}
	symbols := map[string]struct{}{}
	collectJSONSymbols(record.EntitiesJSON, symbols)
	collectJSONSymbols(record.EvidenceJSON, symbols)
	collectJSONSymbols(record.SemanticEvidenceJSON, symbols)
	hints := []string{}
	for _, candidate := range knownMarketOpsSymbols() {
		if candidate != subject && jsonTextMentionsSymbol(record.EvidenceJSON, candidate) {
			symbols[candidate] = struct{}{}
		}
	}
	for symbol := range symbols {
		if symbol != "" && symbol != subject {
			hints = append(hints, fmt.Sprintf("context subject is %s but signal/evidence mentions %s", subject, symbol))
		}
	}
	sort.Strings(hints)
	return hints
}

func compactEvidenceSummaries(raw []byte, limit int) []string {
	if limit <= 0 || len(raw) == 0 || !json.Valid(raw) {
		return nil
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil
	}
	summaries := []string{}
	collectEvidenceSummaries(value, &summaries, limit)
	return summaries
}

func collectEvidenceSummaries(value any, summaries *[]string, limit int) {
	if len(*summaries) >= limit {
		return
	}
	switch typed := value.(type) {
	case map[string]any:
		if summary, ok := typed["summary"].(string); ok {
			if text := strings.TrimSpace(summary); text != "" {
				*summaries = append(*summaries, truncateText(text, 180))
			}
		}
		for _, child := range typed {
			collectEvidenceSummaries(child, summaries, limit)
			if len(*summaries) >= limit {
				return
			}
		}
	case []any:
		for _, child := range typed {
			collectEvidenceSummaries(child, summaries, limit)
			if len(*summaries) >= limit {
				return
			}
		}
	}
}

func truncateText(text string, maxLen int) string {
	if maxLen <= 0 || len(text) <= maxLen {
		return text
	}
	return strings.TrimSpace(text[:maxLen])
}

func jsonTextMentionsSymbol(raw []byte, symbol string) bool {
	if len(raw) == 0 || symbol == "" {
		return false
	}
	return textContainsSymbolToken(string(raw), symbol)
}

func collectJSONSymbols(raw []byte, out map[string]struct{}) {
	if len(raw) == 0 || !json.Valid(raw) {
		return
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return
	}
	collectSymbols(value, out)
}

func collectSymbols(value any, out map[string]struct{}) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			lower := strings.ToLower(key)
			if lower == "symbol" || lower == "ticker" || lower == "subject_symbol" {
				if text, ok := child.(string); ok {
					if symbol := strings.ToUpper(strings.TrimSpace(text)); symbol != "" {
						out[symbol] = struct{}{}
					}
				}
			}
			collectSymbols(child, out)
		}
	case []any:
		for _, child := range typed {
			collectSymbols(child, out)
		}
	}
}

func buildSyncraticAskPrompt(contextWindow storage.SyncraticContextWindowRecord, req syncraticAskRequest, signalDetails []syncraticAskSignalDetail, missingSignalDetails []string) (string, syncraticAskPromptMeta, error) {
	version := firstNonEmpty(strings.TrimSpace(req.PromptBuilderVersion), defaultSyncraticAskPromptVersion)
	maxPromptBytes := req.MaxPromptBytes
	if maxPromptBytes <= 0 {
		maxPromptBytes = 12000
	}
	if maxPromptBytes < 1000 {
		return "", syncraticAskPromptMeta{}, fmt.Errorf("max_prompt_bytes must be at least 1000")
	}
	if maxPromptBytes > 24000 {
		maxPromptBytes = 24000
	}
	caps := map[string]int{"max_alert_ids": 20, "max_signal_ids": 20, "max_signal_details": 5, "max_event_ids": 20, "max_artifact_ids": 20, "max_graph_proposal_ids": 20, "max_label_ids": 20, "max_prompt_bytes": maxPromptBytes}
	payload := map[string]any{
		"prompt_builder_version": version,
		"role":                   "MarketOps surveillance reasoning layer over deterministic SignalOps evidence.",
		"instructions": []string{
			"Use only the supplied JSON context; do not retrieve documents or use external knowledge.",
			"Write analyst-facing narrative in relational natural language. Do not describe what the user wants, what the task is, what the prompt says, or what the context contains.",
			"Do not restate counts or IDs as the main explanation. Interpret the strongest evidence.",
			"Rank top drivers only when analysis_mode is market_interpretation_allowed.",
			"Explain why the combined signal cluster matters for the operator now.",
			"If analysis_mode is data_quality_blocked, do not provide market top-driver interpretation; explain only why evidence cannot support the context subject and recommend evidence rematerialization or mapping review.",
			"Call out contradictions, weak evidence, missing details, or subject-symbol mismatches.",
			"Tie every claim to cited signal_ids, event_ids, metrics, or evidence summaries from the context.",
			"If evidence is too thin, say so specifically instead of giving generic market commentary.",
		},
		"context_metadata":    map[string]any{"tenant_id": contextWindow.TenantID, "app_id": contextWindow.AppID, "domain": contextWindow.Domain, "use_case": contextWindow.UseCase, "context_window_id": contextWindow.ContextWindowID, "subject_symbol": contextWindow.SubjectSymbol, "subject_type": contextWindow.SubjectType, "subject_id": contextWindow.SubjectID, "window_start": contextWindow.WindowStart.UTC().Format(time.RFC3339), "window_end": contextWindow.WindowEnd.UTC().Format(time.RFC3339), "context_strategy": contextWindow.ContextStrategy, "context_builder_version": contextWindow.ContextBuilderVersion, "evidence_digest": contextWindow.EvidenceDigest},
		"evidence_summary":    map[string]any{"signal_types": contextWindow.SignalTypes, "detector_ids": contextWindow.DetectorIDs, "summary_metrics": json.RawMessage(jsonOrDefault(contextWindow.SummaryMetricsJSON, `{}`)), "baseline_refs": json.RawMessage(jsonOrDefault(contextWindow.BaselineRefsJSON, `[]`)), "evaluation_refs": json.RawMessage(jsonOrDefault(contextWindow.EvaluationRefsJSON, `[]`)), "promotion_candidate_refs": json.RawMessage(jsonOrDefault(contextWindow.PromotionCandidateRefsJSON, `[]`))},
		"evidence_ids":        map[string]any{"market_states": limitStrings(contextWindow.MarketStateIDs, 5), "state_transitions": limitStrings(contextWindow.StateTransitionIDs, 20), "market_evidence": limitStrings(contextWindow.MarketOpsEvidenceIDs, 20), "alerts": limitStrings(contextWindow.AlertIDs, caps["max_alert_ids"]), "signals": limitStrings(contextWindow.SignalIDs, caps["max_signal_ids"]), "events": limitStrings(contextWindow.EventIDs, caps["max_event_ids"]), "artifacts": limitStrings(contextWindow.ArtifactIDs, caps["max_artifact_ids"]), "graph_proposals": limitStrings(contextWindow.GraphProposalIDs, caps["max_graph_proposal_ids"]), "labels": limitStrings(contextWindow.LabelIDs, caps["max_label_ids"])},
		"evidence_details":    map[string]any{"signals": signalDetails, "missing_signal_detail_ids": missingSignalDetails, "omitted_signal_detail_count": max(0, len(contextWindow.SignalIDs)-caps["max_signal_details"])},
		"data_quality_checks": syncraticAskDataQualityChecks(signalDetails, missingSignalDetails),
		"output_contract": []string{
			"title: concise finding, not the context_window_id",
			"summary: one sentence stating the operator-relevant pattern",
			"top_drivers: if analysis_mode=data_quality_blocked or evidence is empty, write none; otherwise ranked relational bullets explaining strongest drivers in natural language with cited evidence IDs",
			"explanation: synthesize the cluster; do not merely list signal names",
			"quality_warnings: put first when subject_mismatch_hints exist; state that affected evidence cannot support the context subject",
			"recommendation: action one of observe, review, escalate, no_action plus next checks",
			"uncertainty_notes and cited_evidence_ids",
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", syncraticAskPromptMeta{}, err
	}
	prompt := "You are a non-human MarketOps reasoning client. Produce operator-useful interpretation, not a generic summary. Do not describe the request, task, prompt, JSON, external context, or what the user wants. If analysis_mode is data_quality_blocked, lead with DATA QUALITY WARNING, do not infer that one ticker impacts another, do not describe market impact, and make the recommendation about evidence remediation rather than trading/market action. Use only the JSON context below. Return the requested fields and cite evidence IDs.\nCONTEXT_JSON:\n" + string(raw)
	if len(prompt) > maxPromptBytes {
		return "", syncraticAskPromptMeta{}, fmt.Errorf("prompt exceeds max_prompt_bytes")
	}
	sum := sha256.Sum256([]byte(prompt))
	meta := syncraticAskPromptMeta{PromptBuilderVersion: version, PromptDigest: "sha256:" + hex.EncodeToString(sum[:]), ContextEvidenceDigest: contextWindow.EvidenceDigest, MaxPromptBytes: maxPromptBytes, IncludedRecordDetails: req.IncludeRecordDetails, Caps: caps, PromptBytes: len(prompt)}
	return prompt, meta, nil
}

func syncraticInsightForContext(ctx context.Context, repo storage.QueryRepository, contextWindow storage.SyncraticContextWindowRecord) (storage.SyncraticInsightRecord, error) {
	return syncraticInsightForContextType(ctx, repo, contextWindow, defaultSyncraticInsightType)
}

func syncraticInsightForContextType(ctx context.Context, repo storage.QueryRepository, contextWindow storage.SyncraticContextWindowRecord, insightType string) (storage.SyncraticInsightRecord, error) {
	insightType = firstNonEmpty(strings.TrimSpace(insightType), defaultSyncraticInsightType)
	records, err := repo.ListSyncraticInsights(ctx, storage.SyncraticInsightFilter{TenantID: contextWindow.TenantID, ContextWindowID: contextWindow.ContextWindowID, InsightType: insightType, Limit: 10})
	if err != nil {
		return storage.SyncraticInsightRecord{}, err
	}
	if len(records) > 0 {
		return records[0], nil
	}
	insight := buildSyncraticInsight(contextWindow, insightType, contextWindow.ContextBuilderVersion)
	if err := repo.UpsertSyncraticInsight(ctx, insight); err != nil {
		return storage.SyncraticInsightRecord{}, err
	}
	return insight, nil
}

func syncraticAskAlreadyApplied(insight storage.SyncraticInsightRecord, meta syncraticAskPromptMeta) bool {
	if syncraticAskAnswerIsMetaCommentary(strings.Join([]string{insight.Title, insight.Summary, insight.Explanation}, " ")) {
		return false
	}
	metrics := jsonObjectOrEmpty(insight.MetricsJSON)
	ask, ok := metrics["syncratic_ask"].(map[string]any)
	if !ok {
		return false
	}
	return asString(ask["ask_status"]) == "completed" && asString(ask["prompt_digest"]) == meta.PromptDigest && asString(ask["context_evidence_digest"]) == meta.ContextEvidenceDigest
}

func applySyncraticAskResponse(insight storage.SyncraticInsightRecord, contextWindow storage.SyncraticContextWindowRecord, meta syncraticAskPromptMeta, resp userapi.AskResponse, started, completed time.Time) storage.SyncraticInsightRecord {
	structured := syncraticAskStructuredPayload(resp)
	answer := strings.TrimSpace(resp.Answer)
	if rendered := renderSyncraticAskStructuredExplanation(structured); rendered != "" {
		answer = rendered
	} else if answer == "" {
		answer = firstNonEmpty(asString(structured["answer"]), asString(structured["explanation"]), asString(structured["executive_summary"]), asString(structured["summary"]))
	}
	if answer == "" {
		answer = "Syncratic Ask returned no textual explanation. Review deterministic evidence directly."
	}
	responseQuality := "llm_answer"
	summary := firstNonEmpty(asString(structured["executive_summary"]), asString(structured["summary"]), extractAskString(resp.Raw, "summary"), truncateForSummary(answer), insight.Summary)
	title := firstNonEmpty(asString(structured["title"]), extractAskString(resp.Raw, "title"), insight.Title, fmt.Sprintf("%s Syncratic Ask explanation", contextWindow.SubjectSymbol))
	action := firstNonEmpty(asString(structured["recommended_next_step"]), asString(structured["action"]), extractAskString(resp.Raw, "action"), "review")
	if isDailyNarrativeContextStrategy(contextWindow.ContextStrategy) && syncraticAskOutputNeedsDeterministicFallback(answer, summary, title, structured) {
		fallback := deterministicDailyNarrativeFromContext(contextWindow)
		if fallback.Explanation != "" {
			answer = fallback.Explanation
			summary = firstNonEmpty(fallback.Summary, summary)
			title = firstNonEmpty(fallback.Title, title)
			action = firstNonEmpty(fallback.Action, action)
			responseQuality = "deterministic_fallback_meta_answer"
		}
	} else if !isDailyNarrativeContextStrategy(contextWindow.ContextStrategy) && syncraticAskGenericOutputNeedsDeterministicFallback(answer, summary, title, contextWindow) {
		fallback := deterministicGenericSyncraticNarrativeFromContext(contextWindow)
		if fallback.Explanation != "" {
			answer = fallback.Explanation
			summary = firstNonEmpty(fallback.Summary, summary)
			title = firstNonEmpty(fallback.Title, title)
			action = firstNonEmpty(fallback.Action, action)
			responseQuality = fallback.ResponseQuality
		}
	}
	insight.Explanation = answer
	insight.Summary = summary
	insight.Title = title
	if resp.Confidence > 0 {
		insight.Confidence = float64(resp.Confidence)
	}
	metrics := jsonObjectOrEmpty(insight.MetricsJSON)
	metrics["syncratic_ask"] = map[string]any{"enabled": true, "ask_query_id": resp.QueryID, "ask_status": "completed", "prompt_builder_version": meta.PromptBuilderVersion, "prompt_digest": meta.PromptDigest, "context_window_id": contextWindow.ContextWindowID, "context_evidence_digest": meta.ContextEvidenceDigest, "request_scope": defaultSyncraticAskScope, "request_k": 1, "direct_reasoning": false, "graph_enabled": false, "kee_enabled": false, "included_record_details": meta.IncludedRecordDetails, "prompt_bytes": meta.PromptBytes, "caps": meta.Caps, "response_quality": responseQuality, "response": map[string]any{"confidence": resp.Confidence, "evidence_count": resp.EvidenceCount, "citation_count": len(resp.Citations)}, "started_at": started.Format(time.RFC3339Nano), "completed_at": completed.Format(time.RFC3339Nano), "latency_ms": completed.Sub(started).Milliseconds()}
	insight.MetricsJSON = mustJSON(metrics)
	insight.RecommendationJSON = mustJSON(map[string]any{"action": action, "source": "syncratic_ask", "reason": "LLM-generated explanation over a bounded deterministic SignalOps context window", "ask_query_id": resp.QueryID, "prompt_digest": meta.PromptDigest})
	return insight
}

type deterministicSyncraticNarrative struct {
	Title           string
	Summary         string
	Explanation     string
	Action          string
	ResponseQuality string
}

func syncraticAskGenericOutputNeedsDeterministicFallback(answer, summary, title string, contextWindow storage.SyncraticContextWindowRecord) bool {
	if syncraticAskAnswerIsMetaCommentary(strings.Join([]string{answer, summary, title}, " ")) {
		return true
	}
	if syncraticContextEvidenceCount(contextWindow) == 0 {
		return true
	}
	trimmedSummary := strings.TrimSpace(summary)
	if trimmedSummary == "UNKNOWN" || strings.HasPrefix(trimmedSummary, "{") || strings.HasPrefix(trimmedSummary, "```") {
		return true
	}
	return false
}

func syncraticAskOutputNeedsDeterministicFallback(answer, summary, title string, structured map[string]any) bool {
	trimmed := strings.TrimSpace(answer)
	if syncraticAskAnswerIsMetaCommentary(trimmed) {
		return true
	}
	if len(structured) == 0 && (strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "```")) {
		return true
	}
	summaryText := strings.TrimSpace(summary)
	titleText := strings.TrimSpace(title)
	if summaryText == "UNKNOWN" || strings.HasPrefix(summaryText, "{") || strings.HasPrefix(summaryText, "```") {
		return true
	}
	if titleText == "MARKETOPS Syncratic context" {
		return true
	}
	return false
}

func syncraticAskAnswerIsMetaCommentary(answer string) bool {
	text := strings.ToLower(strings.TrimSpace(answer))
	if text == "" {
		return false
	}
	markers := []string{"the prompt", "the json", "json provided", "json from external context", "the user specified", "they specified", "the instructions", "context includes a json", "main artifact here is the json", "main goal is to generate", "they want me to", "the task is to", "the context includes", "provided external context", "caller-supplied external context"}
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func deterministicGenericSyncraticNarrativeFromContext(contextWindow storage.SyncraticContextWindowRecord) deterministicSyncraticNarrative {
	symbol := firstNonEmpty(strings.TrimSpace(contextWindow.SubjectSymbol), strings.TrimSpace(contextWindow.SubjectID), "the selected asset")
	session := contextWindow.WindowStart.UTC().Format("2006-01-02")
	evidenceCount := syncraticContextEvidenceCount(contextWindow)
	if evidenceCount == 0 {
		summary := fmt.Sprintf("%s does not have enough persisted Syncratic evidence in this context window for an analyst-facing market interpretation.", symbol)
		return deterministicSyncraticNarrative{
			Title:           fmt.Sprintf("%s Syncratic evidence gap", symbol),
			Summary:         summary,
			Explanation:     "DATA QUALITY WARNING:\nSession date: " + session + ". " + summary + "\n\nContextual read:\n- The context window contains no supporting signals, alerts, events, artifacts, market-state evidence, opportunities, or outcomes. Because the evidence set is empty, Syncratic Ask cannot responsibly identify strongest drivers, explain a cluster, or infer opportunity posture for this asset.\n\nWeak evidence:\n- The prior response described the request structure rather than persisted MarketOps evidence. That is not a valid analyst narrative.\n\nAnalyst follow-ups:\n- Rematerialize the Syncratic context after Market State, Risk/Reward, Review Queue, or signal evidence exists for this asset.\n- Review asset-to-evidence mapping if the asset should already have current persisted MarketOps evidence.",
			Action:          "rematerialize_or_review_mapping",
			ResponseQuality: "deterministic_fallback_empty_context",
		}
	}
	parts := []string{}
	if len(contextWindow.SignalIDs) > 0 {
		parts = append(parts, fmt.Sprintf("%d signal reference(s)", len(contextWindow.SignalIDs)))
	}
	if len(contextWindow.AlertIDs) > 0 {
		parts = append(parts, fmt.Sprintf("%d alert reference(s)", len(contextWindow.AlertIDs)))
	}
	if len(contextWindow.MarketStateIDs)+len(contextWindow.MarketOpsEvidenceIDs) > 0 {
		parts = append(parts, "market-state evidence")
	}
	if len(contextWindow.OpportunityIDs) > 0 {
		parts = append(parts, "Review Queue opportunity evidence")
	}
	if len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%d persisted evidence reference(s)", evidenceCount))
	}
	summary := fmt.Sprintf("%s has a bounded Syncratic context with %s, but the AI response was not usable as analyst prose.", symbol, strings.Join(parts, ", "))
	return deterministicSyncraticNarrative{
		Title:           fmt.Sprintf("%s Syncratic evidence review", symbol),
		Summary:         summary,
		Explanation:     "Executive summary:\nSession date: " + session + ". " + summary + "\n\nContextual read:\n- The available context is sufficient for evidence review, but the generated response described the prompt/request instead of interpreting persisted MarketOps evidence. SignalOps therefore preserved a deterministic evidence-first fallback.\n\nAnalyst follow-ups:\n- Inspect the cited context window and underlying evidence references before using this as an asset-level thesis.\n- Regenerate after richer Market State, Risk/Reward, Review Queue, or signal evidence is available if a deeper narrative is needed.",
		Action:          "review_context_evidence",
		ResponseQuality: "deterministic_fallback_meta_answer",
	}
}

func syncraticContextEvidenceCount(contextWindow storage.SyncraticContextWindowRecord) int {
	return len(contextWindow.MarketStateIDs) +
		len(contextWindow.StateTransitionIDs) +
		len(contextWindow.MarketOpsEvidenceIDs) +
		len(contextWindow.HypothesisEvaluationIDs) +
		len(contextWindow.OpportunityIDs) +
		len(contextWindow.OutcomeIDs) +
		len(contextWindow.CalibrationSummaryIDs) +
		len(contextWindow.AlertIDs) +
		len(contextWindow.SignalIDs) +
		len(contextWindow.EventIDs) +
		len(contextWindow.ArtifactIDs) +
		len(contextWindow.GraphProposalIDs) +
		len(contextWindow.LabelIDs)
}

func deterministicDailyNarrativeFromContext(contextWindow storage.SyncraticContextWindowRecord) deterministicSyncraticNarrative {
	root := jsonObjectOrEmpty(contextWindow.SummaryMetricsJSON)
	sections := mapFromAny(root["sections"])
	session := contextWindow.WindowStart.UTC().Format("2006-01-02")
	switch contextWindow.ContextStrategy {
	case dailyNarrativeStrategySRI:
		return deterministicSRINarrative(session, mapFromAny(sections["sri"]))
	case dailyNarrativeStrategyRiskReward:
		return deterministicRiskRewardNarrative(session, mapFromAny(sections["risk_reward"]))
	case dailyNarrativeStrategyReviewQueue:
		return deterministicReviewQueueNarrative(session, mapFromAny(sections["review_queue"]))
	case dailyNarrativeStrategyOverview:
		return deterministicOverviewNarrative(session, sections)
	default:
		return deterministicSyncraticNarrative{}
	}
}

func deterministicOverviewNarrative(session string, sections map[string]any) deterministicSyncraticNarrative {
	sri := deterministicSRINarrative(session, mapFromAny(sections["sri"]))
	rr := deterministicRiskRewardNarrative(session, mapFromAny(sections["risk_reward"]))
	review := deterministicReviewQueueNarrative(session, mapFromAny(sections["review_queue"]))
	parts := []string{}
	for _, item := range []deterministicSyncraticNarrative{sri, rr, review} {
		if item.Summary != "" {
			parts = append(parts, item.Summary)
		}
	}
	if len(parts) == 0 {
		return deterministicSyncraticNarrative{}
	}
	return deterministicSyncraticNarrative{
		Title:   "MarketOps daily overview · " + session,
		Summary: strings.Join(parts, " "),
		Explanation: "Executive summary:\nSession date: " + session + ". " + strings.Join(parts, " ") +
			"\n\nContextual read:\n- Sector leadership, Risk/Reward breadth, and active Review Queue pressure should be read together. A strong sector tape with mostly neutral Risk/Reward breadth is a focus signal for drill-down, not a broad confirmation signal." +
			"\n\nWhat changed:\n- " + strings.Join(parts, "\n- ") +
			"\n\nContradictions or weak evidence:\n- Overview synthesis is compacted from focused section summaries. Use the focused SRI, Risk/Reward, and Review Queue cards for artifact-level validation before escalating a hypothesis." +
			"\n\nAnalyst follow-ups:\n- Inspect focused narratives for the concrete symbols, sectors, scores, and stale-evidence warnings behind this overview.\n- Treat this as explainability over persisted evidence, not a trading instruction.",
		Action: "review_focused_narratives",
	}
}

func deterministicSRINarrative(session string, section map[string]any) deterministicSyncraticNarrative {
	leaders := sliceFromAny(section["leaders"])
	if len(leaders) == 0 {
		return deterministicSyncraticNarrative{}
	}
	leaderTexts := []string{}
	for idx, item := range leaders {
		m := mapFromAny(item)
		leaderTexts = append(leaderTexts, sriLeaderNarrative(idx, m, len(leaders)))
		if len(leaderTexts) >= 5 {
			break
		}
	}
	contextBullets := []string{}
	if len(leaders) > 0 {
		top := mapFromAny(leaders[0])
		contextBullets = append(contextBullets, fmt.Sprintf("%s is the primary leadership pocket in the sampled rotation table and remains in a %s posture", humanizeSegment(asString(top["segment_id"])), naturalState(asString(top["state"]))))
	}
	if len(leaders) >= 2 {
		tail := mapFromAny(leaders[len(leaders)-1])
		contextBullets = append(contextBullets, fmt.Sprintf("%s is the weakest sampled pocket and should be treated as a laggard until its rotation posture improves", humanizeSegment(asString(tail["segment_id"]))))
	}
	summary := "Sector Rotation leadership was concentrated in " + strings.Join(topSegments(leaders, 3), ", ") + "."
	return deterministicSyncraticNarrative{Title: "Sector Rotation daily overview · " + session, Summary: summary, Explanation: "Executive summary:\nSession date: " + session + ". " + summary + " The section reads as a leadership map rather than a market-wide recommendation.\n\nContextual read:\n- " + strings.Join(contextBullets, "\n- ") + "\n\nTop drivers:\n- " + strings.Join(leaderTexts, "\n- ") + "\n\nContradictions or weak evidence:\n- Rank-change fields may be unavailable for some segments; treat missing rank-change as a data-quality limitation, not neutral evidence.\n\nAnalyst follow-ups:\n- Inspect the SRI progression chart for whether leadership is persistent or a one-session move.\n- Compare leading and lagging segments against watchlist holdings before treating rotation as actionable.", Action: "review_sri_progression"}
}

func deterministicRiskRewardNarrative(session string, section map[string]any) deterministicSyncraticNarrative {
	breadth := mapFromAny(section["breadth"])
	examples := sliceFromAny(section["top_examples"])
	bullish := intFromAny(breadth["bullish"])
	bearish := intFromAny(breadth["bearish"])
	neutral := intFromAny(breadth["neutral"])
	unavailable := intFromAny(breadth["unavailable"])
	total := bullish + bearish + neutral + unavailable
	breadthText := naturalBreadthText(bullish, bearish, neutral, unavailable)
	drivers := []string{}
	for idx, item := range examples {
		m := mapFromAny(item)
		drivers = append(drivers, riskRewardDriverNarrative(idx, m))
		if len(drivers) >= 6 {
			break
		}
	}
	driverSummary := "No representative directional exceptions were available in the bounded sample."
	if len(drivers) > 0 {
		driverSummary = "The strongest sampled exceptions were " + strings.Join(drivers[:minInt(len(drivers), 3)], "; ") + "."
	}
	summary := breadthText + " " + driverSummary
	neutralContext := naturalNeutralContext(neutral, total)
	imbalance := naturalDirectionalImbalance(bullish, bearish)
	return deterministicSyncraticNarrative{Title: "Risk/Reward daily evolution · " + session, Summary: summary, Explanation: "Executive summary:\nSession date: " + session + ". " + summary + "\n\nContextual read:\n- " + neutralContext + " Directional breadth showed " + imbalance + ", so the named symbols should be treated as focused exceptions inside the broader monitored group.\n\nWhat changed:\n- " + breadthText + "\n\nTop drivers:\n- " + strings.Join(drivers, "\n- ") + "\n\nContradictions or weak evidence:\n- The breadth mix is still mostly neutral, so directional examples should be reviewed as a focused subset rather than a broad market call.\n\nAnalyst follow-ups:\n- Compare these symbols against Market State and Review Queue convergence before promoting any hypothesis.", Action: "compare_risk_reward_with_market_state"}
}

func deterministicReviewQueueNarrative(session string, section map[string]any) deterministicSyncraticNarrative {
	counts := mapFromAny(section["status_counts"])
	examples := sliceFromAny(section["active_examples"])
	activeCount := intFromAny(counts["active"])
	expiredCount := intFromAny(counts["expired"])
	totalCount := activeCount + expiredCount
	countText := naturalReviewQueueText(activeCount, expiredCount)
	drivers := []string{}
	for _, item := range examples {
		m := mapFromAny(item)
		drivers = append(drivers, reviewQueueDriverNarrative(m))
		if len(drivers) >= 6 {
			break
		}
	}
	driverSummary := "No active examples were available for current triage."
	if len(drivers) > 0 {
		driverSummary = "Active examples were led by " + strings.Join(drivers[:minInt(len(drivers), 3)], "; ") + "."
	}
	activeShare := naturalActiveShare(activeCount, totalCount)
	summary := countText + " " + driverSummary
	return deterministicSyncraticNarrative{Title: "Review Queue daily brief · " + session, Summary: summary, Explanation: "Executive summary:\nSession date: " + session + ". " + summary + "\n\nContextual read:\n- " + activeShare + ", so analyst attention should stay on live items and keep expired rows out of primary triage.\n\nWhat changed:\n- " + countText + "; expired items should remain separated from current analyst work.\n\nTop drivers:\n- " + strings.Join(drivers, "\n- ") + "\n\nContradictions or weak evidence:\n- High expired volume can overwhelm the view; focus first on active opportunities with current evaluation dates.\n\nAnalyst follow-ups:\n- Inspect active opportunity details and suppress expired rows from primary triage unless reviewing historical performance.", Action: "review_active_opportunities"}
}

func topSegments(items []any, limit int) []string {
	out := []string{}
	for _, item := range items {
		segment := humanizeSegment(asString(mapFromAny(item)["segment_id"]))
		if segment != "" {
			out = append(out, segment)
		}
		if len(out) >= limit {
			break
		}
	}
	return out
}

func sriLeaderNarrative(index int, item map[string]any, total int) string {
	segment := humanizeSegment(asString(item["segment_id"]))
	state := naturalState(asString(item["state"]))
	switch {
	case index == 0:
		return fmt.Sprintf("%s is leading the monitored rotation group and currently has the clearest leadership posture", segment)
	case index <= 2:
		return fmt.Sprintf("%s remains near the top of the rotation stack with a %s posture", segment, state)
	case index >= total-2:
		return fmt.Sprintf("%s is still in the weaker part of the rotation stack and needs confirmation before it can be treated as improving", segment)
	default:
		return fmt.Sprintf("%s is in the middle of the rotation stack, which makes it more of a monitoring candidate than a leadership signal", segment)
	}
}

func naturalState(state string) string {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case "LEADING":
		return "leading"
	case "IMPROVING":
		return "improving"
	case "WEAKENING":
		return "weakening"
	case "LAGGING":
		return "lagging"
	case "NEUTRAL":
		return "neutral"
	default:
		return "unclassified"
	}
}

func naturalBreadthText(bullish, bearish, neutral, unavailable int) string {
	switch {
	case neutral > bullish+bearish:
		return "Risk/Reward breadth remains mostly neutral across the monitored group."
	case bullish > bearish:
		return "Risk/Reward breadth leans constructive, with bullish reads outnumbering bearish reads."
	case bearish > bullish:
		return "Risk/Reward breadth leans cautious, with bearish reads outnumbering bullish reads."
	case unavailable > 0:
		return "Risk/Reward breadth is mixed, with some assets still unavailable for evaluation."
	default:
		return "Risk/Reward breadth is balanced across the monitored group."
	}
}

func naturalNeutralContext(neutral, total int) string {
	if total <= 0 {
		return "The selected universe does not yet have enough breadth coverage for a reliable contextual read."
	}
	share := float64(neutral) / float64(total)
	switch {
	case share >= 0.70:
		return "Most monitored assets are still neutral, which limits broad directional conviction."
	case share >= 0.45:
		return "A meaningful portion of the monitored group remains neutral, so directional signals are still selective."
	default:
		return "Neutral conditions are no longer dominating the monitored group, so directional breadth deserves closer review."
	}
}

func naturalDirectionalImbalance(bullish, bearish int) string {
	diff := bullish - bearish
	switch {
	case diff >= 10:
		return "a clear bullish tilt"
	case diff > 0:
		return "a modest bullish tilt"
	case diff <= -10:
		return "a clear bearish tilt"
	case diff < 0:
		return "a modest bearish tilt"
	default:
		return "a balanced directional profile"
	}
}

func riskRewardDriverNarrative(index int, item map[string]any) string {
	symbol := asString(item["symbol"])
	direction := strings.ToLower(asString(item["direction"]))
	if direction == "" {
		direction = "directional"
	}
	leadIn := "is a monitored"
	if index == 0 {
		leadIn = "is the clearest"
	}
	return fmt.Sprintf("%s %s %s exception in the group and is showing a possible %s opportunity; evidence looks %s with a %s risk posture", symbol, leadIn, direction, direction, naturalEvidenceTone(floatFromAny(item["confidence"])), naturalRiskTone(asString(item["risk_level"])))
}

func naturalEvidenceTone(confidence float64) string {
	switch {
	case confidence >= 0.75:
		return "well supported"
	case confidence >= 0.55:
		return "constructive but still worth confirming"
	case confidence > 0:
		return "early and needs more confirmation"
	default:
		return "unclear from the bounded sample"
	}
}

func naturalRiskTone(risk string) string {
	switch strings.ToLower(strings.TrimSpace(risk)) {
	case "low":
		return "lower"
	case "medium":
		return "manageable but not clean"
	case "high":
		return "elevated"
	default:
		return "unclassified"
	}
}

func naturalReviewQueueText(active, expired int) string {
	switch {
	case active > 0 && expired > active:
		return "Review Queue is dominated by expired items, but there are still active opportunities that need current triage."
	case active > expired:
		return "Review Queue is primarily active, so the current triage set should drive analyst attention."
	case active > 0:
		return "Review Queue has a small active set that should be separated from historical or expired noise."
	default:
		return "Review Queue has no active opportunities in the bounded sample."
	}
}

func naturalActiveShare(active, total int) string {
	if total <= 0 {
		return "The queue does not have enough current items to form a triage view"
	}
	share := float64(active) / float64(total)
	switch {
	case share >= 0.50:
		return "Active opportunities make up a substantial part of the queue"
	case share >= 0.10:
		return "Active opportunities are present but still a minority of the queue"
	case active > 0:
		return "Active opportunities are a small minority of the queue"
	default:
		return "The queue is effectively historical for this window"
	}
}

func reviewQueueDriverNarrative(item map[string]any) string {
	symbol := asString(item["symbol"])
	direction := strings.ToLower(asString(item["direction"]))
	if direction == "" {
		direction = "directional"
	}
	when := asString(item["last_evaluated_date"])
	if when == "" {
		when = "the current window"
	}
	summary := asString(item["summary"])
	if summary != "" {
		return fmt.Sprintf("%s remains an active %s review item last evaluated %s; the persisted summary points to %s", symbol, direction, when, summary)
	}
	return fmt.Sprintf("%s remains an active %s review item last evaluated %s", symbol, direction, when)
}

func mapFromAny(value any) map[string]any {
	m, _ := value.(map[string]any)
	if m == nil {
		return map[string]any{}
	}
	return m
}

func sliceFromAny(value any) []any {
	items, _ := value.([]any)
	return items
}

func formatJSONNumber(value any) string {
	switch typed := value.(type) {
	case float64:
		if typed == float64(int64(typed)) {
			return fmt.Sprintf("%d", int64(typed))
		}
		return fmt.Sprintf("%.2f", typed)
	case int:
		return fmt.Sprintf("%d", typed)
	case int64:
		return fmt.Sprintf("%d", typed)
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

func floatFromAny(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case string:
		var out float64
		if _, err := fmt.Sscanf(strings.TrimSpace(typed), "%f", &out); err == nil {
			return out
		}
	}
	return 0
}

func intFromAny(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		var out int
		if _, err := fmt.Sscanf(strings.TrimSpace(typed), "%d", &out); err == nil {
			return out
		}
	}
	return 0
}

func humanizeSegment(segment string) string {
	text := strings.TrimSpace(segment)
	text = strings.TrimPrefix(text, "sri_sector_")
	text = strings.TrimPrefix(text, "sri_industry_")
	text = strings.ReplaceAll(text, "_", " ")
	return text
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func syncraticAskStructuredPayload(resp userapi.AskResponse) map[string]any {
	for _, candidate := range []string{resp.Answer, string(resp.Raw)} {
		if payload := parseSyncraticAskStructuredPayload(candidate); len(payload) > 0 {
			return payload
		}
	}
	return map[string]any{}
}

func parseSyncraticAskStructuredPayload(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil
	}
	if answer := asString(payload["answer"]); strings.HasPrefix(strings.TrimSpace(answer), "{") {
		if nested := parseSyncraticAskStructuredPayload(answer); len(nested) > 0 {
			return nested
		}
	}
	return payload
}

func renderSyncraticAskStructuredExplanation(payload map[string]any) string {
	if len(payload) == 0 {
		return ""
	}
	sections := []struct {
		key   string
		label string
	}{
		{key: "executive_summary", label: "Executive summary"},
		{key: "what_changed", label: "What changed"},
		{key: "top_drivers", label: "Top drivers"},
		{key: "contradictions_or_weak_evidence", label: "Contradictions or weak evidence"},
		{key: "analyst_followups", label: "Analyst follow-ups"},
		{key: "cited_artifacts", label: "Cited artifacts"},
		{key: "data_quality_warnings", label: "Data quality warnings"},
	}
	parts := []string{}
	for _, section := range sections {
		if rendered := renderSyncraticAskValue(payload[section.key]); rendered != "" {
			parts = append(parts, section.label+":\n"+rendered)
		}
	}
	return strings.Join(parts, "\n\n")
}

func renderSyncraticAskValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case []any:
		lines := []string{}
		for _, item := range typed {
			if rendered := renderSyncraticAskValue(item); rendered != "" {
				lines = append(lines, "- "+rendered)
			}
		}
		return strings.Join(lines, "\n")
	case map[string]any:
		if text := firstNonEmpty(asString(typed["summary"]), asString(typed["description"]), asString(typed["text"]), asString(typed["reason"])); text != "" {
			return text
		}
		raw, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return string(raw)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

func extractAskString(raw json.RawMessage, key string) string {
	if len(raw) == 0 {
		return ""
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return asString(value[key])
}

func jsonObjectOrEmpty(raw []byte) map[string]any {
	out := map[string]any{}
	if len(raw) == 0 {
		return out
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func asString(value any) string {
	s, _ := value.(string)
	return strings.TrimSpace(s)
}

func truncateForSummary(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 240 {
		return value
	}
	return value[:237] + "..."
}

func limitStrings(values []string, limit int) []string {
	values = uniqueSorted(values)
	if limit <= 0 || len(values) <= limit {
		return values
	}
	return values[:limit]
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
	if req.IncludeAllAssets {
		maxContexts, maxInsights, maxCandidates, minEvidence = maxAssets, maxAssets, maxAssets, 0
	}
	var jobs storage.SyncraticIntelligenceJobRepository
	if req.EnqueueBriefs && !req.DryRun {
		var ok bool
		jobs, ok = repo.(storage.SyncraticIntelligenceJobRepository)
		if !ok {
			return syncraticMaterializeResponse{}, fmt.Errorf("syncratic intelligence queue is unavailable")
		}
	}
	sessionDate := windowEnd.UTC()
	if strings.TrimSpace(req.SessionDate) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(req.SessionDate))
		if err != nil {
			return syncraticMaterializeResponse{}, fmt.Errorf("session_date must be YYYY-MM-DD")
		}
		sessionDate = parsed
	}
	ledger, err := loadSyncraticContextLedger(ctx, repo, tenantID, windowStart, windowEnd, req.SignalLimit, req.AlertLimit)
	if err != nil {
		return syncraticMaterializeResponse{}, err
	}
	assets, err := repo.ListMarketOpsAssets(ctx, tenantID, universeGroup, true, maxAssets)
	if err != nil {
		return syncraticMaterializeResponse{}, err
	}
	resp := syncraticMaterializeResponse{TenantID: tenantID, UniverseGroup: universeGroup, ContextStrategy: strategy, ContextBuilderVersion: builderVersion, WindowStart: windowStart.UTC(), WindowEnd: windowEnd.UTC(), DryRun: req.DryRun}
	criticalAlerts := ledger.Alerts
	plannedContextWindows := 0
	plannedInsights := 0
	for _, asset := range assets {
		resp.ScannedAssets++
		decision := syncraticMaterializeDecision{SubjectSymbol: strings.ToUpper(strings.TrimSpace(asset.Ticker))}
		if resp.CandidateWindows >= maxCandidates {
			decision.Action = "skipped"
			decision.Reason = "candidate_budget_cap"
			resp.SkippedBudgetCap++
			resp.Decisions = append(resp.Decisions, decision)
			continue
		}
		contextWindow, err := buildSyncraticContextWindowWithLedger(ctx, repo, tenantID, asset.Ticker, strategy, windowStart, windowEnd, builderVersion, nil, ledger)
		if err != nil {
			return resp, err
		}
		decision.SubjectSymbol = contextWindow.SubjectSymbol
		decision.SignalCount = len(contextWindow.SignalIDs)
		decision.AlertCount = len(contextWindow.AlertIDs)
		decision.ArtifactCount = len(contextWindow.ArtifactIDs)
		decision.GraphProposalCount = len(contextWindow.GraphProposalIDs)
		decision.LabelCount = len(contextWindow.LabelIDs)
		decision.EvidenceCount = len(contextWindow.SignalIDs) + len(contextWindow.AlertIDs)
		decision.EvidenceDigest = contextWindow.EvidenceDigest
		decision.ContextWindowID = contextWindow.ContextWindowID
		for _, alert := range criticalAlerts {
			if timeInWindow(alert.LastObservedAt, windowStart, windowEnd) && recordEvidenceMatchesSymbol(asset.Ticker, alert.EntitiesJSON, alert.EvidenceJSON) {
				decision.CriticalAlert = true
				break
			}
		}
		decision.RelatedEvidence = len(contextWindow.GraphProposalIDs) > 0 || len(contextWindow.LabelIDs) > 0
		if !req.IncludeAllAssets && decision.EvidenceCount < minEvidence && !decision.CriticalAlert && !decision.RelatedEvidence {
			decision.Action = "skipped"
			decision.Reason = "below_threshold"
			resp.SkippedBelowThreshold++
			resp.Decisions = append(resp.Decisions, decision)
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
			decision.Action = "skipped"
			decision.Reason = "unchanged_evidence_digest"
			resp.SkippedUnchanged++
			resp.Decisions = append(resp.Decisions, decision)
			continue
		}
		if plannedContextWindows >= maxContexts || plannedInsights >= maxInsights {
			decision.Action = "skipped"
			decision.Reason = "materialization_budget_cap"
			resp.SkippedBudgetCap++
			resp.Decisions = append(resp.Decisions, decision)
			continue
		}
		if req.DryRun {
			plannedContextWindows++
			plannedInsights++
			decision.Action = "would_materialize"
			decision.Reason = "eligible"
			resp.Decisions = append(resp.Decisions, decision)
			continue
		}
		if err := repo.UpsertSyncraticContextWindow(ctx, contextWindow); err != nil {
			return resp, err
		}
		insight := buildSyncraticInsight(contextWindow, firstNonEmpty(req.InsightType, defaultSyncraticInsightType), builderVersion)
		if err := repo.UpsertSyncraticInsight(ctx, insight); err != nil {
			return resp, err
		}
		if jobs != nil {
			job := storage.SyncraticIntelligenceJobRecord{JobID: stableSyncraticID("synjob", tenantID, contextWindow.ContextWindowID, contextWindow.EvidenceDigest), TenantID: tenantID, AppID: "marketops", UseCase: "daily_market_surveillance", SubjectSymbol: contextWindow.SubjectSymbol, SessionDate: sessionDate, ContextWindowID: contextWindow.ContextWindowID, EvidenceDigest: contextWindow.EvidenceDigest, MaxAttempts: 3}
			if err := jobs.UpsertSyncraticIntelligenceJob(ctx, job); err != nil {
				return resp, err
			}
			resp.QueuedJobIDs = append(resp.QueuedJobIDs, job.JobID)
		}
		plannedContextWindows++
		plannedInsights++
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

func recordEvidenceMatchesSymbol(symbol string, requiredRaw []byte, supportingRaw ...[]byte) bool {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" || !jsonPayloadHasExactSymbol(requiredRaw, symbol) {
		return false
	}
	for _, raw := range supportingRaw {
		for other := range extractKnownSymbols(raw) {
			if other != "" && other != symbol {
				return false
			}
		}
	}
	return true
}

func jsonPayloadHasExactSymbol(raw []byte, symbol string) bool {
	if len(raw) == 0 || symbol == "" || !json.Valid(raw) {
		return false
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return false
	}
	symbols := map[string]struct{}{}
	collectSymbols(value, symbols)
	_, ok := symbols[strings.ToUpper(strings.TrimSpace(symbol))]
	return ok
}

func extractKnownSymbols(raw []byte) map[string]struct{} {
	out := map[string]struct{}{}
	if len(raw) == 0 {
		return out
	}
	if json.Valid(raw) {
		var value any
		if err := json.Unmarshal(raw, &value); err == nil {
			collectSymbols(value, out)
		}
	}
	for _, candidate := range knownMarketOpsSymbols() {
		if textContainsSymbolToken(string(raw), candidate) {
			out[candidate] = struct{}{}
		}
	}
	return out
}

// textContainsSymbolToken only recognizes a symbol as a complete token. This
// prevents short symbols such as V and MA from being inferred from ordinary
// words (for example, "volatility", "gamma", or the JSON key "summary").
// Structured symbol fields are still handled by collectSymbols.
func textContainsSymbolToken(text, symbol string) bool {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if text == "" || symbol == "" {
		return false
	}
	upper := strings.ToUpper(text)
	for start := 0; start < len(upper); {
		index := strings.Index(upper[start:], symbol)
		if index < 0 {
			return false
		}
		index += start
		end := index + len(symbol)
		if (index == 0 || !isSymbolTokenCharacter(upper[index-1])) && (end == len(upper) || !isSymbolTokenCharacter(upper[end])) {
			return true
		}
		start = index + 1
	}
	return false
}

func isSymbolTokenCharacter(value byte) bool {
	return (value >= 'A' && value <= 'Z') || (value >= '0' && value <= '9')
}

func knownMarketOpsSymbols() []string {
	return []string{"AAPL", "MSFT", "NVDA", "AMZN", "META", "GOOGL", "GOOG", "TSLA", "MS", "GE", "MA", "V", "MU", "SPY"}
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
