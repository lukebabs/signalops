import ReactECharts from 'echarts-for-react';
import type { SchedulerRun } from '../types';

// Minimal, real ECharts usage: provider requests across the loaded recent runs.
export function RunsBarChart({ runs }: { runs: SchedulerRun[] }) {
  if (!runs.length) return null;
  const option = {
    grid: { left: 48, right: 16, top: 16, bottom: 40 },
    tooltip: { trigger: 'axis' },
    xAxis: {
      type: 'category',
      data: runs.map((r, i) => `${r.source_id} #${i + 1}`),
      axisLabel: { fontSize: 10, rotate: 30 },
    },
    yAxis: { type: 'value', minInterval: 1 },
    series: [
      {
        name: 'Provider requests',
        type: 'bar',
        data: runs.map((r) => r.provider_requests),
        itemStyle: { color: '#1f7a6b' },
      },
    ],
  };
  return <ReactECharts option={option} style={{ height: 200 }} />;
}
