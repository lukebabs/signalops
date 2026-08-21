import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';

import { api } from '../api/client';
import { useTenant } from '../auth/session';
import { EmptyState, ErrorState, LoadingState } from '../components/States';
import { RefreshButton } from '../components/RefreshButton';
import { ThemedEChart } from '../components/ThemedEChart';
import { formatPercent, formatUtc } from '../lib/format';
import type { MarketOpsSignalAssuranceEffectiveness, MarketOpsSignalAssuranceEffectivenessObservation } from '../types';

const pct = (value?: number) => value == null || !Number.isFinite(value) ? '-' : formatPercent(value);
const label = (value: string) => value.replace(/_/g, ' ').replace(/\b\w/g, (letter: string) => letter.toUpperCase());

export function MarketOpsSignalAssuranceEffectivenessPanel() {
  const tenantId = useTenant();
  const [source, setSource] = useState('');
  const [dimension, setDimension] = useState('benchmark_coverage');
  const [mode, setMode] = useState('');
  const overview = useQuery({
    queryKey: ['saf-effectiveness', tenantId, source, 'overall', mode],
    queryFn: () => api.getMarketOpsSignalAssuranceEffectiveness(tenantId, source, 'overall', mode),
    staleTime: 30_000,
  });
  const diagnostics = useQuery({
    queryKey: ['saf-effectiveness', tenantId, source, dimension, mode],
    queryFn: () => api.getMarketOpsSignalAssuranceEffectiveness(tenantId, source, dimension, mode),
    staleTime: 30_000,
  });
  const recommendations = useQuery({
    queryKey: ['saf-recommendations', tenantId, source, mode],
    queryFn: () => api.getMarketOpsSignalAssuranceRecommendations(tenantId, source, mode),
    staleTime: 30_000,
  });
  const progressionSource = source || 'LEGACY';
  const progression = useQuery({
    queryKey: ['saf-effectiveness-daily-progression', tenantId, progressionSource, mode],
    queryFn: () => api.listMarketOpsSignalAssuranceEffectivenessObservations(tenantId, progressionSource, 'overall', 'all', mode, 1500),
    staleTime: 30_000,
  });
  const refresh = () => {
    void overview.refetch();
    void diagnostics.refetch();
    void recommendations.refetch();
    void progression.refetch();
  };
  const loading = overview.isLoading || diagnostics.isLoading || recommendations.isLoading;
  const error = overview.error || diagnostics.error || recommendations.error;
  const rows = diagnostics.data?.effectiveness ?? [];

  return (
    <section className="space-y-3 rounded border border-gray-200 bg-white p-3">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="text-base font-semibold">Effectiveness & viability</h2>
          <p className="max-w-3xl text-xs text-gray-500">
            Benchmark coverage is shown first so unresolved sector mapping is explicit. Directional accuracy is one input, not a performance claim. Viability is a read-only research gate:
            insufficient evidence, missing benchmarks, or adverse outcome profiles cannot be treated as a pass.
          </p>
        </div>
        <RefreshButton onClick={refresh} loading={overview.isFetching || diagnostics.isFetching || recommendations.isFetching} />
      </div>
      <div className="flex flex-wrap items-end gap-3 rounded border border-gray-200 bg-gray-50 p-3">
        <label className="text-xs font-medium text-gray-700">Evidence
          <select value={source} onChange={(event) => setSource(event.target.value)} className="mt-1 block rounded border border-gray-300 bg-white px-2 py-1.5 text-sm">
            <option value="">All evidence classes</option><option value="SAF">SAF validated only</option><option value="LEGACY">Historical outcomes only</option>
          </select>
        </label>
        <label className="text-xs font-medium text-gray-700">Breakdown
          <select value={dimension} onChange={(event) => setDimension(event.target.value)} className="mt-1 block rounded border border-gray-300 bg-white px-2 py-1.5 text-sm">
            <option value="algorithm_version">Algorithm / version</option><option value="signal_type">Signal type</option><option value="direction">Direction</option><option value="confidence_band">Confidence band</option><option value="horizon">Evaluation horizon</option><option value="benchmark_coverage">Benchmark coverage</option>
          </select>
        </label>
        <label className="text-xs font-medium text-gray-700">SAF mode
          <select value={mode} onChange={(event) => setMode(event.target.value)} className="mt-1 block rounded border border-gray-300 bg-white px-2 py-1.5 text-sm">
            <option value="">All modes</option><option value="RESEARCH">Research</option><option value="LIVE">Live</option><option value="BACKTEST">Backtest</option>
          </select>
        </label>
      </div>
      {loading ? <LoadingState label="Calculating effectiveness..." /> : error ? <ErrorState error={error} /> : <>
        <div className="grid gap-3 md:grid-cols-2">{(overview.data?.effectiveness ?? []).map((row) => <SummaryCard key={row.evidence_source} row={row} />)}</div>
        <p className="rounded border border-amber-200 bg-amber-50 p-2 text-xs text-amber-800">{overview.data?.evidence_source_note}</p>
        {overview.data?.watchlist_context ? <p data-testid="saf-viability-cohort" className="rounded border border-blue-200 bg-blue-50 p-2 text-xs text-blue-900">Selected viability cohort: <strong>{overview.data.watchlist_context.list_name || "Tenant default"}</strong> - {overview.data.watchlist_context.member_count} assets. This scope is used for every effectiveness row below.</p> : null}
        <DailyProgressionChart observations={progression.data?.observations ?? []} evidenceSource={progressionSource} loading={progression.isLoading} error={progression.error} />
        <div className="overflow-x-auto rounded border border-gray-200">
          <table className="min-w-full text-sm">
            <thead className="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500"><tr>
              <th className="px-3 py-2">Evidence</th><th className="px-3 py-2">Cohort</th><th className="px-3 py-2">Accuracy</th><th className="px-3 py-2">95% interval</th><th className="px-3 py-2">Matured sample</th><th className="px-3 py-2">Coverage</th><th className="px-3 py-2">Return & benchmarks</th><th className="px-3 py-2">Viability</th>
            </tr></thead>
            <tbody className="divide-y divide-gray-100">{rows.map((row) => <tr key={`${row.evidence_source}:${row.dimension_value}`}>
              <td className="px-3 py-2 text-xs font-medium">{row.evidence_source === 'SAF' ? 'SAF validated' : 'Historical outcome'}</td>
              <td className="px-3 py-2 font-mono text-xs">{row.dimension_value}</td>
              <td className="px-3 py-2 font-semibold">{pct(row.directional_accuracy)}{row.exploratory ? <span className="ml-1 rounded border border-amber-200 bg-amber-50 px-1 py-0.5 text-[10px] font-medium text-amber-700">Exploratory</span> : null}</td>
              <td className="px-3 py-2 text-xs">{pct(row.accuracy_lower_bound)} - {pct(row.accuracy_upper_bound)}</td>
              <td className="px-3 py-2 text-xs">{row.directional_hits}/{row.sample_size}</td>
              <td className="px-3 py-2 text-xs">{row.censored_count ? `${row.censored_count} active` : 'terminal'}{row.excluded_count ? ` - ${row.excluded_count} excluded` : ''}</td>
              <td className="px-3 py-2 text-xs">{pct(row.average_return)}<div className="mt-1 text-[11px] text-gray-500">SPY excess {pct(row.average_relative_return)} · sector excess {pct(row.average_sector_relative_return)}</div><div className="text-[11px] text-gray-500">coverage: SPY {row.broad_market_benchmark_sample_size}/{row.sample_size} · sector {row.sector_benchmark_sample_size}/{row.sample_size}</div></td>
              <ViabilityCell row={row} />
            </tr>)}</tbody>
          </table>
          {rows.length === 0 ? <EmptyState message="No effectiveness records match the current filter." /> : null}
        </div>
        <section className="rounded border border-gray-200">
          <div className="border-b border-gray-200 bg-gray-50 px-3 py-2"><h3 className="text-sm font-semibold">Improvement candidates</h3><p className="text-xs text-gray-500">Evidence-backed recommendations only; nothing here changes an algorithm or sends an alert.</p></div>
          {(recommendations.data?.recommendations.length ?? 0) === 0 ? <EmptyState message="No statistically supported improvement candidates yet." /> : <div className="divide-y divide-gray-100">{recommendations.data?.recommendations.map((item) => <div key={item.recommendation_id} className="p-3"><div className="flex flex-wrap items-center gap-2"><span className={item.priority === 'high' ? 'rounded border border-red-200 bg-red-50 px-1.5 py-0.5 text-[11px] font-medium text-red-700' : 'rounded border border-amber-200 bg-amber-50 px-1.5 py-0.5 text-[11px] font-medium text-amber-700'}>{label(item.priority)}</span><strong className="text-sm">{item.dimension_value}</strong><span className="text-xs text-gray-500">{item.evidence_source} - n={item.sample_size} - accuracy {pct(item.directional_accuracy)} - upper 95% bound {pct(item.accuracy_upper_bound)}</span></div><p className="mt-1 text-xs text-gray-700">{item.summary}</p></div>)}</div>}
        </section>
      </>}
    </section>
  );
}


