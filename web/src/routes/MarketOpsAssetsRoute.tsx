import { Fragment, useMemo, useRef, useState } from 'react';
import { Link } from '@tanstack/react-router';
import { CircleDollarSign, X, ArrowDown, ArrowUp } from 'lucide-react';
import ReactECharts from 'echarts-for-react';
import { useMarketOpsAssets, useMarketOpsAssetQuotes, useMarketOpsIntradayConditions, useMarketOpsAssetAlgorithmObservations, useMarketOpsHypothesisEvaluations, useMarketOpsAlgorithmAdjudications, useMarketOpsQuantitativeSeries, useMarketOpsRiskRewardSummaries, useMarketOpsSignalOverview } from '../api/queries';
import { useMarketOpsOptionsCoverage, useMarketOpsOptionsDistributions, useMarketOpsOptionsChain, useMarketOpsIntelligenceReadiness } from '../api/queries';
import { api, isApiError } from '../api/client';
import { EmptyState, ErrorState, LoadingState } from '../components/States';
import { MetricTile } from '../components/MetricTile';
import { JsonViewer } from '../components/JsonViewer';
import { OptionsQualityBadge } from '../components/OptionsQualityBadge';
import { RiskRewardPanel } from '../components/RiskRewardPanel';
import { SignalOverviewCoverageChart, TechnicalScoreDistributionChart } from '../components/SignalOverviewAggregateCharts';
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
import type { AlgorithmResult, MarketOpsAssetQuote, MarketOpsEODZScore, MarketOpsIntradayConditionSnapshot, MarketOpsRiskRewardSummary } from "../types";

// Read-only MarketOps asset universe (G071 frontend) + G128 per-asset options
// intelligence panel. The universe table is backend data only; selecting a row
// opens a read-only options panel (coverage / distribution / chain) that
// performs no ingestion and never calls live-preview (which stays 501).

const CHAIN_LIMITS = [100, 200, 500];

type AssetSortKey = 'default' | 'rank' | 'asset' | 'market' | 'intraday' | 'riskReward' | 'updated';
type AssetColumnKey = Exclude<AssetSortKey, 'default'>;

const ASSET_COLUMN_WIDTHS_KEY = 'signalops.marketops.assets.column-widths.v1';
const ASSET_SELECTION_COLUMN_WIDTH = 44;
const DEFAULT_ASSET_COLUMN_WIDTHS: Record<AssetColumnKey, number> = { rank: 70, asset: 290, market: 220, intraday: 230, riskReward: 170, updated: 205 };

function assetRowKey(asset: { universe_group: string; ticker: string }): string {
  return asset.universe_group + ":" + asset.ticker;
}

function assetSectorLabel(asset: { display_sector?: string; sector?: string }): string {
  return (asset.display_sector || asset.sector || "unclassified").toLocaleLowerCase().replace(/[-_]+/g, " · ").replace(/\s+/g, " ").trim();
}

