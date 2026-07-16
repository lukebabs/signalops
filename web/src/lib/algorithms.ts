// Pure display helpers for the G109 algorithm execution visibility UI.
// Algorithm definition / execution-request / result / summary JSON arrives
// already-parsed from the gateway (typed `unknown` on flexible fields). Narrow
// with type guards only; never JSON.parse. Missing/malformed values collapse to
// 0 / '' / [] and must never throw. These power a read-only review surface —
// they never start executions, edit definitions, or convert results to signals.

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null && !Array.isArray(v);
}

function asNumber(v: unknown): number {
  if (typeof v === 'number' && Number.isFinite(v)) return v;
  if (typeof v === 'string' && v !== '') {
    const n = Number(v);
    if (Number.isFinite(n)) return n;
  }
  return 0;
}

function asString(v: unknown): string {
  return typeof v === 'string' ? v : '';
}

function asStringArray(v: unknown): string[] {
  return Array.isArray(v) ? v.filter((x): x is string => typeof x === 'string') : [];
}

// Coerce a severity-counts map (gateway returns map[string]int) into a stable,
// display-ordered array. Unknown severities sort last alphabetically.
const SEVERITY_RANK: Record<string, number> = {
  critical: 0,
  high: 1,
  medium: 2,
  low: 3,
  info: 4,
};

export interface AlgorithmSeverityCount {
  severity: string;
  count: number;
}

export function algorithmSeverityCountEntries(counts: unknown): AlgorithmSeverityCount[] {
  if (!isRecord(counts)) return [];
  return Object.entries(counts)
    .filter(([, c]) => typeof c === 'number')
    .map(([severity, count]) => ({ severity, count: count as number }))
    .sort((a, b) => {
      const ra = SEVERITY_RANK[a.severity] ?? 99;
      const rb = SEVERITY_RANK[b.severity] ?? 99;
      if (ra !== rb) return ra - rb;
      return a.severity.localeCompare(b.severity);
    });
}

// Generic count-map -> ordered entries for the G116 summary breakdowns
// (status / proposed signal type / algorithm id / reviewer). Sorts by count
// desc then key asc so the display is stable and the heaviest buckets lead.
export interface AlgorithmCountEntry {
  key: string;
  count: number;
}

export function algorithmCountEntries(counts: unknown): AlgorithmCountEntry[] {
  if (!isRecord(counts)) return [];
  return Object.entries(counts)
    .filter(([, c]) => typeof c === 'number')
    .map(([key, count]) => ({ key, count: count as number }))
    .sort((a, b) => b.count - a.count || a.key.localeCompare(b.key));
}

// Restrained severity token colors, mirroring the app-wide severity palette.
const SEVERITY_STYLES: Record<string, string> = {
  critical: 'text-red-700',
  high: 'text-orange-700',
  medium: 'text-amber-700',
  low: 'text-gray-700',
  info: 'text-gray-500',
};

export function algorithmSeverityStyle(severity: string): string {
  return SEVERITY_STYLES[severity] ?? 'text-gray-600';
}

// Restrained algorithm-definition status colors. active -> green, draft -> amber,
// disabled/deprecated -> muted gray. Unknown future values fall back to gray.
const DEFINITION_STATUS_STYLES: Record<string, string> = {
  active: 'border-emerald-200 bg-emerald-50 text-emerald-700',
  draft: 'border-amber-200 bg-amber-50 text-amber-700',
  disabled: 'border-gray-200 bg-gray-50 text-gray-500',
  deprecated: 'border-gray-200 bg-gray-50 text-gray-400',
};

export function algorithmDefinitionStatusStyle(status: string): string {
  return DEFINITION_STATUS_STYLES[status] ?? 'border-gray-200 bg-gray-50 text-gray-600';
}

// Restrained execution-request status colors. succeeded -> green, running ->
// blue, failed -> red, queued/canceled -> gray. Unknown values fall back to gray.
const EXECUTION_STATUS_STYLES: Record<string, string> = {
  succeeded: 'border-emerald-200 bg-emerald-50 text-emerald-700',
  running: 'border-blue-200 bg-blue-50 text-blue-700',
  failed: 'border-red-200 bg-red-50 text-red-700',
  queued: 'border-gray-200 bg-gray-50 text-gray-600',
  canceled: 'border-gray-200 bg-gray-50 text-gray-400',
};

