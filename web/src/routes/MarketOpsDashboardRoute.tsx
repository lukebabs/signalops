import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { ThemedEChart as ReactECharts } from "../components/ThemedEChart";
import { useTenant } from "../auth/session";
import { useMarketOpsIntradayConditions, useMarketOpsSignalOverview, useSyncraticInsights } from "../api/queries";
import { api } from "../api/client";
import { LoadingState, EmptyState, ErrorState } from "../components/States";
import { TechnicalScoreDistributionChart } from "../components/SignalOverviewAggregateCharts";
import { formatUtc } from "../lib/format";
import { MarketOpsWatchlistSelector, useMarketOpsWatchlistContext } from "../components/MarketOpsWatchlistContext";
import {
  classifySyncraticNarrativeQuality,
  summarizeSyncraticAsk,
  summarizeSyncraticInsight,
  SYNCRATIC_NARRATIVE_QUALITY_STYLES,
} from "../lib/syncratic";
import {
  compactNarrative,
  dashboardSyncraticNarrativeCards,
  DASHBOARD_DAILY_NARRATIVE_INSIGHT_TYPE,
  narrativeStrategy,
} from "../lib/marketopsDashboardSyncratic";
import type {
  MarketOpsSignalOverviewMember,
  MarketOpsSignalOverviewPoint,
  MarketOpsSignalOverviewResponse,
  MarketOpsSignalOverviewWindow,
  MarketOpsIntradayConditionSnapshot,
  SyncraticInsight,
} from "../types";

const WINDOWS: MarketOpsSignalOverviewWindow[] = [
  "10_trade_days",
  "30_trade_days",
  "60_trade_days",
];
const label = (value: string) =>
  ({
    bullish: "Bullish",
    bearish: "Bearish",
    neutral: "Neutral",
    insufficient_inputs: "Insufficient inputs",
    unprocessed: "Not materialized",
    positive: "Positive",
    negative: "Negative",
    no_active_condition: "No active condition",
    unavailable: "Unavailable / stale",
  })[value] ?? value.replace(/_/g, " ");
const color = (value: string) =>
  value === "bullish" || value === "positive"
    ? "#15803d"
    : value === "bearish" || value === "negative"
      ? "#dc2626"
      : value === "neutral"
        ? "#6b7280"
        : value === "insufficient_inputs"
          ? "#d97706"
          : value === "unprocessed" || value === "no_active_condition"
            ? "#94a3b8"
            : "#d97706";

