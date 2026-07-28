import { describe, expect, it } from 'vitest';
import {
  appIdFromPathname,
  metadataFilterForApp,
  defaultRouteForApp,
  navForApp,
} from './appRouting';
import { CONSOLE_PROFILE, CYBEROPS_PROFILE, MARKETOPS_PROFILE } from './appProfiles';

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

  it('treats /cyberops and /cyberops/* as cyberops', () => {
    expect(appIdFromPathname('/cyberops')).toBe('cyberops');
    expect(appIdFromPathname('/cyberops/')).toBe('cyberops');
    expect(appIdFromPathname('/cyberops/signals')).toBe('cyberops');
    expect(appIdFromPathname('/cyberops/alerts')).toBe('cyberops');
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

  it('scopes cyberops to its security signals without forcing use_case', () => {
    expect(metadataFilterForApp('cyberops')).toEqual({ app_id: 'cyberops', domain: 'security' });
    expect(metadataFilterForApp('cyberops')).not.toHaveProperty('use_case');
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

  it('opens CyberOps directly on its scoped signals workflow', () => {
    expect(defaultRouteForApp(CYBEROPS_PROFILE)).toBe('/cyberops/signals');
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

  it('keeps MarketOps navigation focused on analyst workbenches', () => {
    const nav = navForApp('marketops');
    const labels = nav.map((n) => n.label);
    expect(labels).not.toContain('Providers');
    expect(labels).not.toContain('Health');
    expect(labels).not.toContain('Algorithms');
    // Every marketops nav entry targets a /marketops/* route.
    expect(nav.every((n) => n.to.startsWith('/marketops/'))).toBe(true);
  });

  it('does not include idempotency, runs, or rules in the marketops nav', () => {
    const labels = navForApp('marketops').map((n) => n.label);
    expect(labels).not.toContain('Idempotency');
    expect(labels).not.toContain('Runs');
    expect(labels).not.toContain('Rules');
  });

  it('keeps CyberOps focused on its signal triage workflow', () => {
    const nav = navForApp('cyberops');
    expect(nav.map((item) => item.label)).toEqual(['Signals', 'Alerts', 'Insights']);
    expect(nav.every((item) => item.to.startsWith('/cyberops/'))).toBe(true);
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

  it('keeps Market State as a direct drill-down route, not visible navigation', () => {
    expect(navForApp('marketops').some((n) => n.module === 'market_state')).toBe(false);
    expect(navForApp('console').some((n) => n.to === '/marketops/state')).toBe(false);
  });

  it('exposes the MarketOps opportunities workbench route only under marketops (G139)', () => {
    const opportunities = navForApp('marketops').find((n) => n.module === 'opportunities');
    expect(opportunities).toBeDefined();
    expect(opportunities?.to).toBe('/marketops/opportunities');
    expect(opportunities?.label).toBe('Opportunities');
    // It sits near Assets and DSM.
    const marketopsNav = navForApp('marketops');
    const dsmIndex = marketopsNav.findIndex((n) => n.module === 'dsm');
    const opportunitiesIndex = marketopsNav.findIndex((n) => n.module === 'opportunities');
    expect(opportunitiesIndex).toBeGreaterThan(-1);
    expect(Math.abs(opportunitiesIndex - dsmIndex)).toBeLessThanOrEqual(1);
    // Console parity: Opportunities does not leak into console nav.
    expect(navForApp('console').some((n) => n.to === '/marketops/opportunities')).toBe(false);
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

  it('exposes algorithm management only in Administration', () => {
    expect(navForApp('marketops').some((n) => n.module === 'algorithms')).toBe(false);
    const algorithms = navForApp('console').find((n) => n.module === 'algorithms');
    expect(algorithms?.to).toBe('/admin/algorithms');
  });
});
