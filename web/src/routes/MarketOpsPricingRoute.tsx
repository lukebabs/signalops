import { useMemo } from 'react';
import { useSearch } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import { useTenant } from '../auth/session';
import { useSubscription } from '../subscriber/SubscriptionContext';
import { BillingSupportPanel, featureName, productSort, SubscriptionPlanGrid } from '../subscriber/SubscriptionPlanCards';

export function MarketOpsPricingRoute() {
  const search = useSearch({ strict: false }) as { source_feature?: string; return_url?: string };
  const tenantId = useTenant();
  const subscription = useSubscription();
  const productsQ = useQuery({ queryKey: ['subscriber-subscription-products'], queryFn: api.listSubscriberSubscriptionProducts, staleTime: 60_000 });
  const products = useMemo(() => (productsQ.data?.products ?? []).slice().sort(productSort), [productsQ.data?.products]);
  const checkoutEnabled = Boolean(productsQ.data?.checkout_enabled);
  const sourceFeature = search.source_feature || '';

  return <div className="mx-auto max-w-6xl space-y-5">
    <section className="rounded border border-brand-100 bg-brand-50 p-5 dark:border-brand-900 dark:bg-brand-950/30">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="text-xs font-semibold uppercase tracking-wide text-brand-700 dark:text-brand-200">MarketOps subscription journey</p>
          <h1 className="mt-1 text-2xl font-semibold text-gray-950 dark:text-gray-50">Increase analytical depth when the research question requires it.</h1>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-gray-700 dark:text-gray-200">Explorer helps you discover what deserves attention. Professional adds the valuation, distress, earnings, options, and sector evidence needed to understand why. Institutional turns the workflow into a governed operating layer for teams.</p>
        </div>
        <div className="rounded border border-brand-200 bg-white px-3 py-2 text-xs text-gray-700 dark:border-brand-800 dark:bg-gray-950 dark:text-gray-200">
          Current access: <span className="font-semibold">{subscription.subscription?.display_name ?? 'Unprovisioned'}</span>
        </div>
      </div>
      {sourceFeature ? <p className="mt-3 rounded border border-brand-200 bg-white px-3 py-2 text-xs text-gray-700 dark:border-brand-800 dark:bg-gray-950 dark:text-gray-200">You arrived from a locked <span className="font-semibold">{featureName(sourceFeature)}</span> workflow. Choose the depth that answers the next research question.</p> : null}
    </section>

    {productsQ.isLoading ? <p className="text-sm text-gray-500">Loading configured plans…</p> : productsQ.isError ? <p className="rounded border border-red-200 bg-red-50 p-3 text-sm text-red-800">Plan configuration is unavailable.</p> : <SubscriptionPlanGrid products={products} currentProductKey={subscription.subscription?.product_key} checkoutEnabled={checkoutEnabled} tenantId={tenantId} />}

    <section className="rounded border border-gray-200 bg-white p-4 text-sm text-gray-700 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-200">
      <h2 className="text-sm font-semibold text-gray-950 dark:text-gray-50">Checkout status</h2>
      <p className="mt-1 text-xs leading-5 text-gray-600 dark:text-gray-300">Explorer and Professional use Stripe Checkout. MarketOps records an internal checkout reference first; access changes only after the signed Stripe webhook confirms the subscription. A return from Stripe alone never grants access.</p>{!checkoutEnabled ? <p className="mt-2 rounded border border-amber-300 bg-amber-50 px-3 py-2 text-xs text-amber-900 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-100">Checkout is not yet enabled because the Gateway is missing a Stripe API key. Billing mappings can still be reviewed by administrators.</p> : null}
      {search.return_url ? <p className="mt-2 text-xs text-gray-500 dark:text-gray-400">Trigger context retained for post-activation return: <code className="break-all">{search.return_url}</code></p> : null}
    </section>

    <BillingSupportPanel tenantId={tenantId} subscription={subscription.subscription} />
  </div>;
}
