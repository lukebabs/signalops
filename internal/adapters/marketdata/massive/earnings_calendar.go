package massive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
	"github.com/lukebabs/signalops/pkg/broker"
	"github.com/lukebabs/signalops/pkg/contracts"
)

const (
	DatasetMarketEventCalendar = "market_event_calendar"
	EventTypeEarningsCalendar  = "market_data.massive.earnings_calendar"
)

type EarningsCalendarRecord struct {
	ProviderEventID string
	ProviderRequestID string
	Symbol string
	CompanyName string
	EventDate time.Time
	KnownAt time.Time
	ProviderLastUpdated *time.Time
	DateStatus string
	EventTime string
	FiscalPeriod string
	FiscalYear *int
	Importance *int
	Raw map[string]any
}

type EarningsCalendarBatch struct {
	Records []EarningsCalendarRecord
	ProviderRequestID string
	PaginationComplete bool
}

type earningsCalendarResponse struct {
	NextURL string `json:"next_url"`
	RequestID json.RawMessage `json:"request_id"`
	Results []earningsCalendarResult `json:"results"`
}

type earningsCalendarResult struct {
	ProviderEventID string `json:"benzinga_id"`
	Symbol string `json:"ticker"`
	CompanyName string `json:"company_name"`
	Date string `json:"date"`
	DateStatus string `json:"date_status"`
	EventTime string `json:"time"`
	FiscalPeriod string `json:"fiscal_period"`
	FiscalYear *int `json:"fiscal_year"`
	Importance *int `json:"importance"`
	LastUpdated string `json:"last_updated"`
	Raw map[string]any `json:"-"`
}

func (r *earningsCalendarResult) UnmarshalJSON(value []byte) error {
	type alias earningsCalendarResult
	var decoded alias
	if err := json.Unmarshal(value, &decoded); err != nil {
		return err
	}
	var raw map[string]any
	if err := json.Unmarshal(value, &raw); err != nil {
		return err
	}
	*r = earningsCalendarResult(decoded)
	r.Raw = raw
	return nil
}

func (c *Client) ListEarningsCalendar(ctx context.Context, startDate, endDate time.Time, limit int) (EarningsCalendarBatch, error) {
	start, err := dayUTC(startDate, "start_date")
	if err != nil {
		return EarningsCalendarBatch{}, err
	}
	end, err := dayUTC(endDate, "end_date")
	if err != nil {
		return EarningsCalendarBatch{}, err
	}
	if end.Before(start) {
		return EarningsCalendarBatch{}, errors.New("earnings calendar end date must not precede start date")
	}
	if limit <= 0 || limit > 5000 {
		limit = 5000
	}
	query := url.Values{}
	query.Set("date.gte", dateKey(start))
	query.Set("date.lte", dateKey(end))
	query.Set("limit", fmt.Sprintf("%d", limit))
	query.Set("sort", "date.asc,ticker.asc,last_updated.asc")
	var response earningsCalendarResponse
	if err := c.getJSON(ctx, "/benzinga/v1/earnings", query, &response); err != nil {
		return EarningsCalendarBatch{}, err
	}
	requestID := strings.Trim(strings.TrimSpace(string(response.RequestID)), "\"")
	records := make([]EarningsCalendarRecord, 0, len(response.Results))
	for _, result := range response.Results {
		symbol := normalizeSymbol(result.Symbol)
		eventDate, parseErr := parseDate(result.Date)
		if parseErr != nil || symbol == "" {
			continue
		}
		var lastUpdated *time.Time
		if value := strings.TrimSpace(result.LastUpdated); value != "" {
			if parsed, parseErr := time.Parse(time.RFC3339, value); parseErr == nil {
				parsed = parsed.UTC()
				lastUpdated = &parsed
			}
		}
		records = append(records, EarningsCalendarRecord{
			ProviderEventID: strings.TrimSpace(result.ProviderEventID),
			ProviderRequestID: requestID,
			Symbol: symbol,
			CompanyName: strings.TrimSpace(result.CompanyName),
			EventDate: eventDate,
			ProviderLastUpdated: lastUpdated,
			DateStatus: strings.ToLower(strings.TrimSpace(result.DateStatus)),
			EventTime: strings.TrimSpace(result.EventTime),
			FiscalPeriod: strings.TrimSpace(result.FiscalPeriod),
			FiscalYear: result.FiscalYear,
			Importance: result.Importance,
			Raw: result.Raw,
		})
	}
	return EarningsCalendarBatch{Records: records, ProviderRequestID: requestID, PaginationComplete: strings.TrimSpace(response.NextURL) == ""}, nil
}

type EarningsCalendarPullConfig struct {
	TenantID string
	SourceID string
	Environment string
	RawTopic string
	KnownAt time.Time
	WindowStart time.Time
	WindowEnd time.Time
	Companies []MegacapCompanySeed
	ProviderLimit int
	MaxEvents int
	DryRun bool
	AcknowledgeWrites bool
	PublishTimeout time.Duration
	PublishRepository storage.PublishRepository
}

