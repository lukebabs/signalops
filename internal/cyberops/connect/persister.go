package connect

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
	"github.com/lukebabs/signalops/pkg/broker"
)

type Repository interface {
	PersistCyberOpsConnectRaw(context.Context, storage.CyberOpsConnectRawRecord, storage.CyberOpsOutboxRecord, []byte) (storage.CyberOpsPersistResult, error)
}

type ProcessResult struct {
	Ignored          bool
	Duplicate        bool
	IntegrityFailure bool
}

type Processor struct {
	Repository    Repository
	AcceptedTopic string
	Now           func() time.Time
}

func (p Processor) Process(ctx context.Context, message broker.ConsumedMessage) (ProcessResult, error) {
	if !IsCandidate(message.Value, message.Headers) {
		return ProcessResult{Ignored: true}, nil
	}
	if p.Repository == nil || p.AcceptedTopic == "" {
		return ProcessResult{}, fmt.Errorf("cyberops connect processor is not configured")
	}
	event, err := Validate(message.Value)
	if err != nil {
		return ProcessResult{}, err
	}
	now := time.Now().UTC()
	if p.Now != nil {
		now = p.Now().UTC()
	}
	headers, err := json.Marshal(map[string]string{"content_type": "application/json", "signalops_schema_id": SchemaID, "connect_ingress_event_id": event.IngressEventID, "connect_delivery_id": stringValue(event.Metadata["delivery_id"])})
	if err != nil {
		return ProcessResult{}, err
	}
	raw := storage.CyberOpsConnectRawRecord{TenantID: event.TenantID, ConnectIngressEventID: event.IngressEventID, EventID: event.IngressEventID, SourceID: event.SourceID, EventType: EventType, OccurredAt: event.OccurredAt, IngestedAt: event.IngestedAt, Hostname: event.Hostname, Application: event.Application, Severity: event.Severity, Facility: event.Facility, Message: event.Message, RawEventJSON: event.Raw, ConnectMetadataJSON: mustJSON(event.Metadata), PayloadHash: event.PayloadHash}
	outbox := storage.CyberOpsOutboxRecord{OutboxID: outboxID(event), TenantID: event.TenantID, Topic: p.AcceptedTopic, MessageKey: event.IngressEventID, MessageValue: event.Raw, HeadersJSON: headers, CorrelationID: event.CorrelationID, CausationID: event.CausationID, TraceID: event.TraceID, CreatedAt: now}
	result, err := p.Repository.PersistCyberOpsConnectRaw(ctx, raw, outbox, event.Lineage)
	if err != nil {
		return ProcessResult{}, err
	}
	return ProcessResult{Duplicate: result.Duplicate, IntegrityFailure: result.IntegrityFailure}, nil
}

func outboxID(event Event) string {
	sum := sha256.Sum256([]byte(event.TenantID + "|" + event.IngressEventID + "|" + event.PayloadHash))
	return "cyberout_" + hex.EncodeToString(sum[:16])
}
func mustJSON(value any) []byte { data, _ := json.Marshal(value); return data }
