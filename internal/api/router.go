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
	"strings"
	"time"

	"github.com/lukebabs/signalops/pkg/broker"
)

const (
	defaultMaxRawEventBytes = 1 << 20
	defaultPublishTimeout   = 5 * time.Second
)

// RouterConfig contains process-local API wiring options.
type RouterConfig struct {
	ServiceName string
	Publisher   broker.Publisher
	RawTopic    string
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
