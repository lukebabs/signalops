import { afterEach, describe, expect, it, vi } from 'vitest';
import { getAccessToken, setAccessTokenForTest } from './session';

function jwtWithExp(exp: number): string {
  const encode = (value: unknown) => btoa(JSON.stringify(value)).replace(/=/g, '').replace(/\+/g, '-').replace(/\//g, '_');
  return `${encode({ alg: 'none' })}.${encode({ exp })}.signature`;
}

afterEach(() => {
  vi.useRealTimers();
  setAccessTokenForTest(null);
});

describe('session access token expiry guard', () => {
  it('does not return an expired JWT from the module token cache', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-08-25T14:00:00Z'));
    setAccessTokenForTest(jwtWithExp(Math.floor(Date.now() / 1000) - 1));

    expect(getAccessToken()).toBeNull();
  });

  it('keeps non-JWT opaque test tokens and valid JWTs', () => {
    setAccessTokenForTest('jwt-abc');
    expect(getAccessToken()).toBe('jwt-abc');

    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-08-25T14:00:00Z'));
    const token = jwtWithExp(Math.floor(Date.now() / 1000) + 120);
    setAccessTokenForTest(token);
    expect(getAccessToken()).toBe(token);
  });
});