export function MarketOpsAssetsRoute() {
  const TENANT_ID = useTenant();
  const [selectedTicker, setSelectedTicker] = useState<string | null>(null);
  const [editSelectedAssetKey, setEditSelectedAssetKey] = useState<string | null>(null);
  const [displayNameInput, setDisplayNameInput] = useState('');
  const [displayNameBusy, setDisplayNameBusy] = useState(false);
  const [displayNameError, setDisplayNameError] = useState<string | null>(null);
  const [assetSort, setAssetSort] = useState<{ key: AssetSortKey; direction: 'asc' | 'desc' }>({ key: 'default', direction: 'asc' });
  const [displaySectorInput, setDisplaySectorInput] = useState('');
  const [displaySectorBusy, setDisplaySectorBusy] = useState(false);
  const [displaySectorError, setDisplaySectorError] = useState<string | null>(null);
  const [columnWidths, setColumnWidths] = useState<Record<AssetColumnKey, number>>(() => {
    if (typeof window === 'undefined') return DEFAULT_ASSET_COLUMN_WIDTHS;
    try {
      const stored = JSON.parse(window.localStorage.getItem(ASSET_COLUMN_WIDTHS_KEY) || '{}') as Partial<Record<AssetColumnKey, unknown>>;
      return (Object.keys(DEFAULT_ASSET_COLUMN_WIDTHS) as AssetColumnKey[]).reduce((widths, key) => {
        const value = stored[key];
        widths[key] = typeof value === 'number' && Number.isFinite(value) ? Math.max(64, Math.min(560, value)) : DEFAULT_ASSET_COLUMN_WIDTHS[key];
        return widths;
      }, {} as Record<AssetColumnKey, number>);
    } catch {
      return DEFAULT_ASSET_COLUMN_WIDTHS;
    }
  });
  const columnWidthsRef = useRef(columnWidths);
  columnWidthsRef.current = columnWidths;
  const query = useMarketOpsAssets({
    tenant_id: TENANT_ID,
    universe_group: 'top50_megacap',
    active_only: true,
    limit: 50,
  });

  // Sort defensively by rank so the displayed order is stable regardless of
  // backend ordering; slice() avoids mutating the cached response.
  const watchlistQ = useMarketOpsAssets({ tenant_id: TENANT_ID, universe_group: "analyst_watchlist", active_only: true, limit: 50 });
  const data = [...(query.data?.assets ?? []), ...(watchlistQ.data?.assets ?? [])].slice().sort((a, b) => a.universe_group === b.universe_group ? a.rank - b.rank : a.universe_group === "top50_megacap" ? -1 : 1);
  const active = data.filter((a) => a.is_active).length;
  const sectors = new Set(data.map((a) => a.sector_key || a.sector).filter(Boolean)).size;
  const industries = new Set(data.map((a) => a.industry_key || a.industry).filter(Boolean)).size;
  const quotesQ = useMarketOpsAssetQuotes(TENANT_ID, "top50_megacap");
  const watchlistQuotesQ = useMarketOpsAssetQuotes(TENANT_ID, "analyst_watchlist");
  const quoteMap = new Map([...(quotesQ.data?.quotes ?? []), ...(watchlistQuotesQ.data?.quotes ?? [])].map((q) => [q.ticker.toUpperCase(), q]));
  const conditionsQ = useMarketOpsIntradayConditions(TENANT_ID, "top50_megacap");
  const watchlistConditionsQ = useMarketOpsIntradayConditions(TENANT_ID, "analyst_watchlist");
  const conditionMap = new Map([...(conditionsQ.data?.snapshots ?? []), ...(watchlistConditionsQ.data?.snapshots ?? [])].map((snapshot) => [snapshot.ticker.toUpperCase(), snapshot]));
  const riskRewardQ = useMarketOpsRiskRewardSummaries(TENANT_ID, "top50_megacap");
  const watchlistRiskRewardQ = useMarketOpsRiskRewardSummaries(TENANT_ID, "analyst_watchlist");
  const riskRewardMap = new Map([...(riskRewardQ.data?.summaries ?? []), ...(watchlistRiskRewardQ.data?.summaries ?? [])].map((summary) => [summary.ticker.toUpperCase(), summary]));

  function toggleAssetEditSelection(asset: { ticker: string; universe_group: string; display_name?: string; company: string; display_sector?: string; sector?: string }) {
    const key = assetRowKey(asset);
    if (editSelectedAssetKey === key) {
      setEditSelectedAssetKey(null);
      setDisplayNameError(null);
      setDisplaySectorError(null);
      return;
    }
    setEditSelectedAssetKey(key);
    setDisplayNameError(null);
    setDisplaySectorError(null);
    setDisplayNameInput(asset.display_name || asset.company);
    setDisplaySectorInput(asset.display_sector || asset.sector || "");
  }

  function cancelDisplayNameEdit(asset: { display_name?: string; company: string }) {
    setDisplayNameInput(asset.display_name || asset.company);
    setDisplayNameError(null);
  }

  function cancelDisplaySectorEdit(asset: { display_sector?: string; sector?: string }) {
    setDisplaySectorInput(asset.display_sector || asset.sector || "");
    setDisplaySectorError(null);
  }

  async function saveDisplayName(event: React.FormEvent, asset: { ticker: string; universe_group: string }) {
    event.preventDefault();
    const displayName = displayNameInput.trim();
    if (!displayName) {
      setDisplayNameError("A display name is required.");
      return;
    }
    setDisplayNameBusy(true);
    setDisplayNameError(null);
    try {
      await api.updateMarketOpsAssetDisplayName(TENANT_ID, asset.ticker, { universe_group: asset.universe_group, display_name: displayName });
      await Promise.all([query.refetch(), watchlistQ.refetch()]);

    } catch (error) {
      setDisplayNameError(error instanceof Error ? error.message : "Display name could not be updated.");
    } finally {
      setDisplayNameBusy(false);
    }
  }

  async function saveDisplaySector(event: React.FormEvent, asset: { ticker: string; universe_group: string }) {
    event.preventDefault();
    const displaySector = displaySectorInput.trim();
    if (!displaySector) {
      setDisplaySectorError("A sector label is required.");
      return;
    }
    setDisplaySectorBusy(true);
    setDisplaySectorError(null);
    try {
      await api.updateMarketOpsAssetDisplaySector(TENANT_ID, asset.ticker, { universe_group: asset.universe_group, display_sector: displaySector });
      await Promise.all([query.refetch(), watchlistQ.refetch()]);

    } catch (error) {
      setDisplaySectorError(error instanceof Error ? error.message : "Sector label could not be updated.");
    } finally {
      setDisplaySectorBusy(false);
    }
  }

  const sortedData = useMemo(() => {
    if (assetSort.key === 'default') return data;
    const valueFor = (asset: typeof data[number]): string | number | null => {
      const quote = quoteMap.get(asset.ticker.toUpperCase());
      const condition = conditionMap.get(asset.ticker.toUpperCase());
      const riskReward = riskRewardMap.get(asset.ticker.toUpperCase());
      switch (assetSort.key) {
        case 'rank': return asset.rank;
        case 'asset': return (asset.display_name || asset.company || asset.ticker).toLocaleLowerCase();
        case 'market': return quote?.change_percent ?? null;
        case 'intraday': return condition?.conditions.reduce((score, item) => Math.max(score, item.score), Number.NEGATIVE_INFINITY) ?? null;
        case 'riskReward': return riskReward?.score ?? null;
        case 'updated': return quote?.timestamp ? Date.parse(quote.timestamp) : null;
        default: return null;
      }
    };
    return data.slice().sort((left, right) => {
      const leftValue = valueFor(left);
      const rightValue = valueFor(right);
      if (leftValue == null && rightValue == null) return left.ticker.localeCompare(right.ticker);
      if (leftValue == null) return 1;
      if (rightValue == null) return -1;
      const comparison = typeof leftValue === 'string' && typeof rightValue === 'string' ? leftValue.localeCompare(rightValue) : Number(leftValue) - Number(rightValue);
      return comparison === 0 ? left.ticker.localeCompare(right.ticker) : comparison * (assetSort.direction === 'asc' ? 1 : -1);
    });
  }, [assetSort, conditionMap, data, quoteMap, riskRewardMap]);

  function toggleAssetSort(key: Exclude<AssetSortKey, 'default'>) {
    setAssetSort((current) => current.key === key ? { key, direction: current.direction === 'asc' ? 'desc' : 'asc' } : { key, direction: key === 'asset' || key === 'rank' ? 'asc' : 'desc' });
  }

  function beginColumnResize(event: React.PointerEvent<HTMLButtonElement>, key: AssetColumnKey) {
    event.preventDefault();
    event.stopPropagation();
    const startX = event.clientX;
    const startWidth = columnWidthsRef.current[key];
    const onMove = (moveEvent: PointerEvent) => {
      const nextWidth = Math.max(64, Math.min(560, startWidth + moveEvent.clientX - startX));
      setColumnWidths((current) => ({ ...current, [key]: nextWidth }));
    };
    const onUp = () => {
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup", onUp);
      window.localStorage.setItem(ASSET_COLUMN_WIDTHS_KEY, JSON.stringify(columnWidthsRef.current));
    };
    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", onUp);
  }

  function resetColumnWidths() {
    setColumnWidths(DEFAULT_ASSET_COLUMN_WIDTHS);
    window.localStorage.removeItem(ASSET_COLUMN_WIDTHS_KEY);
  }

  const totalColumnWidth = ASSET_SELECTION_COLUMN_WIDTH + (Object.values(columnWidths) as number[]).reduce((total, width) => total + width, 0);


  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-lg font-semibold">Assets</h1>
          <p className="text-xs text-gray-500">Tenant {TENANT_ID} · combined market universe</p>
        </div>
      </div>

      <WatchlistControls tenantId={TENANT_ID} onChanged={() => { void query.refetch(); void watchlistQ.refetch(); }} />

      <div className="grid grid-cols-2 gap-2 md:grid-cols-5">
        <MetricTile label="Universe Assets" value={data.length} />
        <MetricTile label="Active Assets" value={active} />
        <MetricTile label="Sectors" value={sectors} />
        <MetricTile label="Industries" value={industries} />
        <MetricTile label="Market Quotes" value={quoteMap.size} />
      </div>

      {query.isLoading ? (
        <LoadingState />
      ) : query.isError ? (
        <ErrorState error={query.error} />
      ) : data.length ? (
        <div className="space-y-1">
          <div className="flex items-center justify-between px-1">
            <p className="text-xs text-gray-500 md:hidden">Swipe horizontally to view all asset columns.</p>
            <button type="button" onClick={resetColumnWidths} className="ml-auto text-xs text-brand-700 underline hover:text-brand-800">Reset column widths</button>
          </div>
          <div
            className="max-w-full overflow-x-auto rounded border border-gray-200 bg-white overscroll-x-contain"
            role="region"
            aria-label="Scrollable asset table"
            tabIndex={0}
          >
          <table className="table-fixed divide-y divide-gray-200 text-sm" style={{ width: totalColumnWidth }}>
            <colgroup><col style={{ width: ASSET_SELECTION_COLUMN_WIDTH }} />{(Object.keys(DEFAULT_ASSET_COLUMN_WIDTHS) as AssetColumnKey[]).map((key) => <col key={key} style={{ width: columnWidths[key] }} />)}</colgroup>
            <thead className="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500">
              <tr>
                <th className="w-11 px-2 py-2 text-center"><span className="sr-only">Edit selection</span></th>
                <SortableAssetHeader label="Rank" sortKey="rank" activeSort={assetSort} onSort={toggleAssetSort} width={columnWidths.rank} onResize={beginColumnResize} />
                <SortableAssetHeader label="Asset" sortKey="asset" activeSort={assetSort} onSort={toggleAssetSort} width={columnWidths.asset} onResize={beginColumnResize} />
                <SortableAssetHeader label="Current Market Data ⓘ" sortKey="market" activeSort={assetSort} onSort={toggleAssetSort} width={columnWidths.market} onResize={beginColumnResize} title="Latest delayed intraday price while the market is open; latest completed EOD close otherwise. Sorts by displayed percentage move." />
                <SortableAssetHeader label="Intraday Conditions ⓘ" sortKey="intraday" activeSort={assetSort} onSort={toggleAssetSort} width={columnWidths.intraday} onResize={beginColumnResize} title="Sorts by the strongest current 15-minute condition score for each asset." />
                <SortableAssetHeader label="Risk/Reward ⓘ" sortKey="riskReward" activeSort={assetSort} onSort={toggleAssetSort} width={columnWidths.riskReward} onResize={beginColumnResize} title="Sorts by the latest persisted post-close technical Risk/Reward score." />
                <SortableAssetHeader label="Updated" sortKey="updated" activeSort={assetSort} onSort={toggleAssetSort} width={columnWidths.updated} onResize={beginColumnResize} title="Sorts by the timestamp of the displayed quote." />
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {sortedData.map((a) => (
                <Fragment key={a.ticker}>
                <tr
                  onClick={() => setSelectedTicker((current) => current === a.ticker ? null : a.ticker)}
                  className={`cursor-pointer align-top hover:bg-gray-50 ${selectedTicker === a.ticker ? 'bg-brand-50' : ''}`}
                >
                  <td className="px-2 py-2 text-center"><input type="checkbox" checked={editSelectedAssetKey === assetRowKey(a)} onClick={(event) => event.stopPropagation()} onChange={() => toggleAssetEditSelection(a)} aria-label={"Select " + a.ticker + " for display-name and sector editing"} title="Select this asset to edit its display name and sector" className="h-4 w-4 rounded border-gray-300 text-brand-700 focus:ring-brand-600" /></td>
                  <td className="px-3 py-2 text-xs text-gray-500">{a.rank}</td>
                  <td className="px-3 py-2">
                    <div className="flex items-start gap-2">
                      <CircleDollarSign size={16} className="mt-0.5 text-brand-700" />
                      <div>
                        {editSelectedAssetKey === assetRowKey(a) ? (
                          <form onSubmit={(event) => void saveDisplayName(event, a)} onClick={(event) => event.stopPropagation()} className="flex items-center gap-1">
                            <input aria-label={"Display name for " + a.ticker} value={displayNameInput} onChange={(event) => setDisplayNameInput(event.target.value)} maxLength={120} autoFocus className="w-44 rounded border border-gray-300 px-1.5 py-0.5 text-sm text-gray-900" />
                            <button type="submit" disabled={displayNameBusy} className="rounded bg-brand-700 px-1.5 py-0.5 text-[10px] font-medium text-white disabled:opacity-50">Save</button>
                            <button type="button" disabled={displayNameBusy} onClick={() => cancelDisplayNameEdit(a)} className="text-[10px] text-gray-600 underline">Cancel</button>
                          </form>
                        ) : <div className="font-medium text-gray-900">{a.display_name || a.company || "—"}</div>}
                        {editSelectedAssetKey === assetRowKey(a) && displayNameError ? <div className="mt-1 text-[10px] text-red-700">{displayNameError}</div> : null}
                        <div className="flex items-center gap-1 font-mono text-xs text-gray-500"><span>{a.ticker}</span>{editSelectedAssetKey === assetRowKey(a) ? (
                          <form onSubmit={(event) => void saveDisplaySector(event, a)} onClick={(event) => event.stopPropagation()} className="flex items-center gap-1 font-sans">
                            <input aria-label={"Sector label for " + a.ticker} value={displaySectorInput} onChange={(event) => setDisplaySectorInput(event.target.value)} maxLength={48} className="w-28 rounded border border-gray-300 px-1 py-0.5 text-[10px] text-gray-900" />
                            <button type="submit" disabled={displaySectorBusy} className="rounded bg-brand-700 px-1 py-0.5 text-[9px] font-medium text-white disabled:opacity-50">Save</button>
                            <button type="button" disabled={displaySectorBusy} onClick={() => cancelDisplaySectorEdit(a)} className="text-[9px] text-gray-600 underline">Cancel</button>
                          </form>
                        ) : <span className="inline-flex max-w-40 rounded bg-slate-100 px-1.5 py-0.5 font-sans text-[10px] text-slate-600" title={a.display_sector || a.sector || "Unclassified sector"}><span className="truncate">{assetSectorLabel(a)}</span></span>}</div>
                        {editSelectedAssetKey === assetRowKey(a) && displaySectorError ? <div className="mt-1 text-[10px] text-red-700">{displaySectorError}</div> : null}
                        <div className="flex items-center gap-2 text-xs text-gray-500">
                          <span>{a.asset_type}</span>
                          <Link to="/marketops/state" search={{ symbol: a.ticker }} onClick={(e) => e.stopPropagation()} className="text-brand-700 underline hover:text-brand-800">Open Market State</Link>
                        </div>
                      </div>
                    </div>
                  </td>
                  <MarketDataCell quote={quoteMap.get(a.ticker.toUpperCase())} />
                  <IntradayConditionsCell snapshot={conditionMap.get(a.ticker.toUpperCase())} loading={conditionsQ.isLoading} error={conditionsQ.isError} />
                  <RiskRewardCell summary={riskRewardMap.get(a.ticker.toUpperCase())} loading={riskRewardQ.isLoading} error={riskRewardQ.isError} />
                  <MarketDataUpdatedCell quote={quoteMap.get(a.ticker.toUpperCase())} refreshedAt={quotesQ.data?.refreshed_at ?? null} />
                </tr>
                {selectedTicker === a.ticker ? (
                  <tr className="bg-brand-50">
                    <td colSpan={7} className="p-3">
                      <AssetOptionsPanel tenantId={TENANT_ID} symbol={selectedTicker} onClose={() => setSelectedTicker(null)} />
                    </td>
                  </tr>
                ) : null}
                </Fragment>
              ))}
            </tbody>
          </table>
          </div>
        </div>
      ) : (
        <EmptyState message="No MarketOps assets found for this tenant." />
      )}

      {!selectedTicker ? (
        <div className="rounded border border-gray-200 bg-white p-3">
          <EmptyState message="Select an asset to expand its intraday evolution and persisted options intelligence." />
        </div>
      ) : null}
    </div>
  );
}
function SortableAssetHeader({ label, sortKey, activeSort, onSort, width, onResize, title }: { label: string; sortKey: AssetColumnKey; activeSort: { key: AssetSortKey; direction: 'asc' | 'desc' }; onSort: (key: AssetColumnKey) => void; width: number; onResize: (event: React.PointerEvent<HTMLButtonElement>, key: AssetColumnKey) => void; title?: string }) {
  const active = activeSort.key === sortKey;
  return <th className="relative px-3 py-2" style={{ width }} aria-sort={active ? (activeSort.direction === "asc" ? "ascending" : "descending") : "none"} title={title}><button type="button" onClick={() => onSort(sortKey)} className="inline-flex max-w-full items-center gap-1 pr-1 hover:text-gray-800">{label}<span aria-hidden="true" className={active ? "text-brand-700" : "text-gray-300"}>{active ? (activeSort.direction === "asc" ? "↑" : "↓") : "↕"}</span></button><button type="button" aria-label={"Resize " + label + " column"} onPointerDown={(event) => onResize(event, sortKey)} className="absolute right-0 top-0 h-full w-2 cursor-col-resize touch-none border-r border-transparent hover:border-brand-500" /></th>;
}

