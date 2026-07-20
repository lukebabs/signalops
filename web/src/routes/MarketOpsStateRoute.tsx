import { useState } from 'react';
import { useNavigate, useSearch, Link } from '@tanstack/react-router';
import { useQueryClient } from '@tanstack/react-query';
import { LineChart, RotateCw, ChevronDown, ChevronRight } from 'lucide-react';
import {
  useMarketOpsStates,
  useMarketOpsMarketStateLineage,
  useMarketOpsFeatureDefinitions,
  useMarketOpsStateTransitions,
  useMarketOpsHypothesisEvaluations,
  useMarketOpsHypothesesList,
  useMarketOpsBacktestCalibrationSummaries,
  useMarketOpsEvidence,
  useAlgorithmSignalProposals,
} from '../api/queries';
import { isApiError } from '../api/client';
import { LoadingState, ErrorState, EmptyState } from '../components/States';
import { JsonViewer } from '../components/JsonViewer';
import { CopyButton } from '../components/CopyButton';
import { formatUtc } from '../lib/format';
import {
  summarizeMarketOpsState,
  summarizeMarketOpsFeatureDefinition,
  summarizeMarketOpsFeatureObservation,
  summarizeMarketOpsStateTransition,
  summarizeMarketOpsHypothesisEvaluation,
  selectPriorState,
  observationSurfaceCellId,
  groupObservationsBySurfaceCell,
  SURFACE_CELLS,
  isMaterialTransition,
  parseHypothesisCalibrationReport,
  parseHypothesisRequirements,
  requirementMatches,
  qualityReason,
  qualityStateStyle,
  formatNullableNumber,
  formatNullablePercent,
  type CalibrationSegmentMap,
  type MarketOpsMarketStateView,
  type MarketOpsFeatureObservationView,
  type MarketOpsStateTransitionView,
  type MarketOpsHypothesisEvaluationStateView,
} from '../lib/marketopsState';
import { useTenant } from '../auth/session';
import type { MarketOpsHypothesisDefinition } from '../types';

// G147 Market State analyst experience (read-only composition over G136-G146).
// Scope bar + four tabs (Overview / Surface / Transitions / Hypotheses). All
// mutation surfaces (provider acquisition, state/evaluation build, lifecycle
// promotion, proposal review, materialization) are intentionally absent. Analyst
// context persists in URL search params.

const TABS = ['overview', 'surface', 'transitions', 'hypotheses'] as const;
type Tab = (typeof TABS)[number];
const inputCls = 'rounded border border-gray-300 px-2 py-1 text-sm';
const DOMAIN_BUCKETS = ['underlying', 'volatility', 'positioning', 'premium', 'liquidity_quality', 'event', 'other'] as const;

