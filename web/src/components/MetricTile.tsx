import type { ReactNode } from 'react';

export function MetricTile({
  label,
  value,
  hint,
}: {
  label: string;
  value: ReactNode;
  hint?: ReactNode;
}) {
  return (
    <div className="h-full rounded border border-gray-200 bg-white p-3">
      <div className="text-xs uppercase tracking-wide text-gray-500">{label}</div>
      <div className="mt-1 text-base font-semibold text-gray-900">{value}</div>
      {hint && <div className="mt-0.5 break-all text-xs text-gray-500">{hint}</div>}
    </div>
  );
}