function AlgorithmResultLine({ result }: { result: AlgorithmResult }) {
  const payload = result.result_payload && typeof result.result_payload === "object" ? result.result_payload as Record<string, unknown> : {};
  const featureKey = typeof payload.feature === "string" ? payload.feature : "";
  const feature = featureKey ? featureKey.replace(/_/g, " ") : result.result_type;
  const quality = typeof payload.call_put_oi_ratio_quality === "string" ? payload.call_put_oi_ratio_quality : null;
  const rawValue = typeof payload.value === "number" ? payload.value : null;
  const putCall = rawValue != null && rawValue > 0 ? featureKey === "call_put_volume_ratio" ? 1 / rawValue : featureKey === "put_call_volume_ratio" ? rawValue : null : null;
  const sentiment = putCall == null ? null : putCall < 1 ? "bullish · calls elevated" : putCall > 1 ? "bearish · puts elevated" : "neutral";
  return <div className="border-t border-violet-100 pt-1 first:border-t-0 first:pt-0"><div className="text-xs font-medium text-violet-900">{result.algorithm_id.replace("signalops.algorithms.", "").replace(/_/g, " ")} · {feature} · {result.severity}</div><div className="text-[11px] text-gray-600">Score {result.score.toFixed(2)} · confidence {(result.confidence * 100).toFixed(0)}% · {formatUtc(result.created_at)}</div>{putCall != null ? <div className={putCall < 1 ? "text-[11px] text-green-700" : putCall > 1 ? "text-[11px] text-red-700" : "text-[11px] text-gray-600"}>Put/call volume {putCall.toFixed(2)} · {sentiment}</div> : null}{quality ? <div className="text-[11px] text-gray-500">Options-ratio quality: {quality}</div> : null}</div>;
}

