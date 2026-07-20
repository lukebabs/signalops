package massive

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoadClientConfigFromEnvUsesFallbacks(t *testing.T) {
	t.Setenv("SIGNALOPS_MASSIVE_API_KEY", "")
	t.Setenv("MASSIVE_API_KEY", "")
	t.Setenv("API_KEY", "local-key")
	t.Setenv("SIGNALOPS_MASSIVE_BASE_URL", "https://example.test")

	cfg := LoadClientConfigFromEnv()

	if cfg.APIKey != "local-key" {
		t.Fatalf("api key fallback not loaded")
	}
	if cfg.BaseURL != "https://example.test" {
		t.Fatalf("base url = %q", cfg.BaseURL)
	}
}

func TestNewClientRequiresAPIKey(t *testing.T) {
	_, err := NewClient(ClientConfig{BaseURL: "https://example.test"})
	if err == nil {
		t.Fatal("expected missing api key error")
	}
}

func TestClientListOptionContracts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/reference/options/contracts" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		assertQuery(t, r.URL.Query(), "underlying_ticker", "SPY")
		assertQuery(t, r.URL.Query(), "as_of", "2026-07-06")
		assertQuery(t, r.URL.Query(), "limit", "10")
		assertQuery(t, r.URL.Query(), "apiKey", "test-key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"results":[{
				"id":"contract-1",
				"ticker":"O:SPY260116C00600000",
				"underlying_ticker":"SPY",
				"contract_type":"call",
				"expiration_date":"2026-01-16",
				"strike_price":600
			}]
		}`))
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	records, err := client.ListOptionContracts(context.Background(), "spy", time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC), 10)
	if err != nil {
		t.Fatalf("list option contracts: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("record count = %d", len(records))
	}
	record := records[0]
	if record.ProviderContractID != "contract-1" || record.OptionTicker != "O:SPY260116C00600000" {
		t.Fatalf("record = %+v", record)
	}
	if record.UnderlyingSymbol != "SPY" || record.ContractType != "call" || record.StrikePrice != 600 {
		t.Fatalf("record = %+v", record)
	}
	if record.ObservationDate.Format("2006-01-02") != "2026-07-06" {
		t.Fatalf("observation date = %s", record.ObservationDate)
	}
}

func TestClientListOptionChainSnapshot(t *testing.T) {
	calls := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/v3/snapshot/options/NVDA" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		assertQuery(t, r.URL.Query(), "apiKey", "test-key")
		if calls == 1 {
			assertQuery(t, r.URL.Query(), "limit", "2")
			assertQuery(t, r.URL.Query(), "expiration_date.gte", "2026-07-20")
			assertQuery(t, r.URL.Query(), "expiration_date.lte", "2026-11-17")
			assertQuery(t, r.URL.Query(), "strike_price.gte", "70.00000000")
			assertQuery(t, r.URL.Query(), "strike_price.lte", "130.00000000")
		}
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			_, _ = w.Write([]byte(`{
				"request_id":"req-1",
				"next_url":"` + server.URL + `/v3/snapshot/options/NVDA?cursor=next",
				"results":[{
					"day":{"open":10.1,"high":11.2,"low":9.9,"close":10.8,"volume":123,"vwap":10.6,"last_updated":1783814400000000000},
					"details":{"ticker":"O:NVDA260116C00100000","contract_type":"call","expiration_date":"2026-01-16","strike_price":100,"exercise_style":"american","shares_per_contract":100},
					"last_quote":{"bid":10.4,"ask":10.8,"last_updated":1783814400000000000},
					"greeks":{"delta":0.51,"gamma":0.02,"theta":-0.01,"vega":0.2},
					"implied_volatility":0.45,
					"open_interest":1543,
					"underlying_asset":{"ticker":"NVDA","price":172.5}
				}]
			}`))
			return
		}
		_, _ = w.Write([]byte(`{
			"request_id":"req-2",
			"results":[{
				"day":{"close":8.4,"volume":50,"last_updated":1783814400000000000},
				"details":{"ticker":"O:NVDA260116P00100000","contract_type":"put","expiration_date":"2026-01-16","strike_price":"100"},
				"open_interest":900,
				"underlying_asset":{"ticker":"NVDA","price":172.5}
			}]
		}`))
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	strikeMin, strikeMax := 70.0, 130.0
	batch, err := client.ListOptionChainSnapshotFilteredWithMetadata(context.Background(), "nvda", OptionChainSnapshotFilter{Limit: 2, MaxPages: 2, ExpirationDateGTE: time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC), ExpirationDateLTE: time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC), StrikePriceGTE: &strikeMin, StrikePriceLTE: &strikeMax})
	if err != nil {
		t.Fatalf("list option chain snapshot: %v", err)
	}
	records := batch.Records
	if len(records) != 2 || calls != 2 || batch.PagesFetched != 2 || !batch.PaginationComplete || len(batch.ProviderRequestIDs) != 2 {
		t.Fatalf("records/calls = %d/%d", len(records), calls)
	}
	call := records[0]
	if call.OptionTicker != "O:NVDA260116C00100000" || call.ContractType != "call" || call.StrikePrice != 100 {
		t.Fatalf("call record = %+v", call)
	}
	if call.OpenInterest == nil || *call.OpenInterest != 1543 || call.ProviderRequestID != "req-1" {
		t.Fatalf("open interest = %v", call.OpenInterest)
	}
	if call.Bid == nil || *call.Bid != 10.4 || call.Ask == nil || *call.Ask != 10.8 || call.QuoteTimestamp == nil || call.ExerciseStyle != "american" || call.SharesPerContract == nil || *call.SharesPerContract != 100 {
		t.Fatalf("quote metadata = %+v", call)
	}
	if call.UnderlyingClose == nil || *call.UnderlyingClose != 172.5 {
		t.Fatalf("underlying close = %v", call.UnderlyingClose)
	}
	if call.ImpliedVolatility == nil || *call.ImpliedVolatility != 0.45 || call.Delta == nil || *call.Delta != 0.51 {
		t.Fatalf("iv/delta = %v/%v", call.ImpliedVolatility, call.Delta)
	}
	put := records[1]
	if put.ContractType != "put" || put.StrikePrice != 100 || put.OpenInterest == nil || *put.OpenInterest != 900 {
		t.Fatalf("put record = %+v", put)
	}
}

func TestClientReportsIncompleteBoundedPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"request_id": "req-bounded", "next_url": "https://api.massive.test/next", "results": []any{}})
	}))
	defer server.Close()
	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	batch, err := client.ListOptionChainSnapshotFilteredWithMetadata(context.Background(), "AAPL", OptionChainSnapshotFilter{Limit: 10, MaxPages: 1})
	if err != nil {
		t.Fatal(err)
	}
	if batch.PagesFetched != 1 || batch.PaginationComplete || len(batch.ProviderRequestIDs) != 1 {
		t.Fatalf("bounded pagination metadata = %+v", batch)
	}
}

func TestClientGetEquityDailyBar(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/aggs/ticker/QQQ/range/1/day/2026-07-06/2026-07-06" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		assertQuery(t, r.URL.Query(), "apiKey", "test-key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"ticker":"QQQ",
			"results":[{"o":500.25,"h":505.00,"l":499.50,"c":501.75,"v":980000,"vw":502.10,"t":1783296000000}]
		}`))
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	record, err := client.GetEquityDailyBar(context.Background(), "qqq", time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("get equity daily bar: %v", err)
	}
	if record.Symbol != "QQQ" {
		t.Fatalf("symbol = %q", record.Symbol)
	}
	if record.Close == nil || *record.Close != 501.75 {
		t.Fatalf("close = %v", record.Close)
	}
	if record.Volume == nil || *record.Volume != 980000 {
		t.Fatalf("volume = %v", record.Volume)
	}
}

func TestClientErrorsDoNotLeakAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad key secret-key", http.StatusUnauthorized)
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "secret-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.GetEquityDailyBar(context.Background(), "SPY", time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "secret-key") {
		t.Fatalf("error leaked api key: %v", err)
	}
}

func TestDefaultBaseURLIsParseable(t *testing.T) {
	if _, err := url.Parse(DefaultBaseURL); err != nil {
		t.Fatalf("default base url parse: %v", err)
	}
}

func TestLoadClientConfigPrefersSpecificEnv(t *testing.T) {
	clearAPIEnv(t)
	t.Setenv("API_KEY", "generic-key")
	t.Setenv("MASSIVE_API_KEY", "massive-key")
	t.Setenv("SIGNALOPS_MASSIVE_API_KEY", "specific-key")

	cfg := LoadClientConfigFromEnv()
	if cfg.APIKey != "specific-key" {
		t.Fatalf("api key = %q", cfg.APIKey)
	}
}

func clearAPIEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"SIGNALOPS_MASSIVE_API_KEY", "MASSIVE_API_KEY", "API_KEY"} {
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
	}
}

func assertQuery(t *testing.T, values url.Values, key string, want string) {
	t.Helper()
	if got := values.Get(key); got != want {
		t.Fatalf("query %s = %q, want %q", key, got, want)
	}
}

func TestAggregateResponseBuildsOptionRecord(t *testing.T) {
	var response aggregateBarsResponse
	if err := json.Unmarshal([]byte(`{"results":[{"o":1.1,"h":2.2,"l":1.0,"c":1.8,"v":100,"vw":1.7,"t":1783296000000}]}`), &response); err != nil {
		t.Fatalf("decode aggregate response: %v", err)
	}
	records := response.optionRecords("O:SPY260116C00600000", "SPY", time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC))
	if len(records) != 1 {
		t.Fatalf("record count = %d", len(records))
	}
	if records[0].OptionTicker != "O:SPY260116C00600000" || records[0].UnderlyingSymbol != "SPY" {
		t.Fatalf("record = %+v", records[0])
	}
	if records[0].Close == nil || *records[0].Close != 1.8 {
		t.Fatalf("close = %v", records[0].Close)
	}
}
