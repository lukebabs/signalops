import { describe, expect, it } from 'vitest';
import { duration, formatUtc, orDash, truncate } from './format';

describe('format helpers', () => {
  it('formats UTC timestamps and preserves invalid values', () => {
    expect(formatUtc('2026-07-08T03:46:26.123Z')).toBe('2026-07-08 03:46:26Z');
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
});
