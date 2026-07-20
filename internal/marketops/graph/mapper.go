package graph

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const (
	defaultAppID   = "marketops"
	defaultDomain  = "market_data"
	defaultUseCase = "daily_market_surveillance"
)

var sourceOrder = []string{
	storage.MarketOpsGraphProposalSourceMarketState,
	storage.MarketOpsGraphProposalSourceStateTransition,
	storage.MarketOpsGraphProposalSourceHypothesisDefinition,
	storage.MarketOpsGraphProposalSourceHypothesisEvaluation,
	storage.MarketOpsGraphProposalSourceOpportunity,
	storage.MarketOpsGraphProposalSourceOutcome,
}

type Repository interface {
	ListMarketOpsMarketStates(context.Context, storage.MarketOpsMarketStateFilter) ([]storage.MarketOpsMarketStateRecord, error)
	ListMarketOpsStateTransitions(context.Context, storage.MarketOpsStateTransitionFilter) ([]storage.MarketOpsStateTransitionRecord, error)
	ListMarketOpsHypothesisDefinitions(context.Context, storage.MarketOpsHypothesisDefinitionFilter) ([]storage.MarketOpsHypothesisDefinitionRecord, error)
	ListMarketOpsHypothesisEvaluations(context.Context, storage.MarketOpsHypothesisEvaluationFilter) ([]storage.MarketOpsHypothesisEvaluationRecord, error)
	ListMarketOpsOpportunities(context.Context, storage.MarketOpsOpportunityFilter) ([]storage.MarketOpsOpportunityRecord, error)
	ListMarketOpsSignalOutcomes(context.Context, storage.MarketOpsSignalOutcomeFilter) ([]storage.MarketOpsSignalOutcomeRecord, error)
	UpsertMarketOpsDSMGraphProposal(context.Context, storage.MarketOpsDSMGraphProposalRecord) error
}

type Config struct {
	TenantID         string
	Symbol           string
	SessionStart     time.Time
	SessionEnd       time.Time
	SourceTypes      []string
	MaxSourceRecords int
	MaxProposals     int
	DryRun           bool
}

type Result struct {
	TenantID        string         `json:"tenant_id"`
	Symbol          string         `json:"symbol"`
	SessionStart    string         `json:"session_start"`
	SessionEnd      string         `json:"session_end"`
	SourceTypes     []string       `json:"source_types"`
	SourceRecords   int            `json:"source_records"`
	SourceCounts    map[string]int `json:"source_counts"`
	CandidateCounts map[string]int `json:"candidate_counts"`
	QualityCounts   map[string]int `json:"quality_counts"`
	SkippedReasons  map[string]int `json:"skipped_reasons"`
	Proposals       int            `json:"proposals"`
	Written         int            `json:"written"`
	ProposalIDs     []string       `json:"proposal_ids"`
	DryRun          bool           `json:"dry_run"`
}

type sourceIdentity struct {
	SourceType string
	ID         string
	Version    string
	Symbol     string
	Refs       map[string]any
	Lineage    map[string]any
}

type candidate struct {
	Type         string
	NodeID       string
	FromNode     string
	Relationship string
	ToNode       string
	Labels       []string
	Properties   map[string]any
}