type DailyProgressionPoint = {
  date: string;
  sample: number;
  hits: number;
  accuracy: number;
  cumulativeSample: number;
  cumulativeHits: number;
  cumulativeAccuracy: number;
  averageDirectionalReturn?: number;
  averageRelativeReturn?: number;
  averageSectorRelativeReturn?: number;
  broadMarketMatched: number;
  sectorMatched: number;
};

const finite = (value?: number): value is number => value != null && Number.isFinite(value);
const average = (values: number[]) => values.length ? values.reduce((sum, value) => sum + value, 0) / values.length : undefined;
const sourceLabel = (source: string) => source === 'SAF' ? 'SAF validated' : 'Historical outcome';

function buildDailyProgression(observations: MarketOpsSignalAssuranceEffectivenessObservation[]): DailyProgressionPoint[] {
  const buckets = new Map<string, MarketOpsSignalAssuranceEffectivenessObservation[]>();
  for (const observation of observations) {
    if (observation.directional_hit == null || !observation.outcome_at) continue;
    const date = observation.outcome_at.slice(0, 10);
    if (!date) continue;
    const rows = buckets.get(date) ?? [];
    rows.push(observation);
    buckets.set(date, rows);
  }
  let cumulativeSample = 0;
  let cumulativeHits = 0;
  return Array.from(buckets.entries()).sort(([left], [right]) => left.localeCompare(right)).map(([date, rows]) => {
    const sample = rows.length;
    const hits = rows.filter((row) => row.directional_hit).length;
    cumulativeSample += sample;
    cumulativeHits += hits;
    const directionalReturns = rows.map((row) => row.directional_return).filter(finite);
    const relativeReturns = rows.map((row) => row.relative_return).filter(finite);
    const sectorReturns = rows.map((row) => row.sector_relative_return).filter(finite);
    return {
      date,
      sample,
      hits,
      accuracy: sample ? hits / sample : 0,
      cumulativeSample,
      cumulativeHits,
      cumulativeAccuracy: cumulativeSample ? cumulativeHits / cumulativeSample : 0,
      averageDirectionalReturn: average(directionalReturns),
      averageRelativeReturn: average(relativeReturns),
      averageSectorRelativeReturn: average(sectorReturns),
      broadMarketMatched: rows.filter((row) => row.broad_market_benchmark_state === 'matched').length,
      sectorMatched: rows.filter((row) => row.sector_benchmark_state === 'matched').length,
    };
  });
}

