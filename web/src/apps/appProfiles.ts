import type { AppProfile } from '../types';

// Static fallback used when GET /v1/app-profiles fails (e.g. transient 401 or
// network error), so the existing SignalOps Console UI remains fully usable
// without the backend profile list. default_route is the frontend console
// index, not the backend's "/dashboard" label.
export const CONSOLE_PROFILE: AppProfile = {
  app_id: 'console',
  label: 'Administration',
  default_route: '/admin/dashboard',
  domains: ['market_data', 'crm', 'security', 'operations', 'iot', 'procurement', 'custom'],
  enabled_modules: [
    'dashboard',
    'runs',
    'raw_events',
    'normalized',
    'idempotency',
    'sources',
    'pipelines',
    'rules',
    'replay',
    'signals',
    'alerts',
    'insights',
    'health',
    'storage',
  ],
  dashboard_profile: 'console.default',
  landing_summary: 'Platform administration and governed operations.',
  route_prefix: '/admin',
};

// MarketOps fallback mirrors the backend's marketops profile. Pairs with
// CONSOLE_PROFILE so /marketops/* routes scope to app_id=marketops even if the
// GET /v1/app-profiles request has not resolved yet (or failed). The
// default_route matches the registered frontend route, not a backend label.
export const MARKETOPS_PROFILE: AppProfile = {
  app_id: 'marketops',
  label: 'MarketOps',
  default_route: '/marketops/dashboard',
  domains: ['market_data'],
  enabled_modules: [
    'dashboard',
    'symbols',
    'option_contracts',
    'signals',
    'alerts',
    'replay',
    'providers',
    'pipelines',
    'health',
  ],
  dashboard_profile: 'marketdata.default',
  landing_summary: 'Strategic financial context and daily market evidence for disciplined analyst review.',
  route_prefix: '/marketops',
};

// CyberOps reuses the generic SignalOps investigation workflow. Keeping this
// profile locally means the app selector can expose CyberOps while a gateway
// rollout is still serving an older /v1/app-profiles response.
export const CYBEROPS_PROFILE: AppProfile = {
  app_id: "cyberops",
  label: "CyberOps",
  default_route: "/cyberops/dashboard",
  domains: ["security"],
  enabled_modules: ["dashboard", "anomalies", "signals", "alerts", "insights", "settings"],
  dashboard_profile: "security.default",
  landing_summary: "Firewall evidence, deterministic detections, and focused security investigation.",
  route_prefix: "/cyberops",
};