export function MarketOpsDashboardRoute() {
  const tenantId = useTenant();
  const navigate = useNavigate();
  const watchlist = useMarketOpsWatchlistContext();
  const [window, setWindow] =
    useState<MarketOpsSignalOverviewWindow>("10_trade_days");
  const [drilldown, setDrilldown] = useState<{
    title: string;
    members: MarketOpsSignalOverviewMember[];
  } | null>(null);
  const [expandedNarrativeId, setExpandedNarrativeId] = useState<string | null>(null);
  const query = useMarketOpsSignalOverview(tenantId, "all_active", window);
  const reversalQ = useQuery({
    queryKey: ["marketops-eroc", tenantId],
    queryFn: () => api.getMarketOpsEROC(tenantId),
    refetchInterval: 5 * 60 * 1000,
  });
  const eventsQ = useQuery({
    queryKey: ["marketops-material-events", tenantId],
    queryFn: () => api.getMarketOpsMaterialEvents(tenantId),
    refetchInterval: 5 * 60 * 1000,
  });
  const narrativesQ = useSyncraticInsights({
    tenant_id: tenantId,
    subject_symbol: "MARKETOPS",
    insight_type: DASHBOARD_DAILY_NARRATIVE_INSIGHT_TYPE,
    status: "active",
    limit: 20,
  });
  const intradayConditionsQ = useMarketOpsIntradayConditions(tenantId, "all_active");
  const data = query.data;
  const openAsset = (symbol: string) =>
    void navigate({ to: "/marketops/state", search: { symbol } });
  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h1 className="text-lg font-semibold">MarketOps Dashboard</h1>
          <p className="text-xs text-gray-500">
            Persisted research breadth across the selected watchlist. Evidence, not a recommendation.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <MarketOpsWatchlistSelector />
          <select
            aria-label="Signal overview window"
            value={window}
            onChange={(event) => {
              setWindow(event.target.value as MarketOpsSignalOverviewWindow);
              setDrilldown(null);
            }}
            className="rounded border border-gray-300 px-2 py-1 text-xs"
          >
            {WINDOWS.map((value) => (
              <option key={value} value={value}>
                {value.replace("_trade_days", " days")}
              </option>
            ))}
          </select>
        </div>
      </div>
      {query.isLoading && !data ? (
        <LoadingState label="Loading MarketOps dashboard..." />
      ) : query.isError ? (
        <ErrorState error={query.error} />
      ) : !data ? (
        <EmptyState message="No persisted signal overview is available for this scope." />
      ) : (
        <div className="grid gap-3 xl:grid-cols-[minmax(0,4fr)_minmax(260px,1fr)]">
          <main data-testid="dashboard-primary-content" className="min-w-0 space-y-3">
            <div className="rounded border border-gray-200 bg-gray-50 p-2 text-xs text-gray-600 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-300">
              Watchlist assets {" "}
              <strong className="text-gray-900 dark:text-gray-100">{data.asset_count}</strong>{watchlist.context?.list_name ? <> · {watchlist.context.list_name}</> : null} ·
              generated {formatUtc(data.generated_at)} · select any chart segment
              to inspect its represented assets.
            </div>
            <div className="grid gap-3 xl:grid-cols-2">
              <div className="order-1 min-w-0">
                <Timeline
                  title="Risk/Reward breadth"
                  subtitle="Distinct assets by persisted EOD technical posture."
                  points={data.risk_reward.points}
                  onDrilldown={setDrilldown}
                />
                {drilldown?.title.startsWith("Risk/Reward breadth") ? (
                  <AssetDrilldown
                    drilldown={drilldown}
                    onClose={() => setDrilldown(null)}
                    openAsset={openAsset}
                  />
                ) : null}
              </div>
              <div className="order-3 min-w-0">
                <TechnicalScoreDistributionChart
                  data={data}
                  onDrilldown={setDrilldown}
                />
                {drilldown?.title.startsWith("Technical score distribution") ? (
                  <AssetDrilldown
                    drilldown={drilldown}
                    onClose={() => setDrilldown(null)}
                    openAsset={openAsset}
                  />
                ) : null}
              </div>
              <div className="order-6 min-w-0 xl:col-span-2">
                <ExhaustiveReversalQueue
                  rows={reversalQ.data?.results ?? []}
                  loading={reversalQ.isLoading}
                />
              </div>
              <div className="order-2 min-w-0">
                <OptionsFlowExtremes
                  data={data.options_flow_extremes}
                  onDrilldown={setDrilldown}
                />
                {drilldown?.title.startsWith("Options-flow extremes") ? (
                  <AssetDrilldown
                    drilldown={drilldown}
                    onClose={() => setDrilldown(null)}
                    openAsset={openAsset}
                  />
                ) : null}
              </div>
              <div className="order-4 min-w-0">
                <Intraday data={data.intraday} onDrilldown={setDrilldown} />
                {drilldown?.title.startsWith("Current intraday breadth") ? (
                  <AssetDrilldown
                    drilldown={drilldown}
                    onClose={() => setDrilldown(null)}
                    openAsset={openAsset}
                  />
                ) : null}
              </div>
              <div className="order-5 min-w-0 xl:col-span-2">
                <UpcomingEarnings
                  events={eventsQ.data?.events ?? []}
                  loading={eventsQ.isLoading}
                  onOpen={openAsset}
                />
              </div>
            </div>
            <DashboardSyncraticNarratives
              narratives={narrativesQ.data?.syncratic_insights ?? []}
              loading={narrativesQ.isLoading}
              error={narrativesQ.isError ? narrativesQ.error : null}
              expandedNarrativeId={expandedNarrativeId}
              onToggleNarrative={(id) => setExpandedNarrativeId(expandedNarrativeId === id ? null : id)}
              openSyncratic={(insight) => void navigate({ to: "/marketops/syncratic", search: insight ? { tab: syncraticTabForInsight(insight), insight_id: insight.syncratic_insight_id } : {} })}
            />
          </main>
          <MarketIntelligenceReel
            snapshots={intradayConditionsQ.data?.snapshots ?? []}
            loading={intradayConditionsQ.isLoading}
            error={intradayConditionsQ.isError ? intradayConditionsQ.error : null}
            watchlist={watchlist}
            openAsset={openAsset}
            openReel={() => void navigate({ to: "/marketops/indicator-reel" })}
          />
        </div>
      )}
    </div>
  );
}


