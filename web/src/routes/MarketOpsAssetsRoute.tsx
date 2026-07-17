import { useState } from 'react';
import { CircleDollarSign, X } from 'lucide-react';
import ReactECharts from 'echarts-for-react';
import { useMarketOpsAssets } from '../api/queries';
import { useMarketOpsOptionsCoverage, useMarketOpsOptionsDistributions, useMarketOpsOptionsChain } from '../api/queries';
import { isApiError } from '../api/client';
import { EmptyState, ErrorState, LoadingState } from '../components/States';
import { StatusBadge } from '../components/StatusBadge';
import { MetricTile } from '../components/MetricTile';
import { JsonViewer } from '../components/JsonViewer';
import { formatUtc } from '../lib/format';
import {
  summarizeMarketOpsOptionsCoverage,
  summarizeMarketOpsOptionsDistribution,
  summarizeMarketOpsOptionsChainRow,
  marketOpsOptionsDateOnly,
  type MarketOpsOptionsDistributionView,
  type MarketOpsOptionsBucketEntry,
} from '../lib/marketopsOptions';
import { useTenant } from '../auth/session';

// Read-only MarketOps asset universe (G071 frontend) + G128 per-asset options
// intelligence panel. The universe table is backend data only; selecting a row
// opens a read-only options panel (coverage / distribution / chain) that
// performs no ingestion and never calls live-preview (which stays 501).

const CHAIN_LIMITS = [100, 200, 500];

