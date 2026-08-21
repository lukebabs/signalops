import { Fragment, useState } from 'react';
import { useQuery } from '@tanstack/react-query';

import { api } from '../api/client';
import { useTenant } from '../auth/session';
import { EmptyState, ErrorState, LoadingState } from '../components/States';
import { formatPercent, formatUtc } from '../lib/format';
import type { MarketOpsSignalAssuranceEffectivenessObservation } from '../types';

const pct = (value?: number) => value == null || !Number.isFinite(value) ? '-' : formatPercent(value);
const label = (value: string) => value.replace(/_/g, ' ').replace(/\b\w/g, (letter: string) => letter.toUpperCase());

export function MarketOpsSignalAssuranceDrilldownPanel() {
  const tenantId = useTenant();
  const [source, setSource] = useState('LEGACY');
  const [dimension, setDimension] = useState('benchmark_coverage');
  const [cohort, setCohort] = useState('');
  const [selected, setSelected] = useState<MarketOpsSignalAssuranceEffectivenessObservation | null>(null);
  const cohorts = useQuery({ queryKey: ['saf-drilldown-cohorts', tenantId, source, dimension], queryFn: () => api.getMarketOpsSignalAssuranceEffectiveness(tenantId, source, dimension), staleTime: 30_000 });
  const observations = useQuery({ queryKey: ['saf-drilldown-observations', tenantId, source, dimension, cohort], queryFn: () => api.listMarketOpsSignalAssuranceEffectivenessObservations(tenantId, source, dimension, cohort), enabled: Boolean(cohort), staleTime: 30_000 });
  const opportunity = useQuery({ queryKey: ['saf-drilldown-opportunity', tenantId, selected?.reference_id], queryFn: () => api.getMarketOpsOpportunity(selected?.reference_id || '', tenantId), enabled: selected?.evidence_source === 'LEGACY' && Boolean(selected?.reference_id), staleTime: 30_000 });
  const assertion = useQuery({ queryKey: ['saf-drilldown-assertion', tenantId, selected?.reference_id], queryFn: () => api.getMarketOpsSignalAssuranceAssertion(selected?.reference_id || '', tenantId), enabled: selected?.evidence_source === 'SAF' && Boolean(selected?.reference_id), staleTime: 30_000 });
  const chooseCohort = (value: string) => {
    setSelected(null);
    setCohort((current) => current === value ? '' : value);
  };
  const resetScope = (nextSource: string, nextDimension = dimension) => {
    setSource(nextSource);
    setDimension(nextDimension);
    setCohort('');
    setSelected(null);
  };

  return <section className="space-y-3 rounded border border-gray-200 bg-white p-3 dark:border-gray-700 dark:bg-gray-900">
    <div>
      <h2 className="text-base font-semibold text-gray-900 dark:text-gray-100">Analyst drill-down</h2>
      <p className="max-w-3xl text-xs text-gray-500 dark:text-gray-400">Trace any effectiveness cohort to every included terminal observation, then open its immutable source audit. Historical outcomes link to the opportunity and its input signal/evidence identifiers; SAF records link to the assertion, resolved contract, baseline, and provenance.</p>
    </div>
    <div className="flex flex-wrap items-end gap-3 rounded border border-gray-200 bg-gray-50 p-3 dark:border-gray-700 dark:bg-gray-800">
      <label className="text-xs font-medium text-gray-700 dark:text-gray-300">Evidence
        <select value={source} onChange={(event) => resetScope(event.target.value)} className="mt-1 block rounded border border-gray-300 bg-white px-2 py-1.5 text-sm text-gray-900 dark:border-gray-600 dark:bg-gray-950 dark:text-gray-100"><option value="LEGACY">Historical outcomes</option><option value="SAF">SAF validated</option></select>
      </label>
      <label className="text-xs font-medium text-gray-700 dark:text-gray-300">Cohort dimension
        <select value={dimension} onChange={(event) => resetScope(source, event.target.value)} className="mt-1 block rounded border border-gray-300 bg-white px-2 py-1.5 text-sm text-gray-900 dark:border-gray-600 dark:bg-gray-950 dark:text-gray-100"><option value="direction">Direction</option><option value="horizon">Horizon</option><option value="confidence_band">Confidence band</option><option value="algorithm_version">Algorithm / version</option><option value="signal_type">Signal type</option><option value="benchmark_coverage">Benchmark coverage</option></select>
      </label>
      <span className="pb-1 text-xs text-gray-500 dark:text-gray-400">Click a cohort row to expand observations inline.</span>
    </div>
    {cohorts.isLoading ? <LoadingState label="Loading cohorts..." /> : cohorts.isError ? <ErrorState error={cohorts.error} /> : <div className="overflow-x-auto rounded border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-950">
      <table className="min-w-full text-sm">
        <thead className="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500 dark:bg-gray-800 dark:text-gray-400"><tr><th className="px-3 py-2">Cohort</th><th className="px-3 py-2">Accuracy</th><th className="px-3 py-2">Sample</th><th className="px-3 py-2">State</th></tr></thead>
        <tbody className="divide-y divide-gray-100 dark:divide-gray-800">{(cohorts.data?.effectiveness ?? []).map((row) => {
          const open = cohort === row.dimension_value;
          return <Fragment key={row.dimension_value}>
            <tr role="button" tabIndex={0} onClick={() => chooseCohort(row.dimension_value)} onKeyDown={(event) => { if (event.key === 'Enter' || event.key === ' ') chooseCohort(row.dimension_value); }} className={open ? 'cursor-pointer bg-brand-50 text-gray-900 dark:bg-brand-950/30 dark:text-gray-100' : 'cursor-pointer text-gray-900 hover:bg-gray-50 dark:text-gray-100 dark:hover:bg-gray-800/70'}>
              <td className="px-3 py-2 font-mono text-xs">{row.dimension_value}</td>
              <td className="px-3 py-2 font-semibold">{pct(row.directional_accuracy)}</td>
              <td className="px-3 py-2 text-xs text-gray-600 dark:text-gray-400">{row.directional_hits}/{row.sample_size}{row.exploratory ? ' - exploratory' : ''}</td>
              <td className="px-3 py-2 text-xs font-medium text-brand-700 dark:text-brand-200">{open ? 'Expanded' : 'Click to inspect'}</td>
            </tr>
            {open ? <tr className="bg-brand-50/40 dark:bg-brand-950/20"><td colSpan={4} className="p-0"><ExpandedObservationCohort
              cohort={cohort}
              observations={observations.data?.observations ?? []}
              loading={observations.isLoading}
              error={observations.error}
              selected={selected}
              setSelected={setSelected}
              opportunity={opportunity}
              assertion={assertion}
              close={() => { setCohort(''); setSelected(null); }}
            /></td></tr> : null}
          </Fragment>;
        })}</tbody>
      </table>
      {(cohorts.data?.effectiveness.length ?? 0) === 0 ? <EmptyState message="No cohorts match this evidence class." /> : null}
    </div>}
  </section>;
}

