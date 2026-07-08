package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
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
	defaultStreamInterval   = 5 * time.Second
)

var supportedDashboardStreamChannels = map[string]struct{}{
	"health":         {},
	"scheduler_run":  {},
	"runs":           {},
	"raw_event":      {},
	"raw_events":     {},
	"provider_usage": {},
	"heartbeat":      {},
}

// RouterConfig contains process-local API wiring options.
type RouterConfig struct {
	ServiceName       string
	Publisher         broker.Publisher
	RawTopic          string
	QueryRepository   storage.QueryRepository
	PublishRepository storage.PublishRepository
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

	mux.HandleFunc("GET /v1/normalized-events", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		records, err := repo.ListNormalizedEventLedger(r.Context(), storage.RawEventLedgerFilter{
			TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), SourceID: strings.TrimSpace(r.URL.Query().Get("source_id")),
			Dataset: strings.TrimSpace(r.URL.Query().Get("dataset")), Limit: queryLimit(r, 50),
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list normalized events")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"normalized_events": normalizedEventResponses(records)})
	})

	mux.HandleFunc("GET /v1/normalized-events/{event_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		record, err := repo.GetNormalizedEventLedger(r.Context(), r.PathValue("event_id"))
		if err != nil {
			writeQueryError(w, err, "normalized_event_not_found", "normalized event not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"normalized_event": normalizedEventResponse(record)})
	})

	mux.HandleFunc("GET /v1/signals", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		records, err := repo.ListSignalLedger(r.Context(), storage.SignalLedgerFilter{
			TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), SourceID: strings.TrimSpace(r.URL.Query().Get("source_id")),
			Dataset: strings.TrimSpace(r.URL.Query().Get("dataset")), DetectorID: strings.TrimSpace(r.URL.Query().Get("detector_id")),
			Severity: strings.TrimSpace(r.URL.Query().Get("severity")), Limit: queryLimit(r, 50),
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list signals")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"signals": signalResponses(records)})
	})

	mux.HandleFunc("GET /v1/signals/{signal_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		record, err := repo.GetSignalLedger(r.Context(), r.PathValue("signal_id"))
		if err != nil {
			writeQueryError(w, err, "signal_not_found", "signal not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"signal": signalResponse(record)})
	})

	mux.HandleFunc("GET /v1/alerts", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		records, err := repo.ListAlertLedger(r.Context(), storage.AlertLedgerFilter{
			TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), SourceID: strings.TrimSpace(r.URL.Query().Get("source_id")),
			Dataset: strings.TrimSpace(r.URL.Query().Get("dataset")), Severity: strings.TrimSpace(r.URL.Query().Get("severity")),
			Status: strings.TrimSpace(r.URL.Query().Get("status")), Limit: queryLimit(r, 50),
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list alerts")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"alerts": alertResponses(records)})
	})

	mux.HandleFunc("GET /v1/alerts/{alert_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		record, err := repo.GetAlertLedger(r.Context(), r.PathValue("alert_id"))
		if err != nil {
			writeQueryError(w, err, "alert_not_found", "alert not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"alert": alertResponse(record)})
	})

	mux.HandleFunc("GET /v1/insights", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		records, err := repo.ListInsightLedger(r.Context(), storage.InsightLedgerFilter{
			TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), SourceID: strings.TrimSpace(r.URL.Query().Get("source_id")),
			Dataset: strings.TrimSpace(r.URL.Query().Get("dataset")), InsightType: strings.TrimSpace(r.URL.Query().Get("insight_type")),
			Status: strings.TrimSpace(r.URL.Query().Get("status")), Limit: queryLimit(r, 50),
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list insights")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"insights": insightResponses(records)})
	})

	mux.HandleFunc("GET /v1/insights/{insight_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		record, err := repo.GetInsightLedger(r.Context(), r.PathValue("insight_id"))
		if err != nil {
			writeQueryError(w, err, "insight_not_found", "insight not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"insight": insightResponse(record)})
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

	mux.HandleFunc("GET /v1/tenants/{tenant_id}/catalog/sources", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		tenantID := strings.TrimSpace(r.PathValue("tenant_id"))
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "missing_path", "tenant_id is required")
			return
		}
		sources, err := repo.ListCatalogSources(r.Context(), tenantID, queryLimit(r, 50))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list catalog sources")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"sources": catalogSourceResponses(sources)})
	})

	mux.HandleFunc("GET /v1/tenants/{tenant_id}/catalog/pipelines", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		tenantID := strings.TrimSpace(r.PathValue("tenant_id"))
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "missing_path", "tenant_id is required")
			return
		}
		pipelines, err := repo.ListCatalogPipelines(r.Context(), tenantID, queryLimit(r, 50))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list catalog pipelines")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"pipelines": catalogPipelineResponses(pipelines)})
	})

	mux.HandleFunc("GET /v1/tenants/{tenant_id}/catalog/rules", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		tenantID := strings.TrimSpace(r.PathValue("tenant_id"))
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "missing_path", "tenant_id is required")
			return
		}
		rules, err := repo.ListCatalogRules(r.Context(), tenantID, queryLimit(r, 50))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list catalog rules")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"rules": catalogRuleResponses(rules)})
	})

	mux.HandleFunc("GET /v1/streams/dashboard", func(w http.ResponseWriter, r *http.Request) {
		channels, err := dashboardStreamChannels(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_channel", err.Error())
			return
		}
		streamDashboard(w, r, serviceName, cfg.QueryRepository, channels, defaultStreamInterval)
	})

	mux.HandleFunc("POST /v1/events/raw", func(w http.ResponseWriter, r *http.Request) {
		if cfg.Publisher == nil || rawTopic == "" || cfg.PublishRepository == nil {
			writeError(w, http.StatusServiceUnavailable, "ingest_unavailable", "raw event ingestion is not fully configured")
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
		acceptedAt := time.Now().UTC()
		ingest, err := rawIngestPersistenceFields(fields, acceptedAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_event", err.Error())
			return
		}

		publishCtx, cancel := context.WithTimeout(r.Context(), defaultPublishTimeout)
		result, err := cfg.Publisher.Publish(publishCtx, broker.Message{
			Topic:         rawTopic,
			Key:           idempotencyKey,
			Value:         payload,
			Headers:       rawEventHeaders(eventID, idempotencyKey),
			CorrelationID: correlationID,
			CausationID:   causationID,
			TraceID:       traceID,
			PublishedAt:   acceptedAt,
		})
		cancel()
		if err != nil {
			writeError(w, http.StatusBadGateway, "publish_failed", "failed to publish raw event")
			return
		}
		ledger, idempotency, err := publishedRawEventRecords(payload, ingest, eventID, idempotencyKey, correlationID, causationID, traceID, result, acceptedAt)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "persistence_mapping_failed", "failed to map published raw event")
			return
		}
		persistCtx, persistCancel := context.WithTimeout(r.Context(), defaultPublishTimeout)
		err = cfg.PublishRepository.PersistPublishedRawEvent(persistCtx, ledger, idempotency)
		persistCancel()
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "persistence_failed", "raw event was published but its audit state could not be persisted")
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

