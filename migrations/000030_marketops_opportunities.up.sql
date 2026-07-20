-- G139: research-only MarketOps opportunity ledger.

CREATE TABLE IF NOT EXISTS marketops_opportunities (
  opportunity_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  app_id text NOT NULL DEFAULT 'marketops',
  asset_id text NOT NULL,
  symbol text NOT NULL,
  opened_session_date date NOT NULL,
  last_evaluated_date date NOT NULL,
  direction text NOT NULL CHECK (direction IN ('upside','downside','non_directional')),
  horizon text NOT NULL,
  lifecycle_status text NOT NULL CHECK (lifecycle_status IN ('emerging','active','strengthening','weakening','invalidated','resolved','expired')),
  opportunity_score double precision NOT NULL CHECK (opportunity_score >= 0 AND opportunity_score <= 1),
  confidence_score double precision NOT NULL CHECK (confidence_score >= 0 AND confidence_score <= 1),
  domain_diversity_score double precision NOT NULL CHECK (domain_diversity_score >= 0 AND domain_diversity_score <= 1),
  conflict_score double precision NOT NULL CHECK (conflict_score >= 0 AND conflict_score <= 1),
  hypothesis_evaluation_ids text[] NOT NULL,
  conflicting_evaluation_ids text[] NOT NULL DEFAULT '{}',
  signal_ids text[] NOT NULL DEFAULT '{}',
  supporting_evidence_ids text[] NOT NULL DEFAULT '{}',
  invalidating_evidence_ids text[] NOT NULL DEFAULT '{}',
  summary text NOT NULL,
  opportunity_payload jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(opportunity_payload) = 'object'),
  version integer NOT NULL CHECK (version > 0),
  research_only boolean NOT NULL DEFAULT true,
  build_run_id text NOT NULL,
  deterministic_key text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, deterministic_key, version),
  CHECK (cardinality(hypothesis_evaluation_ids) > 0)
);

CREATE INDEX IF NOT EXISTS idx_marketops_opportunities_symbol_session
  ON marketops_opportunities (tenant_id, symbol, last_evaluated_date DESC, opportunity_score DESC);
CREATE INDEX IF NOT EXISTS idx_marketops_opportunities_work_queue
  ON marketops_opportunities (tenant_id, lifecycle_status, direction, horizon, opportunity_score DESC);
CREATE INDEX IF NOT EXISTS idx_marketops_opportunities_evaluations
  ON marketops_opportunities USING gin (hypothesis_evaluation_ids);
