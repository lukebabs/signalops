// Pure display helpers for Syncratic synthesized insights + deterministic
// context windows (G088 frontend). Insight/window JSON arrives already-parsed
// from the gateway (typed `unknown` on the flexible fields). Narrow with type
// guards only; never JSON.parse. Missing/malformed values collapse to 0 / '' and
// must never throw. These power read-only review surfaces — they never mutate
// insight lifecycle or write graph state.

export const SYNCRATIC_INSIGHT_STATUSES = [
  'active',
  'reviewed',
  'dismissed',
  'archived',
  'superseded',
] as const;

// The backend currently only emits "active"; archived/superseded are reserved
// for forward compatibility.
export const SYNCRATIC_CONTEXT_WINDOW_STATUSES = ['active', 'archived', 'superseded'] as const;

export const SYNCRATIC_SEVERITIES = ['info', 'low', 'medium', 'high', 'critical'] as const;

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null && !Array.isArray(v);
}

// Coerce a value to a finite number defensively: numbers pass, numeric strings
// parse, everything else (null/missing/objects) is 0.
function asNumber(v: unknown): number {
  if (typeof v === 'number' && Number.isFinite(v)) return v;
  if (typeof v === 'string' && v !== '') {
    const n = Number(v);
    if (Number.isFinite(n)) return n;
  }
  return 0;
}

// Coerce an unknown into string[], dropping non-strings so a malformed array
// never breaks a count or a rendered id list.
function asStringArray(v: unknown): string[] {
  return Array.isArray(v) ? v.filter((x): x is string => typeof x === 'string') : [];
}

function asString(v: unknown): string {
  return typeof v === 'string' ? v : '';
}

// Coerce to boolean defensively: strict booleans pass; the literal strings
// "true"/"false" parse; everything else (missing/objects) is false.
function asBoolean(v: unknown): boolean {
  if (typeof v === 'boolean') return v;
  if (typeof v === 'string') return v.toLowerCase() === 'true';
  return false;
}

// Restrained severity token colors, mirroring the Alert/Insight severity palette.
export const SYNCRATIC_SEVERITY_STYLES: Record<string, string> = {
  critical: 'text-red-700',
  high: 'text-orange-700',
  medium: 'text-amber-700',
  low: 'text-gray-700',
  info: 'text-gray-500',
};

export function syncraticSeverityStyle(severity: string): string {
  return SYNCRATIC_SEVERITY_STYLES[severity] ?? 'text-gray-600';
}

// Restrained insight-status token colors. active -> blue, reviewed -> green,
// dismissed/archived -> muted gray, superseded -> violet. Unknown future values
// fall back to gray.
export const SYNCRATIC_INSIGHT_STATUS_STYLES: Record<string, string> = {
  active: 'text-blue-700',
  reviewed: 'text-green-700',
  dismissed: 'text-gray-500',
  archived: 'text-gray-400',
  superseded: 'text-violet-700',
};

export function syncraticInsightStatusStyle(status: string): string {
  return SYNCRATIC_INSIGHT_STATUS_STYLES[status] ?? 'text-gray-600';
}

export interface SyncraticInsightSummary {
  insightId: string;
  contextWindowId: string;
  insightType: string;
  subjectSymbol: string;
  subjectType: string;
  subjectId: string;
  status: string;
  severity: string;
  confidence: number;
  title: string;
  summary: string;
  explanation: string;
  builderVersion: string;
  createdAt: string;
  updatedAt: string;
  supportingAlertIds: string[];
  supportingSignalIds: string[];
  supportingEventIds: string[];
  supportingArtifactIds: string[];
  relatedGraphProposalIds: string[];
  relatedLabelIds: string[];
  alertCount: number;
  signalCount: number;
  eventCount: number;
  artifactCount: number;
  graphProposalCount: number;
  labelCount: number;
}

const EMPTY_INSIGHT_SUMMARY: SyncraticInsightSummary = {
  insightId: '',
  contextWindowId: '',
  insightType: '',
  subjectSymbol: '',
  subjectType: '',
  subjectId: '',
  status: '',
  severity: '',
  confidence: 0,
  title: '',
  summary: '',
  explanation: '',
  builderVersion: '',
  createdAt: '',
  updatedAt: '',
  supportingAlertIds: [],
  supportingSignalIds: [],
  supportingEventIds: [],
  supportingArtifactIds: [],
  relatedGraphProposalIds: [],
  relatedLabelIds: [],
  alertCount: 0,
  signalCount: 0,
  eventCount: 0,
  artifactCount: 0,
  graphProposalCount: 0,
  labelCount: 0,
};

