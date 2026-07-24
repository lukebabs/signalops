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
  ],
  dashboard_profile: 'console.default',
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
};
