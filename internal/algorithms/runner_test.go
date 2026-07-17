package algorithms

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

type fakeAlgorithmRepository struct {
	events            []storage.NormalizedEventLedgerRecord
	requests          []storage.AlgorithmExecutionRequestRecord
	results           []storage.AlgorithmResultRecord
	lastEventFilter   storage.MarketOpsBacktestEventFilter
	insertResultCalls int
}

func (f *fakeAlgorithmRepository) UpsertAlgorithmDefinition(context.Context, storage.AlgorithmDefinitionRecord) error {
	return nil
}
func (f *fakeAlgorithmRepository) ListAlgorithmDefinitions(context.Context, storage.AlgorithmDefinitionFilter) ([]storage.AlgorithmDefinitionRecord, error) {
	return nil, nil
}
func (f *fakeAlgorithmRepository) GetAlgorithmDefinition(context.Context, string, string) (storage.AlgorithmDefinitionRecord, error) {
	return storage.AlgorithmDefinitionRecord{}, storage.ErrNotFound
}
func (f *fakeAlgorithmRepository) UpsertAlgorithmExecutionRequest(_ context.Context, record storage.AlgorithmExecutionRequestRecord) error {
	for i, existing := range f.requests {
		if existing.TenantID == record.TenantID && existing.ExecutionRequestID == record.ExecutionRequestID {
			f.requests[i] = record
			return nil
		}
	}
	f.requests = append(f.requests, record)
	return nil
}
func (f *fakeAlgorithmRepository) ListAlgorithmExecutionRequests(context.Context, storage.AlgorithmExecutionRequestFilter) ([]storage.AlgorithmExecutionRequestRecord, error) {
	return f.requests, nil
}
func (f *fakeAlgorithmRepository) GetAlgorithmExecutionRequest(_ context.Context, tenantID string, executionRequestID string) (storage.AlgorithmExecutionRequestRecord, error) {
	for _, record := range f.requests {
		if record.TenantID == tenantID && record.ExecutionRequestID == executionRequestID {
			return record, nil
		}
	}
	return storage.AlgorithmExecutionRequestRecord{}, storage.ErrNotFound
}
func (f *fakeAlgorithmRepository) InsertAlgorithmResult(_ context.Context, record storage.AlgorithmResultRecord) error {
	f.insertResultCalls++
	for _, existing := range f.results {
		if existing.TenantID == record.TenantID && existing.AlgorithmResultID == record.AlgorithmResultID {
			return nil
		}
	}
	f.results = append(f.results, record)
	return nil
}
func (f *fakeAlgorithmRepository) ListAlgorithmResults(context.Context, storage.AlgorithmResultFilter) ([]storage.AlgorithmResultRecord, error) {
	return f.results, nil
}
func (f *fakeAlgorithmRepository) GetAlgorithmResult(_ context.Context, tenantID string, algorithmResultID string) (storage.AlgorithmResultRecord, error) {
	for _, record := range f.results {
		if record.TenantID == tenantID && record.AlgorithmResultID == algorithmResultID {
			return record, nil
		}
	}
	return storage.AlgorithmResultRecord{}, storage.ErrNotFound
}
func (f *fakeAlgorithmRepository) InsertAlgorithmSignalProposal(context.Context, storage.AlgorithmSignalProposalRecord) (bool, error) {
	return false, nil
}
func (f *fakeAlgorithmRepository) ListAlgorithmSignalProposals(context.Context, storage.AlgorithmSignalProposalFilter) ([]storage.AlgorithmSignalProposalRecord, error) {
	return nil, nil
}
func (f *fakeAlgorithmRepository) GetAlgorithmSignalProposal(context.Context, string, string) (storage.AlgorithmSignalProposalRecord, error) {
	return storage.AlgorithmSignalProposalRecord{}, storage.ErrNotFound
}
func (f *fakeAlgorithmRepository) SummarizeAlgorithmSignalProposals(context.Context, storage.AlgorithmSignalProposalFilter) (storage.AlgorithmSignalProposalSummaryRecord, error) {
	return storage.AlgorithmSignalProposalSummaryRecord{}, nil
}
func (f *fakeAlgorithmRepository) MutateAlgorithmSignalProposal(context.Context, storage.AlgorithmSignalProposalMutation) (storage.AlgorithmSignalProposalRecord, error) {
	return storage.AlgorithmSignalProposalRecord{}, storage.ErrNotFound
}
func (f *fakeAlgorithmRepository) UpsertAlgorithmSignalMaterialization(context.Context, storage.AlgorithmSignalMaterializationRecord) (storage.AlgorithmSignalMaterializationRecord, error) {
	return storage.AlgorithmSignalMaterializationRecord{}, nil
}

