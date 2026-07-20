import { useState } from 'react';
import { useNavigate, useSearch } from '@tanstack/react-router';
import { useQueryClient } from '@tanstack/react-query';
import {
  Telescope,
  RotateCw,
  Eraser,
  ArrowLeft,
  ArrowUpRight,
  ArrowDownRight,
  ArrowRightLeft,
  ChevronDown,
  ChevronRight,
  AlertTriangle,
} from 'lucide-react';
import {
  useMarketOpsOpportunities,
  useMarketOpsOpportunity,
  useMarketOpsHypothesisEvaluations,
  useMarketOpsHypothesis,
  useMarketOpsEvidence,
  useMarketOpsMarketStateLineage,
} from '../api/queries';
import { isApiError } from '../api/client';
import { LoadingState, ErrorState, EmptyState } from '../components/States';
import { JsonViewer } from '../components/JsonViewer';
import { CopyButton } from '../components/CopyButton';
import { formatUtc } from '../lib/format';
import { marketOpsOptionsDateOnly } from '../lib/marketopsOptions';
import {
  summarizeMarketOpsOpportunity,
  summarizeMarketOpsHypothesisEvaluation,
  formatScore,
  aggregateOpportunityRejectionReasons,
  opportunityLifecycleStyle,
  directionLabel,
  directionTone,
  type MarketOpsOpportunityView,
  type MarketOpsOpportunityContribution,
  type MarketOpsHypothesisEvaluationView,
} from '../lib/marketopsOpportunities';
import { useTenant } from '../auth/session';
import type {
  MarketOpsOpportunityFilter,
  MarketOpsOpportunityLifecycle,
  MarketOpsOpportunityDirection,
} from '../types';

// G139 MarketOps Opportunities workbench (read-only). Turns compatible hypothesis
// evaluations into a fast triage queue with a master/detail inspection flow.
// Contributions / conflicts are read from opportunity_payload; per-contribution
// reason_codes and evidence/hypothesis/lineage detail are lazy supporting reads.
// No review, trade, materialization, or build mutation.

const LIFECYCLE_STATUSES = ['', 'emerging', 'active', 'strengthening', 'weakening', 'invalidated', 'resolved', 'expired'];
const DIRECTIONS = ['', 'upside', 'downside', 'non_directional'];
const RESEARCH_ONLY = ['', 'true', 'false'];
const LIMITS = [25, 50, 100, 200];
const inputCls = 'rounded border border-gray-300 px-2 py-1 text-sm';