function QuantitativeCorroborationPanel({ eod, loading, error }: { eod: MarketOpsEODZScore[]; loading: boolean; error: boolean }) {
  return <div className="rounded border border-violet-100 bg-violet-50 p-2"><div className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-violet-700">Quantitative corroboration</div><p className="mb-2 text-[11px] text-violet-700">One strongest usable z-score per EOD across the latest three trading days. Independent evidence; it does not alter research hypotheses or recommendations.</p>{loading ? <div className="text-xs text-gray-500">Loading curated EOD observations…</div> : error ? <div className="text-xs text-red-700">Curated algorithm observations are unavailable.</div> : eod.length ? <div className="space-y-2">{eod.map((item) => <div key={item.trade_date}><div className="text-[10px] font-semibold text-violet-700">{item.trade_date}</div>{item.algorithm_result ? <AlgorithmResultLine result={item.algorithm_result} /> : <div className="text-xs text-gray-500">No usable z-score · {item.reason ?? "No post-close z-score was available."}</div>}</div>)}</div> : <div className="text-xs text-gray-500">No post-close z-score is available for the latest trading days.</div>}</div>;
}

function AlgorithmEvidencePanel({ results, loading, error }: { results: AlgorithmResult[]; loading: boolean; error: boolean }) {
  const recent = results.slice(0, 5);
  const grouped = recent.reduce<Record<string, AlgorithmResult[]>>((groups, result) => {
    const payload = result.result_payload && typeof result.result_payload === "object" ? result.result_payload as Record<string, unknown> : {};
    const observed = typeof payload.observation_time === "string" ? payload.observation_time.slice(0, 10) : result.created_at.slice(0, 10);
    (groups[observed] ??= []).push(result);
    return groups;
  }, {});
  const dates = Object.keys(grouped).sort().reverse();
  return <div className="rounded border border-violet-100 bg-violet-50 p-2"><div className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-violet-700">Algorithm evidence</div><p className="mb-2 text-[11px] text-violet-700">The five most recent raw platform observations, retained for deeper analysis. The selected EOD z-score is intentionally excluded.</p>{loading ? <div className="text-xs text-gray-500">Loading algorithm evidence…</div> : error ? <div className="text-xs text-red-700">Algorithm evidence is unavailable.</div> : dates.length ? <div className="space-y-3">{dates.map((date) => <div key={date}><div className="text-[10px] font-semibold text-violet-700">{date}</div><div className="space-y-2">{grouped[date].map((result) => <AlgorithmResultLine key={result.algorithm_result_id} result={result} />)}</div></div>)}</div> : <div className="text-xs text-gray-500">No additional platform outputs are available for this asset.</div>}</div>;
}

function RiskRewardCell({ summary, loading, error }: { summary?: MarketOpsRiskRewardSummary; loading: boolean; error: boolean }) {
  if (error && !summary) return <td className="px-3 py-2 text-xs text-red-700" title="The persisted post-close Risk/Reward summary could not be loaded. This does not indicate an asset-level signal.">Risk/Reward unavailable</td>;
  if (loading && !summary) return <td className="px-3 py-2 text-xs text-gray-500" title="Loading the latest persisted post-close technical Risk/Reward analysis.">Loading EOD analysis…</td>;
  if (!summary) return <td className="px-3 py-2 text-xs text-gray-500" title="No persisted post-close Risk/Reward analysis is available yet. The scheduled EOD algorithm will populate this after its first completed run.">Awaiting EOD analysis</td>;
  const direction = summary.direction === "bullish" ? "Bullish" : summary.direction === "bearish" ? "Bearish" : "Neutral";
  const score = `${summary.score > 0 ? "+" : ""}${summary.score.toFixed(0)}`;
  const tone = summary.direction === "bullish" ? "text-green-700" : summary.direction === "bearish" ? "text-red-700" : "text-gray-700";
  const confidence = `${Math.round(summary.confidence * 100)}%`;
  const scoreChange = summary.score_change;
  const evolution = scoreChange == null ? "Awaiting prior EOD" : scoreChange > 0 ? `↑ Improving · +${scoreChange.toFixed(0)}` : scoreChange < 0 ? `↓ Regressing · ${scoreChange.toFixed(0)}` : "→ Unchanged · 0";
  const evolutionTone = scoreChange == null || scoreChange === 0 ? "text-gray-500" : scoreChange > 0 ? "text-green-700" : "text-red-700";
  const title = `Post-close ${summary.trade_date} · ${direction} technical posture · score ${score}/100 · confidence ${confidence} · ATR risk ${summary.risk_level}.${scoreChange == null ? " No prior persisted trading-session score is available." : ` ${evolution} versus ${summary.previous_trade_date ?? "the prior persisted session"}.`} Research-only; not a recommendation.`;
  return <td className="px-3 py-2 text-xs" title={title}><div className={`font-medium ${tone}`}>{direction} · {score}</div><div className={`text-[10px] ${evolutionTone}`}>{evolution}</div></td>;
}