function DashboardSyncraticNarratives({
  narratives,
  loading,
  error,
  expandedNarrativeId,
  onToggleNarrative,
  openSyncratic,
}: {
  narratives: SyncraticInsight[];
  loading: boolean;
  error: unknown;
  expandedNarrativeId: string | null;
  onToggleNarrative: (insightId: string) => void;
  openSyncratic: (insight?: SyncraticInsight) => void;
}) {
  const cards = dashboardSyncraticNarrativeCards(narratives);
  const expandedCard = cards.find((card) => card.insight.syncratic_insight_id === expandedNarrativeId) ?? null;

  return (
    <section className="rounded border border-gray-200 bg-white p-3 dark:border-gray-700 dark:bg-gray-900">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div>
          <div className="text-sm font-semibold text-gray-900 dark:text-gray-100">
            Syncratic narrative digest
          </div>
          <p className="text-[11px] leading-5 text-gray-500 dark:text-gray-400">
            Tenant/global MarketOps context. Select a card for a short summary snippet; open Syncratic Intelligence for the full narrative and evidence workspace.
          </p>
        </div>
        <button
          type="button"
          onClick={() => openSyncratic()}
          className="rounded bg-brand-600 px-2.5 py-1 text-xs font-medium text-white hover:bg-brand-700"
        >
          Open Syncratic Intelligence
        </button>
      </div>
      {loading ? (
        <div className="py-4 text-xs text-gray-500 dark:text-gray-400">
          Loading Syncratic narratives…
        </div>
      ) : error ? (
        <div className="mt-2 rounded border border-amber-200 bg-amber-50 px-2 py-1 text-xs text-amber-800 dark:border-amber-800 dark:bg-amber-950/50 dark:text-amber-200">
          Syncratic narratives are currently unavailable. Dashboard market breadth remains available.
        </div>
      ) : cards.length ? (
        <div className="mt-3 grid gap-2 md:grid-cols-2 xl:grid-cols-4">
          {cards.map(({ label: cardLabel, insight }) => {
            const summary = summarizeSyncraticInsight(insight);
            const quality = classifySyncraticNarrativeQuality(insight);
            const text = compactNarrative(summary.explanation || summary.summary);
            const ask = summarizeSyncraticAsk(insight);
            return (
              <button
                key={summary.insightId}
                type="button"
                data-testid={`dashboard-syncratic-card-${insightStrategyTestId(insight)}`}
                onClick={() => onToggleNarrative(summary.insightId)}
                className={`rounded border p-2 text-left hover:border-brand-300 hover:bg-brand-50 dark:hover:border-brand-500 dark:hover:bg-brand-950/30 ${expandedNarrativeId === summary.insightId ? "border-brand-300 bg-brand-50 dark:border-brand-500 dark:bg-brand-950/30" : "border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-950"}` }
              >
                <div className="flex flex-wrap items-center gap-1.5">
                  <span className="rounded bg-brand-100 px-1.5 py-0.5 text-[11px] font-medium text-brand-800 dark:bg-brand-950 dark:text-brand-200">
                    {cardLabel}
                  </span>
                  <span className={`rounded border px-1.5 py-0.5 text-[11px] font-medium ${SYNCRATIC_NARRATIVE_QUALITY_STYLES[quality.quality]}`}>
                    {quality.label}
                  </span>
                </div>
                <div className="mt-1 line-clamp-2 text-xs font-semibold text-gray-900 dark:text-gray-100">
                  {summary.title || cardLabel}
                </div>
                <p className="mt-1 line-clamp-4 text-xs leading-5 text-gray-600 dark:text-gray-300">
                  {text || "Narrative exists, but no compact summary text is available."}
                </p>
                <div className="mt-2 flex items-center justify-between gap-2 text-[11px] text-gray-500 dark:text-gray-400">
                  <span>Updated {formatUtc(summary.updatedAt)}</span>
                  <span className="font-medium text-brand-700 dark:text-brand-300">{expandedCard?.insight.syncratic_insight_id === summary.insightId ? "Showing snippet" : "View snippet"}</span>
                </div>
              </button>
            );
          })}
        </div>
      ) : (
        <div className="mt-2 rounded border border-gray-200 bg-gray-50 px-2 py-3 text-xs text-gray-600 dark:border-gray-700 dark:bg-gray-950 dark:text-gray-300">
          No Syncratic daily narratives are available yet. Open Syncratic Intelligence to inspect or materialize persisted context.
        </div>
      )}

      {expandedCard ? (
        <DashboardSyncraticFullNarrative
          label={expandedCard.label}
          insight={expandedCard.insight}
          openSyncratic={openSyncratic}
        />
      ) : null}
    </section>
  );
}

