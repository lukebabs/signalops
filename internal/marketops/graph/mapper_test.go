package graph

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

type fakeRepository struct {
	states        []storage.MarketOpsMarketStateRecord
	transitions   []storage.MarketOpsStateTransitionRecord
	definitions   []storage.MarketOpsHypothesisDefinitionRecord
	evaluations   []storage.MarketOpsHypothesisEvaluationRecord
	opportunities []storage.MarketOpsOpportunityRecord
	outcomes      []storage.MarketOpsSignalOutcomeRecord
	writes        map[string]storage.MarketOpsDSMGraphProposalRecord
}

func (f *fakeRepository) ListMarketOpsMarketStates(context.Context, storage.MarketOpsMarketStateFilter) ([]storage.MarketOpsMarketStateRecord, error) {
	return append([]storage.MarketOpsMarketStateRecord(nil), f.states...), nil
}
func (f *fakeRepository) ListMarketOpsStateTransitions(context.Context, storage.MarketOpsStateTransitionFilter) ([]storage.MarketOpsStateTransitionRecord, error) {
	return append([]storage.MarketOpsStateTransitionRecord(nil), f.transitions...), nil
}
func (f *fakeRepository) ListMarketOpsHypothesisDefinitions(context.Context, storage.MarketOpsHypothesisDefinitionFilter) ([]storage.MarketOpsHypothesisDefinitionRecord, error) {
	return append([]storage.MarketOpsHypothesisDefinitionRecord(nil), f.definitions...), nil
}
func (f *fakeRepository) ListMarketOpsHypothesisEvaluations(context.Context, storage.MarketOpsHypothesisEvaluationFilter) ([]storage.MarketOpsHypothesisEvaluationRecord, error) {
	return append([]storage.MarketOpsHypothesisEvaluationRecord(nil), f.evaluations...), nil
}
func (f *fakeRepository) ListMarketOpsOpportunities(context.Context, storage.MarketOpsOpportunityFilter) ([]storage.MarketOpsOpportunityRecord, error) {
	return append([]storage.MarketOpsOpportunityRecord(nil), f.opportunities...), nil
}
func (f *fakeRepository) ListMarketOpsSignalOutcomes(context.Context, storage.MarketOpsSignalOutcomeFilter) ([]storage.MarketOpsSignalOutcomeRecord, error) {
	return append([]storage.MarketOpsSignalOutcomeRecord(nil), f.outcomes...), nil
}
func (f *fakeRepository) UpsertMarketOpsDSMGraphProposal(_ context.Context, record storage.MarketOpsDSMGraphProposalRecord) error {
	if f.writes == nil {
		f.writes = map[string]storage.MarketOpsDSMGraphProposalRecord{}
	}
	f.writes[record.ProposalID] = record
	return nil
}

