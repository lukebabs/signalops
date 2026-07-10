#!/bin/sh
set -eu

SOURCE_DB_URL="${SIGNALOPS_DATABASE_URL:-postgres://signalops:signalops@postgres:5432/signalops?sslmode=disable}"
TARGET_DB_URL="${SIGNALOPS_TEMPORAL_DATABASE_URL:-postgres://signalops:signalops@timescaledb:5432/signalops_temporal?sslmode=disable}"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

copy_table() {
  name="$1"
  columns="$2"
  conflict="$3"
  updates="$4"
  order_by="$5"
  file="$TMP_DIR/${name}.csv"

  echo "export ${name} from relational postgres"
  psql "$SOURCE_DB_URL" -v ON_ERROR_STOP=1 -c "\\copy (SELECT ${columns} FROM ${name} ORDER BY ${order_by}) TO '${file}' WITH (FORMAT csv)"

  echo "import ${name} into timescaledb"
  psql "$TARGET_DB_URL" -v ON_ERROR_STOP=1 <<SQL
CREATE TEMP TABLE backfill_${name} (LIKE ${name} INCLUDING DEFAULTS);
\\copy backfill_${name} (${columns}) FROM '${file}' WITH (FORMAT csv)
INSERT INTO ${name} (${columns})
SELECT ${columns} FROM backfill_${name}
ON CONFLICT (${conflict}) DO UPDATE SET
${updates};
SQL
}

copy_table \
  raw_event_ledger \
  "event_id, tenant_id, source_id, source_adapter, dataset, idempotency_key, observation_time, processing_time, broker_topic, broker_partition, broker_offset, payload, entity_hints, created_at" \
  "event_id, observation_time" \
  "  tenant_id = EXCLUDED.tenant_id,
  source_id = EXCLUDED.source_id,
  source_adapter = EXCLUDED.source_adapter,
  dataset = EXCLUDED.dataset,
  idempotency_key = EXCLUDED.idempotency_key,
  processing_time = EXCLUDED.processing_time,
  broker_topic = EXCLUDED.broker_topic,
  broker_partition = EXCLUDED.broker_partition,
  broker_offset = EXCLUDED.broker_offset,
  payload = EXCLUDED.payload,
  entity_hints = EXCLUDED.entity_hints,
  created_at = EXCLUDED.created_at" \
  "observation_time, event_id"

copy_table \
  normalized_event_ledger \
  "event_id, tenant_id, source_id, source_adapter, dataset, idempotency_key, schema_id, schema_version, observation_time, processing_time, confidence, raw_topic, raw_partition, raw_offset, normalized_topic, normalized_partition, normalized_offset, normalized_payload, entities, evidence, metadata, event, created_at, updated_at" \
  "event_id, observation_time" \
  "  tenant_id = EXCLUDED.tenant_id,
  source_id = EXCLUDED.source_id,
  source_adapter = EXCLUDED.source_adapter,
  dataset = EXCLUDED.dataset,
  idempotency_key = EXCLUDED.idempotency_key,
  schema_id = EXCLUDED.schema_id,
  schema_version = EXCLUDED.schema_version,
  processing_time = EXCLUDED.processing_time,
  confidence = EXCLUDED.confidence,
  raw_topic = EXCLUDED.raw_topic,
  raw_partition = EXCLUDED.raw_partition,
  raw_offset = EXCLUDED.raw_offset,
  normalized_topic = EXCLUDED.normalized_topic,
  normalized_partition = EXCLUDED.normalized_partition,
  normalized_offset = EXCLUDED.normalized_offset,
  normalized_payload = EXCLUDED.normalized_payload,
  entities = EXCLUDED.entities,
  evidence = EXCLUDED.evidence,
  metadata = EXCLUDED.metadata,
  event = EXCLUDED.event,
  created_at = EXCLUDED.created_at,
  updated_at = EXCLUDED.updated_at" \
  "observation_time, event_id"

copy_table \
  signal_ledger \
  "signal_id, tenant_id, source_id, source_domain, source_adapter, ingestion_mode, dataset, event_ids, artifact_ids, signal_type, detector_id, detector_version, model_version, signal_time, observation_time, effective_time, processing_time, window_start, window_end, confidence, severity, entities, supporting_metrics, graph_targets, semantic_evidence, evidence, recommendation, correlation_id, trace_id, causation_id, replay_job_id, broker_topic, broker_partition, broker_offset, event, created_at, updated_at" \
  "signal_id, signal_time" \
  "  tenant_id = EXCLUDED.tenant_id,
  source_id = EXCLUDED.source_id,
  source_domain = EXCLUDED.source_domain,
  source_adapter = EXCLUDED.source_adapter,
  ingestion_mode = EXCLUDED.ingestion_mode,
  dataset = EXCLUDED.dataset,
  event_ids = EXCLUDED.event_ids,
  artifact_ids = EXCLUDED.artifact_ids,
  signal_type = EXCLUDED.signal_type,
  detector_id = EXCLUDED.detector_id,
  detector_version = EXCLUDED.detector_version,
  model_version = EXCLUDED.model_version,
  observation_time = EXCLUDED.observation_time,
  effective_time = EXCLUDED.effective_time,
  processing_time = EXCLUDED.processing_time,
  window_start = EXCLUDED.window_start,
  window_end = EXCLUDED.window_end,
  confidence = EXCLUDED.confidence,
  severity = EXCLUDED.severity,
  entities = EXCLUDED.entities,
  supporting_metrics = EXCLUDED.supporting_metrics,
  graph_targets = EXCLUDED.graph_targets,
  semantic_evidence = EXCLUDED.semantic_evidence,
  evidence = EXCLUDED.evidence,
  recommendation = EXCLUDED.recommendation,
  correlation_id = EXCLUDED.correlation_id,
  trace_id = EXCLUDED.trace_id,
  causation_id = EXCLUDED.causation_id,
  replay_job_id = EXCLUDED.replay_job_id,
  broker_topic = EXCLUDED.broker_topic,
  broker_partition = EXCLUDED.broker_partition,
  broker_offset = EXCLUDED.broker_offset,
  event = EXCLUDED.event,
  created_at = EXCLUDED.created_at,
  updated_at = EXCLUDED.updated_at" \
  "signal_time, signal_id"

echo "temporal backfill complete"
