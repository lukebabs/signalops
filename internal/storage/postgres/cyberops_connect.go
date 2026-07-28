package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) PersistCyberOpsConnectRaw(ctx context.Context, raw storage.CyberOpsConnectRawRecord, outbox storage.CyberOpsOutboxRecord, lineage []byte) (storage.CyberOpsPersistResult, error) {
	if strings.TrimSpace(raw.TenantID) == "" || strings.TrimSpace(raw.ConnectIngressEventID) == "" || len(raw.RawEventJSON) == 0 || len(raw.ConnectMetadataJSON) == 0 {
		return storage.CyberOpsPersistResult{}, fmt.Errorf("cyberops connect raw identity and evidence are required")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return storage.CyberOpsPersistResult{}, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT INTO cyberops_connect_raw_events
(tenant_id,connect_ingress_event_id,event_id,source_id,event_type,occurred_at,ingested_at,hostname,application,severity,facility,message,raw_event,connect_metadata,payload_hash)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
ON CONFLICT (tenant_id,connect_ingress_event_id) DO NOTHING`,
		raw.TenantID, raw.ConnectIngressEventID, raw.EventID, raw.SourceID, raw.EventType, raw.OccurredAt, raw.IngestedAt, nullable(raw.Hostname), nullable(raw.Application), raw.Severity, raw.Facility, raw.Message, raw.RawEventJSON, raw.ConnectMetadataJSON, raw.PayloadHash)
	if err != nil {
		return storage.CyberOpsPersistResult{}, fmt.Errorf("insert cyberops connect raw: %w", err)
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return storage.CyberOpsPersistResult{}, err
	}
	if inserted == 0 {
		var existingEvent, existingHash string
		var existingMetadata []byte
		if err := tx.QueryRowContext(ctx, `SELECT event_id,payload_hash,connect_metadata FROM cyberops_connect_raw_events WHERE tenant_id=$1 AND connect_ingress_event_id=$2`, raw.TenantID, raw.ConnectIngressEventID).Scan(&existingEvent, &existingHash, &existingMetadata); err != nil {
			return storage.CyberOpsPersistResult{}, err
		}
		if existingHash == raw.PayloadHash && sameImmutableLineage(existingMetadata, lineage) {
			if err := tx.Commit(); err != nil {
				return storage.CyberOpsPersistResult{}, err
			}
			return storage.CyberOpsPersistResult{Duplicate: true}, nil
		}
		failureID := integrityFailureID(raw.TenantID, raw.ConnectIngressEventID, raw.PayloadHash)
		_, err = tx.ExecContext(ctx, `INSERT INTO cyberops_connect_integrity_failures
(failure_id,tenant_id,connect_ingress_event_id,existing_event_id,existing_payload_hash,incoming_payload_hash,existing_lineage,incoming_lineage,received_event,first_seen_at,last_seen_at,occurrence_count,resolution_status)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$10,1,'open')
ON CONFLICT (tenant_id,connect_ingress_event_id,incoming_payload_hash) DO UPDATE
SET last_seen_at=EXCLUDED.last_seen_at, occurrence_count=cyberops_connect_integrity_failures.occurrence_count+1`,
			failureID, raw.TenantID, raw.ConnectIngressEventID, existingEvent, existingHash, raw.PayloadHash, existingMetadata, lineage, raw.RawEventJSON, time.Now().UTC())
		if err != nil {
			return storage.CyberOpsPersistResult{}, fmt.Errorf("persist integrity failure: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return storage.CyberOpsPersistResult{}, err
		}
		return storage.CyberOpsPersistResult{IntegrityFailure: true}, nil
	}
	headers := outbox.HeadersJSON
	if len(headers) == 0 {
		headers = []byte(`{}`)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO cyberops_connect_outbox
(outbox_id,tenant_id,topic,message_key,message_value,headers,correlation_id,causation_id,trace_id)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		outbox.OutboxID, raw.TenantID, outbox.Topic, outbox.MessageKey, outbox.MessageValue, headers, outbox.CorrelationID, outbox.CausationID, outbox.TraceID)
	if err != nil {
		return storage.CyberOpsPersistResult{}, fmt.Errorf("insert cyberops outbox: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return storage.CyberOpsPersistResult{}, err
	}
	return storage.CyberOpsPersistResult{}, nil
}

func (r *Repository) ListCyberOpsConnectRaw(ctx context.Context, filter storage.CyberOpsConnectRawFilter) ([]storage.CyberOpsConnectRawRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT tenant_id,connect_ingress_event_id,event_id,source_id,event_type,occurred_at,ingested_at,COALESCE(hostname,''),COALESCE(application,''),severity,facility,message,raw_event,connect_metadata,payload_hash,created_at
FROM cyberops_connect_raw_events WHERE tenant_id=$1 AND ($2::timestamptz IS NULL OR occurred_at >= $2) AND ($3::timestamptz IS NULL OR occurred_at <= $3)
AND ($4='' OR hostname=$4) AND ($5='' OR application=$5) AND ($6::int IS NULL OR severity=$6) AND ($7::int IS NULL OR facility=$7)
AND ($8='' OR source_id=$8) AND ($9='' OR connect_metadata->>'connector_id'=$9) AND ($10='' OR event_type=$10)
AND ($11='' OR to_tsvector('simple',message) @@ plainto_tsquery('simple',$11))
ORDER BY occurred_at DESC, connect_ingress_event_id DESC LIMIT $12`,
		filter.TenantID, nullableTime(filter.From), nullableTime(filter.To), filter.Hostname, filter.Application, filter.Severity, filter.Facility, filter.ProducerID, filter.ConnectorID, filter.EventType, filter.Search, limit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list cyberops connect raw: %w", err)
	}
	defer rows.Close()
	out := []storage.CyberOpsConnectRawRecord{}
	for rows.Next() {
		record, err := scanCyberOpsConnectRaw(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

func (r *Repository) GetCyberOpsConnectRaw(ctx context.Context, tenantID, ingressEventID string) (storage.CyberOpsConnectRawRecord, error) {
	record, err := scanCyberOpsConnectRaw(r.db.QueryRowContext(ctx, `SELECT tenant_id,connect_ingress_event_id,event_id,source_id,event_type,occurred_at,ingested_at,COALESCE(hostname,''),COALESCE(application,''),severity,facility,message,raw_event,connect_metadata,payload_hash,created_at FROM cyberops_connect_raw_events WHERE tenant_id=$1 AND connect_ingress_event_id=$2`, tenantID, ingressEventID))
	if err == sql.ErrNoRows {
		return storage.CyberOpsConnectRawRecord{}, storage.ErrNotFound
	}
	return record, err
}

func (r *Repository) ListPendingCyberOpsOutbox(ctx context.Context, max int) ([]storage.CyberOpsOutboxRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT outbox_id,tenant_id,topic,message_key,message_value,headers,correlation_id,causation_id,trace_id,created_at,published_at,attempts FROM cyberops_connect_outbox WHERE published_at IS NULL ORDER BY created_at LIMIT $1`, limit(max))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []storage.CyberOpsOutboxRecord{}
	for rows.Next() {
		var item storage.CyberOpsOutboxRecord
		if err := rows.Scan(&item.OutboxID, &item.TenantID, &item.Topic, &item.MessageKey, &item.MessageValue, &item.HeadersJSON, &item.CorrelationID, &item.CausationID, &item.TraceID, &item.CreatedAt, &item.PublishedAt, &item.Attempts); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *Repository) MarkCyberOpsOutboxPublished(ctx context.Context, id string, publishedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE cyberops_connect_outbox SET published_at=$2,attempts=attempts+1 WHERE outbox_id=$1 AND published_at IS NULL`, id, publishedAt)
	return err
}
func (r *Repository) MarkCyberOpsOutboxAttempt(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE cyberops_connect_outbox SET attempts=attempts+1 WHERE outbox_id=$1 AND published_at IS NULL`, id)
	return err
}

type cyberOpsRawScanner interface{ Scan(...any) error }

func scanCyberOpsConnectRaw(s cyberOpsRawScanner) (storage.CyberOpsConnectRawRecord, error) {
	var item storage.CyberOpsConnectRawRecord
	if err := s.Scan(&item.TenantID, &item.ConnectIngressEventID, &item.EventID, &item.SourceID, &item.EventType, &item.OccurredAt, &item.IngestedAt, &item.Hostname, &item.Application, &item.Severity, &item.Facility, &item.Message, &item.RawEventJSON, &item.ConnectMetadataJSON, &item.PayloadHash, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}
func sameImmutableLineage(existingMetadata, incomingLineage []byte) bool {
	existing, err := canonicalImmutableLineage(existingMetadata)
	if err != nil {
		return false
	}
	incoming, err := canonicalImmutableLineage(incomingLineage)
	if err != nil {
		return false
	}
	return bytes.Equal(existing, incoming)
}

func canonicalImmutableLineage(raw []byte) ([]byte, error) {
	var lineage map[string]any
	if err := json.Unmarshal(raw, &lineage); err != nil {
		return nil, err
	}
	delete(lineage, "processing_run_id")
	delete(lineage, "delivery_id")
	return json.Marshal(lineage)
}

func integrityFailureID(tenant, ingress, hash string) string {
	sum := sha256.Sum256([]byte(tenant + "|" + ingress + "|" + hash))
	return "cyberint_" + hex.EncodeToString(sum[:12])
}
func nullable(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
func nullableTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}
func limit(value int) int {
	if value <= 0 {
		return 50
	}
	if value > 200 {
		return 200
	}
	return value
}
func marshalHeaders(value map[string]string) []byte { data, _ := json.Marshal(value); return data }
