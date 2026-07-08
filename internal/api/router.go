package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
	"github.com/lukebabs/signalops/pkg/broker"
)

const (
	defaultMaxRawEventBytes = 1 << 20
	defaultPublishTimeout   = 5 * time.Second
)

// RouterConfig contains process-local API wiring options.
type RouterConfig struct {
	ServiceName     string
	Publisher       broker.Publisher
	RawTopic        string
	QueryRepository storage.QueryRepository
}

// NewRouter creates the HTTP routes owned by the SignalOps gateway.
func NewRouter(cfg RouterConfig) http.Handler {
	mux := http.NewServeMux()
	serviceName := cfg.ServiceName
	if serviceName == "" {
		serviceName = "signalops"
	}
	rawTopic := cfg.RawTopic

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"service": serviceName,
			"time":    time.Now().UTC().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "ready",
			"service": serviceName,
			"time":    time.Now().UTC().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("GET /v1/scheduler/runs", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		runs, err := repo.ListSchedulerRuns(r.Context(), queryLimit(r, 50))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list scheduler runs")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"runs": schedulerRunResponses(runs)})
	})

	mux.HandleFunc("GET /v1/scheduler/runs/{run_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		record, err := repo.GetSchedulerRun(r.Context(), r.PathValue("run_id"))
		if err != nil {
			writeQueryError(w, err, "scheduler_run_not_found", "scheduler run not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"run": schedulerRunResponse(record)})
	})

	mux.HandleFunc("GET /v1/provider-usage", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		records, err := repo.ListProviderUsage(r.Context(), strings.TrimSpace(r.URL.Query().Get("run_id")), queryLimit(r, 50))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list provider usage")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"provider_usage": providerUsageResponses(records)})
	})

	mux.HandleFunc("GET /v1/raw-events", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		records, err := repo.ListRawEventLedger(r.Context(), storage.RawEventLedgerFilter{
			TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")),
			SourceID: strings.TrimSpace(r.URL.Query().Get("source_id")),
			Dataset:  strings.TrimSpace(r.URL.Query().Get("dataset")),
			Limit:    queryLimit(r, 50),
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list raw events")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"raw_events": rawEventResponses(records)})
	})

	mux.HandleFunc("GET /v1/raw-events/{event_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		record, err := repo.GetRawEventLedger(r.Context(), r.PathValue("event_id"))
		if err != nil {
			writeQueryError(w, err, "raw_event_not_found", "raw event not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"raw_event": rawEventResponse(record)})
	})

	mux.HandleFunc("GET /v1/idempotency", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		sourceID := strings.TrimSpace(r.URL.Query().Get("source_id"))
		key := strings.TrimSpace(r.URL.Query().Get("idempotency_key"))
		if tenantID == "" || sourceID == "" || key == "" {
			writeError(w, http.StatusBadRequest, "missing_query", "tenant_id, source_id, and idempotency_key are required")
			return
		}
		record, err := repo.GetIdempotencyRecord(r.Context(), tenantID, sourceID, key)
		if err != nil {
			writeQueryError(w, err, "idempotency_not_found", "idempotency record not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"idempotency": idempotencyResponse(record)})
	})

	mux.HandleFunc("POST /v1/events/raw", func(w http.ResponseWriter, r *http.Request) {
		if cfg.Publisher == nil || rawTopic == "" {
			writeError(w, http.StatusServiceUnavailable, "broker_unavailable", "raw event ingestion is not configured")
			return
		}

		payload, fields, err := readJSONObject(w, r, defaultMaxRawEventBytes)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}

		eventID := firstNonEmpty(headerValue(r, "X-SignalOps-Event-ID"), jsonStringField(fields, "event_id"), newID("evt"))
		idempotencyKey := firstNonEmpty(headerValue(r, "X-Idempotency-Key"), jsonStringField(fields, "idempotency_key"), eventID)
		correlationID := firstNonEmpty(headerValue(r, "X-Correlation-ID"), jsonStringField(fields, "correlation_id"), newID("corr"))
		causationID := firstNonEmpty(headerValue(r, "X-Causation-ID"), jsonStringField(fields, "causation_id"))
		traceID := firstNonEmpty(headerValue(r, "X-Trace-ID"), jsonStringField(fields, "trace_id"))

		ctx, cancel := context.WithTimeout(r.Context(), defaultPublishTimeout)
		defer cancel()

		result, err := cfg.Publisher.Publish(ctx, broker.Message{
			Topic:         rawTopic,
			Key:           idempotencyKey,
			Value:         payload,
			Headers:       rawEventHeaders(eventID, idempotencyKey),
			CorrelationID: correlationID,
			CausationID:   causationID,
			TraceID:       traceID,
			PublishedAt:   time.Now().UTC(),
		})
		if err != nil {
			writeError(w, http.StatusBadGateway, "publish_failed", "failed to publish raw event")
			return
		}

		writeJSON(w, http.StatusAccepted, map[string]any{
			"status":          "accepted",
			"event_id":        eventID,
			"idempotency_key": idempotencyKey,
			"correlation_id":  correlationID,
			"topic":           result.Topic,
			"partition":       result.Partition,
			"offset":          result.Offset,
		})
	})

	return mux
}

type schedulerRunDTO struct {
	RunID            string          `json:"run_id"`
	TenantID         string          `json:"tenant_id"`
	SourceID         string          `json:"source_id"`
	SourceAdapter    string          `json:"source_adapter"`
	Datasets         []string        `json:"datasets"`
	ObservationDate  time.Time       `json:"observation_date"`
	DryRun           bool            `json:"dry_run"`
	Status           string          `json:"status"`
	StartedAt        time.Time       `json:"started_at"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty"`
	EventsBuilt      int             `json:"events_built"`
	EventsPublished  int             `json:"events_published"`
	ProviderRequests int             `json:"provider_requests"`
	ProviderRetries  int             `json:"provider_retries"`
	Failures         int             `json:"failures"`
	Config           json.RawMessage `json:"config"`
	Report           json.RawMessage `json:"report"`
	ErrorMessage     string          `json:"error_message,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type providerUsageDTO struct {
	UsageID      string          `json:"usage_id"`
	RunID        string          `json:"run_id"`
	Provider     string          `json:"provider"`
	Dataset      string          `json:"dataset"`
	RequestCount int             `json:"request_count"`
	RetryCount   int             `json:"retry_count"`
	EventCount   int             `json:"event_count"`
	Budget       json.RawMessage `json:"budget"`
	CreatedAt    time.Time       `json:"created_at"`
}

type rawEventDTO struct {
	EventID         string          `json:"event_id"`
	TenantID        string          `json:"tenant_id"`
	SourceID        string          `json:"source_id"`
	SourceAdapter   string          `json:"source_adapter"`
	Dataset         string          `json:"dataset"`
	IdempotencyKey  string          `json:"idempotency_key"`
	ObservationTime time.Time       `json:"observation_time"`
	ProcessingTime  time.Time       `json:"processing_time"`
	BrokerTopic     string          `json:"broker_topic,omitempty"`
	BrokerPartition *int32          `json:"broker_partition,omitempty"`
	BrokerOffset    *int64          `json:"broker_offset,omitempty"`
	Payload         json.RawMessage `json:"payload"`
	EntityHints     json.RawMessage `json:"entity_hints"`
	CreatedAt       time.Time       `json:"created_at"`
}

type idempotencyDTO struct {
	TenantID       string          `json:"tenant_id"`
	SourceID       string          `json:"source_id"`
	IdempotencyKey string          `json:"idempotency_key"`
	EventID        string          `json:"event_id"`
	SourceAdapter  string          `json:"source_adapter"`
	Dataset        string          `json:"dataset"`
	Topic          string          `json:"topic,omitempty"`
	Partition      *int32          `json:"partition,omitempty"`
	Offset         *int64          `json:"offset,omitempty"`
	PayloadHash    string          `json:"payload_hash,omitempty"`
	Status         string          `json:"status"`
	Metadata       json.RawMessage `json:"metadata"`
	FirstSeenAt    time.Time       `json:"first_seen_at"`
	LastSeenAt     time.Time       `json:"last_seen_at"`
}

func schedulerRunResponses(records []storage.SchedulerRunRecord) []schedulerRunDTO {
	items := make([]schedulerRunDTO, 0, len(records))
	for _, record := range records {
		items = append(items, schedulerRunResponse(record))
	}
	return items
}

func schedulerRunResponse(record storage.SchedulerRunRecord) schedulerRunDTO {
	return schedulerRunDTO{
		RunID:            record.RunID,
		TenantID:         record.TenantID,
		SourceID:         record.SourceID,
		SourceAdapter:    record.SourceAdapter,
		Datasets:         record.Datasets,
		ObservationDate:  record.ObservationDate,
		DryRun:           record.DryRun,
		Status:           record.Status,
		StartedAt:        record.StartedAt,
		CompletedAt:      record.CompletedAt,
		EventsBuilt:      record.EventsBuilt,
		EventsPublished:  record.EventsPublished,
		ProviderRequests: record.ProviderRequests,
		ProviderRetries:  record.ProviderRetries,
		Failures:         record.Failures,
		Config:           jsonRawOrEmptyObject(record.ConfigJSON),
		Report:           jsonRawOrEmptyObject(record.ReportJSON),
		ErrorMessage:     record.ErrorMessage,
		CreatedAt:        record.CreatedAt,
		UpdatedAt:        record.UpdatedAt,
	}
}

func providerUsageResponses(records []storage.ProviderUsageRecord) []providerUsageDTO {
	items := make([]providerUsageDTO, 0, len(records))
	for _, record := range records {
		items = append(items, providerUsageDTO{
			UsageID:      record.UsageID,
			RunID:        record.RunID,
			Provider:     record.Provider,
			Dataset:      record.Dataset,
			RequestCount: record.RequestCount,
			RetryCount:   record.RetryCount,
			EventCount:   record.EventCount,
			Budget:       jsonRawOrEmptyObject(record.BudgetJSON),
			CreatedAt:    record.CreatedAt,
		})
	}
	return items
}

func rawEventResponses(records []storage.RawEventLedgerRecord) []rawEventDTO {
	items := make([]rawEventDTO, 0, len(records))
	for _, record := range records {
		items = append(items, rawEventResponse(record))
	}
	return items
}

func rawEventResponse(record storage.RawEventLedgerRecord) rawEventDTO {
	return rawEventDTO{
		EventID:         record.EventID,
		TenantID:        record.TenantID,
		SourceID:        record.SourceID,
		SourceAdapter:   record.SourceAdapter,
		Dataset:         record.Dataset,
		IdempotencyKey:  record.IdempotencyKey,
		ObservationTime: record.ObservationTime,
		ProcessingTime:  record.ProcessingTime,
		BrokerTopic:     record.BrokerTopic,
		BrokerPartition: record.BrokerPartition,
		BrokerOffset:    record.BrokerOffset,
		Payload:         jsonRawOrEmptyObject(record.PayloadJSON),
		EntityHints:     jsonRawOrEmptyArray(record.EntityHintsJSON),
		CreatedAt:       record.CreatedAt,
	}
}

func idempotencyResponse(record storage.IdempotencyRecord) idempotencyDTO {
	return idempotencyDTO{
		TenantID:       record.TenantID,
		SourceID:       record.SourceID,
		IdempotencyKey: record.IdempotencyKey,
		EventID:        record.EventID,
		SourceAdapter:  record.SourceAdapter,
		Dataset:        record.Dataset,
		Topic:          record.Topic,
		Partition:      record.Partition,
		Offset:         record.Offset,
		PayloadHash:    record.PayloadHash,
		Status:         record.Status,
		Metadata:       jsonRawOrEmptyObject(record.MetadataJSON),
		FirstSeenAt:    record.FirstSeenAt,
		LastSeenAt:     record.LastSeenAt,
	}
}

func jsonRawOrEmptyObject(value []byte) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(value)
}

func jsonRawOrEmptyArray(value []byte) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(`[]`)
	}
	return json.RawMessage(value)
}

func requireQueryRepository(w http.ResponseWriter, repo storage.QueryRepository) (storage.QueryRepository, bool) {
	if repo == nil {
		writeError(w, http.StatusServiceUnavailable, "storage_unavailable", "query storage is not configured")
		return nil, false
	}
	return repo, true
}

func writeQueryError(w http.ResponseWriter, err error, notFoundCode string, notFoundMessage string) {
	if errors.Is(err, storage.ErrNotFound) {
		writeError(w, http.StatusNotFound, notFoundCode, notFoundMessage)
		return
	}
	writeError(w, http.StatusInternalServerError, "query_failed", "query failed")
}

func queryLimit(r *http.Request, fallback int) int {
	value := strings.TrimSpace(r.URL.Query().Get("limit"))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	if parsed > 200 {
		return 200
	}
	return parsed
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{
		"error":   code,
		"message": message,
	})
}

func readJSONObject(w http.ResponseWriter, r *http.Request, maxBytes int64) ([]byte, map[string]json.RawMessage, error) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBytes))
	if err != nil {
		return nil, nil, fmt.Errorf("request body exceeds %d bytes or cannot be read", maxBytes)
	}
	defer r.Body.Close()

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil, nil, errors.New("request body must be a valid JSON object")
	}
	if fields == nil {
		return nil, nil, errors.New("request body must be a JSON object")
	}

	return body, fields, nil
}

func rawEventHeaders(eventID, idempotencyKey string) map[string]string {
	return map[string]string{
		"content_type":            "application/json",
		"signalops_event_id":      eventID,
		"signalops_idempotency":   idempotencyKey,
		"signalops_ingest_route":  "/v1/events/raw",
		"signalops_ingest_format": "raw_signal_event.v1",
	}
}

func jsonStringField(fields map[string]json.RawMessage, key string) string {
	value, ok := fields[key]
	if !ok {
		return ""
	}

	var decoded string
	if err := json.Unmarshal(value, &decoded); err != nil {
		return ""
	}
	return strings.TrimSpace(decoded)
}

func headerValue(r *http.Request, key string) string {
	return strings.TrimSpace(r.Header.Get(key))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func newID(prefix string) string {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UTC().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(buf[:])
}
