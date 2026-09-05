import { useMemo, useState } from 'react';

import { useMarketOpsSignalAssuranceAssertions } from '../api/queries';
import { useTenant } from '../auth/session';
import { EmptyState, ErrorState, LoadingState } from '../components/States';
import { RefreshButton } from '../components/RefreshButton';
import { SyncraticExplainabilityCard } from '../components/SyncraticExplainabilityCard';
import { formatUtc } from '../lib/format';
import { MarketOpsSignalAssuranceDrilldownPanel } from './MarketOpsSignalAssuranceDrilldownPanel';
import { MarketOpsSignalAssuranceDailyProgressionPanel, MarketOpsSignalAssuranceEffectivenessPanel } from './MarketOpsSignalAssuranceEffectivenessPanel';
import type { MarketOpsSignalAssuranceAssertion } from '../types';

const title = (value: string) => value.replace(/_/g, ' ').toLowerCase().replace(/\b\w/g, (letter: string) => letter.toUpperCase());
const tone = (state: string) => state === 'MATERIALIZED' ? 'border-green-200 bg-green-50 text-green-700 dark:border-green-800 dark:bg-green-950/40 dark:text-green-300' : state === 'INVALIDATED' ? 'border-red-200 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-950/40 dark:text-red-300' : state === 'EXPIRED' ? 'border-gray-200 bg-gray-50 text-gray-600 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300' : 'border-blue-200 bg-blue-50 text-blue-700 dark:border-blue-800 dark:bg-blue-950/40 dark:text-blue-200';

