package opportunities

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	marketopsstate "github.com/lukebabs/signalops/internal/marketops/state"
	"github.com/lukebabs/signalops/internal/storage"
)

const (
	Version        = 1
	DefaultHorizon = "5_to_20_sessions"
)

type BuildResult struct {
	Opportunities     []storage.MarketOpsOpportunityRecord
	Evaluations       int
	Triggered         int
	OverlapSuppressed int
	ConflictLinks     int
	SkippedReasons    map[string]int
}

type candidate struct {
	evaluation storage.MarketOpsHypothesisEvaluationRecord
	definition storage.MarketOpsHypothesisDefinitionRecord
	direction  string
	horizon    string
	family     string
}

type groupKey struct {
	tenantID, assetID, symbol, session, direction, horizon string
}

func Build(runID string, definitions []storage.MarketOpsHypothesisDefinitionRecord, evaluations []storage.MarketOpsHypothesisEvaluationRecord) (BuildResult, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return BuildResult{}, fmt.Errorf("opportunity build run id is required")
	}
	result := BuildResult{Evaluations: len(evaluations), SkippedReasons: map[string]int{}}
	definitionByID := map[string]storage.MarketOpsHypothesisDefinitionRecord{}
	for _, definition := range definitions {
		definitionByID[definitionID(definition.TenantID, definition.HypothesisKey, definition.HypothesisVersion)] = definition
	}
	candidates := []candidate{}
	invalidating := []storage.MarketOpsHypothesisEvaluationRecord{}
	for _, evaluation := range evaluations {
		if evaluation.AppID != "" && evaluation.AppID != "marketops" {
			result.SkippedReasons["wrong_app_id"]++
			continue
		}
		if evaluation.Invalidated {
			invalidating = append(invalidating, evaluation)
			result.SkippedReasons["invalidated_evaluation"]++
			continue
		}
		if !evaluation.Eligible {
			result.SkippedReasons["ineligible_evaluation"]++
			continue
		}
		if !evaluation.Triggered {
			result.SkippedReasons["eligible_not_triggered"]++
			continue
		}
		definition, ok := definitionByID[definitionID(evaluation.TenantID, evaluation.HypothesisKey, evaluation.HypothesisVersion)]
		if !ok {
			result.SkippedReasons["missing_hypothesis_definition"]++
			continue
		}
		if definition.LifecycleStatus != storage.MarketOpsHypothesisLifecycleResearch {
			result.SkippedReasons["non_research_hypothesis_definition"]++
			continue
		}
		if evaluation.TriggerScore == nil || evaluation.ConfidenceScore == nil {
			result.SkippedReasons["missing_required_score"]++
			continue
		}
		direction, horizon, family := opportunityProfile(evaluation, definition)
		if !validDirection(direction) {
			result.SkippedReasons["unresolved_direction"]++
			continue
		}
		if strings.TrimSpace(horizon) == "" {
			result.SkippedReasons["unresolved_horizon"]++
			continue
		}
		result.Triggered++
		candidates = append(candidates, candidate{evaluation: evaluation, definition: definition, direction: direction, horizon: horizon, family: family})
	}

	groups := map[groupKey][]candidate{}
	for _, item := range candidates {
		key := groupKey{item.evaluation.TenantID, item.evaluation.AssetID, item.evaluation.Symbol, item.evaluation.SessionDate.Format("2006-01-02"), item.direction, item.horizon}
		groups[key] = append(groups[key], item)
	}
	for key, members := range groups {
		selected, suppressed := strongestPerFamily(members)
		result.OverlapSuppressed += len(suppressed)
		rawConflicts := conflictingCandidates(key, candidates)
		conflicts, _ := strongestPerFamily(rawConflicts)
		result.ConflictLinks += len(conflicts)
		record, err := buildRecord(runID, key, selected, suppressed, conflicts, invalidating)
		if err != nil {
			return result, err
		}
		result.Opportunities = append(result.Opportunities, record)
	}
	sort.Slice(result.Opportunities, func(i, j int) bool {
		if result.Opportunities[i].OpportunityScore == result.Opportunities[j].OpportunityScore {
			return result.Opportunities[i].OpportunityID < result.Opportunities[j].OpportunityID
		}
		return result.Opportunities[i].OpportunityScore > result.Opportunities[j].OpportunityScore
	})
	return result, nil
}