export function algorithmExecutionStatusStyle(status: string): string {
  return EXECUTION_STATUS_STYLES[status] ?? 'border-gray-200 bg-gray-50 text-gray-600';
}

// Restrained algorithm signal proposal review-status colors (G113/G114). Mirrors
// the spec tone guidance: proposed -> neutral/pending gray-blue, reviewed ->
// positive/complete-but-not-production emerald (NOT a deploy/accept green),
// rejected -> negative red, superseded -> muted secondary gray. Unknown future
// values fall back to neutral gray. Never implies `reviewed` == accepted/deployed.
const PROPOSAL_STATUS_STYLES: Record<string, string> = {
  proposed: 'border-blue-200 bg-blue-50 text-blue-700',
  reviewed: 'border-emerald-200 bg-emerald-50 text-emerald-700',
  rejected: 'border-red-200 bg-red-50 text-red-700',
  superseded: 'border-gray-200 bg-gray-100 text-gray-500',
};

export function algorithmProposalStatusStyle(status: string): string {
  return PROPOSAL_STATUS_STYLES[status] ?? 'border-gray-200 bg-gray-50 text-gray-600';
}

export interface AlgorithmDefinitionSummary {
  algorithmId: string;
  name: string;
  description: string;
  algorithmType: string;
  runtimeType: string;
  inputFeatures: string[];
  inputEventTypes: string[];
  version: string;
  status: string;
  createdAt: string;
  updatedAt: string;
}

const EMPTY_DEFINITION: AlgorithmDefinitionSummary = {
  algorithmId: '',
  name: '',
  description: '',
  algorithmType: '',
  runtimeType: '',
  inputFeatures: [],
  inputEventTypes: [],
  version: '',
  status: '',
  createdAt: '',
  updatedAt: '',
};

export function summarizeAlgorithmDefinition(d: unknown): AlgorithmDefinitionSummary {
  if (!isRecord(d)) return { ...EMPTY_DEFINITION };
  return {
    algorithmId: asString(d.algorithm_id),
    name: asString(d.name),
    description: asString(d.description),
    algorithmType: asString(d.algorithm_type),
    runtimeType: asString(d.runtime_type),
    inputFeatures: asStringArray(d.input_features),
    inputEventTypes: asStringArray(d.input_event_types),
    version: asString(d.version),
    status: asString(d.status),
    createdAt: asString(d.created_at),
    updatedAt: asString(d.updated_at),
  };
}

export interface AlgorithmExecutionRequestSummary {
  executionRequestId: string;
  algorithmId: string;
  algorithmVersion: string;
  eventIds: string[];
  featureRefs: string[];
  entityRefs: string[];
  windowRef: string;
  correlationId: string;
  status: string;
  requestedBy: string;
  errorMessage: string;
  createdAt: string;
  updatedAt: string;
}

const EMPTY_EXECUTION_REQUEST: AlgorithmExecutionRequestSummary = {
  executionRequestId: '',
  algorithmId: '',
  algorithmVersion: '',
  eventIds: [],
  featureRefs: [],
  entityRefs: [],
  windowRef: '',
  correlationId: '',
  status: '',
  requestedBy: '',
  errorMessage: '',
  createdAt: '',
  updatedAt: '',
};

export function summarizeAlgorithmExecutionRequest(r: unknown): AlgorithmExecutionRequestSummary {
  if (!isRecord(r)) return { ...EMPTY_EXECUTION_REQUEST };
  return {
    executionRequestId: asString(r.execution_request_id),
    algorithmId: asString(r.algorithm_id),
    algorithmVersion: asString(r.algorithm_version),
    eventIds: asStringArray(r.event_ids),
    featureRefs: asStringArray(r.feature_refs),
    entityRefs: asStringArray(r.entity_refs),
    windowRef: asString(r.window_ref),
    correlationId: asString(r.correlation_id),
    status: asString(r.status),
    requestedBy: asString(r.requested_by),
    errorMessage: asString(r.error_message),
    createdAt: asString(r.created_at),
    updatedAt: asString(r.updated_at),
  };
}