export function MarketOpsSignalAssuranceRoute() {
  const tenantId = useTenant();
  const [symbol, setSymbol] = useState('');
  const [mode, setMode] = useState('');
  const [state, setState] = useState('');
  const [selectedAssertionId, setSelectedAssertionId] = useState<string | null>(null);
  const filter = useMemo(() => ({ tenant_id: tenantId, symbol: symbol.trim().toUpperCase() || undefined, evaluation_mode: mode || undefined, state: state || undefined, limit: 100 }), [tenantId, symbol, mode, state]);
  const assertions = useMarketOpsSignalAssuranceAssertions(filter);
  const active = Boolean(symbol || mode || state);
  const clear = () => { setSymbol(''); setMode(''); setState(''); };

  return <div className="space-y-3 text-gray-900 dark:text-gray-100">
    <div className="flex flex-wrap items-start justify-between gap-3">
      <div>
        <h1 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Signal Assurance</h1>
        <p className="max-w-3xl text-xs text-gray-500 dark:text-gray-400">Post-confirmation evidence for MarketOps signals. Each assertion preserves its resolved validation contract, immutable baseline, and point-in-time provenance. Research-only results are not trade instructions.</p>
      </div>
      <RefreshButton onClick={() => void assertions.refetch()} loading={assertions.isFetching} />
    </div>


    <SyncraticExplainabilityCard
      surface="Signal Assurance"
      description="Use Syncratic to explain signal viability trends, strongest/weakest algorithm evidence, and where confirmation quality suggests calibration work."
    />

    <MarketOpsSignalAssuranceDrilldownPanel />
    <MarketOpsSignalAssuranceDailyProgressionPanel />

    <div className="flex flex-wrap items-end gap-3 rounded border border-gray-200 bg-gray-50 p-3 dark:border-gray-700 dark:bg-gray-800">
      <label className="text-xs font-medium text-gray-700 dark:text-gray-300">Symbol
        <input value={symbol} onChange={(event) => setSymbol(event.target.value)} placeholder="Ticker" className="mt-1 block w-28 rounded border border-gray-300 bg-white px-2 py-1.5 font-mono text-sm text-gray-900 dark:border-gray-600 dark:bg-gray-950 dark:text-gray-100" />
      </label>
      <label className="text-xs font-medium text-gray-700 dark:text-gray-300">Mode
        <select value={mode} onChange={(event) => setMode(event.target.value)} className="mt-1 block rounded border border-gray-300 bg-white px-2 py-1.5 text-sm text-gray-900 dark:border-gray-600 dark:bg-gray-950 dark:text-gray-100"><option value="">All modes</option><option value="RESEARCH">Research</option><option value="LIVE">Live</option><option value="BACKTEST">Backtest</option></select>
      </label>
      <label className="text-xs font-medium text-gray-700 dark:text-gray-300">State
        <select value={state} onChange={(event) => setState(event.target.value)} className="mt-1 block rounded border border-gray-300 bg-white px-2 py-1.5 text-sm text-gray-900 dark:border-gray-600 dark:bg-gray-950 dark:text-gray-100"><option value="">All states</option><option value="ACTIVE">Active</option><option value="MATERIALIZED">Materialized</option><option value="INVALIDATED">Invalidated</option><option value="EXPIRED">Expired</option><option value="SUPERSEDED">Superseded</option></select>
      </label>
      {active ? <button type="button" onClick={clear} className="rounded border border-gray-300 bg-white px-3 py-1.5 text-xs font-medium text-gray-700 hover:bg-yellow-50 dark:border-gray-600 dark:bg-gray-950 dark:text-gray-200 dark:hover:bg-gray-800">Clear filters</button> : null}
      <span className="pb-1 text-xs text-gray-500 dark:text-gray-400">{assertions.data?.assertions.length ?? 0} assertions</span>
    </div>

    {assertions.isLoading ? <LoadingState /> : assertions.isError ? <ErrorState error={assertions.error} /> : (assertions.data?.assertions.length ?? 0) === 0 ? <EmptyState message="No assurance assertions yet. A successful reviewed MarketOps signal materialization creates the first RESEARCH assertion automatically." /> : <>
      <div className="space-y-2 md:hidden" aria-label="Mobile Signal Assurance assertions">
        {assertions.data?.assertions.map((assertion) => <SignalAssuranceMobileCard key={assertion.assertion_id} assertion={assertion} selected={selectedAssertionId === assertion.assertion_id} onToggle={() => setSelectedAssertionId((current) => current === assertion.assertion_id ? null : assertion.assertion_id)} />)}
      </div>
      <div className="hidden overflow-x-auto rounded border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-900 md:block">
        <table className="min-w-full text-sm">
          <thead className="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500 dark:bg-gray-800 dark:text-gray-400"><tr><th className="px-3 py-2">Signal</th><th className="px-3 py-2">Direction</th><th className="px-3 py-2">State</th><th className="px-3 py-2">Mode</th><th className="px-3 py-2">Validation contract</th><th className="px-3 py-2">Confirmed</th><th className="px-3 py-2">Audit</th></tr></thead>
          <tbody className="divide-y divide-gray-100 dark:divide-gray-800">{assertions.data?.assertions.map((assertion) => <tr key={assertion.assertion_id} className="hover:bg-gray-50 dark:hover:bg-gray-800/70"><td className="px-3 py-2"><div className="font-mono font-semibold text-gray-900 dark:text-gray-100">{assertion.symbol}</div><div className="text-xs text-gray-500 dark:text-gray-400">{title(assertion.signal_type)}</div></td><td className="px-3 py-2 text-xs font-medium">{title(assertion.direction)}</td><td className="px-3 py-2"><span className={`inline-flex rounded border px-1.5 py-0.5 text-[11px] font-medium ${tone(assertion.state)}`}>{title(assertion.state)}</span></td><td className="px-3 py-2 text-xs">{title(assertion.evaluation_mode)}{assertion.evaluation_run_id ? <div className="mt-1 font-mono text-[11px] text-gray-500 dark:text-gray-400">{assertion.evaluation_run_id}</div> : null}</td><td className="px-3 py-2 text-xs"><div className="font-mono">{assertion.validation_contract_id}</div><div className="text-gray-500 dark:text-gray-400">v{assertion.validation_contract_version}</div></td><td className="whitespace-nowrap px-3 py-2 text-xs text-gray-600 dark:text-gray-400">{formatUtc(assertion.confirmed_at)}</td><td className="px-3 py-2"><details><summary className="cursor-pointer text-xs text-brand-700 underline dark:text-brand-200">Baseline</summary><pre className="mt-2 max-w-md overflow-auto rounded bg-gray-50 p-2 text-[10px] text-gray-700 dark:bg-gray-950 dark:text-gray-200">{JSON.stringify({ baseline_snapshot: assertion.baseline_snapshot, baseline_provenance: assertion.baseline_provenance }, null, 2)}</pre></details></td></tr>)}</tbody>
        </table>
      </div>
    </>}

    <MarketOpsSignalAssuranceEffectivenessPanel />
  </div>;
}