function insightStrategyTestId(insight: SyncraticInsight): string {
  return narrativeStrategy(insight).replace(/[^a-z0-9]+/gi, "-").replace(/^-|-$/g, "").toLowerCase() || "unknown";
}

function syncraticTabForInsight(insight: SyncraticInsight): string {
  const strategy = narrativeStrategy(insight);
  if (strategy === "marketops_sri_daily_v1") return "sri";
  if (strategy === "marketops_risk_reward_daily_v1") return "risk_reward";
  if (strategy === "marketops_review_queue_daily_v1") return "review_queue";
  return "daily";
}

function DashboardSyncraticFullNarrative({
  label,
  insight,
  openSyncratic,
}: {
  label: string;
  insight: SyncraticInsight;
  openSyncratic: (insight?: SyncraticInsight) => void;
}) {
  const summary = summarizeSyncraticInsight(insight);
  const quality = classifySyncraticNarrativeQuality(insight);
  const snippet = snippetText(summary.summary || summary.explanation, 420);
  return (
    <div data-testid="dashboard-syncratic-explainability" className="mt-3 rounded border border-brand-200 bg-brand-50/40 p-3 dark:border-brand-800 dark:bg-brand-950/20">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div>
          <div className="flex flex-wrap items-center gap-1.5">
            <span className="rounded bg-brand-100 px-1.5 py-0.5 text-[11px] font-medium text-brand-800 dark:bg-brand-950 dark:text-brand-200">{label}</span>
            <span className={`rounded border px-1.5 py-0.5 text-[11px] font-medium ${SYNCRATIC_NARRATIVE_QUALITY_STYLES[quality.quality]}`}>{quality.label}</span>
            <span className="text-[11px] text-gray-500 dark:text-gray-400">Updated {formatUtc(summary.updatedAt)}</span>
          </div>
          <div className="mt-1 text-sm font-semibold text-gray-900 dark:text-gray-100">{summary.title || label}</div>
        </div>
        <button type="button" onClick={() => openSyncratic(insight)} className="rounded bg-brand-600 px-2.5 py-1 text-xs font-medium text-white hover:bg-brand-700">Open Syncratic Intelligence</button>
      </div>
      <div className="mt-3">
        <div className="mb-1 text-xs font-semibold text-gray-700 dark:text-gray-200">Summary snippet</div>
        <p className="rounded border border-brand-200 bg-white p-3 text-sm leading-6 text-gray-800 dark:border-brand-800 dark:bg-gray-950 dark:text-gray-100">
          {snippet || "Narrative exists, but no summary snippet is available."}
        </p>
        <div className="mt-1 text-[11px] text-gray-500 dark:text-gray-400">
          Dashboard intentionally shows only a bounded summary. Use Syncratic Intelligence for the full narrative, provenance, and evidence workspace.
        </div>
      </div>
    </div>
  );
}


function snippetText(value: string, maxLength: number): string {
  const compact = compactNarrative(value || "");
  if (compact.length <= maxLength) return compact;
  return compact.slice(0, maxLength).replace(/\s+\S*$/, "") + "…";
}

