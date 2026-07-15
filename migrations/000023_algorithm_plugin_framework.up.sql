-- Generic SignalOps algorithm plugin framework substrate.

CREATE TABLE IF NOT EXISTS algorithm_definitions (
  tenant_id text NOT NULL,
  algorithm_id text NOT NULL,
  name text NOT NULL,
  description text NOT NULL DEFAULT '',
  algorithm_type text NOT NULL CHECK (algorithm_type IN ('anomaly_detection', 'clustering', 'forecasting', 'classification', 'change_point_detection', 'trend_detection')),
  runtime_type text NOT NULL CHECK (runtime_type IN ('python_plugin', 'container_plugin', 'http_plugin')),
  input_features text[] NOT NULL DEFAULT '{}',
  input_event_types text[] NOT NULL DEFAULT '{}',
  output_schema jsonb NOT NULL DEFAULT '{}'::jsonb,
  config_schema jsonb NOT NULL DEFAULT '{}'::jsonb,
  default_config jsonb NOT NULL DEFAULT '{}'::jsonb,
  version text NOT NULL,
  status text NOT NULL CHECK (status IN ('draft', 'active', 'disabled', 'deprecated')),
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, algorithm_id)
);

CREATE INDEX IF NOT EXISTS idx_algorithm_definitions_tenant_status
  ON algorithm_definitions (tenant_id, status, algorithm_id);
CREATE INDEX IF NOT EXISTS idx_algorithm_definitions_type_runtime
  ON algorithm_definitions (tenant_id, algorithm_type, runtime_type, algorithm_id);

CREATE TABLE IF NOT EXISTS algorithm_execution_requests (
  tenant_id text NOT NULL,
  execution_request_id text NOT NULL,
  algorithm_id text NOT NULL,
  algorithm_version text NOT NULL,
  event_ids text[] NOT NULL DEFAULT '{}',
  feature_refs text[] NOT NULL DEFAULT '{}',
  entity_refs text[] NOT NULL DEFAULT '{}',
  window_ref text NOT NULL DEFAULT '',
  config jsonb NOT NULL DEFAULT '{}'::jsonb,
  correlation_id text NOT NULL,
  status text NOT NULL CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'canceled')),
  requested_by text NOT NULL DEFAULT 'operator-local',
  result jsonb NOT NULL DEFAULT '{}'::jsonb,
  error_message text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, execution_request_id)
);

CREATE INDEX IF NOT EXISTS idx_algorithm_execution_requests_algorithm_status
  ON algorithm_execution_requests (tenant_id, algorithm_id, status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_algorithm_execution_requests_correlation
  ON algorithm_execution_requests (tenant_id, correlation_id);

CREATE TABLE IF NOT EXISTS algorithm_results (
  tenant_id text NOT NULL,
  algorithm_result_id text NOT NULL,
  algorithm_id text NOT NULL,
  algorithm_version text NOT NULL,
  execution_request_id text NOT NULL,
  result_type text NOT NULL,
  score double precision NOT NULL CHECK (score >= 0),
  confidence double precision NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
  severity text NOT NULL CHECK (severity IN ('info', 'low', 'medium', 'high', 'critical')),
  result_payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  source_event_ids text[] NOT NULL DEFAULT '{}',
  feature_value_ids text[] NOT NULL DEFAULT '{}',
  evidence_refs text[] NOT NULL DEFAULT '{}',
  correlation_id text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, algorithm_result_id)
);

CREATE INDEX IF NOT EXISTS idx_algorithm_results_algorithm_created
  ON algorithm_results (tenant_id, algorithm_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_algorithm_results_execution
  ON algorithm_results (tenant_id, execution_request_id);
CREATE INDEX IF NOT EXISTS idx_algorithm_results_severity
  ON algorithm_results (tenant_id, severity, created_at DESC);

INSERT INTO algorithm_definitions (
  tenant_id, algorithm_id, name, description, algorithm_type, runtime_type,
  input_features, input_event_types, output_schema, config_schema, default_config,
  version, status, metadata
) VALUES
  ('tenant-local', 'signalops.algorithms.zscore_anomaly_v1', 'Z-Score Anomaly', 'Standard-library statistical anomaly baseline for bounded time-series windows.', 'anomaly_detection', 'python_plugin', ARRAY['value'], ARRAY['normalized_event'], '{"type":"object"}'::jsonb, '{"type":"object"}'::jsonb, '{"z_threshold":3.0}'::jsonb, 'v1', 'draft', '{"library":"python_stdlib","phase":"G107"}'::jsonb),
  ('tenant-local', 'signalops.algorithms.river_anomaly_v1', 'River Streaming Anomaly', 'Online anomaly detection plugin using River for streaming time-series use cases.', 'anomaly_detection', 'python_plugin', ARRAY['value'], ARRAY['normalized_event'], '{"type":"object"}'::jsonb, '{"type":"object"}'::jsonb, '{}'::jsonb, 'v1', 'draft', '{"library":"river","phase":"G108"}'::jsonb),
  ('tenant-local', 'signalops.algorithms.ruptures_change_point_v1', 'Ruptures Change Point', 'Change-point detection plugin for time-series regime shifts.', 'change_point_detection', 'python_plugin', ARRAY['value'], ARRAY['normalized_event'], '{"type":"object"}'::jsonb, '{"type":"object"}'::jsonb, '{}'::jsonb, 'v1', 'draft', '{"library":"ruptures","phase":"G108"}'::jsonb),
  ('tenant-local', 'signalops.algorithms.statsmodels_forecast_v1', 'Statsmodels Forecast', 'Transparent statistical forecasting plugin for time-series baselines.', 'forecasting', 'python_plugin', ARRAY['value'], ARRAY['normalized_event'], '{"type":"object"}'::jsonb, '{"type":"object"}'::jsonb, '{}'::jsonb, 'v1', 'draft', '{"library":"statsmodels","phase":"G109"}'::jsonb),
  ('tenant-local', 'signalops.algorithms.sklearn_classifier_v1', 'Scikit-Learn Classifier', 'Generic supervised classifier plugin for labeled tabular time-series features.', 'classification', 'python_plugin', ARRAY['features'], ARRAY['normalized_event'], '{"type":"object"}'::jsonb, '{"type":"object"}'::jsonb, '{}'::jsonb, 'v1', 'draft', '{"library":"scikit-learn","phase":"G109","requires_labels":true}'::jsonb),
  ('tenant-local', 'signalops.algorithms.sklearn_isolation_forest_v1', 'Scikit-Learn Isolation Forest', 'Batch anomaly detection plugin for time-series feature vectors.', 'anomaly_detection', 'python_plugin', ARRAY['features'], ARRAY['normalized_event'], '{"type":"object"}'::jsonb, '{"type":"object"}'::jsonb, '{}'::jsonb, 'v1', 'draft', '{"library":"scikit-learn","phase":"G109"}'::jsonb)
ON CONFLICT (tenant_id, algorithm_id) DO UPDATE SET
  name = EXCLUDED.name,
  description = EXCLUDED.description,
  algorithm_type = EXCLUDED.algorithm_type,
  runtime_type = EXCLUDED.runtime_type,
  input_features = EXCLUDED.input_features,
  input_event_types = EXCLUDED.input_event_types,
  output_schema = EXCLUDED.output_schema,
  config_schema = EXCLUDED.config_schema,
  default_config = EXCLUDED.default_config,
  version = EXCLUDED.version,
  status = EXCLUDED.status,
  metadata = EXCLUDED.metadata,
  updated_at = now();
