import ReactECharts from 'echarts-for-react';
import type { ComponentProps } from 'react';
import { useTheme } from '../theme/theme';

type ChartProps = ComponentProps<typeof ReactECharts>;

// Centralizes ECharts' canvas theme so axes, legends, and tooltips stay readable.
export function ThemedEChart(props: ChartProps) {
  const { resolvedTheme } = useTheme();
  return <ReactECharts {...props} theme={resolvedTheme === 'dark' ? 'dark' : undefined} />;
}
