import { useMemo, type ReactNode } from 'react';
import { Link, useNavigate, useSearch } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { LogOut, UserRound } from 'lucide-react';
import { api } from '../api/client';
import { displayIdentity } from '../auth/claims';
import { useAuth, useTenant } from '../auth/session';
import { useTheme, type ThemePreference } from '../theme/theme';
import { useSubscription } from '../subscriber/SubscriptionContext';
import { BillingSupportPanel, productSort, SubscriptionPlanGrid } from '../subscriber/SubscriptionPlanCards';

type SettingsTab = 'account' | 'subscription' | 'billing' | 'preferences' | 'watchlists';
const tabs: Array<{ key: SettingsTab; label: string }> = [
  { key: 'account', label: 'Account' },
  { key: 'subscription', label: 'Subscription' },
  { key: 'billing', label: 'Billing & refunds' },
  { key: 'preferences', label: 'Preferences' },
  { key: 'watchlists', label: 'Watchlist defaults' },
];

export function MarketOpsSettingsRoute() {
  const search = useSearch({ strict: false }) as { tab?: string };
  const navigate = useNavigate();
  const activeTab = tabs.some((tab) => tab.key === search.tab) ? search.tab as SettingsTab : 'account';
  const tenantId = useTenant();
  const auth = useAuth();
  const { preference, setPreference } = useTheme();
  const subscription = useSubscription();
  const enrollmentQ = useQuery({ queryKey: ['session', 'enrollment', 'settings'], queryFn: api.getSessionEnrollment, staleTime: 30_000, retry: false });
  const productsQ = useQuery({ queryKey: ['subscriber-subscription-products'], queryFn: api.listSubscriberSubscriptionProducts, staleTime: 60_000 });
  const products = useMemo(() => (productsQ.data?.products ?? []).slice().sort(productSort), [productsQ.data?.products]);
  const checkoutEnabled = Boolean(productsQ.data?.checkout_enabled);
  const watchlistQ = useQuery({ queryKey: ['subscriber-watchlist-context', tenantId, 'settings'], queryFn: () => api.getSubscriberWatchlistContext(tenantId), enabled: Boolean(tenantId), staleTime: 60_000, retry: false });

  function setTab(tab: SettingsTab) {
    void navigate({ to: '/marketops/settings', search: { tab } });
  }

  return <div className="mx-auto max-w-6xl space-y-5">
    <section className="rounded border border-brand-100 bg-brand-50 p-5 dark:border-brand-900 dark:bg-brand-950/30">
      <p className="text-xs font-semibold uppercase tracking-wide text-brand-700 dark:text-brand-200">User settings</p>
      <div className="mt-2 flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold text-gray-950 dark:text-gray-50">Manage your MarketOps experience</h1>
          <p className="mt-1 max-w-3xl text-sm text-gray-700 dark:text-gray-200">Settings are now centered on your profile, package, billing, preferences, and watchlist defaults. Operational tools have moved to MarketOps Tools for privileged users.</p>
        </div>
        <Link to="/marketops/profile" className="inline-flex items-center gap-2 rounded border border-brand-200 bg-white px-3 py-2 text-sm text-brand-800 hover:bg-brand-50 dark:border-brand-800 dark:bg-gray-950 dark:text-brand-100"><UserRound size={15} /> View profile</Link>
      </div>
    </section>

    <div className="flex gap-2 overflow-x-auto border-b border-gray-200 pb-1 dark:border-gray-700" role="tablist" aria-label="MarketOps settings sections">
      {tabs.map((tab) => <button key={tab.key} type="button" role="tab" aria-selected={activeTab === tab.key} onClick={() => setTab(tab.key)} className={`whitespace-nowrap rounded-t px-3 py-2 text-sm font-medium ${activeTab === tab.key ? 'border-b-2 border-brand-600 text-brand-700 dark:text-brand-200' : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800'}`}>{tab.label}</button>)}
    </div>

    {activeTab === 'account' ? <section className="grid gap-4 lg:grid-cols-2">
      <Card title="Account summary">
        <dl className="space-y-2 text-sm"><Row label="User" value={displayIdentity(auth.claims) || enrollmentQ.data?.display_name || '—'} /><Row label="Email" value={auth.claims?.email || enrollmentQ.data?.email || '—'} /><Row label="Tenant" value={tenantId} /><Row label="Enrollment" value={enrollmentQ.data?.state || (enrollmentQ.isLoading ? 'Loading…' : 'Unknown')} /></dl>
        <button type="button" onClick={() => void auth.signOut()} className="mt-4 inline-flex items-center gap-2 rounded border border-gray-300 px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-100"><LogOut size={15} /> Sign out</button>
      </Card>
      <Card title="Current package">
        <dl className="space-y-2 text-sm"><Row label="Tier" value={subscription.subscription?.display_name || 'Unprovisioned'} /><Row label="Status" value={subscription.subscription?.status || '—'} /><Row label="Source" value={subscription.subscription?.source === 'tenant_seat' ? 'Institutional seat' : subscription.subscription?.source || '—'} /><Row label="Period end" value={formatOptionalDate(subscription.subscription?.current_period_ends_at)} /></dl>
        <button type="button" onClick={() => setTab('subscription')} className="mt-4 rounded bg-brand-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-brand-700">Review upgrade options</button>
      </Card>
    </section> : null}

    {activeTab === 'subscription' ? <section className="space-y-4">
      <Card title="Your subscription depth"><p className="text-sm text-gray-600 dark:text-gray-300">Choose the analytical depth that fits the work. Access changes only after Stripe confirms the subscription webhook.</p></Card>
      {productsQ.isLoading ? <p className="text-sm text-gray-500">Loading configured plans…</p> : productsQ.isError ? <p className="rounded border border-red-200 bg-red-50 p-3 text-sm text-red-800">Plan configuration is unavailable.</p> : <SubscriptionPlanGrid products={products} currentProductKey={subscription.subscription?.product_key} checkoutEnabled={checkoutEnabled} tenantId={tenantId} />}
    </section> : null}

    {activeTab === 'billing' ? <BillingSupportPanel tenantId={tenantId} subscription={subscription.subscription} /> : null}

    {activeTab === 'preferences' ? <Card title="Preferences">
      <label className="block text-sm font-medium text-gray-700 dark:text-gray-200">Color theme<select value={preference} onChange={(event) => setPreference(event.target.value as ThemePreference)} className="mt-1 block w-full max-w-xs rounded border border-gray-300 bg-white px-2 py-1.5 text-sm text-gray-900 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"><option value="system">System</option><option value="light">Light</option><option value="dark">Dark</option></select></label>
      <p className="mt-4 rounded border border-dashed border-gray-300 p-3 text-xs text-gray-500 dark:border-gray-700 dark:text-gray-400">Notification preferences will be added after the messaging policy is finalized.</p>
    </Card> : null}

    {activeTab === 'watchlists' ? <Card title="Watchlist defaults">
      <dl className="space-y-2 text-sm"><Row label="Selected watchlist" value={watchlistQ.data?.list_name || (watchlistQ.isLoading ? 'Loading…' : 'Not selected')} /><Row label="Selection mode" value={watchlistQ.data?.selection_mode || '—'} /><Row label="Coverage source" value="Global catalog with tenant/user watchlist context" /></dl>
      <Link to="/marketops/watchlists" className="mt-4 inline-flex rounded border border-gray-300 px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-100">Manage watchlists</Link>
    </Card> : null}
  </div>;
}

function Card({ title, children }: { title: string; children: ReactNode }) {
  return <section className="rounded border border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-900"><h2 className="text-sm font-semibold text-gray-950 dark:text-gray-50">{title}</h2><div className="mt-3">{children}</div></section>;
}

function Row({ label, value }: { label: string; value: string }) {
  return <div className="flex justify-between gap-3"><dt className="text-gray-500 dark:text-gray-400">{label}</dt><dd className="text-right font-medium text-gray-900 dark:text-gray-100">{value}</dd></div>;
}

function formatOptionalDate(value?: string) {
  if (!value) return '—';
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}
