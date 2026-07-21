package main

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/adapters/marketdata/massive"
	"github.com/lukebabs/signalops/internal/storage"
	"github.com/lukebabs/signalops/pkg/broker"
)

type fakeReconciliationRepo struct {
	assets     []storage.MarketOpsAssetRecord
	normalized map[string]bool
	raw        map[string]storage.RawEventLedgerRecord
	tasks      map[string]storage.EquityReconciliationTaskRecord
	claims     []string
	enqueues   int
}

func newFakeReconciliationRepo(missing ...string) *fakeReconciliationRepo {
	missingSet := map[string]bool{}
	for _, symbol := range missing {
		missingSet[symbol] = true
	}
	repo := &fakeReconciliationRepo{
		normalized: map[string]bool{},
		raw:        map[string]storage.RawEventLedgerRecord{},
		tasks:      map[string]storage.EquityReconciliationTaskRecord{},
	}
	for rank := 1; rank <= 50; rank++ {
		symbol := "S" + twoDigits(rank)
		repo.assets = append(repo.assets, storage.MarketOpsAssetRecord{
			TenantID: "tenant-local", SourceID: "src-massive", UniverseGroup: "top50_megacap",
			Rank: rank, Ticker: symbol, TickerKey: symbol, Company: symbol,
			CompanyKey: symbol, Sector: "Technology", SectorKey: "technology", IsActive: true,
		})
		repo.normalized[symbol] = !missingSet[symbol]
	}
	return repo
}

func twoDigits(value int) string {
	if value < 10 {
		return "0" + string(rune('0'+value))
	}
	return string([]byte{byte('0' + value/10), byte('0' + value%10)})
}

func (r *fakeReconciliationRepo) ListMarketOpsAssets(context.Context, string, string, bool, int) ([]storage.MarketOpsAssetRecord, error) {
	return append([]storage.MarketOpsAssetRecord(nil), r.assets...), nil
}

func (r *fakeReconciliationRepo) HasNormalizedEquity(_ context.Context, _, _, symbol string, _ time.Time) (bool, error) {
	return r.normalized[symbol], nil
}

func (r *fakeReconciliationRepo) FindRawEquityEvent(_ context.Context, _, _, symbol string, _ time.Time) (storage.RawEventLedgerRecord, error) {
	record, ok := r.raw[symbol]
	if !ok {
		return storage.RawEventLedgerRecord{}, storage.ErrNotFound
	}
	return record, nil
}

func (r *fakeReconciliationRepo) EnqueueEquityReconciliationTask(_ context.Context, record storage.EquityReconciliationTaskRecord) (storage.EquityReconciliationTaskRecord, error) {
	for _, task := range r.tasks {
		if task.Symbol == record.Symbol {
			return task, nil
		}
	}
	now := time.Now().UTC()
	record.CreatedAt, record.UpdatedAt = now, now
	r.tasks[record.TaskID] = record
	r.enqueues++
	return record, nil
}

func (r *fakeReconciliationRepo) ListEquityReconciliationTasks(context.Context, string, string, string, time.Time) ([]storage.EquityReconciliationTaskRecord, error) {
	tasks := make([]storage.EquityReconciliationTaskRecord, 0, len(r.tasks))
	for _, task := range r.tasks {
		tasks = append(tasks, task)
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].UniverseRank < tasks[j].UniverseRank })
	return tasks, nil
}

func (r *fakeReconciliationRepo) ClaimNextEquityReconciliationTask(_ context.Context, _, _, _ string, _ time.Time, claimedAt time.Time, _ time.Duration) (storage.EquityReconciliationTaskRecord, error) {
	var selected *storage.EquityReconciliationTaskRecord
	for id, task := range r.tasks {
		if task.Status != storage.EquityReconciliationStatusQueued || task.NextAttemptAt.After(claimedAt) {
			continue
		}
		if selected == nil || task.UniverseRank < selected.UniverseRank {
			copy := task
			copy.TaskID = id
			selected = &copy
		}
	}
	if selected == nil {
		return storage.EquityReconciliationTaskRecord{}, storage.ErrNotFound
	}
	selected.Status = storage.EquityReconciliationStatusRunning
	r.tasks[selected.TaskID] = *selected
	r.claims = append(r.claims, selected.Symbol)
	return *selected, nil
}

func (r *fakeReconciliationRepo) UpdateEquityReconciliationTask(_ context.Context, record storage.EquityReconciliationTaskRecord) error {
	r.tasks[record.TaskID] = record
	return nil
}

func (r *fakeReconciliationRepo) RequeueFailedEquityReconciliationTasks(context.Context, string, string, string, time.Time, time.Time) (int, error) {
	count := 0
	for id, task := range r.tasks {
		if task.Status == storage.EquityReconciliationStatusFailed {
			task.Status = storage.EquityReconciliationStatusQueued
			task.ProviderAttempts = 0
			r.tasks[id] = task
			count++
		}
	}
	return count, nil
}

func (r *fakeReconciliationRepo) UpsertIdempotencyRecord(context.Context, storage.IdempotencyRecord) error {
	return nil
}

func (r *fakeReconciliationRepo) UpsertRawEventLedger(_ context.Context, record storage.RawEventLedgerRecord) error {
	r.raw[symbolFromPayload(record.PayloadJSON)] = record
	return nil
}

func (r *fakeReconciliationRepo) PersistPublishedRawEvent(_ context.Context, record storage.RawEventLedgerRecord, _ storage.IdempotencyRecord) error {
	symbol := symbolFromPayload(record.PayloadJSON)
	r.raw[symbol] = record
	return nil
}