func Map(ctx context.Context, repo Repository, cfg Config) (Result, error) {
	cfg.normalize()
	if err := cfg.validate(); err != nil {
		return Result{}, err
	}
	result := Result{
		TenantID: cfg.TenantID, Symbol: cfg.Symbol,
		SessionStart: cfg.SessionStart.Format("2006-01-02"), SessionEnd: cfg.SessionEnd.Format("2006-01-02"),
		SourceTypes: append([]string(nil), cfg.SourceTypes...), SourceCounts: map[string]int{},
		CandidateCounts: map[string]int{}, QualityCounts: map[string]int{}, SkippedReasons: map[string]int{},
		DryRun: cfg.DryRun,
	}
	selected := make(map[string]bool, len(cfg.SourceTypes))
	for _, sourceType := range cfg.SourceTypes {
		selected[sourceType] = true
	}
	records, err := loadSourceRecords(ctx, repo, cfg, selected, &result)
	if err != nil {
		return result, err
	}
	if len(records) >= cfg.MaxSourceRecords && result.SkippedReasons["max_source_records_reached"] == 0 {
		result.SkippedReasons["max_source_records_reached"]++
	}
	if len(records) == 0 {
		result.SkippedReasons["no_source_records"]++
		return result, nil
	}
	proposals := make([]storage.MarketOpsDSMGraphProposalRecord, 0, min(cfg.MaxProposals, len(records)*3+1))
	add := func(source sourceIdentity, item candidate) bool {
		if len(proposals) >= cfg.MaxProposals {
			result.SkippedReasons["max_proposals_reached"]++
			return false
		}
		proposal, buildErr := buildProposal(cfg, source, item)
		if buildErr != nil {
			result.SkippedReasons["candidate_encoding_failed"]++
			return true
		}
		proposals = append(proposals, proposal)
		result.CandidateCounts[item.Type]++
		return true
	}
	first := records[0].source()
	add(first, candidate{
		Type: "node_candidate", NodeID: assetNodeID(cfg.Symbol), Labels: []string{"Asset", "Ticker"},
		Properties: map[string]any{"symbol": cfg.Symbol, "asset_id": assetNodeID(cfg.Symbol)},
	})
	for _, record := range records {
		if len(proposals) >= cfg.MaxProposals {
			break
		}
		record.emit(add, &result)
	}
	sort.Slice(proposals, func(i, j int) bool { return proposals[i].ProposalID < proposals[j].ProposalID })
	result.Proposals = len(proposals)
	result.ProposalIDs = make([]string, 0, len(proposals))
	for _, proposal := range proposals {
		result.ProposalIDs = append(result.ProposalIDs, proposal.ProposalID)
		if !cfg.DryRun {
			if err := repo.UpsertMarketOpsDSMGraphProposal(ctx, proposal); err != nil {
				return result, fmt.Errorf("write graph proposal %s: %w", proposal.ProposalID, err)
			}
			result.Written++
		}
	}
	return result, nil
}

type sourceRecord interface {
	source() sourceIdentity
	emit(func(sourceIdentity, candidate) bool, *Result)
}