func (f *fakeAlgorithmRepository) ListAlgorithmSignalMaterializations(context.Context, storage.AlgorithmSignalMaterializationFilter) ([]storage.AlgorithmSignalMaterializationRecord, error) {
	return nil, nil
}
func (f *fakeAlgorithmRepository) GetAlgorithmSignalMaterialization(context.Context, string, string) (storage.AlgorithmSignalMaterializationRecord, error) {
	return storage.AlgorithmSignalMaterializationRecord{}, storage.ErrNotFound
}
func (f *fakeAlgorithmRepository) ListMarketOpsBacktestNormalizedEvents(_ context.Context, filter storage.MarketOpsBacktestEventFilter) ([]storage.NormalizedEventLedgerRecord, error) {
	f.lastEventFilter = filter
	start := filter.Offset
	if start >= len(f.events) {
		return []storage.NormalizedEventLedgerRecord{}, nil
	}
	end := start + filter.Limit
	if end > len(f.events) {
		end = len(f.events)
	}
	return f.events[start:end], nil
}

func TestRunZScoreWritesDeterministicResults(t *testing.T) {
	now := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	repo := &fakeAlgorithmRepository{events: []storage.NormalizedEventLedgerRecord{
		normalizedEvent("evt-1", "AAPL", 1, now),
		normalizedEvent("evt-2", "AAPL", 2, now.Add(time.Hour)),
		normalizedEvent("evt-3", "AAPL", 9, now.Add(2*time.Hour)),
	}}
	cfg := Config{ExecutionRequestID: "algexec-1", TenantID: "tenant-local", Symbols: []string{"aapl"}, WindowStart: now.Add(-time.Hour), WindowEnd: now.Add(24 * time.Hour), MaxRecords: 10, BatchSize: 2, ZThreshold: 1.5, MinSamples: 3}
	result, err := Run(context.Background(), repo, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExecutionRequest.Status != storage.AlgorithmExecutionStatusSucceeded || result.Metrics.Results != 3 || result.Metrics.UsableSamples != 3 {
		t.Fatalf("result = %+v request=%+v", result.Metrics, result.ExecutionRequest)
	}
	if len(repo.requests) != 1 || repo.requests[0].Status != storage.AlgorithmExecutionStatusSucceeded {
		t.Fatalf("requests = %+v", repo.requests)
	}
	if len(repo.results) != 3 {
		t.Fatalf("results = %d", len(repo.results))
	}
	firstID := repo.results[0].AlgorithmResultID
	if firstID == "" || repo.results[0].ResultType != "z_score" || repo.results[0].FeatureValueIDs[0] != "evt-1:daily_return_pct" || repo.results[0].EvidenceRefs[0] != "normalized_event:evt-1" {
		t.Fatalf("first result = %+v", repo.results[0])
	}
	if repo.lastEventFilter.Symbols[0] != "AAPL" || repo.lastEventFilter.Limit != 2 {
		t.Fatalf("event filter = %+v", repo.lastEventFilter)
	}
	var payload map[string]any
	if err := json.Unmarshal(repo.results[2].ResultPayloadJSON, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["feature"] != DefaultZScoreFeature || payload["symbol"] != "AAPL" {
		t.Fatalf("payload = %#v", payload)
	}
	if _, err := Run(context.Background(), repo, cfg); err != nil {
		t.Fatal(err)
	}
	if len(repo.results) != 3 || repo.results[0].AlgorithmResultID != firstID || repo.insertResultCalls != 6 {
		t.Fatalf("idempotent results len=%d first=%s calls=%d", len(repo.results), repo.results[0].AlgorithmResultID, repo.insertResultCalls)
	}
}

func TestRunZScoreSkipsWhenMinSamplesNotMet(t *testing.T) {
	now := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	repo := &fakeAlgorithmRepository{events: []storage.NormalizedEventLedgerRecord{normalizedEvent("evt-1", "MSFT", 1, now)}}
	result, err := Run(context.Background(), repo, Config{ExecutionRequestID: "algexec-2", TenantID: "tenant-local", WindowStart: now.Add(-time.Hour), WindowEnd: now.Add(time.Hour), MinSamples: 3})
	if err != nil {
		t.Fatal(err)
	}
	if result.Metrics.Scanned != 1 || result.Metrics.UsableSamples != 1 || result.Metrics.Results != 0 || len(repo.results) != 0 {
		t.Fatalf("metrics=%+v results=%d", result.Metrics, len(repo.results))
	}
}

func TestRunRejectsUnsupportedAlgorithm(t *testing.T) {
	now := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	repo := &fakeAlgorithmRepository{}
	_, err := Run(context.Background(), repo, Config{ExecutionRequestID: "algexec-3", TenantID: "tenant-local", AlgorithmID: "signalops.algorithms.unknown_v1", WindowStart: now.Add(-time.Hour), WindowEnd: now.Add(time.Hour)})
	if err == nil {
		t.Fatal("expected unsupported algorithm error")
	}
	if len(repo.requests) != 1 || repo.requests[0].Status != storage.AlgorithmExecutionStatusFailed {
		t.Fatalf("requests = %+v", repo.requests)
	}
}

func normalizedEvent(eventID string, symbol string, dailyReturn float64, observationTime time.Time) storage.NormalizedEventLedgerRecord {
	payload, _ := json.Marshal(map[string]any{"symbol": symbol, "features": map[string]any{"daily_return_pct": dailyReturn}})
	return storage.NormalizedEventLedgerRecord{EventID: eventID, TenantID: "tenant-local", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SourceID: "src-massive", SourceAdapter: "market_data.massive", Dataset: "equity_eod_prices", ObservationTime: observationTime, NormalizedPayload: payload}
}

func TestRunAdditionalAlgorithmAdaptersWriteResults(t *testing.T) {
	now := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name       string
		algorithm  string
		resultType string
		wantCount  int
	}{
		{name: "river", algorithm: RiverAnomalyAlgorithmID, resultType: "online_anomaly_score", wantCount: 4},
		{name: "ruptures", algorithm: RupturesChangePointAlgorithmID, resultType: "change_point_score", wantCount: 3},
		{name: "statsmodels", algorithm: StatsmodelsForecastAlgorithmID, resultType: "forecast_residual", wantCount: 4},
		{name: "sklearn classifier", algorithm: SklearnClassifierAlgorithmID, resultType: "classifier_label", wantCount: 4},
		{name: "sklearn isolation", algorithm: SklearnIsolationForestID, resultType: "isolation_score", wantCount: 4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeAlgorithmRepository{events: []storage.NormalizedEventLedgerRecord{
				normalizedFeatureEvent("evt-1", "AAPL", "open_close_move_pct", 1, now),
				normalizedFeatureEvent("evt-2", "AAPL", "open_close_move_pct", 1.2, now.Add(time.Hour)),
				normalizedFeatureEvent("evt-3", "AAPL", "open_close_move_pct", 1.1, now.Add(2*time.Hour)),
				normalizedFeatureEvent("evt-4", "AAPL", "open_close_move_pct", 7.5, now.Add(3*time.Hour)),
			}}
			result, err := Run(context.Background(), repo, Config{ExecutionRequestID: "algexec-" + tc.name, TenantID: "tenant-local", AlgorithmID: tc.algorithm, Feature: "open_close_move_pct", Symbols: []string{"AAPL"}, WindowStart: now.Add(-time.Hour), WindowEnd: now.Add(24 * time.Hour), MaxRecords: 10, BatchSize: 3, ZThreshold: 1, MinSamples: 3})
			if err != nil {
				t.Fatal(err)
			}
			if result.ExecutionRequest.Status != storage.AlgorithmExecutionStatusSucceeded || result.Metrics.Results != tc.wantCount || len(repo.results) != tc.wantCount {
				t.Fatalf("metrics=%+v requests=%+v results=%d", result.Metrics, repo.requests, len(repo.results))
			}
			if repo.results[0].AlgorithmID != tc.algorithm || repo.results[0].ResultType != tc.resultType {
				t.Fatalf("result = %+v", repo.results[0])
			}
			var payload map[string]any
			if err := json.Unmarshal(repo.results[len(repo.results)-1].ResultPayloadJSON, &payload); err != nil {
				t.Fatal(err)
			}
			if payload["algorithm_id"] != tc.algorithm || payload["feature"] != "open_close_move_pct" || payload["symbol"] != "AAPL" {
				t.Fatalf("payload = %#v", payload)
			}
		})
	}
}

