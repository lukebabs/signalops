package fmp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetFundamentalSnapshotUsesFiveFMPCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("apikey") != "test-key" {
			t.Fatalf("missing fmp api key header")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/stable/income-statement-ttm":
			_, _ = w.Write([]byte(`[{"revenue":100,"netIncome":10,"ebitda":20,"operatingIncome":15,"fillingDate":"2026-07-01"}]`))
		case "/stable/cash-flow-statement-ttm":
			_, _ = w.Write([]byte(`[{"netCashProvidedByOperatingActivities":18,"capitalExpenditure":4,"fillingDate":"2026-07-01"}]`))
		case "/stable/balance-sheet-statement-ttm":
			_, _ = w.Write([]byte(`[{"totalDebt":30,"cashAndCashEquivalents":5,"totalStockholdersEquity":60,"totalAssets":100,"fillingDate":"2026-07-01"}]`))
		case "/stable/profile":
			_, _ = w.Write([]byte(`[{"mktCap":200}]`))
		case "/stable/income-statement":
			_, _ = w.Write([]byte(`[{"revenue":100},{"revenue":90},{"revenue":80},{"revenue":70}]`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := client.GetFundamentalSnapshot(context.Background(), "acme")
	if err != nil {
		t.Fatal(err)
	}
	if client.Calls() != 5 || snapshot.Ticker != "ACME" || snapshot.Revenue3YAgo != 70 || snapshot.EnterpriseValue != 225 || snapshot.InvestedCapital != 95 {
		t.Fatalf("unexpected snapshot: %+v calls=%d", snapshot, client.Calls())
	}
}
