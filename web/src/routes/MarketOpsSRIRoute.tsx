import { Fragment, useEffect, useMemo, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "../api/client";
import { useTenant } from "../auth/session";
import { ThemedEChart } from "../components/ThemedEChart";
import { EmptyState, ErrorState, LoadingState } from "../components/States";
import { RefreshButton } from "../components/RefreshButton";
import { formatPercent } from "../lib/format";
import type { MarketOpsSRIETFMakeupResponse, MarketOpsSRISnapshot } from "../types";

type Tab = "rankings" | "progression";
type MetricKey = "composite_score" | "relative_strength_score" | "momentum_score" | "momentum_acceleration";
type IndicatorTone = { accent: string; badge: string; dot: string; text: string; chart: string };

const metrics: Array<{ key: MetricKey; label: string }> = [
  { key: "composite_score", label: "Composite score" },
  { key: "relative_strength_score", label: "Relative strength" },
  { key: "momentum_score", label: "Momentum" },
  { key: "momentum_acceleration", label: "Acceleration" },
];

const title = (value: string) => value.replace(/^sri_(sector|industry)_/, "").replace(/_/g, " ").toLowerCase().replace(/\b\w/g, (letter) => letter.toUpperCase());
const score = (value?: number) => value == null ? "—" : value.toFixed(1);
const primaryETF = (item: MarketOpsSRISnapshot) => {
  if (item.primary_etf) return item.primary_etf;
  if (item.input_provenance && typeof item.input_provenance === "object" && "primary_etf" in item.input_provenance) {
    const value = (item.input_provenance as Record<string, unknown>).primary_etf;
    return typeof value === "string" ? value : "ETF";
  }
  return "ETF";
};

const stateTone = (state: string): IndicatorTone => {
  switch (state) {
    case "LEADING": return { accent: "border-l-emerald-500", badge: "border-emerald-200 bg-emerald-50 text-emerald-800", dot: "bg-emerald-500", text: "text-emerald-700", chart: "#059669" };
    case "IMPROVING": return { accent: "border-l-sky-500", badge: "border-sky-200 bg-sky-50 text-sky-800", dot: "bg-sky-500", text: "text-sky-700", chart: "#0284c7" };
    case "WEAKENING": return { accent: "border-l-amber-500", badge: "border-amber-200 bg-amber-50 text-amber-800", dot: "bg-amber-500", text: "text-amber-700", chart: "#d97706" };
    case "LAGGING": return { accent: "border-l-rose-500", badge: "border-rose-200 bg-rose-50 text-rose-800", dot: "bg-rose-500", text: "text-rose-700", chart: "#e11d48" };
    default: return { accent: "border-l-slate-400", badge: "border-slate-200 bg-slate-50 text-slate-700", dot: "bg-slate-400", text: "text-slate-700", chart: "#64748b" };
  }
};

const scoreTone = (value?: number) => {
  if (value == null) return "text-slate-500";
  if (value >= 75) return "text-emerald-700";
  if (value >= 60) return "text-sky-700";
  if (value >= 40) return "text-slate-700";
  if (value >= 25) return "text-amber-700";
  return "text-rose-700";
};

const qualityTone = (quality: string) => quality === "usable"
  ? "border-emerald-200 bg-emerald-50 text-emerald-800"
  : quality === "partial"
    ? "border-amber-200 bg-amber-50 text-amber-800"
    : "border-rose-200 bg-rose-50 text-rose-800";

function Metric({ label, value }: { label: string; value?: number }) {
  return <div><dt className="text-gray-500">{label}</dt><dd className={"font-semibold " + scoreTone(value)}>{score(value)}</dd></div>;
}

function StateBadge({ state }: { state: string }) {
  const tone = stateTone(state);
  return <span className={"inline-flex items-center gap-1 rounded border px-1.5 py-0.5 text-[11px] font-medium " + tone.badge}><i aria-hidden className={"h-1.5 w-1.5 rounded-full " + tone.dot} />{title(state)}</span>;
}

function SnapshotCard({ item, onSelect, selected }: { item: MarketOpsSRISnapshot; onSelect?: () => void; selected?: boolean }) {
  const tone = stateTone(item.state);
  const body = <>
    <div className="flex items-start justify-between gap-2">
      <div>
        <div className="font-mono text-xs text-gray-500">{primaryETF(item)} · {title(item.segment_id)}</div>
        <div className={"mt-1 text-2xl font-semibold tabular-nums " + scoreTone(item.composite_score)}>{score(item.composite_score)}</div>
        <div className="text-[11px] font-medium text-gray-500">Composite score</div>
      </div>
      <StateBadge state={item.state} />
    </div>
    <dl className="mt-3 grid grid-cols-2 gap-2 text-xs">
      <div><dt className="text-gray-500">Rank</dt><dd className={"font-semibold " + tone.text}>{item.rank ?? "—"}</dd></div>
      <div><dt className="text-gray-500">Quality</dt><dd><span className={"inline-flex rounded border px-1 py-0.5 text-[10px] font-medium " + qualityTone(item.quality_state)}>{title(item.quality_state)}{item.evidence_quality != null ? " · " + formatPercent(item.evidence_quality) : ""}</span></dd></div>
      <Metric label="Relative strength" value={item.relative_strength_score} />
      <Metric label="Momentum" value={item.momentum_score} />
      <Metric label="Acceleration" value={item.momentum_acceleration} />
      <div><dt className="text-gray-500">Source session</dt><dd className="font-medium tabular-nums">{item.session_date}</dd></div>
    </dl>
  </>;
  if (!onSelect) {
    return <article className={"rounded border border-gray-200 border-l-4 bg-white p-3 shadow-sm " + tone.accent}>{body}<details className="mt-3"><summary className="cursor-pointer text-xs font-medium text-brand-700">Method and inputs</summary><pre className="mt-2 max-h-48 overflow-auto rounded bg-gray-50 p-2 text-[10px] text-gray-700">{JSON.stringify({ components: item.components, quality_flags: item.quality_flags, input_provenance: item.input_provenance, algorithm_version: item.algorithm_version }, null, 2)}</pre></details></article>;
  }
  return <button type="button" onClick={onSelect} aria-pressed={selected} className={"w-full rounded border border-gray-200 border-l-4 bg-white p-3 text-left shadow-sm transition hover:border-brand-400 focus:outline-none focus:ring-2 focus:ring-brand-500 " + tone.accent + (selected ? " ring-2 ring-brand-500" : "")}>{body}<div className="mt-3 text-xs font-medium text-brand-700">{selected ? "Detail open" : "Open 60-session progression"}</div></button>;
}

function RankingsTab({ snapshots, type, state }: { snapshots: MarketOpsSRISnapshot[]; type: string; state: string }) {
  const filtered = snapshots.filter((item) => (!type || item.segment_id.includes("_" + type + "_")) && (!state || item.state === state));
  return <>{filtered.length ? <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">{filtered.map((item) => <SnapshotCard key={item.snapshot_id} item={item} />)}</div> : <EmptyState message="No Sector Rotation Intelligence snapshots match the current filters." />}</>;
}

function ProgressionChart({ snapshot, history }: { snapshot: MarketOpsSRISnapshot; history: MarketOpsSRISnapshot[] }) {
  const [metric, setMetric] = useState<MetricKey>("composite_score");
  const points = useMemo(() => history.slice().reverse(), [history]);
  const metricLabel = metrics.find((item) => item.key === metric)?.label ?? "Composite score";
  const option = useMemo(() => ({
    grid: { left: 42, right: 18, top: 32, bottom: 44 },
    tooltip: {
      trigger: "axis",
      formatter: (items: Array<{ dataIndex?: number }>) => {
        const point = points[items?.[0]?.dataIndex ?? 0];
        if (!point) return "";
        const value = point[metric];
        return "<b>" + point.session_date + "</b><br/>" + metricLabel + ": " + score(value) + "<br/>State: " + title(point.state) + "<br/>Rank: " + (point.rank ?? "—") + "<br/>Quality: " + title(point.quality_state);
      },
    },
    xAxis: { type: "category", data: points.map((point) => point.session_date), axisLabel: { fontSize: 10, interval: Math.max(0, Math.ceil(points.length / 7) - 1), formatter: (value: string) => value.slice(5) } },
    yAxis: { type: "value", min: 0, max: 100, name: "score", nameTextStyle: { fontSize: 9 }, axisLabel: { fontSize: 9 } },
    series: [{
      name: metricLabel,
      type: "line",
      smooth: true,
      connectNulls: false,
      lineStyle: { color: "#4f46e5", width: 2 },
      symbolSize: 7,
      data: points.map((point) => ({ value: point[metric] ?? null, itemStyle: { color: stateTone(point.state).chart } })),
      markLine: metric === "composite_score" ? { silent: true, symbol: "none", label: { show: false }, data: [{ yAxis: 75 }, { yAxis: 60 }, { yAxis: 40 }, { yAxis: 25 }], lineStyle: { color: "#cbd5e1", type: "dashed" } } : undefined,
    }],
  }), [metric, metricLabel, points]);

  return <section className="rounded border border-brand-200 bg-brand-50 p-3" aria-label={primaryETF(snapshot) + " progression"}>
    <div className="flex flex-wrap items-start justify-between gap-2">
      <div><h2 className="text-sm font-semibold text-brand-900">{primaryETF(snapshot)} · {title(snapshot.segment_id)} progression</h2><p className="text-xs text-brand-800">Persisted price-led Sector Rotation Intelligence history. Point color represents the recorded state; ranks and quality are available in the tooltip.</p></div>
      <label className="text-xs font-medium text-brand-900">Metric<select value={metric} onChange={(event) => setMetric(event.target.value as MetricKey)} className="ml-2 rounded border border-brand-300 bg-white px-2 py-1 text-xs">{metrics.map((item) => <option key={item.key} value={item.key}>{item.label}</option>)}</select></label>
    </div>
    {points.length ? <div className="mt-3 rounded bg-white p-2"><ThemedEChart option={option} style={{ height: 300 }} /></div> : <EmptyState message="No persisted progression history is available for this ETF." />}
  </section>;
}

function ETFMakeup({ makeup, loading, error }: { makeup?: MarketOpsSRIETFMakeupResponse; loading: boolean; error: unknown }) {
  if (loading) return <LoadingState label="Loading current issuer holdings..." />;
  if (error) return <ErrorState error={error} />;
  if (!makeup || makeup.availability !== "available" || !makeup.snapshot) {
    return <div className="rounded border border-amber-200 bg-amber-50 p-3 text-xs text-amber-900"><div className="font-semibold">Current ETF makeup is not available</div><p className="mt-1">{makeup?.reason ?? "A current issuer-published holdings snapshot has not yet been collected."}</p></div>;
  }
  const snapshot = makeup.snapshot;
  return <section className="rounded border border-brand-200 bg-brand-50 p-3" aria-label={(makeup.etf_symbol ?? "ETF") + " current makeup"}>
    <div className="flex flex-wrap items-start justify-between gap-2">
      <div><h2 className="text-sm font-semibold text-brand-900">{makeup.etf_symbol} · current ETF makeup</h2><p className="text-xs text-brand-800">{snapshot.fund_name} · effective {snapshot.effective_date} · {snapshot.holdings_count} holdings</p></div>
      <a href={snapshot.source_url} target="_blank" rel="noreferrer" className="rounded border border-brand-300 bg-white px-2 py-1 text-xs font-medium text-brand-800 hover:bg-brand-100">Issuer source ↗</a>
    </div>
    <dl className="mt-3 grid grid-cols-3 gap-2 rounded bg-white p-2 text-xs"><div><dt className="text-gray-500">Constituents</dt><dd className="font-semibold tabular-nums">{snapshot.holdings_count}</dd></div><div><dt className="text-gray-500">Reported weight</dt><dd className="font-semibold tabular-nums">{snapshot.total_weight.toFixed(2)}%</dd></div><div><dt className="text-gray-500">Top 10 weight</dt><dd className="font-semibold tabular-nums">{snapshot.top_ten_weight.toFixed(2)}%</dd></div></dl>
    <div className="mt-3 max-w-full overflow-x-auto rounded border border-gray-200 bg-white overscroll-x-contain"><table className="min-w-[680px] divide-y divide-gray-200 text-xs"><thead className="bg-gray-50 text-left uppercase tracking-wide text-gray-500"><tr><th className="px-2 py-2">#</th><th className="px-2 py-2">Holding</th><th className="px-2 py-2">Ticker</th><th className="px-2 py-2 text-right">Weight</th><th className="px-2 py-2">Sector</th></tr></thead><tbody className="divide-y divide-gray-100">{makeup.holdings.map((holding) => <tr key={holding.rank}><td className="px-2 py-2 tabular-nums text-gray-500">{holding.rank}</td><td className="px-2 py-2 font-medium text-gray-900">{holding.name}</td><td className="px-2 py-2 font-mono text-gray-700">{holding.ticker || "—"}</td><td className="px-2 py-2 text-right font-semibold tabular-nums text-gray-900">{holding.weight.toFixed(2)}%</td><td className="px-2 py-2 text-gray-600">{holding.sector || "—"}</td></tr>)}</tbody></table></div>
    <p className="mt-2 text-[11px] text-brand-800">{makeup.evidence_note ?? "Current issuer-published representation only; it does not affect Sector Rotation Intelligence scores or reconstruct historical holdings."}</p>
  </section>;
}

function ETFDetail({ snapshot, history, historyLoading, historyError, makeup, makeupLoading, makeupError }: { snapshot: MarketOpsSRISnapshot; history: MarketOpsSRISnapshot[]; historyLoading: boolean; historyError: unknown; makeup?: MarketOpsSRIETFMakeupResponse; makeupLoading: boolean; makeupError: unknown }) {
  const [detailTab, setDetailTab] = useState<"progression" | "makeup">("progression");
  useEffect(() => setDetailTab("progression"), [snapshot.segment_id]);
  return <div className="space-y-2"><div role="tablist" aria-label={primaryETF(snapshot) + " detail views"} className="flex border-b border-brand-200"><button role="tab" aria-selected={detailTab === "progression"} onClick={() => setDetailTab("progression")} className={"border-b-2 px-3 py-2 text-xs font-medium " + (detailTab === "progression" ? "border-brand-600 text-brand-800" : "border-transparent text-gray-600")}>Progression</button><button role="tab" aria-selected={detailTab === "makeup"} onClick={() => setDetailTab("makeup")} className={"border-b-2 px-3 py-2 text-xs font-medium " + (detailTab === "makeup" ? "border-brand-600 text-brand-800" : "border-transparent text-gray-600")}>ETF makeup</button></div>{detailTab === "progression" ? historyLoading ? <LoadingState label={"Loading " + primaryETF(snapshot) + " progression..."} /> : historyError ? <ErrorState error={historyError} /> : <ProgressionChart snapshot={snapshot} history={history} /> : <ETFMakeup makeup={makeup} loading={makeupLoading} error={makeupError} />}</div>;
}

function ProgressionTab({ tenantId, snapshots }: { tenantId: string; snapshots: MarketOpsSRISnapshot[] }) {
  type SortKey = "rank" | "etf" | "score" | "state" | "strength" | "momentum" | "quality" | "session";
  const [selected, setSelected] = useState<MarketOpsSRISnapshot | null>(null);
  const [sort, setSort] = useState<{ key: SortKey; direction: "asc" | "desc" }>({ key: "rank", direction: "asc" });
  const detailRef = useRef<HTMLDivElement | null>(null);
  const historyQ = useQuery({
    queryKey: ["marketops-sri-history", tenantId, selected?.segment_id],
    queryFn: () => api.getMarketOpsSRIHistory(tenantId, selected?.segment_id ?? "", 60),
    enabled: selected != null,
    staleTime: 30_000,
  });
  const makeupQ = useQuery({
    queryKey: ["marketops-sri-etf-makeup", tenantId, selected?.segment_id],
    queryFn: () => api.getMarketOpsSRIETFMakeup(tenantId, selected?.segment_id ?? "", 25),
    enabled: selected != null,
    staleTime: 5 * 60_000,
  });
  const cards = useMemo(() => {
    const valueFor = (item: MarketOpsSRISnapshot): string | number => {
      switch (sort.key) {
        case "etf": return primaryETF(item) + " " + title(item.segment_id);
        case "score": return item.composite_score ?? -1;
        case "state": return item.state;
        case "strength": return item.relative_strength_score ?? -1;
        case "momentum": return item.momentum_score ?? -1;
        case "quality": return item.evidence_quality ?? -1;
        case "session": return item.session_date;
        default: return item.rank ?? Number.MAX_SAFE_INTEGER;
      }
    };
    return snapshots
      .filter((item) => item.primary_etf || primaryETF(item) !== "ETF")
      .slice()
      .sort((a, b) => {
        const left = valueFor(a);
        const right = valueFor(b);
        const comparison = typeof left === "string" && typeof right === "string" ? left.localeCompare(right) : Number(left) - Number(right);
        return (comparison || primaryETF(a).localeCompare(primaryETF(b))) * (sort.direction === "asc" ? 1 : -1);
      });
  }, [snapshots, sort]);
  useEffect(() => { if (selected) detailRef.current?.scrollIntoView({ behavior: "smooth", block: "nearest" }); }, [selected]);

  const toggleSort = (key: SortKey) => setSort((current) => current.key === key ? { key, direction: current.direction === "asc" ? "desc" : "asc" } : { key, direction: key === "etf" || key === "state" || key === "quality" || key === "session" ? "asc" : "desc" });
  const heading = (label: string, key: SortKey) => <button type="button" onClick={() => toggleSort(key)} className="inline-flex items-center gap-1 text-left hover:text-brand-700">{label}<span aria-hidden className={sort.key === key ? "text-brand-700" : "text-gray-300"}>{sort.key === key ? (sort.direction === "asc" ? "↑" : "↓") : "↕"}</span></button>;

  if (!cards.length) return <EmptyState message="No scored primary ETF snapshots are ready for progression analysis." />;
  return <div className="space-y-3">
    <div className="rounded border border-gray-200 bg-gray-50 p-2 text-xs text-gray-600">{cards.length} primary ETFs · select an ETF row to reveal its persisted 60-session score progression or current issuer makeup. This is analytical context, not a rotation claim or recommendation.</div>
    <div className="space-y-1">
      <p className="px-1 text-xs text-gray-500 md:hidden">Swipe horizontally to view all ETF columns.</p>
      <div className="max-w-full overflow-x-auto rounded border border-gray-200 bg-white overscroll-x-contain" role="region" aria-label="Scrollable ETF progression table" tabIndex={0}>
        <table className="min-w-[980px] table-fixed divide-y divide-gray-200 text-sm">
          <colgroup><col className="w-16" /><col className="w-64" /><col className="w-28" /><col className="w-32" /><col className="w-72" /><col className="w-44" /><col className="w-32" /></colgroup>
          <thead className="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500"><tr>
            <th className="px-3 py-2">{heading("Rank", "rank")}</th>
            <th className="px-3 py-2">{heading("ETF / segment", "etf")}</th>
            <th className="px-3 py-2">{heading("Sector Rotation score", "score")}</th>
            <th className="px-3 py-2">{heading("Context", "state")}</th>
            <th className="px-3 py-2"><div className="flex items-center gap-5">{heading("Relative strength", "strength")}{heading("Momentum", "momentum")}</div></th>
            <th className="px-3 py-2">{heading("Quality", "quality")}</th>
            <th className="px-3 py-2">{heading("Updated", "session")}</th>
          </tr></thead>
          <tbody className="divide-y divide-gray-100">{cards.map((item) => {
            const tone = stateTone(item.state);
            const isSelected = selected?.segment_id === item.segment_id;
            return <Fragment key={item.snapshot_id}>
              <tr onClick={() => setSelected((current) => current?.segment_id === item.segment_id ? null : item)} className={"cursor-pointer align-top hover:bg-gray-50 " + (isSelected ? "bg-brand-50" : "")} aria-expanded={isSelected}>
                <td className={"px-3 py-3 text-xs font-semibold tabular-nums " + tone.text}>{item.rank ?? "—"}</td>
                <td className="px-3 py-3"><div className="flex items-start gap-2"><span className="mt-0.5 inline-flex min-w-10 justify-center rounded bg-brand-50 px-1.5 py-0.5 font-mono text-xs font-semibold text-brand-800">{primaryETF(item)}</span><div><div className="font-medium text-gray-900">{title(item.segment_id)}</div><div className="text-xs text-gray-500">Primary sector ETF · open detail</div></div></div></td>
                <td className="px-3 py-3"><div className={"text-base font-semibold tabular-nums " + scoreTone(item.composite_score)}>{score(item.composite_score)}</div><div className="text-[10px] text-gray-500">Composite / 100</div></td>
                <td className="px-3 py-3"><StateBadge state={item.state} /></td>
                <td className="px-3 py-3 text-xs"><div className="grid grid-cols-3 gap-4"><div><div className="text-gray-500">RS</div><div className={"font-semibold tabular-nums " + scoreTone(item.relative_strength_score)}>{score(item.relative_strength_score)}</div></div><div><div className="text-gray-500">Momentum</div><div className={"font-semibold tabular-nums " + scoreTone(item.momentum_score)}>{score(item.momentum_score)}</div></div><div><div className="text-gray-500">Accel.</div><div className={"font-semibold tabular-nums " + scoreTone(item.momentum_acceleration)}>{score(item.momentum_acceleration)}</div></div></div></td>
                <td className="px-3 py-3 text-xs"><span className={"inline-flex rounded border px-1.5 py-0.5 text-[10px] font-medium " + qualityTone(item.quality_state)}>{title(item.quality_state)}</span><div className="mt-1 text-[10px] text-gray-500">Evidence {item.evidence_quality != null ? formatPercent(item.evidence_quality) : "—"}</div></td>
                <td className="px-3 py-3 text-xs text-gray-600"><div className="font-medium tabular-nums">{item.session_date}</div><div className="mt-1 text-[10px] text-brand-700">{isSelected ? "Detail open" : "Open detail"}</div></td>
              </tr>
              {isSelected ? <tr className="bg-brand-50"><td colSpan={7} className="p-3"><div ref={detailRef}>{<ETFDetail snapshot={item} history={historyQ.data?.snapshots ?? []} historyLoading={historyQ.isLoading} historyError={historyQ.isError ? historyQ.error : undefined} makeup={makeupQ.data} makeupLoading={makeupQ.isLoading} makeupError={makeupQ.isError ? makeupQ.error : undefined} />}</div></td></tr> : null}
            </Fragment>;
          })}</tbody>
        </table>
      </div>
    </div>
  </div>;
}

export function MarketOpsSRIRoute() {
  const tenantId = useTenant();
  const [tab, setTab] = useState<Tab>("rankings");
  const [type, setType] = useState("");
  const [state, setState] = useState("");
  const rankingsQ = useQuery({ queryKey: ["marketops-sri-rankings", tenantId], queryFn: () => api.getMarketOpsSRIRankings(tenantId), staleTime: 30_000 });
  const legend = ["LEADING", "IMPROVING", "NEUTRAL", "WEAKENING", "LAGGING"];
  const refresh = () => void rankingsQ.refetch();

  return <div className="space-y-3">
    <div className="flex flex-wrap items-start justify-between gap-3">
      <div><h1 className="text-lg font-semibold">Sector Rotation Intelligence</h1><p className="max-w-3xl text-xs text-gray-500">Research-only, price-led market-segment context. This foundation ranks relative strength and momentum; it does not claim rotation, breadth, diffusion, flows, or a trade recommendation.</p></div>
      <RefreshButton onClick={refresh} loading={rankingsQ.isFetching} />
    </div>

    <div role="tablist" aria-label="Sector Rotation Intelligence views" className="flex border-b border-gray-200">
      <button role="tab" aria-selected={tab === "rankings"} onClick={() => setTab("rankings")} className={"border-b-2 px-3 py-2 text-sm font-medium " + (tab === "rankings" ? "border-brand-600 text-brand-700" : "border-transparent text-gray-600 hover:text-gray-900")}>Rankings</button>
      <button role="tab" aria-selected={tab === "progression"} onClick={() => setTab("progression")} className={"border-b-2 px-3 py-2 text-sm font-medium " + (tab === "progression" ? "border-brand-600 text-brand-700" : "border-transparent text-gray-600 hover:text-gray-900")}>ETF progression</button>
    </div>

    {tab === "rankings" ? <div className="flex flex-wrap gap-3 rounded border border-gray-200 bg-gray-50 p-3">
      <label className="text-xs font-medium text-gray-700">Segment<select value={type} onChange={(event) => setType(event.target.value)} className="mt-1 block rounded border border-gray-300 bg-white px-2 py-1.5 text-sm"><option value="">All types</option><option value="sector">Sectors</option><option value="industry">Industries</option></select></label>
      <label className="text-xs font-medium text-gray-700">Context<select value={state} onChange={(event) => setState(event.target.value)} className="mt-1 block rounded border border-gray-300 bg-white px-2 py-1.5 text-sm"><option value="">All states</option><option value="LEADING">Leading</option><option value="IMPROVING">Improving</option><option value="NEUTRAL">Neutral</option><option value="WEAKENING">Weakening</option><option value="LAGGING">Lagging</option></select></label>
      <div className="self-end pb-1 text-xs text-gray-500">Score: <span className="font-medium text-emerald-700">75+ strong</span> · <span className="font-medium text-sky-700">60+ positive</span> · <span className="font-medium text-amber-700">&lt;40 weak</span></div>
    </div> : null}

    <div aria-label="Sector Rotation Intelligence context color legend" className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-gray-600"><span className="font-medium text-gray-700">Context legend:</span>{legend.map((item) => { const tone = stateTone(item); return <span key={item} className="inline-flex items-center gap-1"><i aria-hidden className={"h-2 w-2 rounded-full " + tone.dot} />{title(item)}</span>; })}</div>

    {rankingsQ.isLoading ? <LoadingState label="Loading sector context..." /> : rankingsQ.isError ? <ErrorState error={rankingsQ.error} /> : <>
      <p className="rounded border border-amber-200 bg-amber-50 p-2 text-xs text-amber-800">{rankingsQ.data?.evidence_note}</p>
      {tab === "rankings" ? <RankingsTab snapshots={rankingsQ.data?.snapshots ?? []} type={type} state={state} /> : <ProgressionTab tenantId={tenantId} snapshots={rankingsQ.data?.snapshots ?? []} />}
    </>}
  </div>;
}
