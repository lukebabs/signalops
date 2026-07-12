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

// G081 MarketOps back-test workspace (isolated experimental runs). Mirrors the
// gateway DTOs in internal/api/marketops_backtests.go. Back-test rows are
// experiment records: signals come from marketops_backtest_signals (not the
// production signal_ledger) and graph proposals from
// marketops_backtest_graph_proposals (not production marketops_dsm_graph_proposals).
// The synchronous runner always returns a terminal run, but `started` is kept in
// the union for forward compatibility. Run status has no `canceled` state today.
export type MarketOpsBacktestRunStatus = 'started' | 'succeeded' | 'failed' | string;
export type MarketOpsBacktestRecommendation =
  | 'auto_accept_candidate'
  | 'auto_reject_candidate'
  | 'manual_review_required'
  | 'supersede_candidate'
  | string;

// Metrics JSON parsed by the gateway from marketops_backtest_runs.metrics.
// Every field is optional + permissive (`[key: string]: unknown`) so a
// forward-shaped or partial payload renders blanks instead of throwing.
export interface MarketOpsBacktestMetrics {
  run_id?: string;
  scanned?: number;
  signals?: number;
  artifacts?: number;
  graph_proposals?: number;
  policy_results?: number;
  recommendation_counts?: Record<string, number>;
  batches?: number;
  max_records?: number;
  batch_size?: number;
  started_at?: string;
  completed_at?: string;
  [key: string]: unknown;
}

export interface MarketOpsBacktestRun {
  run_id: string;
  tenant_id: string;
  app_id: string;
  domain: string;
  use_case: string;
  source_id: string;
  source_adapter: string;
  dataset: string;
  detector_id: string;
  detector_version: string;
  status: MarketOpsBacktestRunStatus;
  requested_by: string;
  window_start: string;
  window_end: string;
  started_at: string;
  completed_at?: string;
  filters: unknown;
  parameters: unknown;
  metrics: unknown;
  error_message?: string;
  created_at: string;
  updated_at: string;
}

export interface MarketOpsBacktestRunsResponse {
  backtest_runs: MarketOpsBacktestRun[];
}
export interface MarketOpsBacktestRunResponse {
  backtest_run: MarketOpsBacktestRun;
}

export interface MarketOpsBacktestCreateRequest {
  run_id?: string;
  tenant_id: string;
  source_id?: string;
  source_adapter?: string;
  dataset?: string;
  detector_id?: string;
  detector_version?: string;
  requested_by?: string;
  window_start: string;
  window_end: string;
  symbols?: string[];
  max_records?: number;
  batch_size?: number;
  auto_accept_confidence?: number;
}

export interface MarketOpsBacktestCreateResponse {
  backtest_run: MarketOpsBacktestRun;
  metrics: unknown;
}

// Each back-test signal wraps a production-shaped SignalRecord under `signal`.
export interface MarketOpsBacktestSignal {
  run_id: string;
  signal: SignalRecord;
}
export interface MarketOpsBacktestSignalsResponse {
  backtest_signals: MarketOpsBacktestSignal[];
}

// Each back-test graph proposal wraps a MarketOpsDSMGraphProposal under
// `graph_proposal`. Recommendation/reason are NOT on the proposal — they live
// on the paired policy_results entry (joined by proposal_id in the UI).
export interface MarketOpsBacktestGraphProposal {
  run_id: string;
  graph_proposal: MarketOpsDSMGraphProposal;
}

export interface MarketOpsBacktestPolicyResult {
  run_id: string;
  policy_result_id: string;
  proposal_id: string;
  artifact_id: string;
  signal_id: string;
  tenant_id: string;
  subject_symbol: string;
  candidate_type: string;
  recommendation: MarketOpsBacktestRecommendation;
  reason: string;
  policy_version: string;
  confidence: number;
  decision_inputs: unknown;
  created_at: string;
}

export interface MarketOpsBacktestGraphProposalsResponse {
  backtest_graph_proposals: MarketOpsBacktestGraphProposal[];
  policy_results: MarketOpsBacktestPolicyResult[];
}

export interface MarketOpsBacktestRunFilter {
  tenant_id?: string;
  app_id?: string;
  domain?: string;
  use_case?: string;
  source_id?: string;
  dataset?: string;
  detector_id?: string;
  status?: MarketOpsBacktestRunStatus | '';
  limit?: number;
}

export interface MarketOpsBacktestSignalFilter {
  tenant_id?: string;
  signal_type?: string;
  limit?: number;
}

export interface MarketOpsBacktestGraphProposalFilter {
  tenant_id?: string;
  signal_type?: string;
  subject_symbol?: string;
  candidate_type?: string;
  recommendation?: MarketOpsBacktestRecommendation | '';
  limit?: number;
}


// G082 persisted MarketOps back-test calibration summaries. These are stored
// snapshots over isolated back-test runs, not production ledger rows.
export interface MarketOpsBacktestCalibrationSummary {
  summary_id: string;
  tenant_id: string;
  app_id: string;
  domain: string;
  use_case: string;
  source_id: string;
  dataset: string;
  detector_id: string;
  status_filter: string;
  requested_by: string;
  run_ids: string[];
  run_count: number;
  succeeded_count: number;
  failed_count: number;
  zero_input_count: number;
  scanned: number;
  signals: number;
  artifacts: number;
  graph_proposals: number;
  policy_results: number;
  signal_yield: number;
  policy_results_per_signal: number;
  recommendation_counts: Record<string, number>;
  recommendation_shares: Record<string, number>;
  dominant_recommendation: { key?: string; count?: number; share?: number } | Record<string, unknown>;
  filters: unknown;
  parameters: unknown;
  created_at: string;
}

