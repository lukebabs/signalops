package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestGetMarketOpsFeatureDefinitions(t *testing.T) {
	repo := &fakeQueryRepository{marketOpsFeatureDefinitions: []storage.MarketOpsFeatureDefinitionRecord{validMarketOpsFeatureDefinitionRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodGet, "/v1/marketops/features/definitions?tenant_id=tenant-1&feature_key=underlying.return_1d&feature_version=v1&domain=underlying_momentum&status=active&limit=10", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if repo.lastFeatureDefinitionFilter.FeatureKey != "underlying.return_1d" || repo.lastFeatureDefinitionFilter.Domain != "underlying_momentum" || repo.lastFeatureDefinitionFilter.Limit != 10 {
		t.Fatalf("filter = %+v", repo.lastFeatureDefinitionFilter)
	}
	var response map[string][]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["feature_definitions"][0]["value_type"] != "numeric" {
		t.Fatalf("response = %+v", response)
	}
}

func TestGetMarketOpsFeatureObservations(t *testing.T) {
	repo := &fakeQueryRepository{marketOpsFeatureObservations: []storage.MarketOpsFeatureObservationRecord{validMarketOpsFeatureObservationRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodGet, "/v1/marketops/features/observations?tenant_id=tenant-1&app_id=marketops&asset_id=asset:AAPL&symbol=aapl&feature_key=underlying.return_1d&feature_version=v1&domain=underlying_momentum&quality_state=usable&dimensions=%7B%22target_dte%22%3A30%7D&session_start=2026-07-01&session_end=2026-07-19&limit=25", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	filter := repo.lastFeatureObservationFilter
	if filter.Symbol != "aapl" || filter.QualityState != storage.MarketOpsQualityUsable || filter.SessionStart.Format("2006-01-02") != "2026-07-01" || string(filter.DimensionsJSON) != `{"target_dte":30}` || filter.Limit != 25 {
		t.Fatalf("filter = %+v", filter)
	}
	var response map[string][]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	observation := response["feature_observations"][0]
	if observation["feature_observation_id"] != "mfo-1" || observation["numeric_value"].(float64) != 0.012 {
		t.Fatalf("observation = %+v", observation)
	}
}

func TestGetMarketOpsStatesAndLineage(t *testing.T) {
	state := validMarketOpsMarketStateRecord()
	state.FeatureObservationIDs = []string{"mfo-1", "mfo-missing"}
	repo := &fakeQueryRepository{
		marketOpsMarketStates:        []storage.MarketOpsMarketStateRecord{state},
		marketOpsFeatureObservations: []storage.MarketOpsFeatureObservationRecord{validMarketOpsFeatureObservationRecord()},
	}
	router := NewRouter(RouterConfig{QueryRepository: repo})

	listReq := httptest.NewRequest(http.MethodGet, "/v1/marketops/states?tenant_id=tenant-1&symbol=AAPL&state_schema_version=marketops.state.v1&quality_state=usable&session_start=2026-07-19&session_end=2026-07-19", nil)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", listRec.Code, listRec.Body.String())
	}
	if repo.lastMarketStateFilter.Symbol != "AAPL" || repo.lastMarketStateFilter.StateSchemaVersion != "marketops.state.v1" {
		t.Fatalf("filter = %+v", repo.lastMarketStateFilter)
	}

	lineageReq := httptest.NewRequest(http.MethodGet, "/v1/marketops/states/mstate-1/lineage", nil)
	lineageRec := httptest.NewRecorder()
	router.ServeHTTP(lineageRec, lineageReq)
	if lineageRec.Code != http.StatusOK {
		t.Fatalf("lineage status = %d body=%s", lineageRec.Code, lineageRec.Body.String())
	}
	var response map[string]map[string]any
	if err := json.Unmarshal(lineageRec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	lineage := response["lineage"]
	if lineage["market_state"].(map[string]any)["market_state_id"] != "mstate-1" {
		t.Fatalf("lineage = %+v", lineage)
	}
	if len(lineage["feature_observations"].([]any)) != 1 || lineage["source_event_ids"].([]any)[0] != "event-1" {
		t.Fatalf("lineage = %+v", lineage)
	}
	if lineage["missing_feature_observation_ids"].([]any)[0] != "mfo-missing" {
		t.Fatalf("lineage = %+v", lineage)
	}
}

func TestGetMarketOpsTransitions(t *testing.T) {
	repo := &fakeQueryRepository{marketOpsStateTransitions: []storage.MarketOpsStateTransitionRecord{validMarketOpsStateTransitionRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodGet, "/v1/marketops/transitions?tenant_id=tenant-1&symbol=AAPL&current_state_id=mstate-1&feature_key=underlying.return_1d&feature_version=v1&transition_type=zscore&quality_state=usable&session_start=2026-07-01&session_end=2026-07-19&limit=20", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if repo.lastStateTransitionFilter.TransitionType != "zscore" || repo.lastStateTransitionFilter.CurrentStateID != "mstate-1" {
		t.Fatalf("filter = %+v", repo.lastStateTransitionFilter)
	}
	var response map[string][]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["transitions"][0]["transition_id"] != "mtrans-1" {
		t.Fatalf("response = %+v", response)
	}
}

func TestGetMarketOpsEvidenceListAndDetail(t *testing.T) {
	repo := &fakeQueryRepository{marketOpsEvidence: []storage.MarketOpsEvidenceRecord{validMarketOpsEvidenceRecord()}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	listReq := httptest.NewRequest(http.MethodGet, "/v1/marketops/evidence?tenant_id=tenant-1&symbol=AAPL&evidence_type=return_expansion&evidence_version=v1&domain=underlying_momentum&direction=up&limit=10", nil)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", listRec.Code, listRec.Body.String())
	}
	if repo.lastEvidenceFilter.EvidenceType != "return_expansion" || repo.lastEvidenceFilter.Direction != "up" {
		t.Fatalf("filter = %+v", repo.lastEvidenceFilter)
	}

	detailReq := httptest.NewRequest(http.MethodGet, "/v1/marketops/evidence/mevidence-1", nil)
	detailRec := httptest.NewRecorder()
	router.ServeHTTP(detailRec, detailReq)
	if detailRec.Code != http.StatusOK {
		t.Fatalf("detail status = %d body=%s", detailRec.Code, detailRec.Body.String())
	}
	var response map[string]map[string]any
	if err := json.Unmarshal(detailRec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["evidence"]["statement"] != "AAPL one-day return expanded above its recent baseline." {
		t.Fatalf("response = %+v", response)
	}
}

func TestMarketOpsStateReadsRejectInvalidSessionRange(t *testing.T) {
	router := NewRouter(RouterConfig{QueryRepository: &fakeQueryRepository{}})
	for _, path := range []string{
		"/v1/marketops/states?session_start=2026-07-20&session_end=2026-07-19",
		"/v1/marketops/transitions?session_start=not-a-date",
		"/v1/marketops/features/observations?dimensions=%5B%5D",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("path=%s status=%d body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

func validMarketOpsFeatureDefinitionRecord() storage.MarketOpsFeatureDefinitionRecord {
	return storage.MarketOpsFeatureDefinitionRecord{TenantID: "tenant-1", FeatureKey: "underlying.return_1d", FeatureVersion: "v1",
		Domain: "underlying_momentum", Title: "One-day return", Description: "Point-in-time close return.", ValueType: "numeric",
		Unit: "ratio", CalculationSpec: []byte(`{"method":"close_return"}`), RequiredInputs: []byte(`["equity_eod.close"]`),
		QualityPolicy: []byte(`{"minimum_source_count":1}`), Status: storage.MarketOpsFeatureDefinitionStatusActive}
}

func validMarketOpsFeatureObservationRecord() storage.MarketOpsFeatureObservationRecord {
	value, quality := 0.012, 0.98
	return storage.MarketOpsFeatureObservationRecord{FeatureObservationID: "mfo-1", TenantID: "tenant-1", AppID: "marketops",
		AssetID: "asset:AAPL", Symbol: "AAPL", SessionDate: time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC),
		AsOfTime: time.Date(2026, 7, 19, 20, 0, 0, 0, time.UTC), FeatureKey: "underlying.return_1d", FeatureVersion: "v1",
		DimensionsJSON: []byte(`{}`), NumericValue: &value, QualityState: storage.MarketOpsQualityUsable, QualityScore: &quality,
		QualityDetailsJSON: []byte(`{"source_count":1}`), SourceEventIDs: []string{"event-1"}, SourceArtifactIDs: []string{"artifact-1"},
		CalculationRunID: "feature-run-1", DeterministicKey: "tenant-1:AAPL:2026-07-19:underlying.return_1d:v1"}
}

func validMarketOpsMarketStateRecord() storage.MarketOpsMarketStateRecord {
	quality := 0.96
	return storage.MarketOpsMarketStateRecord{MarketStateID: "mstate-1", TenantID: "tenant-1", AppID: "marketops",
		AssetID: "asset:AAPL", Symbol: "AAPL", SessionDate: time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC),
		AsOfTime: time.Date(2026, 7, 19, 20, 5, 0, 0, time.UTC), StateSchemaVersion: "marketops.state.v1",
		StatePayloadJSON: []byte(`{"underlying_momentum":{"return_1d":0.012}}`), FeatureObservationIDs: []string{"mfo-1"},
		FeatureCount: 1, RequiredFeatureCount: 1, CompletenessRatio: 1, QualityState: storage.MarketOpsQualityUsable,
		QualityScore: &quality, QualitySummaryJSON: []byte(`{"usable":1}`), EligibleHypotheses: []string{},
		BuildRunID: "state-run-1", DeterministicKey: "tenant-1:AAPL:2026-07-19:marketops.state.v1"}
}

func validMarketOpsStateTransitionRecord() storage.MarketOpsStateTransitionRecord {
	lookback, persistence := 20, 2
	current, baseline, delta, zscore, percentile := 0.012, 0.003, 0.009, 2.4, 0.97
	return storage.MarketOpsStateTransitionRecord{TransitionID: "mtrans-1", TenantID: "tenant-1", AppID: "marketops",
		AssetID: "asset:AAPL", Symbol: "AAPL", SessionDate: time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC),
		AsOfTime: time.Date(2026, 7, 19, 20, 6, 0, 0, time.UTC), CurrentStateID: "mstate-1", BaselineStateID: "mstate-0",
		FeatureKey: "underlying.return_1d", FeatureVersion: "v1", DimensionsJSON: []byte(`{}`), TransitionType: "zscore",
		LookbackSessions: &lookback, CurrentValue: &current, BaselineValue: &baseline, TransitionValue: &delta,
		ZScore: &zscore, Percentile: &percentile, PersistenceSessions: &persistence, Direction: "up",
		QualityState: storage.MarketOpsQualityUsable, TransitionPayloadJSON: []byte(`{"threshold":2}`),
		CalculationRunID: "transition-run-1", DeterministicKey: "tenant-1:AAPL:2026-07-19:return-zscore:v1"}
}

func validMarketOpsEvidenceRecord() storage.MarketOpsEvidenceRecord {
	magnitude, rarity, persistence, quality := 0.74, 0.97, 0.66, 0.96
	return storage.MarketOpsEvidenceRecord{EvidenceID: "mevidence-1", TenantID: "tenant-1", AppID: "marketops",
		AssetID: "asset:AAPL", Symbol: "AAPL", SessionDate: time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC),
		AsOfTime: time.Date(2026, 7, 19, 20, 7, 0, 0, time.UTC), EvidenceType: "return_expansion", EvidenceVersion: "v1",
		Domain: "underlying_momentum", Direction: "up", Magnitude: &magnitude, RarityScore: &rarity,
		PersistenceScore: &persistence, QualityScore: &quality, Statement: "AAPL one-day return expanded above its recent baseline.",
		EvidencePayloadJSON: []byte(`{"observed":true}`), SourceFeatureIDs: []string{"mfo-1"},
		SourceTransitionIDs: []string{"mtrans-1"}, DeterministicKey: "tenant-1:AAPL:2026-07-19:return-expansion:v1"}
}