type EarningsCalendarPullReport struct {
	KnownAt string `json:"known_at"`
	WindowStart string `json:"window_start"`
	WindowEnd string `json:"window_end"`
	DryRun bool `json:"dry_run"`
	ProviderRequests int `json:"provider_requests"`
	ProviderResults int `json:"provider_results"`
	MatchedEvents int `json:"matched_events"`
	EventsBuilt int `json:"events_built"`
	EventsPublished int `json:"events_published"`
	Topic string `json:"topic"`
	RequestID string `json:"request_id,omitempty"`
}

type EarningsCalendarPullClient interface {
	ListEarningsCalendar(ctx context.Context, startDate, endDate time.Time, limit int) (EarningsCalendarBatch, error)
}

func RunEarningsCalendarPull(ctx context.Context, cfg EarningsCalendarPullConfig, client EarningsCalendarPullClient, publisher broker.Publisher) (EarningsCalendarPullReport, error) {
	if cfg.KnownAt.IsZero() {
		cfg.KnownAt = time.Now().UTC()
	} else {
		cfg.KnownAt = cfg.KnownAt.UTC()
	}
	if strings.TrimSpace(cfg.Environment) == "" {
		cfg.Environment = broker.DefaultEnvironment
	}
	if strings.TrimSpace(cfg.RawTopic) == "" {
		cfg.RawTopic = broker.TopicName(cfg.Environment, broker.RawTopic)
	}
	report := EarningsCalendarPullReport{
		KnownAt: cfg.KnownAt.Format(time.RFC3339Nano),
		WindowStart: dateKey(cfg.WindowStart),
		WindowEnd: dateKey(cfg.WindowEnd),
		DryRun: cfg.DryRun,
		Topic: cfg.RawTopic,
	}
	if client == nil {
		return report, errors.New("earnings calendar client is required")
	}
	if strings.TrimSpace(cfg.TenantID) == "" || strings.TrimSpace(cfg.SourceID) == "" {
		return report, errors.New("tenant id and source id are required")
	}
	if cfg.WindowStart.IsZero() || cfg.WindowEnd.IsZero() || cfg.WindowEnd.Before(cfg.WindowStart) {
		return report, errors.New("valid earnings calendar window is required")
	}
	if cfg.WindowEnd.Sub(cfg.WindowStart) > 366*24*time.Hour {
		return report, errors.New("earnings calendar window must not exceed 366 days")
	}
	if len(cfg.Companies) == 0 || len(cfg.Companies) > 50 {
		return report, errors.New("earnings calendar company count must be between 1 and 50")
	}
	if cfg.MaxEvents <= 0 || cfg.MaxEvents > 500 {
		return report, errors.New("earnings calendar max events must be between 1 and 500")
	}
	if !cfg.DryRun && !cfg.AcknowledgeWrites {
		return report, errors.New("earnings calendar writes require --acknowledge-writes")
	}
	if !cfg.DryRun && publisher == nil {
		return report, errors.New("broker publisher is required when dry run is disabled")
	}
	batch, err := client.ListEarningsCalendar(ctx, cfg.WindowStart, cfg.WindowEnd, cfg.ProviderLimit)
	report.ProviderRequests = 1
	if err != nil {
		return report, fmt.Errorf("fetch Massive earnings calendar: %w", err)
	}
	report.ProviderResults = len(batch.Records)
	report.RequestID = batch.ProviderRequestID
	if !batch.PaginationComplete {
		return report, errors.New("Massive earnings calendar result exceeded the bounded provider limit")
	}
	allowed := make(map[string]struct{}, len(cfg.Companies))
	for _, company := range cfg.Companies {
		allowed[normalizeSymbol(company.Ticker)] = struct{}{}
	}
	records := make([]EarningsCalendarRecord, 0, len(batch.Records))
	seen := map[string]struct{}{}
	for _, record := range batch.Records {
		if _, ok := allowed[normalizeSymbol(record.Symbol)]; !ok {
			continue
		}
		record.KnownAt = cfg.KnownAt
		key := strings.Join([]string{record.Symbol, dateKey(record.EventDate), record.ProviderEventID, record.FiscalPeriod, fmt.Sprint(record.FiscalYear)}, "|")
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		left := strings.Join([]string{records[i].Symbol, dateKey(records[i].EventDate), records[i].ProviderEventID}, "|")
		right := strings.Join([]string{records[j].Symbol, dateKey(records[j].EventDate), records[j].ProviderEventID}, "|")
		return left < right
	})
	report.MatchedEvents = len(records)
	if len(records) > cfg.MaxEvents {
		return report, fmt.Errorf("earnings calendar matched %d events; max-events is %d", len(records), cfg.MaxEvents)
	}
	scheduledCfg := normalizeScheduledPullConfig(ScheduledPullConfig{
		TenantID: cfg.TenantID,
		SourceID: cfg.SourceID,
		Environment: cfg.Environment,
		RawTopic: cfg.RawTopic,
		ObservationDate: cfg.KnownAt,
		ProcessingAt: cfg.KnownAt,
		DryRun: cfg.DryRun,
		PublishTimeout: cfg.PublishTimeout,
		MaxEventsBuilt: cfg.MaxEvents,
		MaxEventsPublished: cfg.MaxEvents,
		PublishRepository: cfg.PublishRepository,
	})
	scheduledReport := ScheduledPullReport{DryRun: cfg.DryRun, Topic: cfg.RawTopic, EventsByDataset: map[string]int{}}
	for _, record := range records {
		record := record
		if err := buildAndMaybePublish(ctx, scheduledCfg, publisher, &scheduledReport, DatasetMarketEventCalendar, func(adapterCfg AdapterConfig) (contracts.RawSignalEvent, error) {
			return BuildEarningsCalendarEvent(adapterCfg, record)
		}); err != nil {
			return report, fmt.Errorf("%s earnings event: %w", record.Symbol, err)
		}
	}
	report.EventsBuilt = scheduledReport.EventsBuilt
	report.EventsPublished = scheduledReport.EventsPublished
	return report, nil
}

