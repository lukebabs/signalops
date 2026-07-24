import ReactECharts from 'echarts-for-react';

import type { MarketOpsSignalOverviewMember, MarketOpsSignalOverviewResponse } from '../types';

type Drilldown = { title: string; members: MarketOpsSignalOverviewMember[] };

function uniqueMembers(members: MarketOpsSignalOverviewMember[]) {
  return [...new Map(members.map((member) => [member.ticker, member])).values()];
}

export function SignalOverviewCoverageChart({ data }: { data: MarketOpsSignalOverviewResponse }) {
  const risk = data.risk_reward.points;
  const hypothesis = data.hypotheses.points;
  const dates = [...new Set([...risk, ...hypothesis].map((point) => point.trade_date))].sort();
  const coverage = (points: typeof risk, date: string) => {
    const point = points.find((item) => item.trade_date === date);
    const members = point ? uniqueMembers(point.categories.flatMap((category) => category.members)) : [];
    return data.asset_count ? Number((members.length / data.asset_count * 100).toFixed(1)) : 0;
  };
  const option = { grid: { left: 42, right: 16, top: 38, bottom: 34 }, tooltip: { trigger: 'axis', valueFormatter: (value: number) => `${value.toFixed(1)}%` }, legend: { data: ['Risk/Reward observed', 'Triggered hypotheses'], top: 0 }, xAxis: { type: 'category', data: dates, axisLabel: { fontSize: 9 } }, yAxis: { type: 'value', min: 0, max: 100, name: '% assets', nameTextStyle: { fontSize: 9 }, axisLabel: { fontSize: 9 } }, series: [{ name: 'Risk/Reward observed', type: 'line', connectNulls: false, data: dates.map((date) => coverage(risk, date)), itemStyle: { color: '#4f46e5' } }, { name: 'Triggered hypotheses', type: 'line', connectNulls: false, data: dates.map((date) => coverage(hypothesis, date)), itemStyle: { color: '#d97706' } }] };
  return <ChartShell title="Research observation coverage" subtitle="Share of selected active assets represented by a persisted Risk/Reward result or at least one triggered hypothesis; absence is not treated as neutral."><ReactECharts option={option} style={{ height: 260 }} /></ChartShell>;
}

export function TechnicalScoreDistributionChart({ data, onDrilldown }: { data: MarketOpsSignalOverviewResponse; onDrilldown: (value: Drilldown) => void }) {
  const latest = data.risk_reward.points.at(-1);
  const bins = [{ label: '≤ −50', match: (score: number) => score <= -50 }, { label: '−49 to −11', match: (score: number) => score > -50 && score < -10 }, { label: '−10 to +10', match: (score: number) => score >= -10 && score <= 10 }, { label: '+11 to +49', match: (score: number) => score > 10 && score < 50 }, { label: '≥ +50', match: (score: number) => score >= 50 }];
  const members = latest ? latest.categories.flatMap((category) => category.members).filter((member) => member.score != null) : [];
  const grouped = bins.map((bin) => ({ ...bin, members: members.filter((member) => bin.match(member.score!)) }));
  const option = { grid: { left: 38, right: 12, top: 28, bottom: 42 }, tooltip: { trigger: 'axis' }, xAxis: { type: 'category', data: grouped.map((bin) => bin.label), axisLabel: { fontSize: 9 } }, yAxis: { type: 'value', minInterval: 1, name: 'assets', nameTextStyle: { fontSize: 9 }, axisLabel: { fontSize: 9 } }, series: [{ type: 'bar', data: grouped.map((bin, index) => ({ value: bin.members.length, itemStyle: { color: index < 2 ? '#dc2626' : index === 2 ? '#6b7280' : '#15803d' } })) }] };
  const events = { click: (event: { dataIndex?: number }) => { const bin = grouped[event.dataIndex ?? -1]; if (latest && bin?.members.length) onDrilldown({ title: `Technical score distribution · ${latest.trade_date} · ${bin.label}`, members: bin.members }); } };
  return <ChartShell title="Technical score distribution" subtitle={latest ? `Latest persisted Risk/Reward snapshot · ${latest.trade_date}. Select a bar to inspect its assets.` : 'No persisted Risk/Reward snapshot is available.'}>{members.length ? <ReactECharts option={option} onEvents={events} style={{ height: 260 }} /> : <div className="py-12 text-xs text-gray-500">No scored assets in the selected window.</div>}</ChartShell>;
}

function ChartShell({ title, subtitle, children }: { title: string; subtitle: string; children: React.ReactNode }) {
  return <div className="rounded border border-gray-200 bg-white p-3"><div className="mb-1"><div className="text-xs font-semibold text-gray-800">{title}</div><p className="text-[11px] text-gray-500">{subtitle}</p></div>{children}</div>;
}