// Tolerantly summarize a Syncratic insight into display values + evidence counts.
// Never throws: a missing or non-object insight yields the empty summary.
export function summarizeSyncraticInsight(insight: unknown): SyncraticInsightSummary {
  if (!isRecord(insight)) return { ...EMPTY_INSIGHT_SUMMARY };
  const i = insight;
  const supportingAlertIds = asStringArray(i.supporting_alert_ids);
  const supportingSignalIds = asStringArray(i.supporting_signal_ids);
  const supportingEventIds = asStringArray(i.supporting_event_ids);
  const supportingArtifactIds = asStringArray(i.supporting_artifact_ids);
  const relatedGraphProposalIds = asStringArray(i.related_graph_proposal_ids);
  const relatedLabelIds = asStringArray(i.related_label_ids);
  return {
    insightId: asString(i.syncratic_insight_id),
    contextWindowId: asString(i.context_window_id),
    insightType: asString(i.insight_type),
    subjectSymbol: asString(i.subject_symbol),
    subjectType: asString(i.subject_type),
    subjectId: asString(i.subject_id),
    status: asString(i.status),
    severity: asString(i.severity),
    confidence: asNumber(i.confidence),
    title: asString(i.title),
    summary: asString(i.summary),
    explanation: asString(i.explanation),
    builderVersion: asString(i.builder_version),
    createdAt: asString(i.created_at),
    updatedAt: asString(i.updated_at),
    supportingAlertIds,
    supportingSignalIds,
    supportingEventIds,
    supportingArtifactIds,
    relatedGraphProposalIds,
    relatedLabelIds,
    alertCount: supportingAlertIds.length,
    signalCount: supportingSignalIds.length,
    eventCount: supportingEventIds.length,
    artifactCount: supportingArtifactIds.length,
    graphProposalCount: relatedGraphProposalIds.length,
    labelCount: relatedLabelIds.length,
  };
}

export interface SyncraticContextWindowSummary {
  contextWindowId: string;
  contextStrategy: string;
  contextBuilderVersion: string;
  windowStart: string;
  windowEnd: string;
  evidenceDigest: string;
  idempotencyKey: string;
  subjectSymbol: string;
  subjectType: string;
  subjectId: string;
  status: string;
  createdAt: string;
  updatedAt: string;
  signalTypes: string[];
  detectorIds: string[];
  eventIds: string[];
  signalIds: string[];
  alertIds: string[];
  artifactIds: string[];
  graphProposalIds: string[];
  labelIds: string[];
  eventCount: number;
  signalCount: number;
  alertCount: number;
  artifactCount: number;
  graphProposalCount: number;
  labelCount: number;
}

const EMPTY_CW_SUMMARY: SyncraticContextWindowSummary = {
  contextWindowId: '',
  contextStrategy: '',
  contextBuilderVersion: '',
  windowStart: '',
  windowEnd: '',
  evidenceDigest: '',
  idempotencyKey: '',
  subjectSymbol: '',
  subjectType: '',
  subjectId: '',
  status: '',
  createdAt: '',
  updatedAt: '',
  signalTypes: [],
  detectorIds: [],
  eventIds: [],
  signalIds: [],
  alertIds: [],
  artifactIds: [],
  graphProposalIds: [],
  labelIds: [],
  eventCount: 0,
  signalCount: 0,
  alertCount: 0,
  artifactCount: 0,
  graphProposalCount: 0,
  labelCount: 0,
};