type rawIngestFields struct {
	TenantID        string
	SourceID        string
	SourceAdapter   string
	Dataset         string
	ObservationTime time.Time
	ProcessingTime  time.Time
	EntityHintsJSON []byte
}

func rawIngestPersistenceFields(fields map[string]json.RawMessage, acceptedAt time.Time) (rawIngestFields, error) {
	result := rawIngestFields{
		TenantID:        jsonStringField(fields, "tenant_id"),
		SourceID:        jsonStringField(fields, "source_id"),
		SourceAdapter:   jsonStringField(fields, "source_adapter"),
		Dataset:         jsonStringField(fields, "dataset"),
		ProcessingTime:  acceptedAt,
		EntityHintsJSON: []byte("[]"),
	}
	for name, value := range map[string]string{"tenant_id": result.TenantID, "source_id": result.SourceID, "source_adapter": result.SourceAdapter, "dataset": result.Dataset} {
		if value == "" {
			return rawIngestFields{}, fmt.Errorf("%s is required", name)
		}
	}
	observationTime, err := parseEventTime(fields, "observation_time")
	if err != nil {
		return rawIngestFields{}, err
	}
	result.ObservationTime = observationTime
	if jsonStringField(fields, "processing_time") != "" {
		result.ProcessingTime, err = parseEventTime(fields, "processing_time")
		if err != nil {
			return rawIngestFields{}, err
		}
	}
	if raw, ok := fields["entity_hints"]; ok {
		var hints []json.RawMessage
		if err := json.Unmarshal(raw, &hints); err != nil {
			return rawIngestFields{}, errors.New("entity_hints must be an array")
		}
		result.EntityHintsJSON = append([]byte(nil), raw...)
	}
	return result, nil
}

