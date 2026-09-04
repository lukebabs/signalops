import { Link } from '@tanstack/react-router';
import { WatchlistControls } from './MarketOpsAssetsRoute';
import { hasPlatformAdmin, rolesFromClaims } from '../auth/claims';
import { useAuth, useTenant } from '../auth/session';

export function MarketOpsToolsRoute() {
  const tenant = useTenant();
  const { claims } = useAuth();
  const roles = rolesFromClaims(claims);
  const allowed = hasPlatformAdmin(claims) || roles.includes('signalops:operator');
  if (!allowed) {
    return <div className="mx-auto max-w-2xl rounded border border-amber-300 bg-amber-50 p-5 text-sm text-amber-900 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-100">
      <h1 className="text-lg font-semibold">MarketOps tools are restricted</h1>
      <p className="mt-2">These operational controls are available only to MarketOps operators and administrators. User preferences and subscription actions are available in Settings.</p>
      <Link to="/marketops/settings" className="mt-4 inline-flex rounded border border-amber-400 px-3 py-1.5 text-sm font-medium hover:bg-amber-100 dark:border-amber-700 dark:hover:bg-amber-900">Open user settings</Link>
    </div>;
  }
  return <div className="space-y-3">
    <div><h1 className="text-lg font-semibold text-gray-950 dark:text-gray-50">MarketOps Tools</h1><p className="text-xs text-gray-500 dark:text-gray-400">Operational controls moved out of user Settings. These tools are for privileged workflows only.</p></div>
    <section className="space-y-2 rounded border border-gray-200 bg-white p-3 dark:border-gray-700 dark:bg-gray-900"><div><h2 className="text-sm font-semibold text-gray-950 dark:text-gray-50">Analyst assets</h2><p className="text-xs text-gray-500 dark:text-gray-400">Validate and add symbols to the analyst watchlist. Optional history backfill starts after onboarding.</p></div><WatchlistControls tenantId={tenant} onChanged={() => undefined}/></section>
    <section className="space-y-2 rounded border border-gray-200 bg-white p-3 dark:border-gray-700 dark:bg-gray-900"><div><h2 className="text-sm font-semibold text-gray-950 dark:text-gray-50">Research validation</h2><p className="text-xs text-gray-500 dark:text-gray-400">Back-tests evaluate deterministic research logic and calibration evidence. They are operational tooling, not a default analyst workflow.</p></div><Link to="/marketops/backtests" className="inline-flex rounded border border-brand-600 px-3 py-1.5 text-xs font-medium text-brand-700 hover:bg-brand-50 dark:text-brand-200 dark:hover:bg-brand-950">Open Back-Tests</Link></section>
  </div>;
}
