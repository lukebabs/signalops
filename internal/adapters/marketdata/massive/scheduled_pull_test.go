package massive

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/lukebabs/signalops/pkg/broker"
)

type fakeScheduledClient struct {
	equityRecords map[string]EquityEODPriceRecord
	optionRecords map[string][]OptionContractDailyRecord
	err           error
}

func (f fakeScheduledClient) GetEquityDailyBar(_ context.Context, symbol string, _ time.Time) (EquityEODPriceRecord, error) {
	if f.err != nil {
		return EquityEODPriceRecord{}, f.err
	}
	return f.equityRecords[symbol], nil
}

func (f fakeScheduledClient) ListOptionContracts(_ context.Context, underlying string, _ time.Time, _ int) ([]OptionContractDailyRecord, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.optionRecords[underlying], nil
}

type fakePublisher struct {
	messages []broker.Message
	err      error
}

func (f *fakePublisher) Publish(_ context.Context, msg broker.Message) (broker.PublishResult, error) {
	if f.err != nil {
		return broker.PublishResult{}, f.err
	}
	f.messages = append(f.messages, msg)
	return broker.PublishResult{Topic: msg.Topic, Partition: 1, Offset: int64(len(f.messages) - 1)}, nil
}

func (f *fakePublisher) Close(context.Context) error { return nil }

func TestRunScheduledPullDryRunBuildsEvents(t *testing.T) {
	client := fakeScheduledClient{
		equityRecords: map[string]EquityEODPriceRecord{
			"MSFT": {ProviderEventID: "msft-2026-07-06", Symbol: "MSFT", ObservationDate: testObservationDate()},
		},
		optionRecords: map[string][]OptionContractDailyRecord{
			"MSFT": {{ProviderContractID: "contract-1", OptionTicker: "O:MSFT260116C00400000", UnderlyingSymbol: "MSFT", ContractType: "call", ExpirationDate: time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC), StrikePrice: 400, ObservationDate: testObservationDate()}},
		},
	}
	publisher := &fakePublisher{}

	report, err := RunScheduledPull(context.Background(), ScheduledPullConfig{
		TenantID:         "tenant-1",
		SourceID:         "src-massive",
		ObservationDate:  testObservationDate(),
		Companies:        []MegacapCompanySeed{{Ticker: "MSFT"}},
		IncludeEquityEOD: true,
		IncludeOptions:   true,
		DryRun:           true,
	}, client, publisher)
	if err != nil {
		t.Fatalf("run scheduled pull: %v", err)
	}
	if report.EventsBuilt != 2 || report.EventsPublished != 0 {
		t.Fatalf("report = %+v", report)
	}
	if report.EventsByDataset[DatasetEquityEODPrices] != 1 || report.EventsByDataset[DatasetOptionsContractsDaily] != 1 {
		t.Fatalf("events by dataset = %+v", report.EventsByDataset)
	}
	if len(publisher.messages) != 0 {
		t.Fatalf("dry run published %d messages", len(publisher.messages))
	}
}

func TestRunScheduledPullPublishesRawEvents(t *testing.T) {
	client := fakeScheduledClient{
		equityRecords: map[string]EquityEODPriceRecord{
			"NVDA": {ProviderEventID: "nvda-2026-07-06", Symbol: "NVDA", ObservationDate: testObservationDate()},
		},
	}
	publisher := &fakePublisher{}

	report, err := RunScheduledPull(context.Background(), ScheduledPullConfig{
		TenantID:         "tenant-1",
		SourceID:         "src-massive",
		Environment:      "test",
		ObservationDate:  testObservationDate(),
		Companies:        []MegacapCompanySeed{{Ticker: "NVDA"}},
		IncludeEquityEOD: true,
		DryRun:           false,
	}, client, publisher)
	if err != nil {
		t.Fatalf("run scheduled pull: %v", err)
	}
	if report.EventsPublished != 1 || len(publisher.messages) != 1 {
		t.Fatalf("published report/messages = %+v/%d", report, len(publisher.messages))
	}
	msg := publisher.messages[0]
	if msg.Topic != "signalops.test.raw.v1" {
		t.Fatalf("topic = %q", msg.Topic)
	}
	if msg.Key == "" || msg.Headers["signalops_idempotency"] != msg.Key {
		t.Fatalf("key/headers = %q/%+v", msg.Key, msg.Headers)
	}
	var decoded map[string]any
	if err := json.Unmarshal(msg.Value, &decoded); err != nil {
		t.Fatalf("message value json: %v", err)
	}
	if decoded["source_adapter"] != AdapterID || decoded["dataset"] != DatasetEquityEODPrices {
		t.Fatalf("decoded = %+v", decoded)
	}
}