func BuildEarningsCalendarEvent(cfg AdapterConfig, record EarningsCalendarRecord) (contracts.RawSignalEvent, error) {
	if err := validateConfig(cfg); err != nil {
		return contracts.RawSignalEvent{}, err
	}
	symbol := normalizeSymbol(record.Symbol)
	if symbol == "" {
		return contracts.RawSignalEvent{}, errors.New("earnings calendar symbol is required")
	}
	eventDate, err := dayUTC(record.EventDate, "event_date")
	if err != nil {
		return contracts.RawSignalEvent{}, err
	}
	knownAt := record.KnownAt.UTC()
	if knownAt.IsZero() {
		knownAt = processingTime(cfg)
	}
	providerID := strings.TrimSpace(record.ProviderEventID)
	identity := providerID
	if identity == "" {
		identity = strings.Join([]string{symbol, dateKey(eventDate), record.FiscalPeriod, fmt.Sprint(record.FiscalYear)}, "|")
	}
	eventID := stableID("evt", cfg.TenantID, cfg.SourceID, DatasetMarketEventCalendar, identity, dateKey(eventDate), dateKey(knownAt))
	idempotencyKey := stableID("idem", cfg.TenantID, cfg.SourceID, DatasetMarketEventCalendar, identity, dateKey(eventDate), dateKey(knownAt))
	payload := map[string]any{
		"provider": "massive_benzinga",
		"dataset": DatasetMarketEventCalendar,
		"provider_event_id": providerID,
		"provider_request_id": strings.TrimSpace(record.ProviderRequestID),
		"symbol": symbol,
		"event_type": "earnings",
		"event_date": dateKey(eventDate),
		"known_at": knownAt.Format(time.RFC3339Nano),
	}
	for key, value := range map[string]string{
		"company_name": record.CompanyName,
		"date_status": record.DateStatus,
		"event_time": record.EventTime,
		"fiscal_period": record.FiscalPeriod,
	} {
		if value = strings.TrimSpace(value); value != "" {
			payload[key] = value
		}
	}
	if record.FiscalYear != nil {
		payload["fiscal_year"] = *record.FiscalYear
	}
	if record.Importance != nil {
		payload["importance"] = *record.Importance
	}
	if record.ProviderLastUpdated != nil && !record.ProviderLastUpdated.IsZero() {
		payload["provider_last_updated"] = record.ProviderLastUpdated.UTC().Format(time.RFC3339Nano)
	}
	if record.Raw != nil {
		payload["raw"] = record.Raw
	}
	return contracts.RawSignalEvent{
		EventEnvelope: contracts.EventEnvelope{
			TenantID: strings.TrimSpace(cfg.TenantID),
			SourceID: strings.TrimSpace(cfg.SourceID),
			AppID: "marketops",
			Domain: contracts.SourceDomainMarketData,
			UseCase: "daily_market_surveillance",
			SourceDomain: contracts.SourceDomainMarketData,
			SourceAdapter: AdapterID,
			IngestionMode: contracts.IngestionModeScheduledPull,
			Dataset: DatasetMarketEventCalendar,
			EventID: eventID,
			EventType: EventTypeEarningsCalendar,
			SchemaID: RawSignalSchemaID,
			SchemaVersion: RawSignalSchemaVersion,
			ObservationAt: knownAt,
			EffectiveAt: eventDate,
			ProcessingAt: processingTime(cfg),
			OccurredAt: eventDate,
			ObservedAt: knownAt,
			Metadata: metadata(DatasetMarketEventCalendar),
			CorrelationID: correlationID(cfg, eventID),
			IdempotencyKey: idempotencyKey,
			TraceID: strings.TrimSpace(cfg.TraceID),
		},
		Payload: payload,
		EntityHints: []contracts.EntityHint{{Type: "ticker", ExternalID: symbol}},
	}, nil
}