export function MarketOpsOpportunitiesRoute() {
  const TENANT_ID = useTenant();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const search = useSearch({ strict: false }) as { opportunity_id?: string };
  const selectedId = search.opportunity_id || null;

  // Filter state. Only ?opportunity_id= is URL-persisted (refresh/back/forward).
  const [symbol, setSymbol] = useState('');
  const [lifecycle, setLifecycle] = useState('');
  const [direction, setDirection] = useState('');
  const [horizon, setHorizon] = useState('');
  const [sessionStart, setSessionStart] = useState('');
  const [sessionEnd, setSessionEnd] = useState('');
  const [researchOnly, setResearchOnly] = useState('');
  const [limit, setLimit] = useState(50);

  const filter: MarketOpsOpportunityFilter = {
    tenant_id: TENANT_ID,
    symbol: symbol.trim().toUpperCase() || undefined,
    direction: (direction || undefined) as MarketOpsOpportunityDirection | undefined,
    horizon: horizon.trim() || undefined,
    lifecycle_status: (lifecycle || undefined) as MarketOpsOpportunityLifecycle | undefined,
    research_only: researchOnly === '' ? undefined : researchOnly === 'true',
    session_start: sessionStart || undefined,
    session_end: sessionEnd || undefined,
    limit,
  };

  const listQ = useMarketOpsOpportunities(filter);
  const opportunities = (listQ.data?.opportunities ?? [])
    .map(summarizeMarketOpsOpportunity)
    .slice()
    .sort((a, b) => b.opportunityScore - a.opportunityScore || b.lastEvaluatedDate.localeCompare(a.lastEvaluatedDate));
  const listEmpty = !listQ.isLoading && !listQ.isError && opportunities.length === 0;

  // Empty-queue diagnostics: one bounded hypothesis-evaluations read using the
  // same symbol/date scope. Enabled only when the queue is empty.
  const dxQ = useMarketOpsHypothesisEvaluations(
    {
      tenant_id: TENANT_ID,
      symbol: filter.symbol,
      session_start: filter.session_start,
      session_end: filter.session_end,
      limit: 200,
    },
    listEmpty,
  );

  function selectOpportunity(id: string | null) {
    void navigate({ to: '/marketops/opportunities', search: id ? { opportunity_id: id } : {} });
  }
  function refresh() {
    void queryClient.invalidateQueries({ queryKey: ['marketops-opportunities'] });
  }
  function clearFilters() {
    setSymbol('');
    setLifecycle('');
    setDirection('');
    setHorizon('');
    setSessionStart('');
    setSessionEnd('');
    setResearchOnly('');
    setLimit(50);
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between gap-2">
        <div>
          <h1 className="flex items-center gap-1 text-lg font-semibold">
            <Telescope size={18} className="text-brand-700" /> Opportunities
          </h1>
          <p className="text-xs text-gray-500">
            Triage queue · {opportunities.length} opportunity{opportunities.length === 1 ? '' : 'ies'} · tenant {TENANT_ID}
          </p>
        </div>
        <div className="flex items-center gap-1">
          <button type="button" onClick={refresh} title="Refresh" aria-label="Refresh opportunities" className={`${inputCls} inline-flex items-center gap-1 bg-white`}>
            <RotateCw size={14} className={listQ.isFetching ? 'animate-spin' : ''} />
          </button>
          <button type="button" onClick={clearFilters} title="Clear filters" aria-label="Clear filters" className={`${inputCls} inline-flex items-center gap-1 bg-white`}>
            <Eraser size={14} />
          </button>
        </div>
      </div>

      {/* Filter toolbar — no per-filter cards; controls wrap cleanly. */}
      <div className="flex flex-wrap items-center gap-2">
        <input value={symbol} onChange={(e) => setSymbol(e.target.value)} className={inputCls} aria-label="Filter by symbol" placeholder="symbol" />
        <select value={lifecycle} onChange={(e) => setLifecycle(e.target.value)} className={inputCls} aria-label="Filter by lifecycle status">
          {LIFECYCLE_STATUSES.map((s) => (<option key={s} value={s}>{s || 'any lifecycle'}</option>))}
        </select>
        <div className="inline-flex overflow-hidden rounded border border-gray-300 text-sm">
          {DIRECTIONS.map((d) => (
            <button
              key={d || 'all'}
              type="button"
              onClick={() => setDirection(d)}
              className={`px-2 py-1 ${direction === d ? 'bg-brand-600 text-white' : 'bg-white text-gray-600 hover:bg-gray-50'}`}
              aria-label={d ? `direction ${d}` : 'any direction'}
            >
              {d ? directionLabel(d) : 'any'}
            </button>
          ))}
        </div>
        <input value={horizon} onChange={(e) => setHorizon(e.target.value)} className={inputCls} aria-label="Filter by horizon" placeholder="horizon" />
        <input type="date" value={sessionStart} onChange={(e) => setSessionStart(e.target.value)} className={inputCls} aria-label="Session start" />
        <input type="date" value={sessionEnd} onChange={(e) => setSessionEnd(e.target.value)} className={inputCls} aria-label="Session end" />
        <select value={researchOnly} onChange={(e) => setResearchOnly(e.target.value)} className={inputCls} aria-label="Filter by research-only">
          {RESEARCH_ONLY.map((r) => (<option key={r} value={r}>{r === '' ? 'all' : r === 'true' ? 'research-only' : 'operational'}</option>))}
        </select>
        <select value={limit} onChange={(e) => setLimit(Number(e.target.value))} className={inputCls} aria-label="Page limit">
          {LIMITS.map((n) => (<option key={n} value={n}>{n}</option>))}
        </select>
      </div>

      <div className="flex flex-col gap-3 lg:flex-row">
        {/* Queue pane (hidden on mobile once a detail is open). */}
        <div className={`${selectedId ? 'hidden lg:block' : ''} lg:w-2/5 lg:min-w-[360px]`}>
          <div className="space-y-2">
            {listQ.isLoading && !listQ.data ? (
              <QueueSkeleton />
            ) : listQ.isError ? (
              <div className="rounded border border-red-200 bg-red-50 p-3 text-sm text-red-800">
                <div>Opportunity queue unavailable{isApiError(listQ.error) ? `: ${listQ.error.message}` : ''}.</div>
                <button type="button" onClick={refresh} className="mt-1 inline-flex items-center gap-1 rounded border border-red-300 bg-white px-2 py-1 text-xs text-red-700 hover:bg-red-50">
                  <RotateCw size={14} /> Retry
                </button>
              </div>
            ) : opportunities.length ? (
              opportunities.map((o) => (
                <OpportunityQueueRow key={o.opportunityId} opportunity={o} selected={selectedId === o.opportunityId} onSelect={() => selectOpportunity(o.opportunityId)} />
              ))
            ) : (
              <EmptyQueueDiagnostics
                loading={dxQ.isLoading}
                failed={dxQ.isError}
                error={dxQ.error}
                aggregation={aggregateOpportunityRejectionReasons((dxQ.data?.hypothesis_evaluations ?? []).map(summarizeMarketOpsHypothesisEvaluation))}
                onClear={clearFilters}
              />
            )}
          </div>
        </div>

        {/* Detail pane (hidden on mobile when nothing is selected). */}
        <div className={`${selectedId ? '' : 'hidden lg:block'} flex-1`}>
          {selectedId ? (
            <OpportunityDetail opportunityId={selectedId} tenantId={TENANT_ID} onBack={() => selectOpportunity(null)} />
          ) : (
            <div className="rounded border border-gray-200 bg-white p-3">
              <EmptyState message="Select an opportunity to inspect its evidence." />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function QueueSkeleton() {
  return (
    <>
      {Array.from({ length: 6 }).map((_, i) => (
        <div key={i} className="h-20 animate-pulse rounded border border-gray-200 bg-gray-100" />
      ))}
    </>
  );
}

function LifecycleBadge({ status }: { status: string }) {
  return (
    <span className={`inline-flex shrink-0 items-center whitespace-nowrap rounded border px-1.5 py-0.5 text-[11px] font-medium ${opportunityLifecycleStyle(status)}`}>
      {status || '—'}
    </span>
  );
}

function DirectionBadge({ direction }: { direction: string }) {
  const Icon = direction === 'upside' ? ArrowUpRight : direction === 'downside' ? ArrowDownRight : ArrowRightLeft;
  return (
    <span className={`inline-flex items-center gap-0.5 text-[11px] font-medium ${directionTone(direction)}`}>
      <Icon size={12} /> {directionLabel(direction)}
    </span>
  );
}

function OpportunityQueueRow({
  opportunity: o,
  selected,
  onSelect,
}: {
  opportunity: MarketOpsOpportunityView;
  selected: boolean;
  onSelect: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onSelect}
      className={`w-full rounded border p-2 text-left align-top transition-colors ${selected ? 'border-brand-400 bg-brand-50' : 'border-gray-200 bg-white hover:bg-gray-50'}`}
    >
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <span className="font-mono text-sm font-semibold text-gray-900">{o.symbol || '—'}</span>
          <DirectionBadge direction={o.direction} />
        </div>
        <div className="flex items-center gap-1">
          {o.researchOnly ? (
            <span className="inline-flex items-center rounded border border-amber-200 bg-amber-50 px-1.5 py-0.5 text-[11px] font-medium text-amber-700">Research</span>
          ) : null}
          <LifecycleBadge status={o.lifecycleStatus} />
        </div>
      </div>
      <p className="mt-1 line-clamp-2 text-xs text-gray-600">{o.summary || '—'}</p>
      <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-0.5 text-[11px] text-gray-500">
        <span>Score <strong className="text-gray-800">{formatScore(o.opportunityScore)}</strong></span>
        <span>Conf <strong className="text-gray-800">{formatScore(o.confidenceScore)}</strong></span>
        <span>{o.hypothesisEvaluationIds.length} hyp · {o.hypothesisFamilies.length} domain</span>
        {o.conflictScore > 0 ? (
          <span className="inline-flex items-center gap-0.5 font-medium text-amber-700"><AlertTriangle size={11} /> conflict {formatScore(o.conflictScore)}</span>
        ) : null}
        <span>{o.lastEvaluatedDate ? formatUtc(o.lastEvaluatedDate) : '—'}</span>
        <span>{o.horizon || '—'}</span>
      </div>
    </button>
  );
}

function OpportunityDetail({ opportunityId, tenantId, onBack }: { opportunityId: string; tenantId: string; onBack: () => void }) {
  const detailQ = useMarketOpsOpportunity(opportunityId, tenantId);
  const view = detailQ.data ? summarizeMarketOpsOpportunity(detailQ.data.opportunity) : null;
  // Linked hypothesis-evaluations scoped to this opportunity's symbol + session:
  // powers per-contribution reason_codes. One bounded request.
  const sessionDate = view ? marketOpsOptionsDateOnly(view.lastEvaluatedDate) : '';
  const evalsQ = useMarketOpsHypothesisEvaluations(
    { tenant_id: tenantId, symbol: view?.symbol, session_start: sessionDate, session_end: sessionDate, limit: 200 },
    !!view && !!view?.symbol,
  );
  const evalsById = new Map<string, MarketOpsHypothesisEvaluationView>();
  for (const e of (evalsQ.data?.hypothesis_evaluations ?? []).map(summarizeMarketOpsHypothesisEvaluation)) {
    evalsById.set(e.evaluationId, e);
  }

  return (
    <div className="rounded border border-gray-200 bg-white p-3">
      <div className="mb-2 flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <button type="button" onClick={onBack} className="inline-flex items-center gap-1 rounded border border-gray-300 bg-white px-2 py-1 text-xs text-gray-600 hover:bg-gray-50 lg:hidden" aria-label="Back to queue">
            <ArrowLeft size={14} /> Back
          </button>
          <div className="text-xs font-semibold text-gray-700">Opportunity Detail</div>
          {detailQ.isLoading && !detailQ.data ? <span className="text-[11px] text-gray-400">loading…</span> : null}
        </div>
        {view ? <CopyButton value={view.opportunityId} /> : null}
      </div>

      {detailQ.isError ? (
        <div className="rounded border border-amber-200 bg-amber-50 p-2 text-xs text-amber-700">
          Detail unavailable{isApiError(detailQ.error) ? `: ${detailQ.error.message}` : ''}; the queue selection is retained.
        </div>
      ) : !view ? (
        <LoadingState label="Loading opportunity..." />
      ) : (
        <div className="space-y-3">
          {/* 1. Decision snapshot */}
          <div className="rounded border border-gray-200 bg-gray-50 p-2">
            <div className="flex flex-wrap items-center gap-2">
              <span className="font-mono text-base font-semibold text-gray-900">{view.symbol || '—'}</span>
              <DirectionBadge direction={view.direction} />
              <LifecycleBadge status={view.lifecycleStatus} />
              {view.researchOnly ? (
                <span className="inline-flex items-center rounded border border-amber-200 bg-amber-50 px-1.5 py-0.5 text-[11px] font-medium text-amber-700">Research-only</span>
              ) : (
                <span className="inline-flex items-center rounded border border-emerald-200 bg-emerald-50 px-1.5 py-0.5 text-[11px] font-medium text-emerald-700">Operational</span>
              )}
            </div>
            <div className="mt-1 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-gray-700">
              <span>Horizon <strong>{view.horizon || '—'}</strong></span>
              <span>Opportunity <strong>{formatScore(view.opportunityScore)}</strong></span>
              <span>Confidence <strong>{formatScore(view.confidenceScore)}</strong></span>
              <span>Domain diversity <strong>{formatScore(view.domainDiversityScore)}</strong></span>
              <span>Conflict <strong>{formatScore(view.conflictScore)}</strong></span>
            </div>
          </div>

          {/* 2. Why now */}
          <Section title="Why now">
            <p className="text-xs text-gray-700">{view.summary || '—'}</p>
            <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-[11px] text-gray-500">
              <span>{view.contributions.length} contribution{view.contributions.length === 1 ? '' : 's'}</span>
              <span>{view.hypothesisFamilies.length} independent domain{view.hypothesisFamilies.length === 1 ? '' : 's'}</span>
              <span>Last evaluated {view.lastEvaluatedDate ? formatUtc(view.lastEvaluatedDate) : '—'}</span>
              <span>Opened {view.openedSessionDate ? formatUtc(view.openedSessionDate) : '—'}</span>
            </div>
          </Section>

          {/* 3. Contributions */}
          <Section title={`Contributions (${view.contributions.length})`}>
            {view.contributions.length ? (
              <div className="space-y-1">
                {view.contributions.map((c) => (
                  <ContributionRow
                    key={c.evaluationId || `${c.hypothesisKey}:${c.hypothesisVersion}`}
                    contribution={c}
                    tenantId={tenantId}
                    reasonCodes={evalsById.get(c.evaluationId)?.reasonCodes ?? []}
                    marketStateId={evalsById.get(c.evaluationId)?.marketStateId ?? ''}
                  />
                ))}
              </div>
            ) : (
              <p className="text-[11px] text-gray-400">No contribution detail embedded in the opportunity payload.</p>
            )}
          </Section>

          {/* 4. Conflicts and limits */}
          <Section title="Conflicts and limits">
            <div className="grid grid-cols-1 gap-1 text-xs text-gray-700 md:grid-cols-2">
              <IdList label="Opposing evaluations" ids={view.conflictingEvaluationIds} tone="amber" />
              <IdList label="Invalidating evidence" ids={view.invalidatingEvidenceIds} tone="red" />
              <IdList label="Overlap-suppressed evaluations" ids={view.overlapSuppressedEvaluationIds} tone="gray" />
              <div>Conflict score <strong>{formatScore(view.conflictScore)}</strong></div>
            </div>
          </Section>

          {/* 5. Evidence */}
          <Section title={`Evidence (${view.supportingEvidenceIds.length})`}>
            <EvidenceList evidenceIds={view.supportingEvidenceIds} />
          </Section>

          {/* 6. Audit */}
          <Section title="Audit">
            <div className="grid grid-cols-2 gap-1 text-[11px] text-gray-600">
              <div>Version <strong className="text-gray-800">{view.version || '—'}</strong></div>
              <div>Scoring <code>{view.scoringVersion || '—'}</code></div>
              <div className="col-span-2">Build run <code className="break-all">{view.buildRunId || '—'}</code></div>
              <div className="col-span-2">Deterministic key <code className="break-all">{view.deterministicKey || '—'}</code></div>
              <div>Created {formatUtc(view.createdAt)}</div>
              <div>Updated {formatUtc(view.updatedAt)}</div>
            </div>
            <div className="mt-1">
              <JsonViewer label="Opportunity payload JSON" value={view.opportunityPayload} />
            </div>
          </Section>
        </div>
      )}
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <div className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-gray-500">{title}</div>
      {children}
    </div>
  );
}

function IdList({ label, ids, tone }: { label: string; ids: string[]; tone: 'amber' | 'red' | 'gray' }) {
  const toneCls = tone === 'amber' ? 'text-amber-700' : tone === 'red' ? 'text-red-700' : 'text-gray-500';
  return (
    <div>
      <div className="text-gray-500">{label}: {ids.length ? <span className={`font-medium ${toneCls}`}>{ids.length}</span> : <span className="text-gray-400">0</span>}</div>
      {ids.length ? (
        <code className={`break-all text-[11px] ${toneCls}`} title={ids.join(', ')}>{ids.slice(0, 6).join(', ')}{ids.length > 6 ? ` … (+${ids.length - 6})` : ''}</code>
      ) : null}
    </div>
  );
}

// One contribution row; expandable to the hypothesis definition + state lineage.
function ContributionRow({
  contribution: c,
  tenantId,
  reasonCodes,
  marketStateId,
}: {
  contribution: MarketOpsOpportunityContribution;
  tenantId: string;
  reasonCodes: string[];
  marketStateId: string;
}) {
  const [open, setOpen] = useState(false);
  return (
    <div className="rounded border border-gray-200 bg-white">
      <button type="button" onClick={() => setOpen((v) => !v)} className="flex w-full items-center justify-between gap-2 p-2 text-left">
        <div className="flex flex-wrap items-center gap-2 text-xs">
          {open ? <ChevronDown size={12} className="text-gray-400" /> : <ChevronRight size={12} className="text-gray-400" />}
          <code className="text-gray-800">{c.hypothesisKey || '—'}<span className="text-gray-400"> v{c.hypothesisVersion || '—'}</span></code>
          {c.domain ? <span className="rounded border border-gray-200 px-1.5 text-[11px] text-gray-600">{c.domain}</span> : null}
        </div>
        <div className="flex items-center gap-2 text-[11px] text-gray-500">
          <span>trigger <strong className="text-gray-800">{formatScore(c.triggerScore)}</strong></span>
          <span>conf <strong className="text-gray-800">{formatScore(c.confidenceScore)}</strong></span>
          <span>quality <strong className="text-gray-800">{formatScore(c.qualityScore)}</strong></span>
        </div>
      </button>
      {(reasonCodes.length || open) && (
        <div className="border-t border-gray-100 px-2 pb-2 pt-1 text-[11px] text-gray-600">
          {reasonCodes.length ? (
            <div className="mb-1 flex flex-wrap items-center gap-1">
              <span className="text-gray-400">reasons:</span>
              {reasonCodes.map((r) => (
                <span key={r} className="rounded border border-gray-200 bg-gray-50 px-1.5 py-0.5 text-gray-600" title={r}>{r}</span>
              ))}
            </div>
          ) : null}
          {open ? <ContributionDetail hypothesisKey={c.hypothesisKey} hypothesisVersion={c.hypothesisVersion} tenantId={tenantId} marketStateId={marketStateId} /> : null}
        </div>
      )}
    </div>
  );
}

function ContributionDetail({
  hypothesisKey,
  hypothesisVersion,
  tenantId,
  marketStateId,
}: {
  hypothesisKey: string;
  hypothesisVersion: string;
  tenantId: string;
  marketStateId: string;
}) {
  const hypQ = useMarketOpsHypothesis(hypothesisKey || null, hypothesisVersion || null, tenantId);
  const hyp = hypQ.data?.hypothesis;
  return (
    <div className="space-y-1">
      {hypQ.isLoading ? <div className="text-gray-400">Loading hypothesis…</div> : null}
      {hypQ.isError ? <div className="text-red-600">Hypothesis unavailable.</div> : null}
      {hyp ? (
        <div className="space-y-0.5">
          <div className="text-gray-700">{hyp.title || hyp.hypothesis_key}</div>
          {hyp.description ? <div className="text-gray-500">{hyp.description}</div> : null}
          <div className="text-gray-400">domain {hyp.domain || '—'} · direction {hyp.direction || '—'} · owner {hyp.owner || '—'}</div>
        </div>
      ) : null}
      <StateLineageAffordance marketStateId={marketStateId} />
    </div>
  );
}

function StateLineageAffordance({ marketStateId }: { marketStateId: string }) {
  const [open, setOpen] = useState(false);
  const lineageQ = useMarketOpsMarketStateLineage(open && marketStateId ? marketStateId : null);
  if (!marketStateId) return null;
  return (
    <div>
      <button type="button" onClick={() => setOpen((v) => !v)} className="inline-flex items-center gap-1 text-gray-500 hover:text-gray-700">
        {open ? <ChevronDown size={11} /> : <ChevronRight size={11} />} state lineage
      </button>
      {open && lineageQ.isLoading ? <div className="text-gray-400">Loading lineage…</div> : null}
      {open && lineageQ.isError ? <div className="text-red-600">Lineage unavailable.</div> : null}
      {open && lineageQ.data ? (
        <div className="text-gray-500">
          <div>{lineageQ.data.lineage.source_event_ids.length} source event(s) · {lineageQ.data.lineage.source_artifact_ids.length} artifact(s)</div>
          {lineageQ.data.lineage.missing_feature_observation_ids.length ? (
            <div className="text-amber-700">{lineageQ.data.lineage.missing_feature_observation_ids.length} missing feature observation(s)</div>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}

// Supporting evidence list with lazy per-row detail expansion.
function EvidenceList({ evidenceIds }: { evidenceIds: string[] }) {
  const [expandedId, setExpandedId] = useState<string | null>(null);
  if (!evidenceIds.length) return <p className="text-[11px] text-gray-400">No supporting evidence IDs.</p>;
  return (
    <div className="space-y-1">
      {evidenceIds.map((id) => (
        <div key={id} className="rounded border border-gray-200 bg-white">
          <button type="button" onClick={() => setExpandedId((cur) => (cur === id ? null : id))} className="flex w-full items-center justify-between gap-2 p-2 text-left text-xs">
            <span className="flex items-center gap-1">
              {expandedId === id ? <ChevronDown size={12} className="text-gray-400" /> : <ChevronRight size={12} className="text-gray-400" />}
              <code className="break-all text-gray-700">{id}</code>
            </span>
            <CopyButton value={id} />
          </button>
          {expandedId === id ? <EvidenceDetail evidenceId={id} /> : null}
        </div>
      ))}
    </div>
  );
}

function EvidenceDetail({ evidenceId }: { evidenceId: string }) {
  const evQ = useMarketOpsEvidence(evidenceId);
  const ev = evQ.data?.evidence;
  return (
    <div className="border-t border-gray-100 px-2 pb-2 pt-1 text-[11px] text-gray-600">
      {evQ.isLoading ? <div className="text-gray-400">Loading evidence…</div> : null}
      {evQ.isError ? <div className="text-red-600">Evidence unavailable.</div> : null}
      {ev ? (
        <div className="space-y-0.5">
          {ev.statement ? <div className="text-gray-700">{ev.statement}</div> : null}
          <div className="text-gray-500">
            {ev.evidence_type || '—'} · {ev.domain || '—'}{ev.direction ? ` · ${ev.direction}` : ''} · magnitude {formatScore(ev.magnitude)}
          </div>
          <JsonViewer label="Evidence payload JSON" value={ev.evidence_payload} />
        </div>
      ) : null}
    </div>
  );
}

function EmptyQueueDiagnostics({
  loading,
  failed,
  error,
  aggregation,
  onClear,
}: {
  loading: boolean;
  failed: boolean;
  error: unknown;
  aggregation: { evaluated: number; eligible: number; triggered: number; entries: { token: string; label: string; count: number }[] };
  onClear: () => void;
}) {
  const noSource = !loading && !failed && aggregation.evaluated === 0;
  return (
    <div className="rounded border border-gray-200 bg-white p-3 text-sm">
      <div className="text-xs font-semibold text-gray-700">No eligible opportunities in this scope</div>
      {loading ? (
        <div className="mt-1 text-[11px] text-gray-400">Loading rejection diagnostics…</div>
      ) : failed ? (
        <div className="mt-1 text-[11px] text-red-600">
          Diagnostics unavailable{isApiError(error) ? `: ${error.message}` : ''}.
        </div>
      ) : noSource ? (
        <p className="mt-1 text-[11px] text-gray-500">
          No source hypothesis evaluations in this scope — there is no evidence to rank yet. This is distinct from evaluations blocked by quality or coverage gates.
        </p>
      ) : (
        <>
          <div className="mt-1 flex flex-wrap items-center gap-x-4 gap-y-1 text-[11px] text-gray-600">
            <span>Evaluated <strong className="text-gray-800">{aggregation.evaluated}</strong></span>
            <span>Eligible <strong className="text-gray-800">{aggregation.eligible}</strong></span>
            <span>Triggered <strong className="text-gray-800">{aggregation.triggered}</strong></span>
          </div>
          {aggregation.entries.length ? (
            <>
              <div className="mt-2 text-[11px] font-medium text-gray-500">Most frequent rejection reasons</div>
              <ul className="mt-1 space-y-0.5 text-[11px] text-gray-700">
                {aggregation.entries.map((e) => (
                  <li key={e.token} className="flex items-center justify-between gap-2" title={e.token}>
                    <span>{e.label}</span>
                    <strong className="text-gray-800">{e.count}</strong>
                  </li>
                ))}
              </ul>
            </>
          ) : (
            <p className="mt-1 text-[11px] text-gray-500">Evaluations are present but none carry rejection reason codes.</p>
          )}
        </>
      )}
      <button type="button" onClick={onClear} className="mt-2 inline-flex items-center gap-1 rounded border border-gray-300 bg-white px-2 py-1 text-xs text-gray-700 hover:bg-gray-50">
        <Eraser size={14} /> Clear filters
      </button>
      <p className="mt-1 text-[11px] text-gray-400">G139 performs no provider ingestion or build action from this page.</p>
    </div>
  );
}