function ExpandedObservationCohort({ cohort, observations, loading, error, selected, setSelected, opportunity, assertion, close }: {
  cohort: string;
  observations: MarketOpsSignalAssuranceEffectivenessObservation[];
  loading: boolean;
  error: unknown;
  selected: MarketOpsSignalAssuranceEffectivenessObservation | null;
  setSelected: (row: MarketOpsSignalAssuranceEffectivenessObservation | null) => void;
  opportunity: ReturnType<typeof useQuery>;
  assertion: ReturnType<typeof useQuery>;
  close: () => void;
}) {
  return <section className="space-y-2 border-t border-brand-200 p-3 dark:border-brand-900">
    <div className="flex flex-wrap items-start justify-between gap-3">
      <div>
        <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100">Included observations: {cohort}</h3>
        <p className="text-xs text-gray-600 dark:text-gray-400">Only complete terminal observations that contribute to this effectiveness figure appear here. Click an observation row to expand its audit trail inline.</p>
      </div>
      <button type="button" onClick={(event) => { event.stopPropagation(); close(); }} className="rounded border border-gray-300 bg-white px-2 py-1 text-xs font-medium text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:bg-gray-950 dark:text-gray-200 dark:hover:bg-gray-800">Close observations</button>
    </div>
    {loading ? <LoadingState label="Loading included observations..." /> : error ? <ErrorState error={error} /> : <div className="overflow-x-auto rounded border border-brand-200 bg-white dark:border-brand-900 dark:bg-gray-950">
      <table className="min-w-full text-sm">
        <thead className="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500 dark:bg-gray-800 dark:text-gray-400"><tr><th className="px-3 py-2">Signal</th><th className="px-3 py-2">Origin</th><th className="px-3 py-2">Outcome</th><th className="px-3 py-2">Match</th><th className="px-3 py-2">Aligned move</th><th className="px-3 py-2">Benchmarks</th><th className="px-3 py-2">Audit</th></tr></thead>
        <tbody className="divide-y divide-gray-100 dark:divide-gray-800">{observations.map((row) => {
          const open = selected?.observation_id === row.observation_id;
          return <Fragment key={row.observation_id}>
            <tr role="button" tabIndex={0} onClick={() => setSelected(open ? null : row)} onKeyDown={(event) => { if (event.key === 'Enter' || event.key === ' ') setSelected(open ? null : row); }} className={open ? 'cursor-pointer bg-yellow-50 text-gray-900 dark:bg-yellow-950/30 dark:text-gray-100' : 'cursor-pointer text-gray-900 hover:bg-gray-50 dark:text-gray-100 dark:hover:bg-gray-800/70'}>
              <td className="px-3 py-2"><div className="font-mono font-semibold">{row.symbol}</div><div className="text-xs text-gray-500 dark:text-gray-400">{label(row.direction)} - {row.horizon_sessions} session{row.horizon_sessions === 1 ? '' : 's'}</div></td>
              <td className="px-3 py-2 text-xs text-gray-600 dark:text-gray-400">{row.origin_at ? formatUtc(row.origin_at) : '-'}</td>
              <td className="px-3 py-2 text-xs text-gray-600 dark:text-gray-400">{row.outcome_at ? formatUtc(row.outcome_at) : '-'}</td>
              <td className={row.directional_hit ? 'px-3 py-2 text-xs font-semibold text-green-700 dark:text-green-300' : 'px-3 py-2 text-xs font-semibold text-red-700 dark:text-red-300'}>{row.directional_hit ? 'Matched' : 'Missed'}</td>
              <td className="px-3 py-2 text-xs">{pct(row.directional_return)}</td>
              <td className="px-3 py-2 text-xs text-gray-600 dark:text-gray-400"><div>SPY: {row.broad_market_benchmark_state || 'not recorded'}</div><div>Sector: {row.sector_benchmark_state || 'not recorded'}</div></td>
              <td className="px-3 py-2 text-xs font-medium text-brand-700 dark:text-brand-200">{open ? 'Hide audit' : 'Open audit'}</td>
            </tr>
            {open ? <tr className="bg-yellow-50/40 dark:bg-yellow-950/20"><td colSpan={7} className="p-0"><ObservationAudit selected={row} opportunity={opportunity} assertion={assertion} onClose={() => setSelected(null)} /></td></tr> : null}
          </Fragment>;
        })}</tbody>
      </table>
      {observations.length === 0 ? <EmptyState message="No included observations in this cohort." /> : null}
    </div>}
  </section>;
}

