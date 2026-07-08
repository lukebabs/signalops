// Types mirror the gateway DTOs in internal/api/router.go.
// Optional (`omitempty`) fields may be absent in real responses.

export type RunStatus = 'started' | 'succeeded' | 'failed' | 'canceled';
export type IdempotencyStatus =
  | 'accepted'
  | 'published'
  | 'processed'
  | 'failed'
  | 'duplicate';

export interface HealthResponse {
  status: string; // "ok" for /healthz, "ready" for /readyz
  service: string;
  time: string;
}

export interface SchedulerRun {
  run_id: string;
  tenant_id: string;
  source_id: string;
  source_adapter: string;
  datasets: string[];
  observation_date: string;
  dry_run: boolean;
  status: string;
  started_at: string;
  completed_at?: string;
  events_built: number;
  events_published: number;
  provider_requests: number;
  provider_retries: number;
  failures: number;
  config: unknown;
  report: unknown;
  error_message?: string;
  created_at: string;
  updated_at: string;
}

export interface SchedulerRunsResponse {
  runs: SchedulerRun[];
}
export interface SchedulerRunResponse {
  run: SchedulerRun;
}

export interface ProviderUsage {
  usage_id: string;
  run_id: string;
  provider: string;
  dataset: string;
  request_count: number;
  retry_count: number;
  event_count: number;
  budget: unknown;
  created_at: string;
}
export interface ProviderUsageResponse {
  provider_usage: ProviderUsage[];
}

export interface RawEventFilter {
  tenant_id?: string;
  source_id?: string;
  dataset?: string;
  limit?: number;
}

export interface RawEvent {
  event_id: string;
  tenant_id: string;
  source_id: string;
  source_adapter: string;
  dataset: string;
  idempotency_key: string;
  observation_time: string;
  processing_time: string;
  broker_topic?: string;
  broker_partition?: number;
  broker_offset?: number;
  payload: unknown;
  entity_hints: unknown;
  created_at: string;
}
export interface RawEventsResponse {
  raw_events: RawEvent[];
}
export interface RawEventResponse {
  raw_event: RawEvent;
}

export interface IdempotencyRecord {
  tenant_id: string;
  source_id: string;
  idempotency_key: string;
  event_id: string;
  source_adapter: string;
  dataset: string;
  topic?: string;
  partition?: number;
  offset?: number;
  payload_hash?: string;
  status: string;
  metadata: unknown;
  first_seen_at: string;
  last_seen_at: string;
}
export interface IdempotencyResponse {
  idempotency: IdempotencyRecord;
}

export type CatalogSourceStatus = 'active' | 'inactive' | 'deprecated';

export interface CatalogSource {
  tenant_id: string;
  source_id: string;
  source_domain: string;
  source_adapter: string;
  display_name: string;
  description: string;
  status: string;
  ingestion_modes: string[];
  datasets: string[];
  metadata: unknown;
  created_at: string;
  updated_at: string;
}

export interface CatalogSourcesResponse {
  sources: CatalogSource[];
}

export type CatalogPipelineStatus = 'active' | 'inactive' | 'deprecated';

export interface CatalogPipeline {
  tenant_id: string;
  pipeline_id: string;
  source_id: string;
  source_domain: string;
  pipeline_name: string;
  description: string;
  status: string;
  stages: string[];
  input_datasets: string[];
  output_topics: string[];
  metadata: unknown;
  created_at: string;
  updated_at: string;
}

export interface CatalogPipelinesResponse {
  pipelines: CatalogPipeline[];
}

export type CatalogRuleStatus = 'active' | 'inactive' | 'deprecated';
export type CatalogRuleSeverity = 'info' | 'low' | 'medium' | 'high' | 'critical';

export interface CatalogRule {
  tenant_id: string;
  rule_id: string;
  rule_name: string;
  description: string;
  rule_type: string;
  severity: string;
  status: string;
  version: number;
  source_id?: string;
  pipeline_id?: string;
  dataset_scope: string[];
  entity_scope: string[];
  expression: unknown;
  actions: string[];
  metadata: unknown;
  created_at: string;
  updated_at: string;
}

export interface CatalogRulesResponse {
  rules: CatalogRule[];
}
