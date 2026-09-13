import { useEffect } from 'react';
import { Link, useSearch } from '@tanstack/react-router';
import { useQueryClient } from '@tanstack/react-query';
import { CheckCircle2, Clock3, RotateCw } from 'lucide-react';
import { useTenant } from '../auth/session';
import { useSubscriberSubscription } from '../api/queries';

export function MarketOpsSubscriptionReturnRoute() {
  const search = useSearch({ strict: false }) as { session_id?: string };
  const tenantId = useTenant();
  const queryClient = useQueryClient();
  const subscriptionQ = useSubscriberSubscription(tenantId, { refetchIntervalMs: 5_000 });
  const subscription = subscriptionQ.data?.subscription ?? null;
  const active = Boolean(subscription && ['trialing', 'active', 'past_due'].includes(subscription.status));

  useEffect(() => {
    void queryClient.invalidateQueries({ queryKey: ['subscriber-subscription', tenantId] });
  }, [queryClient, tenantId, search.session_id]);

  return <section className="mx-auto max-w-3xl space-y-4">
    <div className="rounded border border-brand-100 bg-brand-50 p-5 dark:border-brand-900 dark:bg-brand-950/30">
      <p className="text-xs font-semibold uppercase tracking-wide text-brand-700 dark:text-brand-200">MarketOps subscription</p>
      <h1 className="mt-1 text-2xl font-semibold text-gray-950 dark:text-gray-50">{active ? 'Subscription active' : 'Payment received; activation pending'}</h1>
      <p className="mt-2 text-sm leading-6 text-gray-700 dark:text-gray-200">{active ? `Your ${subscription?.display_name ?? 'MarketOps'} access is active. You can return to MarketOps and continue analysis.` : 'Stripe redirected you back to MarketOps. Access changes only after the signed Stripe webhook confirms the subscription, so this page will refresh briefly while activation lands.'}</p>
    </div>

    <div className="rounded border border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-900">
      <div className="flex items-start gap-3">
        {active ? <CheckCircle2 className="mt-0.5 text-emerald-600" size={20} /> : <Clock3 className="mt-0.5 text-amber-600" size={20} />}
        <div className="min-w-0 flex-1">
          <h2 className="text-sm font-semibold text-gray-950 dark:text-gray-50">Webhook-authoritative activation</h2>
          <p className="mt-1 text-xs leading-5 text-gray-600 dark:text-gray-300">Session ID: <code className="break-all">{search.session_id || 'not supplied'}</code></p>
          <p className="mt-1 text-xs leading-5 text-gray-600 dark:text-gray-300">Current state: {subscriptionQ.isFetching ? 'checking subscription state…' : active ? `${subscription?.display_name} · ${subscription?.status}` : 'not active yet'}</p>
        </div>
      </div>
      <div className="mt-4 flex flex-wrap gap-2">
        <button type="button" onClick={() => subscriptionQ.refetch()} className="inline-flex items-center gap-2 rounded border border-gray-300 px-3 py-1.5 text-sm text-gray-800 dark:border-gray-600 dark:text-gray-100"><RotateCw size={15} /> Check again</button>
        <Link to="/marketops/dashboard" className="rounded bg-gray-900 px-3 py-1.5 text-sm font-medium text-white dark:bg-gray-100 dark:text-gray-900">Return to dashboard</Link>
        <Link to="/marketops/pricing" className="rounded border border-gray-300 px-3 py-1.5 text-sm text-gray-800 dark:border-gray-600 dark:text-gray-100">Back to pricing</Link>
      </div>
    </div>
  </section>;
}
