package options

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestNormalizedEventFromDistributionBuildsAlgorithmReadyFeaturePayload(t *testing.T) {
	tradeDate := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	record := storage.MarketOpsOptionsDistributionRecord{
		TenantID: "tenant-local", Symbol: "nvda", TradeDate: tradeDate, WindowName: DefaultWindowName, SourceID: "src-massive", Provider: "massive",
		TradeDays: 10, ContractCount: 200, TotalCallOpenInterest: 1200, TotalPutOpenInterest: 800, CallPutOpenInterestRatio: 1.5,
		CallPutVolumeRatio: 1.25, RatioDelta: 0.2, RatioChangePct: 15.384615, RatioZScore: 2.1, ChangePointScore: 2.1, Confidence: 0.525,
		MoneynessDistributionJSON:  []byte(`{"100-105%":{"call_open_interest":900}}`),
		ExpirationDistributionJSON: []byte(`{"8-30d":{"put_open_interest":400}}`),
		MetricsJSON:                []byte(`{"primary_metric":"open_interest","open_interest_zero_count":1,"open_interest_positive_count":1,"open_interest_zero_rate":0.5,"open_interest_quality":"partial_zero","call_put_oi_denominator_is_zero":false,"call_put_oi_ratio_quality":"partial_zero"}`),
		SourceTradeDates:           []time.Time{tradeDate.AddDate(0, 0, -1), tradeDate},
	}

	event, err := NormalizedEventFromDistribution(record, tradeDate.Add(2*time.Hour))
	if err != nil {
		t.Fatalf("event error = %v", err)
	}
	if event.Dataset != DistributionFeatureDataset || event.AppID != "marketops" || event.Domain != "market_data" || event.UseCase != "daily_market_surveillance" {
		t.Fatalf("event metadata = %+v", event)
	}
	if event.EventID == "" || event.IdempotencyKey == "" {
		t.Fatalf("event identity = %+v", event)
	}
	var payload map[string]any
	if err := json.Unmarshal(event.NormalizedPayload, &payload); err != nil {
		t.Fatalf("payload JSON error = %v", err)
	}
	if payload["symbol"] != "NVDA" || payload["dataset"] != DistributionFeatureDataset {
		t.Fatalf("payload identity = %+v", payload)
	}
	features := payload["features"].(map[string]any)
	if features["call_put_open_interest_ratio"].(float64) != 1.5 || features["call_put_oi_zscore"].(float64) != 2.1 {
		t.Fatalf("features = %+v", features)
	}
	var eventJSON map[string]any
	if err := json.Unmarshal(event.EventJSON, &eventJSON); err != nil {
		t.Fatalf("event JSON error = %v", err)
	}
	if eventJSON["ingestion_mode"] != "derived_feature" {
		t.Fatalf("event JSON = %+v", eventJSON)
	}
}
