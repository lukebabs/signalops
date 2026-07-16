package proposals

import (
	"context"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

type fakeProposalRepository struct {
	results        []storage.AlgorithmResultRecord
	proposals      []storage.AlgorithmSignalProposalRecord
	lastFilter     storage.AlgorithmResultFilter
	insertAttempts int
}

func (f *fakeProposalRepository) UpsertAlgorithmDefinition(context.Context, storage.AlgorithmDefinitionRecord) error {
	return nil
}
func (f *fakeProposalRepository) ListAlgorithmDefinitions(context.Context, storage.AlgorithmDefinitionFilter) ([]storage.AlgorithmDefinitionRecord, error) {
	return nil, nil
}
func (f *fakeProposalRepository) GetAlgorithmDefinition(context.Context, string, string) (storage.AlgorithmDefinitionRecord, error) {
	return storage.AlgorithmDefinitionRecord{}, storage.ErrNotFound
}
func (f *fakeProposalRepository) UpsertAlgorithmExecutionRequest(context.Context, storage.AlgorithmExecutionRequestRecord) error {
	return nil
}
func (f *fakeProposalRepository) ListAlgorithmExecutionRequests(context.Context, storage.AlgorithmExecutionRequestFilter) ([]storage.AlgorithmExecutionRequestRecord, error) {
	return nil, nil
}
func (f *fakeProposalRepository) GetAlgorithmExecutionRequest(context.Context, string, string) (storage.AlgorithmExecutionRequestRecord, error) {
	return storage.AlgorithmExecutionRequestRecord{}, storage.ErrNotFound
}
func (f *fakeProposalRepository) InsertAlgorithmResult(context.Context, storage.AlgorithmResultRecord) error {
	return nil
}
func (f *fakeProposalRepository) ListAlgorithmResults(_ context.Context, filter storage.AlgorithmResultFilter) ([]storage.AlgorithmResultRecord, error) {
	f.lastFilter = filter
	return f.results, nil
}
func (f *fakeProposalRepository) GetAlgorithmResult(context.Context, string, string) (storage.AlgorithmResultRecord, error) {
	return storage.AlgorithmResultRecord{}, storage.ErrNotFound
}
func (f *fakeProposalRepository) InsertAlgorithmSignalProposal(_ context.Context, record storage.AlgorithmSignalProposalRecord) (bool, error) {
	f.insertAttempts++
	for _, existing := range f.proposals {
		if existing.TenantID == record.TenantID && existing.ProposalID == record.ProposalID {
			return false, nil
		}
	}
	f.proposals = append(f.proposals, record)
	return true, nil
}
func (f *fakeProposalRepository) ListAlgorithmSignalProposals(context.Context, storage.AlgorithmSignalProposalFilter) ([]storage.AlgorithmSignalProposalRecord, error) {
	return f.proposals, nil
}
func (f *fakeProposalRepository) GetAlgorithmSignalProposal(context.Context, string, string) (storage.AlgorithmSignalProposalRecord, error) {
	return storage.AlgorithmSignalProposalRecord{}, storage.ErrNotFound
}
func (f *fakeProposalRepository) SummarizeAlgorithmSignalProposals(context.Context, storage.AlgorithmSignalProposalFilter) (storage.AlgorithmSignalProposalSummaryRecord, error) {
	return storage.AlgorithmSignalProposalSummaryRecord{}, nil
}
func (f *fakeProposalRepository) MutateAlgorithmSignalProposal(context.Context, storage.AlgorithmSignalProposalMutation) (storage.AlgorithmSignalProposalRecord, error) {
	return storage.AlgorithmSignalProposalRecord{}, storage.ErrNotFound
}

func (f *fakeProposalRepository) ListAlgorithmSignalMaterializations(context.Context, storage.AlgorithmSignalMaterializationFilter) ([]storage.AlgorithmSignalMaterializationRecord, error) {
	return nil, nil
}

func (f *fakeProposalRepository) GetAlgorithmSignalMaterialization(context.Context, string, string) (storage.AlgorithmSignalMaterializationRecord, error) {
	return storage.AlgorithmSignalMaterializationRecord{}, storage.ErrNotFound
}

func TestGenerateCreatesStableSignalProposals(t *testing.T) {
	now := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	repo := &fakeProposalRepository{results: []storage.AlgorithmResultRecord{
		{AlgorithmResultID: "algres-1", TenantID: "tenant-local", AlgorithmID: "signalops.algorithms.ruptures_change_point_v1", AlgorithmVersion: "v1", ExecutionRequestID: "algexec-1", ResultType: "change_point_score", Score: 2.5, Confidence: 0.9, Severity: "high", ResultPayloadJSON: []byte(`{"symbol":"AAPL"}`), SourceEventIDs: []string{"evt-1"}, EvidenceRefs: []string{"normalized_event:evt-1"}, CorrelationID: "corr-1", CreatedAt: now},
	}}
	result, err := Generate(context.Background(), repo, Config{TenantID: "tenant-local", ExecutionRequestID: "algexec-1", MinConfidence: 0.5, CreatedBy: "analyst-1"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Scanned != 1 || result.Proposed != 1 || result.Skipped != 0 || len(repo.proposals) != 1 {
		t.Fatalf("result=%+v proposals=%d", result, len(repo.proposals))
	}
	proposal := repo.proposals[0]
	if proposal.ProposalID == "" || proposal.ProposedSignalType != "signalops.algorithm.change_point_candidate" || proposal.Status != storage.AlgorithmSignalProposalStatusProposed || proposal.CreatedBy != "analyst-1" {
		t.Fatalf("proposal=%+v", proposal)
	}
	firstID := proposal.ProposalID
	result, err = Generate(context.Background(), repo, Config{TenantID: "tenant-local", ExecutionRequestID: "algexec-1", MinConfidence: 0.5, CreatedBy: "analyst-1"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Proposed != 0 || result.Skipped != 1 || len(repo.proposals) != 1 || repo.proposals[0].ProposalID != firstID || repo.insertAttempts != 2 {
		t.Fatalf("idempotency result=%+v proposals=%d first=%s attempts=%d", result, len(repo.proposals), repo.proposals[0].ProposalID, repo.insertAttempts)
	}
}

func TestGenerateSkipsLowConfidenceAndUnsupportedResults(t *testing.T) {
	repo := &fakeProposalRepository{results: []storage.AlgorithmResultRecord{
		{AlgorithmResultID: "algres-low", TenantID: "tenant-local", AlgorithmID: "signalops.algorithms.zscore_anomaly_v1", AlgorithmVersion: "v1", ExecutionRequestID: "algexec-1", ResultType: "z_score", Score: 1.2, Confidence: 0.2, Severity: "low", CorrelationID: "corr-1"},
		{AlgorithmResultID: "algres-unknown", TenantID: "tenant-local", AlgorithmID: "signalops.algorithms.zscore_anomaly_v1", AlgorithmVersion: "v1", ExecutionRequestID: "algexec-1", ResultType: "unknown", Score: 1.2, Confidence: 0.9, Severity: "low", CorrelationID: "corr-1"},
	}}
	result, err := Generate(context.Background(), repo, Config{TenantID: "tenant-local", MinConfidence: 0.5, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if result.Scanned != 2 || result.Proposed != 0 || result.Skipped != 2 || len(repo.proposals) != 0 {
		t.Fatalf("result=%+v proposals=%d", result, len(repo.proposals))
	}
	if repo.lastFilter.TenantID != "tenant-local" || repo.lastFilter.Limit != 10 {
		t.Fatalf("filter=%+v", repo.lastFilter)
	}
}