func buildRecord(runID string, key groupKey, members, suppressed, conflicts []candidate, invalidating []storage.MarketOpsHypothesisEvaluationRecord) (storage.MarketOpsOpportunityRecord, error) {
	session, err := time.Parse("2006-01-02", key.session)
	if err != nil {
		return storage.MarketOpsOpportunityRecord{}, err
	}
	evaluationIDs, evidenceIDs, domains, keys := []string{}, []string{}, []string{}, []string{}
	contributions := []map[string]any{}
	researchOnly := true
	var maxScore, qualityTotal, persistenceTotal, confidenceTotal float64
	for _, item := range members {
		evaluation := item.evaluation
		evaluationIDs = append(evaluationIDs, evaluation.EvaluationID)
		evidenceIDs = append(evidenceIDs, evaluation.EvidenceIDs...)
		domains = append(domains, item.definition.Domain)
		keys = append(keys, evaluation.HypothesisKey)
		trigger, confidence := score(evaluation.TriggerScore), score(evaluation.ConfidenceScore)
		maxScore = math.Max(maxScore, trigger)
		qualityTotal += score(evaluation.QualityScore)
		persistenceTotal += score(evaluation.PersistenceScore)
		confidenceTotal += confidence
		contributions = append(contributions, map[string]any{
			"evaluation_id": evaluation.EvaluationID, "hypothesis_key": evaluation.HypothesisKey,
			"hypothesis_version": evaluation.HypothesisVersion, "domain": item.definition.Domain,
			"trigger_score": evaluation.TriggerScore, "confidence_score": evaluation.ConfidenceScore,
			"quality_score": evaluation.QualityScore,
		})
	}
	count := float64(len(members))
	diversity := clamp(float64(len(unique(domains))) / 3)
	corroboration := clamp((count - 1) / 2)
	conflictScore := clamp(float64(len(conflicts)) / math.Max(count+float64(len(conflicts)), 1))
	quality, persistence, confidence := qualityTotal/count, persistenceTotal/count, confidenceTotal/count
	opportunityScore := clamp(.35*maxScore + .2*quality + .15*persistence + .15*diversity + .15*corroboration - .25*conflictScore)
	confidenceScore := clamp(confidence*(.75+.25*diversity) - .25*conflictScore)
	lifecycle := storage.MarketOpsOpportunityEmerging
	if len(members) >= 2 && len(unique(domains)) >= 2 && conflictScore < .5 {
		lifecycle = storage.MarketOpsOpportunityActive
	}
	conflictIDs := candidateEvaluationIDs(conflicts)
	invalidatingEvidence := invalidatingEvidenceIDs(key, invalidating)
	identity, err := marketopsstate.NewIdentity(marketopsstate.IdentityOpportunity, key.tenantID, key.assetID, key.session, key.direction, key.horizon, fmt.Sprint(Version))
	if err != nil {
		return storage.MarketOpsOpportunityRecord{}, err
	}
	sort.Strings(keys)
	summary := opportunitySummary(key.symbol, key.direction, key.horizon, keys, len(unique(domains)), len(conflicts))
	payload, _ := json.Marshal(map[string]any{
		"scoring_version": "marketops.opportunity_score.v1", "contributions": contributions,
		"overlap_suppressed_evaluation_ids": candidateEvaluationIDs(suppressed),
		"conflicting_evaluation_ids":        conflictIDs, "hypothesis_families": unique(domains),
		"research_only": researchOnly,
	})
	return storage.MarketOpsOpportunityRecord{
		OpportunityID: identity.ID, TenantID: key.tenantID, AppID: "marketops", AssetID: key.assetID,
		Symbol: key.symbol, OpenedSessionDate: session, LastEvaluatedDate: session, Direction: key.direction,
		Horizon: key.horizon, LifecycleStatus: lifecycle, OpportunityScore: opportunityScore,
		ConfidenceScore: confidenceScore, DomainDiversityScore: diversity, ConflictScore: conflictScore,
		HypothesisEvaluationIDs: unique(evaluationIDs), ConflictingEvaluationIDs: conflictIDs,
		SignalIDs: []string{}, SupportingEvidenceIDs: unique(evidenceIDs),
		InvalidatingEvidenceIDs: invalidatingEvidence, Summary: summary, OpportunityPayloadJSON: payload,
		Version: Version, ResearchOnly: researchOnly, BuildRunID: runID, DeterministicKey: identity.DeterministicKey,
	}, nil
}

