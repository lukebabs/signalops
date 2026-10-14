import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  consumeRedirectPath,
  DEFAULT_POST_LOGIN_PATH,
  rememberRedirectPath,
  sanitizeRedirectPath,
} from './oidc';

const REDIRECT_PATH_KEY = 'signalops.auth.redirectPath';

function installSessionStorage(initial: Record<string, string> = {}) {
  const values = new Map(Object.entries(initial));
  vi.stubGlobal('sessionStorage', {
    getItem: (key: string) => values.get(key) ?? null,
    setItem: (key: string, value: string) => values.set(key, value),
    removeItem: (key: string) => values.delete(key),
  });
  return values;
}

afterEach(() => vi.unstubAllGlobals());

describe('OIDC post-login redirect state', () => {
  it('preserves an in-app MarketOps destination', () => {
    const storage = installSessionStorage();

    rememberRedirectPath('/marketops/assets?group=analyst_watchlist');

    expect(storage.get(REDIRECT_PATH_KEY)).toBe('/marketops/assets?group=analyst_watchlist');
    expect(consumeRedirectPath()).toBe('/marketops/assets?group=analyst_watchlist');
    expect(storage.has(REDIRECT_PATH_KEY)).toBe(false);
  });

  it('never stores or restores the signed-out auth route after a successful login', () => {
    const storage = installSessionStorage({ [REDIRECT_PATH_KEY]: '/auth/signed-out' });

    expect(consumeRedirectPath()).toBe(DEFAULT_POST_LOGIN_PATH);
    expect(storage.has(REDIRECT_PATH_KEY)).toBe(false);

    rememberRedirectPath('/auth/signed-out');
    expect(storage.get(REDIRECT_PATH_KEY)).toBe(DEFAULT_POST_LOGIN_PATH);
  });

  it('rejects malformed and external redirect paths', () => {
    expect(sanitizeRedirectPath(undefined)).toBe(DEFAULT_POST_LOGIN_PATH);
    expect(sanitizeRedirectPath('https://untrusted.example')).toBe(DEFAULT_POST_LOGIN_PATH);
    expect(sanitizeRedirectPath('//untrusted.example')).toBe(DEFAULT_POST_LOGIN_PATH);
    expect(sanitizeRedirectPath('/auth/callback')).toBe(DEFAULT_POST_LOGIN_PATH);
  });
});
