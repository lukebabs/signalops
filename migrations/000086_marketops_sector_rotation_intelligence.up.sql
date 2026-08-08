-- SRI Foundation MVP: research-only, price-led segment context.
CREATE TABLE sri_segments (
 tenant_id text NOT NULL, segment_id text NOT NULL, segment_key text NOT NULL, name text NOT NULL,
 segment_type text NOT NULL CHECK (segment_type IN ('sector','industry','benchmark')),
 parent_segment_key text NOT NULL DEFAULT '', active boolean NOT NULL DEFAULT true,
 registry_version text NOT NULL, metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (tenant_id, segment_id), UNIQUE (tenant_id, segment_key, registry_version)
);
CREATE TABLE sri_etf_registry (
 tenant_id text NOT NULL, etf_symbol text NOT NULL, segment_id text NOT NULL, role text NOT NULL CHECK (role IN ('primary','secondary','context')),
 benchmark_priority integer NOT NULL DEFAULT 1, active boolean NOT NULL DEFAULT true,
 registry_version text NOT NULL, config jsonb NOT NULL DEFAULT '{}'::jsonb,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (tenant_id, etf_symbol, segment_id, registry_version),
 FOREIGN KEY (tenant_id, segment_id) REFERENCES sri_segments (tenant_id, segment_id)
);
CREATE TABLE sri_segment_snapshots (
 snapshot_id text PRIMARY KEY, tenant_id text NOT NULL, segment_id text NOT NULL, session_date date NOT NULL,
 as_of_time timestamptz NOT NULL, state text NOT NULL, composite_score double precision,
 relative_strength_score double precision, momentum_score double precision, momentum_acceleration double precision,
 rank integer, rank_change_5d integer, evidence_quality double precision CHECK (evidence_quality >= 0 AND evidence_quality <= 1),
 quality_state text NOT NULL, quality_flags jsonb NOT NULL DEFAULT '[]'::jsonb, components jsonb NOT NULL DEFAULT '{}'::jsonb,
 input_provenance jsonb NOT NULL DEFAULT '{}'::jsonb, algorithm_version text NOT NULL, configuration_version text NOT NULL,
 calculation_run_id text NOT NULL, deterministic_key text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE (tenant_id, deterministic_key), UNIQUE (tenant_id, segment_id, session_date, algorithm_version)
);
CREATE INDEX idx_sri_snapshots_rank ON sri_segment_snapshots (tenant_id, session_date DESC, rank);
CREATE INDEX idx_sri_snapshots_segment ON sri_segment_snapshots (tenant_id, segment_id, session_date DESC);
CREATE TABLE sri_segment_state_events (
 event_id text PRIMARY KEY, tenant_id text NOT NULL, segment_id text NOT NULL, session_date date NOT NULL,
 previous_state text NOT NULL DEFAULT '', new_state text NOT NULL, reason_codes jsonb NOT NULL DEFAULT '[]'::jsonb,
 snapshot_id text NOT NULL REFERENCES sri_segment_snapshots(snapshot_id), algorithm_version text NOT NULL,
 deterministic_key text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), UNIQUE (tenant_id, deterministic_key)
);