func loadSourceRecords(ctx context.Context, repo Repository, cfg Config, selected map[string]bool, result *Result) ([]sourceRecord, error) {
	all := make([]sourceRecord, 0, cfg.MaxSourceRecords)
	appendRecord := func(record sourceRecord) bool {
		if len(all) >= cfg.MaxSourceRecords {
			result.SkippedReasons["max_source_records_reached"]++
			return false
		}
		all = append(all, record)
		source := record.source()
		result.SourceCounts[source.SourceType]++
		result.SourceRecords++
		return true
	}
	if selected[storage.MarketOpsGraphProposalSourceMarketState] {
		items, err := repo.ListMarketOpsMarketStates(ctx, storage.MarketOpsMarketStateFilter{TenantID: cfg.TenantID, AppID: defaultAppID, Symbol: cfg.Symbol, SessionStart: cfg.SessionStart, SessionEnd: cfg.SessionEnd, Limit: cfg.MaxSourceRecords})
		if err != nil {
			return nil, fmt.Errorf("list market states: %w", err)
		}
		sort.Slice(items, func(i, j int) bool { return items[i].MarketStateID < items[j].MarketStateID })
		for i := range items {
			if !appendRecord(stateSource{items[i]}) {
				break
			}
			result.QualityCounts["market_state:"+valueOrUnknown(items[i].QualityState)]++
		}
	}
	if selected[storage.MarketOpsGraphProposalSourceStateTransition] && len(all) < cfg.MaxSourceRecords {
		items, err := repo.ListMarketOpsStateTransitions(ctx, storage.MarketOpsStateTransitionFilter{TenantID: cfg.TenantID, AppID: defaultAppID, Symbol: cfg.Symbol, SessionStart: cfg.SessionStart, SessionEnd: cfg.SessionEnd, Limit: cfg.MaxSourceRecords})
		if err != nil {
			return nil, fmt.Errorf("list state transitions: %w", err)
		}
		sort.Slice(items, func(i, j int) bool { return items[i].TransitionID < items[j].TransitionID })
		for i := range items {
			if !appendRecord(transitionSource{items[i]}) {
				break
			}
			result.QualityCounts["state_transition:"+valueOrUnknown(items[i].QualityState)]++
		}
	}
	needEvaluations := selected[storage.MarketOpsGraphProposalSourceHypothesisDefinition] || selected[storage.MarketOpsGraphProposalSourceHypothesisEvaluation]
	var evaluations []storage.MarketOpsHypothesisEvaluationRecord
	if needEvaluations && len(all) < cfg.MaxSourceRecords {
		items, err := repo.ListMarketOpsHypothesisEvaluations(ctx, storage.MarketOpsHypothesisEvaluationFilter{TenantID: cfg.TenantID, AppID: defaultAppID, Symbol: cfg.Symbol, SessionStart: cfg.SessionStart, SessionEnd: cfg.SessionEnd, Limit: cfg.MaxSourceRecords * 4})
		if err != nil {
			return nil, fmt.Errorf("list hypothesis evaluations: %w", err)
		}
		sort.Slice(items, func(i, j int) bool { return items[i].EvaluationID < items[j].EvaluationID })
		evaluations = items
	}
	if selected[storage.MarketOpsGraphProposalSourceHypothesisDefinition] && len(all) < cfg.MaxSourceRecords {
		definitions, err := repo.ListMarketOpsHypothesisDefinitions(ctx, storage.MarketOpsHypothesisDefinitionFilter{TenantID: cfg.TenantID, Limit: cfg.MaxSourceRecords * 4})
		if err != nil {
			return nil, fmt.Errorf("list hypothesis definitions: %w", err)
		}
		referenced := map[string]bool{}
		for _, evaluation := range evaluations {
			referenced[evaluation.HypothesisKey+"\x00"+evaluation.HypothesisVersion] = true
		}
		sort.Slice(definitions, func(i, j int) bool {
			return definitions[i].HypothesisKey+"\x00"+definitions[i].HypothesisVersion < definitions[j].HypothesisKey+"\x00"+definitions[j].HypothesisVersion
		})
		for i := range definitions {
			if !referenced[definitions[i].HypothesisKey+"\x00"+definitions[i].HypothesisVersion] {
				continue
			}
			if !appendRecord(definitionSource{definitions[i], cfg.Symbol}) {
				break
			}
			result.QualityCounts["hypothesis_definition:"+valueOrUnknown(definitions[i].LifecycleStatus)]++
		}
	}
	if selected[storage.MarketOpsGraphProposalSourceHypothesisEvaluation] && len(all) < cfg.MaxSourceRecords {
		for i := range evaluations {
			if !appendRecord(evaluationSource{evaluations[i]}) {
				break
			}
			quality := "evaluated"
			if evaluations[i].Invalidated {
				quality = "invalidated"
			} else if evaluations[i].Triggered {
				quality = "triggered"
			} else if evaluations[i].Eligible {
				quality = "eligible"
			}
			result.QualityCounts["hypothesis_evaluation:"+quality]++
		}
	}
	if selected[storage.MarketOpsGraphProposalSourceOpportunity] && len(all) < cfg.MaxSourceRecords {
		items, err := repo.ListMarketOpsOpportunities(ctx, storage.MarketOpsOpportunityFilter{TenantID: cfg.TenantID, AppID: defaultAppID, Symbol: cfg.Symbol, SessionStart: cfg.SessionStart, SessionEnd: cfg.SessionEnd, Limit: cfg.MaxSourceRecords})
		if err != nil {
			return nil, fmt.Errorf("list opportunities: %w", err)
		}
		sort.Slice(items, func(i, j int) bool { return items[i].OpportunityID < items[j].OpportunityID })
		for i := range items {
			if !appendRecord(opportunitySource{items[i]}) {
				break
			}
			result.QualityCounts["opportunity:"+valueOrUnknown(items[i].LifecycleStatus)]++
		}
	}
	if selected[storage.MarketOpsGraphProposalSourceOutcome] && len(all) < cfg.MaxSourceRecords {
		items, err := repo.ListMarketOpsSignalOutcomes(ctx, storage.MarketOpsSignalOutcomeFilter{TenantID: cfg.TenantID, AppID: defaultAppID, Symbol: cfg.Symbol, OriginStart: cfg.SessionStart, OriginEnd: cfg.SessionEnd, Limit: cfg.MaxSourceRecords})
		if err != nil {
			return nil, fmt.Errorf("list outcomes: %w", err)
		}
		sort.Slice(items, func(i, j int) bool { return items[i].OutcomeID < items[j].OutcomeID })
		for i := range items {
			if items[i].OutcomeStatus != storage.MarketOpsOutcomeMatured {
				result.SkippedReasons["outcome_not_matured"]++
				continue
			}
			if !appendRecord(outcomeSource{items[i]}) {
				break
			}
			result.QualityCounts["outcome:"+valueOrUnknown(items[i].OutcomeStatus)]++
		}
	}
	return all, nil
}

