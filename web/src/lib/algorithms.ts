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
