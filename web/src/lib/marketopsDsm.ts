// Pure parsing helpers for the MarketOps DSM workbench (G076 frontend).
//
// The DSM-specific signal data lives in SignalRecord fields typed `unknown`
// (entities, supporting_metrics, semantic_evidence, graph_targets,
// recommendation). The gateway deserializes the whole JSON response, so these
// fields arrive as already-parsed JS values (objects/arrays) — NOT strings.
// Therefore: narrow with type guards only; never JSON.parse them (that would
// throw on an already-parsed array). Missing/malformed values render as `-` /
// empty and must never throw.
import type { SignalRecord, MarketOpsDSMGraphProposal } from '../types';

export const MARKETOPS_DSM_DETECTOR_ID = 'marketops.dsm.taxonomy_v1';
export const MARKETOPS_DSM_USE_CASE = 'daily_market_surveillance';

export const MARKETOPS_DSM_SIGNAL_TYPES = [
  'marketops.dsm.volatility_expansion',
  'marketops.dsm.price_quality_exception',
  'marketops.dsm.accumulation',
  'marketops.dsm.divergence',
  'marketops.dsm.hedging_pressure',
  'marketops.dsm.speculative_call_pressure',
  'marketops.dsm.speculative_put_pressure',
  'marketops.dsm.pinning_risk',
] as const;

export type DsmFamily = 'equity' | 'option' | 'quality' | 'unknown';

const DSM_PREFIX = 'marketops.dsm.';

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null && !Array.isArray(v);
}

export function dsmShortType(signalType: string): string {
  if (typeof signalType !== 'string') return '';
  return signalType.startsWith(DSM_PREFIX) ? signalType.slice(DSM_PREFIX.length) : signalType;
}

const FAMILY_BY_TYPE: Record<string, DsmFamily> = {
  'marketops.dsm.price_quality_exception': 'quality',
  'marketops.dsm.hedging_pressure': 'option',
  'marketops.dsm.speculative_call_pressure': 'option',
  'marketops.dsm.speculative_put_pressure': 'option',
  'marketops.dsm.pinning_risk': 'option',
  'marketops.dsm.volatility_expansion': 'equity',
  'marketops.dsm.accumulation': 'equity',
  'marketops.dsm.divergence': 'equity',
};

export function dsmFamily(signalType: string): DsmFamily {
  return FAMILY_BY_TYPE[signalType] ?? 'unknown';
}

export interface DsmArtifactProposal {
  artifact_id?: string;
  artifact_type?: string;
  signal_type?: string;
  source_event_id?: string;
  subject?: { type?: string; id?: string; symbol?: string };
  severity?: string;
  confidence?: number;
  summary?: string;
  features?: Record<string, unknown>;
  quality_issues?: string[];
}

// Pull the nested artifact proposal out of semantic_evidence[0]. The detector
// nests it under `artifact`; fall back to the element itself if a future shape
// inlines it. Returns null for any malformed/missing shape.
export function getArtifactProposal(signal: SignalRecord): DsmArtifactProposal | null {
  const se = signal.semantic_evidence;
  if (!Array.isArray(se) || se.length === 0) return null;
  const first = se[0];
  if (!isRecord(first)) return null;
  const artifact = isRecord(first.artifact) ? first.artifact : first;
  if (!isRecord(artifact)) return null;
  const proposal: DsmArtifactProposal = {};
  if (typeof artifact.artifact_id === 'string') proposal.artifact_id = artifact.artifact_id;
  if (typeof artifact.artifact_type === 'string') proposal.artifact_type = artifact.artifact_type;
  if (typeof artifact.signal_type === 'string') proposal.signal_type = artifact.signal_type;
  if (typeof artifact.source_event_id === 'string') proposal.source_event_id = artifact.source_event_id;
  if (typeof artifact.severity === 'string') proposal.severity = artifact.severity;
  if (typeof artifact.confidence === 'number') proposal.confidence = artifact.confidence;
  if (typeof artifact.summary === 'string') proposal.summary = artifact.summary;
  if (isRecord(artifact.features)) proposal.features = artifact.features;
  if (Array.isArray(artifact.quality_issues)) proposal.quality_issues = artifact.quality_issues as string[];
  if (isRecord(artifact.subject)) {
    const s = artifact.subject;
    proposal.subject = {};
    if (typeof s.type === 'string') proposal.subject.type = s.type;
    if (typeof s.id === 'string') proposal.subject.id = s.id;
    if (typeof s.symbol === 'string') proposal.subject.symbol = s.symbol;
  }
  return proposal;
}

export function getArtifactId(signal: SignalRecord): string | null {
  const proposal = getArtifactProposal(signal);
  if (proposal?.artifact_id) return proposal.artifact_id;
  // Fallbacks: semantic_evidence[0].artifact_id, then recommendation.artifact_ids[0].
  const se = signal.semantic_evidence;
  if (Array.isArray(se) && isRecord(se[0]) && typeof se[0].artifact_id === 'string') {
    return se[0].artifact_id;
  }
  const rec = signal.recommendation;
  if (
    isRecord(rec) &&
    Array.isArray(rec.artifact_ids) &&
    typeof rec.artifact_ids[0] === 'string'
  ) {
    return rec.artifact_ids[0];
  }
  return null;
}