function DailyProgressionChart({ observations, evidenceSource, loading, error }: { observations: MarketOpsSignalAssuranceEffectivenessObservation[]; evidenceSource: string; loading: boolean; error: unknown }) {
  const points = useMemo(() => buildDailyProgression(observations), [observations]);
  const latest = points.at(-1);
  const axisInterval = Math.max(0, Math.ceil(points.length / 7) - 1);
  const option = useMemo(() => ({
    animation: false,
    grid: { left: 44, right: 44, top: 48, bottom: 42 },
    legend: { top: 0, textStyle: { fontSize: 11 } },
    tooltip: {
      trigger: 'axis',
      formatter: (items: Array<{ dataIndex?: number }>) => {
        const point = points[items?.[0]?.dataIndex ?? 0];
        if (!point) return '';
        return [
          point.date,
          `Daily accuracy: ${pct(point.accuracy)} (${point.hits}/${point.sample})`,
          `Cumulative accuracy: ${pct(point.cumulativeAccuracy)} (${point.cumulativeHits}/${point.cumulativeSample})`,
          `Avg aligned return: ${pct(point.averageDirectionalReturn)}`,
          `SPY excess: ${pct(point.averageRelativeReturn)}`,
          `Sector excess: ${pct(point.averageSectorRelativeReturn)}`,
          `Benchmark coverage: SPY ${point.broadMarketMatched}/${point.sample} · sector ${point.sectorMatched}/${point.sample}`,
        ].join('<br/>');
      },
    },
    xAxis: { type: 'category', data: points.map((point) => point.date), axisLabel: { fontSize: 10, interval: axisInterval, formatter: (value: string) => value.slice(5) } },
    yAxis: [
      { type: 'value', min: 0, max: 1, axisLabel: { fontSize: 10, formatter: (value: number) => `${Math.round(value * 100)}%` } },
      { type: 'value', min: 0, axisLabel: { fontSize: 10 }, splitLine: { show: false } },
    ],
    series: [
      { name: 'Daily hit rate', type: 'line', smooth: true, symbolSize: 5, data: points.map((point) => point.accuracy), lineStyle: { color: '#2563eb', width: 2 }, itemStyle: { color: '#2563eb' } },
      { name: 'Cumulative hit rate', type: 'line', smooth: true, symbolSize: 5, data: points.map((point) => point.cumulativeAccuracy), lineStyle: { color: '#16a34a', width: 2 }, itemStyle: { color: '#16a34a' } },
      { name: 'Avg aligned return', type: 'line', smooth: true, symbolSize: 4, data: points.map((point) => point.averageDirectionalReturn ?? null), lineStyle: { color: '#d97706', width: 2 }, itemStyle: { color: '#d97706' } },
      { name: 'Daily sample', type: 'bar', yAxisIndex: 1, data: points.map((point) => point.sample), itemStyle: { color: '#cbd5e1' } },
    ],
  }), [axisInterval, points]);

  if (loading) return <LoadingState label="Loading SAF daily progression..." />;
  if (error) return <ErrorState error={error} />;
  return <section data-testid="saf-daily-progression" className="rounded border border-gray-200 bg-white p-3">
    <div className="flex flex-wrap items-start justify-between gap-3">
      <div>
        <h3 className="text-sm font-semibold">Daily progression</h3>
        <p className="text-xs text-gray-500">Historical progression for {sourceLabel(evidenceSource)} observations, grouped by outcome date.</p>
      </div>
      {latest ? <div className="grid grid-cols-2 gap-2 text-xs sm:grid-cols-3">
        <div className="rounded border border-gray-200 bg-gray-50 p-2"><div className="text-gray-500">Latest day</div><div className="font-semibold">{latest.date}</div><div>{pct(latest.accuracy)} · {latest.hits}/{latest.sample}</div></div>
        <div className="rounded border border-gray-200 bg-gray-50 p-2"><div className="text-gray-500">Cumulative</div><div className="font-semibold">{pct(latest.cumulativeAccuracy)}</div><div>{latest.cumulativeHits}/{latest.cumulativeSample}</div></div>
        <div className="rounded border border-gray-200 bg-gray-50 p-2"><div className="text-gray-500">Avg aligned return</div><div className="font-semibold">{pct(latest.averageDirectionalReturn)}</div><div>SPY excess {pct(latest.averageRelativeReturn)}</div></div>
      </div> : null}
    </div>
    {points.length ? <div className="mt-3"><ThemedEChart option={option} style={{ height: 320 }} /></div> : <EmptyState message="No terminal SAF observations are available for daily progression under this filter." />}
  </section>;
}

