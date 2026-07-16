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

// Format a 0..1 ratio as a human-readable percentage. Whole percents render
// without a decimal (1 -> "100%", 0 -> "0%"); fractional ratios get one decimal
// place (0.333 -> "33.3%"). Non-finite values collapse to —.
export function formatPercent(ratio: number): string {
  if (!Number.isFinite(ratio)) return '—';
  const pct = ratio * 100;
  return Number.isInteger(pct) ? `${pct}%` : `${pct.toFixed(1)}%`;
}

export function truncate(value: string, n = 24): string {
  return value.length > n ? `${value.slice(0, n - 1)}…` : value;
}

// datetime-local inputs yield a naive "YYYY-MM-DDTHH:mm[:ss]" string with no
// timezone. The replay backend parses window_start/window_end as RFC3339, and
// every SignalOps timestamp is UTC, so the entered wall-clock is treated as UTC.
export function toRfc3339Utc(local: string): string {
  const v = local.trim();
  if (!v) return '';
  // Already timezone-qualified (Z or offset) — pass through.
  if (/[zZ]$|[+-]\d\d:?\d\d$/.test(v)) return v;
  // Ensure a seconds component before appending the UTC designator.
  const withSeconds = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}$/.test(v) ? v : `${v}:00`;
  return `${withSeconds}Z`;
}

// Pre-fill a datetime-local input from a UTC ISO string (UTC wall-clock).
export function toDatetimeLocal(iso: string): string {
  const v = iso.trim();
  if (!v) return '';
  return v.slice(0, 16);
}