// Tolerantly summarize a Syncratic context window into display values +
// evidence reference counts. Never throws.
export function summarizeSyncraticContextWindow(cw: unknown): SyncraticContextWindowSummary {
  if (!isRecord(cw)) return { ...EMPTY_CW_SUMMARY };
  const w = cw;
  const eventIds = asStringArray(w.event_ids);
  const signalIds = asStringArray(w.signal_ids);
  const alertIds = asStringArray(w.alert_ids);
  const artifactIds = asStringArray(w.artifact_ids);
  const graphProposalIds = asStringArray(w.graph_proposal_ids);
  const labelIds = asStringArray(w.label_ids);
  return {
    contextWindowId: asString(w.context_window_id),
    contextStrategy: asString(w.context_strategy),
    contextBuilderVersion: asString(w.context_builder_version),
    windowStart: asString(w.window_start),
    windowEnd: asString(w.window_end),
    evidenceDigest: asString(w.evidence_digest),
    idempotencyKey: asString(w.idempotency_key),
    subjectSymbol: asString(w.subject_symbol),
    subjectType: asString(w.subject_type),
    subjectId: asString(w.subject_id),
    status: asString(w.status),
    createdAt: asString(w.created_at),
    updatedAt: asString(w.updated_at),
    signalTypes: asStringArray(w.signal_types),
    detectorIds: asStringArray(w.detector_ids),
    eventIds,
    signalIds,
    alertIds,
    artifactIds,
    graphProposalIds,
    labelIds,
    eventCount: eventIds.length,
    signalCount: signalIds.length,
    alertCount: alertIds.length,
    artifactCount: artifactIds.length,
    graphProposalCount: graphProposalIds.length,
    labelCount: labelIds.length,
  };
}

// Materialization counter classification. `kind` distinguishes scan throughput,
// materialized outputs, and expected skips. skipped_below_threshold /
// skipped_unchanged / skipped_budget_cap are NORMAL outcomes (quiet assets,
// unchanged digests, budget caps) — the UI renders them as non-error, never red.
export type SyncraticMaterializationCounterKind = 'scanned' | 'materialized' | 'skipped';

export interface SyncraticMaterializationCounter {
  key: string;
  label: string;
  value: number;
  kind: SyncraticMaterializationCounterKind;
}

const MATERIALIZE_COUNTERS: { key: string; label: string; kind: SyncraticMaterializationCounterKind }[] = [
  { key: 'scanned_assets', label: 'Scanned assets', kind: 'scanned' },
  { key: 'candidate_windows', label: 'Candidate windows', kind: 'scanned' },
  { key: 'materialized_context_windows', label: 'Context windows', kind: 'materialized' },
  { key: 'materialized_insights', label: 'Insights', kind: 'materialized' },
  { key: 'skipped_below_threshold', label: 'Skipped · below threshold', kind: 'skipped' },
  { key: 'skipped_unchanged', label: 'Skipped · unchanged', kind: 'skipped' },
  { key: 'skipped_budget_cap', label: 'Skipped · budget cap', kind: 'skipped' },
];

// Build the ordered, display-formatted materialization counters. Tolerates any
// result shape (missing/non-object) and never throws; a missing result yields an
// empty list. Skip counters keep kind 'skipped' so the UI shows them as normal.
export function summarizeSyncraticMaterialization(result: unknown): SyncraticMaterializationCounter[] {
  if (!isRecord(result)) return [];
  return MATERIALIZE_COUNTERS.map(({ key, label, kind }) => ({
    key,
    label,
    value: asNumber((result as Record<string, unknown>)[key]),
    kind,
  }));
}

// Shorten a context window / insight id for compact table cells while keeping
// the full id available elsewhere. Returns the last segment after the final '_'
// when present, else the original string.
export function shortSyncraticId(id: string): string {
  const v = typeof id === 'string' ? id : '';
  const idx = v.lastIndexOf('_');
  return idx >= 0 && idx < v.length - 1 ? v.slice(idx + 1) : v;
}

// --- G090 Syncratic Ask enrichment helpers ---------------------------------
// Ask metadata lives under metrics.syncratic_ask. These helpers read it
// tolerantly (never throw) and never surface raw prompt text, bearer tokens,
// upstream bodies, or provider payloads — only the sanitized scalar fields the
// detail panel renders.

export interface SyncraticAskSummary {
  present: boolean;
  enabled: boolean;
  askQueryId: string;
  askStatus: string;
  promptBuilderVersion: string;
  promptDigest: string;
  contextWindowId: string;
  contextEvidenceDigest: string;
  requestScope: string;
  requestK: number;
  directReasoning: boolean;
  graphEnabled: boolean;
  keeEnabled: boolean;
  includedRecordDetails: boolean;
  promptBytes: number;
  latencyMs: number;
  startedAt: string;
  completedAt: string;
  responseConfidence: number;
  responseEvidenceCount: number;
  responseCitationCount: number;
}

