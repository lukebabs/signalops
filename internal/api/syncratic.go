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
	defaultSyncraticBuilderVersion   = "syncratic.context_builder.v1"
	defaultSyncraticInsightType      = "marketops.syncratic.multi_event_context"
	defaultSyncraticEODInsightType   = "marketops.syncratic.eod_overview.v1"
	defaultSyncraticAskDrilldownType = "marketops.syncratic.ask_drilldown.v1"
	defaultSyncraticAskPromptVersion = "marketops.syncratic.ask_prompt.v1"
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
	if err != nil { return storage.SyncraticContextWindowRecord{}, err }
	states, err := repo.ListMarketOpsMarketStates(ctx, storage.MarketOpsMarketStateFilter{TenantID: tenantID, AppID: "marketops", Symbol: subjectSymbol, SessionStart: windowStart, SessionEnd: windowEnd, Limit: 1})
	if err != nil { return storage.SyncraticContextWindowRecord{}, err }
	transitions, err := repo.ListMarketOpsStateTransitions(ctx, storage.MarketOpsStateTransitionFilter{TenantID: tenantID, AppID: "marketops", Symbol: subjectSymbol, SessionStart: windowStart, SessionEnd: windowEnd, Limit: 100})
	if err != nil { return storage.SyncraticContextWindowRecord{}, err }
	evidence, err := repo.ListMarketOpsEvidence(ctx, storage.MarketOpsEvidenceFilter{TenantID: tenantID, AppID: "marketops", Symbol: subjectSymbol, SessionStart: windowStart, SessionEnd: windowEnd, Limit: 100})
	if err != nil { return storage.SyncraticContextWindowRecord{}, err }
	if err != nil {
		return storage.SyncraticContextWindowRecord{}, err
	}

	allowedTypes := stringSet(signalTypes)
	record := storage.SyncraticContextWindowRecord{TenantID: tenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SubjectType: "ticker", SubjectID: subjectSymbol, SubjectSymbol: subjectSymbol, WindowStart: windowStart.UTC(), WindowEnd: windowEnd.UTC(), ContextStrategy: strategy, ContextBuilderVersion: builderVersion, Status: "active", ContextPayloadVersion: "signalops.syncratic.market_state_session.v2"}
	detectors := map[string]struct{}{}
	signalTypeSet := map[string]struct{}{}
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
	for _, state := range states {
		record.MarketStateIDs = append(record.MarketStateIDs, state.MarketStateID)
		if state.QualityState != "" && state.QualityState != "complete" { record.QualityWarningsJSON = mustJSON([]map[string]string{{"kind": "market_state_quality", "value": state.QualityState}}) }
	}
	for _, transition := range transitions { record.StateTransitionIDs = append(record.StateTransitionIDs, transition.TransitionID) }
	for _, item := range evidence { record.MarketOpsEvidenceIDs = append(record.MarketOpsEvidenceIDs, item.EvidenceID) }
	record.MarketStateIDs = uniqueSorted(record.MarketStateIDs)
	record.StateTransitionIDs = uniqueSorted(record.StateTransitionIDs)
	record.MarketOpsEvidenceIDs = uniqueSorted(record.MarketOpsEvidenceIDs)
	stateSummary := make([]map[string]any, 0, len(states))
	for _, state := range states { stateSummary = append(stateSummary, map[string]any{"market_state_id": state.MarketStateID, "quality_state": state.QualityState, "completeness_ratio": state.CompletenessRatio, "eligible_hypotheses": state.EligibleHypotheses, "state_payload": json.RawMessage(jsonOrDefault(state.StatePayloadJSON, `{}`))}) }
	eodEvidence := make([]map[string]any, 0, len(evidence))
	for _, item := range evidence { eodEvidence = append(eodEvidence, map[string]any{"evidence_id": item.EvidenceID, "type": item.EvidenceType, "domain": item.Domain, "direction": item.Direction, "statement": item.Statement, "magnitude": item.Magnitude, "quality_score": item.QualityScore, "payload": json.RawMessage(jsonOrDefault(item.EvidencePayloadJSON, `{}`))}) }
	record.SignalTypes = setKeys(signalTypeSet)
	record.DetectorIDs = setKeys(detectors)
	metrics := map[string]any{"signal_count": len(record.SignalIDs), "alert_count": len(record.AlertIDs), "event_count": len(record.EventIDs), "artifact_count": len(record.ArtifactIDs), "graph_proposal_count": len(record.GraphProposalIDs), "label_count": len(record.LabelIDs), "market_state": stateSummary, "market_evidence": eodEvidence, "subject_symbol": subjectSymbol, "context_strategy": strategy}
	record.SummaryMetricsJSON = mustJSON(metrics)
	record.BaselineRefsJSON = []byte(`[]`)
	record.EvaluationRefsJSON = []byte(`[]`)
	record.PromotionCandidateRefsJSON = mustJSON(promotionRefs)
	if len(record.QualityWarningsJSON) == 0 { record.QualityWarningsJSON = []byte(`[]`) }
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
	if insightType == "" && contextWindow.ContextStrategy == "market_state_session_v2" { insightType = defaultSyncraticAskDrilldownType }
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
	}
	includeRefs := false
	directReasoning := true
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
	text := strings.ToUpper(string(raw))
	return strings.Contains(text, strings.ToUpper(symbol))
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
			"top_drivers: if analysis_mode=data_quality_blocked, write none; otherwise ranked bullets with signal_id, type, severity, confidence, key metrics, and why it matters",
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
	prompt := "You are a non-human MarketOps reasoning client. Produce operator-useful interpretation, not a generic summary. If analysis_mode is data_quality_blocked, lead with DATA QUALITY WARNING, do not infer that one ticker impacts another, do not describe market impact, and make the recommendation about evidence remediation rather than trading/market action. Use only the JSON context below. Return the requested fields and cite evidence IDs.\nCONTEXT_JSON:\n" + string(raw)
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
	metrics := jsonObjectOrEmpty(insight.MetricsJSON)
	ask, ok := metrics["syncratic_ask"].(map[string]any)
	if !ok {
		return false
	}
	return asString(ask["ask_status"]) == "completed" && asString(ask["prompt_digest"]) == meta.PromptDigest && asString(ask["context_evidence_digest"]) == meta.ContextEvidenceDigest
}

