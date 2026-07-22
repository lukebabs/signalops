CREATE TABLE IF NOT EXISTS marketops_algorithm_adjudications (
  tenant_id text NOT NULL,
  adjudication_id text NOT NULL,
  hypothesis_evaluation_id text NOT NULL,
  algorithm_result_id text NOT NULL,
  hypothesis_key text NOT NULL,
  hypothesis_version text NOT NULL,
  symbol text NOT NULL,
  session_date date NOT NULL,
  verdict text NOT NULL CHECK (verdict IN ('confirmed','contradicted','inconclusive')),
  confidence double precision NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
  explanation jsonb NOT NULL DEFAULT '{}'::jsonb,
  correlation_id text NOT NULL,
  adjudicator_version text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, adjudication_id),
  UNIQUE (tenant_id, hypothesis_evaluation_id, algorithm_result_id, adjudicator_version)
);
CREATE INDEX IF NOT EXISTS idx_marketops_algorithm_adjudications_symbol_session
  ON marketops_algorithm_adjudications (tenant_id, symbol, session_date DESC);
