package broker

import (
	"context"
	"time"
)

// Message is the durable broker envelope exchanged between SignalOps services.
type Message struct {
	Topic         string
	Key           string
	Value         []byte
	Headers       map[string]string
	CorrelationID string
	CausationID   string
	TraceID       string
	PublishedAt   time.Time
}

// PublishResult contains broker acknowledgement details.
type PublishResult struct {
	Topic     string
	Partition int32
	Offset    int64
}

// ConsumedMessage is a broker message delivered to a consumer.
type ConsumedMessage struct {
	Message
	Partition int32
	Offset    int64
}

// Publisher publishes durable messages to the broker.
type Publisher interface {
	Publish(ctx context.Context, msg Message) (PublishResult, error)
	Close(ctx context.Context) error
}

// Consumer consumes durable messages from the broker.
type Consumer interface {
	Consume(ctx context.Context) (ConsumedMessage, error)
	Commit(ctx context.Context, msg ConsumedMessage) error
	Close(ctx context.Context) error
}

// Client groups broker publishing and consumer construction.
type Client interface {
	Publisher
	NewConsumer(groupID string, topics []string) (Consumer, error)
}