function IntradayConditionsCell({ snapshot, loading, error }: { snapshot?: MarketOpsIntradayConditionSnapshot; loading: boolean; error: boolean }) {
  if (error && !snapshot) return <td className="px-3 py-2 text-xs text-red-700" title="The persisted intraday-condition request failed; this does not mean monitoring has not run.">Monitor data unavailable</td>;
  if (loading && !snapshot) return <td className="px-3 py-2 text-xs text-gray-500" title="Loading the latest persisted 15-minute condition snapshot.">Loading monitor…</td>;
  if (!snapshot) return <td className="px-3 py-2 text-xs text-gray-500" title="No persisted 15-minute condition snapshot is available yet.">Awaiting monitor</td>;
  if (!snapshot.conditions.length) return <td className="px-3 py-2 text-xs text-gray-500" title="The latest monitor snapshot found no price-action condition above its thresholds.">No active condition</td>;
  const top = snapshot.conditions.slice().sort((a, b) => b.score - a.score).slice(0, 2);
  return <td className="px-3 py-2 text-xs"><div className="space-y-1">{top.map((item) => <div key={item.key} title={item.evidence + " " + item.interpretation} className={item.tone === "positive" ? "cursor-help text-green-700" : item.tone === "negative" ? "cursor-help text-red-700" : "cursor-help text-gray-700"}>{item.title}</div>)}{snapshot.conditions.length > 2 ? <div className="text-[10px] text-gray-500">+{snapshot.conditions.length - 2} more</div> : null}<div className="text-[10px] text-gray-400">{snapshot.stale ? "Stale" : "15m monitor"}</div></div></td>;
}

function MarketDataCell({ quote }: { quote?: MarketOpsAssetQuote }) {
  if (quote?.price == null) {
    return <td className="px-3 py-2 text-xs text-gray-500" title="No delayed intraday aggregate or completed daily close is currently available from the market-data provider.">Unavailable</td>;
  }
  const change = quote.change_percent;
  const positive = change != null && change > 0;
  const negative = change != null && change < 0;
  const status = quote.market_status === 'extended' ? '15-minute delayed extended session' : quote.market_status === 'regular' ? '15-minute delayed regular session' : 'EOD close';
  const valueHelp = `${status}. ${change == null ? 'Daily change is unavailable.' : `Move versus the prior completed close: ${change > 0 ? '+' : ''}${change.toFixed(2)}%.`} ${quote.timestamp ? `As of ${formatUtc(quote.timestamp)}.` : ''}`;
  const hasRange = quote.week52_low != null && quote.week52_high != null && quote.week52_high > quote.week52_low;
  const rangePosition = hasRange ? Math.max(0, Math.min(100, ((quote.price - quote.week52_low!) / (quote.week52_high! - quote.week52_low!)) * 100)) : null;
  const rangeHelp = hasRange ? `52-week range: low ${quote.week52_low!.toFixed(2)}, high ${quote.week52_high!.toFixed(2)}. Current price sits at ${rangePosition!.toFixed(0)}% of that range; nearer the right edge means nearer the 52-week high.` : '';
  return (
    <td className="px-3 py-2 text-xs">
      <span title={valueHelp} className={positive ? 'inline-flex cursor-help items-center gap-1 font-medium text-green-700' : negative ? 'inline-flex cursor-help items-center gap-1 font-medium text-red-700' : 'inline-flex cursor-help items-center gap-1 text-gray-700'}>
        {positive ? <ArrowUp size={14} aria-label="Up" /> : negative ? <ArrowDown size={14} aria-label="Down" /> : null}
        {quote.price.toFixed(2)} · {change == null ? '—' : `${change > 0 ? '+' : ''}${change.toFixed(2)}%`}
        {quote.market_status === 'end_of_day' ? <span className="rounded bg-gray-100 px-1 text-[9px] font-medium text-gray-600">EOD</span> : quote.market_status === 'extended' ? <span className="rounded bg-blue-100 px-1 text-[9px] font-medium text-blue-700">Extended</span> : <span className="rounded bg-green-100 px-1 text-[9px] font-medium text-green-700">Regular</span>}
      </span>
      {hasRange ? (
        <span className="mt-1 block w-28 cursor-help" title={rangeHelp}>
          <span className="relative block h-1 rounded bg-gray-200"><span className="absolute left-0 top-0 h-1 rounded bg-brand-500" style={{ width: `${rangePosition}%` }} /></span>
          <span className="mt-0.5 flex justify-between text-[9px] text-gray-400"><span>{quote.week52_low!.toFixed(0)}</span><span>{quote.week52_high!.toFixed(0)}</span></span>
        </span>
      ) : null}
    </td>
  );
}