type stateSource struct {
	value storage.MarketOpsMarketStateRecord
}

func (s stateSource) source() sourceIdentity {
	return sourceIdentity{SourceType: storage.MarketOpsGraphProposalSourceMarketState, ID: s.value.MarketStateID, Version: s.value.StateSchemaVersion, Symbol: s.value.Symbol,
		Refs: map[string]any{"market_state_id": s.value.MarketStateID}, Lineage: map[string]any{"feature_observation_ids": bounded(s.value.FeatureObservationIDs)}}
}
func (s stateSource) emit(add func(sourceIdentity, candidate) bool, _ *Result) {
	src := s.source()
	node := "market_state:" + s.value.MarketStateID
	add(src, candidate{Type: "node_candidate", NodeID: node, Labels: []string{"MarketState"}, Properties: map[string]any{"market_state_id": s.value.MarketStateID, "session_date": date(s.value.SessionDate), "state_schema_version": s.value.StateSchemaVersion, "quality_state": s.value.QualityState, "completeness_ratio": s.value.CompletenessRatio}})
	add(src, candidate{Type: "relationship_candidate", FromNode: node, Relationship: "STATE_OF", ToNode: assetNodeID(s.value.Symbol), Properties: map[string]any{"session_date": date(s.value.SessionDate)}})
}

type transitionSource struct {
	value storage.MarketOpsStateTransitionRecord
}

func (s transitionSource) source() sourceIdentity {
	return sourceIdentity{SourceType: storage.MarketOpsGraphProposalSourceStateTransition, ID: s.value.TransitionID, Version: s.value.FeatureVersion, Symbol: s.value.Symbol,
		Refs: map[string]any{"transition_id": s.value.TransitionID}, Lineage: map[string]any{"current_state_id": s.value.CurrentStateID, "baseline_state_id": s.value.BaselineStateID}}
}
func (s transitionSource) emit(add func(sourceIdentity, candidate) bool, _ *Result) {
	src := s.source()
	node := "state_transition:" + s.value.TransitionID
	add(src, candidate{Type: "node_candidate", NodeID: node, Labels: []string{"StateTransition"}, Properties: map[string]any{"transition_id": s.value.TransitionID, "session_date": date(s.value.SessionDate), "feature_key": s.value.FeatureKey, "feature_version": s.value.FeatureVersion, "transition_type": s.value.TransitionType, "direction": s.value.Direction, "quality_state": s.value.QualityState}})
	add(src, candidate{Type: "relationship_candidate", FromNode: node, Relationship: "TRANSITION_OF", ToNode: "market_state:" + s.value.CurrentStateID, Properties: map[string]any{"session_date": date(s.value.SessionDate)}})
}