function MarketIntelligenceReel({
  snapshots,
  loading,
  error,
  watchlist,
  openAsset,
  openReel,
}: {
  snapshots: MarketOpsIntradayConditionSnapshot[];
  loading: boolean;
  error: unknown;
  watchlist: ReturnType<typeof useMarketOpsWatchlistContext>;
  openAsset: (symbol: string) => void;
  openReel: () => void;
}) {
  const items = snapshots
    .filter((snapshot) => !watchlist.available || watchlist.tickerSet.has(snapshot.ticker.toUpperCase()))
    .flatMap((snapshot) =>
      snapshot.conditions.map((condition) => ({ snapshot, condition })),
    )
    .sort(
      (left, right) =>
        Number(right.condition.score ?? 0) - Number(left.condition.score ?? 0) ||
        String(right.snapshot.as_of_time).localeCompare(String(left.snapshot.as_of_time)) ||
        left.snapshot.ticker.localeCompare(right.snapshot.ticker),
    )
    .slice(0, 10);
  return (
    <aside className="min-w-0 space-y-3 xl:sticky xl:top-3 xl:self-start" data-testid="dashboard-market-intelligence-reel">
      <section className="rounded border border-gray-200 bg-white p-3 dark:border-gray-700 dark:bg-gray-900">
        <div className="flex items-start justify-between gap-2">
          <div>
            <div className="text-sm font-semibold text-gray-900 dark:text-gray-100">Market Intelligence</div>
            <p className="mt-1 text-[11px] leading-5 text-gray-500 dark:text-gray-400">Dynamic reel of current intraday conditions for the selected watchlist.</p>
          </div>
          <button type="button" onClick={openReel} className="shrink-0 rounded bg-brand-600 px-2 py-1 text-[11px] font-medium text-white hover:bg-brand-700">Open</button>
        </div>
        {loading ? (
          <div className="py-4 text-xs text-gray-500 dark:text-gray-400">Loading reel…</div>
        ) : error ? (
          <div className="mt-2 rounded border border-amber-200 bg-amber-50 px-2 py-1 text-xs text-amber-800 dark:border-amber-800 dark:bg-amber-950/50 dark:text-amber-200">Market Intelligence is currently unavailable.</div>
        ) : items.length ? (
          <div className="mt-3 max-h-[760px] space-y-2 overflow-y-auto pr-1">
            {items.map(({ snapshot, condition }) => (
              <button
                key={`${snapshot.snapshot_id}-${condition.key}`}
                type="button"
                onClick={() => openAsset(snapshot.ticker)}
                className={`w-full rounded border p-2 text-left transition-colors hover:border-brand-300 ${condition.tone === "positive" ? "border-green-200 bg-green-50 dark:border-green-900 dark:bg-green-950/30" : condition.tone === "negative" ? "border-red-200 bg-red-50 dark:border-red-900 dark:bg-red-950/30" : "border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-950"}`}
              >
                <div className="flex items-start justify-between gap-2">
                  <span className="font-mono text-xs font-semibold text-gray-900 dark:text-gray-100">{snapshot.ticker}</span>
                  <span className="text-[10px] text-gray-500 dark:text-gray-400">{formatUtc(snapshot.as_of_time)}</span>
                </div>
                <div className="mt-1 text-xs font-semibold text-gray-800 dark:text-gray-100">{condition.title}</div>
                <p className="mt-1 line-clamp-3 text-[11px] leading-5 text-gray-600 dark:text-gray-300">{condition.evidence || condition.interpretation}</p>
                <div className="mt-1 flex flex-wrap items-center gap-1 text-[10px] text-gray-500 dark:text-gray-400">
                  <span>{snapshot.market_status.replace(/_/g, " ")}</span>
                  <span>· score {Number(condition.score ?? 0).toFixed(1)}</span>
                  {snapshot.stale ? <span className="text-amber-700 dark:text-amber-300">· stale</span> : null}
                </div>
              </button>
            ))}
          </div>
        ) : (
          <div className="mt-3 rounded border border-gray-200 bg-gray-50 px-2 py-3 text-xs text-gray-600 dark:border-gray-700 dark:bg-gray-950 dark:text-gray-300">No current intraday condition exceeds the reel threshold for this watchlist.</div>
        )}
      </section>
    </aside>
  );
}

function UpcomingEarnings({
  events,
  loading,
  onOpen,
}: {
  events: any[];
  loading: boolean;
  onOpen: (symbol: string) => void;
}) {
  const upcoming = events
    .filter((event) => Number(event.days_to_event) >= 0)
    .slice(0, 8);
  return (
    <section className="rounded border border-gray-200 bg-white p-3">
      <div className="flex items-center justify-between gap-2">
        <div>
          <div className="text-xs font-semibold text-gray-800">
            Upcoming earnings
          </div>
          <p className="text-[11px] text-gray-500">
            Point-in-time-known dates that can materially change price, options,
            and volume conditions. Context only; FMP does not provide event
            timing.
          </p>
        </div>
        <a
          href="/marketops/earnings"
          className="text-xs text-brand-700 hover:underline"
        >
          Open Earnings Opportunity Intelligence
        </a>
      </div>
      {loading ? (
        <div className="py-4 text-xs text-gray-500">
          Loading earnings calendar…
        </div>
      ) : upcoming.length ? (
        <div className="mt-2 grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
          {upcoming.map((event) => (
            <button
              key={event.event_id}
              type="button"
              onClick={() => onOpen(event.symbol)}
              className="rounded border border-gray-200 bg-gray-50 p-2 text-left hover:border-brand-300 hover:bg-brand-50"
            >
              <div className="flex items-center justify-between gap-2">
                <span className="font-mono text-xs font-semibold text-gray-900">
                  {event.symbol}
                </span>
                <span className="text-xs font-semibold text-brand-700">
                  {event.days_to_event === 0
                    ? "Today"
                    : String(event.days_to_event) + "d"}
                </span>
              </div>
              <div className="mt-1 text-xs text-gray-700">
                {event.event_date}
              </div>
              <div className="mt-1 text-[11px] text-gray-500">
                Earnings · {event.status?.replace(/_/g, " ")}
              </div>
            </button>
          ))}
        </div>
      ) : (
        <div className="py-4 text-xs text-gray-500">
          No point-in-time-known earnings events are currently persisted.
        </div>
      )}
    </section>
  );
}

