package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

type subscriberGlobalRiskRewardOverviewFake struct {
	*fakeQueryRepository
	snapshots     []storage.MarketOpsRiskRewardSnapshotRecord
	options       []storage.MarketOpsOptionsDistributionRecord
	symbols       []string
	optionSymbols []string
}

func (f *subscriberGlobalRiskRewardOverviewFake) ListSubscriberGlobalRiskRewardSnapshots(_ context.Context, symbols []string, _ time.Time, _ int) ([]storage.MarketOpsRiskRewardSnapshotRecord, error) {
	f.symbols = append([]string(nil), symbols...)
	return f.snapshots, nil
}

func (f *subscriberGlobalRiskRewardOverviewFake) ListSubscriberGlobalOptionsDistributions(_ context.Context, symbols []string, _ time.Time, _ int) ([]storage.MarketOpsOptionsDistributionRecord, error) {
	f.optionSymbols = append([]string(nil), symbols...)
	return f.options, nil
}

func TestSubscriberSignalOverviewUsesGlobalRiskRewardProjection(t *testing.T) {
	fixture := newTestAuthFixture(t)
	tradeDate := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)
	repo := &subscriberGlobalRiskRewardOverviewFake{
		fakeQueryRepository: &fakeQueryRepository{
			algorithmResults: []storage.AlgorithmResultRecord{{
				TenantID: "tenant-pilot-b", AlgorithmID: riskRewardAlgorithmID,
				ResultPayloadJSON: []byte("{\"symbol\":\"AAPL\",\"observation_time\":\"2026-08-13T20:00:00Z\",\"technical_direction\":\"bearish\",\"technical_score\":-99}"),
			}},
			marketOpsAssets: []storage.MarketOpsAssetRecord{{TenantID: "tenant-pilot-b", Ticker: "AAPL", IsActive: true}},
		},
		snapshots: []storage.MarketOpsRiskRewardSnapshotRecord{{
			SnapshotID: "global-risk-aapl-20260814", TenantID: "platform-global", Symbol: "AAPL",
			SessionDate: tradeDate, ObservedAt: tradeDate.Add(20 * time.Hour),
			TechnicalScore: 42, TechnicalDirection: "bullish", RiskLevel: "medium",
			Confidence: .82, UsableInputCount: 6, RequiredInputCount: 8, Eligible: true,
			ResultPayloadJSON: []byte("{\"symbol\":\"AAPL\",\"observation_time\":\"2026-08-14T20:00:00Z\",\"technical_score\":42,\"technical_direction\":\"bullish\",\"confidence_basis\":{\"usable_technical_inputs\":6}}"),
		}},
		options: []storage.MarketOpsOptionsDistributionRecord{{TenantID: "platform-global", Symbol: "AAPL", TradeDate: tradeDate, WindowName: "10_trade_days", ContractCount: 10, TotalCallVolume: 2000, TotalPutVolume: 200}},
	}
	router := NewRouter(RouterConfig{
		Auth:                          fixture.authCfg,
		QueryRepository:               repo,
		SubscriberListsEnabled:        true,
		SubscriberListsPilotTenants:   map[string]struct{}{"tenant-pilot-b": {}},
		SubscriberWatchlistRepository: &subscriberWatchlistAPIFake{},
	})
	token := fixture.token(t, map[string]any{"tenant_id": "tenant-pilot-b"})
	request := httptest.NewRequest(http.MethodGet, "/v1/tenants/tenant-pilot-b/marketops/assets/signal-overview?window=10_trade_days", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(request, token))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	sort.Strings(repo.symbols)
	if len(repo.symbols) != 1 || repo.symbols[0] != "AAPL" || len(repo.optionSymbols) != 1 || repo.optionSymbols[0] != "AAPL" {
		t.Fatalf("global reader symbols risk=%v options=%v, want [AAPL]", repo.symbols, repo.optionSymbols)
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	points := response["risk_reward"].(map[string]any)["points"].([]any)
	if len(points) != 1 {
		t.Fatalf("risk/reward points=%#v", points)
	}
	point := points[0].(map[string]any)
	if point["trade_date"] != "2026-08-14" {
		t.Fatalf("subscriber dashboard used stale tenant-local data: %#v", point)
	}
	flow := response["options_flow_extremes"].(map[string]any)
	if flow["as_of"] != "2026-08-14" {
		t.Fatalf("subscriber dashboard used stale tenant-local Options data: %#v", flow)
	}
}

type subscriberGlobalMarketStateFake struct {
	*fakeQueryRepository
	states  []storage.MarketOpsMarketStateRecord
	symbols []string
}

func (f *subscriberGlobalMarketStateFake) ListSubscriberGlobalMarketOpsMarketStates(_ context.Context, symbols []string, _ storage.MarketOpsMarketStateFilter) ([]storage.MarketOpsMarketStateRecord, error) {
	f.symbols = append([]string(nil), symbols...)
	return f.states, nil
}

func TestSubscriberMarketStateUsesGlobalProjection(t *testing.T) {
	fixture := newTestAuthFixture(t)
	globalDate := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)
	globalState := validMarketOpsMarketStateRecord()
	globalState.MarketStateID = "global-state-aapl-20260814"
	globalState.TenantID = "platform-global"
	globalState.SessionDate = globalDate
	globalState.AsOfTime = globalDate.Add(20 * time.Hour)
	legacyState := validMarketOpsMarketStateRecord()
	legacyState.TenantID = "tenant-pilot-b"
	legacyState.SessionDate = globalDate.AddDate(0, 0, -1)
	repo := &subscriberGlobalMarketStateFake{
		fakeQueryRepository: &fakeQueryRepository{marketOpsMarketStates: []storage.MarketOpsMarketStateRecord{legacyState}},
		states:              []storage.MarketOpsMarketStateRecord{globalState},
	}
	router := NewRouter(RouterConfig{
		Auth:                          fixture.authCfg,
		QueryRepository:               repo,
		SubscriberListsEnabled:        true,
		SubscriberListsPilotTenants:   map[string]struct{}{"tenant-pilot-b": {}},
		SubscriberWatchlistRepository: &subscriberWatchlistAPIFake{},
	})
	request := httptest.NewRequest(http.MethodGet, "/v1/marketops/states?tenant_id=tenant-pilot-b&symbol=AAPL", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(request, fixture.token(t, map[string]any{"tenant_id": "tenant-pilot-b"})))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if len(repo.symbols) != 1 || repo.symbols[0] != "AAPL" {
		t.Fatalf("global state symbols=%v, want [AAPL]", repo.symbols)
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	states := response["market_states"].([]any)
	if len(states) != 1 {
		t.Fatalf("market states=%#v", states)
	}
	state := states[0].(map[string]any)
	if state["market_state_id"] != globalState.MarketStateID || state["session_date"] != "2026-08-14T00:00:00Z" || state["tenant_id"] != "platform-global" {
		t.Fatalf("subscriber market state did not use the global projection: %#v", state)
	}
}