type definitionSource struct {
	value  storage.MarketOpsHypothesisDefinitionRecord
	symbol string
}

func (s definitionSource) source() sourceIdentity {
	return sourceIdentity{SourceType: storage.MarketOpsGraphProposalSourceHypothesisDefinition, ID: s.value.HypothesisKey, Version: s.value.HypothesisVersion, Symbol: s.symbol,
		Refs: map[string]any{"hypothesis_key": s.value.HypothesisKey, "hypothesis_version": s.value.HypothesisVersion}, Lineage: map[string]any{}}
}
func (s definitionSource) emit(add func(sourceIdentity, candidate) bool, _ *Result) {
	node := hypothesisNodeID(s.value.HypothesisKey, s.value.HypothesisVersion)
	add(s.source(), candidate{Type: "node_candidate", NodeID: node, Labels: []string{"HypothesisDefinition"}, Properties: map[string]any{"hypothesis_key": s.value.HypothesisKey, "hypothesis_version": s.value.HypothesisVersion, "domain": s.value.Domain, "direction": s.value.Direction, "lifecycle_status": s.value.LifecycleStatus}})
}

type evaluationSource struct {
	value storage.MarketOpsHypothesisEvaluationRecord
}

func (s evaluationSource) source() sourceIdentity {
	return sourceIdentity{SourceType: storage.MarketOpsGraphProposalSourceHypothesisEvaluation, ID: s.value.EvaluationID, Version: s.value.HypothesisVersion, Symbol: s.value.Symbol,
		Refs: map[string]any{"evaluation_id": s.value.EvaluationID}, Lineage: map[string]any{"market_state_id": s.value.MarketStateID, "evidence_ids": bounded(s.value.EvidenceIDs)}}
}
func (s evaluationSource) emit(add func(sourceIdentity, candidate) bool, _ *Result) {
	src := s.source()
	node := "hypothesis_evaluation:" + s.value.EvaluationID
	add(src, candidate{Type: "node_candidate", NodeID: node, Labels: []string{"HypothesisEvaluation"}, Properties: map[string]any{"evaluation_id": s.value.EvaluationID, "session_date": date(s.value.SessionDate), "hypothesis_key": s.value.HypothesisKey, "hypothesis_version": s.value.HypothesisVersion, "eligible": s.value.Eligible, "triggered": s.value.Triggered, "invalidated": s.value.Invalidated}})
	add(src, candidate{Type: "relationship_candidate", FromNode: node, Relationship: "EVALUATES_STATE", ToNode: "market_state:" + s.value.MarketStateID, Properties: map[string]any{"session_date": date(s.value.SessionDate)}})
	add(src, candidate{Type: "relationship_candidate", FromNode: node, Relationship: "INSTANCE_OF", ToNode: hypothesisNodeID(s.value.HypothesisKey, s.value.HypothesisVersion), Properties: map[string]any{"hypothesis_version": s.value.HypothesisVersion}})
}

type opportunitySource struct {
	value storage.MarketOpsOpportunityRecord
}

func (s opportunitySource) source() sourceIdentity {
	return sourceIdentity{SourceType: storage.MarketOpsGraphProposalSourceOpportunity, ID: s.value.OpportunityID, Version: strconv.Itoa(s.value.Version), Symbol: s.value.Symbol,
		Refs: map[string]any{"opportunity_id": s.value.OpportunityID}, Lineage: map[string]any{"hypothesis_evaluation_ids": bounded(s.value.HypothesisEvaluationIDs), "supporting_evidence_ids": bounded(s.value.SupportingEvidenceIDs)}}
}
func (s opportunitySource) emit(add func(sourceIdentity, candidate) bool, _ *Result) {
	src := s.source()
	node := "opportunity:" + s.value.OpportunityID
	if !add(src, candidate{Type: "node_candidate", NodeID: node, Labels: []string{"Opportunity"}, Properties: map[string]any{"opportunity_id": s.value.OpportunityID, "opened_session_date": date(s.value.OpenedSessionDate), "direction": s.value.Direction, "horizon": s.value.Horizon, "lifecycle_status": s.value.LifecycleStatus, "version": s.value.Version, "research_only": s.value.ResearchOnly}}) {
		return
	}
	for _, evaluationID := range bounded(s.value.HypothesisEvaluationIDs) {
		if !add(src, candidate{Type: "relationship_candidate", FromNode: node, Relationship: "SUPPORTED_BY_EVALUATION", ToNode: "hypothesis_evaluation:" + evaluationID, Properties: map[string]any{"opportunity_version": s.value.Version}}) {
			return
		}
	}
}