const EMPTY_ASK_SUMMARY: SyncraticAskSummary = {
  present: false,
  enabled: false,
  askQueryId: '',
  askStatus: '',
  promptBuilderVersion: '',
  promptDigest: '',
  contextWindowId: '',
  contextEvidenceDigest: '',
  requestScope: '',
  requestK: 0,
  directReasoning: false,
  graphEnabled: false,
  keeEnabled: false,
  includedRecordDetails: false,
  promptBytes: 0,
  latencyMs: 0,
  startedAt: '',
  completedAt: '',
  responseConfidence: 0,
  responseEvidenceCount: 0,
  responseCitationCount: 0,
};

// Read metrics.syncratic_ask off an insight (or any object carrying metrics).
// Returns present:false (with the empty summary) when Ask metadata is absent —
// the signal that the insight is deterministic, not Ask-enriched.
export function summarizeSyncraticAsk(insight: unknown): SyncraticAskSummary {
  if (!isRecord(insight)) return { ...EMPTY_ASK_SUMMARY };
  const metrics = (insight as Record<string, unknown>).metrics;
  const ask = isRecord(metrics) ? (metrics as Record<string, unknown>).syncratic_ask : undefined;
  if (!isRecord(ask)) return { ...EMPTY_ASK_SUMMARY };
  const response = isRecord(ask.response) ? ask.response : {};
  return {
    present: true,
    enabled: asBoolean(ask.enabled),
    askQueryId: asString(ask.ask_query_id),
    askStatus: asString(ask.ask_status),
    promptBuilderVersion: asString(ask.prompt_builder_version),
    promptDigest: asString(ask.prompt_digest),
    contextWindowId: asString(ask.context_window_id),
    contextEvidenceDigest: asString(ask.context_evidence_digest),
    requestScope: asString(ask.request_scope),
    requestK: asNumber(ask.request_k),
    directReasoning: asBoolean(ask.direct_reasoning),
    graphEnabled: asBoolean(ask.graph_enabled),
    keeEnabled: asBoolean(ask.kee_enabled),
    includedRecordDetails: asBoolean(ask.included_record_details),
    promptBytes: asNumber(ask.prompt_bytes),
    latencyMs: asNumber(ask.latency_ms),
    startedAt: asString(ask.started_at),
    completedAt: asString(ask.completed_at),
    responseConfidence: asNumber(response.confidence),
    responseEvidenceCount: asNumber(response.evidence_count),
    responseCitationCount: asNumber(response.citation_count),
  };
}

export interface SyncraticAskRouteSummary {
  contextWindowId: string;
  syncraticInsightId: string;
  askQueryId: string;
  askStatus: string;
  promptDigest: string;
  updated: boolean;
  skippedReason: string;
  promptBuilderVersion: string;
}

const EMPTY_ASK_ROUTE: SyncraticAskRouteSummary = {
  contextWindowId: '',
  syncraticInsightId: '',
  askQueryId: '',
  askStatus: '',
  promptDigest: '',
  updated: false,
  skippedReason: '',
  promptBuilderVersion: '',
};

// Read the ask_result envelope from the POST .../ask route response. Never
// throws; a non-object result yields the empty route summary.
export function summarizeSyncraticAskRouteResult(result: unknown): SyncraticAskRouteSummary {
  if (!isRecord(result)) return { ...EMPTY_ASK_ROUTE };
  return {
    contextWindowId: asString(result.context_window_id),
    syncraticInsightId: asString(result.syncratic_insight_id),
    askQueryId: asString(result.ask_query_id),
    askStatus: asString(result.ask_status),
    promptDigest: asString(result.prompt_digest),
    updated: asBoolean(result.updated),
    skippedReason: asString(result.skipped_reason),
    promptBuilderVersion: asString(result.prompt_builder_version),
  };
}

// Data-quality subject-mismatch detection over the rendered text fields. The
// Ask prompt instructs the model to lead with "DATA QUALITY WARNING" when
// evidence cannot support the context subject, so matching that phrase is the
// primary signal; subject mismatch and "does not support" cover phrasing
// variants. Never inferred from subject symbol alone.
const DATA_QUALITY_PHRASES = ['data quality warning', 'subject mismatch', 'does not support'];

export function detectSyncraticDataQualityWarning(insight: unknown): boolean {
  if (!isRecord(insight)) return false;
  const text = `${asString(insight.title)} ${asString(insight.summary)} ${asString(insight.explanation)}`.toLowerCase();
  return DATA_QUALITY_PHRASES.some((phrase) => text.includes(phrase));
}

