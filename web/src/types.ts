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
  app_id?: string;
  domain?: string;
  use_case?: string;
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

export interface NormalizedEventFilter {
  tenant_id?: string;
  app_id?: string;
  domain?: string;
  use_case?: string;
  source_id?: string;
  dataset?: string;
  limit?: number;
}

export interface NormalizedEvent {
  event_id: string;
  tenant_id: string;
  source_id: string;
  source_adapter: string;
  dataset: string;
  schema_id: string;
  schema_version: string;
  observation_time: string;
  processing_time: string;
  confidence: number;
  entities: unknown;
  evidence: unknown;
  metadata: unknown;
  event: unknown;
  raw_topic: string;
  raw_partition: number;
  raw_offset: number;
  normalized_topic: string;
  normalized_partition: number;
  normalized_offset: number;
  created_at: string;
  updated_at: string;
}

export interface NormalizedEventsResponse {
  normalized_events: NormalizedEvent[];
}

export interface NormalizedEventResponse {
  normalized_event: NormalizedEvent;
}

export interface SignalFilter {
  tenant_id?: string;
  app_id?: string;
  domain?: string;
  use_case?: string;
  source_id?: string;
  dataset?: string;
  detector_id?: string;
  severity?: string;
  limit?: number;
}

export interface SignalRecord {
  signal_id: string;
  tenant_id: string;
  source_id: string;
  source_adapter: string;
  dataset: string;
  detector_id: string;
  detector_version: string;
  model_version: string;
  signal_type: string;
  severity: string;
  confidence: number;
  event_ids: string[];
  window_start: string;
  window_end: string;
  entities: unknown;
  supporting_metrics: unknown;
  graph_targets: unknown;
  semantic_evidence: unknown;
  evidence: unknown;
  recommendation: unknown;
  event: unknown;
  broker_topic: string;
  broker_partition: number;
  broker_offset: number;
  created_at: string;
  updated_at: string;
}

export interface SignalsResponse {
  signals: SignalRecord[];
}

export interface SignalResponse {
  signal: SignalRecord;
}

export type AlertStatus = 'open' | 'acknowledged' | 'resolved' | 'suppressed';
export type InsightStatus = 'active' | 'reviewed' | 'dismissed' | 'archived';

export interface AlertFilter {
  tenant_id?: string;
  app_id?: string;
  domain?: string;
  use_case?: string;
  source_id?: string;
  dataset?: string;
  severity?: string;
  status?: string;
  limit?: number;
}

export interface AlertRecord {
  alert_id: string;
  tenant_id: string;
  source_id: string;
  source_domain: string;
  source_adapter: string;
  dataset: string;
  signal_id: string;
  detector_id: string;
  alert_type: string;
  severity: string;
  status: string;
  title: string;
  summary: string;
  confidence: number;
  event_ids: string[];
  entities: unknown;
  evidence: unknown;
  recommendation: unknown;
  correlation_id: string;
  first_observed_at: string;
  last_observed_at: string;
  acknowledged_at?: string;
  acknowledged_by?: string;
  resolved_at?: string;
  resolved_by?: string;
  metadata: unknown;
  created_at: string;
  updated_at: string;
}

export interface AlertsResponse {
  alerts: AlertRecord[];
}

export interface AlertResponse {
  alert: AlertRecord;
}

export interface InsightFilter {
  tenant_id?: string;
  app_id?: string;
  domain?: string;
  use_case?: string;
  source_id?: string;
  dataset?: string;
  insight_type?: string;
  status?: string;
  limit?: number;
}

export interface InsightRecord {
  insight_id: string;
  tenant_id: string;
  source_id: string;
  source_domain: string;
  source_adapter: string;
  dataset: string;
  signal_id: string;
  detector_id: string;
  insight_type: string;
  status: string;
  title: string;
  summary: string;
  confidence: number;
  severity: string;
  event_ids: string[];
  entities: unknown;
  supporting_metrics: unknown;
  semantic_evidence: unknown;
  recommendation: unknown;
  correlation_id: string;
  observed_at: string;
  reviewed_at?: string;
  reviewed_by?: string;
  metadata: unknown;
  created_at: string;
  updated_at: string;
}

export interface InsightsResponse {
  insights: InsightRecord[];
}

export interface InsightResponse {
  insight: InsightRecord;
}

export type AlertLifecycleAction = 'acknowledge' | 'resolve' | 'suppress';
export type InsightLifecycleAction = 'review' | 'dismiss' | 'archive';

export interface LifecycleMutationRequest {
  actor?: string;
  note?: string;
  reason?: string;
}

export interface AlertLifecycleMutationOptions extends LifecycleMutationRequest {
  alertId: string;
  action: AlertLifecycleAction;
}

export interface InsightLifecycleMutationOptions extends LifecycleMutationRequest {
  insightId: string;
  action: InsightLifecycleAction;
}

