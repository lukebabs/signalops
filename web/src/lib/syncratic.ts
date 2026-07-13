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