function AssetDrilldown({
  drilldown,
  onClose,
  openAsset,
}: {
  drilldown: { title: string; members: MarketOpsSignalOverviewMember[] };
  onClose: () => void;
  openAsset: (symbol: string) => void;
}) {
  return (
    <div className="mt-3 rounded border border-violet-200 bg-violet-50 p-3">
      <div className="flex items-center justify-between gap-2">
        <div>
          <div className="text-xs font-semibold text-violet-800">
            {drilldown.title}
          </div>
          <p className="text-[11px] text-violet-700">
            {drilldown.members.length} represented assets · select a ticker to
            open Market State.
          </p>
        </div>
        <button
          type="button"
          onClick={onClose}
          className="text-xs text-violet-700 underline"
        >
          Close
        </button>
      </div>
      <div className="mt-2 grid gap-1 sm:grid-cols-2">
        {drilldown.members.map((member, index) => (
          <button
            key={member.ticker + "-" + index}
            type="button"
            onClick={() => openAsset(member.ticker)}
            className="rounded border border-violet-100 bg-white px-2 py-1 text-left text-xs hover:border-violet-400"
          >
            <span className="font-mono font-semibold text-violet-800">
              {member.ticker}
            </span>
            <span className="ml-1 text-gray-600">{member.label}</span>
            {member.score != null ? (
              <span className="ml-1 text-gray-500">
                · {member.score.toFixed(0)}
              </span>
            ) : null}
          </button>
        ))}
      </div>
    </div>
  );
}

function Timeline({
  title,
  subtitle,
  points,
  onDrilldown,
}: {
  title: string;
  subtitle: string;
  points: MarketOpsSignalOverviewPoint[];
  onDrilldown: (value: {
    title: string;
    members: MarketOpsSignalOverviewMember[];
  }) => void;
}) {
  const directional = ["bullish", "neutral", "bearish"];
  const categories =
    title === "Risk/Reward breadth"
      ? [...directional, "insufficient_inputs", "unprocessed"]
      : directional;
  const value = (point: MarketOpsSignalOverviewPoint, key: string) =>
    key === "insufficient_inputs"
      ? (point.coverage?.insufficient_inputs ?? 0)
      : key === "unprocessed"
        ? (point.coverage?.unprocessed ?? 0)
        : (point.categories.find((category) => category.key === key)?.count ??
          0);
  const option = {
    grid: { left: 38, right: 12, top: 42, bottom: 40 },
    tooltip: { trigger: "axis" },
    legend: { data: categories.map(label), top: 0 },
    xAxis: {
      type: "category",
      data: points.map((point) => point.trade_date),
      axisLabel: { fontSize: 9 },
    },
    yAxis: {
      type: "value",
      minInterval: 1,
      name: "assets",
      nameTextStyle: { fontSize: 9 },
    },
    series: categories.map((key) => ({
      name: label(key),
      type: "bar",
      stack: "breadth",
      data: points.map((point) => value(point, key)),
      itemStyle: { color: color(key) },
    })),
  };
  return (
    <section className="rounded border border-gray-200 bg-white p-3">
      <div className="mb-1">
        <div className="text-xs font-semibold text-gray-800">{title}</div>
        <p className="text-[11px] text-gray-500">{subtitle}</p>
      </div>
      {points.length ? (
        <ReactECharts
          option={option}
          onEvents={{
            click: (event: { seriesName?: string; dataIndex?: number }) => {
              const key = categories.find(
                (item) => label(item) === event.seriesName,
              );
              const point = points[event.dataIndex ?? -1];
              const members = key
                ? (point?.categories.find((category) => category.key === key)
                    ?.members ?? [])
                : [];
              if (key && point && members.length)
                onDrilldown({
                  title: `${title} · ${point.trade_date} · ${label(key)}`,
                  members,
                });
            },
          }}
          style={{ height: 260 }}
        />
      ) : (
        <div className="py-12 text-xs text-gray-500">
          No persisted observations in this window.
        </div>
      )}
    </section>
  );
}

