import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import { useTenant } from '../auth/session';
import { EmptyState, ErrorState, LoadingState } from '../components/States';
import { RefreshButton } from '../components/RefreshButton';
import { formatPercent } from '../lib/format';

const score = (value?: number) => value == null ? '—' : value.toFixed(1);
const title = (value: string) => value.replace(/_/g, ' ').toLowerCase().replace(/\b\w/g, (letter) => letter.toUpperCase());

type IndicatorTone = { accent: string; badge: string; dot: string; text: string };

const stateTone = (state: string): IndicatorTone => {
  switch (state) {
    case 'LEADING': return { accent: 'border-l-emerald-500', badge: 'border-emerald-200 bg-emerald-50 text-emerald-800', dot: 'bg-emerald-500', text: 'text-emerald-700' };
    case 'IMPROVING': return { accent: 'border-l-sky-500', badge: 'border-sky-200 bg-sky-50 text-sky-800', dot: 'bg-sky-500', text: 'text-sky-700' };
    case 'WEAKENING': return { accent: 'border-l-amber-500', badge: 'border-amber-200 bg-amber-50 text-amber-800', dot: 'bg-amber-500', text: 'text-amber-700' };
    case 'LAGGING': return { accent: 'border-l-rose-500', badge: 'border-rose-200 bg-rose-50 text-rose-800', dot: 'bg-rose-500', text: 'text-rose-700' };
    default: return { accent: 'border-l-slate-400', badge: 'border-slate-200 bg-slate-50 text-slate-700', dot: 'bg-slate-400', text: 'text-slate-700' };
  }
};

const scoreTone = (value?: number) => {
  if (value == null) return 'text-slate-500';
  if (value >= 75) return 'text-emerald-700';
  if (value >= 60) return 'text-sky-700';
  if (value >= 40) return 'text-slate-700';
  if (value >= 25) return 'text-amber-700';
  return 'text-rose-700';
};

const qualityTone = (quality: string) => quality === 'usable'
  ? 'border-emerald-200 bg-emerald-50 text-emerald-800'
  : quality === 'partial'
    ? 'border-amber-200 bg-amber-50 text-amber-800'
    : 'border-rose-200 bg-rose-50 text-rose-800';

function Metric({ label, value }: { label: string; value?: number }) {
  return <div><dt className="text-gray-500">{label}</dt><dd className={'font-semibold ' + scoreTone(value)}>{score(value)}</dd></div>;
}

