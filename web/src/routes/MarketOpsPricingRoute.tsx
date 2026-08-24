import { useMemo } from 'react';
import { useSearch } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { CheckCircle2, CircleDollarSign, Mail } from 'lucide-react';
import { api } from '../api/client';
import { useSubscription } from '../subscriber/SubscriptionContext';
import type { SubscriberSubscriptionProduct } from '../types';

const tierPositioning: Record<string, { headline: string; description: string; bullets: string[] }> = {
  explorer: {
    headline: 'Discover what deserves your attention.',
    description: 'Built for structured daily market discovery without professional research workflow complexity.',
    bullets: ['Market dashboards', 'Public signals', 'Sector rotation discovery', 'Limited watchlists'],
  },
  professional: {
    headline: 'Understand why opportunities matter.',
    description: 'Built for serious investors, analysts, and advisors who need deeper analytical context.',
    bullets: ['Value Intelligence', 'Distressed Opportunity Intelligence', 'Earnings Opportunity Intelligence', 'Options signals', 'Detailed Sector Rotation', 'Research workflows'],
  },
  institutional: {
    headline: 'Operationalize investment intelligence at scale.',
    description: 'Designed for investment teams that need governance, automation, portfolio-scale workflows, and integration.',
    bullets: ['Signal Assurance analytics', 'Portfolio analysis', 'Batch screening', 'Historical replay', 'Custom universes', 'APIs and tenant controls'],
  },
};

export function MarketOpsPricingRoute() {
  const search = useSearch({ strict: false }) as { source_feature?: string; return_url?: string };
  const subscription = useSubscription();
  const productsQ = useQuery({ queryKey: ['subscriber-subscription-products'], queryFn: api.listSubscriberSubscriptionProducts, staleTime: 60_000 });
  const products = useMemo(() => (productsQ.data?.products ?? []).slice().sort(productSort), [productsQ.data?.products]);
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

    {productsQ.isLoading ? <p className="text-sm text-gray-500">Loading configured plans…</p> : productsQ.isError ? <p className="rounded border border-red-200 bg-red-50 p-3 text-sm text-red-800">Plan configuration is unavailable.</p> : <section className="grid gap-4 lg:grid-cols-3">{products.map((product) => <PlanCard key={product.product_key} product={product} current={subscription.subscription?.product_key === product.product_key} />)}</section>}

    <section className="rounded border border-gray-200 bg-white p-4 text-sm text-gray-700 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-200">
      <h2 className="text-sm font-semibold text-gray-950 dark:text-gray-50">Checkout status</h2>
      <p className="mt-1 text-xs leading-5 text-gray-600 dark:text-gray-300">Self-service Stripe Checkout is intentionally not enabled in this release. This page provides pricing context and records upgrade intent. Access changes still require the governed Subscription Administration or a future webhook-confirmed Checkout release.</p>
      {search.return_url ? <p className="mt-2 text-xs text-gray-500 dark:text-gray-400">Trigger context retained for future Checkout return: <code className="break-all">{search.return_url}</code></p> : null}
    </section>
  </div>;
}

function PlanCard({ product, current }: { product: SubscriberSubscriptionProduct; current: boolean }) {
  const info = tierPositioning[product.product_key] ?? { headline: product.display_name, description: '', bullets: [] };
  const institutional = product.product_key === 'institutional';
  return <article className={`rounded border bg-white p-4 shadow-sm dark:bg-gray-900 ${current ? 'border-brand-400 ring-1 ring-brand-300 dark:border-brand-500' : 'border-gray-200 dark:border-gray-700'}`}>
    <div className="flex items-start justify-between gap-2">
      <div>
        <h2 className="text-lg font-semibold text-gray-950 dark:text-gray-50">{product.display_name}</h2>
        <p className="mt-1 text-sm font-medium text-brand-700 dark:text-brand-200">{info.headline}</p>
      </div>
      {current ? <span className="rounded bg-brand-100 px-2 py-0.5 text-xs font-medium text-brand-800 dark:bg-brand-950 dark:text-brand-100">Current</span> : null}
    </div>
    <p className="mt-3 min-h-12 text-xs leading-5 text-gray-600 dark:text-gray-300">{info.description}</p>
    <div className="mt-3 rounded bg-gray-50 p-3 dark:bg-gray-950">
      {institutional ? <div className="flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-gray-100"><Mail size={16} /> Contact Sales</div> : <div className="space-y-1"><PriceLine label="Monthly" value={shortStripePrice(product.stripe_monthly_price_id)} /><PriceLine label="Annual" value={shortStripePrice(product.stripe_annual_price_id)} /></div>}
    </div>
    <ul className="mt-3 space-y-2 text-xs text-gray-700 dark:text-gray-200">{info.bullets.map((bullet) => <li key={bullet} className="flex gap-2"><CheckCircle2 size={14} className="mt-0.5 shrink-0 text-emerald-600" />{bullet}</li>)}</ul>
    <div className="mt-4">
      <button type="button" disabled className="inline-flex items-center gap-2 rounded bg-gray-900 px-3 py-1.5 text-sm font-medium text-white opacity-60 disabled:cursor-not-allowed dark:bg-gray-100 dark:text-gray-900"><CircleDollarSign size={15} /> {institutional ? 'Contact Sales coming soon' : 'Checkout coming soon'}</button>
    </div>
  </article>;
}

function PriceLine({ label, value }: { label: string; value: string }) {
  return <div className="flex items-center justify-between gap-3 text-sm"><span className="text-gray-500 dark:text-gray-400">{label}</span><code className="text-xs text-gray-900 dark:text-gray-100">{value}</code></div>;
}

function shortStripePrice(value: string | undefined): string {
  return value ? value : 'Configured in Stripe/Admin';
}

function featureName(value: string): string {
  return value.replace(/_/g, ' ').replace(/\b\w/g, (match) => match.toUpperCase());
}

function productSort(a: SubscriberSubscriptionProduct, b: SubscriberSubscriptionProduct): number {
  const order: Record<string, number> = { explorer: 1, professional: 2, institutional: 3 };
  return (order[a.product_key] ?? 99) - (order[b.product_key] ?? 99);
}
