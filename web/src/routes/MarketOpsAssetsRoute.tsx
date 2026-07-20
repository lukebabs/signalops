import { useState } from 'react';
import { Link } from '@tanstack/react-router';
import { CircleDollarSign, X } from 'lucide-react';
import ReactECharts from 'echarts-for-react';
import { useMarketOpsAssets } from '../api/queries';
import { useMarketOpsOptionsCoverage, useMarketOpsOptionsDistributions, useMarketOpsOptionsChain, useMarketOpsIntelligenceReadiness } from '../api/queries';
import { isApiError } from '../api/client';
import { EmptyState, ErrorState, LoadingState } from '../components/States';
import { StatusBadge } from '../components/StatusBadge';
import { MetricTile } from '../components/MetricTile';
import { JsonViewer } from '../components/JsonViewer';
import { OptionsQualityBadge } from '../components/OptionsQualityBadge';
import { formatUtc } from '../lib/format';
import { formatZeroRate } from '../lib/optionsQuality';
import {
  summarizeMarketOpsIntelligenceReadinessSymbol,
  summarizeMarketOpsIntelligenceReadinessAggregate,
  rolloutStatusStyle,
  dimensionStateStyle,
  formatCoverageRatio,
  type MarketOpsIntelligenceReadinessSymbolView,
} from '../lib/marketopsReadiness';
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
  const [tab, setTab] = useState<'assets' | 'readiness'>('assets');
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

      <div className="flex flex-wrap gap-1 border-b border-gray-200">
        {(['assets', 'readiness'] as const).map((t) => (
          <button
            key={t}
            type="button"
            onClick={() => setTab(t)}
            className={`-mb-px border-b-2 px-3 py-1.5 text-sm ${tab === t ? 'border-brand-600 font-semibold text-brand-700' : 'border-transparent text-gray-500 hover:text-gray-700'}`}
          >
            {t === 'assets' ? 'Assets' : 'Intelligence readiness'}
          </button>
        ))}
      </div>

      {tab === 'readiness' ? (
        <IntelligenceReadinessPanel tenantId={TENANT_ID} />
      ) : (
        <>
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
                        <div className="flex items-center gap-2 text-xs text-gray-500">
                          <span>{a.asset_type}</span>
                          <Link
                            to="/marketops/state"
                            search={{ symbol: a.ticker }}
                            onClick={(e) => e.stopPropagation()}
                            className="text-brand-700 underline hover:text-brand-800"
                          >
                            Open Market State
                          </Link>
                        </div>
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
        </>
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

            {/* G132 quality summary: ratio quality, open-interest quality, zero rate/counts, denominator-zero. */}
            <div className="flex flex-wrap items-center gap-2 text-[11px]">
              <OptionsQualityBadge quality={latest.ratioQuality} label={`Ratio ${latest.ratioQuality}`} />
              {latest.quality.openInterestQuality ? (
                <span className="inline-flex items-center gap-1 rounded border border-gray-200 bg-white px-1.5 py-0.5 text-gray-600">
                  OI <span className="font-medium">{latest.quality.openInterestQuality}</span>
                </span>
              ) : null}
              <span className="text-gray-600">Zero rate <strong className="text-gray-800">{formatZeroRate(latest.quality.openInterestZeroRate)}</strong></span>
              <span className="text-gray-600">Zero/positive <strong className="text-gray-800">{latest.quality.openInterestZeroCount ?? 0}/{latest.quality.openInterestPositiveCount ?? 0}</strong></span>
              {latest.quality.callPutOiDenominatorIsZero ? (
                <span className="inline-flex items-center rounded border border-red-300 bg-red-50 px-1.5 py-0.5 font-medium text-red-700">
                  Denominator zero — ratio not interpretable
                </span>
              ) : null}
            </div>

            {latest.provider || latest.sourceId ? (
              <div className="text-[11px] text-gray-500">
                Provider <span className="text-gray-700">{latest.provider || '—'}</span>
                {' · '}source <code className="text-gray-700">{latest.sourceId || '—'}</code>
              </div>
            ) : null}

            {chartRows.length > 1 ? <OptionsRatioChart rows={chartRows} /> : null}

            {/* G132 quality trend across snapshots so analysts can see quality change over time. */}
            {chartRows.length ? (
              <div className="flex flex-wrap items-center gap-1.5 text-[11px]">
                <span className="text-gray-500">Quality trend:</span>
                {chartRows.map((r) => (
                  <span key={r.tradeDate} className="inline-flex items-center gap-1">
                    <span className="text-gray-500">{marketOpsOptionsDateOnly(r.tradeDate)}</span>
                    <OptionsQualityBadge quality={r.ratioQuality} />
                  </span>
                ))}
              </div>
            ) : null}

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

            {/* G132 quality details disclosure. */}
            <details className="rounded border border-gray-200 bg-white p-2 text-xs">
              <summary className="cursor-pointer font-medium text-gray-600">Quality details</summary>
              <div className="mt-1 grid grid-cols-2 gap-1 text-gray-700">
                <div>Ratio quality: <strong>{latest.ratioQuality}</strong></div>
                <div>Open-interest quality: <strong>{latest.quality.openInterestQuality || '—'}</strong></div>
                <div>Zero count: <strong>{latest.quality.openInterestZeroCount ?? '—'}</strong></div>
                <div>Positive count: <strong>{latest.quality.openInterestPositiveCount ?? '—'}</strong></div>
                <div>Zero rate: <strong>{formatZeroRate(latest.quality.openInterestZeroRate)}</strong></div>
                <div>Denominator zero: <strong>{latest.quality.callPutOiDenominatorIsZero ? 'yes' : 'no'}</strong></div>
              </div>
              <p className="mt-1 text-[11px] text-gray-400">
                Quality is derived from persisted chain rows; non-usable ratios are skipped by the G131 proposal gate, not persisted as proposals.
              </p>
            </details>
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
                    <td className="whitespace-nowrap px-3 py-2 text-xs text-gray-800"><OpenInterestCell value={r.openInterest} /></td>
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

function OpenInterestCell({ value }: { value: number | null }) {
  if (value === null) return <span className="text-gray-400">—</span>;
  if (value === 0) {
    return (
      <span className="inline-flex items-center gap-1" title="Provider returned open_interest=0 for this contract">
        <span>0</span>
        <span className="rounded bg-amber-50 px-1 text-[10px] font-medium uppercase text-amber-700">zero OI</span>
      </span>
    );
  }
  return <>{String(value)}</>;
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

// G148-C intelligence readiness (read-only). One aggregate request serves the
// whole view — never per-symbol fan-out. Cards (not a wide table) keep reasons
// readable on narrow screens without horizontal-only interaction. Missing state
// renders as "Not observed," never zero coverage. No execution controls.
const ROLLOUT_STATUSES = ['', 'not_observed', 'inspection_ready', 'research_evaluation_ready', 'review_ready', 'blocked'];
const READINESS_LIMITS = [10, 25, 50];
const readinessInputCls = 'rounded border border-gray-300 px-2 py-1 text-sm';

function IntelligenceReadinessPanel({ tenantId }: { tenantId: string }) {
  const [universeGroup, setUniverseGroup] = useState('top50_megacap');
  const [symbols, setSymbols] = useState('');
  const [latestSession, setLatestSession] = useState('');
  const [rolloutStatus, setRolloutStatus] = useState('');
  const [limit, setLimit] = useState(50);

  const readinessQ = useMarketOpsIntelligenceReadiness({
    tenant_id: tenantId,
    universe_group: universeGroup.trim() || undefined,
    symbols: symbols.trim() || undefined,
    latest_session_date: latestSession || undefined,
    rollout_status: (rolloutStatus || undefined) as MarketOpsIntelligenceReadinessSymbolView['rolloutStatus'] | undefined,
    limit,
  });
  const aggregate = readinessQ.data ? summarizeMarketOpsIntelligenceReadinessAggregate(readinessQ.data.readiness.aggregate) : null;
  const rows = (readinessQ.data?.readiness.symbols ?? []).map(summarizeMarketOpsIntelligenceReadinessSymbol);
  const unauthorized =
    readinessQ.isError && isApiError(readinessQ.error) && (readinessQ.error.status === 401 || readinessQ.error.status === 403);
  const stale = readinessQ.isFetching && !!readinessQ.data;

  return (
    <div className="space-y-3">
      <div className="rounded border border-amber-200 bg-amber-50 px-2 py-1 text-[11px] font-medium text-amber-700">
        Research readiness only — production readiness is unsupported.
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <input value={universeGroup} onChange={(e) => setUniverseGroup(e.target.value)} className={readinessInputCls} aria-label="Universe group" placeholder="universe group" />
        <input value={symbols} onChange={(e) => setSymbols(e.target.value)} className={readinessInputCls} aria-label="Symbols (CSV)" placeholder="symbols (e.g. AAPL,MSFT)" />
        <input type="date" value={latestSession} onChange={(e) => setLatestSession(e.target.value)} className={readinessInputCls} aria-label="Latest session date" />
        <select value={rolloutStatus} onChange={(e) => setRolloutStatus(e.target.value)} className={readinessInputCls} aria-label="Rollout status">
          {ROLLOUT_STATUSES.map((r) => (<option key={r || 'any'} value={r}>{r || 'any rollout'}</option>))}
        </select>
        <select value={limit} onChange={(e) => setLimit(Number(e.target.value))} className={readinessInputCls} aria-label="Limit">
          {READINESS_LIMITS.map((n) => (<option key={n} value={n}>{n}</option>))}
        </select>
      </div>

      {readinessQ.isLoading && !readinessQ.data ? (
        <LoadingState label="Loading intelligence readiness..." />
      ) : unauthorized ? (
        <div className="rounded border border-red-200 bg-red-50 p-3 text-sm text-red-800">
          Unauthorized — this view requires an authenticated MarketOps session.
        </div>
      ) : readinessQ.isError ? (
        <ErrorState error={readinessQ.error} />
      ) : !aggregate || rows.length === 0 ? (
        <div className="rounded border border-gray-200 bg-white p-3">
          <EmptyState message="No durable cohort readiness exists for this scope." />
        </div>
      ) : (
        <div className="space-y-2">
          {/* Aggregate-first summary. */}
          <div className="rounded border border-gray-200 bg-gray-50 p-2 text-xs text-gray-700">
            <div className="flex flex-wrap items-center gap-x-4 gap-y-1">
              <span>Observed symbols <strong className="text-gray-800">{aggregate.symbolCount}</strong></span>
              <span>Latest session <strong className="text-gray-800">{aggregate.latestSessionDate ?? '—'}</strong></span>
              <span className="text-gray-500">production_ready_supported=<strong className="text-gray-800">false</strong></span>
              {stale ? <span className="text-[11px] text-gray-400">refreshing…</span> : null}
            </div>
            {aggregate.rolloutStatus.length ? (
              <div className="mt-1 flex flex-wrap items-center gap-1.5">
                <span className="text-gray-500">Rollout:</span>
                {aggregate.rolloutStatus.map((e) => (
                  <span key={e.key} className={`inline-flex items-center gap-1 rounded border px-1.5 py-0.5 text-[11px] font-medium ${rolloutStatusStyle(e.key)}`}>
                    {e.key} <strong>{e.count}</strong>
                  </span>
                ))}
              </div>
            ) : null}
          </div>

          {rows.map((s) => (
            <ReadinessSymbolCard key={s.resultId || s.symbol} s={s} />
          ))}
          {rows.length >= limit ? (
            <p className="text-[11px] text-gray-400">Showing the first {rows.length} symbols — narrow the scope or raise the limit to see more.</p>
          ) : null}
        </div>
      )}
    </div>
  );
}

function ReadinessDimensionBadge({ label, state }: { label: string; state: string }) {
  return (
    <span className="inline-flex items-center gap-1">
      <span className="text-[10px] uppercase text-gray-400">{label}</span>
      <span className={`inline-flex items-center rounded border px-1.5 py-0.5 text-[11px] font-medium ${dimensionStateStyle(state)}`}>{state || '—'}</span>
    </span>
  );
}

function ReadinessSymbolCard({ s }: { s: MarketOpsIntelligenceReadinessSymbolView }) {
  const observed = s.observed;
  return (
    <div className="rounded border border-gray-200 bg-white p-2">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <span className="font-mono text-sm font-semibold text-gray-900">{s.symbol || '—'}</span>
          <span className="text-[11px] text-gray-500">{s.universeGroup || '—'}</span>
        </div>
        <span className={`inline-flex items-center rounded border px-1.5 py-0.5 text-[11px] font-medium ${rolloutStatusStyle(s.rolloutStatus)}`}>rollout: {s.rolloutStatus || '—'}</span>
      </div>

      <div className="mt-1 text-[11px] text-gray-600">
        {observed ? (
          <>
            Latest {s.latestStateDate ? formatUtc(s.latestStateDate) : '—'} · schema <code>{s.latestStateSchemaVersion || '—'}</code> · quality {s.latestStateQuality || '—'}
            {' · '}completeness {formatCoverageRatio(s.latestStateCompleteness, true)}
            {' · '}required features {formatCoverageRatio(s.requiredFeatureCoverage, true)}
            {' · '}surface {formatCoverageRatio(s.surfaceCoverage, true)}
          </>
        ) : (
          <span className="text-gray-400">Not observed — no persisted market state.</span>
        )}
      </div>

      <div className="mt-1 flex flex-wrap items-center gap-1.5">
        <ReadinessDimensionBadge label="coverage" state={s.coverageState} />
        <ReadinessDimensionBadge label="evaluation" state={s.evaluationState} />
        <ReadinessDimensionBadge label="governance" state={s.governanceState} />
        <ReadinessDimensionBadge label="calibration" state={s.calibrationState} />
        <ReadinessDimensionBadge label="outcome" state={s.outcomeState} />
        {s.calibrationBelowMinimum ? (
          <span className="inline-flex items-center rounded border border-amber-300 bg-amber-50 px-1.5 py-0.5 text-[11px] font-medium text-amber-700">calibration below minimum</span>
        ) : null}
      </div>

      <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-0.5 text-[11px] text-gray-600">
        <span>eval {s.triggeredCount}/{s.eligibleCount}/{s.evaluationCount}</span>
        <span>opportunities {s.opportunityCount}</span>
        <span>outcomes {s.maturedOutcomeCount} matured / {s.pendingOutcomeCount} pending</span>
        <span>exact calibrations {s.exactCalibrationCount}</span>
      </div>

      {s.readinessReasons.length ? (
        <div className="mt-1 text-[11px] text-gray-700">
          <span className="text-gray-400">reasons: </span>{s.readinessReasons.join(' · ')}
        </div>
      ) : null}

      <div className="mt-1 flex flex-wrap items-center justify-between gap-2 text-[11px] text-gray-500">
        <span>cohort run <code className="break-all">{s.runId || '—'}</code></span>
        <div className="flex items-center gap-2">
          <details className="text-gray-500">
            <summary className="cursor-pointer">stage + inputs</summary>
            <div className="mt-1 space-y-1">
              <JsonViewer label="Stage status" value={s.stageStatus} />
              <JsonViewer label="Stage errors" value={s.stageErrors} />
              <JsonViewer label="Input coverage" value={s.inputCoverage} />
              <JsonViewer label="Proposal status counts" value={s.proposalStatusCounts} />
            </div>
          </details>
          {observed ? (
            <Link to="/marketops/state" search={{ symbol: s.symbol }} className="text-brand-700 underline hover:text-brand-800">Open Market State</Link>
          ) : null}
        </div>
      </div>
    </div>
  );
}
