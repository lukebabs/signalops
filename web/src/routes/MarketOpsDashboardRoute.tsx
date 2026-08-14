import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { ThemedEChart as ReactECharts } from "../components/ThemedEChart";
import { useTenant } from "../auth/session";
import { useMarketOpsSignalOverview } from "../api/queries";
import { api } from "../api/client";
import { LoadingState, EmptyState, ErrorState } from "../components/States";
import { TechnicalScoreDistributionChart } from "../components/SignalOverviewAggregateCharts";
import { formatUtc } from "../lib/format";
import { MarketOpsWatchlistSelector, useMarketOpsWatchlistContext } from "../components/MarketOpsWatchlistContext";
import type {
  MarketOpsSignalOverviewMember,
  MarketOpsSignalOverviewPoint,
  MarketOpsSignalOverviewResponse,
  MarketOpsSignalOverviewWindow,
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
        <>
          <div className="rounded border border-gray-200 bg-gray-50 p-2 text-xs text-gray-600">
            Watchlist assets {" "}
            <strong className="text-gray-900">{data.asset_count}</strong>{watchlist.context?.list_name ? <> · {watchlist.context.list_name}</> : null} ·
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
        </>
      )}
    </div>
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
          Open EEOM
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