func parseEventTime(fields map[string]json.RawMessage, name string) (time.Time, error) {
	value := jsonStringField(fields, name)
	if value == "" {
		return time.Time{}, fmt.Errorf("%s is required", name)
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be an RFC3339 timestamp", name)
	}
	return parsed.UTC(), nil
}

func publishedRawEventRecords(payload []byte, ingest rawIngestFields, eventID, idempotencyKey, correlationID, causationID, traceID string, result broker.PublishResult, publishedAt time.Time) (storage.RawEventLedgerRecord, storage.IdempotencyRecord, error) {
	partition, offset := result.Partition, result.Offset
	metadata, err := json.Marshal(map[string]any{
		"correlation_id": correlationID,
		"causation_id":   causationID,
		"trace_id":       traceID,
		"route":          "/v1/events/raw",
		"published_at":   publishedAt.Format(time.RFC3339Nano),
	})
	if err != nil {
		return storage.RawEventLedgerRecord{}, storage.IdempotencyRecord{}, err
	}
	hash := sha256.Sum256(payload)
	ledger := storage.RawEventLedgerRecord{
		EventID: eventID, TenantID: ingest.TenantID, SourceID: ingest.SourceID,
		SourceAdapter: ingest.SourceAdapter, Dataset: ingest.Dataset, IdempotencyKey: idempotencyKey,
		ObservationTime: ingest.ObservationTime, ProcessingTime: ingest.ProcessingTime,
		BrokerTopic: result.Topic, BrokerPartition: &partition, BrokerOffset: &offset,
		PayloadJSON: payload, EntityHintsJSON: ingest.EntityHintsJSON,
	}
	idempotency := storage.IdempotencyRecord{
		TenantID: ingest.TenantID, SourceID: ingest.SourceID, IdempotencyKey: idempotencyKey,
		EventID: eventID, SourceAdapter: ingest.SourceAdapter, Dataset: ingest.Dataset,
		Topic: result.Topic, Partition: &partition, Offset: &offset,
		PayloadHash: "sha256:" + hex.EncodeToString(hash[:]), Status: storage.IdempotencyStatusPublished,
		MetadataJSON: metadata,
	}
	return ledger, idempotency, nil
}

type streamChannelSet map[string]bool

type sseEvent struct {
	Event string
	ID    string
	Data  any
}

func dashboardStreamChannels(r *http.Request) (streamChannelSet, error) {
	value := strings.TrimSpace(r.URL.Query().Get("channels"))
	if value == "" {
		return streamChannelSet{
			"health":         true,
			"scheduler_run":  true,
			"raw_event":      true,
			"provider_usage": true,
			"heartbeat":      true,
		}, nil
	}

	channels := streamChannelSet{}
	for _, part := range strings.Split(value, ",") {
		channel := strings.TrimSpace(part)
		if channel == "" {
			continue
		}
		if _, ok := supportedDashboardStreamChannels[channel]; !ok {
			return nil, fmt.Errorf("unsupported stream channel %q", channel)
		}
		switch channel {
		case "runs":
			channel = "scheduler_run"
		case "raw_events":
			channel = "raw_event"
		}
		channels[channel] = true
	}
	if len(channels) == 0 {
		return nil, errors.New("at least one stream channel is required")
	}
	return channels, nil
}

