import { describe, expect, it } from 'vitest';
import { resolveAuthConfig } from './config';

describe('auth config', () => {
  it('defaults to auth disabled with the Syncratic IdP defaults', () => {
    const c = resolveAuthConfig({});
    expect(c.authEnabled).toBe(false);
    expect(c.issuer).toBe('https://auth.syncratic.co/realms/syncratic');
    expect(c.clientId).toBe('signalops-web');
    expect(c.audience).toBe('signalops-api');
    expect(c.realm).toBe('syncratic');
  });

  it('enables auth only for the literal "true" (case-insensitive)', () => {
    expect(resolveAuthConfig({ VITE_SIGNALOPS_AUTH_ENABLED: 'true' }).authEnabled).toBe(true);
    expect(resolveAuthConfig({ VITE_SIGNALOPS_AUTH_ENABLED: 'TRUE' }).authEnabled).toBe(true);
    expect(resolveAuthConfig({ VITE_SIGNALOPS_AUTH_ENABLED: 'false' }).authEnabled).toBe(false);
    expect(resolveAuthConfig({ VITE_SIGNALOPS_AUTH_ENABLED: '1' }).authEnabled).toBe(false);
    expect(resolveAuthConfig({ VITE_SIGNALOPS_AUTH_ENABLED: '' }).authEnabled).toBe(false);
  });

  it('honors explicit overrides', () => {
    const c = resolveAuthConfig({
      VITE_SIGNALOPS_AUTH_ENABLED: 'true',
      VITE_SIGNALOPS_AUTH_ISSUER: 'https://example/realms/x',
      VITE_SIGNALOPS_AUTH_CLIENT_ID: 'cid',
      VITE_SIGNALOPS_AUTH_AUDIENCE: 'aud',
      VITE_SIGNALOPS_AUTH_REALM: 'x',
    });
    expect(c).toEqual({
      authEnabled: true,
      issuer: 'https://example/realms/x',
      clientId: 'cid',
      audience: 'aud',
      realm: 'x',
    });
  });
});