// Ticker resolution: prefer entities[0].external_id; fall back to the artifact
// subject symbol; finally '-'.
export function getTicker(signal: SignalRecord): string {
  const entities = signal.entities;
  if (Array.isArray(entities)) {
    for (const e of entities) {
      if (isRecord(e) && typeof e.external_id === 'string' && e.external_id) return e.external_id;
    }
  }
  const symbol = getArtifactProposal(signal)?.subject?.symbol;
  return typeof symbol === 'string' && symbol ? symbol : '-';
}

export function getMetric(signal: SignalRecord, key: string): string | number | null {
  const metrics = signal.supporting_metrics;
  if (!isRecord(metrics)) return null;
  const v = metrics[key];
  if (typeof v === 'number' || typeof v === 'string') return v;
  return null;
}

export interface DsmGraphCounts {
  nodes: number;
  relationships: number;
}

// Count graph target candidates by their `type` discriminator
// ("node_candidate" / "relationship_candidate").
export function graphTargetCounts(signal: SignalRecord): DsmGraphCounts {
  const targets = signal.graph_targets;
  if (!Array.isArray(targets)) return { nodes: 0, relationships: 0 };
  let nodes = 0;
  let relationships = 0;
  for (const t of targets) {
    if (!isRecord(t)) continue;
    if (t.type === 'node_candidate') nodes++;
    else if (t.type === 'relationship_candidate') relationships++;
  }
  return { nodes, relationships };
}

// Total candidate count for compact table cells.
export function countGraphTargets(signal: SignalRecord): number {
  const { nodes, relationships } = graphTargetCounts(signal);
  return nodes + relationships;
}

// True when this signal's id is in the set of ids with alert/insight coverage.
export function hasLifecycleMatch(signal: SignalRecord, signalIds: Set<string>): boolean {
  return signalIds.has(signal.signal_id);
}

// G079 graph proposal ledger summaries (read-only). The persisted proposal
// records come straight from the gateway typed as MarketOpsDSMGraphProposal,
// but every value is still narrowed defensively so a malformed/forward-shaped
// payload renders blanks instead of throwing. Backend nil slices (e.g. labels)
// can arrive as JSON `null`, so never assume array-ness without a guard.

export interface GraphProposalSummary {
  total: number;
  node: number;
  relationship: number;
  byStatus: Record<string, number>;
}

// Tally persisted proposals into total / node / relationship counts plus a
// per-status histogram. `byStatus` only carries statuses that actually occur.
export function summarizeGraphProposals(proposals: unknown): GraphProposalSummary {
  const summary: GraphProposalSummary = { total: 0, node: 0, relationship: 0, byStatus: {} };
  if (!Array.isArray(proposals)) return summary;
  for (const p of proposals) {
    if (!isRecord(p)) continue;
    summary.total++;
    if (p.candidate_type === 'node_candidate') summary.node++;
    else if (p.candidate_type === 'relationship_candidate') summary.relationship++;
    const status = typeof p.status === 'string' && p.status ? p.status : 'unknown';
    summary.byStatus[status] = (summary.byStatus[status] ?? 0) + 1;
  }
  return summary;
}

// Render a proposal's labels as a comma-separated string, tolerating null/nil
// slices and non-array shapes. Returns 'none' for the empty case so the ledger
// shows an explicit absence rather than a blank cell.
export function formatGraphProposalLabels(labels: unknown): string {
  if (!Array.isArray(labels) || labels.length === 0) return 'none';
  return labels.filter((l): l is string => typeof l === 'string' && l.length > 0).join(', ') || 'none';
}

// Compact one-line subject for a ledger row: node candidates show their node id;
// relationship candidates show `from —relationship→ to`. Falls back to '—' when
// the relevant fields are missing.
export function graphProposalSubjectLine(p: MarketOpsDSMGraphProposal): string {
  if (!isRecord(p)) return '—';
  if (p.candidate_type === 'relationship_candidate') {
    const from = typeof p.from_node === 'string' ? p.from_node : '';
    const rel = typeof p.relationship === 'string' ? p.relationship : '';
    const to = typeof p.to_node === 'string' ? p.to_node : '';
    if (!from && !rel && !to) return '—';
    return `${from} —${rel}→ ${to}`.trim();
  }
  const node = typeof p.node_id === 'string' ? p.node_id : '';
  return node || '—';
}

// True when the proposal has any decision metadata worth showing in the detail
// block (only present after an operator decision on the backend side).
export function graphProposalHasDecision(p: MarketOpsDSMGraphProposal): boolean {
  if (!isRecord(p)) return false;
  return !!(p.reviewed_by || p.decision_note || p.decided_at);
}