func TestMapIsDeterministicSourceAwareAndIdempotent(t *testing.T) {
	repo := fixtureRepository()
	cfg := fixtureConfig()
	cfg.DryRun = true
	first, err := Map(context.Background(), repo, cfg)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Map(context.Background(), repo, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first.ProposalIDs, second.ProposalIDs) {
		t.Fatalf("proposal ids changed: %v != %v", first.ProposalIDs, second.ProposalIDs)
	}
	if first.Proposals != 13 || first.Written != 0 || len(repo.writes) != 0 {
		t.Fatalf("dry-run result = %+v writes=%d", first, len(repo.writes))
	}

	cfg.DryRun = false
	written, err := Map(context.Background(), repo, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if written.Written != written.Proposals || len(repo.writes) != written.Proposals {
		t.Fatalf("write result = %+v unique writes=%d", written, len(repo.writes))
	}
	again, err := Map(context.Background(), repo, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.writes) != written.Proposals || !reflect.DeepEqual(written.ProposalIDs, again.ProposalIDs) {
		t.Fatalf("idempotency failed: writes=%d first=%v second=%v", len(repo.writes), written.ProposalIDs, again.ProposalIDs)
	}
	for _, proposal := range repo.writes {
		if proposal.ProposalSource == "" || proposal.ProposalSource == storage.MarketOpsGraphProposalSourceDSMSignal {
			t.Fatalf("proposal source = %q", proposal.ProposalSource)
		}
		if proposal.ArtifactID != "" || proposal.SignalID != "" || proposal.Severity != "" || proposal.Confidence != 0 {
			t.Fatalf("fabricated legacy fields: %+v", proposal)
		}
	}
}

func TestMapHonorsCapsAndMaturedOutcomeRule(t *testing.T) {
	repo := fixtureRepository()
	cfg := fixtureConfig()
	cfg.MaxSourceRecords = 2
	cfg.MaxProposals = 2
	result, err := Map(context.Background(), repo, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result.SourceRecords != 2 || result.Proposals != 2 {
		t.Fatalf("capped result = %+v", result)
	}
	if result.SkippedReasons["max_source_records_reached"] == 0 || result.SkippedReasons["max_proposals_reached"] == 0 {
		t.Fatalf("missing cap reasons = %+v", result.SkippedReasons)
	}

	repo = fixtureRepository()
	repo.states, repo.transitions, repo.definitions, repo.evaluations, repo.opportunities = nil, nil, nil, nil, nil
	cfg = fixtureConfig()
	cfg.SourceTypes = []string{storage.MarketOpsGraphProposalSourceOutcome}
	result, err = Map(context.Background(), repo, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result.SourceCounts[storage.MarketOpsGraphProposalSourceOutcome] != 1 || result.SkippedReasons["outcome_not_matured"] != 1 {
		t.Fatalf("outcome result = %+v", result)
	}
}

func TestMapReportsTruthfulZeroCandidateResult(t *testing.T) {
	result, err := Map(context.Background(), &fakeRepository{}, fixtureConfig())
	if err != nil {
		t.Fatal(err)
	}
	if result.Proposals != 0 || result.SkippedReasons["no_source_records"] != 1 {
		t.Fatalf("zero result = %+v", result)
	}
}

func fixtureConfig() Config {
	return Config{
		TenantID: "tenant-1", Symbol: "AAPL",
		SessionStart:     time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		SessionEnd:       time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC),
		SourceTypes:      append([]string(nil), sourceOrder...),
		MaxSourceRecords: 100, MaxProposals: 100, DryRun: true,
	}
}

func fixtureRepository() *fakeRepository {
	session := time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)
	matured := session.AddDate(0, 0, 2)
	return &fakeRepository{
		states:        []storage.MarketOpsMarketStateRecord{{MarketStateID: "state-1", TenantID: "tenant-1", AppID: "marketops", AssetID: "ticker:AAPL", Symbol: "AAPL", SessionDate: session, StateSchemaVersion: "v1", FeatureObservationIDs: []string{"feature-1"}, CompletenessRatio: .14, QualityState: "partial"}},
		transitions:   []storage.MarketOpsStateTransitionRecord{{TransitionID: "transition-1", TenantID: "tenant-1", AppID: "marketops", Symbol: "AAPL", SessionDate: session, CurrentStateID: "state-1", BaselineStateID: "state-0", FeatureKey: "return_1d", FeatureVersion: "v1", TransitionType: "absolute_difference", Direction: "up", QualityState: "valid"}},
		definitions:   []storage.MarketOpsHypothesisDefinitionRecord{{TenantID: "tenant-1", HypothesisKey: "pinning", HypothesisVersion: "v1", Domain: "options", Direction: "neutral", LifecycleStatus: "research"}},
		evaluations:   []storage.MarketOpsHypothesisEvaluationRecord{{EvaluationID: "evaluation-1", TenantID: "tenant-1", AppID: "marketops", HypothesisKey: "pinning", HypothesisVersion: "v1", MarketStateID: "state-1", Symbol: "AAPL", SessionDate: session, Eligible: true, EvidenceIDs: []string{"evidence-1"}}},
		opportunities: []storage.MarketOpsOpportunityRecord{{OpportunityID: "opportunity-1", TenantID: "tenant-1", AppID: "marketops", Symbol: "AAPL", OpenedSessionDate: session, Direction: "up", Horizon: "5d", LifecycleStatus: "emerging", HypothesisEvaluationIDs: []string{"evaluation-1"}, SupportingEvidenceIDs: []string{"evidence-1"}, Version: 1, ResearchOnly: true}},
		outcomes: []storage.MarketOpsSignalOutcomeRecord{
			{OutcomeID: "outcome-1", TenantID: "tenant-1", AppID: "marketops", SourceType: storage.MarketOpsOutcomeSourceHypothesisEvaluation, SourceID: "evaluation-1", Symbol: "AAPL", Direction: "up", OriginSessionDate: session, MaturedSessionDate: &matured, OutcomeStatus: storage.MarketOpsOutcomeMatured, CalculationVersion: "v1"},
			{OutcomeID: "outcome-pending", TenantID: "tenant-1", AppID: "marketops", SourceType: storage.MarketOpsOutcomeSourceOpportunity, SourceID: "opportunity-1", Symbol: "AAPL", OriginSessionDate: session, OutcomeStatus: storage.MarketOpsOutcomePending, CalculationVersion: "v1"},
		},
	}
}
