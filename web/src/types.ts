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
  display_sector: string;
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
  display_name: string;
  display_sector: string;
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
export interface MarketOpsAssetCreateRequest { ticker:string; company?:string; sector?:string; industry?:string; }
export interface MarketOpsAssetDisplayNameRequest { universe_group: string; display_name: string; }
export interface MarketOpsAssetDisplaySectorRequest { universe_group: string; display_sector: string; }
export interface MarketOpsTickerValidation { ticker:string; company:string; exchange:string; sector:string; industry:string; }
export interface MarketOpsAssetOnboardRequest { ticker:string; backfill_equity_history:boolean; start_date?:string; end_date?:string; }
export interface MarketOpsAssetBackfillJob { backfill_job_id:string; tenant_id:string; symbol:string; universe_group:string; start_date:string; end_date:string; status:string; requested_by:string; requested_sessions:number; completed_sessions:number; failed_sessions:number; provider_requests:number; error_message?:string; result:unknown; started_at?:string; completed_at?:string; created_at:string; updated_at:string; }
export interface MarketOpsAssetBackfillJobsResponse { backfill_jobs:MarketOpsAssetBackfillJob[]; }
export interface MarketOpsAssetBackfillCreateRequest { start_date:string; end_date:string; requested_by?:string; }

export interface MarketOpsAssetQuote {
  ticker: string;
  price?: number | null;
  previous_close?: number | null;
  change?: number | null;
  change_percent?: number | null;
  week52_low?: number | null;
  week52_high?: number | null;
  timestamp?: string | null;
  market_status?: string | null;
  stale?: boolean;
  error?: string | null;
}

export interface MarketOpsAssetQuotesResponse {
  quotes: MarketOpsAssetQuote[];
  refreshed_at?: string | null;
}

export interface MarketOpsIntradayCondition {
  key: string; title: string; tone: "positive" | "negative" | "neutral" | string; score: number; evidence: string; interpretation: string; analyst_question: string;
}
export interface MarketOpsIntradayConditionSnapshot {
  snapshot_id: string; ticker: string; as_of_time: string; market_status: string; stale: boolean; conditions: MarketOpsIntradayCondition[]; source: Record<string, unknown>; created_at: string;
}
export interface MarketOpsIntradayConditionsResponse { snapshots: MarketOpsIntradayConditionSnapshot[]; }

export interface MarketOpsAssetFilter {
  tenant_id?: string;
  universe_group?: string;
  active_only?: boolean;
  limit?: number;
}

// G128 MarketOps asset options intelligence (read-only). Served by
// GET /v1/tenants/{tenant_id}/marketops/assets/{symbol}/options/{coverage|
// distribution|chain}. Persisted options-chain coverage, derived call/put
// distribution snapshots, and chain rows. contract_type/window are permissive
// (| string) so unknown tokens render neutrally. Chain numeric columns are
// server-nullable (omitempty pointers) — typed optional and rendered as absent.
// moneyness_distribution / expiration_distribution / metrics / raw_payload
// arrive already parsed (typed unknown / Record); never JSON.parse them.
export type MarketOpsOptionContractType = 'call' | 'put' | string;
export type MarketOpsOptionsWindowName = '10_trade_days' | string;

export interface MarketOpsOptionsBucketTotals {
  call_open_interest?: number;
  put_open_interest?: number;
  call_volume?: number;
  put_volume?: number;
  contract_count?: number;
}

export interface MarketOpsOptionsCoverage {
  tenant_id: string;
  symbol: string;
  trade_day_count: number;
  contract_count: number;
  first_trade_date: string;
  last_trade_date: string;
  last_updated_at: string;
}

export interface MarketOpsOptionsCoverageResponse {
  options_coverage: MarketOpsOptionsCoverage;
}

export interface MarketOpsOptionsDistribution {
  tenant_id: string;
  symbol: string;
  trade_date: string;
  window_name: string;
  source_id: string;
  provider: string;
  trade_days: number;
  contract_count: number;
  call_contract_count: number;
  put_contract_count: number;
  total_call_open_interest: number;
  total_put_open_interest: number;
  total_call_volume: number;
  total_put_volume: number;
  missing_open_interest_count: number;
  call_put_open_interest_ratio: number;
  call_put_volume_ratio: number;
  ratio_delta: number;
  ratio_change_pct: number;
  ratio_zscore: number;
  change_point_score: number;
  confidence: number;
  moneyness_distribution: Record<string, MarketOpsOptionsBucketTotals>;
  expiration_distribution: Record<string, MarketOpsOptionsBucketTotals>;
  metrics: unknown;
  source_trade_dates: string[];
  created_at: string;
  updated_at: string;
}

export interface MarketOpsOptionsDistributionsResponse {
  options_distributions: MarketOpsOptionsDistribution[];
}

export interface MarketOpsOptionsChainRow {
  tenant_id: string;
  symbol: string;
  trade_date: string;
  option_ticker: string;
  provider: string;
  source_id: string;
  ingestion_run_id: string;
  contract_type: string;
  expiration_date: string;
  strike_price: number;
  underlying_close?: number;
  moneyness?: number;
  open?: number;
  high?: number;
  low?: number;
  close?: number;
  vwap?: number;
  volume?: number;
  open_interest?: number;
  implied_volatility?: number;
  delta?: number;
  gamma?: number;
  theta?: number;
  vega?: number;
  provider_request_id: string;
  payload_hash: string;
  raw_payload: unknown;
  created_at: string;
  updated_at: string;
}

export interface MarketOpsOptionsChainResponse {
  options_chain: MarketOpsOptionsChainRow[];
}

// Chain list filter. trade_date is YYYY-MM-DD (date-only). The gateway clamps
// limit to a 200 maximum even though the spec default is 500.
export interface MarketOpsOptionsChainFilter {
  trade_date?: string;
  contract_type?: MarketOpsOptionContractType | '';
  limit?: number;
}

export interface MarketOpsOptionsDistributionFilter {
  window?: string;
  limit?: number;
}

// G139 MarketOps Opportunities workbench (read-only). Served by
// GET /v1/marketops/opportunities and /v1/marketops/opportunities/{opportunity_id},
// plus supporting reads (hypothesis-evaluations, hypotheses, evidence, state
// lineage). lifecycle/direction are permissive (| string) so unknown future
// tokens render neutrally. opportunity_payload + the various JSON rule/payload
// fields arrive already parsed (typed unknown; render via JsonViewer, never
// JSON.parse). Hypothesis-evaluation and evidence scores are server-nullable
// (omitempty pointers) — typed optional and rendered as unavailable when absent.
export type MarketOpsOpportunityLifecycle =
  | 'emerging'
  | 'active'
  | 'strengthening'
  | 'weakening'
  | 'invalidated'
  | 'resolved'
  | 'expired'
  | string;