function MarketDataUpdatedCell({ quote, refreshedAt }: { quote?: MarketOpsAssetQuote; refreshedAt?: string | null }) {
  if (quote?.market_status === 'end_of_day') {
    const sessionDate = quote.timestamp ? quote.timestamp.slice(0, 10) : 'latest session';
    return <td className="px-3 py-2 text-xs text-gray-600" title={`Latest completed daily close for ${sessionDate}. The quote view was last refreshed ${refreshedAt ? formatUtc(refreshedAt) : 'recently'}.`}>EOD · {sessionDate}</td>;
  }
  if (refreshedAt) {
    const session = quote?.market_status === 'extended' ? 'Extended' : 'Regular';
    return <td className="px-3 py-2 text-xs text-gray-600" title={`The ${session.toLowerCase()}-session quote view was last refreshed at ${formatUtc(refreshedAt)}. Prices are delayed by the market-data entitlement.`}>{session} · {formatUtc(refreshedAt)}</td>;
  }
  return <td className="px-3 py-2 text-xs text-gray-500">Awaiting refresh</td>;
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
  const [seriesWindow, setSeriesWindow] = useState("10_trade_days");
  const seriesQ = useMarketOpsQuantitativeSeries(tenantId, symbol, seriesWindow);
  const coverageQ = useMarketOpsOptionsCoverage(tenantId, symbol);
  const distQ = useMarketOpsOptionsDistributions(tenantId, symbol, { window: '10_trade_days', limit: 10 });
  const intradayQ = useMarketOpsIntradayConditions(tenantId, "top50_megacap", symbol);
  const corroborationQ = useMarketOpsAssetAlgorithmObservations(tenantId, symbol);
  const [analysisTab, setAnalysisTab] = useState<"overview" | "algorithm_evidence">("overview");
  const hypothesisQ = useMarketOpsHypothesisEvaluations({ tenant_id: tenantId, symbol, triggered: true, limit: 12 });
  const adjudicationsQ = useMarketOpsAlgorithmAdjudications({ tenant_id: tenantId, symbol, limit: 12 });
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

      <div className="flex gap-1 border-b border-gray-200">
        <button type="button" onClick={() => setAnalysisTab("overview")} className={`-mb-px border-b-2 px-2 py-1 text-xs ${analysisTab === "overview" ? "border-violet-600 font-semibold text-violet-700" : "border-transparent text-gray-500"}`}>Overview</button>
        <button type="button" onClick={() => setAnalysisTab("algorithm_evidence")} className={`-mb-px border-b-2 px-2 py-1 text-xs ${analysisTab === "algorithm_evidence" ? "border-violet-600 font-semibold text-violet-700" : "border-transparent text-gray-500"}`}>Algorithm Evidence</button>
      </div>
      {analysisTab === "algorithm_evidence" ? <><AlgorithmEvidencePanel results={corroborationQ.data?.other_outputs ?? []} loading={corroborationQ.isLoading} error={corroborationQ.isError} /><QuantitativeCorroborationPanel eod={corroborationQ.data?.eod_zscores ?? []} loading={corroborationQ.isLoading} error={corroborationQ.isError} /></> : <>
        <QuantitativeSeriesChart points={seriesQ.data?.points ?? []} window={seriesWindow} onWindowChange={setSeriesWindow} loading={seriesQ.isLoading} error={seriesQ.isError} />

        <QuantitativeCorroborationPanel eod={corroborationQ.data?.eod_zscores ?? []} loading={corroborationQ.isLoading} error={corroborationQ.isError} />
        {adjudicationsQ.data?.algorithm_adjudications.length ? <div className="rounded border border-violet-200 bg-white p-2 text-[11px] text-violet-800"><div className="font-semibold uppercase">Independent adjudication</div>{adjudicationsQ.data.algorithm_adjudications.slice(0,4).map((item) => <div key={item.adjudication_id} className="mt-1"><span className={item.verdict === "confirmed" ? "font-medium text-green-700" : item.verdict === "contradicted" ? "font-medium text-red-700" : "font-medium text-gray-600"}>{item.verdict}</span> · {item.hypothesis_key} · confidence {(item.confidence * 100).toFixed(0)}%</div>)}</div> : hypothesisQ.data?.hypothesis_evaluations.length ? <div className="rounded border border-violet-100 bg-white p-2 text-[11px] text-violet-800">Triggered hypotheses await independent platform adjudication.</div> : null}
        <div className="rounded border border-blue-100 bg-blue-50 p-2"><div className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-blue-700">Latest intraday condition</div><p className="mb-2 text-[11px] text-blue-700">Latest 15-minute price-action monitor result; separate from end-of-day Market State hypotheses.</p>{intradayQ.isLoading ? <div className="text-xs text-gray-500">Loading intraday snapshots…</div> : intradayQ.isError ? <div className="text-xs text-red-700">Intraday snapshots are unavailable.</div> : intradayQ.data?.snapshots.length ? <div className="space-y-2">{intradayQ.data.snapshots.slice(0, 1).map((snapshot) => <div key={snapshot.snapshot_id} className="border-t border-blue-100 pt-1 first:border-t-0 first:pt-0"><div className="text-[10px] text-gray-500">{formatUtc(snapshot.as_of_time)} · {snapshot.market_status}</div>{snapshot.conditions.length ? snapshot.conditions.map((item) => <div key={item.key} className="mt-1 text-xs"><span className={item.tone === "positive" ? "font-medium text-green-700" : item.tone === "negative" ? "font-medium text-red-700" : "font-medium text-gray-700"}>{item.title}</span><div className="text-gray-600">{item.evidence}</div><div className="text-gray-500">{item.interpretation}</div></div>) : <div className="text-xs text-gray-500">No condition exceeded the monitor threshold.</div>}</div>)}</div> : <div className="text-xs text-gray-500">No persisted intraday snapshot yet.</div>}</div>
      </>}

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
function QuantitativeSeriesChart({ points, window, onWindowChange, loading, error }: { points: import("../types").MarketOpsQuantitativeSeriesPoint[]; window: string; onWindowChange: (value: string) => void; loading: boolean; error: boolean }) {
  const rows = points.slice().sort((a, b) => a.trade_date.localeCompare(b.trade_date));
  const putCallVolumeRatio = (value?: number) => value != null && value > 0 ? 1 / value : null;
  const sentiment = (value: number | null) => value == null ? "unavailable" : value < 1 ? "bullish · calls elevated" : value > 1 ? "bearish · puts elevated" : "neutral";
  const markers = rows.flatMap((row, index) => row.markers.map((marker) => ({ value: [index, row.eod_close ?? 0], marker, date: row.trade_date })));
  const option = { grid: { left: 48, right: 48, top: 42, bottom: 42 }, tooltip: { trigger: "axis", formatter: (items: any[]) => { const index = items?.[0]?.dataIndex ?? 0; const row = rows[index]; if (!row) return ""; const putCall = putCallVolumeRatio(row.call_put_volume_ratio); const lines = [`<b>${row.trade_date}</b>`, row.eod_close == null ? "EOD close: unavailable" : `EOD close: ${row.eod_close.toFixed(2)}${row.daily_move_pct == null ? "" : ` (${row.daily_move_pct >= 0 ? "+" : ""}${row.daily_move_pct.toFixed(2)}%)`}`, row.call_put_open_interest_ratio == null ? "Call/put OI: unavailable" : `Call/put OI: ${row.call_put_open_interest_ratio.toFixed(2)}`, putCall == null ? "Put/call volume: unavailable" : `Put/call volume: ${putCall.toFixed(2)} · ${sentiment(putCall)}`]; row.markers.forEach((m) => lines.push(`${m.algorithm_id.replace("signalops.algorithms.", "")}: ${m.severity} · ${m.adjudications?.[0]?.verdict ?? "unadjudicated"}`)); return lines.join("<br/>"); } }, legend: { data: ["EOD close", "Call/put OI", "Put/call volume sentiment"], top: 0 }, xAxis: { type: "category", data: rows.map((r) => r.trade_date) }, yAxis: [{ type: "value", name: "EOD close", scale: true }, { type: "value", name: "Ratio", scale: true }], series: [{ name: "EOD close", type: "line", yAxisIndex: 0, connectNulls: false, data: rows.map((r) => r.eod_close ?? null), itemStyle: { color: "#2563eb" }, markPoint: { symbol: "diamond", symbolSize: 12, data: markers.map((m) => ({ coord: m.value, itemStyle: { color: m.marker.adjudications?.[0]?.verdict === "confirmed" ? "#15803d" : m.marker.adjudications?.[0]?.verdict === "contradicted" ? "#dc2626" : "#d97706" }, label: { show: false } })) } }, { name: "Call/put OI", type: "line", yAxisIndex: 1, connectNulls: false, data: rows.map((r) => r.call_put_open_interest_ratio ?? null), itemStyle: { color: "#1f7a6b" } }, { name: "Put/call volume sentiment", type: "line", yAxisIndex: 1, connectNulls: false, data: rows.map((r) => putCallVolumeRatio(r.call_put_volume_ratio)), lineStyle: { color: "#6b7280" }, itemStyle: { color: (params: { value: number }) => params.value < 1 ? "#15803d" : params.value > 1 ? "#dc2626" : "#6b7280" }, markLine: { silent: true, data: [{ yAxis: 1 }], lineStyle: { type: "dashed", color: "#6b7280" }, label: { formatter: "neutral 1.0" } } }] };
  return <div className="rounded border border-violet-100 bg-white p-2"><div className="mb-1 flex items-center justify-between gap-2"><div><div className="text-[11px] font-semibold uppercase tracking-wide text-violet-700">Price, sentiment & corroboration</div><p className="text-[11px] text-gray-500">Put/call volume below 1.0 is bullish (calls elevated); above 1.0 is bearish (puts elevated). Sentiment context, not a recommendation.</p></div><select value={window} onChange={(e) => onWindowChange(e.target.value)} className="rounded border border-gray-300 px-1 py-1 text-xs">{["10_trade_days","30_trade_days","60_trade_days"].map((v) => <option key={v} value={v}>{v.replace("_trade_days"," days")}</option>)}</select></div>{loading ? <div className="text-xs text-gray-500">Loading time series…</div> : error ? <div className="text-xs text-red-700">Quantitative time series is unavailable.</div> : rows.length ? <ReactECharts option={option} style={{ height: 260 }} /> : <div className="text-xs text-gray-500">No persisted price or sentiment series is available yet.</div>}</div>;
}

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

// Aggregate analyst view. All timeline membership is delivered by one bounded
// server response; clicking a segment only changes local drill-down state.
const SIGNAL_OVERVIEW_WINDOWS = ["10_trade_days", "30_trade_days", "60_trade_days"] as const;
const SIGNAL_OVERVIEW_GROUPS = [["all_active", "All active"], ["top50_megacap", "Megacap"], ["analyst_watchlist", "Watchlist"]] as const;
const signalLabel = (value: string) => ({ bullish: "Bullish", bearish: "Bearish", neutral: "Neutral", positive: "Positive", negative: "Negative", no_active_condition: "No active condition", unavailable: "Unavailable / stale" }[value] ?? value.replace(/_/g, " "));
const signalColor = (value: string) => value === "bullish" || value === "positive" ? "#15803d" : value === "bearish" || value === "negative" ? "#dc2626" : value === "neutral" ? "#6b7280" : value === "no_active_condition" ? "#94a3b8" : "#d97706";

function SignalOverviewPanel({ tenantId, onOpenAsset }: { tenantId: string; onOpenAsset: (symbol: string) => void }) {
  const [universeGroup, setUniverseGroup] = useState("all_active");
  const [window, setWindow] = useState<import("../types").MarketOpsSignalOverviewWindow>("60_trade_days");
  const [drilldown, setDrilldown] = useState<{ title: string; members: import("../types").MarketOpsSignalOverviewMember[] } | null>(null);
  const overviewQ = useMarketOpsSignalOverview(tenantId, universeGroup, window);
  const data = overviewQ.data;
  return <div className="space-y-3">
    <div className="flex flex-wrap items-center justify-between gap-2"><div><h2 className="text-base font-semibold">Signal overview</h2><p className="text-xs text-gray-500">Persisted research breadth across the selected active universe. Evidence, not a recommendation.</p></div><div className="flex gap-2"><select aria-label="Signal overview universe" value={universeGroup} onChange={(event) => { setUniverseGroup(event.target.value); setDrilldown(null); }} className="rounded border border-gray-300 px-2 py-1 text-xs">{SIGNAL_OVERVIEW_GROUPS.map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select><select aria-label="Signal overview window" value={window} onChange={(event) => { setWindow(event.target.value as import("../types").MarketOpsSignalOverviewWindow); setDrilldown(null); }} className="rounded border border-gray-300 px-2 py-1 text-xs">{SIGNAL_OVERVIEW_WINDOWS.map((value) => <option key={value} value={value}>{value.replace("_trade_days", " days")}</option>)}</select></div></div>
    {overviewQ.isLoading && !data ? <LoadingState label="Loading signal overview..." /> : overviewQ.isError ? <ErrorState error={overviewQ.error} /> : !data ? <EmptyState message="No persisted signal overview is available for this scope." /> : <><div className="rounded border border-gray-200 bg-gray-50 p-2 text-xs text-gray-600">Active assets <strong className="text-gray-900">{data.asset_count}</strong> · generated {formatUtc(data.generated_at)} · select any chart segment to inspect its represented assets.</div><div className="grid gap-3 xl:grid-cols-2"><SignalTimelineChart title="Risk/Reward breadth" subtitle="Distinct assets by persisted EOD technical posture." points={data.risk_reward.points} onDrilldown={setDrilldown} /><SignalOverviewCoverageChart data={data} /><TechnicalScoreDistributionChart data={data} onDrilldown={setDrilldown} /><SignalTimelineChart title="Triggered-hypothesis breadth" subtitle="Distinct triggered research hypotheses by direction; one asset may appear in more than one direction." points={data.hypotheses.points} onDrilldown={setDrilldown} /><IntradayBreadthChart data={data.intraday} onDrilldown={setDrilldown} /></div>{drilldown ? <div className="rounded border border-violet-200 bg-violet-50 p-3"><div className="flex items-center justify-between gap-2"><div><div className="text-xs font-semibold text-violet-800">{drilldown.title}</div><p className="text-[11px] text-violet-700">{drilldown.members.length} represented asset{drilldown.members.length === 1 ? "" : "s"} · select a ticker to open its analysis.</p></div><button type="button" onClick={() => setDrilldown(null)} className="text-xs text-violet-700 underline">Close</button></div><div className="mt-2 grid gap-1 sm:grid-cols-2 lg:grid-cols-3">{drilldown.members.map((member, index) => <button key={`${member.ticker}-${member.label}-${index}`} type="button" onClick={() => onOpenAsset(member.ticker)} className="rounded border border-violet-100 bg-white px-2 py-1 text-left text-xs hover:border-violet-400"><span className="font-mono font-semibold text-violet-800">{member.ticker}</span><span className="ml-1 text-gray-600">{member.label}</span>{member.score != null ? <span className="ml-1 text-gray-500">· {member.score.toFixed(0)}</span> : null}</button>)}</div></div> : null}</>}
  </div>;
}

function SignalTimelineChart({ title, subtitle, points, onDrilldown }: { title: string; subtitle: string; points: import("../types").MarketOpsSignalOverviewPoint[]; onDrilldown: (value: { title: string; members: import("../types").MarketOpsSignalOverviewMember[] }) => void }) {
  const categories = ["bullish", "neutral", "bearish"];
  const option = { grid: { left: 38, right: 12, top: 42, bottom: 40 }, tooltip: { trigger: "axis" }, legend: { data: categories.map(signalLabel), top: 0 }, xAxis: { type: "category", data: points.map((point) => point.trade_date), axisLabel: { fontSize: 9 } }, yAxis: { type: "value", minInterval: 1, name: "assets", nameTextStyle: { fontSize: 9 } }, series: categories.map((key) => ({ name: signalLabel(key), type: "bar", stack: "breadth", data: points.map((point) => point.categories.find((category) => category.key === key)?.count ?? 0), itemStyle: { color: signalColor(key) } })) };
  const events = { click: (event: { seriesName?: string; dataIndex?: number }) => { const point = points[event.dataIndex ?? -1]; const key = categories.find((category) => signalLabel(category) === event.seriesName); const members = key ? point?.categories.find((category) => category.key === key)?.members ?? [] : []; if (point && key && members.length) onDrilldown({ title: `${title} · ${point.trade_date} · ${signalLabel(key)}`, members }); } };
  return <div className="rounded border border-gray-200 bg-white p-3"><div className="mb-1"><div className="text-xs font-semibold text-gray-800">{title}</div><p className="text-[11px] text-gray-500">{subtitle}</p></div>{points.length ? <ReactECharts option={option} onEvents={events} style={{ height: 260 }} /> : <div className="py-12 text-xs text-gray-500">No persisted observations in this window.</div>}</div>;
}

function IntradayBreadthChart({ data, onDrilldown }: { data: import("../types").MarketOpsSignalOverviewResponse["intraday"]; onDrilldown: (value: { title: string; members: import("../types").MarketOpsSignalOverviewMember[] }) => void }) {
  const option = { tooltip: { trigger: "item" }, legend: { bottom: 0, textStyle: { fontSize: 9 } }, series: [{ type: "pie", radius: ["36%", "68%"], label: { formatter: "{b}: {c}", fontSize: 10 }, data: data.categories.map((category) => ({ name: signalLabel(category.key), value: category.count, itemStyle: { color: signalColor(category.key) } })) }] };
  const events = { click: (event: { name?: string }) => { const category = data.categories.find((item) => signalLabel(item.key) === event.name); if (category?.members.length) onDrilldown({ title: `Current intraday breadth · ${signalLabel(category.key)}`, members: category.members }); } };
  return <div className="rounded border border-gray-200 bg-white p-3 xl:col-span-2"><div className="mb-1"><div className="text-xs font-semibold text-gray-800">Current intraday breadth</div><p className="text-[11px] text-gray-500">Latest persisted 15-minute monitor state; this is a current snapshot, not a historical trend. {data.as_of_time ? `As of ${formatUtc(data.as_of_time)}.` : ""}</p></div><ReactECharts option={option} onEvents={events} style={{ height: 250 }} /></div>;
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



function WatchlistControls({ tenantId, onChanged }: { tenantId: string; onChanged: () => void }) {
  const [ticker, setTicker] = useState("");
  const [validation, setValidation] = useState<import("../types").MarketOpsTickerValidation | null>(null);
  const [backfill, setBackfill] = useState(true);
  const [startDate, setStartDate] = useState(defaultBackfillStart());
  const [endDate, setEndDate] = useState(new Date().toISOString().slice(0, 10));
  const [message, setMessage] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function validate(event: React.FormEvent) {
    event.preventDefault(); setBusy(true); setMessage(null); setValidation(null);
    try { const result = await api.validateMarketOpsWatchlistTicker(tenantId, ticker); setValidation(result.validation); }
    catch (error) { setMessage(error instanceof Error ? error.message : "Ticker validation failed."); }
    finally { setBusy(false); }
  }

  async function onboard() {
    if (!validation) return;
    setBusy(true); setMessage(null);
    try {
      const result = await api.onboardMarketOpsWatchlistAsset(tenantId, { ticker: validation.ticker, backfill_equity_history: backfill, ...(backfill ? { start_date: startDate, end_date: endDate } : {}) });
      setMessage(result.backfill_job ? result.asset.ticker + " added; equity backfill queued." : result.asset.ticker + " added to the analyst watchlist.");
      setTicker(""); setValidation(null); onChanged();
    } catch (error) { setMessage(error instanceof Error ? error.message : "Unable to add asset."); }
    finally { setBusy(false); }
  }

  return <div className="rounded border border-brand-100 bg-brand-50 p-3">
    <form onSubmit={validate} className="flex flex-wrap items-end gap-2">
      <label className="text-xs font-medium text-gray-700">Add analyst asset<input value={ticker} onChange={(event) => setTicker(event.target.value.toUpperCase())} placeholder="Ticker" className="mt-1 block w-28 rounded border border-gray-300 px-2 py-1 font-mono text-sm" required /></label>
      <button type="submit" disabled={busy} className="rounded bg-brand-700 px-3 py-1.5 text-xs font-medium text-white disabled:opacity-50">Validate ticker</button>
    </form>
    {validation ? <div className="mt-3 space-y-2 border-t border-brand-200 pt-3 text-xs">
      <div><span className="font-semibold">{validation.ticker}</span> � {validation.company} � {validation.exchange || "exchange unavailable"}</div>
      <div className="text-gray-600">Sector: {validation.sector || "Not supplied"} � Industry: {validation.industry || "Not supplied"}</div>
      <label className="flex items-center gap-2 text-gray-700"><input type="checkbox" checked={backfill} onChange={(event) => setBackfill(event.target.checked)} /> Backfill 50 trading days of equity history</label>
      {backfill ? <div className="flex flex-wrap gap-2"><label>Start<input type="date" value={startDate} onChange={(event) => setStartDate(event.target.value)} className="ml-1 rounded border border-gray-300 px-1 py-0.5" /></label><label>End<input type="date" value={endDate} onChange={(event) => setEndDate(event.target.value)} className="ml-1 rounded border border-gray-300 px-1 py-0.5" /></label></div> : null}
      <button type="button" disabled={busy} onClick={() => void onboard()} className="rounded border border-brand-700 px-3 py-1.5 text-xs font-medium text-brand-700 disabled:opacity-50">Add validated asset</button>
      <div className="text-[11px] text-gray-600">Provider metadata is authoritative. Historical options analytics are not backfilled.</div>
    </div> : null}
    {message ? <div className="mt-2 text-xs text-brand-800">{message}</div> : null}
  </div>;
}

function defaultBackfillStart(): string {
  const cursor = new Date(); let sessions = 0;
  while (sessions < 50) { cursor.setUTCDate(cursor.getUTCDate() - 1); const day = cursor.getUTCDay(); if (day !== 0 && day !== 6) sessions++; }
  return cursor.toISOString().slice(0, 10);
}
