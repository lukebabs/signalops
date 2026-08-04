package opportunities

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestBuildGroupsCorroborationControlsOverlapAndConflicts(t *testing.T) {
	definitions := []storage.MarketOpsHypothesisDefinitionRecord{
		definition("H001", "momentum_exhaustion", "downside", storage.MarketOpsHypothesisLifecycleResearch),
		definition("H001B", "momentum_exhaustion", "downside", storage.MarketOpsHypothesisLifecycleResearch),
		definition("H006", "divergence", "conditional", storage.MarketOpsHypothesisLifecycleResearch),
		definition("H007", "option_positioning", "conditional", storage.MarketOpsHypothesisLifecycleResearch),
	}
	evaluations := []storage.MarketOpsHypothesisEvaluationRecord{
		evaluation("eval-h001", "H001", .8, "downside", "evidence-1"),
		evaluation("eval-h001b", "H001B", .6, "downside", "evidence-overlap"),
		evaluation("eval-h006", "H006", .75, "downside", "evidence-2"),
		evaluation("eval-h007", "H007", .7, "upside", "evidence-conflict"),
	}
	result, err := Build("run-one", definitions, evaluations)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Opportunities) != 2 || result.OverlapSuppressed != 1 || result.ConflictLinks != 3 {
		t.Fatalf("result=%+v", result)
	}
	downside := findDirection(result.Opportunities, "downside")
	if downside.OpportunityID == "" || downside.LifecycleStatus != storage.MarketOpsOpportunityActive || len(downside.HypothesisEvaluationIDs) != 2 || len(downside.ConflictingEvaluationIDs) != 1 || downside.ConflictScore <= 0 || !downside.ResearchOnly {
		t.Fatalf("downside=%+v", downside)
	}
	if contains(downside.HypothesisEvaluationIDs, "eval-h001b") || !contains(downside.SupportingEvidenceIDs, "evidence-1") || !contains(downside.SupportingEvidenceIDs, "evidence-2") {
		t.Fatalf("overlap/evidence=%+v", downside)
	}
	second, err := Build("run-two", definitions, evaluations)
	if err != nil {
		t.Fatal(err)
	}
	if second.Opportunities[0].OpportunityID != result.Opportunities[0].OpportunityID || second.Opportunities[0].DeterministicKey != result.Opportunities[0].DeterministicKey {
		t.Fatalf("identity changed across run ids: first=%+v second=%+v", result.Opportunities[0], second.Opportunities[0])
	}
}

func TestBuildSkipsRejectedNonTriggeredAndUnresolved(t *testing.T) {
	definition := definition("H006", "divergence", "conditional", storage.MarketOpsHypothesisLifecycleResearch)
	ineligible := evaluation("eval-ineligible", "H006", .7, "downside")
	ineligible.Eligible = false
	nonTriggered := evaluation("eval-not-triggered", "H006", .7, "downside")
	nonTriggered.Triggered = false
	unresolved := evaluation("eval-unresolved", "H006", .7, "")
	unresolved.EvaluationPayloadJSON = []byte(`{"checks":{"return_1d":0}}`)
	missingScore := evaluation("eval-missing-score", "H006", .7, "downside")
	missingScore.ConfidenceScore = nil
	result, err := Build("run", []storage.MarketOpsHypothesisDefinitionRecord{definition}, []storage.MarketOpsHypothesisEvaluationRecord{ineligible, nonTriggered, unresolved, missingScore})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Opportunities) != 0 || result.SkippedReasons["ineligible_evaluation"] != 1 || result.SkippedReasons["eligible_not_triggered"] != 1 || result.SkippedReasons["unresolved_direction"] != 1 || result.SkippedReasons["missing_required_score"] != 1 {
		t.Fatalf("result=%+v", result)
	}
}

func TestBuildConvergenceRequiresCorroborationForLargeCompletedSessionMove(t *testing.T) {
	session := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	move := ConvergenceContribution{TenantID: "tenant-local", AssetID: "ticker:NVDA", Symbol: "NVDA", Source: "large_session_move", Direction: "upside", SessionDate: session, Strength: .3, EvidenceIDs: []string{"move-1"}}
	alone, err := BuildConvergence("move-alone", []ConvergenceContribution{move})
	if err != nil {
		t.Fatal(err)
	}
	if len(alone.Opportunities) != 0 || alone.SkippedReasons["insufficient_independent_sources"] != 1 {
		t.Fatalf("standalone move result=%+v", alone)
	}
	result, err := BuildConvergence("move-corroborated", []ConvergenceContribution{move, {TenantID: "tenant-local", AssetID: "ticker:NVDA", Symbol: "NVDA", Source: "risk_reward", Direction: "upside", SessionDate: session, Strength: .7, EvidenceIDs: []string{"rr-1"}}})
	if err != nil {
		t.Fatal(err)
	}
	upside := findDirection(result.Opportunities, "upside")
	if upside.OpportunityID == "" || !contains(upside.SupportingEvidenceIDs, "move-1") || !contains(upside.SupportingEvidenceIDs, "rr-1") {
		t.Fatalf("corroborated move=%+v", upside)
	}
}

