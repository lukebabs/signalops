import ReactECharts from "echarts-for-react";

import type { MarketOpsSignalOverviewMember, MarketOpsSignalOverviewPoint, MarketOpsSignalOverviewResponse } from "../types";

type Drilldown = { title: string; members: MarketOpsSignalOverviewMember[] };

export type RiskRewardRegimePoint = {
  tradeDate: string;
  median: number;
  lowerQuartile: number;
  upperQuartile: number;
  members: MarketOpsSignalOverviewMember[];
};

function uniqueMembers(members: MarketOpsSignalOverviewMember[]) {
  return [...new Map(members.map((member) => [member.ticker, member])).values()];
}

function percentile(sorted: number[], quantile: number): number {
  if (!sorted.length) return 0;
  const position = (sorted.length - 1) * quantile;
  const lower = Math.floor(position);
  const upper = Math.ceil(position);
  if (lower === upper) return sorted[lower];
  return sorted[lower] + (sorted[upper] - sorted[lower]) * (position - lower);
}

export function riskRewardRegimePoints(points: MarketOpsSignalOverviewPoint[]): RiskRewardRegimePoint[] {
  return points.flatMap((point) => {
    const members = uniqueMembers(point.categories.flatMap((category) => category.members)).filter((member) => typeof member.score === "number" && Number.isFinite(member.score));
    if (!members.length) return [];
    const scores = members.map((member) => member.score as number).sort((left, right) => left - right);
    return [{
      tradeDate: point.trade_date,
      median: percentile(scores, 0.5),
      lowerQuartile: percentile(scores, 0.25),
      upperQuartile: percentile(scores, 0.75),
      members,
    }];
  });
}

export function RiskRewardRegimeChart({ data, onDrilldown }: { data: MarketOpsSignalOverviewResponse; onDrilldown: (value: Drilldown) => void }) {
  const regime = riskRewardRegimePoints(data.risk_reward.points);
  const band = regime.slice(0, -1).map((point, index) => [index, point.lowerQuartile, point.upperQuartile, regime[index + 1].lowerQuartile, regime[index + 1].upperQuartile]);
  const option = {
    grid: { left: 42, right: 16, top: 38, bottom: 34 },
    tooltip: {
      trigger: "axis",
      formatter: (values: Array<{ dataIndex?: number }>) => {
        const point = regime[values[0]?.dataIndex ?? -1];
        if (!point) return "";
        return point.tradeDate + "<br/>Median: " + point.median.toFixed(1) + "<br/>25th–75th: " + point.lowerQuartile.toFixed(1) + " to " + point.upperQuartile.toFixed(1) + "<br/>Scored assets: " + point.members.length;
      },
    },
    legend: { data: ["Median score", "25th–75th range"], top: 0 },
    xAxis: { type: "category", data: regime.map((point) => point.tradeDate), axisLabel: { fontSize: 9 } },
    yAxis: { type: "value", min: -100, max: 100, name: "technical score", nameTextStyle: { fontSize: 9 }, axisLabel: { fontSize: 9 } },
    series: [
      {
        name: "25th–75th range",
        type: "custom",
        silent: true,
        data: band,
        renderItem: (_params: unknown, api: { value: (index: number) => number; coord: (value: [number, number]) => [number, number] }) => {
          const lowerStart = api.coord([api.value(0), api.value(1)]);
          const upperStart = api.coord([api.value(0), api.value(2)]);
          const lowerEnd = api.coord([api.value(0) + 1, api.value(3)]);
          const upperEnd = api.coord([api.value(0) + 1, api.value(4)]);
          return { type: "polygon", shape: { points: [lowerStart, upperStart, upperEnd, lowerEnd] }, style: { fill: "rgba(79, 70, 229, 0.16)", stroke: "none" } };
        },
      },
      {
        name: "Median score",
        type: "line",
        smooth: true,
        showSymbol: false,
        data: regime.map((point) => point.median),
        lineStyle: { width: 2, color: "#4f46e5" },
        itemStyle: { color: "#4f46e5" },
        markLine: { silent: true, symbol: "none", lineStyle: { color: "#64748b", type: "dashed" }, data: [{ yAxis: 0 }] },
      },
    ],
  };
  const onClick = (event: { dataIndex?: number }) => {
    const point = regime[event.dataIndex ?? -1];
    if (!point) return;
    const members = [...point.members].sort((left, right) => (right.score ?? 0) - (left.score ?? 0) || left.ticker.localeCompare(right.ticker));
    onDrilldown({ title: "Risk/Reward regime · " + point.tradeDate, members });
  };
  return <ChartShell title="Risk/Reward regime" subtitle="Median technical score shows aggregate posture; the 25th–75th band shows consensus (narrow) or fragmentation (wide). Select a date to inspect scored assets.">{regime.length ? <ReactECharts option={option} onEvents={{ click: onClick }} style={{ height: 260 }} /> : <div className="py-12 text-xs text-gray-500">No scored Risk/Reward observations are available for this window.</div>}</ChartShell>;
}

export function TechnicalScoreDistributionChart({ data, onDrilldown }: { data: MarketOpsSignalOverviewResponse; onDrilldown: (value: Drilldown) => void }) {
  const latest = data.risk_reward.points.at(-1);
  const bins = [{ label: "≤ −50", match: (score: number) => score <= -50 }, { label: "−49 to −11", match: (score: number) => score > -50 && score < -10 }, { label: "−10 to +10", match: (score: number) => score >= -10 && score <= 10 }, { label: "+11 to +49", match: (score: number) => score > 10 && score < 50 }, { label: "≥ +50", match: (score: number) => score >= 50 }];
  const members = latest ? latest.categories.flatMap((category) => category.members).filter((member) => member.score != null) : [];
  const grouped = bins.map((bin) => ({ ...bin, members: members.filter((member) => bin.match(member.score!)) }));
  const option = { grid: { left: 38, right: 12, top: 28, bottom: 42 }, tooltip: { trigger: "axis" }, xAxis: { type: "category", data: grouped.map((bin) => bin.label), axisLabel: { fontSize: 9 } }, yAxis: { type: "value", minInterval: 1, name: "assets", nameTextStyle: { fontSize: 9 }, axisLabel: { fontSize: 9 } }, series: [{ type: "bar", data: grouped.map((bin, index) => ({ value: bin.members.length, itemStyle: { color: index < 2 ? "#dc2626" : index === 2 ? "#6b7280" : "#15803d" } })) }] };
  const events = { click: (event: { dataIndex?: number }) => { const bin = grouped[event.dataIndex ?? -1]; if (latest && bin?.members.length) onDrilldown({ title: "Technical score distribution · " + latest.trade_date + " · " + bin.label, members: bin.members }); } };
  return <ChartShell title="Technical score distribution" subtitle={latest ? "Latest persisted Risk/Reward snapshot · " + latest.trade_date + ". Select a bar to inspect its assets." : "No persisted Risk/Reward snapshot is available."}>{members.length ? <ReactECharts option={option} onEvents={events} style={{ height: 260 }} /> : <div className="py-12 text-xs text-gray-500">No scored assets in the selected window.</div>}</ChartShell>;
}

function ChartShell({ title, subtitle, children }: { title: string; subtitle: string; children: React.ReactNode }) {
  return <div className="rounded border border-gray-200 bg-white p-3"><div className="mb-1"><div className="text-xs font-semibold text-gray-800">{title}</div><p className="text-[11px] text-gray-500">{subtitle}</p></div>{children}</div>;
}
