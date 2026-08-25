import { useMemo, useState } from 'react';
import { useSearch } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { CheckCircle2, CircleDollarSign, Mail } from 'lucide-react';
import { api } from '../api/client';
import { useTenant } from '../auth/session';
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

type BillingPeriod = 'monthly' | 'annual';

export function MarketOpsPricingRoute() {
  const search = useSearch({ strict: false }) as { source_feature?: string; return_url?: string };
  const tenantId = useTenant();
  const subscription = useSubscription();
  const productsQ = useQuery({ queryKey: ['subscriber-subscription-products'], queryFn: api.listSubscriberSubscriptionProducts, staleTime: 60_000 });
  const products = useMemo(() => (productsQ.data?.products ?? []).slice().sort(productSort), [productsQ.data?.products]);
  const sourceFeature = search.source_feature || '';
  const [workingKey, setWorkingKey] = useState<string>('');
  const [checkoutError, setCheckoutError] = useState<string>('');

  async function startCheckout(product: SubscriberSubscriptionProduct, billingPeriod: BillingPeriod) {
    setCheckoutError('');
    setWorkingKey(`${product.product_key}:${billingPeriod}`);
    try {
      const response = await api.createSubscriberCheckoutSession(tenantId, { product_key: product.product_key, billing_period: billingPeriod });
      window.location.assign(response.checkout_url);
    } catch (error) {
      setCheckoutError(error instanceof Error ? error.message : 'Checkout could not be started.');
      setWorkingKey('');
    }
  }

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

    {checkoutError ? <section className="rounded border border-amber-300 bg-amber-50 p-3 text-sm text-amber-900 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-100"><span className="font-semibold">Checkout unavailable.</span> {checkoutError}</section> : null}

    {productsQ.isLoading ? <p className="text-sm text-gray-500">Loading configured plans…</p> : productsQ.isError ? <p className="rounded border border-red-200 bg-red-50 p-3 text-sm text-red-800">Plan configuration is unavailable.</p> : <section className="grid gap-4 lg:grid-cols-3">{products.map((product) => <PlanCard key={product.product_key} product={product} current={subscription.subscription?.product_key === product.product_key} workingKey={workingKey} onCheckout={startCheckout} />)}</section>}

    <section className="rounded border border-gray-200 bg-white p-4 text-sm text-gray-700 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-200">
      <h2 className="text-sm font-semibold text-gray-950 dark:text-gray-50">Checkout status</h2>
      <p className="mt-1 text-xs leading-5 text-gray-600 dark:text-gray-300">Explorer and Professional use Stripe Checkout. MarketOps records an internal checkout reference first; access changes only after the signed Stripe webhook confirms the subscription. A return from Stripe alone never grants access.</p>
      {search.return_url ? <p className="mt-2 text-xs text-gray-500 dark:text-gray-400">Trigger context retained for post-activation return: <code className="break-all">{search.return_url}</code></p> : null}
    </section>
  </div>;
}

function PlanCard({ product, current, workingKey, onCheckout }: { product: SubscriberSubscriptionProduct; current: boolean; workingKey: string; onCheckout: (product: SubscriberSubscriptionProduct, billingPeriod: BillingPeriod) => void }) {
  const info = tierPositioning[product.product_key] ?? { headline: product.display_name, description: '', bullets: [] };
  const institutional = product.product_key === 'institutional';
  const selfService = product.product_key === 'explorer' || product.product_key === 'professional';
  const monthlyMapped = Boolean(product.stripe_monthly_price_id);
  const annualMapped = Boolean(product.stripe_annual_price_id);
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
    <div className="mt-4 flex flex-wrap gap-2">
      {selfService ? <>
        <button type="button" disabled={!monthlyMapped || Boolean(workingKey)} onClick={() => onCheckout(product, 'monthly')} className="inline-flex items-center gap-2 rounded bg-gray-900 px-3 py-1.5 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-50 dark:bg-gray-100 dark:text-gray-900"><CircleDollarSign size={15} /> {workingKey === `${product.product_key}:monthly` ? 'Opening…' : 'Monthly Checkout'}</button>
        <button type="button" disabled={!annualMapped || Boolean(workingKey)} onClick={() => onCheckout(product, 'annual')} className="inline-flex items-center gap-2 rounded border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-800 disabled:cursor-not-allowed disabled:opacity-50 dark:border-gray-600 dark:text-gray-100"><CircleDollarSign size={15} /> {workingKey === `${product.product_key}:annual` ? 'Opening…' : 'Annual Checkout'}</button>
      </> : <a href="mailto:sales@syncratic.io?subject=MarketOps%20Institutional" className="inline-flex items-center gap-2 rounded bg-gray-900 px-3 py-1.5 text-sm font-medium text-white dark:bg-gray-100 dark:text-gray-900"><Mail size={15} /> Contact Sales</a>}
    </div>
  </article>;
}

function PriceLine({ label, value }: { label: string; value: string }) {
  return <div className="flex items-center justify-between gap-3 text-sm"><span className="text-gray-500 dark:text-gray-400">{label}</span><code className="text-xs text-gray-900 dark:text-gray-100">{value}</code></div>;
}

function shortStripePrice(value: string | undefined): string {
  return value ? value : 'Not mapped';
}

function featureName(value: string): string {
  return value.replace(/_/g, ' ').replace(/\b\w/g, (match) => match.toUpperCase());
}

function productSort(a: SubscriberSubscriptionProduct, b: SubscriberSubscriptionProduct): number {
  const order: Record<string, number> = { explorer: 1, professional: 2, institutional: 3 };
  return (order[a.product_key] ?? 99) - (order[b.product_key] ?? 99);
}