func strongestPerFamily(values []candidate) ([]candidate, []candidate) {
	strongest := map[string]candidate{}
	suppressed := []candidate{}
	for _, item := range values {
		current, exists := strongest[item.family]
		if !exists || greater(item.evaluation, current.evaluation) {
			if exists {
				suppressed = append(suppressed, current)
			}
			strongest[item.family] = item
		} else {
			suppressed = append(suppressed, item)
		}
	}
	selected := make([]candidate, 0, len(strongest))
	for _, item := range strongest {
		selected = append(selected, item)
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].evaluation.EvaluationID < selected[j].evaluation.EvaluationID })
	return selected, suppressed
}

func greater(left, right storage.MarketOpsHypothesisEvaluationRecord) bool {
	leftScore, rightScore := score(left.TriggerScore), score(right.TriggerScore)
	return leftScore > rightScore || (leftScore == rightScore && left.EvaluationID < right.EvaluationID)
}

func conflictingCandidates(key groupKey, values []candidate) []candidate {
	out := []candidate{}
	for _, item := range values {
		if item.evaluation.TenantID != key.tenantID || item.evaluation.AssetID != key.assetID || item.evaluation.SessionDate.Format("2006-01-02") != key.session || item.horizon != key.horizon {
			continue
		}
		if opposite(key.direction, item.direction) {
			out = append(out, item)
		}
	}
	return out
}

func invalidatingEvidenceIDs(key groupKey, values []storage.MarketOpsHypothesisEvaluationRecord) []string {
	out := []string{}
	for _, evaluation := range values {
		if evaluation.TenantID == key.tenantID && evaluation.AssetID == key.assetID && evaluation.SessionDate.Format("2006-01-02") == key.session {
			out = append(out, evaluation.EvidenceIDs...)
		}
	}
	return unique(out)
}

func opportunityProfile(evaluation storage.MarketOpsHypothesisEvaluationRecord, definition storage.MarketOpsHypothesisDefinitionRecord) (string, string, string) {
	payload := struct {
		ResolvedDirection string         `json:"resolved_direction"`
		Horizon           string         `json:"horizon"`
		HypothesisFamily  string         `json:"hypothesis_family"`
		Checks            map[string]any `json:"checks"`
	}{}
	_ = json.Unmarshal(evaluation.EvaluationPayloadJSON, &payload)
	direction := strings.TrimSpace(payload.ResolvedDirection)
	if direction == "" {
		direction = strings.TrimSpace(definition.Direction)
	}
	if direction == "conditional" {
		switch evaluation.HypothesisKey {
		case "H006":
			move, _ := payload.Checks["return_1d"].(float64)
			if move > 0 {
				direction = "downside"
			} else if move < 0 {
				direction = "upside"
			}
		case "H007":
			optionType, _ := payload.Checks["option_type"].(string)
			if optionType == "put" {
				direction = "downside"
			} else if optionType == "call" {
				direction = "upside"
			}
		}
	}
	horizon := strings.TrimSpace(payload.Horizon)
	if horizon == "" {
		horizon = DefaultHorizon
	}
	family := strings.TrimSpace(payload.HypothesisFamily)
	if family == "" {
		family = strings.TrimSpace(definition.Domain)
	}
	if family == "" {
		family = evaluation.HypothesisKey
	}
	return direction, horizon, family
}

func opportunitySummary(symbol, direction, horizon string, hypothesisKeys []string, domains, conflicts int) string {
	conflictText := "no opposing triggered hypothesis"
	if conflicts > 0 {
		conflictText = fmt.Sprintf("%d opposing triggered hypothesis evaluation(s)", conflicts)
	}
	return fmt.Sprintf("%s %s %s opportunity from %s across %d independent domain(s); %s.", symbol, direction, horizon, strings.Join(hypothesisKeys, ", "), domains, conflictText)
}

func candidateEvaluationIDs(values []candidate) []string {
	out := make([]string, 0, len(values))
	for _, item := range values {
		out = append(out, item.evaluation.EvaluationID)
	}
	return unique(out)
}