export function MarketOpsStateRoute() {
  const TENANT_ID = useTenant();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const search = useSearch({ strict: false }) as {
    symbol?: string;
    session_date?: string;
    tab?: string;
    hypothesis_key?: string;
    hypothesis_version?: string;
  };

  const symbol = (search.symbol || '').trim().toUpperCase();
  const requestedTab = (search.tab || 'overview') as Tab;
  const tab: Tab = (TABS as readonly string[]).includes(requestedTab) ? requestedTab : 'overview';

  function setSearch(patch: Record<string, string | undefined>) {
    const next = {
      symbol: search.symbol,
      session_date: search.session_date,
      tab: search.tab,
      hypothesis_key: search.hypothesis_key,
      hypothesis_version: search.hypothesis_version,
      ...patch,
    };
    void navigate({ to: '/marketops/state', search: next });
  }

  // State window for the selected symbol.
  const statesQ = useMarketOpsStates({ tenant_id: TENANT_ID, symbol: symbol || undefined, limit: 50 });
  const states = (statesQ.data?.market_states ?? []).map(summarizeMarketOpsState);
  // Newest revision per session; overall newest first by session then as_of.
  const sessionStates = dedupeNewestRevision(states);
  const sessionDates = sessionStates.map((s) => dateOnly(s.sessionDate));
  const selectedSession = search.session_date && sessionDates.includes(dateOnly(search.session_date))
    ? dateOnly(search.session_date)
    : sessionDates[0] ?? '';
  const selectedState = selectedSession
    ? (sessionStates.find((s) => dateOnly(s.sessionDate) === selectedSession) ?? null)
    : null;
  const priorState = selectedState ? selectPriorState(states, selectedState) : null;

  if (!symbol) {
    return (
      <div className="space-y-3">
        <Header onRefresh={() => queryClient.invalidateQueries({ queryKey: ['marketops-states'] })} refreshing={statesQ.isFetching} />
        <div className="rounded border border-gray-200 bg-white p-3">
          <EmptyState message="Choose an asset to inspect market state" />
          <div className="mt-1 text-xs text-gray-500">
            <Link to="/marketops/assets" className="text-brand-700 underline">Browse Assets</Link> to select a symbol.
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <Header onRefresh={() => queryClient.invalidateQueries({ queryKey: ['marketops-states'] })} refreshing={statesQ.isFetching} />
      <ScopeBar
        symbol={symbol}
        sessionDates={sessionDates}
        selectedSession={selectedSession}
        onSession={(d) => setSearch({ session_date: d, hypothesis_key: undefined, hypothesis_version: undefined })}
        onSymbol={() => setSearch({ symbol: undefined, session_date: undefined, hypothesis_key: undefined, hypothesis_version: undefined })}
        statesQ={statesQ}
        selectedState={selectedState}
      />

      <div className="flex flex-wrap gap-1 border-b border-gray-200">
        {TABS.map((t) => (
          <button
            key={t}
            type="button"
            onClick={() => setSearch({ tab: t })}
            className={`-mb-px border-b-2 px-3 py-1.5 text-sm capitalize ${tab === t ? 'border-brand-600 font-semibold text-brand-700' : 'border-transparent text-gray-500 hover:text-gray-700'}`}
          >
            {t}
          </button>
        ))}
      </div>

      {!selectedState ? (
        <div className="rounded border border-gray-200 bg-white p-3">
          {statesQ.isLoading ? <LoadingState label="Loading market state..." /> : <EmptyState message="No persisted market state for this scope" />}
        </div>
      ) : tab === 'overview' ? (
        <OverviewTab tenantId={TENANT_ID} selectedState={selectedState} priorStateId={priorState?.marketStateId ?? null} priorLabel={priorState ? dateOnly(priorState.sessionDate) : null} />
      ) : tab === 'surface' ? (
        <SurfaceTab tenantId={TENANT_ID} selectedState={selectedState} />
      ) : tab === 'transitions' ? (
        <TransitionsTab tenantId={TENANT_ID} symbol={symbol} selectedState={selectedState} />
      ) : (
        <HypothesesTab
          tenantId={TENANT_ID}
          selectedState={selectedState}
          hypothesisKey={search.hypothesis_key ?? ''}
          hypothesisVersion={search.hypothesis_version ?? ''}
          onSelectHypothesis={(key, version) => setSearch({ hypothesis_key: key || undefined, hypothesis_version: version || undefined })}
        />
      )}
    </div>
  );
}

function Header({ onRefresh, refreshing }: { onRefresh: () => void; refreshing: boolean }) {
  return (
    <div className="flex items-center justify-between gap-2">
      <h1 className="flex items-center gap-1 text-lg font-semibold">
        <LineChart size={18} className="text-brand-700" /> Market State
      </h1>
      <button type="button" onClick={onRefresh} title="Refresh" aria-label="Refresh market state" className={`${inputCls} inline-flex items-center gap-1 bg-white`}>
        <RotateCw size={14} className={refreshing ? 'animate-spin' : ''} />
      </button>
    </div>
  );
}

function ScopeBar({
  symbol,
  sessionDates,
  selectedSession,
  onSession,
  onSymbol,
  statesQ,
  selectedState,
}: {
  symbol: string;
  sessionDates: string[];
  selectedSession: string;
  onSession: (d: string) => void;
  onSymbol: () => void;
  statesQ: ReturnType<typeof useMarketOpsStates>;
  selectedState: MarketOpsMarketStateView | null;
}) {
  return (
    <div className="flex flex-wrap items-center gap-2 rounded border border-gray-200 bg-white p-2">
      <button type="button" onClick={onSymbol} className={`${inputCls} font-mono bg-white`} title="Change symbol">{symbol}</button>
      <select value={selectedSession} onChange={(e) => onSession(e.target.value)} className={inputCls} aria-label="Session date" disabled={!sessionDates.length}>
        {sessionDates.length === 0 ? <option value="">no sessions</option> : sessionDates.map((d) => <option key={d} value={d}>{d}</option>)}
      </select>
      {selectedState ? (
        <>
          <Badge tone={qualityStateStyle(selectedState.qualityState)}>{selectedState.qualityState || '—'}</Badge>
          <span className="text-xs text-gray-600">schema <code>{selectedState.stateSchemaVersion || '—'}</code></span>
          <span className="text-xs text-gray-600">
            completeness <strong>{formatNullablePercent(selectedState.completenessRatio)}</strong>
            {' · '}{selectedState.featureCount} total features · {selectedState.requiredFeatureCount} required
          </span>
          <span className="text-xs text-gray-600">as-of {selectedState.asOfTime ? formatUtc(selectedState.asOfTime) : '—'}</span>
        </>
      ) : statesQ.isError ? (
        <span className="text-xs text-red-600">State unavailable{isApiError(statesQ.error) ? `: ${statesQ.error.message}` : ''}</span>
      ) : null}
    </div>
  );
}

// Overview: lineage observations grouped into ordered domains, with prior exact-match comparison.
function OverviewTab({
  tenantId,
  selectedState,
  priorStateId,
  priorLabel,
}: {
  tenantId: string;
  selectedState: MarketOpsMarketStateView;
  priorStateId: string | null;
  priorLabel: string | null;
}) {
  const lineageQ = useMarketOpsMarketStateLineage(selectedState.marketStateId);
  const priorQ = useMarketOpsMarketStateLineage(priorStateId);
  const defsQ = useMarketOpsFeatureDefinitions({ tenant_id: tenantId, limit: 200 });
  const observations = (lineageQ.data?.lineage.feature_observations ?? []).map(summarizeMarketOpsFeatureObservation);
  const priorObservations = (priorQ.data?.lineage.feature_observations ?? []).map(summarizeMarketOpsFeatureObservation);
  const defs = new Map((defsQ.data?.feature_definitions ?? []).map(summarizeMarketOpsFeatureDefinition).map((d) => [`${d.featureKey}:${d.featureVersion}`, d]));
  const priorBySig = new Map(priorObservations.map((o) => [obsSignature(o), o]));
  const missing = lineageQ.data?.lineage.missing_feature_observation_ids ?? [];

  const grouped = DOMAIN_BUCKETS.map((bucket) => ({
    bucket,
    rows: observations
      .filter((o) => domainBucket(defs.get(`${o.featureKey}:${o.featureVersion}`)?.domain ?? '') === bucket)
      .map((o) => ({ obs: o, prior: priorBySig.get(obsSignature(o)) ?? null })),
  })).filter((g) => g.rows.length);

  return (
    <div className="space-y-3">
      <div className="rounded border border-gray-200 bg-gray-50 p-2 text-xs text-gray-700">
        <div className="flex flex-wrap items-center gap-x-4 gap-y-1">
          <Badge tone={qualityStateStyle(selectedState.qualityState)}>quality {selectedState.qualityState || '—'}</Badge>
          <span>completeness {formatNullablePercent(selectedState.completenessRatio)} · {selectedState.featureCount} total · {selectedState.requiredFeatureCount} required</span>
          <span>eligible hypotheses {selectedState.eligibleHypotheses.length}</span>
          <span>missing features {missing.length}</span>
          <span>prior session {priorLabel ?? '—'}</span>
        </div>
      </div>

      {lineageQ.isLoading ? <LoadingState label="Loading state lineage..." /> : lineageQ.isError ? (
        <ErrorState error={lineageQ.error} />
      ) : observations.length === 0 ? (
        <EmptyState message="No feature observations in this state lineage." />
      ) : (
        grouped.map((g) => (
          <div key={g.bucket} className="rounded border border-gray-200 bg-white p-2">
            <div className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-gray-500">{domainLabel(g.bucket)}</div>
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200 text-sm">
                <thead className="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500">
                  <tr>
                    <th className="px-2 py-1">Feature</th>
                    <th className="px-2 py-1">Current</th>
                    <th className="px-2 py-1">Prior</th>
                    <th className="px-2 py-1">Δ</th>
                    <th className="px-2 py-1">Quality</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {g.rows.map(({ obs, prior }) => {
                    const def = defs.get(`${obs.featureKey}:${obs.featureVersion}`);
                    const cur = obsValue(obs);
                    const prv = prior ? obsValue(prior) : null;
                    const delta = obs.numericValue !== null && prior && prior.numericValue !== null && cur !== null && typeof cur === 'number' && prv !== null && typeof prv === 'number'
                      ? cur - prv
                      : null;
                    return (
                      <tr key={obs.featureObservationId} className="align-top">
                        <td className="px-2 py-1">
                          <div className="text-xs font-medium text-gray-800">{def?.title || obs.featureKey}</div>
                          <div className="text-[11px] text-gray-500">{obs.featureKey} v{obs.featureVersion}</div>
                        </td>
                        <td className="px-2 py-1 text-xs text-gray-800">{formatObsValue(cur, def?.unit ?? null)}</td>
                        <td className="px-2 py-1 text-xs text-gray-600">{prv === null ? '—' : formatObsValue(prv, def?.unit ?? null)}</td>
                        <td className="px-2 py-1 text-xs text-gray-700">{delta === null ? '—' : formatDelta(delta)}</td>
                        <td className="px-2 py-1"><Badge tone={qualityStateStyle(obs.qualityState)}>{obs.qualityState || '—'}</Badge></td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>
        ))
      )}
    </div>
  );
}

// Surface: canonical seven-cell grid from exact persisted feature keys/dimensions.
function SurfaceTab({ tenantId, selectedState }: { tenantId: string; selectedState: MarketOpsMarketStateView }) {
  const lineageQ = useMarketOpsMarketStateLineage(selectedState.marketStateId);
  const transitionsQ = useMarketOpsStateTransitions(
    { tenant_id: tenantId, current_state_id: selectedState.marketStateId, limit: 200 },
    true,
  );
  const observations = (lineageQ.data?.lineage.feature_observations ?? []).map(summarizeMarketOpsFeatureObservation);
  const transitions = (transitionsQ.data?.transitions ?? []).map(summarizeMarketOpsStateTransition);
  const grouped = groupObservationsBySurfaceCell(observations);
  const transitionGroups = new Map<string, MarketOpsStateTransitionView[]>();
  for (const transition of transitions) {
    const cellId = observationSurfaceCellId(transition.featureKey, transition.dimensions);
    if (!cellId) continue;
    transitionGroups.set(cellId, [...(transitionGroups.get(cellId) ?? []), transition]);
  }

  const ivMoves = transitions
    .filter((t) => observationSurfaceCellId(t.featureKey, t.dimensions) && t.featureKey.includes('iv') && t.transitionValue !== null)
    .sort((a, b) => Math.abs(b.transitionValue ?? 0) - Math.abs(a.transitionValue ?? 0));
  const accelerated = transitions.filter((t) => t.transitionType === 'acceleration' && observationSurfaceCellId(t.featureKey, t.dimensions));
  const oiMoves = transitions
    .filter((t) => t.featureKey.includes('oi_change') && t.transitionValue !== null)
    .sort((a, b) => Math.abs(b.transitionValue ?? 0) - Math.abs(a.transitionValue ?? 0));
  const maturityMoves = new Map<number, number>();
  for (const transition of ivMoves) {
    const cell = SURFACE_CELLS.find((item) => item.id === observationSurfaceCellId(transition.featureKey, transition.dimensions));
    if (cell) maturityMoves.set(cell.targetDte, (maturityMoves.get(cell.targetDte) ?? 0) + Math.abs(transition.transitionValue ?? 0));
  }
  const broadestMaturity = Array.from(maturityMoves.entries()).sort((a, b) => b[1] - a[1])[0];
  const unavailable = SURFACE_CELLS.filter((cell) => {
    const rows = grouped.get(cell.id) ?? [];
    return !rows.length || rows.every((row) => !row.qualityState.startsWith('usable'));
  });

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap gap-x-4 gap-y-1 rounded border border-gray-200 bg-gray-50 p-2 text-[11px] text-gray-600">
        <span>largest IV move <strong>{ivMoves[0] ? `${ivMoves[0].featureKey} ${formatNullableNumber(ivMoves[0].transitionValue, 4)}` : 'Unavailable'}</strong></span>
        <span>accelerating cells <strong>{accelerated.length || 'Unavailable'}</strong></span>
        <span>broadest maturity <strong>{broadestMaturity ? `${broadestMaturity[0]} DTE` : 'Unavailable'}</strong></span>
        <span>largest unusual OI change <strong>{oiMoves[0] ? formatNullableNumber(oiMoves[0].transitionValue, 2) : 'Unavailable'}</strong></span>
        <span>unusable cells <strong>{unavailable.length}</strong></span>
      </div>

      {lineageQ.isLoading ? <LoadingState label="Loading state lineage..." /> : lineageQ.isError ? <ErrorState error={lineageQ.error} /> : (
        <div className="grid grid-cols-1 gap-2 md:grid-cols-2 lg:grid-cols-3">
          {SURFACE_CELLS.map((cell) => {
            const cellObs = grouped.get(cell.id) ?? [];
            const cellTransitions = transitionGroups.get(cell.id) ?? [];
            const accessibleSummary = cellObs.length
              ? cellObs.map((o) => `${o.featureKey} ${formatObsValue(obsValue(o), null)} quality ${o.qualityState || 'unavailable'}`).join(', ')
              : 'unavailable';
            return (
              <div key={cell.id} className="rounded border border-gray-200 bg-white p-2" aria-label={`${cell.label}: ${accessibleSummary}`}>
                <div className="mb-1 text-xs font-semibold text-gray-700">{cell.label}</div>
                {cellObs.length ? (
                  <div className="space-y-0.5">
                    {cellObs.map((observation) => {
                      const reason = qualityReason(observation.qualityDetails);
                      return (
                        <div key={observation.featureObservationId} className="flex items-start justify-between gap-2 text-[11px]">
                          <code className="break-all text-gray-600" title={`${observation.featureKey} v${observation.featureVersion}`}>{observation.featureKey}</code>
                          <span className="text-right">
                            <span className="flex items-center justify-end gap-1">
                              <span className="text-gray-800">{formatObsValue(obsValue(observation), null)}</span>
                              <Badge tone={qualityStateStyle(observation.qualityState)}>{observation.qualityState || '—'}</Badge>
                            </span>
                            {reason ? <span className="block text-gray-400">{reason}</span> : null}
                          </span>
                        </div>
                      );
                    })}
                    {cellTransitions.map((transition) => (
                      <div key={transition.transitionId} className="flex items-center justify-between gap-2 border-t border-gray-100 pt-0.5 text-[11px]">
                        <span className="text-gray-500">{transition.transitionType} · {transition.lookbackSessions ?? '—'} sessions</span>
                        <span className="text-gray-700">{formatNullableNumber(transition.transitionValue, 4)}</span>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="text-[11px] text-gray-400">Unavailable · no persisted matching cell</div>
                )}
              </div>
            );
          })}
        </div>
      )}
      {transitionsQ.isError ? <div className="text-[11px] text-amber-700">Surface transitions unavailable; persisted observations remain shown.</div> : null}
      <p className="text-[11px] text-gray-400">ATM cells map by explicit ATM feature key or normalized ATM dimensions; wings map by decimal delta 0.25. Missing values are never zero.</p>
    </div>
  );
}

// Transitions: bounded multi-session timeline with client-side presentation filters.
function TransitionsTab({ tenantId, symbol, selectedState }: { tenantId: string; symbol: string; selectedState: MarketOpsMarketStateView }) {
  const [showAll, setShowAll] = useState(false);
  const [domain, setDomain] = useState('');
  const [transitionType, setTransitionType] = useState('');
  const [lookback, setLookback] = useState('');
  const [direction, setDirection] = useState('');
  const [quality, setQuality] = useState('');
  const transitionsQ = useMarketOpsStateTransitions(
    { tenant_id: tenantId, symbol, session_end: dateOnly(selectedState.sessionDate), limit: 200 },
    true,
  );
  const defsQ = useMarketOpsFeatureDefinitions({ tenant_id: tenantId, limit: 200 });
  const definitions = new Map(
    (defsQ.data?.feature_definitions ?? [])
      .map(summarizeMarketOpsFeatureDefinition)
      .map((definition) => [`${definition.featureKey}:${definition.featureVersion}`, definition]),
  );
  const all = (transitionsQ.data?.transitions ?? []).map(summarizeMarketOpsStateTransition);
  const candidates = showAll ? all : all.filter(isMaterialTransition);
  const rows = candidates.filter((transition) => {
    const transitionDomain = definitions.get(`${transition.featureKey}:${transition.featureVersion}`)?.domain ?? 'other';
    return (!domain || transitionDomain === domain) &&
      (!transitionType || transition.transitionType === transitionType) &&
      (!lookback || String(transition.lookbackSessions ?? '') === lookback) &&
      (!direction || transition.direction === direction) &&
      (!quality || transition.qualityState === quality);
  });
  const groups = new Map<string, MarketOpsStateTransitionView[]>();
  for (const transition of rows) {
    const transitionDomain = definitions.get(`${transition.featureKey}:${transition.featureVersion}`)?.domain ?? 'other';
    const key = `${dateOnly(transition.sessionDate)}|${transitionDomain}`;
    groups.set(key, [...(groups.get(key) ?? []), transition]);
  }
  const sortedGroups = Array.from(groups.entries()).sort((a, b) => b[0].localeCompare(a[0]));
  const domains = uniqueStrings(all.map((t) => definitions.get(`${t.featureKey}:${t.featureVersion}`)?.domain ?? 'other'));
  const types = uniqueStrings(all.map((t) => t.transitionType));
  const lookbacks = uniqueStrings(all.map((t) => t.lookbackSessions === null ? '' : String(t.lookbackSessions)));
  const directions = uniqueStrings(all.map((t) => t.direction ?? ''));
  const qualities = uniqueStrings(all.map((t) => t.qualityState));

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-center gap-2">
        <select className={inputCls} aria-label="Transition domain" value={domain} onChange={(e) => setDomain(e.target.value)}><option value="">all domains</option>{domains.map((v) => <option key={v}>{v}</option>)}</select>
        <select className={inputCls} aria-label="Transition type" value={transitionType} onChange={(e) => setTransitionType(e.target.value)}><option value="">all types</option>{types.map((v) => <option key={v}>{v}</option>)}</select>
        <select className={inputCls} aria-label="Transition lookback" value={lookback} onChange={(e) => setLookback(e.target.value)}><option value="">all lookbacks</option>{lookbacks.map((v) => <option key={v} value={v}>{v} sessions</option>)}</select>
        <select className={inputCls} aria-label="Transition direction" value={direction} onChange={(e) => setDirection(e.target.value)}><option value="">all directions</option>{directions.map((v) => <option key={v}>{v}</option>)}</select>
        <select className={inputCls} aria-label="Transition quality" value={quality} onChange={(e) => setQuality(e.target.value)}><option value="">all quality</option>{qualities.map((v) => <option key={v}>{v}</option>)}</select>
        <button type="button" onClick={() => setShowAll((value) => !value)} className={`${inputCls} bg-white`}>
          {showAll ? 'Show material only' : 'Show all returned transitions'}
        </button>
      </div>
      <div className="text-[11px] text-gray-500">Showing {rows.length} / {all.length} returned transitions through {dateOnly(selectedState.sessionDate)}</div>
      {all.length === 200 ? <div className="text-[11px] text-amber-700">Transition result reached the 200-row bound and may be truncated.</div> : null}

      {transitionsQ.isLoading || defsQ.isLoading ? <LoadingState label="Loading transition timeline..." /> : transitionsQ.isError ? <ErrorState error={transitionsQ.error} /> : sortedGroups.length ? (
        sortedGroups.map(([key, group]) => {
          const [session, groupDomain] = key.split('|');
          return (
            <div key={key} className="rounded border border-gray-200 bg-white">
              <div className="flex items-center justify-between border-b border-gray-100 bg-gray-50 px-2 py-1 text-[11px] font-semibold text-gray-600">
                <span>{session}</span><span>{groupDomain}</span>
              </div>
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200 text-sm">
                  <thead className="text-left text-xs uppercase tracking-wide text-gray-500">
                    <tr><th className="px-2 py-1">Feature</th><th className="px-2 py-1">Type</th><th className="px-2 py-1">Lookback</th><th className="px-2 py-1">Current</th><th className="px-2 py-1">Baseline</th><th className="px-2 py-1">Δ</th><th className="px-2 py-1">Z / pct</th><th className="px-2 py-1">Persistence</th><th className="px-2 py-1">Direction</th><th className="px-2 py-1">Quality</th><th className="px-2 py-1">Audit</th></tr>
                  </thead>
                  <tbody className="divide-y divide-gray-100">{group.map((transition) => <TransitionRow key={transition.transitionId} transition={transition} />)}</tbody>
                </table>
              </div>
            </div>
          );
        })
      ) : (
        <EmptyState message={all.length ? 'No transitions match the current presentation filters.' : 'No transitions returned for this scope.'} />
      )}
    </div>
  );
}

function TransitionRow({ transition }: { transition: MarketOpsStateTransitionView }) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <tr className="align-top">
        <td className="px-2 py-1"><code className="break-all text-[11px] text-gray-700">{transition.featureKey}</code></td>
        <td className="whitespace-nowrap px-2 py-1 text-xs text-gray-700">{transition.transitionType || '—'}</td>
        <td className="whitespace-nowrap px-2 py-1 text-xs text-gray-600">{transition.lookbackSessions ?? '—'}</td>
        <td className="whitespace-nowrap px-2 py-1 text-xs text-gray-800">{formatNullableNumber(transition.currentValue)}</td>
        <td className="whitespace-nowrap px-2 py-1 text-xs text-gray-600">{formatNullableNumber(transition.baselineValue)}</td>
        <td className="whitespace-nowrap px-2 py-1 text-xs text-gray-700">{formatNullableNumber(transition.transitionValue)}</td>
        <td className="whitespace-nowrap px-2 py-1 text-xs text-gray-700">{transition.zscore !== null ? formatNullableNumber(transition.zscore) : formatNullablePercent(transition.percentile)}</td>
        <td className="whitespace-nowrap px-2 py-1 text-xs text-gray-600">{transition.persistenceSessions ?? '—'}</td>
        <td className="whitespace-nowrap px-2 py-1 text-xs text-gray-600">{transition.direction || '—'}</td>
        <td className="px-2 py-1"><Badge tone={qualityStateStyle(transition.qualityState)}>{transition.qualityState || '—'}</Badge></td>
        <td className="px-2 py-1"><button type="button" className="text-xs text-brand-700 underline" onClick={() => setOpen((value) => !value)}>{open ? 'hide' : 'show'}</button></td>
      </tr>
      {open ? (
        <tr><td colSpan={11} className="bg-gray-50 px-3 py-2 text-[11px] text-gray-600">
          <div>current state <code>{transition.currentStateId}</code> · baseline <code>{transition.baselineStateId || '—'}</code> · run <code>{transition.calculationRunId || '—'}</code></div>
          <div>as-of {transition.asOfTime ? formatUtc(transition.asOfTime) : '—'} · deterministic key <code>{transition.deterministicKey || '—'}</code></div>
          <JsonViewer label="Transition payload JSON" value={transition.transitionPayload} />
        </td></tr>
      ) : null}
    </>
  );
}

// Hypotheses: one bounded definition/evaluation/lineage/transition composition.
function HypothesesTab({
  tenantId,
  selectedState,
  hypothesisKey,
  hypothesisVersion,
  onSelectHypothesis,
}: {
  tenantId: string;
  selectedState: MarketOpsMarketStateView;
  hypothesisKey: string;
  hypothesisVersion: string;
  onSelectHypothesis: (key: string, version: string) => void;
}) {
  const defsQ = useMarketOpsHypothesesList({ tenant_id: tenantId, limit: 200 });
  const evalsQ = useMarketOpsHypothesisEvaluations(
    { tenant_id: tenantId, market_state_id: selectedState.marketStateId, limit: 200 },
    true,
  );
  const lineageQ = useMarketOpsMarketStateLineage(selectedState.marketStateId);
  const transitionsQ = useMarketOpsStateTransitions(
    { tenant_id: tenantId, current_state_id: selectedState.marketStateId, limit: 200 },
    true,
  );
  const defs = defsQ.data?.hypotheses ?? [];
  const evals = (evalsQ.data?.hypothesis_evaluations ?? []).map(summarizeMarketOpsHypothesisEvaluation);
  const observations = (lineageQ.data?.lineage.feature_observations ?? []).map(summarizeMarketOpsFeatureObservation);
  const transitions = (transitionsQ.data?.transitions ?? []).map(summarizeMarketOpsStateTransition);
  const evalByKey = new Map(evals.map((evaluation) => [`${evaluation.hypothesisKey}:${evaluation.hypothesisVersion}`, evaluation]));
  const selected = hypothesisKey
    ? defs.find((definition) => definition.hypothesis_key === hypothesisKey && definition.hypothesis_version === hypothesisVersion) ?? null
    : null;

  if (defsQ.isLoading || evalsQ.isLoading || lineageQ.isLoading || transitionsQ.isLoading) {
    return <LoadingState label="Loading hypothesis workbench..." />;
  }
  if (defsQ.isError) return <ErrorState error={defsQ.error} />;
  if (evalsQ.isError) return <ErrorState error={evalsQ.error} />;
  if (lineageQ.isError) return <ErrorState error={lineageQ.error} />;
  if (transitionsQ.isError) return <ErrorState error={transitionsQ.error} />;

  return (
    <div className="flex flex-col gap-3 lg:flex-row">
      <div className={`${selected ? 'hidden lg:block' : ''} lg:w-2/5 lg:min-w-[360px]`}>
        <div className="space-y-1">
          {defs.length === 0 ? (
            <EmptyState message="No hypothesis definitions." />
          ) : (
            defs.map((definition) => {
              const evaluation = evalByKey.get(`${definition.hypothesis_key}:${definition.hypothesis_version}`);
              const isSelected = selected?.hypothesis_key === definition.hypothesis_key && selected?.hypothesis_version === definition.hypothesis_version;
              return (
                <button
                  key={`${definition.hypothesis_key}:${definition.hypothesis_version}`}
                  type="button"
                  onClick={() => onSelectHypothesis(definition.hypothesis_key, definition.hypothesis_version)}
                  className={`w-full rounded border p-2 text-left ${isSelected ? 'border-brand-400 bg-brand-50' : 'border-gray-200 bg-white hover:bg-gray-50'}`}
                >
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-xs font-medium text-gray-800">{definition.title || definition.hypothesis_key}</span>
                    {evaluation ? <Badge tone={qualityStateStyle(evaluation.invalidated ? 'unusable' : evaluation.triggered ? 'usable' : 'degraded')}>{evaluation.invalidated ? 'invalidated' : evaluation.triggered ? 'triggered' : evaluation.eligible ? 'eligible' : 'ineligible'}</Badge> : <span className="text-[11px] text-gray-400">not evaluated</span>}
                  </div>
                  <div className="text-[11px] text-gray-500">{definition.hypothesis_key} v{definition.hypothesis_version} · {definition.domain || '—'} · {definition.lifecycle_status || '—'}</div>
                </button>
              );
            })
          )}
        </div>
      </div>
      <div className={`${selected ? '' : 'hidden lg:block'} flex-1`}>
        {selected ? (
          <div className="space-y-2">
            <button type="button" onClick={() => onSelectHypothesis('', '')} className={`${inputCls} bg-white lg:hidden`}>Back to hypotheses</button>
            <HypothesisDetail
              tenantId={tenantId}
              definition={selected}
              evaluation={evalByKey.get(`${selected.hypothesis_key}:${selected.hypothesis_version}`) ?? null}
              observations={observations}
              transitions={transitions}
            />
          </div>
        ) : (
          <div className="rounded border border-gray-200 bg-white p-3"><EmptyState message="Select a hypothesis to inspect." /></div>
        )}
      </div>
    </div>
  );
}

function HypothesisDetail({
  tenantId,
  definition,
  evaluation,
  observations,
  transitions,
}: {
  tenantId: string;
  definition: MarketOpsHypothesisDefinition;
  evaluation: MarketOpsHypothesisEvaluationStateView | null;
  observations: MarketOpsFeatureObservationView[];
  transitions: MarketOpsStateTransitionView[];
}) {
  const calibQ = useMarketOpsBacktestCalibrationSummaries({
    tenant_id: tenantId,
    app_id: 'marketops',
    domain: 'market_data',
    use_case: 'daily_market_surveillance',
    source_id: 'marketops.research_ledgers',
    dataset: 'hypothesis_evaluations',
    detector_id: `marketops.hypothesis.${definition.hypothesis_key.toLowerCase()}`,
    limit: 25,
  });
  const proposalsQ = useAlgorithmSignalProposals({
    tenant_id: tenantId,
    proposal_source: 'hypothesis_evaluation',
    hypothesis_key: definition.hypothesis_key,
    limit: 50,
  });
  const parsedSummaries = (calibQ.data?.calibration_summaries ?? []).map((summary) => ({
    summary,
    report: parseHypothesisCalibrationReport(summary.parameters, definition.hypothesis_key, definition.hypothesis_version),
  }));
  const exactCalibration = parsedSummaries.find((entry) => entry.report.valid) ?? null;
  const incompatibleCalibration = parsedSummaries[0]?.report ?? null;
  const featureRequirements = parseHypothesisRequirements(definition.required_features);
  const transitionRequirements = parseHypothesisRequirements(definition.required_transitions);
  const proposals = (proposalsQ.data?.algorithm_signal_proposals ?? []).filter(
    (proposal) => proposal.hypothesis_version === definition.hypothesis_version,
  );

  return (
    <div className="space-y-3 rounded border border-gray-200 bg-white p-3">
      <div>
        <div className="text-xs font-semibold text-gray-700">{definition.title || definition.hypothesis_key}</div>
        <div className="text-[11px] text-gray-500">
          {definition.hypothesis_key} v{definition.hypothesis_version} · {definition.domain || '—'} · {definition.direction || '—'} · lifecycle {definition.lifecycle_status || '—'}
        </div>
        {definition.description ? <p className="mt-1 text-xs text-gray-700">{definition.description}</p> : null}
        {typeof definition.rationale === 'string' && definition.rationale ? <p className="mt-1 text-[11px] text-gray-600">{definition.rationale}</p> : null}
      </div>

      <Section title="Required evidence">
        <div className="grid gap-2 md:grid-cols-2">
          <RequirementList title="Features" requirements={featureRequirements} rows={observations} />
          <RequirementList title="Transitions" requirements={transitionRequirements} rows={transitions} />
        </div>
        <JsonViewer label="Quality policy JSON" value={definition.quality_policy} />
      </Section>

      <Section title="Current evaluation">
        {evaluation ? (
          <div className="space-y-1 text-xs text-gray-700">
            <div className="flex flex-wrap items-center gap-2">
              <Badge tone={qualityStateStyle(evaluation.eligible ? 'usable' : 'degraded')}>{evaluation.eligible ? 'eligible' : 'ineligible'}</Badge>
              {evaluation.triggered ? <Badge tone={qualityStateStyle('usable')}>triggered</Badge> : <Badge tone={qualityStateStyle('missing')}>not triggered</Badge>}
              {evaluation.invalidated ? <Badge tone={qualityStateStyle('unusable')}>invalidated</Badge> : null}
            </div>
            <div className="flex flex-wrap items-center gap-x-3 text-[11px] text-gray-600">
              <span>trigger {formatNullableNumber(evaluation.triggerScore)}</span>
              <span>conf {formatNullableNumber(evaluation.confidenceScore)}</span>
              <span>quality {formatNullableNumber(evaluation.qualityScore)}</span>
              <span>rarity {formatNullableNumber(evaluation.rarityScore)}</span>
              <span>magnitude {formatNullableNumber(evaluation.magnitudeScore)}</span>
              <span>persistence {formatNullableNumber(evaluation.persistenceScore)}</span>
              <span>corroboration {formatNullableNumber(evaluation.corroborationScore)}</span>
            </div>
            {evaluation.reasonCodes.length ? (
              <div className="flex flex-wrap items-center gap-1">
                <span className="text-gray-400">reasons:</span>
                {evaluation.reasonCodes.map((reason) => <span key={reason} className="rounded border border-gray-200 bg-gray-50 px-1.5 py-0.5 text-[11px] text-gray-600" title={reason}>{reason}</span>)}
              </div>
            ) : null}
            <JsonViewer label="Evaluation contribution/check payload" value={evaluation.evaluationPayload} />
          </div>
        ) : (
          <p className="text-[11px] text-gray-400">Not evaluated for this state.</p>
        )}
      </Section>

      <Section title="Evidence">
        {evaluation?.evidenceIds.length ? (
          <div className="space-y-1">{evaluation.evidenceIds.map((evidenceId) => <HypothesisEvidenceRow key={evidenceId} evidenceId={evidenceId} />)}</div>
        ) : <p className="text-[11px] text-gray-400">No linked evidence for this evaluation.</p>}
      </Section>

      <Section title="Historical calibration">
        {calibQ.isLoading ? <div className="text-[11px] text-gray-400">Loading calibration...</div> :
         calibQ.isError ? <div className="text-[11px] text-red-600">Calibration unavailable.</div> :
         exactCalibration ? <CalibrationDetail report={exactCalibration.report} /> :
         parsedSummaries.length ? <p className="text-[11px] text-amber-700">{incompatibleCalibration?.incompatibleReason}</p> :
         <p className="text-[11px] text-gray-400">No exact-version calibration report.</p>}
      </Section>

      <Section title="Governance status">
        <div className="mb-1 flex flex-wrap items-center gap-2 text-[11px] text-gray-600">
          <span>lifecycle <strong>{definition.lifecycle_status || '—'}</strong></span>
          <span>approved by <strong>{definition.approved_by || '—'}</strong></span>
          <span>approved at <strong>{definition.approved_at ? formatUtc(definition.approved_at) : '—'}</strong></span>
        </div>
        {proposalsQ.isLoading ? <p className="text-[11px] text-gray-400">Loading proposal status...</p> :
         proposals.length ? (
          <div className="space-y-1">
            {proposals.map((proposal) => (
              <div key={proposal.proposal_id} className="rounded border border-gray-200 bg-gray-50 p-2 text-[11px] text-gray-600">
                <div className="flex flex-wrap items-center gap-2">
                  <Badge tone={qualityStateStyle(proposal.status === 'reviewed' ? 'usable' : proposal.status === 'rejected' ? 'unusable' : 'degraded')}>{proposal.status}</Badge>
                  <span>{proposal.research_only ? 'research-only' : 'production policy evaluated'}</span>
                  <span>{proposal.materialization_eligible ? 'eligibility flag true' : 'materialization ineligible'}</span>
                </div>
                <div><code>{proposal.proposal_id}</code> · source {proposal.proposal_source || '—'} · evaluation <code>{proposal.hypothesis_evaluation_id || '—'}</code></div>
              </div>
            ))}
            <Link to="/marketops/algorithms" className="text-[11px] text-brand-700 underline">Open proposal review in Algorithms</Link>
          </div>
        ) : <p className="text-[11px] text-gray-400">No source-aware proposal for this exact hypothesis version.</p>}
        <p className="mt-1 text-[11px] text-gray-400">Proposal review remains in Algorithms. Hypothesis materialization is unsupported and no action is available here.</p>
      </Section>

      <Section title="Audit">
        <div className="text-[11px] text-gray-500">
          <div>owner {definition.owner || '—'} · definition updated {definition.updated_at ? formatUtc(definition.updated_at) : '—'}</div>
          {evaluation ? <><div>evaluation <code className="break-all">{evaluation.evaluationId}</code> <CopyButton value={evaluation.evaluationId} /></div><div>run <code className="break-all">{evaluation.evaluationRunId || '—'}</code> · key <code>{evaluation.deterministicKey || '—'}</code></div></> : null}
          {exactCalibration ? <div>calibration summary <code className="break-all">{exactCalibration.summary.summary_id}</code></div> : null}
        </div>
        <JsonViewer label="Definition rules JSON" value={{
          required_features: definition.required_features,
          required_transitions: definition.required_transitions,
          eligibility_expression: definition.eligibility_expression,
          trigger_expression: definition.trigger_expression,
          persistence_rule: definition.persistence_rule,
          corroboration_rule: definition.corroboration_rule,
          invalidation_rule: definition.invalidation_rule,
          expected_outcomes: definition.expected_outcomes,
          scoring_config: definition.scoring_config,
          calibration_policy: definition.calibration_policy,
        }} />
      </Section>
    </div>
  );
}

function RequirementList({
  title,
  requirements,
  rows,
}: {
  title: string;
  requirements: ReturnType<typeof parseHypothesisRequirements>;
  rows: Array<MarketOpsFeatureObservationView | MarketOpsStateTransitionView>;
}) {
  return (
    <div>
      <div className="text-[11px] font-medium text-gray-600">{title}</div>
      {requirements.length ? <div className="space-y-0.5">
        {requirements.map((requirement, index) => {
          const matches = rows.filter((row) => requirementMatches(requirement, row.featureKey, row.dimensions));
          const usable = matches.some((row) => row.qualityState.startsWith('usable'));
          return (
            <div key={`${requirement.featureKey}:${index}`} className="flex items-center justify-between gap-2 text-[11px]">
              <code className="text-gray-600">{requirement.featureKey}{Object.keys(requirement.dimensions).length ? ` ${stableDim(requirement.dimensions)}` : ''}</code>
              <Badge tone={qualityStateStyle(usable ? 'usable' : matches.length ? 'unusable' : 'missing')}>{usable ? 'present' : matches.length ? 'unusable' : 'missing'}</Badge>
            </div>
          );
        })}
      </div> : <p className="text-[11px] text-gray-400">No registered requirements.</p>}
    </div>
  );
}

function HypothesisEvidenceRow({ evidenceId }: { evidenceId: string }) {
  const [open, setOpen] = useState(false);
  const evidenceQ = useMarketOpsEvidence(open ? evidenceId : null);
  return (
    <div className="rounded border border-gray-200">
      <button type="button" className="flex w-full items-center justify-between gap-2 p-2 text-left text-[11px]" onClick={() => setOpen((value) => !value)}>
        <span className="flex items-center gap-1">{open ? <ChevronDown size={12} /> : <ChevronRight size={12} />}<code>{evidenceId}</code></span>
        <CopyButton value={evidenceId} />
      </button>
      {open ? <div className="border-t border-gray-100 p-2">
        {evidenceQ.isLoading ? <span className="text-[11px] text-gray-400">Loading evidence...</span> :
         evidenceQ.isError ? <span className="text-[11px] text-red-600">Evidence unavailable.</span> :
         evidenceQ.data?.evidence ? (
          <div className="space-y-1 text-[11px] text-gray-600">
            <div>{evidenceQ.data.evidence.statement || evidenceQ.data.evidence.evidence_type}</div>
            <div>domain {evidenceQ.data.evidence.domain || '—'} · quality {formatNullableNumber(evidenceQ.data.evidence.quality_score)}</div>
            <JsonViewer label="Evidence payload JSON" value={evidenceQ.data.evidence.evidence_payload} />
          </div>
        ) : null}
      </div> : null}
    </div>
  );
}

function CalibrationDetail({ report }: { report: ReturnType<typeof parseHypothesisCalibrationReport> }) {
  if (!report.selectedVersion) return <p className="text-[11px] text-gray-400">No exact-version calibration report.</p>;
  const overall = report.selectedVersion.overall;
  return (
    <div className="space-y-2 text-xs text-gray-700">
      <div className="flex flex-wrap items-center gap-2">
        {overall.belowMinimumSampleSize ? <Badge tone={qualityStateStyle('degraded')}>Below minimum sample</Badge> : <Badge tone={qualityStateStyle('usable')}>Calibration available</Badge>}
        <span className="text-[11px] text-gray-500">promotion_allowed={String(report.promotionAllowed)} · mode {report.mode || '—'} · minimum {report.minimumSampleSize}</span>
      </div>
      <div className="text-[11px] text-gray-500">window {report.windowStart || '—'} to {report.windowEnd || '—'} · as-of {report.asOf || '—'} · symbols {report.symbols.join(', ') || '—'}</div>
      <div className="grid grid-cols-2 gap-1 text-[11px] text-gray-600 md:grid-cols-3">
        <span>evaluations {overall.evaluations}</span><span>eligible {overall.eligibleStates}</span><span>triggers {overall.triggers}</span>
        <span>samples {overall.independentSamples}</span><span>matured {overall.maturedOutcomeSamples}</span><span>hit rate {formatNullablePercent(overall.directionalHitRate)}</span>
        <span>mean ret {formatNullableNumber(overall.meanForwardReturn)}</span><span>median ret {formatNullableNumber(overall.medianForwardReturn)}</span>
        <span>mean fav {formatNullableNumber(overall.meanFavorableExcursion)}</span><span>median fav {formatNullableNumber(overall.medianFavorableExcursion)}</span>
        <span>mean adv {formatNullableNumber(overall.meanAdverseExcursion)}</span><span>median adv {formatNullableNumber(overall.medianAdverseExcursion)}</span>
        <span>drawdown {formatNullablePercent(overall.drawdownIncidence)}</span><span>realized vol Δ {formatNullableNumber(overall.meanRealizedVolChange)}</span><span>calib error {formatNullableNumber(overall.calibrationError)}</span>
      </div>
      <CalibrationSegments title="By horizon" segments={report.selectedVersion.byHorizon} />
      <CalibrationSegments title="By asset" segments={report.selectedVersion.byAsset} />
      <CalibrationSegments title="By year" segments={report.selectedVersion.byYear} />
      <CalibrationSegments title="By volatility regime" segments={report.selectedVersion.byVolatilityRegime} />
      <CalibrationSegments title="By earnings window" segments={report.selectedVersion.byEarningsWindow} />
      {report.warnings.length ? <ul className="list-disc pl-4 text-[11px] text-amber-700">{report.warnings.map((warning) => <li key={warning}>{warning}</li>)}</ul> : null}
      {report.comparison ? <JsonViewer label="Comparison JSON" value={report.comparison} /> : null}
      {report.walkForward.length ? <JsonViewer label="Walk-forward folds JSON" value={report.walkForward} /> : null}
    </div>
  );
}

function CalibrationSegments({ title, segments }: { title: string; segments: CalibrationSegmentMap }) {
  const rows = Object.entries(segments);
  if (!rows.length) return null;
  return (
    <details>
      <summary className="cursor-pointer text-[11px] font-medium text-gray-600">{title} ({rows.length})</summary>
      <div className="mt-1 overflow-x-auto">
        <table className="min-w-full text-[11px] text-gray-600">
          <thead><tr><th className="pr-3 text-left">Segment</th><th className="pr-3 text-right">Samples</th><th className="pr-3 text-right">Matured</th><th className="pr-3 text-right">Hit rate</th><th className="text-right">Mean return</th></tr></thead>
          <tbody>{rows.map(([key, metrics]) => <tr key={key}><td className="pr-3">{key}</td><td className="pr-3 text-right">{metrics.independentSamples}</td><td className="pr-3 text-right">{metrics.maturedOutcomeSamples}</td><td className="pr-3 text-right">{formatNullablePercent(metrics.directionalHitRate)}</td><td className="text-right">{formatNullableNumber(metrics.meanForwardReturn)}</td></tr>)}</tbody>
        </table>
      </div>
    </details>
  );
}

// --- small shared helpers ------------------------------------------------------

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <div className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-gray-500">{title}</div>
      {children}
    </div>
  );
}

function Badge({ tone, children }: { tone: string; children: React.ReactNode }) {
  return <span className={`inline-flex shrink-0 items-center whitespace-nowrap rounded border px-1.5 py-0.5 text-[11px] font-medium ${tone}`}>{children}</span>;
}

function dateOnly(iso: string): string {
  return (iso || '').slice(0, 10);
}

// Newest revision per session_date (by as_of_time then deterministic id).
function dedupeNewestRevision(states: MarketOpsMarketStateView[]): MarketOpsMarketStateView[] {
  const bySession = new Map<string, MarketOpsMarketStateView>();
  for (const s of states) {
    const key = dateOnly(s.sessionDate);
    if (!key) continue;
    const cur = bySession.get(key);
    if (!cur || s.asOfTime > cur.asOfTime || (s.asOfTime === cur.asOfTime && s.deterministicKey > cur.deterministicKey)) {
      bySession.set(key, s);
    }
  }
  return Array.from(bySession.values()).sort((a, b) => b.sessionDate.localeCompare(a.sessionDate));
}

function domainBucket(domain: string): (typeof DOMAIN_BUCKETS)[number] {
  const d = (domain || '').toLowerCase();
  if (d.includes('underlying')) return 'underlying';
  if (d.includes('volatil')) return 'volatility';
  if (d.includes('position')) return 'positioning';
  if (d.includes('premium')) return 'premium';
  if (d.includes('liquid') || d.includes('quality')) return 'liquidity_quality';
  if (d.includes('event')) return 'event';
  return 'other';
}

function domainLabel(bucket: string): string {
  switch (bucket) {
    case 'underlying': return 'Underlying';
    case 'volatility': return 'Volatility & surface';
    case 'positioning': return 'Option positioning';
    case 'premium': return 'Premium';
    case 'liquidity_quality': return 'Liquidity & quality';
    case 'event': return 'Event context';
    default: return 'Other';
  }
}

function obsSignature(o: MarketOpsFeatureObservationView): string {
  return `${o.featureKey}:${o.featureVersion}:${stableDim(o.dimensions)}`;
}

function stableDim(d: unknown): string {
  if (!d || typeof d !== 'object' || Array.isArray(d)) return '';
  try {
    return JSON.stringify(Object.keys(d as Record<string, unknown>).sort().map((k) => [k, (d as Record<string, unknown>)[k]]));
  } catch {
    return '';
  }
}

function uniqueStrings(values: string[]): string[] {
  return Array.from(new Set(values.filter(Boolean))).sort((a, b) => a.localeCompare(b));
}

function obsValue(o: MarketOpsFeatureObservationView): number | string | boolean | null {
  if (o.numericValue !== null) return o.numericValue;
  if (o.textValue !== null) return o.textValue;
  if (o.booleanValue !== null) return o.booleanValue;
  return null;
}

function formatObsValue(v: number | string | boolean | null, unit: string | null): string {
  if (v === null) return '—';
  if (typeof v === 'number') return `${v.toFixed(4)}${unit ? ` ${unit}` : ''}`;
  if (typeof v === 'boolean') return v ? 'true' : 'false';
  return v || '—';
}

function formatDelta(d: number): string {
  return `${d >= 0 ? '+' : ''}${d.toFixed(4)}`;
}
