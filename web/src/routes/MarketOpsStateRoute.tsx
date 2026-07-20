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
  qualityStateStyle,
  formatNullableNumber,
  formatNullablePercent,
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
        onSession={(d) => setSearch({ session_date: d })}
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
          onSelectHypothesis={(key, version) => setSearch({ hypothesis_key: key, hypothesis_version: version })}
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
            completeness <strong>{selectedState.featureCount}/{selectedState.requiredFeatureCount}</strong> ({formatNullablePercent(selectedState.completenessRatio)})
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
          <span>completeness {selectedState.featureCount}/{selectedState.requiredFeatureCount} ({formatNullablePercent(selectedState.completenessRatio)})</span>
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

// Surface: canonical seven-cell grid from lineage observations, by dimensions.
function SurfaceTab({ selectedState }: { tenantId: string; selectedState: MarketOpsMarketStateView }) {
  const lineageQ = useMarketOpsMarketStateLineage(selectedState.marketStateId);
  const observations = (lineageQ.data?.lineage.feature_observations ?? []).map(summarizeMarketOpsFeatureObservation);
  const grouped = groupObservationsBySurfaceCell(observations);

  return (
    <div className="space-y-2">
      {lineageQ.isLoading ? <LoadingState label="Loading state lineage..." /> : lineageQ.isError ? <ErrorState error={lineageQ.error} /> : (
        <div className="grid grid-cols-1 gap-2 md:grid-cols-2 lg:grid-cols-3">
          {SURFACE_CELLS.map((cell) => {
            const cellObs = grouped.get(cell.id) ?? [];
            return (
              <div key={cell.id} className="rounded border border-gray-200 bg-white p-2" aria-label={`Surface cell ${cell.label}`}>
                <div className="mb-1 text-xs font-semibold text-gray-700">{cell.label}</div>
                {cellObs.length ? (
                  <div className="space-y-0.5">
                    {cellObs.map((o) => (
                      <div key={o.featureObservationId} className="flex items-center justify-between gap-2 text-[11px]">
                        <code className="break-all text-gray-600" title={`${o.featureKey} v${o.featureVersion}`}>{o.featureKey}</code>
                        <span className="flex items-center gap-1">
                          <span className="text-gray-800">{formatObsValue(obsValue(o), null)}</span>
                          <Badge tone={qualityStateStyle(o.qualityState)}>{o.qualityState ? o.qualityState[0].toUpperCase() : '—'}</Badge>
                        </span>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="text-[11px] text-gray-400">Unavailable</div>
                )}
              </div>
            );
          })}
        </div>
      )}
      <p className="text-[11px] text-gray-400">Cells are mapped by exact dimensions (option_type/target_dte/target_delta), never by array order. Unavailable cells are operationally meaningful.</p>
    </div>
  );
}

// Transitions: bounded symbol/window with a material presentation filter.
function TransitionsTab({ tenantId, symbol, selectedState }: { tenantId: string; symbol: string; selectedState: MarketOpsMarketStateView }) {
  const [showAll, setShowAll] = useState(false);
  const transitionsQ = useMarketOpsStateTransitions(
    { tenant_id: tenantId, symbol, current_state_id: selectedState.marketStateId, limit: 200 },
    true,
  );
  const all = (transitionsQ.data?.transitions ?? []).map(summarizeMarketOpsStateTransition);
  const material = all.filter(isMaterialTransition);
  const rows = showAll ? all : material;

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="text-[11px] text-gray-500">Showing {rows.length} / {all.length} returned transitions</div>
        <button type="button" onClick={() => setShowAll((v) => !v)} className={`${inputCls} bg-white`}>
          {showAll ? 'Show material only' : 'Show all returned transitions'}
        </button>
      </div>
      {transitionsQ.isLoading ? <LoadingState label="Loading transitions..." /> : transitionsQ.isError ? <ErrorState error={transitionsQ.error} /> : rows.length ? (
        <div className="overflow-x-auto rounded border border-gray-200 bg-white">
          <table className="min-w-full divide-y divide-gray-200 text-sm">
            <thead className="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500">
              <tr>
                <th className="px-2 py-1">Feature</th>
                <th className="px-2 py-1">Type</th>
                <th className="px-2 py-1">Lookback</th>
                <th className="px-2 py-1">Current</th>
                <th className="px-2 py-1">Baseline</th>
                <th className="px-2 py-1">Δ</th>
                <th className="px-2 py-1">Z / pct</th>
                <th className="px-2 py-1">Direction</th>
                <th className="px-2 py-1">Quality</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {rows.map((t) => (
                <tr key={t.transitionId} className="align-top">
                  <td className="px-2 py-1"><code className="break-all text-[11px] text-gray-700">{t.featureKey}</code></td>
                  <td className="whitespace-nowrap px-2 py-1 text-xs text-gray-700">{t.transitionType || '—'}</td>
                  <td className="whitespace-nowrap px-2 py-1 text-xs text-gray-600">{t.lookbackSessions ?? '—'}</td>
                  <td className="whitespace-nowrap px-2 py-1 text-xs text-gray-800">{formatNullableNumber(t.currentValue)}</td>
                  <td className="whitespace-nowrap px-2 py-1 text-xs text-gray-600">{formatNullableNumber(t.baselineValue)}</td>
                  <td className="whitespace-nowrap px-2 py-1 text-xs text-gray-700">{formatNullableNumber(t.transitionValue)}</td>
                  <td className="whitespace-nowrap px-2 py-1 text-xs text-gray-700">{t.zscore !== null ? formatNullableNumber(t.zscore) : formatNullablePercent(t.percentile)}</td>
                  <td className="whitespace-nowrap px-2 py-1 text-xs text-gray-600">{t.direction || '—'}</td>
                  <td className="px-2 py-1"><Badge tone={qualityStateStyle(t.qualityState)}>{t.qualityState || '—'}</Badge></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <EmptyState message={all.length ? 'No transitions hidden by the material filter.' : 'No transitions returned for this scope.'} />
      )}
    </div>
  );
}

// Hypotheses: master/detail over definitions + the selected state's evaluations + calibration.
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
  const defs = defsQ.data?.hypotheses ?? [];
  const evalsQ = useMarketOpsHypothesisEvaluations(
    { tenant_id: tenantId, market_state_id: selectedState.marketStateId, limit: 200 },
    true,
  );
  const evals = (evalsQ.data?.hypothesis_evaluations ?? []).map(summarizeMarketOpsHypothesisEvaluation);
  const evalByKey = new Map(evals.map((e) => [`${e.hypothesisKey}:${e.hypothesisVersion}`, e]));
  const selected = hypothesisKey
    ? defs.find((d) => d.hypothesis_key === hypothesisKey && d.hypothesis_version === hypothesisVersion) ?? null
    : (defs[0] ?? null);

  return (
    <div className="flex flex-col gap-3 lg:flex-row">
      <div className={`${selected ? 'hidden lg:block' : ''} lg:w-2/5 lg:min-w-[360px]`}>
        <div className="space-y-1">
          {defs.length === 0 ? (
            <EmptyState message="No hypothesis definitions." />
          ) : (
            defs.map((d) => {
              const ev = evalByKey.get(`${d.hypothesis_key}:${d.hypothesis_version}`);
              const isSel = selected?.hypothesis_key === d.hypothesis_key && selected?.hypothesis_version === d.hypothesis_version;
              return (
                <button
                  key={`${d.hypothesis_key}:${d.hypothesis_version}`}
                  type="button"
                  onClick={() => onSelectHypothesis(d.hypothesis_key, d.hypothesis_version)}
                  className={`w-full rounded border p-2 text-left ${isSel ? 'border-brand-400 bg-brand-50' : 'border-gray-200 bg-white hover:bg-gray-50'}`}
                >
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-xs font-medium text-gray-800">{d.title || d.hypothesis_key}</span>
                    {ev ? <Badge tone={qualityStateStyle(ev.triggered ? 'usable' : 'degraded')}>{ev.triggered ? 'triggered' : ev.eligible ? 'eligible' : 'ineligible'}</Badge> : <span className="text-[11px] text-gray-400">not evaluated</span>}
                  </div>
                  <div className="text-[11px] text-gray-500">{d.hypothesis_key} v{d.hypothesis_version} · {d.domain || '—'}</div>
                </button>
              );
            })
          )}
        </div>
      </div>
      <div className={`${selected ? '' : 'hidden lg:block'} flex-1`}>
        {selected ? (
          <HypothesisDetail
            tenantId={tenantId}
            definition={selected}
            evaluation={evalByKey.get(`${selected.hypothesis_key}:${selected.hypothesis_version}`) ?? null}
          />
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
}: {
  tenantId: string;
  definition: MarketOpsHypothesisDefinition;
  evaluation: MarketOpsHypothesisEvaluationStateView | null;
}) {
  // Calibration: latest exact-version G145 report for this hypothesis detector.
  const calibQ = useMarketOpsBacktestCalibrationSummaries({
    tenant_id: tenantId,
    app_id: 'marketops',
    domain: 'market_data',
    use_case: 'daily_market_surveillance',
    source_id: 'marketops.research_ledgers',
    dataset: 'hypothesis_evaluations',
    detector_id: `marketops.hypothesis.${definition.hypothesis_key.toLowerCase()}`,
    limit: 5,
  });
  const latestSummary = calibQ.data?.calibration_summaries?.[0] ?? null;
  const report = latestSummary
    ? parseHypothesisCalibrationReport(latestSummary.parameters, definition.hypothesis_key, definition.hypothesis_version)
    : null;

  return (
    <div className="rounded border border-gray-200 bg-white p-3 space-y-3">
      <div>
        <div className="text-xs font-semibold text-gray-700">{definition.title || definition.hypothesis_key}</div>
        <div className="text-[11px] text-gray-500">
          {definition.hypothesis_key} v{definition.hypothesis_version} · {definition.domain || '—'} · {definition.direction || '—'} · lifecycle {definition.lifecycle_status || '—'}
        </div>
        {definition.description ? <p className="mt-1 text-xs text-gray-700">{definition.description}</p> : null}
      </div>

      <Section title="Current evaluation">
        {evaluation ? (
          <div className="space-y-1 text-xs text-gray-700">
            <div className="flex flex-wrap items-center gap-2">
              <Badge tone={qualityStateStyle(evaluation.eligible ? 'usable' : 'degraded')}>{evaluation.eligible ? 'eligible' : 'ineligible'}</Badge>
              {evaluation.triggered ? <Badge tone={qualityStateStyle('usable')}>triggered</Badge> : null}
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
                {evaluation.reasonCodes.map((r) => <span key={r} className="rounded border border-gray-200 bg-gray-50 px-1.5 py-0.5 text-[11px] text-gray-600" title={r}>{r}</span>)}
              </div>
            ) : null}
          </div>
        ) : (
          <p className="text-[11px] text-gray-400">Not evaluated for this state.</p>
        )}
      </Section>

      <Section title="Historical calibration">
        {calibQ.isLoading ? <div className="text-[11px] text-gray-400">Loading calibration...</div> :
         calibQ.isError ? <div className="text-[11px] text-red-600">Calibration unavailable.</div> :
         !latestSummary ? <p className="text-[11px] text-gray-400">No exact-version calibration report.</p> :
         !report?.valid ? <p className="text-[11px] text-amber-700">{report?.incompatibleReason || 'Calibration report is not compatible with marketops.hypothesis_calibration.v1'}</p> :
         report.selectedVersion ? (
          <div className="space-y-1 text-xs text-gray-700">
            <div className="flex flex-wrap items-center gap-2">
              {report.selectedVersion.overall.belowMinimumSampleSize ? <Badge tone={qualityStateStyle('degraded')}>Below minimum sample</Badge> : <Badge tone={qualityStateStyle('usable')}>Calibration available</Badge>}
              {!report.promotionAllowed ? <span className="text-[11px] text-gray-500">promotion not allowed</span> : null}
            </div>
            <div className="grid grid-cols-2 gap-1 text-[11px] text-gray-600 md:grid-cols-3">
              <span>samples {report.selectedVersion.overall.independentSamples}</span>
              <span>matured {report.selectedVersion.overall.maturedOutcomeSamples}</span>
              <span>hit rate {formatNullablePercent(report.selectedVersion.overall.directionalHitRate)}</span>
              <span>mean ret {formatNullableNumber(report.selectedVersion.overall.meanForwardReturn)}</span>
              <span>median ret {formatNullableNumber(report.selectedVersion.overall.medianForwardReturn)}</span>
              <span>fav exc {formatNullableNumber(report.selectedVersion.overall.meanFavorableExcursion)}</span>
              <span>adv exc {formatNullableNumber(report.selectedVersion.overall.meanAdverseExcursion)}</span>
              <span>drawdown {formatNullablePercent(report.selectedVersion.overall.drawdownIncidence)}</span>
              <span>calib err {formatNullableNumber(report.selectedVersion.overall.calibrationError)}</span>
            </div>
            {report.warnings.length ? (
              <ul className="list-disc pl-4 text-[11px] text-amber-700">{report.warnings.map((w) => <li key={w}>{w}</li>)}</ul>
            ) : null}
          </div>
        ) : <p className="text-[11px] text-gray-400">No exact-version calibration report.</p>}
      </Section>

      <Section title="Audit">
        <div className="text-[11px] text-gray-500">
          <div>owner {definition.owner || '—'} · approved by {definition.approved_by || '—'}</div>
          {evaluation ? <><div>evaluation <code className="break-all">{evaluation.evaluationId}</code></div><div>run <code className="break-all">{evaluation.evaluationRunId || '—'}</code></div></> : null}
          {latestSummary ? <div>calibration summary <code className="break-all">{latestSummary.summary_id}</code></div> : null}
        </div>
      </Section>
    </div>
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