function OptionsFlowExtremes({
  data,
  onDrilldown,
}: {
  data: MarketOpsSignalOverviewResponse["options_flow_extremes"];
  onDrilldown: (value: {
    title: string;
    members: MarketOpsSignalOverviewMember[];
  }) => void;
}) {
  const [selected, setSelected] = useState<"call" | "put" | null>(null);
  const categories = data.categories;
  const call = categories.find((item) => item.key === "call_volume_extreme");
  const put = categories.find((item) => item.key === "put_volume_extreme");
  return (
    <section className="rounded border border-gray-200 bg-white p-3">
      <div className="mb-2">
        <div className="text-xs font-semibold text-gray-800">
          Options-flow extremes
        </div>
        <p className="text-[11px] text-gray-500">
          Latest completed EOD · put/call volume &lt;0.30 or &gt;1.20, with at
          least 1,000 contracts. Aggregate flow is context, not a directional
          recommendation.
        </p>
      </div>
      <div className="grid grid-cols-2 gap-2">
        <button
          type="button"
          onClick={() => {
            setSelected("call");
            if (call)
              onDrilldown({
                title: "Options-flow extremes · Call-volume extreme",
                members: call.members,
              });
          }}
          aria-pressed={selected === "call"}
          className={
            selected === "call"
              ? "cursor-pointer rounded border border-green-600 bg-green-100 p-2 text-left shadow-sm ring-1 ring-green-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-green-500 focus-visible:ring-offset-1"
              : "cursor-pointer rounded border border-gray-200 bg-white p-2 text-left transition-all duration-150 hover:-translate-y-0.5 hover:border-green-400 hover:bg-green-100 hover:shadow-md hover:ring-1 hover:ring-green-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-green-500 focus-visible:ring-offset-1"
          }
        >
          <div className="text-[11px] font-medium text-green-800">
            Call-volume extreme
          </div>
          <div className="text-lg font-semibold text-green-800">
            {call?.count ?? 0}
          </div>
          <div className="text-[10px] text-green-700">put/call &lt; 0.30</div>
        </button>
        <button
          type="button"
          onClick={() => {
            setSelected("put");
            if (put)
              onDrilldown({
                title: "Options-flow extremes · Put-volume extreme",
                members: put.members,
              });
          }}
          aria-pressed={selected === "put"}
          className={
            selected === "put"
              ? "cursor-pointer rounded border border-red-600 bg-red-100 p-2 text-left shadow-sm ring-1 ring-red-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500 focus-visible:ring-offset-1"
              : "cursor-pointer rounded border border-gray-200 bg-white p-2 text-left transition-all duration-150 hover:-translate-y-0.5 hover:border-red-400 hover:bg-red-100 hover:shadow-md hover:ring-1 hover:ring-red-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500 focus-visible:ring-offset-1"
          }
        >
          <div className="text-[11px] font-medium text-red-800">
            Put-volume extreme
          </div>
          <div className="text-lg font-semibold text-red-800">
            {put?.count ?? 0}
          </div>
          <div className="text-[10px] text-red-700">put/call &gt; 1.20</div>
        </button>
      </div>
      <div className="mt-2 text-[10px] text-gray-500">
        As of {data.as_of || "unavailable"} · {data.coverage.eligible} eligible
        · {data.coverage.insufficient_activity} below activity threshold ·{" "}
        {data.coverage.missing_or_stale} missing or stale
      </div>
    </section>
  );
}