export interface AlgorithmResultSummary {
  algorithmResultId: string;
  algorithmId: string;
  algorithmVersion: string;
  executionRequestId: string;
  resultType: string;
  score: number;
  confidence: number;
  severity: string;
  sourceEventIds: string[];
  featureValueIds: string[];
  evidenceRefs: string[];
  correlationId: string;
  createdAt: string;
}

const EMPTY_RESULT: AlgorithmResultSummary = {
  algorithmResultId: '',
  algorithmId: '',
  algorithmVersion: '',
  executionRequestId: '',
  resultType: '',
  score: 0,
  confidence: 0,
  severity: '',
  sourceEventIds: [],
  featureValueIds: [],
  evidenceRefs: [],
  correlationId: '',
  createdAt: '',
};

export function summarizeAlgorithmResult(r: unknown): AlgorithmResultSummary {
  if (!isRecord(r)) return { ...EMPTY_RESULT };
  return {
    algorithmResultId: asString(r.algorithm_result_id),
    algorithmId: asString(r.algorithm_id),
    algorithmVersion: asString(r.algorithm_version),
    executionRequestId: asString(r.execution_request_id),
    resultType: asString(r.result_type),
    score: asNumber(r.score),
    confidence: asNumber(r.confidence),
    severity: asString(r.severity),
    sourceEventIds: asStringArray(r.source_event_ids),
    featureValueIds: asStringArray(r.feature_value_ids),
    evidenceRefs: asStringArray(r.evidence_refs),
    correlationId: asString(r.correlation_id),
    createdAt: asString(r.created_at),
  };
}

export interface AlgorithmSignalProposalSummary {
  proposalId: string;
  tenantId: string;
  algorithmResultId: string;
  algorithmId: string;
  algorithmVersion: string;
  executionRequestId: string;
  proposedSignalType: string;
  status: string;
  score: number;
  confidence: number;
  severity: string;
  proposalPayload: unknown;
  rationale: unknown;
  sourceEventIds: string[];
  evidenceRefs: string[];
  correlationId: string;
  createdBy: string;
  reviewedBy: string;
  decisionNote: string;
  decidedAt: string;
  createdAt: string;
  updatedAt: string;
}

const EMPTY_PROPOSAL: AlgorithmSignalProposalSummary = {
  proposalId: '',
  tenantId: '',
  algorithmResultId: '',
  algorithmId: '',
  algorithmVersion: '',
  executionRequestId: '',
  proposedSignalType: '',
  status: '',
  score: 0,
  confidence: 0,
  severity: '',
  proposalPayload: {},
  rationale: {},
  sourceEventIds: [],
  evidenceRefs: [],
  correlationId: '',
  createdBy: '',
  reviewedBy: '',
  decisionNote: '',
  decidedAt: '',
  createdAt: '',
  updatedAt: '',
};

export function summarizeAlgorithmSignalProposal(p: unknown): AlgorithmSignalProposalSummary {
  if (!isRecord(p)) return { ...EMPTY_PROPOSAL };
  return {
    proposalId: asString(p.proposal_id),
    tenantId: asString(p.tenant_id),
    algorithmResultId: asString(p.algorithm_result_id),
    algorithmId: asString(p.algorithm_id),
    algorithmVersion: asString(p.algorithm_version),
    executionRequestId: asString(p.execution_request_id),
    proposedSignalType: asString(p.proposed_signal_type),
    status: asString(p.status),
    score: asNumber(p.score),
    confidence: asNumber(p.confidence),
    severity: asString(p.severity),
    // Flexible JSON: pass through verbatim (already parsed by the gateway).
    proposalPayload: p.proposal_payload ?? {},
    rationale: p.rationale ?? {},
    sourceEventIds: asStringArray(p.source_event_ids),
    evidenceRefs: asStringArray(p.evidence_refs),
    correlationId: asString(p.correlation_id),
    createdBy: asString(p.created_by),
    reviewedBy: asString(p.reviewed_by),
    decisionNote: asString(p.decision_note),
    decidedAt: asString(p.decided_at),
    createdAt: asString(p.created_at),
    updatedAt: asString(p.updated_at),
  };
}