export type MarketOpsOpportunityDirection = 'upside' | 'downside' | 'non_directional' | string;

export interface MarketOpsOpportunity {
  opportunity_id: string;
  tenant_id: string;
  app_id: string;
  asset_id: string;
  symbol: string;
  opened_session_date: string;
  last_evaluated_date: string;
  direction: MarketOpsOpportunityDirection;
  horizon: string;
  lifecycle_status: MarketOpsOpportunityLifecycle;
  opportunity_score: number;
  confidence_score: number;
  domain_diversity_score: number;
  conflict_score: number;
  hypothesis_evaluation_ids: string[];
  conflicting_evaluation_ids: string[];
  signal_ids: string[];
  supporting_evidence_ids: string[];
  invalidating_evidence_ids: string[];
  summary: string;
  opportunity_payload: unknown;
  version: number;
  research_only: boolean;
  build_run_id: string;
  deterministic_key: string;
  created_at: string;
  updated_at: string;
}

export interface MarketOpsOpportunitiesResponse {
  opportunities: MarketOpsOpportunity[];
}

export interface MarketOpsOpportunityResponse {
  opportunity: MarketOpsOpportunity;
}

export interface MarketOpsOpportunityFilter {
  tenant_id?: string;
  app_id?: string;
  opportunity_id?: string;
  asset_id?: string;
  symbol?: string;
  direction?: MarketOpsOpportunityDirection | '';
  horizon?: string;
  lifecycle_status?: MarketOpsOpportunityLifecycle | '';
  research_only?: boolean;
  session_start?: string;
  session_end?: string;
  limit?: number;
}

// Hypothesis evaluations (G138) — linked detail + empty-queue diagnostics.
export interface MarketOpsHypothesisEvaluation {
  evaluation_id: string;
  tenant_id: string;
  app_id: string;
  hypothesis_key: string;
  hypothesis_version: string;
  market_state_id: string;
  asset_id: string;
  symbol: string;
  session_date: string;
  as_of_time: string;
  eligible: boolean;
  triggered: boolean;
  trigger_score?: number;
  confidence_score?: number;
  magnitude_score?: number;
  rarity_score?: number;
  persistence_score?: number;
  corroboration_score?: number;
  quality_score?: number;
  invalidated: boolean;
  evidence_ids: string[];
  reason_codes: string[];
  evaluation_payload: unknown;
  evaluation_run_id: string;
  deterministic_key: string;
  created_at: string;
}

export interface MarketOpsHypothesisEvaluationsResponse {
  hypothesis_evaluations: MarketOpsHypothesisEvaluation[];
}

export interface MarketOpsHypothesisEvaluationFilter {
  tenant_id?: string;
  app_id?: string;
  hypothesis_key?: string;
  hypothesis_version?: string;
  market_state_id?: string;
  asset_id?: string;
  symbol?: string;
  eligible?: boolean;
  triggered?: boolean;
  invalidated?: boolean;
  session_start?: string;
  session_end?: string;
  limit?: number;
}

// Hypothesis definition (G138). Rule/config fields arrive already parsed.
export interface MarketOpsHypothesisDefinition {
  tenant_id: string;
  hypothesis_key: string;
  hypothesis_version: string;
  title: string;
  domain: string;
  direction: string;
  description: string;
  rationale: unknown;
  required_features: unknown;
  required_transitions: unknown;
  quality_policy: unknown;
  eligibility_expression: unknown;
  trigger_expression: unknown;
  persistence_rule: unknown;
  corroboration_rule: unknown;
  invalidation_rule: unknown;
  expected_outcomes: unknown;
  scoring_config: unknown;
  calibration_policy: unknown;
  lifecycle_status: string;
  owner: string;
  approved_by: string;
  approved_at?: string;
  created_at: string;
  updated_at: string;
}

export interface MarketOpsHypothesisResponse {
  hypothesis: MarketOpsHypothesisDefinition;
}

// Evidence (G136/G137). Scores are server-nullable.
export interface MarketOpsEvidence {
  evidence_id: string;
  tenant_id: string;
  app_id: string;
  asset_id: string;
  symbol: string;
  session_date: string;
  as_of_time: string;
  evidence_type: string;
  evidence_version: string;
  domain: string;
  direction?: string;
  magnitude?: number;
  rarity_score?: number;
  persistence_score?: number;
  quality_score?: number;
  statement: string;
  evidence_payload: unknown;
  source_feature_ids: string[];
  source_transition_ids: string[];
  deterministic_key: string;
  created_at: string;
}

export interface MarketOpsEvidenceResponse {
  evidence: MarketOpsEvidence;
}

// Market state lineage (G136/G137) — reachable from an evaluation's market_state_id.
export interface MarketOpsMarketStateLineage {
  market_state: unknown;
  feature_observations: unknown[];
  source_event_ids: string[];
  source_artifact_ids: string[];
  missing_feature_observation_ids: string[];
}

export interface MarketOpsMarketStateLineageResponse {
  lineage: MarketOpsMarketStateLineage;
}

// G147 Market State analyst experience. Read-only composition over the G136-G146
// market-state ledgers. Market state + feature definitions/observations + state
// transitions + signal outcomes + opportunity dispositions (G146 append-only).
// lifecycle/quality/direction/outcome/disposition enums are permissive (| string)
// so unknown future tokens render neutrally. state_payload / dimensions /
// quality_details / quality_summary / *_payload / rule JSON arrive already parsed
// (typed unknown; render via JsonViewer, never JSON.parse). Pointer/optional
// numeric+boolean fields are genuinely nullable — absent vs zero vs false is
// material and must render distinctly.
export interface MarketOpsMarketState {
  market_state_id: string;
  tenant_id: string;
  app_id: string;
  asset_id: string;
  symbol: string;
  session_date: string;
  as_of_time: string;
  state_schema_version: string;
  state_payload: unknown;
  feature_observation_ids: string[];
  feature_count: number;
  required_feature_count: number;
  completeness_ratio: number;
  quality_state: string;
  quality_score?: number;
  quality_summary: unknown;
  eligible_hypotheses: string[];
  build_run_id: string;
  deterministic_key: string;
  created_at: string;
}

export interface MarketOpsMarketStatesResponse {
  market_states: MarketOpsMarketState[];
}