func definitionID(tenantID, key, version string) string {
	return tenantID + "\x00" + key + "\x00" + version
}
func score(value *float64) float64 {
	if value == nil {
		return 0
	}
	return clamp(*value)
}
func validDirection(value string) bool {
	return value == "upside" || value == "downside" || value == "non_directional"
}
func opposite(left, right string) bool {
	return (left == "upside" && right == "downside") || (left == "downside" && right == "upside")
}
func clamp(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}
func unique(values []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

// ConvergenceVersion identifies opportunities built from persisted active
// algorithms rather than the legacy hypothesis-trigger pilot.
const ConvergenceVersion = 2

type ConvergenceContribution struct {
	TenantID, AssetID, Symbol, Source, Direction string
	SessionDate                                  time.Time
	Strength                                     float64
	EvidenceIDs                                  []string
}

// BuildConvergence emits one research-only opportunity when at least two
// independent persisted sources agree on a direction for a completed session.
// Score is axiomatic: 50% agreement (two sources), plus up to 50% average
// source strength. Opposing sources are represented as a conflict penalty.
func BuildConvergence(runID string, inputs []ConvergenceContribution) (BuildResult, error) {
	if strings.TrimSpace(runID) == "" {
		return BuildResult{}, fmt.Errorf("opportunity build run id is required")
	}
	result := BuildResult{SkippedReasons: map[string]int{}}
	groups := map[groupKey][]ConvergenceContribution{}
	for _, input := range inputs {
		input.Symbol = strings.ToUpper(strings.TrimSpace(input.Symbol))
		input.Direction = strings.TrimSpace(input.Direction)
		input.Source = strings.TrimSpace(input.Source)
		if input.Symbol == "" || input.Source == "" || input.SessionDate.IsZero() || !validDirection(input.Direction) {
			result.SkippedReasons["invalid_convergence_contribution"]++
			continue
		}
		if input.AssetID == "" {
			input.AssetID = "ticker:" + input.Symbol
		}
		key := groupKey{input.TenantID, input.AssetID, input.Symbol, input.SessionDate.UTC().Format("2006-01-02"), input.Direction, DefaultHorizon}
		groups[key] = append(groups[key], input)
	}
	for key, values := range groups {
		bySource := map[string]ConvergenceContribution{}
		for _, value := range values {
			if current, ok := bySource[value.Source]; !ok || value.Strength > current.Strength {
				bySource[value.Source] = value
			}
		}
		if len(bySource) < 2 {
			result.SkippedReasons["insufficient_independent_sources"]++
			continue
		}
		members := make([]ConvergenceContribution, 0, len(bySource))
		for _, value := range bySource {
			members = append(members, value)
		}
		sort.Slice(members, func(i, j int) bool { return members[i].Source < members[j].Source })
		strength, evidence, contributions := 0.0, []string{}, []map[string]any{}
		for _, value := range members {
			strength += clamp(value.Strength)
			evidence = append(evidence, value.EvidenceIDs...)
			contributions = append(contributions, map[string]any{"source": value.Source, "direction": value.Direction, "strength": clamp(value.Strength), "evidence_ids": unique(value.EvidenceIDs)})
		}
		conflictCount := 0
		for other, opposing := range groups {
			if other.tenantID == key.tenantID && other.assetID == key.assetID && other.session == key.session && opposite(key.direction, other.direction) {
				seen := map[string]bool{}
				for _, value := range opposing {
					seen[value.Source] = true
				}
				conflictCount += len(seen)
			}
		}
		average := strength / float64(len(members))
		conflict := clamp(float64(conflictCount) / float64(len(members)+conflictCount))
		score := clamp(.5 + .5*average - .25*conflict)
		identity, err := marketopsstate.NewIdentity(marketopsstate.IdentityOpportunity, key.tenantID, key.assetID, key.session, key.direction, key.horizon, fmt.Sprint(ConvergenceVersion))
		if err != nil {
			return result, err
		}
		payload, _ := json.Marshal(map[string]any{"scoring_version": "marketops.convergence_opportunity.v2", "contributions": contributions, "agreement_sources": len(members), "conflict_sources": conflictCount, "score_formula": "0.50 agreement + 0.50 average_strength - 0.25 conflict_ratio", "research_only": true})
		result.Opportunities = append(result.Opportunities, storage.MarketOpsOpportunityRecord{OpportunityID: identity.ID, TenantID: key.tenantID, AppID: "marketops", AssetID: key.assetID, Symbol: key.symbol, OpenedSessionDate: members[0].SessionDate, LastEvaluatedDate: members[0].SessionDate, Direction: key.direction, Horizon: key.horizon, LifecycleStatus: storage.MarketOpsOpportunityActive, OpportunityScore: score, ConfidenceScore: clamp(average * (1 - conflict)), DomainDiversityScore: clamp(float64(len(members)) / 4), ConflictScore: conflict, SupportingEvidenceIDs: unique(evidence), Summary: fmt.Sprintf("%s %s convergence: %d independent sources agree; %d opposing sources.", key.symbol, key.direction, len(members), conflictCount), OpportunityPayloadJSON: payload, Version: ConvergenceVersion, ResearchOnly: true, BuildRunID: runID, DeterministicKey: identity.DeterministicKey})
	}
	// Mixed conviction is an analyst-review condition: material, independent
	// sources disagree in the same completed session. It is non-directional.
	mixedGroups := map[groupKey][]ConvergenceContribution{}
	for key, values := range groups {
		if key.direction != "upside" && key.direction != "downside" {
			continue
		}
		mixedKey := key
		mixedKey.direction = "non_directional"
		for _, value := range values {
			if clamp(value.Strength) >= .20 {
				mixedGroups[mixedKey] = append(mixedGroups[mixedKey], value)
			}
		}
	}
	for key, values := range mixedGroups {
		byDirection := map[string]map[string]ConvergenceContribution{"upside": {}, "downside": {}}
		for _, value := range values {
			current, exists := byDirection[value.Direction][value.Source]
			if !exists || value.Strength > current.Strength {
				byDirection[value.Direction][value.Source] = value
			}
		}
		if len(byDirection["upside"]) == 0 || len(byDirection["downside"]) == 0 {
			continue
		}
		members, evidence, contributions := []ConvergenceContribution{}, []string{}, []map[string]any{}
		for _, direction := range []string{"upside", "downside"} {
			for _, value := range byDirection[direction] {
				members = append(members, value)
			}
		}
		sort.Slice(members, func(i, j int) bool { return members[i].Source < members[j].Source })
		strength := 0.0
		for _, value := range members {
			strength += clamp(value.Strength)
			evidence = append(evidence, value.EvidenceIDs...)
			contributions = append(contributions, map[string]any{"source": value.Source, "direction": value.Direction, "strength": clamp(value.Strength), "evidence_ids": unique(value.EvidenceIDs)})
		}
		average := strength / float64(len(members))
		identity, err := marketopsstate.NewIdentity(marketopsstate.IdentityOpportunity, key.tenantID, key.assetID, key.session, key.direction, key.horizon, fmt.Sprint(ConvergenceVersion))
		if err != nil {
			return result, err
		}
		payload, _ := json.Marshal(map[string]any{"scoring_version": "marketops.convergence_opportunity.v2", "mixed_conviction": true, "contributions": contributions, "agreement_sources": 0, "conflict_sources": len(members), "score_formula": "average material opposing source strength", "research_only": true})
		result.Opportunities = append(result.Opportunities, storage.MarketOpsOpportunityRecord{OpportunityID: identity.ID, TenantID: key.tenantID, AppID: "marketops", AssetID: key.assetID, Symbol: key.symbol, OpenedSessionDate: members[0].SessionDate, LastEvaluatedDate: members[0].SessionDate, Direction: key.direction, Horizon: key.horizon, LifecycleStatus: storage.MarketOpsOpportunityActive, OpportunityScore: average, ConfidenceScore: 0, DomainDiversityScore: clamp(float64(len(members)) / 4), ConflictScore: 1, SupportingEvidenceIDs: unique(evidence), Summary: fmt.Sprintf("%s mixed-conviction review: %d material sources disagree on direction.", key.symbol, len(members)), OpportunityPayloadJSON: payload, Version: ConvergenceVersion, ResearchOnly: true, BuildRunID: runID, DeterministicKey: identity.DeterministicKey})
	}
	sort.Slice(result.Opportunities, func(i, j int) bool {
		return result.Opportunities[i].OpportunityScore > result.Opportunities[j].OpportunityScore
	})
	return result, nil
}