export function MarketOpsAssetsRoute() {
  const TENANT_ID = useTenant();
  const [selectedTicker, setSelectedTicker] = useState<string | null>(null);
  const query = useMarketOpsAssets({
    tenant_id: TENANT_ID,
    universe_group: 'top50_megacap',
    active_only: true,
    limit: 50,
  });

  // Sort defensively by rank so the displayed order is stable regardless of
  // backend ordering; slice() avoids mutating the cached response.
  const data = (query.data?.assets ?? []).slice().sort((a, b) => a.rank - b.rank);
  const active = data.filter((a) => a.is_active).length;
  const sectors = new Set(data.map((a) => a.sector_key || a.sector).filter(Boolean)).size;
  const industries = new Set(data.map((a) => a.industry_key || a.industry).filter(Boolean)).size;
  const sourceIds = new Set(data.map((a) => a.source_id));
  const sourceLabel = sourceIds.size === 1 ? [...sourceIds][0] : `${sourceIds.size} sources`;

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-lg font-semibold">Assets</h1>
          <p className="text-xs text-gray-500">Tenant {TENANT_ID} · top50_megacap</p>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-2 md:grid-cols-5">
        <MetricTile label="Universe Assets" value={data.length} />
        <MetricTile label="Active Assets" value={active} />
        <MetricTile label="Sectors" value={sectors} />
        <MetricTile label="Industries" value={industries} />
        <MetricTile label="Source" value={sourceLabel} />
      </div>

      {query.isLoading ? (
        <LoadingState />
      ) : query.isError ? (
        <ErrorState error={query.error} />
      ) : data.length ? (
        <div className="overflow-hidden rounded border border-gray-200 bg-white">
          <table className="min-w-full divide-y divide-gray-200 text-sm">
            <thead className="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500">
              <tr>
                <th className="px-3 py-2">Rank</th>
                <th className="px-3 py-2">Asset</th>
                <th className="px-3 py-2">Sector</th>
                <th className="px-3 py-2">Industry</th>
                <th className="px-3 py-2">Source</th>
                <th className="px-3 py-2">Status</th>
                <th className="px-3 py-2">Updated</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {data.map((a) => (
                <tr
                  key={`${a.tenant_id}:${a.ticker_key}`}
                  onClick={() => setSelectedTicker(a.ticker)}
                  className={`cursor-pointer align-top hover:bg-gray-50 ${selectedTicker === a.ticker ? 'bg-brand-50' : ''}`}
                >
                  <td className="px-3 py-2 text-xs text-gray-500">{a.rank}</td>
                  <td className="px-3 py-2">
                    <div className="flex items-start gap-2">
                      <CircleDollarSign size={16} className="mt-0.5 text-brand-700" />
                      <div>
                        <div className="font-medium text-gray-900">{a.company || '—'}</div>
                        <div className="font-mono text-xs text-gray-500">{a.ticker}</div>
                        <div className="text-xs text-gray-500">{a.asset_type}</div>
                      </div>
                    </div>
                  </td>
                  <td className="px-3 py-2 text-xs">{a.sector || '—'}</td>
                  <td className="px-3 py-2 text-xs">{a.industry || '—'}</td>
                  <td className="px-3 py-2 font-mono text-xs">{a.source_id}</td>
                  <td className="px-3 py-2"><StatusBadge status={a.is_active ? 'active' : 'inactive'} /></td>
                  <td className="px-3 py-2 text-xs text-gray-600">{formatUtc(a.updated_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <EmptyState message="No MarketOps assets found for this tenant." />
      )}

      {selectedTicker ? (
        <AssetOptionsPanel tenantId={TENANT_ID} symbol={selectedTicker} onClose={() => setSelectedTicker(null)} />
      ) : (
        <div className="rounded border border-gray-200 bg-white p-3">
          <EmptyState message="Select an asset to inspect its persisted options coverage and distribution." />
        </div>
      )}

      {data.length > 0 && (
        <div className="rounded border border-gray-200 bg-white p-3">
          <h2 className="mb-2 text-sm font-semibold">Asset Metadata</h2>
          <JsonViewer value={data.map((a) => ({ ticker: a.ticker, metadata: a.metadata }))} />
        </div>
      )}
    </div>
  );
}

// G128 read-only options intelligence for one asset. Coverage (availability),
// latest distribution summary + rolling ratio chart + moneyness/expiration
// buckets, and a filterable chain table. Loading/error/empty are scoped per
// section and never block asset browsing. No ingestion or live-preview controls.
function AssetOptionsPanel({
  tenantId,
  symbol,
  onClose,
}: {
  tenantId: string;
  symbol: string;
  onClose: () => void;
}) {
  const coverageQ = useMarketOpsOptionsCoverage(tenantId, symbol);
  const distQ = useMarketOpsOptionsDistributions(tenantId, symbol, { window: '10_trade_days', limit: 10 });
  const distRows = (distQ.data?.options_distributions ?? []).map(summarizeMarketOpsOptionsDistribution);
  const chartRows = distRows.slice().sort((a, b) => a.tradeDate.localeCompare(b.tradeDate));
  const latest = distRows.slice().sort((a, b) => b.tradeDate.localeCompare(a.tradeDate))[0] ?? null;

  // Chain controls. The trade-date selector is populated from distribution
  // snapshot dates; the latest snapshot is the default (effective trade date).
  const distDates = Array.from(new Set(distRows.map((r) => marketOpsOptionsDateOnly(r.tradeDate)))).sort().reverse();
  const [tradeDate, setTradeDate] = useState('');
  const [contractType, setContractType] = useState<'' | 'call' | 'put'>('');
  const [chainLimit, setChainLimit] = useState(500);
  const effectiveTradeDate = tradeDate || distDates[0] || '';
  const chainQ = useMarketOpsOptionsChain(tenantId, symbol, {
    trade_date: effectiveTradeDate || undefined,
    contract_type: contractType || undefined,
    limit: chainLimit,
  });
  const chainRows = (chainQ.data?.options_chain ?? []).map(summarizeMarketOpsOptionsChainRow);

  const coverageNotFound =
    coverageQ.isError && isApiError(coverageQ.error) && coverageQ.error.code === 'options_coverage_not_found';

  return (
    <div className="space-y-3 rounded border border-gray-200 bg-white p-3">
      <div className="flex items-center justify-between gap-2">
        <div>
          <h2 className="text-sm font-semibold">Options · <span className="font-mono">{symbol}</span></h2>
          <p className="text-[11px] text-gray-500">Persisted coverage + derived distribution · read-only, no ingestion</p>
        </div>
        <button
          type="button"
          onClick={onClose}
          className="inline-flex items-center gap-1 rounded border border-gray-300 bg-white px-2 py-1 text-xs text-gray-600 hover:bg-gray-50"
        >
          <X size={14} /> Close
        </button>
      </div>

      {/* Coverage strip */}
      <div className="rounded border border-gray-200 bg-gray-50 p-2">
        <div className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-gray-500">Coverage</div>
        {coverageQ.isLoading ? (
          <div className="text-xs text-gray-500">Loading options coverage…</div>
        ) : coverageNotFound ? (
          <div className="text-xs text-amber-700">No persisted options coverage for {symbol}.</div>
        ) : coverageQ.isError ? (
          <div className="text-xs text-red-700">
            Coverage unavailable{isApiError(coverageQ.error) ? `: ${coverageQ.error.message}` : ''}.
          </div>
        ) : coverageQ.data ? (
          (() => {
            const c = summarizeMarketOpsOptionsCoverage(coverageQ.data.options_coverage);
            return (
              <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-gray-700">
                <Stat label="Trade days" value={c.tradeDayCount} />
                <Stat label="Contracts" value={c.contractCount} />
                <Stat label="First trade" value={c.firstTradeDate ? formatUtc(c.firstTradeDate) : '—'} />
                <Stat label="Last trade" value={c.lastTradeDate ? formatUtc(c.lastTradeDate) : '—'} />
                <Stat label="Last updated" value={c.lastUpdatedAt ? formatUtc(c.lastUpdatedAt) : '—'} />
              </div>
            );
          })()
        ) : null}
      </div>

      {/* Latest distribution summary + rolling chart + buckets */}
      <div className="rounded border border-gray-200 bg-gray-50 p-2">
        <div className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-gray-500">Distribution (10_trade_days)</div>
        {distQ.isLoading ? (
          <div className="text-xs text-gray-500">Loading options distribution…</div>
        ) : distQ.isError ? (
          <div className="text-xs text-red-700">
            Distribution unavailable{isApiError(distQ.error) ? `: ${distQ.error.message}` : ''}.
          </div>
        ) : distRows.length === 0 ? (
          <div className="text-xs text-gray-500">No options distribution snapshots yet.</div>
        ) : latest ? (
          <div className="space-y-2">
            <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-gray-700">
              <Stat label="Trade date" value={latest.tradeDate ? formatUtc(latest.tradeDate) : '—'} />
              <Stat label="OI ratio" value={latest.callPutOpenInterestRatio.toFixed(2)} />
              <Stat label="Vol ratio" value={latest.callPutVolumeRatio.toFixed(2)} />
              <Stat label="Δ ratio" value={latest.ratioDelta.toFixed(2)} />
              <Stat label="Z-score" value={latest.ratioZScore.toFixed(2)} />
              <Stat label="Confidence" value={latest.confidence.toFixed(2)} />
              <Stat label="Calls/Puts" value={`${latest.callContractCount}/${latest.putContractCount}`} />
              {latest.missingOpenInterestCount > 0 ? (
                <span className="inline-flex items-center gap-1 rounded border border-amber-300 bg-amber-50 px-1.5 py-0.5 text-[11px] font-medium text-amber-700">
                  Missing OI <strong>{latest.missingOpenInterestCount}</strong>
                </span>
              ) : (
                <Stat label="Missing OI" value={latest.missingOpenInterestCount} />
              )}
            </div>
            {latest.provider || latest.sourceId ? (
              <div className="text-[11px] text-gray-500">
                Provider <span className="text-gray-700">{latest.provider || '—'}</span>
                {' · '}source <code className="text-gray-700">{latest.sourceId || '—'}</code>
              </div>
            ) : null}

            {chartRows.length > 1 ? <OptionsRatioChart rows={chartRows} /> : null}

            <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
              <div>
                <div className="mb-1 text-[11px] font-medium text-gray-600">Moneyness (call vs put OI)</div>
                <BucketBars entries={latest.moneynessBuckets} />
              </div>
              <div>
                <div className="mb-1 text-[11px] font-medium text-gray-600">Expiration (call vs put OI)</div>
                <BucketBars entries={latest.expirationBuckets} />
              </div>
            </div>
          </div>
        ) : null}
      </div>

      {/* Chain table */}
      <div className="rounded border border-gray-200 bg-gray-50 p-2">
        <div className="mb-1 flex flex-wrap items-center justify-between gap-2">
          <div className="text-[11px] font-semibold uppercase tracking-wide text-gray-500">Chain</div>
          <div className="flex flex-wrap items-center gap-2">
            <select
              value={effectiveTradeDate}
              onChange={(e) => setTradeDate(e.target.value)}
              className="rounded border border-gray-300 px-2 py-1 text-xs"
              aria-label="Filter chain by trade date"
              disabled={distDates.length === 0}
            >
              {distDates.length === 0 ? (
                <option value="">no snapshots</option>
              ) : (
                distDates.map((d) => (
                  <option key={d} value={d}>{d}</option>
                ))
              )}
            </select>
            <div className="inline-flex overflow-hidden rounded border border-gray-300 text-xs">
              {(['', 'call', 'put'] as const).map((ct) => (
                <button
                  key={ct || 'all'}
                  type="button"
                  onClick={() => setContractType(ct)}
                  className={`px-2 py-1 ${contractType === ct ? 'bg-brand-600 text-white' : 'bg-white text-gray-600 hover:bg-gray-50'}`}
                >
                  {ct ? ct : 'all'}
                </button>
              ))}
            </div>
            <select
              value={chainLimit}
              onChange={(e) => setChainLimit(Number(e.target.value))}
              className="rounded border border-gray-300 px-2 py-1 text-xs"
              aria-label="Chain row limit"
            >
              {CHAIN_LIMITS.map((n) => (
                <option key={n} value={n}>{n}</option>
              ))}
            </select>
          </div>
        </div>
        {chainQ.isLoading ? (
          <LoadingState label="Loading options chain..." />
        ) : chainQ.isError ? (
          <ErrorState error={chainQ.error} />
        ) : chainRows.length ? (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200 text-sm">
              <thead className="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500">
                <tr>
                  <th className="whitespace-nowrap px-3 py-2">Option</th>
                  <th className="whitespace-nowrap px-3 py-2">Type</th>
                  <th className="whitespace-nowrap px-3 py-2">Expiration</th>
                  <th className="whitespace-nowrap px-3 py-2">Strike</th>
                  <th className="whitespace-nowrap px-3 py-2">Moneyness</th>
                  <th className="whitespace-nowrap px-3 py-2">Open interest</th>
                  <th className="whitespace-nowrap px-3 py-2">Volume</th>
                  <th className="whitespace-nowrap px-3 py-2">IV</th>
                  <th className="whitespace-nowrap px-3 py-2">Delta</th>
                  <th className="whitespace-nowrap px-3 py-2">Underlying</th>
                  <th className="whitespace-nowrap px-3 py-2">Updated</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {chainRows.map((r) => (
                  <tr key={r.optionTicker} className="align-top">
                    <td className="px-3 py-2"><code className="break-all text-[11px] text-gray-700">{r.optionTicker || '—'}</code></td>
                    <td className="whitespace-nowrap px-3 py-2 text-xs"><ContractTypeBadge type={r.contractType} /></td>
                    <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-600">{r.expirationDate ? formatUtc(r.expirationDate) : '—'}</td>
                    <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-800">{r.strikePrice.toFixed(2)}</td>
                    <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-700"><NullNum value={r.moneyness} digits={4} /></td>
                    <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-800"><NullNum value={r.openInterest} /></td>
                    <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-800"><NullNum value={r.volume} /></td>
                    <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-700"><NullNum value={r.impliedVolatility} digits={2} /></td>
                    <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-700"><NullNum value={r.delta} digits={2} /></td>
                    <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-700"><NullNum value={r.underlyingClose} digits={2} /></td>
                    <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-600">{r.updatedAt ? formatUtc(r.updatedAt) : '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <EmptyState message="No chain rows match this filter." />
        )}
        <p className="mt-1 text-[11px] text-gray-400">
          Chain numeric columns are server-nullable; absent open interest / volume render as `—` and weaken call/put ratio interpretation.
        </p>
      </div>
    </div>
  );
}

function Stat({ label, value }: { label: string; value: number | string }) {
  return (
    <span>
      <span className="text-gray-500">{label}: </span>
      <strong className="text-gray-800">{value}</strong>
    </span>
  );
}

// Render a server-nullable numeric as `—` when absent (not 0).
function NullNum({ value, digits = 0 }: { value: number | null; digits?: number }) {
  if (value === null) return <span className="text-gray-400">—</span>;
  return <>{digits > 0 ? value.toFixed(digits) : String(value)}</>;
}

function ContractTypeBadge({ type }: { type: string }) {
  const tone = type === 'call' ? 'text-blue-700' : type === 'put' ? 'text-orange-700' : 'text-gray-600';
  return <span className={`font-medium ${tone}`}>{type || '—'}</span>;
}

// Compact call-vs-put open-interest bars per bucket, with volume readout.
// Scaled to the largest single OI across buckets; empty buckets render as `None`.
function BucketBars({ entries }: { entries: MarketOpsOptionsBucketEntry[] }) {
  if (!entries.length) return <span className="text-[11px] text-gray-400">None</span>;
  const maxOi = Math.max(1, ...entries.map((e) => Math.max(e.callOpenInterest, e.putOpenInterest)));
  return (
    <div className="space-y-1">
      {entries.map((e) => (
        <div key={e.key} className="flex items-center gap-2 text-[11px]">
          <div className="w-20 shrink-0 text-gray-600">{e.key}</div>
          <div className="flex-1 space-y-0.5">
            <div className="flex items-center gap-1">
              <span className="w-3 text-blue-700">C</span>
              <div className="h-2 flex-1 overflow-hidden rounded bg-gray-100">
                <div className="h-2 rounded bg-blue-400" style={{ width: `${(e.callOpenInterest / maxOi) * 100}%` }} />
              </div>
            </div>
            <div className="flex items-center gap-1">
              <span className="w-3 text-orange-700">P</span>
              <div className="h-2 flex-1 overflow-hidden rounded bg-gray-100">
                <div className="h-2 rounded bg-orange-400" style={{ width: `${(e.putOpenInterest / maxOi) * 100}%` }} />
              </div>
            </div>
          </div>
          <div className="w-28 shrink-0 text-right text-gray-500">
            OI <strong className="text-gray-700">{e.callOpenInterest}</strong>/<strong className="text-gray-700">{e.putOpenInterest}</strong>
            <br />vol {e.callVolume}/{e.putVolume}
          </div>
        </div>
      ))}
    </div>
  );
}

// Rolling call/put OI + volume ratio over distribution snapshots (ascending
// trade date). Rendered only when there is more than one snapshot.
function OptionsRatioChart({ rows }: { rows: MarketOpsOptionsDistributionView[] }) {
  const option = {
    grid: { left: 40, right: 16, top: 28, bottom: 40 },
    tooltip: { trigger: 'axis' },
    legend: { data: ['OI ratio', 'Volume ratio'], top: 0 },
    xAxis: { type: 'category', data: rows.map((r) => marketOpsOptionsDateOnly(r.tradeDate)) },
    yAxis: { type: 'value' },
    series: [
      { name: 'OI ratio', type: 'line', smooth: true, data: rows.map((r) => r.callPutOpenInterestRatio), itemStyle: { color: '#1f7a6b' } },
      { name: 'Volume ratio', type: 'line', smooth: true, data: rows.map((r) => r.callPutVolumeRatio), itemStyle: { color: '#b45309' } },
    ],
  };
  return <ReactECharts option={option} style={{ height: 200 }} />;
}