// Insight badge classification. `deterministic` = no Ask metadata. `ask_completed`
// = metrics.syncratic_ask.ask_status === completed. `ask_skipped` is transient —
// only set from the latest route response (there is no persisted skip state). A
// data-quality warning overrides all of the above so a blocked result never reads
// as a valid market thesis.
export type SyncraticAskBadge =
  | 'deterministic'
  | 'ask_completed'
  | 'ask_skipped'
  | 'data_quality';

export function classifySyncraticInsightBadge(
  insight: unknown,
  latestAskStatus?: string,
): SyncraticAskBadge {
  if (detectSyncraticDataQualityWarning(insight)) return 'data_quality';
  if (latestAskStatus === 'skipped') return 'ask_skipped';
  const ask = summarizeSyncraticAsk(insight);
  return ask.present && ask.askStatus === 'completed' ? 'ask_completed' : 'deterministic';
}

export const SYNCRATIC_ASK_BADGE_LABELS: Record<SyncraticAskBadge, string> = {
  deterministic: 'Deterministic',
  ask_completed: 'Ask completed',
  ask_skipped: 'Ask skipped',
  data_quality: 'Data Quality Warning',
};

// Restrained badge chip colors. data_quality leads with amber/red so it is
// visually distinct from a valid ask_completed (emerald) or deterministic (gray).
export const SYNCRATIC_ASK_BADGE_STYLES: Record<SyncraticAskBadge, string> = {
  deterministic: 'border-gray-200 bg-gray-50 text-gray-600',
  ask_completed: 'border-emerald-200 bg-emerald-50 text-emerald-700',
  ask_skipped: 'border-amber-200 bg-amber-50 text-amber-700',
  data_quality: 'border-red-200 bg-red-50 text-red-700',
};

// Sanitized Ask action error mapping. The gateway already strips upstream Syncratic
// bodies before responding (it returns fixed messages for syncratic_ask_failed),
// and these strings carry no prompt text, bearer tokens, or provider payloads.
// `empty_context_window` is handled defensively: it is emitted by context-window
// create/materialize purity filtering rather than the Ask route itself, but the
// operator-facing guidance is the same.
export type SyncraticAskErrorKind =
  | 'network'
  | 'auth'
  | 'empty'
  | 'invalid'
  | 'not_found'
  | 'unavailable'
  | 'failed'
  | 'unknown';

export function classifySyncraticAskError(status: number, code: string): SyncraticAskErrorKind {
  if (status === 0) return 'network';
  if (status === 401 || code === 'unauthorized') return 'auth';
  if (code === 'empty_context_window') return 'empty';
  if (code === 'syncratic_ask_invalid') return 'invalid';
  if (code === 'context_window_not_found') return 'not_found';
  if (code === 'syncratic_ask_unavailable') return 'unavailable';
  if (status === 502 || status === 500 || code === 'syncratic_ask_failed') return 'failed';
  return 'unknown';
}

export const SYNCRATIC_ASK_ERROR_MESSAGES: Record<SyncraticAskErrorKind, string> = {
  network: 'Gateway unreachable.',
  auth: 'Authentication required — please sign in again.',
  empty:
    'No pure supporting evidence exists for this context subject. Review signal/entity mapping or rematerialize after evidence is corrected.',
  invalid: 'Ask request validation failed. Adjust inputs and retry.',
  not_found: 'Syncratic context window not found. It may have been removed.',
  unavailable: 'Syncratic Ask is not configured on this gateway.',
  failed: 'Syncratic Ask failed. Upstream details are not exposed; retry or review gateway logs.',
  unknown: 'Syncratic Ask failed unexpectedly. Retry or review gateway logs.',
};

// Resolve a sanitized Ask error message from a thrown error. Accepts the ApiError
// shape ({ status, code }) or anything else (falls back to the unknown message).
export function messageForSyncraticAskError(error: unknown): string {
  if (isRecord(error) && typeof error.status === 'number' && 'code' in error) {
    return SYNCRATIC_ASK_ERROR_MESSAGES[
      classifySyncraticAskError(error.status, asString(error.code))
    ];
  }
  return SYNCRATIC_ASK_ERROR_MESSAGES.unknown;
}