export interface AlgorithmSignalProposalSummaryView {
  tenantId: string;
  totalProposals: number;
  proposedCount: number;
  reviewedCount: number;
  rejectedCount: number;
  supersededCount: number;
  reviewedRatio: number;
  highCriticalUnreviewedCount: number;
  statusCounts: AlgorithmCountEntry[];
  severityCounts: AlgorithmCountEntry[];
  proposedSignalTypeCounts: AlgorithmCountEntry[];
  algorithmIdCounts: AlgorithmCountEntry[];
  reviewerCounts: AlgorithmCountEntry[];
}

const EMPTY_SUMMARY: AlgorithmSignalProposalSummaryView = {
  tenantId: '',
  totalProposals: 0,
  proposedCount: 0,
  reviewedCount: 0,
  rejectedCount: 0,
  supersededCount: 0,
  reviewedRatio: 0,
  highCriticalUnreviewedCount: 0,
  statusCounts: [],
  severityCounts: [],
  proposedSignalTypeCounts: [],
  algorithmIdCounts: [],
  reviewerCounts: [],
};

// Narrow the G115 summary payload into a display view. Numeric scalars collapse
// to 0; count maps become ordered entries (severity ordered by rank, the rest by
// count desc). Severity entries are normalized to {key,count} so all breakdowns
// share one shape, while keeping severity-rank ordering. Never throws.
export function summarizeAlgorithmSignalProposalSummary(s: unknown): AlgorithmSignalProposalSummaryView {
  if (!isRecord(s)) return { ...EMPTY_SUMMARY };
  return {
    tenantId: asString(s.tenant_id),
    totalProposals: asNumber(s.total_proposals),
    proposedCount: asNumber(s.proposed_count),
    reviewedCount: asNumber(s.reviewed_count),
    rejectedCount: asNumber(s.rejected_count),
    supersededCount: asNumber(s.superseded_count),
    reviewedRatio: asNumber(s.reviewed_ratio),
    highCriticalUnreviewedCount: asNumber(s.high_critical_unreviewed_count),
    statusCounts: algorithmCountEntries(s.status_counts),
    severityCounts: algorithmSeverityCountEntries(s.severity_counts).map((e) => ({ key: e.severity, count: e.count })),
    proposedSignalTypeCounts: algorithmCountEntries(s.proposed_signal_type_counts),
    algorithmIdCounts: algorithmCountEntries(s.algorithm_id_counts),
    reviewerCounts: algorithmCountEntries(s.reviewer_counts),
  };
}

function asBoolean(v: unknown): boolean {
  return v === true;
}

// Restrained algorithm signal materialization preflight-status colors (G119).
// eligible is deliberately NEUTRAL (slate), not a success/deploy green — the
// spec requires it not imply signal creation. duplicate_risk and blocked are both
// warning tones (amber / orange, kept distinct); invalid is an error red. Unknown
// future values fall back to neutral gray. Never reads as accepted/deployed.
const PREFLIGHT_STATUS_STYLES: Record<string, string> = {
  eligible: 'border-slate-200 bg-slate-50 text-slate-700',
  duplicate_risk: 'border-amber-200 bg-amber-50 text-amber-700',
  blocked: 'border-orange-200 bg-orange-50 text-orange-700',
  invalid: 'border-red-200 bg-red-50 text-red-700',
};

export function algorithmPreflightStatusStyle(status: string): string {
  return PREFLIGHT_STATUS_STYLES[status] ?? 'border-gray-200 bg-gray-50 text-gray-600';
}

export interface AlgorithmSignalMaterializationPreflightItemView {
  proposalId: string;
  algorithmResultId: string;
  algorithmId: string;
  executionRequestId: string;
  proposedSignalType: string;
  status: string;
  severity: string;
  confidence: number;
  preflightStatus: string;
  reasons: string[];
  duplicateSignalIds: string[];
  sourceEventIds: string[];
  wouldWrite: boolean;
  materializationPolicy: string;
}