func streamDashboard(w http.ResponseWriter, r *http.Request, serviceName string, repo storage.QueryRepository, channels streamChannelSet, interval time.Duration) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming_unsupported", "response streaming is not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	seen := map[string]struct{}{}
	emit := func(event sseEvent) bool {
		if err := writeSSE(w, event); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	if !emit(sseEvent{Event: "heartbeat", Data: heartbeatPayload(serviceName)}) {
		return
	}

	if repo == nil {
		if channels["health"] {
			if !emit(sseEvent{Event: "error", Data: map[string]string{
				"error":   "storage_unavailable",
				"message": "query storage is not configured",
			}}) {
				return
			}
		}
		streamHeartbeatsUntilCanceled(r, serviceName, interval, emit)
		return
	}

	if !emitDashboardSnapshot(r.Context(), repo, serviceName, channels, seen, emit) {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if channels["heartbeat"] && !emit(sseEvent{Event: "heartbeat", Data: heartbeatPayload(serviceName)}) {
				return
			}
			if !emitDashboardSnapshot(r.Context(), repo, serviceName, channels, seen, emit) {
				return
			}
		}
	}
}

func emitDashboardSnapshot(ctx context.Context, repo storage.QueryRepository, serviceName string, channels streamChannelSet, seen map[string]struct{}, emit func(sseEvent) bool) bool {
	if channels["health"] {
		if !emit(sseEvent{Event: "health", Data: map[string]string{
			"status":  "ok",
			"service": serviceName,
			"time":    time.Now().UTC().Format(time.RFC3339),
		}}) {
			return false
		}
	}
	if channels["scheduler_run"] {
		runs, err := repo.ListSchedulerRuns(ctx, 50)
		if err != nil {
			return emit(sseEvent{Event: "error", Data: map[string]string{"error": "query_failed", "message": "failed to list scheduler runs"}})
		}
		for _, run := range runs {
			key := "scheduler_run:" + run.RunID
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			if !emit(sseEvent{Event: "scheduler_run", ID: run.RunID, Data: schedulerRunResponse(run)}) {
				return false
			}
		}
	}
	if channels["raw_event"] {
		records, err := repo.ListRawEventLedger(ctx, storage.RawEventLedgerFilter{Limit: 50})
		if err != nil {
			return emit(sseEvent{Event: "error", Data: map[string]string{"error": "query_failed", "message": "failed to list raw events"}})
		}
		for _, record := range records {
			key := "raw_event:" + record.EventID
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			if !emit(sseEvent{Event: "raw_event", ID: record.EventID, Data: rawEventResponse(record)}) {
				return false
			}
		}
	}
	if channels["provider_usage"] {
		records, err := repo.ListProviderUsage(ctx, "", 50)
		if err != nil {
			return emit(sseEvent{Event: "error", Data: map[string]string{"error": "query_failed", "message": "failed to list provider usage"}})
		}
		for _, record := range records {
			key := "provider_usage:" + record.UsageID
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			if !emit(sseEvent{Event: "provider_usage", ID: record.UsageID, Data: providerUsageResponses([]storage.ProviderUsageRecord{record})[0]}) {
				return false
			}
		}
	}
	return true
}

func heartbeatPayload(serviceName string) map[string]string {
	return map[string]string{
		"status":  "alive",
		"service": serviceName,
		"time":    time.Now().UTC().Format(time.RFC3339),
	}
}

func streamHeartbeatsUntilCanceled(r *http.Request, serviceName string, interval time.Duration, emit func(sseEvent) bool) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if !emit(sseEvent{Event: "heartbeat", Data: heartbeatPayload(serviceName)}) {
				return
			}
		}
	}
}

func writeSSE(w io.Writer, event sseEvent) error {
	if event.Event != "" {
		if _, err := fmt.Fprintf(w, "event: %s\n", event.Event); err != nil {
			return err
		}
	}
	if event.ID != "" {
		if _, err := fmt.Fprintf(w, "id: %s\n", event.ID); err != nil {
			return err
		}
	}
	data, err := json.Marshal(event.Data)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
		return err
	}
	return nil
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