function ObservationAudit({ selected, opportunity, assertion, onClose }: { selected: MarketOpsSignalAssuranceEffectivenessObservation; opportunity: ReturnType<typeof useQuery>; assertion: ReturnType<typeof useQuery>; onClose: () => void }) {
  return <section className="space-y-2 border-t border-yellow-200 p-3 dark:border-yellow-900">
    <div className="flex items-start justify-between gap-3">
      <div>
        <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100">Observation audit: {selected.symbol}</h3>
        <p className="text-xs text-gray-600 dark:text-gray-400">{selected.evidence_source === 'SAF' ? 'Confirmed SAF assertion audit.' : 'Historical opportunity outcome audit; not an SAF assertion.'}</p>
      </div>
      <button type="button" onClick={(event) => { event.stopPropagation(); onClose(); }} className="rounded border border-gray-300 bg-white px-2 py-1 text-xs font-medium text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:bg-gray-950 dark:text-gray-200 dark:hover:bg-gray-800">Close audit</button>
    </div>
    <dl className="grid gap-2 text-xs sm:grid-cols-2 lg:grid-cols-4">
      <div><dt className="text-gray-500 dark:text-gray-400">Signal score</dt><dd className="font-medium text-gray-900 dark:text-gray-100">{pct(selected.signal_score)}</dd></div>
      <div><dt className="text-gray-500 dark:text-gray-400">Confidence</dt><dd className="font-medium text-gray-900 dark:text-gray-100">{pct(selected.confidence)}</dd></div>
      <div><dt className="text-gray-500 dark:text-gray-400">Raw return</dt><dd className="font-medium text-gray-900 dark:text-gray-100">{pct(selected.absolute_return)}</dd></div>
      <div><dt className="text-gray-500 dark:text-gray-400">Calculation</dt><dd className="font-mono text-gray-900 dark:text-gray-100">{selected.calculation_version || '-'}</dd></div>
    </dl>
    <details open><summary className="cursor-pointer text-xs font-medium text-brand-700 dark:text-brand-200">Observation record</summary><pre className="mt-2 max-h-64 overflow-auto rounded bg-white p-2 text-[10px] text-gray-700 dark:bg-gray-950 dark:text-gray-200">{JSON.stringify(selected, null, 2)}</pre></details>
    {selected.evidence_source === 'LEGACY' ? opportunity.isLoading ? <LoadingState label="Loading opportunity provenance..." /> : opportunity.isError ? <ErrorState error={opportunity.error} /> : <details open><summary className="cursor-pointer text-xs font-medium text-brand-700 dark:text-brand-200">Opportunity provenance and source identifiers</summary><pre className="mt-2 max-h-80 overflow-auto rounded bg-white p-2 text-[10px] text-gray-700 dark:bg-gray-950 dark:text-gray-200">{JSON.stringify((opportunity.data as { opportunity?: unknown } | undefined)?.opportunity, null, 2)}</pre></details> : assertion.isLoading ? <LoadingState label="Loading assertion audit..." /> : assertion.isError ? <ErrorState error={assertion.error} /> : <details open><summary className="cursor-pointer text-xs font-medium text-brand-700 dark:text-brand-200">Assertion baseline, provenance, and validation contract</summary><pre className="mt-2 max-h-80 overflow-auto rounded bg-white p-2 text-[10px] text-gray-700 dark:bg-gray-950 dark:text-gray-200">{JSON.stringify((assertion.data as { assertion?: unknown } | undefined)?.assertion, null, 2)}</pre></details>}
  </section>;
}