function ViabilityCell({ row }: { row: MarketOpsSignalAssuranceEffectiveness }) {
  return <td className="max-w-64 px-3 py-2 text-xs"><div className="font-medium">{label(row.viability_state)}</div><div className="mt-1 text-[11px] text-gray-500" title={row.viability_reasons.join(' ')}>{row.viability_reasons[0]}</div></td>;
}

function SummaryCard({ row }: { row: MarketOpsSignalAssuranceEffectiveness }) {
  return <div className="rounded border border-gray-200 bg-gray-50 p-3"><div className="text-xs font-medium text-gray-500">{row.evidence_source === 'SAF' ? 'SAF validated performance' : 'Historical outcome evidence'}</div><div className="mt-1 text-2xl font-semibold">{pct(row.directional_accuracy)}</div><div className="mt-1 text-xs text-gray-600">{row.directional_hits}/{row.sample_size} matured directional observations - 95% interval {pct(row.accuracy_lower_bound)} - {pct(row.accuracy_upper_bound)}</div><div className="mt-2 text-xs font-medium text-gray-700">{label(row.viability_state)}</div><div className="mt-1 text-[11px] text-gray-500">{row.viability_reasons[0]}</div><div className="mt-2 text-[11px] text-gray-500">{row.censored_count} active/censored - {row.excluded_count} excluded - as of {formatUtc(row.as_of)}</div></div>;
}