const EMPTY_PREFLIGHT_ITEM: AlgorithmSignalMaterializationPreflightItemView = {
  proposalId: '',
  algorithmResultId: '',
  algorithmId: '',
  executionRequestId: '',
  proposedSignalType: '',
  status: '',
  severity: '',
  confidence: 0,
  preflightStatus: '',
  reasons: [],
  duplicateSignalIds: [],
  sourceEventIds: [],
  wouldWrite: false,
  materializationPolicy: '',
};

export function summarizeAlgorithmSignalMaterializationPreflightItem(i: unknown): AlgorithmSignalMaterializationPreflightItemView {
  if (!isRecord(i)) return { ...EMPTY_PREFLIGHT_ITEM };
  return {
    proposalId: asString(i.proposal_id),
    algorithmResultId: asString(i.algorithm_result_id),
    algorithmId: asString(i.algorithm_id),
    executionRequestId: asString(i.execution_request_id),
    proposedSignalType: asString(i.proposed_signal_type),
    status: asString(i.status),
    severity: asString(i.severity),
    confidence: asNumber(i.confidence),
    preflightStatus: asString(i.preflight_status),
    reasons: asStringArray(i.reasons),
    duplicateSignalIds: asStringArray(i.duplicate_signal_ids),
    sourceEventIds: asStringArray(i.source_event_ids),
    wouldWrite: asBoolean(i.would_write),
    materializationPolicy: asString(i.materialization_policy),
  };
}

export interface AlgorithmSignalMaterializationPreflightView {
  tenantId: string;
  policyVersion: string;
  totalProposals: number;
  eligibleCount: number;
  duplicateRiskCount: number;
  blockedCount: number;
  invalidCount: number;
  wouldWriteCount: number;
  reviewedRatio: number;
  minReviewedRatio: number;
  reviewCoverageSatisfied: boolean;
  highCriticalUnreviewedCount: number;
  globalBlockingReasons: AlgorithmCountEntry[];
  itemReasonCounts: AlgorithmCountEntry[];
  items: AlgorithmSignalMaterializationPreflightItemView[];
}

const EMPTY_PREFLIGHT: AlgorithmSignalMaterializationPreflightView = {
  tenantId: '',
  policyVersion: '',
  totalProposals: 0,
  eligibleCount: 0,
  duplicateRiskCount: 0,
  blockedCount: 0,
  invalidCount: 0,
  wouldWriteCount: 0,
  reviewedRatio: 0,
  minReviewedRatio: 0,
  reviewCoverageSatisfied: false,
  highCriticalUnreviewedCount: 0,
  globalBlockingReasons: [],
  itemReasonCounts: [],
  items: [],
};

// Narrow the G118 preflight payload into a read-only display view. Numeric
// scalars collapse to 0, booleans to false, id arrays to []. The two reason maps
// (global_blocking_reasons, item_reason_counts) become ordered entries sorted by
// count desc then token asc so the heaviest reason leads and unknown future
// tokens still render as plain text. Items map through the item summarizer.
// Never throws; never implies materialization.
export function summarizeAlgorithmSignalMaterializationPreflight(p: unknown): AlgorithmSignalMaterializationPreflightView {
  if (!isRecord(p)) return { ...EMPTY_PREFLIGHT };
  const rawItems = Array.isArray(p.items) ? p.items : [];
  return {
    tenantId: asString(p.tenant_id),
    policyVersion: asString(p.policy_version),
    totalProposals: asNumber(p.total_proposals),
    eligibleCount: asNumber(p.eligible_count),
    duplicateRiskCount: asNumber(p.duplicate_risk_count),
    blockedCount: asNumber(p.blocked_count),
    invalidCount: asNumber(p.invalid_count),
    wouldWriteCount: asNumber(p.would_write_count),
    reviewedRatio: asNumber(p.reviewed_ratio),
    minReviewedRatio: asNumber(p.min_reviewed_ratio),
    reviewCoverageSatisfied: asBoolean(p.review_coverage_satisfied),
    highCriticalUnreviewedCount: asNumber(p.high_critical_unreviewed_count),
    globalBlockingReasons: algorithmCountEntries(p.global_blocking_reasons),
    itemReasonCounts: algorithmCountEntries(p.item_reason_counts),
    items: rawItems.map(summarizeAlgorithmSignalMaterializationPreflightItem),
  };
}

