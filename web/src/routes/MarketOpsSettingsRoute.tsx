import { Link } from '@tanstack/react-router';
import { WatchlistControls } from './MarketOpsAssetsRoute';
import { useTenant } from '../auth/session';

export function MarketOpsSettingsRoute() {
  const tenant = useTenant();
  return <div className="space-y-3">
    <div><h1 className="text-lg font-semibold">MarketOps Settings</h1><p className="text-xs text-gray-500">Manage your MarketOps workspace. Shared catalog coverage and analytical subscriptions are centrally governed.</p></div>
    <section className="space-y-2 rounded border border-gray-200 bg-white p-3"><div><h2 className="text-sm font-semibold">Analyst assets</h2><p className="text-xs text-gray-500">Validate and add symbols to the analyst watchlist. Optional history backfill starts after onboarding.</p></div><WatchlistControls tenantId={tenant} onChanged={() => undefined}/></section>
    <section className="space-y-2 rounded border border-gray-200 bg-white p-3"><div><h2 className="text-sm font-semibold">Research validation</h2><p className="text-xs text-gray-500">Back-tests evaluate deterministic research logic and calibration evidence. They are administrative tooling, not an analyst workflow.</p></div><Link to="/marketops/backtests" className="inline-flex rounded border border-brand-600 px-3 py-1.5 text-xs font-medium text-brand-700 hover:bg-brand-50">Open Back-Tests</Link></section>
  </div>;
}