type outcomeSource struct {
	value storage.MarketOpsSignalOutcomeRecord
}

func (s outcomeSource) source() sourceIdentity {
	return sourceIdentity{SourceType: storage.MarketOpsGraphProposalSourceOutcome, ID: s.value.OutcomeID, Version: s.value.CalculationVersion, Symbol: s.value.Symbol,
		Refs: map[string]any{"outcome_id": s.value.OutcomeID, "source_type": s.value.SourceType, "source_id": s.value.SourceID}, Lineage: map[string]any{"outcome_event_ids": bounded(s.value.OutcomeEventIDs), "origin_event_id": s.value.OriginEventID}}
}
func (s outcomeSource) emit(add func(sourceIdentity, candidate) bool, _ *Result) {
	src := s.source()
	node := "outcome:" + s.value.OutcomeID
	add(src, candidate{Type: "node_candidate", NodeID: node, Labels: []string{"Outcome"}, Properties: map[string]any{"outcome_id": s.value.OutcomeID, "origin_session_date": date(s.value.OriginSessionDate), "matured_session_date": optionalDate(s.value.MaturedSessionDate), "direction": s.value.Direction, "outcome_status": s.value.OutcomeStatus, "calculation_version": s.value.CalculationVersion}})
	add(src, candidate{Type: "relationship_candidate", FromNode: node, Relationship: "OUTCOME_OF", ToNode: sourceNodeID(s.value.SourceType, s.value.SourceID), Properties: map[string]any{"source_type": s.value.SourceType}})
}

func buildProposal(cfg Config, source sourceIdentity, item candidate) (storage.MarketOpsDSMGraphProposalRecord, error) {
	properties, err := json.Marshal(item.Properties)
	if err != nil {
		return storage.MarketOpsDSMGraphProposalRecord{}, err
	}
	raw := map[string]any{"type": item.Type, "labels": item.Labels, "properties": item.Properties}
	identity := item.NodeID
	if item.Type == "relationship_candidate" {
		identity = strings.Join([]string{item.FromNode, item.Relationship, item.ToNode}, "\x00")
		raw["from"], raw["relationship"], raw["to"] = item.FromNode, item.Relationship, item.ToNode
	} else {
		raw["node_id"] = item.NodeID
	}
	rawJSON, err := json.Marshal(raw)
	if err != nil {
		return storage.MarketOpsDSMGraphProposalRecord{}, err
	}
	sourceRefs, err := json.Marshal(source.Refs)
	if err != nil {
		return storage.MarketOpsDSMGraphProposalRecord{}, err
	}
	lineageRefs, err := json.Marshal(source.Lineage)
	if err != nil {
		return storage.MarketOpsDSMGraphProposalRecord{}, err
	}
	return storage.MarketOpsDSMGraphProposalRecord{
		ProposalID: stableProposalID(source.SourceType, source.ID, source.Version, item.Type, identity),
		TenantID:   cfg.TenantID, AppID: defaultAppID, Domain: defaultDomain, UseCase: defaultUseCase,
		SourceID: "marketops_intelligence", SourceAdapter: "persisted_marketops", Dataset: "marketops_intelligence_graph",
		ProposalSource: source.SourceType, SourceRecordType: source.SourceType, SourceRecordID: source.ID, SourceRecordVersion: source.Version,
		SourceRefsJSON: sourceRefs, LineageRefsJSON: lineageRefs, SubjectSymbol: cfg.Symbol, CandidateType: item.Type,
		NodeID: item.NodeID, FromNode: item.FromNode, Relationship: item.Relationship, ToNode: item.ToNode,
		Labels: append([]string(nil), item.Labels...), PropertiesJSON: properties, RawCandidate: rawJSON,
		Status: storage.MarketOpsDSMGraphProposalStatusProposed,
	}, nil
}