// Find the preflight item for a given proposal id within a preflight view, or
// null when the proposal is outside the current preflight slice. Used by G123 to
// gate the materialize action on the matching item's status.
export function findPreflightItemByProposalId(
  view: AlgorithmSignalMaterializationPreflightView | null,
  proposalId: string,
): AlgorithmSignalMaterializationPreflightItemView | null {
  if (!view || !proposalId) return null;
  return view.items.find((it) => it.proposalId === proposalId) ?? null;
}

// Restrained algorithm signal materialization status colors (G121/G123).
// succeeded -> success emerald; duplicate -> warning amber; blocked -> warning
// orange (kept distinct from duplicate); failed -> error red; requested/running
// -> in-progress blue; superseded -> muted gray. Unknown values fall back to
// neutral gray.
const MATERIALIZATION_STATUS_STYLES: Record<string, string> = {
  requested: 'border-blue-200 bg-blue-50 text-blue-700',
  running: 'border-blue-200 bg-blue-50 text-blue-700',
  succeeded: 'border-emerald-200 bg-emerald-50 text-emerald-700',
  duplicate: 'border-amber-200 bg-amber-50 text-amber-700',
  blocked: 'border-orange-200 bg-orange-50 text-orange-700',
  failed: 'border-red-200 bg-red-50 text-red-700',
  superseded: 'border-gray-200 bg-gray-100 text-gray-500',
};

export function algorithmMaterializationStatusStyle(status: string): string {
  return MATERIALIZATION_STATUS_STYLES[status] ?? 'border-gray-200 bg-gray-50 text-gray-600';
}

export interface AlgorithmSignalMaterializationView {
  materializationId: string;
  tenantId: string;
  proposalId: string;
  algorithmResultId: string;
  executionRequestId: string;
  algorithmId: string;
  algorithmVersion: string;
  proposedSignalType: string;
  signalId: string;
  materializationStatus: string;
  materializationPolicyVersion: string;
  idempotencyKey: string;
  duplicateOfSignalId: string;
  requestedBy: string;
  requestedAt: string;
  startedAt: string;
  completedAt: string;
  failedAt: string;
  errorCode: string;
  errorMessage: string;
  requestMetadata: unknown;
  preflightSnapshot: unknown;
  signalPayloadPreview: unknown;
  createdAt: string;
  updatedAt: string;
}

const EMPTY_MATERIALIZATION: AlgorithmSignalMaterializationView = {
  materializationId: '',
  tenantId: '',
  proposalId: '',
  algorithmResultId: '',
  executionRequestId: '',
  algorithmId: '',
  algorithmVersion: '',
  proposedSignalType: '',
  signalId: '',
  materializationStatus: '',
  materializationPolicyVersion: '',
  idempotencyKey: '',
  duplicateOfSignalId: '',
  requestedBy: '',
  requestedAt: '',
  startedAt: '',
  completedAt: '',
  failedAt: '',
  errorCode: '',
  errorMessage: '',
  requestMetadata: {},
  preflightSnapshot: {},
  signalPayloadPreview: {},
  createdAt: '',
  updatedAt: '',
};

// Narrow a G121/G122 materialization DTO into a display view. Scalars collapse to
// 0/'', omitempty timestamps to '', and the three JSON fields pass through
// verbatim (already parsed). Never throws.
export function summarizeAlgorithmSignalMaterialization(m: unknown): AlgorithmSignalMaterializationView {
  if (!isRecord(m)) return { ...EMPTY_MATERIALIZATION };
  return {
    materializationId: asString(m.materialization_id),
    tenantId: asString(m.tenant_id),
    proposalId: asString(m.proposal_id),
    algorithmResultId: asString(m.algorithm_result_id),
    executionRequestId: asString(m.execution_request_id),
    algorithmId: asString(m.algorithm_id),
    algorithmVersion: asString(m.algorithm_version),
    proposedSignalType: asString(m.proposed_signal_type),
    signalId: asString(m.signal_id),
    materializationStatus: asString(m.materialization_status),
    materializationPolicyVersion: asString(m.materialization_policy_version),
    idempotencyKey: asString(m.idempotency_key),
    duplicateOfSignalId: asString(m.duplicate_of_signal_id),
    requestedBy: asString(m.requested_by),
    requestedAt: asString(m.requested_at),
    startedAt: asString(m.started_at),
    completedAt: asString(m.completed_at),
    failedAt: asString(m.failed_at),
    errorCode: asString(m.error_code),
    errorMessage: asString(m.error_message),
    requestMetadata: m.request_metadata ?? {},
    preflightSnapshot: m.preflight_snapshot ?? {},
    signalPayloadPreview: m.signal_payload_preview ?? {},
    createdAt: asString(m.created_at),
    updatedAt: asString(m.updated_at),
  };
}

