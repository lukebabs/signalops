-- First-class review ledger for MarketOps DSM graph target candidates derived from persisted artifacts.

CREATE TABLE IF NOT EXISTS marketops_dsm_graph_proposals (
  proposal_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  app_id text NOT NULL DEFAULT 'marketops',
  domain text NOT NULL DEFAULT 'market_data',
  use_case text NOT NULL DEFAULT 'daily_market_surveillance',
  source_id text NOT NULL,
  source_adapter text NOT NULL,
  dataset text NOT NULL,
  artifact_id text NOT NULL REFERENCES marketops_dsm_artifacts(artifact_id) ON DELETE CASCADE,
  signal_id text NOT NULL REFERENCES signal_ledger(signal_id) ON DELETE CASCADE,
  signal_type text NOT NULL,
  detector_id text NOT NULL,
  severity text NOT NULL CHECK (severity IN ('info', 'low', 'medium', 'high', 'critical')),
  confidence double precision NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
  event_ids text[] NOT NULL DEFAULT '{}',
  subject_symbol text NOT NULL DEFAULT '',
  candidate_type text NOT NULL CHECK (candidate_type IN ('node_candidate', 'relationship_candidate')),
  node_id text NOT NULL DEFAULT '',
  from_node text NOT NULL DEFAULT '',
  relationship text NOT NULL DEFAULT '',
  to_node text NOT NULL DEFAULT '',
  labels text[] NOT NULL DEFAULT '{}',
  properties jsonb NOT NULL DEFAULT '{}'::jsonb,
  raw_candidate jsonb NOT NULL DEFAULT '{}'::jsonb,
  status text NOT NULL DEFAULT 'proposed' CHECK (status IN ('proposed', 'accepted', 'rejected', 'superseded')),
  reviewed_by text NOT NULL DEFAULT '',
  decision_note text NOT NULL DEFAULT '',
  decided_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_marketops_dsm_graph_proposals_tenant_time
  ON marketops_dsm_graph_proposals (tenant_id, app_id, domain, use_case, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_marketops_dsm_graph_proposals_artifact
  ON marketops_dsm_graph_proposals (artifact_id);

CREATE INDEX IF NOT EXISTS idx_marketops_dsm_graph_proposals_signal
  ON marketops_dsm_graph_proposals (signal_id);

CREATE INDEX IF NOT EXISTS idx_marketops_dsm_graph_proposals_status_time
  ON marketops_dsm_graph_proposals (tenant_id, status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_marketops_dsm_graph_proposals_symbol_time
  ON marketops_dsm_graph_proposals (tenant_id, subject_symbol, updated_at DESC);
