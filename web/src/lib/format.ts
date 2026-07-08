// Timestamps are displayed as UTC, consistently. Absent optional fields render as —.

export function formatUtc(iso?: string): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (isNaN(d.getTime())) return iso;
  // YYYY-MM-DD HH:MM:SSZ
  return d.toISOString().replace('T', ' ').replace(/\.\d+Z$/, 'Z');
}

export function duration(startedAt?: string, completedAt?: string): string {
  if (!startedAt || !completedAt) return '—';
  const start = new Date(startedAt).getTime();
  const end = new Date(completedAt).getTime();
  const ms = end - start;
  if (isNaN(ms) || ms < 0) return '—';
  if (ms < 1000) return `${ms} ms`;
  const s = ms / 1000;
  if (s < 60) return `${s.toFixed(2)} s`;
  const m = Math.floor(s / 60);
  const rs = Math.round(s % 60);
  return `${m}m ${rs}s`;
}

export function orDash(value: string | number | null | undefined): string {
  if (value === null || value === undefined || value === '') return '—';
  return String(value);
}

export function truncate(value: string, n = 24): string {
  return value.length > n ? `${value.slice(0, n - 1)}…` : value;
}