// Pure G123 materialization-eligibility gate. Encodes the spec's enable/disable
// rules so the UI and tests share one definition. The backend stays authoritative
// — this only controls whether the button is enabled and what reason is shown.
// `hasRecordedMaterialization` should be true when the ledger already has a
// succeeded/duplicate row for the proposal; `globalBlockingActive` when the
// preflight reports any global blocking reason.
export interface MaterializationEligibilityInput {
  proposalStatus: string;
  preflightItem: AlgorithmSignalMaterializationPreflightItemView | null;
  preflightLoading: boolean;
  preflightFailed: boolean;
  globalBlockingActive: boolean;
  canMutate: boolean;
  mutationPending: boolean;
  hasRecordedMaterialization: boolean;
}

export interface MaterializationEligibility {
  canMaterialize: boolean;
  reason: string;
}

export function describeMaterializationEligibility(input: MaterializationEligibilityInput): MaterializationEligibility {
  const {
    proposalStatus,
    preflightItem,
    preflightLoading,
    preflightFailed,
    globalBlockingActive,
    canMutate,
    mutationPending,
    hasRecordedMaterialization,
  } = input;
  if (!canMutate) return { canMaterialize: false, reason: 'Requires operator or admin role.' };
  if (mutationPending) return { canMaterialize: false, reason: 'Materialization in progress…' };
  if (preflightLoading) return { canMaterialize: false, reason: 'Loading materialization preflight…' };
  if (preflightFailed) return { canMaterialize: false, reason: 'Materialization preflight unavailable.' };
  if (!preflightItem) return { canMaterialize: false, reason: 'No preflight row for this proposal.' };
  if (proposalStatus !== 'reviewed') {
    return { canMaterialize: false, reason: `Proposal is ${proposalStatus || 'unknown'}; must be reviewed.` };
  }
  if (hasRecordedMaterialization) return { canMaterialize: false, reason: 'Already materialized.' };
  if (globalBlockingActive) return { canMaterialize: false, reason: 'Global preflight blockers active.' };
  if (preflightItem.preflightStatus !== 'eligible') {
    return { canMaterialize: false, reason: `Preflight status: ${preflightItem.preflightStatus || 'unknown'}.` };
  }
  if (!preflightItem.wouldWrite) return { canMaterialize: false, reason: 'Preflight indicates no write.' };
  return { canMaterialize: true, reason: '' };
}

export interface AlgorithmExecutionSummaryView {
  executionRequest: AlgorithmExecutionRequestSummary;
  resultCount: number;
  severityCounts: AlgorithmSeverityCount[];
  maxScore: number;
  maxConfidence: number;
  topResults: AlgorithmResultSummary[];
}

export function summarizeAlgorithmExecutionSummary(s: unknown): AlgorithmExecutionSummaryView {
  const empty: AlgorithmExecutionSummaryView = {
    executionRequest: summarizeAlgorithmExecutionRequest(null),
    resultCount: 0,
    severityCounts: [],
    maxScore: 0,
    maxConfidence: 0,
    topResults: [],
  };
  if (!isRecord(s)) return empty;
  const rawTop = Array.isArray(s.top_results) ? s.top_results : [];
  return {
    executionRequest: summarizeAlgorithmExecutionRequest(s.execution_request),
    resultCount: asNumber(s.result_count),
    severityCounts: algorithmSeverityCountEntries(s.severity_counts),
    maxScore: asNumber(s.max_score),
    maxConfidence: asNumber(s.max_confidence),
    topResults: rawTop.map(summarizeAlgorithmResult),
  };
}
