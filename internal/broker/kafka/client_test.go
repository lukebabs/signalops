package kafka

import (
	"context"
	"errors"
	"testing"

	"github.com/lukebabs/signalops/pkg/broker"
	"github.com/twmb/franz-go/pkg/kgo"
)

func TestNewClientRequiresBrokers(t *testing.T) {
	client, err := NewClient(Config{})
	if !errors.Is(err, ErrMissingBrokers) {
		t.Fatalf("NewClient() error = %v, want %v", err, ErrMissingBrokers)
	}
	if client != nil {
		t.Fatal("NewClient() returned client for missing brokers")
	}
}

func TestPublishRequiresTopic(t *testing.T) {
	client, err := NewClient(Config{Brokers: []string{"localhost:19092"}})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer func() {
		if err := client.Close(context.Background()); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	_, err = client.Publish(context.Background(), broker.Message{})
	if !errors.Is(err, ErrMissingTopic) {
		t.Fatalf("Publish() error = %v, want %v", err, ErrMissingTopic)
	}
}

func TestHeaderRoundTripMapping(t *testing.T) {
	msg := broker.Message{
		Headers: map[string]string{
			"source": "unit-test",
		},
		CorrelationID: "corr-1",
		CausationID:   "cause-1",
		TraceID:       "trace-1",
	}

	recordHeaders := toRecordHeaders(msg)
	consumed := fromRecord(&kgo.Record{Headers: recordHeaders})

	if consumed.Headers["source"] != "unit-test" {
		t.Fatalf("source header = %q", consumed.Headers["source"])
	}
	if consumed.CorrelationID != msg.CorrelationID {
		t.Fatalf("CorrelationID = %q", consumed.CorrelationID)
	}
	if consumed.CausationID != msg.CausationID {
		t.Fatalf("CausationID = %q", consumed.CausationID)
	}
	if consumed.TraceID != msg.TraceID {
		t.Fatalf("TraceID = %q", consumed.TraceID)
	}
}
