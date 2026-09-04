import { useState } from 'react';
import { CheckCircle2, CircleDollarSign, ExternalLink, Mail } from 'lucide-react';
import { api } from '../api/client';
import type { SubscriberEffectiveSubscription, SubscriberSubscriptionProduct } from '../types';

export type BillingPeriod = 'monthly' | 'annual';

export const tierPositioning: Record<string, { headline: string; description: string; bullets: string[] }> = {
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

export function SubscriptionPlanGrid({
  products,
  currentProductKey,
  checkoutEnabled,
  tenantId,
}: {
  products: SubscriberSubscriptionProduct[];
  currentProductKey?: string;
  checkoutEnabled: boolean;
  tenantId: string;
}) {
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

  return <div className="space-y-3">
    {checkoutError ? <section className="rounded border border-amber-300 bg-amber-50 p-3 text-sm text-amber-900 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-100"><span className="font-semibold">Checkout unavailable.</span> {checkoutError}</section> : null}
    <section className="grid gap-4 lg:grid-cols-3">{products.map((product) => <PlanCard key={product.product_key} product={product} current={currentProductKey === product.product_key} workingKey={workingKey} checkoutEnabled={checkoutEnabled} onCheckout={startCheckout} />)}</section>
  </div>;
}

export function BillingSupportPanel({ tenantId, subscription }: { tenantId: string; subscription: SubscriberEffectiveSubscription | null | undefined }) {
  const [refundReason, setRefundReason] = useState('');
  const [refundAmount, setRefundAmount] = useState('');
  const [refundResult, setRefundResult] = useState<{ state: 'idle' | 'working' | 'success' | 'error'; message: string }>({ state: 'idle', message: '' });
  const [portalResult, setPortalResult] = useState<{ state: 'idle' | 'working' | 'error'; message: string }>({ state: 'idle', message: '' });

  async function requestRefund() {
    setRefundResult({ state: 'working', message: 'Submitting refund request…' });
    try {
      const cents = refundAmount.trim() ? Math.round(Number(refundAmount.trim()) * 100) : undefined;
      if (cents !== undefined && (!Number.isFinite(cents) || cents < 0)) throw new Error('Refund amount must be a valid non-negative number.');
      await api.createSubscriberRefundRequest(tenantId, { reason: refundReason.trim(), requested_amount_cents: cents, currency: 'usd', correlation_id: 'refund-request-' + Date.now() });
      setRefundReason('');
      setRefundAmount('');
      setRefundResult({ state: 'success', message: 'Refund request submitted. A subscription administrator will review it and take action in Stripe if approved.' });
    } catch (error) {
      setRefundResult({ state: 'error', message: error instanceof Error ? error.message : 'Refund request could not be submitted.' });
    }
  }

  async function openCustomerPortal() {
    setPortalResult({ state: 'working', message: 'Opening Stripe customer portal…' });
    try {
      const response = await api.createSubscriberPortalSession(tenantId);
      window.location.assign(response.portal_url);
    } catch (error) {
      setPortalResult({ state: 'error', message: error instanceof Error ? error.message : 'Stripe customer portal could not be opened.' });
    }
  }

  if (!subscription) {
    return <section className="rounded border border-gray-200 bg-white p-4 text-sm text-gray-700 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-200">
      <h2 className="text-sm font-semibold text-gray-950 dark:text-gray-50">Billing & refunds</h2>
      <p className="mt-1 text-xs leading-5 text-gray-600 dark:text-gray-300">Billing actions become available after an active subscription exists.</p>
    </section>;
  }

  return <section className="rounded border border-gray-200 bg-white p-4 text-sm text-gray-700 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-200">
    <div className="flex flex-wrap items-start justify-between gap-3">
      <div>
        <h2 className="text-sm font-semibold text-gray-950 dark:text-gray-50">Billing & refunds</h2>
        <p className="mt-1 text-xs leading-5 text-gray-600 dark:text-gray-300">Manage active Stripe-backed subscriptions in the Stripe customer portal. Refunds remain admin-reviewed so exceptions stay controlled and auditable.</p>
      </div>
      <button type="button" disabled={portalResult.state === 'working'} onClick={openCustomerPortal} className="inline-flex items-center gap-2 rounded border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-800 disabled:cursor-not-allowed disabled:opacity-50 dark:border-gray-600 dark:text-gray-100"><ExternalLink size={15} /> {portalResult.state === 'working' ? 'Opening…' : 'Manage subscription in Stripe'}</button>
    </div>
    {portalResult.state === 'error' ? <p role="status" className="mt-2 rounded border border-amber-300 bg-amber-50 px-3 py-2 text-xs text-amber-900 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-100">{portalResult.message}</p> : null}
    <p className="mt-4 text-xs leading-5 text-gray-600 dark:text-gray-300">Submit a refund request for administrator review. SignalOps records the request; an admin performs approved refunds in Stripe Dashboard.</p>
    <div className="mt-3 grid gap-2 md:grid-cols-[10rem_1fr_auto]">
      <label className="text-xs font-medium text-gray-700 dark:text-gray-300">Amount requested, optional<input inputMode="decimal" value={refundAmount} onChange={(event) => setRefundAmount(event.target.value)} placeholder="24.99" className="mt-1 block w-full rounded border border-gray-300 bg-white px-2 py-1.5 text-sm font-normal text-gray-900 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100" /></label>
      <label className="text-xs font-medium text-gray-700 dark:text-gray-300">Reason<textarea value={refundReason} onChange={(event) => setRefundReason(event.target.value)} rows={2} placeholder="Briefly explain the billing issue." className="mt-1 block w-full rounded border border-gray-300 bg-white px-2 py-1.5 text-sm font-normal text-gray-900 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100" /></label>
      <button type="button" disabled={refundResult.state === 'working' || !refundReason.trim()} onClick={requestRefund} className="self-end rounded bg-gray-900 px-3 py-1.5 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-50 dark:bg-gray-100 dark:text-gray-900">{refundResult.state === 'working' ? 'Submitting…' : 'Request refund'}</button>
    </div>
    {refundResult.state !== 'idle' ? <p role="status" className={`mt-2 text-xs ${refundResult.state === 'error' ? 'text-red-700 dark:text-red-300' : 'text-green-700 dark:text-green-300'}`}>{refundResult.message}</p> : null}
  </section>;
}

export function productSort(a: SubscriberSubscriptionProduct, b: SubscriberSubscriptionProduct): number {
  const order: Record<string, number> = { explorer: 1, professional: 2, institutional: 3 };
  return (order[a.product_key] ?? 99) - (order[b.product_key] ?? 99);
}

export function featureName(value: string): string {
  return value.replace(/_/g, ' ').replace(/\b\w/g, (match) => match.toUpperCase());
}

function PlanCard({ product, current, workingKey, checkoutEnabled, onCheckout }: { product: SubscriberSubscriptionProduct; current: boolean; workingKey: string; checkoutEnabled: boolean; onCheckout: (product: SubscriberSubscriptionProduct, billingPeriod: BillingPeriod) => void }) {
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
      {institutional ? <div className="flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-gray-100"><Mail size={16} /> Contact Sales</div> : <div className="space-y-1"><PriceLine label="Monthly" value={displayPrice(product.monthly_display_price, product.stripe_monthly_price_id)} /><PriceLine label="Annual" value={displayPrice(product.annual_display_price, product.stripe_annual_price_id)} /></div>}
    </div>
    <ul className="mt-3 space-y-2 text-xs text-gray-700 dark:text-gray-200">{info.bullets.map((bullet) => <li key={bullet} className="flex gap-2"><CheckCircle2 size={14} className="mt-0.5 shrink-0 text-emerald-600" />{bullet}</li>)}</ul>
    <div className="mt-4 flex flex-wrap gap-2">
      {selfService ? <>
        <button type="button" disabled={!checkoutEnabled || !monthlyMapped || Boolean(workingKey)} onClick={() => onCheckout(product, 'monthly')} className="inline-flex items-center gap-2 rounded bg-gray-900 px-3 py-1.5 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-50 dark:bg-gray-100 dark:text-gray-900"><CircleDollarSign size={15} /> {workingKey === `${product.product_key}:monthly` ? 'Opening…' : 'Monthly Checkout'}</button>
        <button type="button" disabled={!checkoutEnabled || !annualMapped || Boolean(workingKey)} onClick={() => onCheckout(product, 'annual')} className="inline-flex items-center gap-2 rounded border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-800 disabled:cursor-not-allowed disabled:opacity-50 dark:border-gray-600 dark:text-gray-100"><CircleDollarSign size={15} /> {workingKey === `${product.product_key}:annual` ? 'Opening…' : 'Annual Checkout'}</button>
      </> : <a href="mailto:sales@syncratic.io?subject=MarketOps%20Institutional" className="inline-flex items-center gap-2 rounded bg-gray-900 px-3 py-1.5 text-sm font-medium text-white dark:bg-gray-100 dark:text-gray-900"><Mail size={15} /> Contact Sales</a>}
    </div>
  </article>;
}

function PriceLine({ label, value }: { label: string; value: string }) {
  return <div className="flex items-center justify-between gap-3 text-sm"><span className="text-gray-500 dark:text-gray-400">{label}</span><code className="text-xs text-gray-900 dark:text-gray-100">{value}</code></div>;
}

function displayPrice(displayValue: string | undefined, stripePriceId: string | undefined): string {
  const value = (displayValue ?? '').trim();
  if (value) return value;
  return stripePriceId ? 'Price configured' : 'Not mapped';
}
