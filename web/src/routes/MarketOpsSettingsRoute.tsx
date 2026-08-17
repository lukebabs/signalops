import { Link } from '@tanstack/react-router';
import { WatchlistControls } from './MarketOpsAssetsRoute';
import { useTenant } from '../auth/session';
import { useSubscription } from '../subscriber/SubscriptionContext';

export function MarketOpsSettingsRoute() {
  const tenant = useTenant();
  const { subscription, loading, known } = useSubscription();
  const limits = subscription?.limit_policy ?? {};
  return <div className="space-y-3">
    <div><h1 className="text-lg font-semibold">MarketOps Settings</h1><p className="text-xs text-gray-500">Manage your MarketOps workspace and analytical access. Changes to shared coverage remain centrally governed.</p></div>
    <section className="space-y-2 rounded border border-gray-200 bg-white p-3">
      <div><h2 className="text-sm font-semibold">Subscription and analytical depth</h2><p className="text-xs text-gray-500">Plans expand analytical depth; they do not create separate market-data copies or authorize browser provider polling.</p></div>
      {loading ? <p className="text-xs text-gray-500">Checking subscription…</p> : known && subscription ? <div className="grid gap-2 text-xs sm:grid-cols-3">
        <div><span className="text-gray-500">Plan</span><p className="font-medium">{subscription.display_name}</p></div>
        <div><span className="text-gray-500">Status</span><p className="font-medium capitalize">{subscription.status.replace('_', ' ')}</p></div>
        <div><span className="text-gray-500">Watchlists</span><p className="font-medium">{limits.private_watchlists === -1 ? 'Fair-use governed' : `${limits.private_watchlists ?? 0} private`} · {limits.assets_per_watchlist === -1 ? 'fair-use size' : `${limits.assets_per_watchlist ?? 0} assets/list`}</p></div>
      </div> : known ? <p className="text-xs text-amber-700">No MarketOps subscription is provisioned for this account yet. Explorer, Professional, and Institutional provisioning are managed before commercial enforcement is enabled.</p> : <p className="text-xs text-gray-500">Subscription controls are not enabled in this environment.</p>}
    </section>
    <section className="space-y-2 rounded border border-gray-200 bg-white p-3"><div><h2 className="text-sm font-semibold">Analyst assets</h2><p className="text-xs text-gray-500">Validate and add symbols to the analyst watchlist. Optional history backfill starts after onboarding.</p></div><WatchlistControls tenantId={tenant} onChanged={() => undefined}/></section>
    <section className="space-y-2 rounded border border-gray-200 bg-white p-3"><div><h2 className="text-sm font-semibold">Research validation</h2><p className="text-xs text-gray-500">Back-tests evaluate deterministic research logic and calibration evidence. They are administrative tooling, not an analyst workflow.</p></div><Link to="/marketops/backtests" className="inline-flex rounded border border-brand-600 px-3 py-1.5 text-xs font-medium text-brand-700 hover:bg-brand-50">Open Back-Tests</Link></section>
  </div>;
}