func stableProposalID(parts ...string) string {
	hash := sha256.New()
	for _, part := range append([]string{"marketops.intelligence.graph_proposal.v1"}, parts...) {
		hash.Write([]byte(strings.TrimSpace(part)))
		hash.Write([]byte{0})
	}
	return "graphprop_marketops_intel_v1_" + hex.EncodeToString(hash.Sum(nil))[:24]
}

func (cfg *Config) normalize() {
	cfg.TenantID = strings.TrimSpace(cfg.TenantID)
	cfg.Symbol = strings.ToUpper(strings.TrimSpace(cfg.Symbol))
	selected := map[string]bool{}
	for _, value := range cfg.SourceTypes {
		for _, item := range strings.Split(value, ",") {
			item = strings.TrimSpace(item)
			if item != "" {
				selected[item] = true
			}
		}
	}
	cfg.SourceTypes = cfg.SourceTypes[:0]
	for _, sourceType := range sourceOrder {
		if selected[sourceType] {
			cfg.SourceTypes = append(cfg.SourceTypes, sourceType)
		}
	}
	unknown := make([]string, 0)
	for sourceType := range selected {
		known := false
		for _, allowed := range sourceOrder {
			if sourceType == allowed {
				known = true
				break
			}
		}
		if !known {
			unknown = append(unknown, sourceType)
		}
	}
	sort.Strings(unknown)
	cfg.SourceTypes = append(cfg.SourceTypes, unknown...)
}

func (cfg Config) validate() error {
	if cfg.TenantID == "" || cfg.Symbol == "" {
		return errors.New("tenant-id and symbol are required")
	}
	if cfg.SessionStart.IsZero() || cfg.SessionEnd.IsZero() || cfg.SessionEnd.Before(cfg.SessionStart) {
		return errors.New("session dates are invalid")
	}
	if len(cfg.SourceTypes) == 0 {
		return errors.New("at least one source type is required")
	}
	for _, sourceType := range cfg.SourceTypes {
		known := false
		for _, allowed := range sourceOrder {
			if sourceType == allowed {
				known = true
				break
			}
		}
		if !known {
			return fmt.Errorf("unsupported source type %q", sourceType)
		}
	}
	if cfg.MaxSourceRecords <= 0 || cfg.MaxSourceRecords > 1000 {
		return errors.New("max-source-records must be between 1 and 1000")
	}
	if cfg.MaxProposals <= 0 || cfg.MaxProposals > 5000 {
		return errors.New("max-proposals must be between 1 and 5000")
	}
	return nil
}

func bounded(values []string) []string {
	if len(values) > 50 {
		values = values[:50]
	}
	return append([]string(nil), values...)
}
func assetNodeID(symbol string) string            { return "ticker:" + strings.ToUpper(strings.TrimSpace(symbol)) }
func hypothesisNodeID(key, version string) string { return "hypothesis:" + key + ":" + version }
func sourceNodeID(sourceType, id string) string {
	switch sourceType {
	case storage.MarketOpsOutcomeSourceHypothesisEvaluation:
		return "hypothesis_evaluation:" + id
	case storage.MarketOpsOutcomeSourceOpportunity:
		return "opportunity:" + id
	default:
		return sourceType + ":" + id
	}
}
func date(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format("2006-01-02")
}
func optionalDate(value *time.Time) string {
	if value == nil {
		return ""
	}
	return date(*value)
}
func valueOrUnknown(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	return value
}
