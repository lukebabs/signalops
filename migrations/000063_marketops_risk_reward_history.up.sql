-- Immutable, queryable MarketOps Risk/Reward session revisions.  A later
-- degraded calculation must never erase the evidence of an earlier usable one.
CREATE TABLE IF NOT EXISTS marketops_risk_reward_snapshots (
  snapshot_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  algorithm_result_id text NOT NULL,
  execution_request_id text NOT NULL,
  symbol text NOT NULL,
  session_date date NOT NULL,
  observed_at timestamptz NOT NULL,
  technical_score double precision NOT NULL,
  technical_direction text NOT NULL CHECK (technical_direction IN ('bullish', 'bearish', 'neutral')),
  risk_level text NOT NULL CHECK (risk_level IN ('low', 'medium', 'high', 'unavailable')),
  confidence double precision NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
  usable_input_count integer NOT NULL CHECK (usable_input_count >= 0),
  required_input_count integer NOT NULL CHECK (required_input_count > 0),
  eligible boolean NOT NULL,
  result_payload jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(result_payload) = 'object'),
  input_snapshot jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(input_snapshot) = 'object'),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, algorithm_result_id)
);

CREATE INDEX IF NOT EXISTS idx_marketops_risk_reward_snapshots_symbol_session
  ON marketops_risk_reward_snapshots (tenant_id, symbol, session_date DESC, eligible DESC, usable_input_count DESC, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_marketops_risk_reward_snapshots_retention
  ON marketops_risk_reward_snapshots (tenant_id, session_date);

-- Preserve the currently available historical output.  The snapshot is
-- intentionally append-only; scoring writes fuller input snapshots going forward.
INSERT INTO marketops_risk_reward_snapshots (
  snapshot_id, tenant_id, algorithm_result_id, execution_request_id, symbol,
  session_date, observed_at, technical_score, technical_direction, risk_level,
  confidence, usable_input_count, required_input_count, eligible,
  result_payload, input_snapshot, created_at
)
SELECT
  'rrsnap_' || substr(md5(ar.tenant_id || '|' || ar.algorithm_result_id), 1, 32),
  ar.tenant_id, ar.algorithm_result_id, ar.execution_request_id,
  upper(ar.result_payload->>'symbol'),
  (ar.result_payload->>'observation_time')::timestamptz::date,
  (ar.result_payload->>'observation_time')::timestamptz,
  COALESCE((ar.result_payload->>'technical_score')::double precision, 0),
  CASE WHEN ar.result_payload->>'technical_direction' IN ('bullish', 'bearish', 'neutral') THEN ar.result_payload->>'technical_direction' ELSE 'neutral' END,
  CASE WHEN ar.result_payload->>'risk_level' IN ('low', 'medium', 'high', 'unavailable') THEN ar.result_payload->>'risk_level' ELSE 'unavailable' END,
  ar.confidence,
  COALESCE((ar.result_payload #>> '{confidence_basis,usable_technical_inputs}')::integer, 0),
  GREATEST(COALESCE((ar.result_payload #>> '{confidence_basis,required_technical_inputs}')::integer, 8), 1),
  COALESCE((ar.result_payload #>> '{confidence_basis,usable_technical_inputs}')::integer, 0) >= 5,
  ar.result_payload,
  jsonb_build_object('technical_factors', COALESCE(ar.result_payload->'technical_factors', '[]'::jsonb), 'source_event_ids', to_jsonb(ar.source_event_ids), 'feature_value_ids', to_jsonb(ar.feature_value_ids), 'backfilled', true),
  ar.created_at
FROM algorithm_results ar
WHERE ar.algorithm_id='signalops.algorithms.risk_reward_temporal_v1'
  AND COALESCE(ar.result_payload->>'symbol', '') <> ''
  AND COALESCE(ar.result_payload->>'observation_time', '') <> ''
  AND (ar.result_payload->>'observation_time')::timestamptz::date >= CURRENT_DATE - INTERVAL '365 days'
ON CONFLICT (tenant_id, algorithm_result_id) DO NOTHING;