// Replay jobs mirror the gateway DTOs in internal/api/router.go (replayJobDTO /
// replayJobCreateRequest). The backend persists filters/options/result as raw
// JSON (always present, never omitted), so they are typed `unknown` and rendered
// via JsonViewer. Status/source_kind/replay_mode unions are narrowed for the
// current controls but accept `| string` for forward compatibility.
export type ReplaySourceKind = 'raw_events' | 'normalized_events' | 'signals';
export type ReplayMode = 'original' | 'latest_compatible' | 'explicit';
export type ReplayJobStatus = 'queued' | 'running' | 'succeeded' | 'failed' | 'canceled';

export interface ReplayJob {
  replay_job_id: string;
  tenant_id: string;
  source_id?: string;
  dataset?: string;
  source_kind: ReplaySourceKind | string;
  replay_mode: ReplayMode | string;
  status: ReplayJobStatus | string;
  requested_by: string;
  window_start: string;
  window_end: string;
  started_at?: string;
  completed_at?: string;
  filters: unknown;
  options: unknown;
  result: unknown;
  error_message?: string;
  created_at: string;
  updated_at: string;
}

export interface ReplayJobsResponse {
  replay_jobs: ReplayJob[];
}

export interface ReplayJobResponse {
  replay_job: ReplayJob;
}

export interface ReplayJobCreateRequest {
  tenant_id: string;
  source_id?: string;
  dataset?: string;
  source_kind?: ReplaySourceKind;
  replay_mode?: ReplayMode;
  requested_by?: string;
  window_start: string;
  window_end: string;
  filters?: Record<string, unknown>;
  options?: Record<string, unknown>;
}

export interface ReplayJobFilter {
  tenant_id?: string;
  source_id?: string;
  dataset?: string;
  source_kind?: ReplaySourceKind | '';
  status?: ReplayJobStatus | '';
  limit?: number;
}

// G061 replay result accounting. The worker writes `canceled: false` (bool) on
// normal completion via CompleteReplayJob, while CancelReplayJob merges an
// object into `result.canceled` ({actor, reason, canceled_at}); the union
// tolerates both. Historical G059 results carry only a subset of these fields,
// so every field is optional. `[key: string]: unknown` keeps the shape permissive
// for forward-compatible backend fields without rewrites.
export type ReplayRecordStatus = 'published' | 'failed' | string;

export interface ReplayRecordResult {
  source_id: string;
  key: string;
  status: ReplayRecordStatus;
  topic?: string;
  partition?: number;
  offset?: number;
  attempts?: number;
  error?: string;
}

export interface ReplayCancellationResult {
  actor?: string;
  reason?: string;
  canceled_at?: string;
}

export interface ReplayResult {
  replay_job_id?: string;
  source_kind?: ReplaySourceKind | string;
  scanned?: number;
  published?: number;
  failed?: number;
  batches?: number;
  max_records?: number;
  batch_size?: number;
  canceled?: boolean | ReplayCancellationResult;
  started_at?: string;
  completed_at?: string;
  records?: ReplayRecordResult[];
  [key: string]: unknown;
}

export interface ReplayJobCancelRequest {
  actor?: string;
  reason?: string;
  note?: string;
}

// G064 replay operations observability. `health` (online/stale/error) is
// backend-derived from heartbeat recency and worker status; `unknown` is a
// frontend-only fallback when no heartbeats exist. `job_counts` is always a
// full map of the five replay statuses (0-filled by the backend).
export type ReplayWorkerHealth = 'online' | 'stale' | 'error' | string;
export type ReplayWorkerStatus = 'idle' | 'running' | 'error' | 'stopping' | string;

export interface ReplayWorkerStatusRecord {
  worker_id: string;
  status: ReplayWorkerStatus;
  health: ReplayWorkerHealth;
  process_started_at: string;
  last_seen_at: string;
  last_claimed_at?: string;
  last_claimed_replay_job_id?: string;
  last_completed_at?: string;
  last_completed_replay_job_id?: string;
  last_error_at?: string;
  last_error_message?: string;
  metadata: unknown;
  created_at: string;
  updated_at: string;
}

export interface ReplayOperationsStatus {
  generated_at: string;
  job_counts: Record<string, number>;
  workers: ReplayWorkerStatusRecord[];
  latest_jobs: ReplayJob[];
}

export interface ReplayOperationsStatusResponse {
  replay_status: ReplayOperationsStatus;
}

// G066 app profiles. The backend serves a static list at GET /v1/app-profiles.
export interface AppProfile {
  app_id: string;
  label: string;
  default_route: string;
  domains: string[];
  enabled_modules: string[];
  dashboard_profile: string;
}

export interface AppProfilesResponse {
  app_profiles: AppProfile[];
}