func TestRunCopiesOptionsQualityMetadataIntoResultPayload(t *testing.T) {
	now := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	repo := &fakeAlgorithmRepository{events: []storage.NormalizedEventLedgerRecord{
		normalizedOptionsDistributionEvent("evt-1", "NVDA", 1.0, "usable", now),
		normalizedOptionsDistributionEvent("evt-2", "NVDA", 2.0, "denominator_zero", now.Add(time.Hour)),
	}}
	_, err := Run(context.Background(), repo, Config{ExecutionRequestID: "algexec-quality", TenantID: "tenant-local", Dataset: "options_distribution_daily", Feature: "call_put_open_interest_ratio", Symbols: []string{"NVDA"}, WindowStart: now.Add(-time.Hour), WindowEnd: now.Add(24 * time.Hour), MaxRecords: 10, BatchSize: 10, MinSamples: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.results) != 2 {
		t.Fatalf("results = %d", len(repo.results))
	}
	var payload map[string]any
	if err := json.Unmarshal(repo.results[1].ResultPayloadJSON, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["dataset"] != "options_distribution_daily" || payload["call_put_oi_ratio_quality"] != "denominator_zero" || payload["open_interest_quality"] != "partial_zero" {
		t.Fatalf("payload = %#v", payload)
	}
}

func normalizedOptionsDistributionEvent(eventID string, symbol string, ratio float64, ratioQuality string, observationTime time.Time) storage.NormalizedEventLedgerRecord {
	payload, _ := json.Marshal(map[string]any{"symbol": symbol, "call_put_oi_ratio_quality": ratioQuality, "open_interest_quality": "partial_zero", "open_interest_zero_rate": 0.5, "call_put_oi_denominator_is_zero": ratioQuality == "denominator_zero", "features": map[string]any{"call_put_open_interest_ratio": ratio}})
	return storage.NormalizedEventLedgerRecord{EventID: eventID, TenantID: "tenant-local", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SourceID: "src-massive", SourceAdapter: "market_data.massive", Dataset: "options_distribution_daily", ObservationTime: observationTime, NormalizedPayload: payload}
}

func normalizedFeatureEvent(eventID string, symbol string, feature string, value float64, observationTime time.Time) storage.NormalizedEventLedgerRecord {
	payload, _ := json.Marshal(map[string]any{"symbol": symbol, "features": map[string]any{feature: value}})
	return storage.NormalizedEventLedgerRecord{EventID: eventID, TenantID: "tenant-local", AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SourceID: "src-massive", SourceAdapter: "market_data.massive", Dataset: "equity_eod_prices", ObservationTime: observationTime, NormalizedPayload: payload}
}
