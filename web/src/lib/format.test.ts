import { describe, expect, it } from 'vitest';
import { duration, formatUtc, formatPercent, orDash, truncate, toRfc3339Utc, toDatetimeLocal } from './format';

describe('format helpers', () => {
  it('formats instants in the Eastern Time operating timezone and preserves date-only values', () => {
    expect(formatUtc('2026-07-08T03:46:26.123Z')).toBe('2026-07-07 23:46:26 ET');
    expect(formatUtc('2026-01-08T03:46:26.123Z')).toBe('2026-01-07 22:46:26 ET');
    expect(formatUtc('2026-07-08')).toBe('2026-07-08');
    expect(formatUtc('not-a-date')).toBe('not-a-date');
    expect(formatUtc()).toBe('—');
  });

  it('formats durations defensively', () => {
    expect(duration('2026-07-08T00:00:00Z', '2026-07-08T00:00:00.250Z')).toBe('250 ms');
    expect(duration('2026-07-08T00:00:00Z', '2026-07-08T00:00:02Z')).toBe('2.00 s');
    expect(duration('2026-07-08T00:00:00Z', '2026-07-08T00:01:05Z')).toBe('1m 5s');
    expect(duration('2026-07-08T00:01:00Z', '2026-07-08T00:00:00Z')).toBe('—');
  });

  it('renders fallback and truncation values', () => {
    expect(orDash('')).toBe('—');
    expect(orDash(42)).toBe('42');
    expect(truncate('abcdefghijklmnopqrstuvwxyz', 8)).toBe('abcdefg…');
    expect(truncate('abc', 8)).toBe('abc');
  });

  it('converts datetime-local values to RFC3339 UTC for the replay backend', () => {
    // datetime-local yields a naive wall-clock; treat it as UTC.
    expect(toRfc3339Utc('2026-07-09T00:00')).toBe('2026-07-09T00:00:00Z');
    expect(toRfc3339Utc('2026-07-09T00:00:30')).toBe('2026-07-09T00:00:30Z');
    expect(toRfc3339Utc('')).toBe('');
    // Already timezone-qualified values pass through unchanged.
    expect(toRfc3339Utc('2026-07-09T00:00:00Z')).toBe('2026-07-09T00:00:00Z');
    expect(toRfc3339Utc('2026-07-09T00:00:00+02:00')).toBe('2026-07-09T00:00:00+02:00');
  });

  it('pre-fills datetime-local inputs from UTC ISO strings', () => {
    expect(toDatetimeLocal('2026-07-09T00:00:00Z')).toBe('2026-07-09T00:00');
    expect(toDatetimeLocal('')).toBe('');
  });

  it('formats ratios as human-readable percentages (G116)', () => {
    expect(formatPercent(1)).toBe('100%');
    expect(formatPercent(0)).toBe('0%');
    expect(formatPercent(0.5)).toBe('50%');
    expect(formatPercent(0.3333)).toBe('33.3%');
    expect(formatPercent(NaN)).toBe('—');
  });
});