func symbolFromPayload(value []byte) string {
	for index := 0; index+12 < len(value); index++ {
		if string(value[index:index+10]) == `"symbol":"` {
			start := index + 10
			for end := start; end < len(value); end++ {
				if value[end] == '"' {
					return string(value[start:end])
				}
			}
		}
	}
	return ""
}

type fakeReconciliationClient struct {
	calls []string
	err   error
}

func (c *fakeReconciliationClient) GetEquityDailyBar(_ context.Context, symbol string, date time.Time) (massive.EquityEODPriceRecord, error) {
	c.calls = append(c.calls, symbol)
	if c.err != nil {
		return massive.EquityEODPriceRecord{}, c.err
	}
	closeValue := 100.0
	return massive.EquityEODPriceRecord{Symbol: symbol, ObservationDate: date, Close: &closeValue}, nil
}

func (c *fakeReconciliationClient) ListOptionContracts(context.Context, string, time.Time, int) ([]massive.OptionContractDailyRecord, error) {
	return nil, errors.New("options must not be called")
}

type fakeReconciliationPublisher struct {
	repo      *fakeReconciliationRepo
	published int
}

func (p *fakeReconciliationPublisher) Publish(_ context.Context, message broker.Message) (broker.PublishResult, error) {
	p.published++
	symbol := symbolFromPayload(message.Value)
	if symbol != "" {
		p.repo.normalized[symbol] = true
	}
	return broker.PublishResult{Topic: message.Topic, Partition: 0, Offset: int64(p.published)}, nil
}

func (p *fakeReconciliationPublisher) Close(context.Context) error { return nil }

func reconciliationTestConfig(dryRun bool) equityReconciliationConfig {
	return equityReconciliationConfig{
		TenantID: "tenant-local", SourceID: "src-massive", UniverseGroup: "top50_megacap",
		Environment: "test", ObservationDate: time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC),
		DryRun: dryRun, AcknowledgeWrites: !dryRun, MaxProviderAttempts: 2,
		MaxProviderRequests: 100, Deadline: time.Second, RetryBackoffs: []time.Duration{time.Millisecond, 2 * time.Millisecond},
		NormalizationPoll: time.Millisecond, LeaseDuration: 10 * time.Millisecond,
	}
}

func TestRunEquityReconciliationProcessesMissingSymbolsSequentially(t *testing.T) {
	repo := newFakeReconciliationRepo("S03", "S15")
	client := &fakeReconciliationClient{}
	publisher := &fakeReconciliationPublisher{repo: repo}
	report, err := runEquityReconciliation(context.Background(), reconciliationTestConfig(false), repo, client, publisher)
	if err != nil {
		t.Fatalf("run reconciliation: %v", err)
	}
	if report.FinalComplete != 50 || report.ProviderRequests != 2 || publisher.published != 2 {
		t.Fatalf("report = %+v, published=%d", report, publisher.published)
	}
	if len(repo.claims) != 2 || repo.claims[0] != "S03" || repo.claims[1] != "S15" {
		t.Fatalf("claims = %v", repo.claims)
	}
}

func TestRunEquityReconciliationDryRunDoesNotMutateOrCallProvider(t *testing.T) {
	repo := newFakeReconciliationRepo("S03")
	client := &fakeReconciliationClient{}
	report, err := runEquityReconciliation(context.Background(), reconciliationTestConfig(true), repo, client, nil)
	if err != nil {
		t.Fatalf("run dry reconciliation: %v", err)
	}
	if report.InitiallyComplete != 49 || len(report.MissingSymbols) != 1 || repo.enqueues != 0 || len(client.calls) != 0 {
		t.Fatalf("report=%+v enqueues=%d provider_calls=%v", report, repo.enqueues, client.calls)
	}
}

func TestRunEquityReconciliationReplaysRawBeforeProvider(t *testing.T) {
	repo := newFakeReconciliationRepo("S03")
	repo.raw["S03"] = storage.RawEventLedgerRecord{
		EventID: "evt-3", IdempotencyKey: "idem-3", BrokerTopic: "signalops.test.raw.v1",
		PayloadJSON: []byte(`{"payload":{"symbol":"S03","observation_date":"2026-07-20"}}`),
	}
	client := &fakeReconciliationClient{}
	publisher := &fakeReconciliationPublisher{repo: repo}
	report, err := runEquityReconciliation(context.Background(), reconciliationTestConfig(false), repo, client, publisher)
	if err != nil {
		t.Fatalf("run raw replay reconciliation: %v", err)
	}
	if report.RawReplays != 1 || report.ProviderRequests != 0 || len(client.calls) != 0 {
		t.Fatalf("report=%+v provider_calls=%v", report, client.calls)
	}
}

func TestRunEquityReconciliationBoundsProviderFailures(t *testing.T) {
	repo := newFakeReconciliationRepo("S03")
	client := &fakeReconciliationClient{err: errors.New("provider unavailable")}
	publisher := &fakeReconciliationPublisher{repo: repo}
	report, err := runEquityReconciliation(context.Background(), reconciliationTestConfig(false), repo, client, publisher)
	if err == nil {
		t.Fatal("expected incomplete reconciliation")
	}
	if report.ProviderRequests != 2 || len(client.calls) != 2 {
		t.Fatalf("report=%+v provider_calls=%v", report, client.calls)
	}
	for _, task := range repo.tasks {
		if task.Status != storage.EquityReconciliationStatusFailed {
			t.Fatalf("task status = %s", task.Status)
		}
	}
}
