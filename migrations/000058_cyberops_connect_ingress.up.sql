CREATE TABLE IF NOT EXISTS cyberops_connect_raw_events (
  tenant_id text NOT NULL,
  connect_ingress_event_id text NOT NULL,
  event_id text NOT NULL,
  source_id text NOT NULL,
  event_type text NOT NULL,
  occurred_at timestamptz NOT NULL,
  ingested_at timestamptz NOT NULL,
  hostname text,
  application text,
  severity integer,
  facility integer,
  message text NOT NULL,
  raw_event jsonb NOT NULL,
  connect_metadata jsonb NOT NULL,
  payload_hash text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, connect_ingress_event_id)
);
CREATE INDEX IF NOT EXISTS idx_cyberops_connect_raw_time ON cyberops_connect_raw_events (tenant_id, occurred_at DESC, connect_ingress_event_id);
CREATE INDEX IF NOT EXISTS idx_cyberops_connect_raw_filters ON cyberops_connect_raw_events (tenant_id, hostname, application, severity, facility, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_cyberops_connect_raw_message ON cyberops_connect_raw_events USING gin (to_tsvector('simple', message));

CREATE TABLE IF NOT EXISTS cyberops_connect_integrity_failures (
  failure_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  connect_ingress_event_id text NOT NULL,
  existing_event_id text NOT NULL,
  existing_payload_hash text NOT NULL,
  incoming_payload_hash text NOT NULL,
  existing_lineage jsonb NOT NULL,
  incoming_lineage jsonb NOT NULL,
  received_event jsonb NOT NULL,
  first_seen_at timestamptz NOT NULL,
  last_seen_at timestamptz NOT NULL,
  occurrence_count integer NOT NULL DEFAULT 1,
  resolution_status text NOT NULL DEFAULT 'open'
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_cyberops_connect_integrity_identity ON cyberops_connect_integrity_failures (tenant_id, connect_ingress_event_id, incoming_payload_hash);

CREATE TABLE IF NOT EXISTS cyberops_connect_outbox (
  outbox_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  topic text NOT NULL,
  message_key text NOT NULL,
  message_value bytea NOT NULL,
  headers jsonb NOT NULL DEFAULT '{}'::jsonb,
  correlation_id text NOT NULL,
  causation_id text NOT NULL,
  trace_id text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  published_at timestamptz,
  attempts integer NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_cyberops_connect_outbox_pending ON cyberops_connect_outbox (created_at) WHERE published_at IS NULL;
