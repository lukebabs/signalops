package fmp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGetAnnualFinancialSnapshotUsesAnnualStarterEndpoints(t *testing.T) {
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path+"?"+r.URL.RawQuery)
		if r.Header.Get("apikey") != "test-key" {
			t.Fatalf("missing FMP key header")
		}
		var payload string
		switch r.URL.Path {
		case "/stable/income-statement":
			payload = `[{"date":"2025-12-31","calendarYear":"2025","acceptedDate":"2026-02-12 16:30:00","revenue":100}]`
		case "/stable/balance-sheet-statement":
			payload = `[{"date":"2025-12-31","calendarYear":"2025","acceptedDate":"2026-02-12 16:30:00","totalAssets":80}]`
		case "/stable/cash-flow-statement":
			payload = `[{"date":"2025-12-31","calendarYear":"2025","acceptedDate":"2026-02-12 16:30:00","freeCashFlow":20}]`
		case "/stable/ratios":
			payload = `[{"date":"2025-12-31","calendarYear":"2025","currentRatio":1.2}]`
		case "/stable/key-metrics":
			payload = `[{"date":"2025-12-31","calendarYear":"2025","peRatio":20}]`
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = fmt.Fprint(w, payload)
	}))
	defer server.Close()
	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "test-key"})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := client.GetAnnualFinancialSnapshot(context.Background(), " aapl ")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Ticker != "AAPL" || len(snapshot.Periods) != 1 || snapshot.Periods[0].FiscalYear != "2025" {
		t.Fatalf("unexpected annual snapshot: %#v", snapshot)
	}
	if len(snapshot.RatioReferences) != 1 || len(snapshot.KeyMetricReferences) != 1 || client.Calls() != 5 {
		t.Fatalf("unexpected references/calls ratios=%d metrics=%d calls=%d", len(snapshot.RatioReferences), len(snapshot.KeyMetricReferences), client.Calls())
	}
	for _, request := range requests {
		if !strings.Contains(request, "period=annual") || !strings.Contains(request, "limit=5") || !strings.Contains(request, "symbol=AAPL") {
			t.Fatalf("request did not use annual contract: %s", request)
		}
	}
}

func TestGetAnnualFinancialSnapshotNormalizesClassShareRequestSymbol(t *testing.T) {
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.RawQuery)
		switch r.URL.Path {
		case "/stable/income-statement":
			_, _ = fmt.Fprint(w, `[{"date":"2025-12-31","calendarYear":"2025","acceptedDate":"2026-02-12 16:30:00","revenue":100}]`)
		case "/stable/balance-sheet-statement":
			_, _ = fmt.Fprint(w, `[{"date":"2025-12-31","calendarYear":"2025","acceptedDate":"2026-02-12 16:30:00","totalAssets":80}]`)
		case "/stable/cash-flow-statement":
			_, _ = fmt.Fprint(w, `[{"date":"2025-12-31","calendarYear":"2025","acceptedDate":"2026-02-12 16:30:00","freeCashFlow":20}]`)
		case "/stable/ratios", "/stable/key-metrics":
			_, _ = fmt.Fprint(w, `[{"date":"2025-12-31","calendarYear":"2025"}]`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "test-key"})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := client.GetAnnualFinancialSnapshot(context.Background(), " brk.b ")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Ticker != "BRK.B" {
		t.Fatalf("platform ticker should be preserved, got %s", snapshot.Ticker)
	}
	for _, request := range requests {
		if !strings.Contains(request, "symbol=BRK-B") || strings.Contains(request, "symbol=BRK.B") {
			t.Fatalf("request did not normalize class-share symbol for FMP: %s", request)
		}
	}
}

func TestClientPacesRequests(t *testing.T) {
	client, err := NewClient(ClientConfig{BaseURL: "https://example.test", APIKey: "test-key", MinRequestInterval: 5 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	if err := client.waitForRequestSlot(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := client.waitForRequestSlot(context.Background()); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed < 4*time.Millisecond {
		t.Fatalf("request pacing elapsed=%s", elapsed)
	}
}

func TestMergeAnnualPeriodsRequiresAllThreeStatements(t *testing.T) {
	periods, err := mergeAnnualPeriods(
		[]json.RawMessage{json.RawMessage(`{"date":"2025-12-31"}`)},
		[]json.RawMessage{json.RawMessage(`{"date":"2025-12-31"}`)},
		nil,
	)
	if err != nil || len(periods) != 0 {
		t.Fatalf("periods=%v err=%v", periods, err)
	}
}
