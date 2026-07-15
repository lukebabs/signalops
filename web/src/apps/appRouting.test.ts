import { describe, expect, it } from 'vitest';
import {
  appIdFromPathname,
  metadataFilterForApp,
  defaultRouteForApp,
  navForApp,
} from './appRouting';
import { CONSOLE_PROFILE, MARKETOPS_PROFILE } from './appProfiles';

describe('appIdFromPathname (G067)', () => {
  it('treats the index and all existing console routes as console', () => {
    expect(appIdFromPathname('/')).toBe('console');
    expect(appIdFromPathname('/dashboard')).toBe('console');
    expect(appIdFromPathname('/raw-events')).toBe('console');
    expect(appIdFromPathname('/signals')).toBe('console');
    expect(appIdFromPathname('/marketops-dashboard')).toBe('console'); // not a marketops prefix
  });

  it('treats /marketops and /marketops/* as marketops', () => {
    expect(appIdFromPathname('/marketops')).toBe('marketops');
    expect(appIdFromPathname('/marketops/')).toBe('marketops');
    expect(appIdFromPathname('/marketops/dashboard')).toBe('marketops');
    expect(appIdFromPathname('/marketops/signals')).toBe('marketops');
    expect(appIdFromPathname('/marketops/dsm')).toBe('marketops');
  });

  it('defaults an empty path to console', () => {
    expect(appIdFromPathname('')).toBe('console');
  });
});

describe('metadataFilterForApp (G067)', () => {
  it('returns an empty filter for console (unscoped)', () => {
    expect(metadataFilterForApp('console')).toEqual({});
  });

  it('scopes marketops to app_id + domain without forcing use_case', () => {
    expect(metadataFilterForApp('marketops')).toEqual({
      app_id: 'marketops',
      domain: 'market_data',
    });
  });

  it('never injects use_case globally for marketops', () => {
    expect(metadataFilterForApp('marketops')).not.toHaveProperty('use_case');
  });
});

describe('defaultRouteForApp (G067)', () => {
  it('lands console on the frontend index, not the backend /dashboard label', () => {
    expect(defaultRouteForApp(CONSOLE_PROFILE)).toBe('/');
  });

  it('uses the profile default_route for non-console apps', () => {
    expect(defaultRouteForApp(MARKETOPS_PROFILE)).toBe(MARKETOPS_PROFILE.default_route);
    expect(defaultRouteForApp(MARKETOPS_PROFILE)).toBe('/marketops/dashboard');
  });
});

describe('navForApp (G067)', () => {
  it('keeps the full console nav, including Sources and System', () => {
    const labels = navForApp('console').map((n) => n.label);
    expect(labels).toContain('Sources');
    expect(labels).toContain('System');
    expect(labels).not.toContain('Providers');
    expect(labels).not.toContain('Health');
  });

  it('uses market-facing labels for marketops (Providers, Health) and marketops routes', () => {
    const nav = navForApp('marketops');
    const labels = nav.map((n) => n.label);
    expect(labels).toContain('Providers');
    expect(labels).toContain('Health');
    expect(labels).not.toContain('Sources');
    expect(labels).not.toContain('System');
    // Every marketops nav entry targets a /marketops/* route.
    expect(nav.every((n) => n.to.startsWith('/marketops/'))).toBe(true);
  });

  it('does not include idempotency, runs, or rules in the marketops nav', () => {
    const labels = navForApp('marketops').map((n) => n.label);
    expect(labels).not.toContain('Idempotency');
    expect(labels).not.toContain('Runs');
    expect(labels).not.toContain('Rules');
  });

  it('exposes the MarketOps asset universe route only under marketops (G071)', () => {
    const assets = navForApp('marketops').find((n) => n.module === 'symbols');
    expect(assets).toBeDefined();
    expect(assets?.to).toBe('/marketops/assets');
    expect(assets?.label).toBe('Assets');
    // Console parity: no asset/symbol nav leaks into the console.
    expect(navForApp('console').some((n) => n.to === '/marketops/assets')).toBe(false);
  });

  it('exposes the MarketOps DSM workbench route only under marketops (G076)', () => {
    const dsm = navForApp('marketops').find((n) => n.module === 'dsm');
    expect(dsm).toBeDefined();
    expect(dsm?.to).toBe('/marketops/dsm');
    expect(dsm?.label).toBe('DSM');
    // Generic MarketOps routes are still present alongside the new DSM entry.
    const labels = navForApp('marketops').map((n) => n.label);
    expect(labels).toContain('Signals');
    expect(labels).toContain('Assets');
    // Console parity: DSM does not leak into console nav.
    expect(navForApp('console').some((n) => n.to === '/marketops/dsm')).toBe(false);
  });

  it('exposes the MarketOps back-tests route only under marketops (G081)', () => {
    const backtests = navForApp('marketops').find((n) => n.module === 'backtests');
    expect(backtests).toBeDefined();
    expect(backtests?.to).toBe('/marketops/backtests');
    expect(backtests?.label).toBe('Back-Tests');
    // It sits near DSM (the spec allows near DSM or Replay).
    const marketopsNav = navForApp('marketops');
    const dsmIndex = marketopsNav.findIndex((n) => n.module === 'dsm');
    const backtestsIndex = marketopsNav.findIndex((n) => n.module === 'backtests');
    expect(dsmIndex).toBeGreaterThanOrEqual(0);
    expect(backtestsIndex).toBeGreaterThan(dsmIndex);
    // Console parity: back-tests never appear in console nav.
    expect(navForApp('console').some((n) => n.to === '/marketops/backtests')).toBe(false);
    expect(navForApp('console').map((n) => n.label)).not.toContain('Back-Tests');
  });

  it('exposes the MarketOps algorithms route only under marketops (G109)', () => {
    const algorithms = navForApp('marketops').find((n) => n.module === 'algorithms');
    expect(algorithms).toBeDefined();
    expect(algorithms?.to).toBe('/marketops/algorithms');
    expect(algorithms?.label).toBe('Algorithms');
    // Sits in the operator/evidence area alongside Syncratic.
    const marketopsNav = navForApp('marketops');
    const syncraticIndex = marketopsNav.findIndex((n) => n.module === 'syncratic');
    const algorithmsIndex = marketopsNav.findIndex((n) => n.module === 'algorithms');
    expect(syncraticIndex).toBeGreaterThanOrEqual(0);
    expect(algorithmsIndex).toBeGreaterThan(syncraticIndex);
    // Console parity: algorithms never appear in console nav.
    expect(navForApp('console').some((n) => n.to === '/marketops/algorithms')).toBe(false);
    expect(navForApp('console').map((n) => n.label)).not.toContain('Algorithms');
  });
});
