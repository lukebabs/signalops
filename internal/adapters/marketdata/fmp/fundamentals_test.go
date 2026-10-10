package fmp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestGetFundamentalSnapshotNormalizesClassShareRequestSymbol(t *testing.T) {
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.RawQuery)
		switch r.URL.Path {
		case "/stable/profile":
			_, _ = fmt.Fprint(w, `[{"marketCap":1200}]`)
		case "/stable/income-statement":
			_, _ = fmt.Fprint(w, `[{"date":"2025-12-31","fillingDate":"2026-02-01","revenue":100},{"date":"2024-12-31","revenue":90},{"date":"2023-12-31","revenue":80},{"date":"2022-12-31","revenue":70}]`)
		default:
			_, _ = fmt.Fprint(w, `[{"date":"2025-12-31","fillingDate":"2026-02-01","revenue":100,"netIncome":10,"ebitda":20,"operatingIncome":15,"netCashProvidedByOperatingActivities":12,"capitalExpenditure":2,"totalDebt":30,"cashAndCashEquivalents":5,"totalStockholdersEquity":40,"totalAssets":100}]`)
		}
	}))
	defer server.Close()
	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "test-key"})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := client.GetFundamentalSnapshot(context.Background(), " bf.a ")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Ticker != "BF.A" {
		t.Fatalf("platform ticker should be preserved, got %s", snapshot.Ticker)
	}
	for _, request := range requests {
		if !strings.Contains(request, "symbol=BF-A") || strings.Contains(request, "symbol=BF.A") {
			t.Fatalf("request did not normalize class-share symbol for FMP: %s", request)
		}
	}
}