type normalizedEventDTO struct {
	EventID             string          `json:"event_id"`
	TenantID            string          `json:"tenant_id"`
	SourceID            string          `json:"source_id"`
	SourceAdapter       string          `json:"source_adapter"`
	Dataset             string          `json:"dataset"`
	IdempotencyKey      string          `json:"idempotency_key"`
	SchemaID            string          `json:"schema_id"`
	SchemaVersion       string          `json:"schema_version"`
	ObservationTime     time.Time       `json:"observation_time"`
	ProcessingTime      time.Time       `json:"processing_time"`
	Confidence          float64         `json:"confidence"`
	RawTopic            string          `json:"raw_topic"`
	RawPartition        int32           `json:"raw_partition"`
	RawOffset           int64           `json:"raw_offset"`
	NormalizedTopic     string          `json:"normalized_topic"`
	NormalizedPartition int32           `json:"normalized_partition"`
	NormalizedOffset    int64           `json:"normalized_offset"`
	NormalizedPayload   json.RawMessage `json:"normalized_payload"`
	Entities            json.RawMessage `json:"entities"`
	Evidence            json.RawMessage `json:"evidence"`
	Metadata            json.RawMessage `json:"metadata"`
	Event               json.RawMessage `json:"event"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

type signalDTO struct {
	SignalID          string          `json:"signal_id"`
	TenantID          string          `json:"tenant_id"`
	SourceID          string          `json:"source_id"`
	SourceDomain      string          `json:"source_domain"`
	SourceAdapter     string          `json:"source_adapter"`
	IngestionMode     string          `json:"ingestion_mode"`
	Dataset           string          `json:"dataset"`
	EventIDs          []string        `json:"event_ids"`
	ArtifactIDs       []string        `json:"artifact_ids"`
	SignalType        string          `json:"signal_type"`
	DetectorID        string          `json:"detector_id"`
	DetectorVersion   string          `json:"detector_version"`
	ModelVersion      string          `json:"model_version"`
	SignalTime        time.Time       `json:"timestamp"`
	ObservationTime   time.Time       `json:"observation_time"`
	EffectiveTime     time.Time       `json:"effective_time"`
	ProcessingTime    time.Time       `json:"processing_time"`
	WindowStart       time.Time       `json:"window_start"`
	WindowEnd         time.Time       `json:"window_end"`
	Confidence        float64         `json:"confidence"`
	Severity          string          `json:"severity"`
	Entities          json.RawMessage `json:"entities"`
	SupportingMetrics json.RawMessage `json:"supporting_metrics"`
	GraphTargets      json.RawMessage `json:"graph_targets"`
	SemanticEvidence  json.RawMessage `json:"semantic_evidence"`
	Evidence          json.RawMessage `json:"evidence"`
	Recommendation    json.RawMessage `json:"recommendation"`
	CorrelationID     string          `json:"correlation_id"`
	TraceID           string          `json:"trace_id,omitempty"`
	CausationID       string          `json:"causation_id,omitempty"`
	ReplayJobID       string          `json:"replay_job_id,omitempty"`
	BrokerTopic       string          `json:"broker_topic"`
	BrokerPartition   int32           `json:"broker_partition"`
	BrokerOffset      int64           `json:"broker_offset"`
	Event             json.RawMessage `json:"event"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type alertDTO struct {
	AlertID         string          `json:"alert_id"`
	TenantID        string          `json:"tenant_id"`
	SourceID        string          `json:"source_id"`
	SourceDomain    string          `json:"source_domain"`
	SourceAdapter   string          `json:"source_adapter"`
	Dataset         string          `json:"dataset"`
	SignalID        string          `json:"signal_id"`
	DetectorID      string          `json:"detector_id"`
	AlertType       string          `json:"alert_type"`
	Severity        string          `json:"severity"`
	Status          string          `json:"status"`
	Title           string          `json:"title"`
	Summary         string          `json:"summary"`
	Confidence      float64         `json:"confidence"`
	EventIDs        []string        `json:"event_ids"`
	Entities        json.RawMessage `json:"entities"`
	Evidence        json.RawMessage `json:"evidence"`
	Recommendation  json.RawMessage `json:"recommendation"`
	CorrelationID   string          `json:"correlation_id"`
	FirstObservedAt time.Time       `json:"first_observed_at"`
	LastObservedAt  time.Time       `json:"last_observed_at"`
	AcknowledgedAt  *time.Time      `json:"acknowledged_at,omitempty"`
	AcknowledgedBy  string          `json:"acknowledged_by,omitempty"`
	ResolvedAt      *time.Time      `json:"resolved_at,omitempty"`
	ResolvedBy      string          `json:"resolved_by,omitempty"`
	Metadata        json.RawMessage `json:"metadata"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type insightDTO struct {
	InsightID         string          `json:"insight_id"`
	TenantID          string          `json:"tenant_id"`
	SourceID          string          `json:"source_id"`
	SourceDomain      string          `json:"source_domain"`
	SourceAdapter     string          `json:"source_adapter"`
	Dataset           string          `json:"dataset"`
	SignalID          string          `json:"signal_id"`
	DetectorID        string          `json:"detector_id"`
	InsightType       string          `json:"insight_type"`
	Status            string          `json:"status"`
	Title             string          `json:"title"`
	Summary           string          `json:"summary"`
	Confidence        float64         `json:"confidence"`
	Severity          string          `json:"severity"`
	EventIDs          []string        `json:"event_ids"`
	Entities          json.RawMessage `json:"entities"`
	SupportingMetrics json.RawMessage `json:"supporting_metrics"`
	SemanticEvidence  json.RawMessage `json:"semantic_evidence"`
	Recommendation    json.RawMessage `json:"recommendation"`
	CorrelationID     string          `json:"correlation_id"`
	ObservedAt        time.Time       `json:"observed_at"`
	ReviewedAt        *time.Time      `json:"reviewed_at,omitempty"`
	ReviewedBy        string          `json:"reviewed_by,omitempty"`
	Metadata          json.RawMessage `json:"metadata"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type catalogSourceDTO struct {
	TenantID       string          `json:"tenant_id"`
	SourceID       string          `json:"source_id"`
	SourceDomain   string          `json:"source_domain"`
	SourceAdapter  string          `json:"source_adapter"`
	DisplayName    string          `json:"display_name"`
	Description    string          `json:"description"`
	Status         string          `json:"status"`
	IngestionModes []string        `json:"ingestion_modes"`
	Datasets       []string        `json:"datasets"`
	Metadata       json.RawMessage `json:"metadata"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type catalogPipelineDTO struct {
	TenantID      string          `json:"tenant_id"`
	PipelineID    string          `json:"pipeline_id"`
	SourceID      string          `json:"source_id"`
	SourceDomain  string          `json:"source_domain"`
	PipelineName  string          `json:"pipeline_name"`
	Description   string          `json:"description"`
	Status        string          `json:"status"`
	Stages        []string        `json:"stages"`
	InputDatasets []string        `json:"input_datasets"`
	OutputTopics  []string        `json:"output_topics"`
	Metadata      json.RawMessage `json:"metadata"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type catalogRuleDTO struct {
	TenantID     string          `json:"tenant_id"`
	RuleID       string          `json:"rule_id"`
	RuleName     string          `json:"rule_name"`
	Description  string          `json:"description"`
	RuleType     string          `json:"rule_type"`
	Severity     string          `json:"severity"`
	Status       string          `json:"status"`
	Version      int             `json:"version"`
	SourceID     string          `json:"source_id,omitempty"`
	PipelineID   string          `json:"pipeline_id,omitempty"`
	DatasetScope []string        `json:"dataset_scope"`
	EntityScope  []string        `json:"entity_scope"`
	Expression   json.RawMessage `json:"expression"`
	Actions      []string        `json:"actions"`
	Metadata     json.RawMessage `json:"metadata"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
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

func normalizedEventResponses(records []storage.NormalizedEventLedgerRecord) []normalizedEventDTO {
	items := make([]normalizedEventDTO, 0, len(records))
	for _, record := range records {
		items = append(items, normalizedEventResponse(record))
	}
	return items
}

func normalizedEventResponse(record storage.NormalizedEventLedgerRecord) normalizedEventDTO {
	return normalizedEventDTO{EventID: record.EventID, TenantID: record.TenantID, SourceID: record.SourceID,
		SourceAdapter: record.SourceAdapter, Dataset: record.Dataset, IdempotencyKey: record.IdempotencyKey,
		SchemaID: record.SchemaID, SchemaVersion: record.SchemaVersion, ObservationTime: record.ObservationTime,
		ProcessingTime: record.ProcessingTime, Confidence: record.Confidence, RawTopic: record.RawTopic,
		RawPartition: record.RawPartition, RawOffset: record.RawOffset, NormalizedTopic: record.NormalizedTopic,
		NormalizedPartition: record.NormalizedPartition, NormalizedOffset: record.NormalizedOffset,
		NormalizedPayload: jsonRawOrEmptyObject(record.NormalizedPayload), Entities: jsonRawOrEmptyArray(record.EntitiesJSON),
		Evidence: jsonRawOrEmptyArray(record.EvidenceJSON), Metadata: jsonRawOrEmptyObject(record.MetadataJSON),
		Event: jsonRawOrEmptyObject(record.EventJSON), CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}

func signalResponses(records []storage.SignalLedgerRecord) []signalDTO {
	items := make([]signalDTO, 0, len(records))
	for _, record := range records {
		items = append(items, signalResponse(record))
	}
	return items
}

func signalResponse(record storage.SignalLedgerRecord) signalDTO {
	recommendation := json.RawMessage(record.RecommendationJSON)
	if len(recommendation) == 0 {
		recommendation = json.RawMessage("null")
	}
	return signalDTO{SignalID: record.SignalID, TenantID: record.TenantID, SourceID: record.SourceID,
		SourceDomain: record.SourceDomain, SourceAdapter: record.SourceAdapter, IngestionMode: record.IngestionMode,
		Dataset: record.Dataset, EventIDs: record.EventIDs, ArtifactIDs: record.ArtifactIDs, SignalType: record.SignalType,
		DetectorID: record.DetectorID, DetectorVersion: record.DetectorVersion, ModelVersion: record.ModelVersion,
		SignalTime: record.SignalTime, ObservationTime: record.ObservationTime, EffectiveTime: record.EffectiveTime,
		ProcessingTime: record.ProcessingTime, WindowStart: record.WindowStart, WindowEnd: record.WindowEnd,
		Confidence: record.Confidence, Severity: record.Severity, Entities: jsonRawOrEmptyArray(record.EntitiesJSON),
		SupportingMetrics: jsonRawOrEmptyObject(record.SupportingMetrics), GraphTargets: jsonRawOrEmptyArray(record.GraphTargetsJSON),
		SemanticEvidence: jsonRawOrEmptyArray(record.SemanticEvidenceJSON), Evidence: jsonRawOrEmptyArray(record.EvidenceJSON),
		Recommendation: recommendation, CorrelationID: record.CorrelationID, TraceID: record.TraceID,
		CausationID: record.CausationID, ReplayJobID: record.ReplayJobID, BrokerTopic: record.BrokerTopic,
		BrokerPartition: record.BrokerPartition, BrokerOffset: record.BrokerOffset,
		Event: jsonRawOrEmptyObject(record.EventJSON), CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}

func alertResponses(records []storage.AlertLedgerRecord) []alertDTO {
	items := make([]alertDTO, 0, len(records))
	for _, record := range records {
		items = append(items, alertResponse(record))
	}
	return items
}

func alertResponse(record storage.AlertLedgerRecord) alertDTO {
	recommendation := json.RawMessage(record.RecommendationJSON)
	if len(recommendation) == 0 {
		recommendation = json.RawMessage("null")
	}
	return alertDTO{AlertID: record.AlertID, TenantID: record.TenantID, SourceID: record.SourceID,
		SourceDomain: record.SourceDomain, SourceAdapter: record.SourceAdapter, Dataset: record.Dataset,
		SignalID: record.SignalID, DetectorID: record.DetectorID, AlertType: record.AlertType,
		Severity: record.Severity, Status: record.Status, Title: record.Title, Summary: record.Summary,
		Confidence: record.Confidence, EventIDs: record.EventIDs, Entities: jsonRawOrEmptyArray(record.EntitiesJSON),
		Evidence: jsonRawOrEmptyArray(record.EvidenceJSON), Recommendation: recommendation,
		CorrelationID: record.CorrelationID, FirstObservedAt: record.FirstObservedAt, LastObservedAt: record.LastObservedAt,
		AcknowledgedAt: record.AcknowledgedAt, AcknowledgedBy: record.AcknowledgedBy, ResolvedAt: record.ResolvedAt,
		ResolvedBy: record.ResolvedBy, Metadata: jsonRawOrEmptyObject(record.MetadataJSON), CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt}
}

func insightResponses(records []storage.InsightLedgerRecord) []insightDTO {
	items := make([]insightDTO, 0, len(records))
	for _, record := range records {
		items = append(items, insightResponse(record))
	}
	return items
}

func insightResponse(record storage.InsightLedgerRecord) insightDTO {
	recommendation := json.RawMessage(record.RecommendationJSON)
	if len(recommendation) == 0 {
		recommendation = json.RawMessage("null")
	}
	return insightDTO{InsightID: record.InsightID, TenantID: record.TenantID, SourceID: record.SourceID,
		SourceDomain: record.SourceDomain, SourceAdapter: record.SourceAdapter, Dataset: record.Dataset,
		SignalID: record.SignalID, DetectorID: record.DetectorID, InsightType: record.InsightType,
		Status: record.Status, Title: record.Title, Summary: record.Summary, Confidence: record.Confidence,
		Severity: record.Severity, EventIDs: record.EventIDs, Entities: jsonRawOrEmptyArray(record.EntitiesJSON),
		SupportingMetrics: jsonRawOrEmptyObject(record.SupportingMetrics), SemanticEvidence: jsonRawOrEmptyArray(record.SemanticEvidenceJSON),
		Recommendation: recommendation, CorrelationID: record.CorrelationID, ObservedAt: record.ObservedAt,
		ReviewedAt: record.ReviewedAt, ReviewedBy: record.ReviewedBy, Metadata: jsonRawOrEmptyObject(record.MetadataJSON),
		CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}

func catalogSourceResponses(records []storage.CatalogSourceRecord) []catalogSourceDTO {
	items := make([]catalogSourceDTO, 0, len(records))
	for _, record := range records {
		items = append(items, catalogSourceDTO{
			TenantID:       record.TenantID,
			SourceID:       record.SourceID,
			SourceDomain:   record.SourceDomain,
			SourceAdapter:  record.SourceAdapter,
			DisplayName:    record.DisplayName,
			Description:    record.Description,
			Status:         record.Status,
			IngestionModes: record.IngestionModes,
			Datasets:       record.Datasets,
			Metadata:       jsonRawOrEmptyObject(record.MetadataJSON),
			CreatedAt:      record.CreatedAt,
			UpdatedAt:      record.UpdatedAt,
		})
	}
	return items
}

func catalogPipelineResponses(records []storage.CatalogPipelineRecord) []catalogPipelineDTO {
	items := make([]catalogPipelineDTO, 0, len(records))
	for _, record := range records {
		items = append(items, catalogPipelineDTO{
			TenantID:      record.TenantID,
			PipelineID:    record.PipelineID,
			SourceID:      record.SourceID,
			SourceDomain:  record.SourceDomain,
			PipelineName:  record.PipelineName,
			Description:   record.Description,
			Status:        record.Status,
			Stages:        record.Stages,
			InputDatasets: record.InputDatasets,
			OutputTopics:  record.OutputTopics,
			Metadata:      jsonRawOrEmptyObject(record.MetadataJSON),
			CreatedAt:     record.CreatedAt,
			UpdatedAt:     record.UpdatedAt,
		})
	}
	return items
}

func catalogRuleResponses(records []storage.CatalogRuleRecord) []catalogRuleDTO {
	items := make([]catalogRuleDTO, 0, len(records))
	for _, record := range records {
		items = append(items, catalogRuleDTO{
			TenantID:     record.TenantID,
			RuleID:       record.RuleID,
			RuleName:     record.RuleName,
			Description:  record.Description,
			RuleType:     record.RuleType,
			Severity:     record.Severity,
			Status:       record.Status,
			Version:      record.Version,
			SourceID:     record.SourceID,
			PipelineID:   record.PipelineID,
			DatasetScope: record.DatasetScope,
			EntityScope:  record.EntityScope,
			Expression:   jsonRawOrEmptyObject(record.ExpressionJSON),
			Actions:      record.Actions,
			Metadata:     jsonRawOrEmptyObject(record.MetadataJSON),
			CreatedAt:    record.CreatedAt,
			UpdatedAt:    record.UpdatedAt,
		})
	}
	return items
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