func definition(key, domain, direction, status string) storage.MarketOpsHypothesisDefinitionRecord {
	return storage.MarketOpsHypothesisDefinitionRecord{TenantID: "tenant-local", HypothesisKey: key, HypothesisVersion: "v1", Domain: domain, Direction: direction, LifecycleStatus: status}
}

func evaluation(id, key string, trigger float64, direction string, evidence ...string) storage.MarketOpsHypothesisEvaluationRecord {
	quality, persistence, confidence := .9, .7, .8
	payload, _ := json.Marshal(map[string]any{"resolved_direction": direction, "horizon": DefaultHorizon})
	return storage.MarketOpsHypothesisEvaluationRecord{
		EvaluationID: id, TenantID: "tenant-local", AppID: "marketops", HypothesisKey: key,
		HypothesisVersion: "v1", MarketStateID: "state-1", AssetID: "ticker:AAPL", Symbol: "AAPL",
		SessionDate: time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC), AsOfTime: time.Date(2026, 7, 19, 23, 0, 0, 0, time.UTC),
		Eligible: true, Triggered: true, TriggerScore: &trigger, ConfidenceScore: &confidence,
		PersistenceScore: &persistence, QualityScore: &quality, EvidenceIDs: evidence,
		EvaluationPayloadJSON: payload,
	}
}

func findDirection(values []storage.MarketOpsOpportunityRecord, direction string) storage.MarketOpsOpportunityRecord {
	for _, value := range values {
		if value.Direction == direction {
			return value
		}
	}
	return storage.MarketOpsOpportunityRecord{}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func TestBuildConvergenceRequiresIndependentAlignedSources(t *testing.T) {
	session := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	result, err := BuildConvergence("convergence-run", []ConvergenceContribution{
		{TenantID: "tenant-local", AssetID: "ticker:NVDA", Symbol: "NVDA", Source: "risk_reward", Direction: "upside", SessionDate: session, Strength: .7, EvidenceIDs: []string{"rr-1"}},
		{TenantID: "tenant-local", AssetID: "ticker:NVDA", Symbol: "NVDA", Source: "options_flow", Direction: "upside", SessionDate: session, Strength: .8, EvidenceIDs: []string{"options-1"}},
		{TenantID: "tenant-local", AssetID: "ticker:NVDA", Symbol: "NVDA", Source: "exhaustive_reversal", Direction: "downside", SessionDate: session, Strength: .6, EvidenceIDs: []string{"eroc-1"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Opportunities) != 2 {
		t.Fatalf("opportunities=%+v skipped=%+v", result.Opportunities, result.SkippedReasons)
	}
	record := findDirection(result.Opportunities, "upside")
	if record.Version != ConvergenceVersion || record.LifecycleStatus != storage.MarketOpsOpportunityActive || record.Direction != "upside" || record.ConflictScore <= 0 {
		t.Fatalf("record=%+v", record)
	}
	if !contains(record.SupportingEvidenceIDs, "rr-1") || !contains(record.SupportingEvidenceIDs, "options-1") || contains(record.SupportingEvidenceIDs, "eroc-1") {
		t.Fatalf("evidence=%v", record.SupportingEvidenceIDs)
	}
	var payload map[string]any
	if err := json.Unmarshal(record.OpportunityPayloadJSON, &payload); err != nil || payload["scoring_version"] != "marketops.convergence_opportunity.v2" {
		t.Fatalf("payload=%s err=%v", record.OpportunityPayloadJSON, err)
	}
	mixed := findDirection(result.Opportunities, "non_directional")
	if mixed.OpportunityID == "" || mixed.ConflictScore != 1 || mixed.ConfidenceScore != 0 {
		t.Fatalf("mixed=%+v", mixed)
	}
	if result.SkippedReasons["insufficient_independent_sources"] != 1 {
		t.Fatalf("skipped=%+v", result.SkippedReasons)
	}
}