// G071 MarketOps asset universe (read-only). Served by
// GET /v1/tenants/{tenant_id}/marketops/assets. Strings stay permissive — the
// universe is backend-owned and must not be encoded as a TS union.
export interface MarketOpsAsset {
  tenant_id: string;
  app_id: string;
  domain: string;
  use_case: string;
  source_id: string;
  universe_group: string;
  rank: number;
  ticker: string;
  ticker_key: string;
  company: string;
  company_key: string;
  asset_type: string;
  exchange: string;
  sector: string;
  sector_key: string;
  industry: string;
  industry_key: string;
  is_active: boolean;
  metadata: unknown;
  created_at: string;
  updated_at: string;
}

export interface MarketOpsAssetsResponse {
  assets: MarketOpsAsset[];
}

export interface MarketOpsAssetFilter {
  tenant_id?: string;
  universe_group?: string;
  active_only?: boolean;
  limit?: number;
}


// G078 MarketOps DSM artifacts. Served by
// GET /v1/marketops/dsm/artifacts and /v1/marketops/dsm/artifacts/{artifact_id}.
export interface MarketOpsDSMArtifact {
  artifact_id: string;
  tenant_id: string;
  app_id: string;
  domain: string;
  use_case: string;
  source_id: string;
  source_adapter: string;
  dataset: string;
  signal_id: string;
  signal_type: string;
  detector_id: string;
  severity: string;
  confidence: number;
  event_ids: string[];
  subject_symbol: string;
  artifact_type: string;
  artifact: unknown;
  semantic_evidence: unknown;
  graph_targets: unknown;
  supporting_metrics: unknown;
  quality_issues: string[];
  created_at: string;
  updated_at: string;
}

export interface MarketOpsDSMArtifactsResponse {
  artifacts: MarketOpsDSMArtifact[];
}

export interface MarketOpsDSMArtifactResponse {
  artifact: MarketOpsDSMArtifact;
}

export interface MarketOpsDSMArtifactFilter {
  tenant_id?: string;
  app_id?: string;
  domain?: string;
  use_case?: string;
  signal_type?: string;
  severity?: string;
  subject_symbol?: string;
  limit?: number;
}

// G079 MarketOps DSM graph proposals (read-only). Served by
// GET /v1/marketops/dsm/graph-proposals and /v1/marketops/dsm/graph-proposals/{proposal_id}.
// The backend persists graph target candidates emitted by the detector so
// operators can review what the graph layer would materialize, without the UI
// ever issuing graph writes. `properties` and `raw_candidate` arrive as
// already-parsed JSON from the gateway (typed `unknown`); never JSON.parse them.
// Status/candidate_type are narrowed unions for current rendering but accept
// `| string` for forward compatibility with future backend values.
export type MarketOpsDSMGraphProposalStatus = 'proposed' | 'accepted' | 'rejected' | 'superseded';
export type MarketOpsDSMGraphProposalCandidateType = 'node_candidate' | 'relationship_candidate';

export interface MarketOpsDSMGraphProposal {
  proposal_id: string;
  tenant_id: string;
  app_id: string;
  domain: string;
  use_case: string;
  source_id: string;
  source_adapter: string;
  dataset: string;
  artifact_id: string;
  signal_id: string;
  signal_type: string;
  detector_id: string;
  severity: string;
  confidence: number;
  event_ids: string[];
  subject_symbol: string;
  candidate_type: MarketOpsDSMGraphProposalCandidateType | string;
  node_id: string;
  from_node: string;
  relationship: string;
  to_node: string;
  labels: string[];
  properties: unknown;
  raw_candidate: unknown;
  status: MarketOpsDSMGraphProposalStatus | string;
  reviewed_by: string;
  decision_note: string;
  decided_at?: string;
  created_at: string;
  updated_at: string;
}

export interface MarketOpsDSMGraphProposalsResponse {
  graph_proposals: MarketOpsDSMGraphProposal[];
}

export interface MarketOpsDSMGraphProposalResponse {
  graph_proposal: MarketOpsDSMGraphProposal;
}


export interface MarketOpsDSMGraphProposalDecisionRequest {
  status: MarketOpsDSMGraphProposalStatus;
  note?: string;
}

export interface MarketOpsDSMGraphProposalDecisionOptions extends MarketOpsDSMGraphProposalDecisionRequest {
  proposalId: string;
}

export interface MarketOpsDSMGraphProposalFilter {
  tenant_id?: string;
  app_id?: string;
  domain?: string;
  use_case?: string;
  artifact_id?: string;
  signal_id?: string;
  signal_type?: string;
  subject_symbol?: string;
  candidate_type?: MarketOpsDSMGraphProposalCandidateType | string;
  status?: MarketOpsDSMGraphProposalStatus | string;
  limit?: number;
}