export function MarketOpsSRIRoute() {
  const tenantId = useTenant();
  const [type, setType] = useState('');
  const [state, setState] = useState('');
  const query = useQuery({ queryKey: ['marketops-sri-rankings', tenantId, type, state], queryFn: () => api.getMarketOpsSRIRankings(tenantId, type, state), staleTime: 30_000 });
  const legend = ['LEADING', 'IMPROVING', 'NEUTRAL', 'WEAKENING', 'LAGGING'];

  return <div className="space-y-3">
    <div className="flex flex-wrap items-start justify-between gap-3">
      <div>
        <h1 className="text-lg font-semibold">Sector Rotation Intelligence</h1>
        <p className="max-w-3xl text-xs text-gray-500">Research-only, price-led market-segment context. This foundation ranks relative strength and momentum; it does not claim rotation, breadth, diffusion, flows, or a trade recommendation.</p>
      </div>
      <RefreshButton onClick={() => void query.refetch()} loading={query.isFetching} />
    </div>

    <div className="flex flex-wrap gap-3 rounded border border-gray-200 bg-gray-50 p-3">
      <label className="text-xs font-medium text-gray-700">Segment
        <select value={type} onChange={(event) => setType(event.target.value)} className="mt-1 block rounded border border-gray-300 bg-white px-2 py-1.5 text-sm">
          <option value="">All types</option><option value="sector">Sectors</option><option value="industry">Industries</option>
        </select>
      </label>
      <label className="text-xs font-medium text-gray-700">Context
        <select value={state} onChange={(event) => setState(event.target.value)} className="mt-1 block rounded border border-gray-300 bg-white px-2 py-1.5 text-sm">
          <option value="">All states</option><option value="LEADING">Leading</option><option value="IMPROVING">Improving</option><option value="NEUTRAL">Neutral</option><option value="WEAKENING">Weakening</option><option value="LAGGING">Lagging</option>
        </select>
      </label>
      <div className="self-end pb-1 text-xs text-gray-500">Score: <span className="font-medium text-emerald-700">75+ strong</span> · <span className="font-medium text-sky-700">60+ positive</span> · <span className="font-medium text-amber-700">&lt;40 weak</span></div>
    </div>

    <div aria-label="SRI context color legend" className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-gray-600">
      <span className="font-medium text-gray-700">Context legend:</span>
      {legend.map((item) => { const tone = stateTone(item); return <span key={item} className="inline-flex items-center gap-1"><i aria-hidden className={'h-2 w-2 rounded-full ' + tone.dot} />{title(item)}</span>; })}
    </div>

    {query.isLoading ? <LoadingState label="Loading segment context..." /> : query.isError ? <ErrorState error={query.error} /> : <>
      <p className="rounded border border-amber-200 bg-amber-50 p-2 text-xs text-amber-800">{query.data?.evidence_note}</p>
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {(query.data?.snapshots ?? []).map((item) => {
          const tone = stateTone(item.state);
          return <article key={item.snapshot_id} className={'rounded border border-gray-200 border-l-4 bg-white p-3 shadow-sm ' + tone.accent}>
            <div className="flex items-start justify-between gap-2">
              <div>
                <div className="font-mono text-xs text-gray-500">{item.segment_id.replace(/^sri_/, '')}</div>
                <div className={'mt-1 text-2xl font-semibold tabular-nums ' + scoreTone(item.composite_score)}>{score(item.composite_score)}</div>
                <div className="text-[11px] font-medium text-gray-500">Composite score</div>
              </div>
              <span className={'inline-flex items-center gap-1 rounded border px-1.5 py-0.5 text-[11px] font-medium ' + tone.badge}><i aria-hidden className={'h-1.5 w-1.5 rounded-full ' + tone.dot} />{title(item.state)}</span>
            </div>
            <dl className="mt-3 grid grid-cols-2 gap-2 text-xs">
              <div><dt className="text-gray-500">Rank</dt><dd className={'font-semibold ' + tone.text}>{item.rank ?? '—'}</dd></div>
              <div><dt className="text-gray-500">Quality</dt><dd><span className={'inline-flex rounded border px-1 py-0.5 text-[10px] font-medium ' + qualityTone(item.quality_state)}>{title(item.quality_state)}{item.evidence_quality != null ? ' · ' + formatPercent(item.evidence_quality) : ''}</span></dd></div>
              <Metric label="Relative strength" value={item.relative_strength_score} />
              <Metric label="Momentum" value={item.momentum_score} />
              <Metric label="Acceleration" value={item.momentum_acceleration} />
              <div><dt className="text-gray-500">Session</dt><dd className="font-medium tabular-nums">{item.session_date}</dd></div>
            </dl>
            <details className="mt-3">
              <summary className="cursor-pointer text-xs font-medium text-brand-700">Method and inputs</summary>
              <pre className="mt-2 max-h-48 overflow-auto rounded bg-gray-50 p-2 text-[10px] text-gray-700">{JSON.stringify({ components: item.components, quality_flags: item.quality_flags, input_provenance: item.input_provenance, algorithm_version: item.algorithm_version }, null, 2)}</pre>
            </details>
          </article>;
        })}
      </div>
      {(query.data?.snapshots.length ?? 0) === 0 ? <EmptyState message="No SRI snapshots are ready. The runner requires 61 sessions for each ETF and benchmark before it ranks a segment." /> : null}
    </>}
  </div>;
}