func TestRunScheduledPullRequiresPublisherWhenPublishing(t *testing.T) {
	_, err := RunScheduledPull(context.Background(), ScheduledPullConfig{
		TenantID:         "tenant-1",
		SourceID:         "src-massive",
		ObservationDate:  testObservationDate(),
		Companies:        []MegacapCompanySeed{{Ticker: "NVDA"}},
		IncludeEquityEOD: true,
		DryRun:           false,
	}, fakeScheduledClient{}, nil)
	if err == nil {
		t.Fatal("expected publisher error")
	}
}

func TestRunScheduledPullContinueOnError(t *testing.T) {
	report, err := RunScheduledPull(context.Background(), ScheduledPullConfig{
		TenantID:        "tenant-1",
		SourceID:        "src-massive",
		ObservationDate: testObservationDate(),
		Companies:       []MegacapCompanySeed{{Ticker: "MSFT"}, {Ticker: "NVDA"}},
		IncludeOptions:  true,
		DryRun:          true,
		ContinueOnError: true,
	}, fakeScheduledClient{err: errors.New("provider unavailable")}, nil)
	if err != nil {
		t.Fatalf("run scheduled pull: %v", err)
	}
	if report.Failures != 2 || len(report.Errors) != 2 {
		t.Fatalf("report = %+v", report)
	}
}

func testObservationDate() time.Time {
	return time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC)
}

type flakyEquityClient struct {
	failuresRemaining int
	requests          int
}

func (f *flakyEquityClient) GetEquityDailyBar(_ context.Context, symbol string, date time.Time) (EquityEODPriceRecord, error) {
	f.requests++
	if f.failuresRemaining > 0 {
		f.failuresRemaining--
		return EquityEODPriceRecord{}, errors.New("temporary provider failure")
	}
	return EquityEODPriceRecord{ProviderEventID: symbol + "-event", Symbol: symbol, ObservationDate: date}, nil
}

func (f *flakyEquityClient) ListOptionContracts(context.Context, string, time.Time, int) ([]OptionContractDailyRecord, error) {
	return nil, nil
}

func TestRunScheduledPullRetriesTransientProviderFailures(t *testing.T) {
	client := &flakyEquityClient{failuresRemaining: 1}
	report, err := RunScheduledPull(context.Background(), ScheduledPullConfig{
		TenantID:         "tenant-1",
		SourceID:         "src-massive",
		ObservationDate:  testObservationDate(),
		Companies:        []MegacapCompanySeed{{Ticker: "MSFT"}},
		IncludeEquityEOD: true,
		DryRun:           true,
		MaxRetries:       1,
	}, client, nil)
	if err != nil {
		t.Fatalf("run scheduled pull: %v", err)
	}
	if client.requests != 2 || report.ProviderRequests != 2 || report.ProviderRetries != 1 {
		t.Fatalf("requests/retries = client:%d report:%d/%d", client.requests, report.ProviderRequests, report.ProviderRetries)
	}
	if report.EventsBuilt != 1 || report.Failures != 0 {
		t.Fatalf("report = %+v", report)
	}
}

func TestRunScheduledPullStopsAfterRetryExhaustion(t *testing.T) {
	client := &flakyEquityClient{failuresRemaining: 2}
	report, err := RunScheduledPull(context.Background(), ScheduledPullConfig{
		TenantID:         "tenant-1",
		SourceID:         "src-massive",
		ObservationDate:  testObservationDate(),
		Companies:        []MegacapCompanySeed{{Ticker: "MSFT"}},
		IncludeEquityEOD: true,
		DryRun:           true,
		MaxRetries:       1,
	}, client, nil)
	if err == nil {
		t.Fatal("expected retry exhaustion error")
	}
	if report.ProviderRequests != 2 || report.ProviderRetries != 1 || report.Failures != 1 {
		t.Fatalf("report = %+v", report)
	}
}
