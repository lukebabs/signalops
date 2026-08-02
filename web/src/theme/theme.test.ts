import { describe, expect, it } from 'vitest';
import { readThemePreference, resolveTheme } from './theme';

describe('theme preference', () => {
  it('uses a valid persisted preference and safely falls back to system', () => {
    expect(readThemePreference('dark')).toBe('dark');
    expect(readThemePreference('invalid')).toBe('system');
    expect(readThemePreference(null)).toBe('system');
  });

  it('resolves system preference from the operating system', () => {
    expect(resolveTheme('system', true)).toBe('dark');
    expect(resolveTheme('system', false)).toBe('light');
    expect(resolveTheme('light', true)).toBe('light');
    expect(resolveTheme('dark', false)).toBe('dark');
  });
});