export interface MarketOpsBacktestCalibrationSummariesResponse {
  calibration_summaries: MarketOpsBacktestCalibrationSummary[];
}

export interface MarketOpsBacktestCalibrationSummaryResponse {
  calibration_summary: MarketOpsBacktestCalibrationSummary;
}

export interface MarketOpsBacktestCalibrationSummaryCreateRequest {
  summary_id?: string;
  tenant_id: string;
  app_id?: string;
  domain?: string;
  use_case?: string;
  source_id?: string;
  dataset?: string;
  detector_id?: string;
  status?: MarketOpsBacktestRunStatus | '';
  limit?: number;
  requested_by?: string;
}

export interface MarketOpsBacktestCalibrationSummaryFilter {
  tenant_id?: string;
  app_id?: string;
  domain?: string;
  use_case?: string;
  source_id?: string;
  dataset?: string;
  detector_id?: string;
  limit?: number;
}

// G083 persisted MarketOps back-test calibration baselines + stored
// baseline-to-candidate comparisons. A baseline is a named handle over a G082
// persisted summary; a comparison is a stored snapshot of baseline vs candidate
// metrics. Both are advisory calibration tooling — never production graph state.
export type MarketOpsBacktestCalibrationBaselineStatus = 'active' | 'archived' | string;

export interface MarketOpsBacktestCalibrationBaseline {
  baseline_id: string;
  tenant_id: string;
  app_id: string;
  domain: string;
  use_case: string;
  name: string;
  description: string;
  summary_id: string;
  detector_id: string;
  dataset: string;
  scope: unknown;
  status: MarketOpsBacktestCalibrationBaselineStatus;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface MarketOpsBacktestCalibrationBaselinesResponse {
  calibration_baselines: MarketOpsBacktestCalibrationBaseline[];
}

export interface MarketOpsBacktestCalibrationBaselineResponse {
  calibration_baseline: MarketOpsBacktestCalibrationBaseline;
}

export interface MarketOpsBacktestCalibrationBaselineCreateRequest {
  baseline_id?: string;
  tenant_id: string;
  name: string;
  description?: string;
  summary_id: string;
  scope?: unknown;
  status?: MarketOpsBacktestCalibrationBaselineStatus | '';
  created_by?: string;
}

export interface MarketOpsBacktestCalibrationBaselineFilter {
  tenant_id?: string;
  app_id?: string;
  domain?: string;
  use_case?: string;
  detector_id?: string;
  dataset?: string;
  status?: MarketOpsBacktestCalibrationBaselineStatus | '';
  limit?: number;
}

// Advisory comparison recommendation values. The frontend renders these as
// labels only — never as deploy/promote decisions.
export type MarketOpsBacktestCalibrationComparisonRecommendation =
  | 'needs_more_data'
  | 'regression_candidate'
  | 'improvement_candidate'
  | 'neutral_candidate'
  | 'manual_review_required'
  | string;

// comparison_metrics snapshot for one side of a comparison. Every field is
// optional + permissive so a forward-shaped or partial payload renders blanks.
export interface MarketOpsBacktestCalibrationComparisonSnapshot {
  summary_id?: string;
  run_count?: number;
  zero_input_count?: number;
  zero_input_rate?: number;
  scanned?: number;
  signals?: number;
  policy_results?: number;
  signal_yield?: number;
  policy_results_per_signal?: number;
  recommendation_shares?: Record<string, number>;
  dominant_recommendation?: string;
  [key: string]: unknown;
}

export interface MarketOpsBacktestCalibrationComparisonMetrics {
  baseline?: MarketOpsBacktestCalibrationComparisonSnapshot;
  candidate?: MarketOpsBacktestCalibrationComparisonSnapshot;
  // Deltas is an open map keyed by `<metric>_delta` plus the boolean
  // `dominant_recommendation_changed`; typed permissive so unknown future
  // deltas render instead of throwing.
  deltas?: Record<string, unknown>;
  [key: string]: unknown;
}

export interface MarketOpsBacktestCalibrationComparison {
  comparison_id: string;
  tenant_id: string;
  baseline_id: string;
  baseline_summary_id: string;
  candidate_summary_id: string;
  detector_id: string;
  dataset: string;
  comparison_metrics: MarketOpsBacktestCalibrationComparisonMetrics;
  recommendation: MarketOpsBacktestCalibrationComparisonRecommendation;
  recommendation_reason: string;
  created_by: string;
  created_at: string;
}

export interface MarketOpsBacktestCalibrationComparisonsResponse {
  calibration_comparisons: MarketOpsBacktestCalibrationComparison[];
}

export interface MarketOpsBacktestCalibrationComparisonResponse {
  calibration_comparison: MarketOpsBacktestCalibrationComparison;
}

export interface MarketOpsBacktestCalibrationComparisonCreateRequest {
  comparison_id?: string;
  tenant_id: string;
  baseline_id: string;
  candidate_summary_id: string;
  created_by?: string;
}

export interface MarketOpsBacktestCalibrationComparisonFilter {
  tenant_id?: string;
  baseline_id?: string;
  detector_id?: string;
  dataset?: string;
  recommendation?: MarketOpsBacktestCalibrationComparisonRecommendation | '';
  limit?: number;
}