function SignalAssuranceMobileCard({ assertion, selected, onToggle }: { assertion: MarketOpsSignalAssuranceAssertion; selected: boolean; onToggle: () => void }) {
  return <article data-testid={`saf-mobile-assertion-${assertion.symbol}`} className={selected ? "rounded-lg border border-brand-300 bg-brand-50 p-3 shadow-sm dark:border-brand-700 dark:bg-brand-950/30" : "rounded-lg border border-gray-200 bg-white p-3 shadow-sm dark:border-gray-700 dark:bg-gray-900"}>
    <button type="button" onClick={onToggle} aria-expanded={selected} className="w-full text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-mono text-sm font-semibold text-gray-900 dark:text-gray-100">{assertion.symbol}</span>
            <span className={`inline-flex rounded border px-1.5 py-0.5 text-[11px] font-medium ${tone(assertion.state)}`}>{title(assertion.state)}</span>
          </div>
          <div className="mt-1 text-sm font-medium text-gray-900 dark:text-gray-100">{title(assertion.signal_type)}</div>
          <div className="mt-1 flex flex-wrap gap-2 text-xs text-gray-600 dark:text-gray-400"><span>{title(assertion.direction)}</span><span>{title(assertion.evaluation_mode)}</span><span>{formatUtc(assertion.confirmed_at)}</span></div>
        </div>
        <span className="shrink-0 text-xs font-medium text-brand-700 dark:text-brand-200">{selected ? "Close" : "Inspect"}</span>
      </div>
      <dl className="mt-3 grid gap-2 text-xs">
        <div className="rounded border border-gray-100 bg-white p-2 dark:border-gray-800 dark:bg-gray-950"><dt className="text-[10px] font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">Validation contract</dt><dd className="mt-1 break-all font-mono text-gray-900 dark:text-gray-100">{assertion.validation_contract_id}</dd><dd className="text-gray-500 dark:text-gray-400">v{assertion.validation_contract_version}</dd></div>
        {assertion.evaluation_run_id ? <div className="rounded border border-gray-100 bg-white p-2 dark:border-gray-800 dark:bg-gray-950"><dt className="text-[10px] font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">Evaluation run</dt><dd className="mt-1 break-all font-mono text-gray-900 dark:text-gray-100">{assertion.evaluation_run_id}</dd></div> : null}
      </dl>
    </button>
    {selected ? <div className="mt-3 rounded border border-brand-200 bg-white p-3 dark:border-brand-900 dark:bg-gray-950"><div className="flex items-start justify-between gap-3"><div><h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100">Baseline and provenance</h3><p className="mt-1 text-xs text-gray-600 dark:text-gray-400">Immutable baseline used to confirm this SAF assertion.</p></div><button type="button" onClick={onToggle} className="shrink-0 rounded border border-gray-300 px-2 py-1 text-xs font-medium text-gray-700 dark:border-gray-600 dark:text-gray-200">Close</button></div><pre className="mt-2 max-h-72 overflow-auto rounded bg-gray-50 p-2 text-[10px] text-gray-700 dark:bg-gray-900 dark:text-gray-200">{JSON.stringify({ baseline_snapshot: assertion.baseline_snapshot, baseline_provenance: assertion.baseline_provenance }, null, 2)}</pre></div> : null}
  </article>;
}
