// Pure app-routing helpers. No React, no DOM — fully unit-testable.
import type { AppProfile } from '../types';

// All frontend route paths used by app nav/selector. Must stay a subset of the
// routes registered in router.tsx so TanStack Router's typed <Link to=... />
// accepts them.
export type AppRoutePath =
  | '/admin/dashboard'
  | '/admin/runs'
  | '/admin/raw-events'
  | '/admin/normalized-events'
  | '/admin/idempotency'
  | '/admin/sources'
  | '/admin/pipelines'
  | '/admin/rules'
  | '/admin/replay'
  | '/admin/signals'
  | '/admin/alerts'
  | '/admin/insights'
  | '/admin/algorithms'
  | '/admin/system'
  | '/'
  | '/runs'
  | '/raw-events'
  | '/normalized-events'
  | '/idempotency'
  | '/sources'
  | '/pipelines'
  | '/rules'
  | '/replay'
  | '/signals'
  | '/alerts'
  | '/insights'
  | '/system'
  | '/marketops/dashboard'
  | '/marketops/providers'
  | '/marketops/raw-events'
  | '/marketops/normalized'
  | '/marketops/signals'
  | '/marketops/alerts'
  | '/marketops/insights'
  | '/marketops/replay'
  | '/marketops/pipelines'
  | '/marketops/health'
  | '/marketops/assets'
  | '/marketops/state'
  | '/marketops/dsm'
  | '/marketops/opportunities'
  | '/marketops/backtests'
  | '/marketops/syncratic'
  | '/marketops/algorithms';

export type MetadataFilter = { app_id?: string; domain?: string; use_case?: string };

export interface NavItem {
  module: string;
  to: AppRoutePath;
  label: string;
}

// Detect the active app from the route prefix. /marketops and /marketops/* -> marketops;
// every other path (including "/" and all existing console routes) -> console.
export function appIdFromPathname(pathname: string): string {
  const p = pathname || '/';
  if (p === '/marketops' || p.startsWith('/marketops/')) return 'marketops';
  return 'console';
}

// Metadata filter applied to G066-aware list APIs (raw/normalized/signals/
// alerts/insights). Console is unscoped; MarketOps is scoped to
// app_id=marketops, domain=market_data. use_case is not forced globally.
export function metadataFilterForApp(appId: string): MetadataFilter {
  if (appId === 'marketops') return { app_id: 'marketops', domain: 'market_data' };
  return {};
}

// Resolve the route to land on when selecting an app. Console uses the frontend
// index "/"; other apps use the profile's default_route.
export function defaultRouteForApp(profile: AppProfile): AppRoutePath {
  if (profile.app_id === 'console') return '/';
  return profile.default_route as AppRoutePath;
}

const CONSOLE_NAV: NavItem[] = [
  { module: 'dashboard', to: '/admin/dashboard', label: 'Dashboard' }, { module: 'runs', to: '/admin/runs', label: 'Runs' }, { module: 'raw_events', to: '/admin/raw-events', label: 'Raw Events' }, { module: 'normalized', to: '/admin/normalized-events', label: 'Normalized' }, { module: 'idempotency', to: '/admin/idempotency', label: 'Idempotency' }, { module: 'sources', to: '/admin/sources', label: 'Sources' }, { module: 'pipelines', to: '/admin/pipelines', label: 'Pipelines' }, { module: 'rules', to: '/admin/rules', label: 'Rules' }, { module: 'replay', to: '/admin/replay', label: 'Replay' }, { module: 'signals', to: '/admin/signals', label: 'Signals' }, { module: 'alerts', to: '/admin/alerts', label: 'Alerts' }, { module: 'insights', to: '/admin/insights', label: 'Insights' }, { module: 'algorithms', to: '/admin/algorithms', label: 'Algorithms' }, { module: 'health', to: '/admin/system', label: 'System' },
];

const MARKETOPS_NAV: NavItem[] = [
  { module: 'dashboard', to: '/marketops/dashboard', label: 'Dashboard' },
  { module: 'symbols', to: '/marketops/assets', label: 'Assets' },
  { module: 'market_state', to: '/marketops/state', label: 'Market State' },
  { module: 'signals', to: '/marketops/signals', label: 'Signals' },
  { module: 'dsm', to: '/marketops/dsm', label: 'DSM' },
  { module: 'opportunities', to: '/marketops/opportunities', label: 'Opportunities' },
  { module: 'backtests', to: '/marketops/backtests', label: 'Back-Tests' },
  { module: 'syncratic', to: '/marketops/syncratic', label: 'Syncratic Insights' },
  { module: 'alerts', to: '/marketops/alerts', label: 'Alerts' },
  { module: 'insights', to: '/marketops/insights', label: 'Insights' },
];

// Nav is an explicit per-app route set matching the G067 Required Outcome +
// Routing Work. The backend profile's enabled_modules is consumed for the app
// selector/labels but does not fully cover the desired nav (it omits several
// existing console routes and several MarketOps route aliases), and removing/
// renaming existing console routes is a non-goal — so nav is not gated by
// enabled_modules.
export function navForApp(appId: string): NavItem[] {
  return appId === 'marketops' ? MARKETOPS_NAV : CONSOLE_NAV;
}