function Intraday({
  data,
  onDrilldown,
}: {
  data: MarketOpsSignalOverviewResponse["intraday"];
  onDrilldown: (value: {
    title: string;
    members: MarketOpsSignalOverviewMember[];
  }) => void;
}) {
  const option = {
    tooltip: { trigger: "item" },
    legend: { bottom: 0, textStyle: { fontSize: 9 } },
    series: [
      {
        type: "pie",
        radius: ["36%", "68%"],
        label: { formatter: "{b}: {c}", fontSize: 10 },
        data: data.categories.map((category) => ({
          name: label(category.key),
          value: category.count,
          itemStyle: { color: color(category.key) },
        })),
      },
    ],
  };
  return (
    <section className="rounded border border-gray-200 bg-white p-3 xl:col-span-2">
      <div className="mb-1">
        <div className="text-xs font-semibold text-gray-800">
          Current intraday breadth
        </div>
        <p className="text-[11px] text-gray-500">
          Latest persisted 15-minute monitor state, bounded to the last
          completed EOD record.{" "}
          {data.as_of_time ? `As of ${formatUtc(data.as_of_time)}.` : ""}
        </p>
      </div>
      <ReactECharts
        option={option}
        onEvents={{
          click: (event: { name?: string }) => {
            const category = data.categories.find(
              (item) => label(item.key) === event.name,
            );
            if (category?.members.length)
              onDrilldown({
                title: `Current intraday breadth · ${label(category.key)}`,
                members: category.members,
              });
          },
        }}
        style={{ height: 250 }}
      />
    </section>
  );
}
function ExhaustiveReversalQueue({
  rows,
  loading,
}: {
  rows: any[];
  loading: boolean;
}) {
  const candidates = rows
    .filter(
      (row) =>
        row.trace?.reversal_candidate ||
        row.trace?.regime === "trend_supported",
    )
    .sort(
      (a, b) =>
        Number(Boolean(b.trace?.reversal_candidate)) -
          Number(Boolean(a.trace?.reversal_candidate)) ||
        Math.abs(Number(b.trace?.stance_score ?? b.score ?? 0)) -
          Math.abs(Number(a.trace?.stance_score ?? a.score ?? 0)),
    )
    .slice(0, 8);
  return (
    <section className="rounded border border-gray-200 bg-white p-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <div className="text-xs font-semibold text-gray-800">
            Reversal Review Queue
          </div>
          <p className="text-[11px] text-gray-500">
            Latest completed EOD assessment · fading or climactic extensions
            requiring analyst review.
          </p>
        </div>
        <a
          href="/marketops/eroc"
          className="text-xs text-brand-700 hover:underline"
        >
          Open full view
        </a>
      </div>
      {loading ? (
        <div className="py-6 text-xs text-gray-500">
          Loading current assessment…
        </div>
      ) : candidates.length ? (
        <div className="mt-2 grid gap-2 md:grid-cols-2 xl:grid-cols-4">
          {candidates.map((row) => {
            const trace = row.trace ?? {},
              stance = Number(trace.stance_score ?? row.score ?? 0),
              trend = trace.regime === "trend_supported",
              direction = trend
                ? trace.direction === "BULLISH"
                  ? "Bullish drift"
                  : trace.direction === "BEARISH"
                    ? "Bearish drift"
                    : "Trend context"
                : trace.direction === "BULLISH"
                  ? "Bullish reversal"
                  : trace.direction === "BEARISH"
                    ? "Bearish reversal"
                    : "Review";
            return (
              <a
                key={row.ticker}
                href="/marketops/eroc"
                className="rounded border border-gray-200 bg-gray-50 p-2 hover:border-brand-300 hover:bg-brand-50"
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="font-mono text-xs font-semibold text-gray-900">
                    {row.ticker}
                  </span>
                  <span
                    className={
                      stance >= 0
                        ? "text-xs font-semibold text-green-700"
                        : "text-xs font-semibold text-red-700"
                    }
                  >
                    {trend ? (
                      ""
                    ) : (
                      <>
                        {stance > 0 ? "+" : ""}
                        {(stance / 10).toFixed(1)}
                      </>
                    )}
                  </span>
                </div>
                <div className="mt-1 text-xs text-gray-700">{direction}</div>
                <div className="mt-1 text-[11px] text-gray-500">
                  {trend
                    ? "Trend-supported · sustained participation"
                    : trace.regime === "climactic_extension"
                      ? "Climactic extension"
                      : "Fading drift"}{" "}
                  ·{" "}
                  {trace.evidence_complete
                    ? "complete evidence"
                    : "incomplete evidence"}
                </div>
              </a>
            );
          })}
        </div>
      ) : (
        <div className="py-6 text-xs text-gray-500">
          No current reversal or trend-supported review rows.
        </div>
      )}
    </section>
  );
}