func applySyncraticAskResponse(insight storage.SyncraticInsightRecord, contextWindow storage.SyncraticContextWindowRecord, meta syncraticAskPromptMeta, resp userapi.AskResponse, started, completed time.Time) storage.SyncraticInsightRecord {
	answer := strings.TrimSpace(resp.Answer)
	if answer == "" && len(resp.Raw) > 0 {
		var raw map[string]any
		if json.Unmarshal(resp.Raw, &raw) == nil {
			answer = firstNonEmpty(asString(raw["answer"]), asString(raw["explanation"]), asString(raw["summary"]))
		}
	}
	if answer == "" {
		answer = "Syncratic Ask returned no textual explanation. Review deterministic evidence directly."
	}
	insight.Explanation = answer
	insight.Summary = firstNonEmpty(extractAskString(resp.Raw, "summary"), truncateForSummary(answer), insight.Summary)
	insight.Title = firstNonEmpty(extractAskString(resp.Raw, "title"), insight.Title, fmt.Sprintf("%s Syncratic Ask explanation", contextWindow.SubjectSymbol))
	if resp.Confidence > 0 {
		insight.Confidence = float64(resp.Confidence)
	}
	metrics := jsonObjectOrEmpty(insight.MetricsJSON)
	metrics["syncratic_ask"] = map[string]any{"enabled": true, "ask_query_id": resp.QueryID, "ask_status": "completed", "prompt_builder_version": meta.PromptBuilderVersion, "prompt_digest": meta.PromptDigest, "context_window_id": contextWindow.ContextWindowID, "context_evidence_digest": meta.ContextEvidenceDigest, "request_scope": defaultSyncraticAskScope, "request_k": 1, "direct_reasoning": true, "graph_enabled": false, "kee_enabled": false, "included_record_details": meta.IncludedRecordDetails, "prompt_bytes": meta.PromptBytes, "caps": meta.Caps, "response": map[string]any{"confidence": resp.Confidence, "evidence_count": resp.EvidenceCount, "citation_count": len(resp.Citations)}, "started_at": started.Format(time.RFC3339Nano), "completed_at": completed.Format(time.RFC3339Nano), "latency_ms": completed.Sub(started).Milliseconds()}
	insight.MetricsJSON = mustJSON(metrics)
	insight.RecommendationJSON = mustJSON(map[string]any{"action": firstNonEmpty(extractAskString(resp.Raw, "action"), "review"), "source": "syncratic_ask", "reason": "LLM-generated explanation over a bounded deterministic SignalOps context window", "ask_query_id": resp.QueryID, "prompt_digest": meta.PromptDigest})
	return insight
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
		if !ok { return syncraticMaterializeResponse{}, fmt.Errorf("syncratic intelligence queue is unavailable") }
	}
	sessionDate := windowEnd.UTC()
	if strings.TrimSpace(req.SessionDate) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(req.SessionDate))
		if err != nil { return syncraticMaterializeResponse{}, fmt.Errorf("session_date must be YYYY-MM-DD") }
		sessionDate = parsed
	}
	assets, err := repo.ListMarketOpsAssets(ctx, tenantID, universeGroup, true, maxAssets)
	if err != nil {
		return syncraticMaterializeResponse{}, err
	}
	resp := syncraticMaterializeResponse{TenantID: tenantID, UniverseGroup: universeGroup, ContextStrategy: strategy, ContextBuilderVersion: builderVersion, WindowStart: windowStart.UTC(), WindowEnd: windowEnd.UTC(), DryRun: req.DryRun}
	criticalAlerts, err := repo.ListAlertLedger(ctx, storage.AlertLedgerFilter{TenantID: tenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", Severity: "critical", Limit: alertLimitOrDefault(req.AlertLimit)})
	if err != nil {
		return resp, err
	}
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
		contextWindow, err := buildSyncraticContextWindow(ctx, repo, tenantID, asset.Ticker, strategy, windowStart, windowEnd, builderVersion, nil, req.SignalLimit, req.AlertLimit)
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
			if err := jobs.UpsertSyncraticIntelligenceJob(ctx, job); err != nil { return resp, err }
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
	upper := strings.ToUpper(string(raw))
	for _, candidate := range knownMarketOpsSymbols() {
		if strings.Contains(upper, candidate) {
			out[candidate] = struct{}{}
		}
	}
	return out
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