export interface MarketOpsMarketStateResponse {
  market_state: MarketOpsMarketState;
}

export interface MarketOpsMarketStateFilter {
  tenant_id?: string;
  app_id?: string;
  asset_id?: string;
  symbol?: string;
  state_schema_version?: string;
  quality_state?: string;
  session_start?: string;
  session_end?: string;
  limit?: number;
}

export interface MarketOpsFeatureDefinition {
  tenant_id: string;
  feature_key: string;
  feature_version: string;
  domain: string;
  title: string;
  description: string;
  value_type: string;
  unit?: string;
  calculation_spec: unknown;
  required_inputs: unknown;
  quality_policy: unknown;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface MarketOpsFeatureDefinitionsResponse {
  feature_definitions: MarketOpsFeatureDefinition[];
}

export interface MarketOpsFeatureDefinitionFilter {
  tenant_id?: string;
  feature_key?: string;
  feature_version?: string;
  domain?: string;
  status?: string;
  limit?: number;
}

export interface MarketOpsFeatureObservation {
  feature_observation_id: string;
  tenant_id: string;
  app_id: string;
  asset_id: string;
  symbol: string;
  session_date: string;
  as_of_time: string;
  feature_key: string;
  feature_version: string;
  dimensions: unknown;
  numeric_value?: number;
  text_value?: string;
  boolean_value?: boolean;
  quality_state: string;
  quality_score?: number;
  quality_details: unknown;
  source_event_ids: string[];
  source_artifact_ids: string[];
  calculation_run_id: string;
  deterministic_key: string;
  created_at: string;
}

export interface MarketOpsFeatureObservationsResponse {
  feature_observations: MarketOpsFeatureObservation[];
}

export interface MarketOpsFeatureObservationFilter {
  tenant_id?: string;
  app_id?: string;
  asset_id?: string;
  symbol?: string;
  feature_key?: string;
  feature_version?: string;
  domain?: string;
  quality_state?: string;
  dimensions?: string;
  session_start?: string;
  session_end?: string;
  limit?: number;
}

export interface MarketOpsStateTransition {
  transition_id: string;
  tenant_id: string;
  app_id: string;
  asset_id: string;
  symbol: string;
  session_date: string;
  as_of_time: string;
  current_state_id: string;
  baseline_state_id?: string;
  feature_key: string;
  feature_version: string;
  dimensions: unknown;
  transition_type: string;
  lookback_sessions?: number;
  current_value?: number;
  baseline_value?: number;
  transition_value?: number;
  zscore?: number;
  percentile?: number;
  persistence_sessions?: number;
  direction?: string;
  quality_state: string;
  transition_payload: unknown;
  calculation_run_id: string;
  deterministic_key: string;
  created_at: string;
}

export interface MarketOpsStateTransitionsResponse {
  transitions: MarketOpsStateTransition[];
}

export interface MarketOpsStateTransitionFilter {
  tenant_id?: string;
  app_id?: string;
  asset_id?: string;
  symbol?: string;
  current_state_id?: string;
  feature_key?: string;
  feature_version?: string;
  transition_type?: string;
  quality_state?: string;
  session_start?: string;
  session_end?: string;
  limit?: number;
}

export interface MarketOpsEvidencesResponse {
  evidence: MarketOpsEvidence[];
}

export interface MarketOpsEvidenceFilter {
  tenant_id?: string;
  app_id?: string;
  asset_id?: string;
  symbol?: string;
  evidence_type?: string;
  evidence_version?: string;
  domain?: string;
  direction?: string;
  session_start?: string;
  session_end?: string;
  limit?: number;
}

export interface MarketOpsHypothesesResponse {
  hypotheses: MarketOpsHypothesisDefinition[];
}

export interface MarketOpsHypothesisListFilter {
  tenant_id?: string;
  hypothesis_key?: string;
  hypothesis_version?: string;
  domain?: string;
  lifecycle_status?: string;
  limit?: number;
}

export type MarketOpsOutcomeStatus = 'pending' | 'matured' | 'missing_price' | string;

export interface MarketOpsOutcome {
  outcome_id: string;
  tenant_id: string;
  app_id: string;
  source_type: string;
  source_id: string;
  hypothesis_key?: string;
  hypothesis_version?: string;
  asset_id: string;
  symbol: string;
  direction: string;
  origin_session_date: string;
  horizon_sessions: number;
  matured_session_date?: string;
  outcome_status: MarketOpsOutcomeStatus;
  forward_return?: number;
  max_favorable_excursion?: number;
  max_adverse_excursion?: number;
  maximum_drawdown?: number;
  realized_vol_change?: number;
  directional_hit?: boolean;
  threshold_hit?: boolean;
  days_to_threshold?: number;
  origin_event_id?: string;
  outcome_event_ids: string[];
  outcome_payload: unknown;
  calculation_version: string;
  calculation_run_id: string;
  deterministic_key: string;
  created_at: string;
  updated_at: string;
}

export interface MarketOpsOutcomesResponse {
  outcomes: MarketOpsOutcome[];
}

export interface MarketOpsOutcomeResponse {
  outcome: MarketOpsOutcome;
}

export interface MarketOpsOutcomeFilter {
  tenant_id?: string;
  app_id?: string;
  source_type?: string;
  source_id?: string;
  hypothesis_key?: string;
  hypothesis_version?: string;
  symbol?: string;
  direction?: string;
  outcome_status?: MarketOpsOutcomeStatus | '';
  horizon_sessions?: number;
  session_start?: string;
  session_end?: string;
  limit?: number;
}

// G146 opportunity dispositions (append-only analyst judgment).
export type MarketOpsOpportunityDispositionValue =
  | 'watch'
  | 'advance'
  | 'needs_more_evidence'
  | 'dismiss'
  | 'resolved'
  | string;

export interface MarketOpsOpportunityDisposition {
  disposition_id: string;
  tenant_id: string;
  opportunity_id: string;
  disposition: MarketOpsOpportunityDispositionValue;
  actor: string;
  note: string;
  metadata: unknown;
  created_at: string;
}

export interface MarketOpsOpportunityDispositionsResponse {
  opportunity_dispositions: MarketOpsOpportunityDisposition[];
}

export interface MarketOpsOpportunityDispositionResponse {
  opportunity_disposition: MarketOpsOpportunityDisposition;
}

// POST body. The gateway derives the actor via replayActor (header -> body ->
// operator-local); per the repo convention the client sends NO actor field.
export interface MarketOpsOpportunityDispositionRequest {
  tenant_id: string;
  disposition: MarketOpsOpportunityDispositionValue;
  note?: string;
  metadata?: unknown;
}

export interface MarketOpsOpportunityDispositionFilter {
  tenant_id?: string;
  disposition?: string;
  limit?: number;
}

// G148-C MarketOps intelligence readiness aggregate (read-only). Served by
// GET /v1/marketops/intelligence/readiness. One request serves the whole view —
// never issue per-symbol state/evaluation/opportunity/calibration calls. The
// five readiness dimensions are the flat *_state string fields below. There is
// intentionally no production_ready rollout value; production_ready_supported is
// always false. stage_status/stage_errors/input_coverage/proposal_status_counts
// arrive already parsed (typed unknown). A symbol with an empty
// latest_market_state_id is unobserved.
export type MarketOpsIntelligenceRolloutStatus =
  | 'not_observed'
  | 'inspection_ready'
  | 'research_evaluation_ready'
  | 'review_ready'
  | 'blocked'
  | string;

export interface MarketOpsIntelligenceReadinessDimensionCounts {
  coverage_state?: Record<string, number>;
  evaluation_state?: Record<string, number>;
  governance_state?: Record<string, number>;
  calibration_state?: Record<string, number>;
  outcome_state?: Record<string, number>;
  rollout_status?: Record<string, number>;
}

export interface MarketOpsIntelligenceReadinessAggregate {
  symbol_count: number;
  dimension_counts: MarketOpsIntelligenceReadinessDimensionCounts;
  production_ready_supported: boolean;
  latest_session_date?: string;
}

export interface MarketOpsIntelligenceReadinessSymbol {
  result_id: string;
  run_id: string;
  tenant_id: string;
  universe_group: string;
  symbol: string;
  asset_id: string;
  stage_status: unknown;
  stage_errors: unknown;
  input_coverage: unknown;
  latest_market_state_id: string;
  latest_state_date?: string;
  latest_state_schema_version: string;
  latest_state_quality: string;
  latest_state_completeness: number;
  required_feature_coverage: number;
  surface_coverage: number;
  evaluation_count: number;
  eligible_count: number;
  triggered_count: number;
  evaluation_rejection_reasons: string[];
  opportunity_count: number;
  pending_outcome_count: number;
  matured_outcome_count: number;
  proposal_status_counts: unknown;
  exact_calibration_count: number;
  calibration_below_minimum: boolean;
  coverage_state: string;
  evaluation_state: string;
  governance_state: string;
  calibration_state: string;
  outcome_state: string;
  rollout_status: MarketOpsIntelligenceRolloutStatus;
  readiness_reasons: string[];
  created_at: string;
  updated_at: string;
}

export interface MarketOpsIntelligenceReadiness {
  aggregate: MarketOpsIntelligenceReadinessAggregate;
  symbols: MarketOpsIntelligenceReadinessSymbol[];
}

export interface MarketOpsIntelligenceReadinessResponse {
  readiness: MarketOpsIntelligenceReadiness;
}

// symbols is a comma-separated CSV string; the gateway uppercases + sorts.
export interface MarketOpsIntelligenceReadinessFilter {
  tenant_id?: string;
  universe_group?: string;
  symbols?: string;
  latest_session_date?: string;
  rollout_status?: MarketOpsIntelligenceRolloutStatus | '';
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

// G085 label-aware MarketOps back-test evaluations. An evaluation scores a
// back-test run against synchronized G084 graph-proposal-decision labels and
// stores precision/recall-style metrics. Recommendation tokens are advisory
// evaluation outcomes — never deploy/promote decisions. The token set mirrors
// the G083 advisory comparison recommendations.
export type MarketOpsBacktestEvaluationRecommendation =
  | 'needs_more_data'
  | 'manual_review_required'
  | 'improvement_candidate'
  | 'neutral_candidate'
  | 'regression_candidate'
  | string;

// metrics payload: matched_samples is a capped list of joined
// proposal/label/recommendation rows; scoring_notes carries scoring caveats.
// Both are optional + permissive so a forward-shaped payload renders blanks.
export interface MarketOpsBacktestEvaluationMetrics {
  matched_samples?: unknown[];
  scoring_notes?: string[];
  [key: string]: unknown;
}

export interface MarketOpsBacktestEvaluation {
  evaluation_id: string;
  tenant_id: string;
  app_id: string;
  domain: string;
  use_case: string;
  run_id: string;
  detector_id: string;
  dataset: string;
  label_source: string;
  label_version: string;
  scoring_version: string;
  requested_by: string;
  candidate_count: number;
  labeled_count: number;
  positive_count: number;
  negative_count: number;
  superseded_count: number;
  unresolved_count: number;
  true_positive: number;
  false_positive: number;
  true_negative: number;
  false_negative: number;
  manual_review_count: number;
  unscored_count: number;
  precision: number;
  recall: number;
  specificity: number;
  accuracy: number;
  label_coverage: number;
  recommendation: MarketOpsBacktestEvaluationRecommendation;
  recommendation_note: string;
  metrics: MarketOpsBacktestEvaluationMetrics;
  created_at: string;
}

export interface MarketOpsBacktestEvaluationsResponse {
  backtest_evaluations: MarketOpsBacktestEvaluation[];
}

export interface MarketOpsBacktestEvaluationResponse {
  backtest_evaluation: MarketOpsBacktestEvaluation;
}

// label_source is accepted by the gateway but overridden server-side to the
// canonical G084 source, so it is omitted on create by default.
export interface MarketOpsBacktestEvaluationCreateRequest {
  evaluation_id?: string;
  tenant_id: string;
  run_id: string;
  label_source?: string;
  requested_by?: string;
}

export interface MarketOpsBacktestEvaluationFilter {
  tenant_id?: string;
  app_id?: string;
  domain?: string;
  use_case?: string;
  run_id?: string;
  detector_id?: string;
  dataset?: string;
  recommendation?: MarketOpsBacktestEvaluationRecommendation | '';
  limit?: number;
}

// G086 promotion review candidates. A promotion candidate packages G083
// baseline-comparison evidence and optional G085 evaluation evidence into an
// auditable review record. This is a REVIEW workflow only — recording a
// decision never deploys, edits thresholds, writes graph state, or trains.
export type MarketOpsBacktestPromotionReadinessStatus =
  | 'ready_for_review'
  | 'needs_more_data'
  | 'manual_review_required'
  | 'regression_detected'
  | 'blocked'
  | string;

export type MarketOpsBacktestPromotionCandidateStatus =
  | 'proposed'
  | 'approved_for_promotion'
  | 'rejected'
  | 'deferred'
  | 'superseded'
  | string;

// evidence is a server-built snapshot. Every block is optional + permissive so
// a forward-shaped or partial payload renders blanks instead of throwing.
export interface MarketOpsBacktestPromotionEvidence {
  baseline?: { baseline_id?: string; summary_id?: string; name?: string } & Record<string, unknown>;
  comparison?: {
    comparison_id?: string;
    recommendation?: string;
    recommendation_reason?: string;
    metrics?: Record<string, unknown>;
  } & Record<string, unknown>;
  evaluation?: {
    evaluation_id?: string;
    recommendation?: string;
    recommendation_note?: string;
    precision?: number;
    recall?: number;
    accuracy?: number;
    label_coverage?: number;
    candidate_count?: number;
    labeled_count?: number;
    true_positive?: number;
    false_positive?: number;
    true_negative?: number;
    false_negative?: number;
  } & Record<string, unknown>;
  detector?: { detector_id?: string; detector_version?: string } & Record<string, unknown>;
  run?: { run_id?: string } & Record<string, unknown>;
  policy_version?: string;
  readiness?: { status?: string; reasons?: string[]; [key: string]: unknown };
  [key: string]: unknown;
}

export interface MarketOpsBacktestPromotionCandidate {
  candidate_id: string;
  tenant_id: string;
  app_id: string;
  domain: string;
  use_case: string;
  baseline_id: string;
  comparison_id: string;
  evaluation_id: string;
  run_id: string;
  detector_id: string;
  detector_version: string;
  dataset: string;
  policy_version: string;
  candidate_version: string;
  readiness_status: MarketOpsBacktestPromotionReadinessStatus;
  readiness_reasons: string[];
  evidence: MarketOpsBacktestPromotionEvidence;
  status: MarketOpsBacktestPromotionCandidateStatus;
  requested_by: string;
  reviewed_by: string;
  reviewed_at?: string;
  decision_note: string;
  created_at: string;
  updated_at: string;
}

export interface MarketOpsBacktestPromotionCandidatesResponse {
  promotion_candidates: MarketOpsBacktestPromotionCandidate[];
}

export interface MarketOpsBacktestPromotionCandidateResponse {
  promotion_candidate: MarketOpsBacktestPromotionCandidate;
}

export interface MarketOpsBacktestPromotionCandidateCreateRequest {
  candidate_id?: string;
  tenant_id: string;
  baseline_id: string;
  comparison_id: string;
  evaluation_id?: string;
  candidate_version?: string;
  requested_by?: string;
}

export interface MarketOpsBacktestPromotionCandidateDecisionRequest {
  status: Exclude<MarketOpsBacktestPromotionCandidateStatus, 'proposed'>;
  reviewed_by?: string;
  decision_note?: string;
}

export interface MarketOpsBacktestPromotionCandidateFilter {
  tenant_id?: string;
  app_id?: string;
  domain?: string;
  use_case?: string;
  baseline_id?: string;
  comparison_id?: string;
  evaluation_id?: string;
  run_id?: string;
  detector_id?: string;
  dataset?: string;
  readiness_status?: MarketOpsBacktestPromotionReadinessStatus | '';
  status?: MarketOpsBacktestPromotionCandidateStatus | '';
  limit?: number;
}

// G088 Syncratic synthesized insights + deterministic context windows. An
// insight is a multi-record, pattern-level explanation assembled over a bounded
// evidence window; a context window is the persisted evidence slice it was built
// from. Both are read-only review surfaces — this gate adds no review/dismiss/
// archive mutations (the G088 backend does not expose them) and writes no graph
// state. Flexible JSON fields are typed `unknown` and narrowed defensively.
export type SyncraticInsightStatus =
  | 'active'
  | 'reviewed'
  | 'dismissed'
  | 'archived'
  | 'superseded'
  | string;

// The backend currently only emits "active"; archived/superseded are reserved
// for forward compatibility. Kept permissive (| string) so future values render.
export type SyncraticContextWindowStatus = 'active' | 'archived' | 'superseded' | string;

export type SyncraticSeverity = 'info' | 'low' | 'medium' | 'high' | 'critical' | string;

export interface SyncraticInsightCurrentness {
  is_current: boolean;
  currentness_key: string;
  superseded_by_context_window_id: string;
  superseded_by_syncratic_insight_id: string;
  reason: string;
}

export interface SyncraticInsight {
  syncratic_insight_id: string;
  tenant_id: string;
  app_id: string;
  domain: string;
  use_case: string;
  context_window_id: string;
  insight_type: string;
  subject_type: string;
  subject_id: string;
  subject_symbol: string;
  status: SyncraticInsightStatus;
  severity: SyncraticSeverity;
  confidence: number;
  title: string;
  summary: string;
  explanation: string;
  supporting_alert_ids: string[];
  supporting_signal_ids: string[];
  supporting_event_ids: string[];
  supporting_artifact_ids: string[];
  related_graph_proposal_ids: string[];
  related_label_ids: string[];
  metrics: unknown;
  recommendation: unknown;
  currentness?: SyncraticInsightCurrentness;
  builder_version: string;
  created_at: string;
  updated_at: string;
}

export interface SyncraticContextWindow {
  context_window_id: string;
  tenant_id: string;
  app_id: string;
  domain: string;
  use_case: string;
  subject_type: string;
  subject_id: string;
  subject_symbol: string;
  window_start: string;
  window_end: string;
  context_strategy: string;
  context_builder_version: string;
  signal_types: string[];
  detector_ids: string[];
  event_ids: string[];
  signal_ids: string[];
  alert_ids: string[];
  artifact_ids: string[];
  graph_proposal_ids: string[];
  label_ids: string[];
  baseline_refs: unknown;
  evaluation_refs: unknown;
  promotion_candidate_refs: unknown;
  summary_metrics: unknown;
  evidence_digest: string;
  idempotency_key: string;
  status: SyncraticContextWindowStatus;
  created_at: string;
  updated_at: string;
}

// G091/G092 budgeted materialization. Actions/reasons are narrowed for current
// rendering but accept `| string` so unknown future tokens render in a neutral
// style instead of failing the UI.
export type SyncraticMaterializationAction =
  | 'would_materialize'
  | 'materialized'
  | 'skipped'
  | string;

export type SyncraticMaterializationReason =
  | 'eligible'
  | 'below_threshold'
  | 'unchanged_evidence_digest'
  | 'candidate_budget_cap'
  | 'materialization_budget_cap'
  | string;

export interface SyncraticMaterializationDecision {
  subject_symbol: string;
  action: SyncraticMaterializationAction;
  reason: SyncraticMaterializationReason;
  evidence_count: number;
  signal_count: number;
  alert_count: number;
  artifact_count: number;
  graph_proposal_count: number;
  label_count: number;
  critical_alert: boolean;
  related_evidence: boolean;
  evidence_digest?: string;
  context_window_id?: string;
}

export interface SyncraticMaterializationResult {
  tenant_id: string;
  universe_group: string;
  context_strategy: string;
  context_builder_version: string;
  window_start: string;
  window_end: string;
  dry_run: boolean;
  scanned_assets: number;
  candidate_windows: number;
  materialized_context_windows: number;
  materialized_insights: number;
  skipped_below_threshold: number;
  skipped_unchanged: number;
  skipped_budget_cap: number;
  context_window_ids: string[];
  syncratic_insight_ids: string[];
  decisions: SyncraticMaterializationDecision[];
}

export interface SyncraticInsightsResponse {
  syncratic_insights: SyncraticInsight[];
}

export interface SyncraticInsightResponse {
  syncratic_insight: SyncraticInsight;
}

export interface SyncraticContextWindowsResponse {
  context_windows: SyncraticContextWindow[];
}

export interface SyncraticContextWindowResponse {
  context_window: SyncraticContextWindow;
}

export interface SyncraticMaterializationResponse {
  materialization: SyncraticMaterializationResult;
}

export interface SyncraticInsightFilter {
  tenant_id?: string;
  app_id?: string;
  domain?: string;
  use_case?: string;
  context_window_id?: string;
  insight_type?: string;
  subject_symbol?: string;
  status?: SyncraticInsightStatus | '';
  limit?: number;
}

export interface SyncraticContextWindowFilter {
  tenant_id?: string;
  app_id?: string;
  domain?: string;
  use_case?: string;
  subject_symbol?: string;
  context_strategy?: string;
  status?: SyncraticContextWindowStatus | '';
  limit?: number;
}

// signal_limit/alert_limit exist on the backend (default 1000) but are omitted —
// the bounded materialize form does not expose them and defaults are safe.
// dry_run (G091/G092) selects preview mode: dry_run=true returns a 200 preview
// with decisions[] and writes nothing; dry_run=false creates/updates rows (201).
export interface SyncraticMaterializeRequest {
  tenant_id: string;
  universe_group?: string;
  context_strategy?: string;
  context_builder_version?: string;
  window_start: string;
  window_end: string;
  min_evidence_count?: number;
  max_assets?: number;
  max_candidate_windows?: number;
  max_context_windows?: number;
  max_insights?: number;
  dry_run?: boolean;
}

// G090 operator-triggered Syncratic Ask enrichment. Mirrors the gateway DTOs in
// internal/api/syncratic.go (syncraticAskRequest / syncraticAskResult). The route
// operates on an already-persisted context window: force=false skips when the
// prompt+evidence digest is unchanged; force=true regenerates. prompt_builder_version
// and include_record_details are accepted by the backend but omitted here (defaults).
export interface SyncraticAskRequest {
  tenant_id: string;
  max_prompt_bytes?: number;
  force?: boolean;
}

// ask_status is "completed" (updated insight written) or "skipped"
// (unchanged_prompt_and_evidence, force=false, no write). updated mirrors ask_status
// for convenience. Fields are always present in the route response.
export interface SyncraticAskResult {
  context_window_id: string;
  syncratic_insight_id: string;
  ask_query_id: string;
  ask_status: string;
  prompt_digest: string;
  updated: boolean;
  skipped_reason: string;
  prompt_builder_version: string;
}

// The route always returns the full refreshed insight under syncratic_insight, even
// on skip (the pre-existing insight row). Typed as SyncraticInsight; summarizeSyncratic*
// helpers tolerate a malformed/empty payload without throwing.
export interface SyncraticAskResponse {
  ask_result: SyncraticAskResult;
  syncratic_insight: SyncraticInsight;
}

// G109 algorithm execution visibility (read-only). Mirrors the gateway DTOs in
// internal/api/algorithms.go. Flexible JSON fields (output_schema, config_schema,
// default_config, metadata, config, result, result_payload) arrive already-parsed
// from the gateway and are typed `unknown`; render via JsonViewer, never JSON.parse.
// status/severity/result_type/runtime_type are permissive (| string) so unknown
// future tokens render in a neutral style instead of failing.
export interface AlgorithmDefinition {
  algorithm_id: string;
  tenant_id: string;
  name: string;
  description: string;
  algorithm_type: string;
  runtime_type: string;
  input_features: string[];
  input_event_types: string[];
  output_schema: unknown;
  config_schema: unknown;
  default_config: unknown;
  version: string;
  status: string;
  metadata: unknown;
  created_at: string;
  updated_at: string;
}

export interface AlgorithmExecutionRequest {
  execution_request_id: string;
  tenant_id: string;
  algorithm_id: string;
  algorithm_version: string;
  event_ids: string[];
  feature_refs: string[];
  entity_refs: string[];
  window_ref: string;
  config: unknown;
  correlation_id: string;
  status: string;
  requested_by: string;
  result: unknown;
  error_message: string;
  created_at: string;
  updated_at: string;
}

export interface AlgorithmResult {
  algorithm_result_id: string;
  tenant_id: string;
  algorithm_id: string;
  algorithm_version: string;
  execution_request_id: string;
  result_type: string;
  score: number;
  confidence: number;
  severity: string;
  result_payload: unknown;
  source_event_ids: string[];
  feature_value_ids: string[];
  evidence_refs: string[];
  correlation_id: string;
  created_at: string;
}

export interface AlgorithmExecutionSummary {
  execution_request: AlgorithmExecutionRequest;
  result_count: number;
  severity_counts: Record<string, number>;
  max_score: number;
  max_confidence: number;
  top_results: AlgorithmResult[];
}

export interface AlgorithmDefinitionFilter {
  tenant_id?: string;
  algorithm_type?: string;
  runtime_type?: string;
  status?: string;
  limit?: number;
}

export interface AlgorithmExecutionRequestFilter {
  tenant_id?: string;
  algorithm_id?: string;
  status?: string;
  correlation_id?: string;
  limit?: number;
}

export interface AlgorithmResultFilter {
  tenant_id?: string;
  algorithm_id?: string;
  execution_request_id?: string;
  result_type?: string;
  severity?: string;
  correlation_id?: string;
  limit?: number;
}

export interface AlgorithmDefinitionsResponse {
  algorithm_definitions: AlgorithmDefinition[];
}
export interface AlgorithmDefinitionResponse {
  algorithm_definition: AlgorithmDefinition;
}
export interface AlgorithmExecutionRequestsResponse {
  algorithm_execution_requests: AlgorithmExecutionRequest[];
}
export interface AlgorithmExecutionRequestResponse {
  algorithm_execution_request: AlgorithmExecutionRequest;
}
export interface AlgorithmExecutionSummaryResponse {
  algorithm_execution_summary: AlgorithmExecutionSummary;
}
export interface AlgorithmResultsResponse {
  algorithm_results: AlgorithmResult[];
}
export interface AlgorithmResultResponse {
  algorithm_result: AlgorithmResult;
}

// G113/G114 algorithm signal proposals review surface. Mirrors the G111/G112
// gateway DTOs in internal/api/algorithms.go (algorithmSignalProposalDTO).
// This is a REVIEW-ONLY ledger: it never materializes production signals,
// alerts, insights, or graph proposals — the status union intentionally has no
// `accepted` token. proposal_payload / rationale arrive already-parsed from the
// gateway (typed `unknown`); render via JsonViewer, never JSON.parse.
// decided_at is omitempty on the backend (absent until a decision is recorded).
// status/severity are permissive (| string) so unknown future tokens render in a
// neutral style instead of failing the UI.
export type AlgorithmSignalProposalStatus =
  | 'proposed'
  | 'reviewed'
  | 'rejected'
  | 'superseded'
  | string;

export interface AlgorithmSignalProposal {
  proposal_id: string;
  tenant_id: string;
  proposal_source?: string;
  algorithm_result_id: string;
  algorithm_id: string;
  algorithm_version: string;
  execution_request_id: string;
  hypothesis_evaluation_id?: string;
  hypothesis_key?: string;
  hypothesis_version?: string;
  hypothesis_lifecycle_status?: string;
  proposed_signal_type: string;
  status: AlgorithmSignalProposalStatus;
  score: number;
  confidence: number;
  severity: string;
  proposal_payload: unknown;
  rationale: unknown;
  source_event_ids: string[];
  evidence_refs: string[];
  correlation_id: string;
  research_only?: boolean;
  materialization_eligible?: boolean;
  eligibility_snapshot?: unknown;
  created_by: string;
  reviewed_by: string;
  decision_note: string;
  decided_at?: string;
  created_at: string;
  updated_at: string;
}

export interface AlgorithmSignalProposalFilter {
  tenant_id?: string;
  proposal_source?: string;
  algorithm_id?: string;
  execution_request_id?: string;
  algorithm_result_id?: string;
  hypothesis_evaluation_id?: string;
  hypothesis_key?: string;
  status?: AlgorithmSignalProposalStatus | '';
  severity?: string;
  correlation_id?: string;
  limit?: number;
}

// Decision body for POST /v1/algorithms/signal-proposals/{proposal_id}/decision.
// The gateway derives the reviewer via replayActor (header -> body -> operator-
// local), so no actor header is sent — matching the promotion-candidate decision.
// metadata is optional; the backend defaults it to {}.
export interface AlgorithmSignalProposalDecisionRequest {
  tenant_id: string;
  status: AlgorithmSignalProposalStatus;
  note?: string;
  metadata?: unknown;
}

export interface AlgorithmSignalProposalsResponse {
  algorithm_signal_proposals: AlgorithmSignalProposal[];
}

export interface AlgorithmSignalProposalResponse {
  algorithm_signal_proposal: AlgorithmSignalProposal;
}

// G115/G116 algorithm signal proposal review-coverage summary. Mirrors the
// algorithmSignalProposalSummaryDTO in internal/api/algorithms.go. Read-only
// aggregate over the G111/G112 ledger; carries no materialization semantics.
// Count maps are backend-owned string->int; the *_counts fields are typed
// Record<string, number> and narrowed defensively in the lib summarizer.
export interface AlgorithmSignalProposalSummary {
  tenant_id: string;
  total_proposals: number;
  proposed_count: number;
  reviewed_count: number;
  rejected_count: number;
  superseded_count: number;
  reviewed_ratio: number;
  high_critical_unreviewed_count: number;
  status_counts: Record<string, number>;
  severity_counts: Record<string, number>;
  proposed_signal_type_counts: Record<string, number>;
  algorithm_id_counts: Record<string, number>;
  reviewer_counts: Record<string, number>;
}

export interface AlgorithmSignalProposalSummaryResponse {
  algorithm_signal_proposal_summary: AlgorithmSignalProposalSummary;
}

// G119 algorithm signal materialization preflight (read-only). Mirrors the
// algorithmSignalProposalMaterializationPreflightDTO in internal/api/algorithms.go
// (G118 backend). This is a READ-ONLY preflight: it forecasts whether reviewed
// proposals *would* be eligible for a later materialization gate — it writes no
// production signal, alert, insight, graph proposal, or policy. preflight_status
// is permissive (| string) so unknown future tokens render in a neutral style.
// global_blocking_reasons / item_reason_counts are backend-owned string->int maps;
// reasons / duplicate_signal_ids / source_event_ids are backend-owned string[].
export type AlgorithmSignalMaterializationPreflightStatus =
  | 'eligible'
  | 'duplicate_risk'
  | 'blocked'
  | 'invalid'
  | string;

export interface AlgorithmSignalMaterializationPreflightItem {
  proposal_id: string;
  algorithm_result_id: string;
  algorithm_id: string;
  execution_request_id: string;
  proposed_signal_type: string;
  status: string;
  severity: string;
  confidence: number;
  preflight_status: AlgorithmSignalMaterializationPreflightStatus;
  reasons: string[];
  duplicate_signal_ids: string[];
  source_event_ids: string[];
  would_write: boolean;
  materialization_policy: string;
}

export interface AlgorithmSignalMaterializationPreflight {
  tenant_id: string;
  policy_version: string;
  total_proposals: number;
  eligible_count: number;
  duplicate_risk_count: number;
  blocked_count: number;
  invalid_count: number;
  would_write_count: number;
  reviewed_ratio: number;
  min_reviewed_ratio: number;
  review_coverage_satisfied: boolean;
  high_critical_unreviewed_count: number;
  global_blocking_reasons: Record<string, number>;
  item_reason_counts: Record<string, number>;
  items: AlgorithmSignalMaterializationPreflightItem[];
}

export interface AlgorithmSignalMaterializationPreflightResponse {
  algorithm_signal_materialization_preflight: AlgorithmSignalMaterializationPreflight;
}

// Preflight filter: the coupled proposal-list filters (tenant id, algorithm id,
// execution request id, algorithm result id, status, severity, correlation id,
// limit) plus the two preflight-only params. min_reviewed_ratio defaults to 1 and
// policy_version to materialization_preflight.v1 (applied in the API client).
export interface AlgorithmSignalMaterializationPreflightFilter {
  tenant_id?: string;
  algorithm_id?: string;
  execution_request_id?: string;
  algorithm_result_id?: string;
  status?: AlgorithmSignalProposalStatus | '';
  severity?: string;
  correlation_id?: string;
  limit?: number;
  min_reviewed_ratio?: number;
  policy_version?: string;
}

// G121/G122/G123 algorithm signal materialization. Mirrors the
// algorithmSignalMaterializationDTO in internal/api/algorithms.go. The G122 POST
// materializes a reviewed eligible proposal into a production signal; it is the
// first write control on this surface. materialization_status is permissive
// (| string) so unknown future tokens render in a neutral style. The POST returns
// 201/200 with this envelope for succeeded/duplicate/blocked (branch on status);
// only not-found/auth/server failures throw. requested_at is always present;
// started_at/completed_at/failed_at are omitempty (absent until set). The three
// JSON fields arrive already-parsed (typed unknown; render via JsonViewer).
export type AlgorithmSignalMaterializationStatus =
  | 'requested'
  | 'running'
  | 'succeeded'
  | 'duplicate'
  | 'blocked'
  | 'failed'
  | 'superseded'
  | string;

export interface AlgorithmSignalMaterialization {
  materialization_id: string;
  tenant_id: string;
  proposal_id: string;
  algorithm_result_id: string;
  execution_request_id: string;
  algorithm_id: string;
  algorithm_version: string;
  proposed_signal_type: string;
  signal_id: string;
  materialization_status: AlgorithmSignalMaterializationStatus;
  materialization_policy_version: string;
  idempotency_key: string;
  duplicate_of_signal_id: string;
  requested_by: string;
  requested_at: string;
  started_at?: string;
  completed_at?: string;
  failed_at?: string;
  error_code: string;
  error_message: string;
  request_metadata: unknown;
  preflight_snapshot: unknown;
  signal_payload_preview: unknown;
  created_at: string;
  updated_at: string;
}

// POST /v1/algorithms/signal-proposals/{proposal_id}/materializations body.
// tenant_id is read from the body first (then query); policy_version is the fixed
// algorithm_materialization.v1 default; requested_by and idempotency_key are
// derived server-side (never sent). metadata defaults to {} when omitted.
export interface AlgorithmSignalMaterializationRequest {
  tenant_id: string;
  policy_version?: string;
  metadata?: unknown;
}

export interface AlgorithmSignalMaterializationResponse {
  algorithm_signal_materialization: AlgorithmSignalMaterialization;
}

export interface AlgorithmSignalMaterializationsResponse {
  algorithm_signal_materializations: AlgorithmSignalMaterialization[];
}

// Ledger list filter. tenant_id is required by the gateway (defaults to
// tenant-local in the client); the rest are optional narrowers.
export interface AlgorithmSignalMaterializationFilter {
  tenant_id?: string;
  proposal_id?: string;
  algorithm_result_id?: string;
  execution_request_id?: string;
  algorithm_id?: string;
  status?: AlgorithmSignalMaterializationStatus | '';
  signal_id?: string;
  limit?: number;
}

export interface MarketOpsAlgorithmAdjudication { adjudication_id:string; tenant_id:string; hypothesis_evaluation_id:string; algorithm_result_id:string; hypothesis_key:string; hypothesis_version:string; symbol:string; session_date:string; verdict:'confirmed'|'contradicted'|'inconclusive'; confidence:number; explanation:unknown; correlation_id:string; adjudicator_version:string; created_at:string; }
export interface MarketOpsAlgorithmAdjudicationFilter { tenant_id?:string; symbol?:string; hypothesis_evaluation_id?:string; correlation_id?:string; limit?:number; }
export interface MarketOpsAlgorithmAdjudicationsResponse { algorithm_adjudications:MarketOpsAlgorithmAdjudication[]; }

export interface MarketOpsQuantitativeSeriesPoint { trade_date:string; eod_close?:number; daily_move_pct?:number; call_put_open_interest_ratio?:number; call_put_volume_ratio?:number; ratio_quality?:string; markers:Array<{algorithm_result_id:string;algorithm_id:string;severity:string;score:number;confidence:number;direction?:string;adjudications?:Array<{verdict:string;hypothesis_key:string}>}>; }
export interface MarketOpsQuantitativeSeriesResponse { symbol:string; window:string; points:MarketOpsQuantitativeSeriesPoint[]; }
export interface MarketOpsEODZScore { trade_date:string; algorithm_result:AlgorithmResult | null; status:'selected' | 'no_usable_zscore'; reason?:string; }
export type MarketOpsRiskRewardDirection = "bullish" | "bearish" | "neutral";
export type MarketOpsRiskRewardFactor = { key:string; direction:MarketOpsRiskRewardDirection; weight:number; value:number; detail:string; };
export type MarketOpsRiskRewardCorroboration = { direction:MarketOpsRiskRewardDirection | "unavailable"; agreement:"aligned" | "bullish" | "bearish" | "neutral"; put_call_volume_ratio?:number; deviation_pct?:number; };
export type MarketOpsRiskRewardPoint = { algorithm_result_id:string; trade_date:string; score:number; direction:MarketOpsRiskRewardDirection; risk_level:"low" | "medium" | "high" | "unavailable"; confidence:number; severity:"info" | "low" | "medium" | "high"; technical_factors:MarketOpsRiskRewardFactor[]; speculative_corroboration:MarketOpsRiskRewardCorroboration; research_only:true; };
export type MarketOpsRiskRewardResponse = { latest?:MarketOpsRiskRewardPoint; history:MarketOpsRiskRewardPoint[]; };
export type MarketOpsRiskRewardSummary = { ticker:string; trade_date:string; direction:MarketOpsRiskRewardDirection; score:number; confidence:number; risk_level:"low" | "medium" | "high" | "unavailable"; research_only:true; };
export interface MarketOpsRiskRewardSummariesResponse { summaries:MarketOpsRiskRewardSummary[]; }
export interface MarketOpsAssetAlgorithmObservationsResponse { symbol:string; eod_zscores:MarketOpsEODZScore[]; other_outputs:AlgorithmResult[]; risk_reward?:MarketOpsRiskRewardResponse; }
