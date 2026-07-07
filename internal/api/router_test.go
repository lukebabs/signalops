package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lukebabs/signalops/pkg/broker"
)

type fakePublisher struct {
	msg broker.Message
	err error
}

func (p *fakePublisher) Publish(ctx context.Context, msg broker.Message) (broker.PublishResult, error) {
	p.msg = msg
	if p.err != nil {
		return broker.PublishResult{}, p.err
	}
	return broker.PublishResult{Topic: msg.Topic, Partition: 2, Offset: 42}, nil
}

func (p *fakePublisher) Close(ctx context.Context) error {
	return nil
}

func TestPostRawEventPublishesMessage(t *testing.T) {
	publisher := &fakePublisher{}
	router := NewRouter(RouterConfig{
		ServiceName: "test-gateway",
		Publisher:   publisher,
		RawTopic:    "signalops.test.raw.v1",
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/events/raw", bytes.NewBufferString(`{
		"event_id":"evt-123",
		"idempotency_key":"idem-123",
		"correlation_id":"corr-payload",
		"payload":{"symbol":"SPY"}
	}`))
	req.Header.Set("X-Correlation-ID", "corr-header")
	req.Header.Set("X-Causation-ID", "cause-header")
	req.Header.Set("X-Trace-ID", "trace-header")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if publisher.msg.Topic != "signalops.test.raw.v1" {
		t.Fatalf("topic = %q", publisher.msg.Topic)
	}
	if publisher.msg.Key != "idem-123" {
		t.Fatalf("key = %q", publisher.msg.Key)
	}
	if publisher.msg.CorrelationID != "corr-header" {
		t.Fatalf("correlation id = %q", publisher.msg.CorrelationID)
	}
	if publisher.msg.CausationID != "cause-header" {
		t.Fatalf("causation id = %q", publisher.msg.CausationID)
	}
	if publisher.msg.TraceID != "trace-header" {
		t.Fatalf("trace id = %q", publisher.msg.TraceID)
	}
	if publisher.msg.Headers["signalops_event_id"] != "evt-123" {
		t.Fatalf("event header = %q", publisher.msg.Headers["signalops_event_id"])
	}

	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("response JSON error = %v", err)
	}
	if response["status"] != "accepted" {
		t.Fatalf("response status = %v", response["status"])
	}
	if response["topic"] != "signalops.test.raw.v1" {
		t.Fatalf("response topic = %v", response["topic"])
	}
	if response["offset"].(float64) != 42 {
		t.Fatalf("response offset = %v", response["offset"])
	}
}

func TestPostRawEventRejectsInvalidJSON(t *testing.T) {
	router := NewRouter(RouterConfig{
		Publisher: &fakePublisher{},
		RawTopic:  "signalops.test.raw.v1",
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/events/raw", bytes.NewBufferString(`[]`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestPostRawEventRequiresPublisher(t *testing.T) {
	router := NewRouter(RouterConfig{RawTopic: "signalops.test.raw.v1"})

	req := httptest.NewRequest(http.MethodPost, "/v1/events/raw", bytes.NewBufferString(`{"event_id":"evt-123"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestPostRawEventHandlesPublishFailure(t *testing.T) {
	router := NewRouter(RouterConfig{
		Publisher: &fakePublisher{err: errors.New("publish failed")},
		RawTopic:  "signalops.test.raw.v1",
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/events/raw", bytes.NewBufferString(`{"event_id":"evt-123"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}
