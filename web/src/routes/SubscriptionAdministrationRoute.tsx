import { useEffect, useMemo, useState, type FormEvent, type ReactNode } from 'react';
import { api } from '../api/client';
import { getAccessToken, useAuth, useTenant } from '../auth/session';
import { hasSubscriptionAdministrator } from '../auth/claims';
import type { SubscriberSubscriptionAdministrationResponse, SubscriberSubscriptionFeature, SubscriberSubscriptionProduct, SubscriberUserActivityResponse, SubscriberUserActivityEventRecord, SubscriberUpgradeInteractionRecord, SubscriberRefundRequestAdminRecord } from '../types';
import { MetricTile } from '../components/MetricTile';
import { StatusBadge } from '../components/StatusBadge';

type SubjectPlan = 'explorer' | 'professional';
type SubscriptionStatus = 'trialing' | 'active' | 'past_due' | 'suspended' | 'canceled';
type SeatRole = 'member' | 'tenant_admin';
type SeatStatus = 'active' | 'revoked';
type MutationResult = { kind: string; state: 'idle' | 'working' | 'success' | 'error'; message: string };
type SubscriptionAdminTab = 'overview' | 'settings' | 'billing' | 'refunds' | 'users' | 'activity' | 'funnel' | 'audit' | 'webhooks';

const subscriptionTabs: Array<{ key: SubscriptionAdminTab; label: string; description: string }> = [
  { key: 'overview', label: 'Overview', description: 'Current subscription posture and mapped billing summary.' },
  { key: 'settings', label: 'Tier settings', description: 'Feature policies, limits, active state, and revisions.' },
  { key: 'billing', label: 'Stripe billing', description: 'Product, subject, and tenant Stripe mappings.' },
  { key: 'refunds', label: 'Refund requests', description: 'Admin-only refund intake and Stripe Dashboard action tracking.' },
  { key: 'users', label: 'Users & seats', description: 'Subscriber enrollment, tenant contracts, Institutional seats, and selected-user activity.' },
  { key: 'activity', label: 'User activity', description: 'Searchable login, logout, feature-view, and mutation activity.' },
  { key: 'funnel', label: 'Upgrade funnel', description: 'Contextual upgrade prompt impressions, clicks, and source attribution.' },
  { key: 'audit', label: 'Audit log', description: 'Searchable governance events for subscription changes.' },
  { key: 'webhooks', label: 'Webhook ledger', description: 'Searchable Stripe webhook processing evidence.' },
];

const features: Array<{ key: SubscriberSubscriptionFeature; label: string; depth: 'Explorer' | 'Professional' | 'Institutional' }> = [
  { key: 'market_dashboards', label: 'Market Dashboards', depth: 'Explorer' },
  { key: 'public_signals', label: 'Public Signals', depth: 'Explorer' },
  { key: 'sector_rotation_discovery', label: 'Sector Rotation Discovery', depth: 'Explorer' },
  { key: 'value_intelligence', label: 'Value Intelligence', depth: 'Professional' },
  { key: 'distressed_opportunity_intelligence', label: 'Distressed Opportunity Intelligence', depth: 'Professional' },
  { key: 'earnings_opportunity_intelligence', label: 'Earnings Opportunity Intelligence', depth: 'Professional' },
  { key: 'sector_rotation_detail', label: 'Sector Rotation Details', depth: 'Professional' },
  { key: 'options_signals', label: 'Options Signals', depth: 'Professional' },
  { key: 'earnings_calendar', label: 'Earnings Calendar', depth: 'Professional' },
  { key: 'research_reports', label: 'Research Reports', depth: 'Professional' },
  { key: 'syncratic_explainability', label: 'Syncratic Explainability', depth: 'Professional' },
  { key: 'signal_assurance_analytics', label: 'Signal Assurance Analytics', depth: 'Institutional' },
  { key: 'portfolio_analysis', label: 'Portfolio Analysis', depth: 'Institutional' },
  { key: 'batch_screening', label: 'Batch Screening', depth: 'Institutional' },
  { key: 'historical_replay', label: 'Historical Replay', depth: 'Institutional' },
  { key: 'strategy_validation', label: 'Strategy Validation', depth: 'Institutional' },
  { key: 'custom_universes', label: 'Custom Universes', depth: 'Institutional' },
  { key: 'api', label: 'APIs', depth: 'Institutional' },
  { key: 'white_label', label: 'White-Label Deployment', depth: 'Institutional' },
];

const requestHeaders = () => ({
  'Content-Type': 'application/json',
  ...(getAccessToken() ? { Authorization: `Bearer ${getAccessToken()}` } : {}),
});
function correlation(prefix: string) { return prefix + '-' + Date.now(); }
async function provision(path: string, body: unknown): Promise<void> {
  const response = await fetch(path, { method: path.endsWith('/seats') ? 'PUT' : 'POST', headers: requestHeaders(), body: JSON.stringify(body) });
  if (!response.ok) {
    const payload = await response.json().catch(() => null) as { message?: string; error?: string } | null;
    throw new Error(payload?.message ?? payload?.error ?? `Request failed (${response.status})`);
  }
}

export function SubscriptionAdministrationRoute() {
  const { authEnabled, claims } = useAuth();
  const currentTenant = useTenant();
  const allowed = !authEnabled || hasSubscriptionAdministrator(claims);
  const [tenantFilter, setTenantFilter] = useState(currentTenant);
  const [snapshot, setSnapshot] = useState<SubscriberSubscriptionAdministrationResponse | null>(null);
  const [activitySnapshot, setActivitySnapshot] = useState<SubscriberUserActivityResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState('');
  const [result, setResult] = useState<MutationResult>({ kind: '', state: 'idle', message: '' });
  const [editingProductKey, setEditingProductKey] = useState('');
  const [limitDraft, setLimitDraft] = useState('{}');
  const [featureDraft, setFeatureDraft] = useState<Partial<Record<SubscriberSubscriptionFeature, boolean>>>({});
  const [trialDraft, setTrialDraft] = useState(0);
  const [displayDraft, setDisplayDraft] = useState('');
  const [freeDraft, setFreeDraft] = useState(false);
  const [activeDraft, setActiveDraft] = useState(true);
  const [stripeProductDraft, setStripeProductDraft] = useState('');
  const [stripeMonthlyDraft, setStripeMonthlyDraft] = useState('');
  const [stripeAnnualDraft, setStripeAnnualDraft] = useState('');
  const [monthlyDisplayDraft, setMonthlyDisplayDraft] = useState('');
  const [annualDisplayDraft, setAnnualDisplayDraft] = useState('');
  const [subjectPlan, setSubjectPlan] = useState<SubjectPlan>('explorer');
  const [subjectStatus, setSubjectStatus] = useState<SubscriptionStatus>('active');
  const [subjectTenant, setSubjectTenant] = useState(currentTenant);
  const [subject, setSubject] = useState('');
  const [subjectStripeCustomer, setSubjectStripeCustomer] = useState('');
  const [subjectStripeSubscription, setSubjectStripeSubscription] = useState('');
  const [subjectPeriodEnd, setSubjectPeriodEnd] = useState('');
  const [tenant, setTenant] = useState(currentTenant);
  const [tenantStatus, setTenantStatus] = useState<SubscriptionStatus>('active');
  const [tenantStripeCustomer, setTenantStripeCustomer] = useState('');
  const [tenantStripeSubscription, setTenantStripeSubscription] = useState('');
  const [tenantPeriodEnd, setTenantPeriodEnd] = useState('');
  const [seatTenant, setSeatTenant] = useState(currentTenant);
  const [seatSubject, setSeatSubject] = useState('');
  const [seatRole, setSeatRole] = useState<SeatRole>('member');
  const [seatStatus, setSeatStatus] = useState<SeatStatus>('active');
  const [activeTab, setActiveTab] = useState<SubscriptionAdminTab>('overview');
  const [auditSearch, setAuditSearch] = useState('');
  const [activitySearch, setActivitySearch] = useState('');
  const [funnelSearch, setFunnelSearch] = useState('');
  const [selectedActivitySubject, setSelectedActivitySubject] = useState('');
  const [webhookSearch, setWebhookSearch] = useState('');
  const [refundSearch, setRefundSearch] = useState('');
  const [refundStatusDrafts, setRefundStatusDrafts] = useState<Record<string, string>>({});
  const [refundNoteDrafts, setRefundNoteDrafts] = useState<Record<string, string>>({});

  const roleDescription = useMemo(() => authEnabled
    ? 'The signed signalops:subscription_admin role is required. Every change is persisted with the signed actor and correlation identifier.'
    : 'Authentication is disabled in this environment; production requires the signed platform subscription-admin role.', [authEnabled]);

  const selectedProduct = snapshot?.products.find((product) => product.product_key === editingProductKey) ?? snapshot?.products[0] ?? null;

  const refresh = async (tenantId = tenantFilter) => {
    if (!allowed || !tenantId.trim()) return;
    setLoading(true); setLoadError('');
    try {
      const [data, activity] = await Promise.all([
        api.getSubscriberSubscriptionAdministration(tenantId.trim()),
        api.getSubscriberUserActivity(tenantId.trim(), { limit: 250 }),
      ]);
      setSnapshot(data);
      setActivitySnapshot(activity);
      if (!editingProductKey && data.products[0]) seedProductEditor(data.products[0]);
    } catch (error) {
      setLoadError(error instanceof Error ? error.message : 'Subscription administration failed to load.');
    } finally {
      setLoading(false);
    }
  };

  const seedProductEditor = (product: SubscriberSubscriptionProduct) => {
    setEditingProductKey(product.product_key);
    setDisplayDraft(product.display_name);
    setTrialDraft(product.trial_days);
    setFreeDraft(product.is_free);
    setActiveDraft(product.active ?? true);
    setStripeProductDraft(product.stripe_product_id ?? '');
    setStripeMonthlyDraft(product.stripe_monthly_price_id ?? '');
    setStripeAnnualDraft(product.stripe_annual_price_id ?? '');
    setMonthlyDisplayDraft(product.monthly_display_price ?? '');
    setAnnualDisplayDraft(product.annual_display_price ?? '');
    setFeatureDraft(product.feature_policy ?? {});
    setLimitDraft(JSON.stringify(product.limit_policy ?? {}, null, 2));
  };

  useEffect(() => { void refresh(currentTenant); }, [allowed, currentTenant]);

  if (!allowed) return <section className="rounded border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800"><h1 className="font-semibold">Subscription Administration</h1><p className="mt-1">The platform subscription administrator role is required. MarketOps tenant roles cannot create or alter analytical subscriptions.</p></section>;

  const run = async (kind: string, fn: () => Promise<void>) => {
    setResult({ kind, state: 'working', message: 'Submitting audited subscription governance change…' });
    try { await fn(); setResult({ kind, state: 'success', message: 'Subscription governance change recorded successfully.' }); await refresh(); }
    catch (error) { setResult({ kind, state: 'error', message: error instanceof Error ? error.message : 'Provisioning failed.' }); }
  };

  const subjectSubmit = (event: FormEvent) => { event.preventDefault(); void run('subject', () => provision('/v1/administration/subscriptions/subject', { tenant_id: subjectTenant.trim(), subject: subject.trim(), product_key: subjectPlan, status: subjectStatus, correlation_id: correlation('subject-plan') })); };
  const tenantSubmit = (event: FormEvent) => { event.preventDefault(); void run('tenant', () => provision('/v1/administration/subscriptions/tenant', { tenant_id: tenant.trim(), product_key: 'institutional', status: tenantStatus, correlation_id: correlation('institutional-contract') })); };
  const seatSubmit = (event: FormEvent) => { event.preventDefault(); void run('seat', () => provision('/v1/administration/subscriptions/seats', { tenant_id: seatTenant.trim(), subject: seatSubject.trim(), seat_role: seatRole, status: seatStatus, correlation_id: correlation('institutional-seat') })); };
  const productSubmit = (event: FormEvent) => {
    event.preventDefault();
    void run('product', async () => {
      const limits = JSON.parse(limitDraft) as Record<string, number>;
      await api.updateSubscriberSubscriptionProduct(editingProductKey, { display_name: displayDraft.trim(), is_free: freeDraft, trial_days: trialDraft, feature_policy: featureDraft, limit_policy: limits, active: activeDraft, correlation_id: correlation('subscription-product-policy') });
    });
  };
  const productBillingSubmit = (event: FormEvent) => { event.preventDefault(); void run('product-billing', async () => { await api.updateSubscriberSubscriptionProductBilling(editingProductKey, { stripe_product_id: stripeProductDraft.trim(), stripe_monthly_price_id: stripeMonthlyDraft.trim(), stripe_annual_price_id: stripeAnnualDraft.trim(), monthly_display_price: monthlyDisplayDraft.trim(), annual_display_price: annualDisplayDraft.trim(), correlation_id: correlation('stripe-product') }); }); };
  const subjectBillingSubmit = (event: FormEvent) => { event.preventDefault(); void run('subject-billing', async () => { await api.updateSubscriberSubjectSubscriptionBilling({ tenant_id: subjectTenant.trim(), subject: subject.trim(), stripe_customer_id: subjectStripeCustomer.trim(), stripe_subscription_id: subjectStripeSubscription.trim(), status: subjectStatus, current_period_ends_at: subjectPeriodEnd.trim() || undefined, correlation_id: correlation('stripe-subject') }); }); };
  const tenantBillingSubmit = (event: FormEvent) => { event.preventDefault(); void run('tenant-billing', async () => { await api.updateSubscriberTenantSubscriptionBilling({ tenant_id: tenant.trim(), stripe_customer_id: tenantStripeCustomer.trim(), stripe_subscription_id: tenantStripeSubscription.trim(), status: tenantStatus, current_period_ends_at: tenantPeriodEnd.trim() || undefined, correlation_id: correlation('stripe-tenant') }); }); };
  const refundSubmit = (request: SubscriberRefundRequestAdminRecord) => {
    const status = refundStatusDrafts[request.refund_request_id] || request.status;
    const adminNote = refundNoteDrafts[request.refund_request_id] ?? request.admin_note ?? '';
    void run('refund-' + request.refund_request_id, async () => {
      await api.updateSubscriberRefundRequest(request.refund_request_id, { tenant_id: request.tenant_id, status, admin_note: adminNote, correlation_id: correlation('refund-request') });
    });
  };
  const notice = (kind: string) => result.kind === kind && result.state !== 'idle' ? <p role="status" className={`mt-2 text-xs ${result.state === 'error' ? 'text-red-700' : result.state === 'success' ? 'text-green-700' : 'text-gray-600'}`}>{result.message}</p> : null;

  return <div className="space-y-3">
    <div className="flex flex-wrap items-end justify-between gap-3">
      <div>
        <h1 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Subscription Administration</h1>
        <p className="text-xs text-gray-500 dark:text-gray-400">Govern enrolled users, tenant contracts, seats, analytical-depth tiers, entitlement policy, limits, billing mappings, and audit evidence.</p>
      </div>
      <div className="flex items-end gap-2">
        <Field label="Tenant filter" value={tenantFilter} onChange={setTenantFilter} />
        <button onClick={() => void refresh()} disabled={loading || !tenantFilter.trim()} className="rounded border border-gray-300 bg-white px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50 disabled:opacity-50 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200 dark:hover:bg-gray-800">{loading ? 'Loading…' : 'Refresh'}</button>
      </div>
    </div>
    <Summary snapshot={snapshot} />
    <section className="rounded border border-gray-200 bg-white p-3 text-xs text-gray-600 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-300">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h2 className="text-sm font-semibold text-gray-900 dark:text-gray-100">Platform control plane</h2>
          <p className="mt-1">{roleDescription}</p>
        </div>
        <StatusBadge status="active" />
      </div>
      <p className="mt-1 text-gray-500 dark:text-gray-400">Use immutable OIDC subjects, not email addresses. Tenant-level MarketOps roles do not grant subscription administration authority. This page never triggers provider polling or creates tenant-specific market-data copies.</p>
    </section>
    {loadError ? <section className="rounded border border-red-200 bg-red-50 p-3 text-xs text-red-800">{loadError}</section> : null}
    <nav aria-label="Subscription administration sections" className="flex gap-1 overflow-x-auto border-b border-gray-200 text-sm dark:border-gray-700">{subscriptionTabs.map((tab) => <button key={tab.key} type="button" onClick={() => setActiveTab(tab.key)} className={`shrink-0 border-b-2 px-3 py-2 font-medium ${activeTab === tab.key ? 'border-brand-600 text-brand-700 dark:border-brand-400 dark:text-brand-200' : 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-100'}`}>{tab.label}</button>)}</nav>
    <p className="text-xs text-gray-500 dark:text-gray-400">{subscriptionTabs.find((tab) => tab.key === activeTab)?.description}</p>
    {activeTab === 'overview' ? <div className="grid gap-4 xl:grid-cols-2"><ProductBillingMap products={snapshot?.products ?? []} /><SubjectTables snapshot={snapshot} compact /></div> : null}
    {activeTab === 'settings' ? <section className="rounded border border-gray-200 bg-white p-4"><h2 className="text-sm font-semibold">Tier policy and entitlement alignment</h2><p className="mt-1 text-xs text-gray-500">Feature policy is the server-side entitlement contract used by subscription enforcement. Updating a product increments its revision and writes audit evidence.</p><div className="mt-3 grid gap-3 lg:grid-cols-3">{snapshot?.products.map((product) => <button key={product.product_key} onClick={() => seedProductEditor(product)} className={`rounded border p-3 text-left text-xs ${editingProductKey === product.product_key ? 'border-brand-600 bg-brand-50' : 'border-gray-200 bg-white'}`}><div className="flex items-center justify-between"><span className="text-sm font-semibold">{product.display_name}</span><StatusPill value={product.active === false ? 'inactive' : 'active'} /></div><p className="mt-1 text-gray-500">{product.billing_scope} · revision {product.revision} · trial {product.trial_days} days</p><p className="mt-2 text-gray-700">{Object.entries(product.feature_policy ?? {}).filter(([, enabled]) => enabled).length} enabled features · limits {Object.keys(product.limit_policy ?? {}).length}</p></button>)}</div>{selectedProduct ? <form onSubmit={productSubmit} className="mt-4 space-y-3 rounded border border-gray-200 p-3"><div className="grid gap-2 sm:grid-cols-4"><Field label="Display name" value={displayDraft} onChange={setDisplayDraft} required /><NumberField label="Trial days" value={trialDraft} onChange={setTrialDraft} /><Check label="Free tier" checked={freeDraft} onChange={setFreeDraft} /><Check label="Product active" checked={activeDraft} onChange={setActiveDraft} /></div><div className="grid gap-2 md:grid-cols-3">{features.map((feature) => <Check key={feature.key} label={`${feature.label} (${feature.depth})`} checked={Boolean(featureDraft[feature.key])} onChange={(checked) => setFeatureDraft((current) => ({ ...current, [feature.key]: checked }))} />)}</div><label className="block text-xs font-medium text-gray-700">Limit policy JSON<textarea value={limitDraft} onChange={(event) => setLimitDraft(event.target.value)} rows={4} className="mt-1 block w-full rounded border border-gray-300 px-2 py-1.5 font-mono text-xs" /></label><button disabled={!editingProductKey || result.state === 'working'} className="rounded bg-brand-700 px-3 py-1.5 text-sm text-white disabled:opacity-50">Save tier policy</button>{notice('product')}</form> : null}</section> : null}
    {activeTab === 'billing' ? <section className="rounded border border-gray-200 bg-white p-4"><h2 className="text-sm font-semibold">Admin-managed Stripe billing</h2><p className="mt-1 text-xs text-gray-500">Map Stripe IDs created in Stripe Dashboard. This does not enable Checkout, customer portal, or automatic access creation from unknown Stripe events.</p><div className="mt-3"><ProductBillingMap products={snapshot?.products ?? []} selectedProductKey={editingProductKey} onSelectProduct={seedProductEditor} /></div><div className="mt-3 grid gap-4 xl:grid-cols-3"><form onSubmit={productBillingSubmit} className="space-y-2 rounded border border-gray-200 p-3"><h3 className="text-xs font-semibold uppercase tracking-wide text-gray-500">Product mapping</h3><p className="text-[11px] text-gray-500">Selected product: {selectedProduct?.display_name ?? 'none selected'}. Choose a different tier from Configured Stripe products above.</p><Field label="Stripe product ID" value={stripeProductDraft} onChange={setStripeProductDraft} placeholder="prod_..." /><Field label="Monthly price ID" value={stripeMonthlyDraft} onChange={setStripeMonthlyDraft} placeholder="price_..." /><Field label="Annual price ID" value={stripeAnnualDraft} onChange={setStripeAnnualDraft} placeholder="price_..." /><Field label="Monthly display price" value={monthlyDisplayDraft} onChange={setMonthlyDisplayDraft} placeholder="$24.99/mo" /><Field label="Annual display price" value={annualDisplayDraft} onChange={setAnnualDisplayDraft} placeholder="$249/yr" /><button disabled={!editingProductKey || result.state === 'working'} className="rounded bg-gray-900 px-3 py-1.5 text-sm text-white disabled:opacity-50">Save product billing</button>{notice('product-billing')}</form><form onSubmit={subjectBillingSubmit} className="space-y-2 rounded border border-gray-200 p-3"><h3 className="text-xs font-semibold uppercase tracking-wide text-gray-500">Subject Stripe mapping</h3><Field label="Tenant ID" value={subjectTenant} onChange={setSubjectTenant} /><Field label="OIDC subject" value={subject} onChange={setSubject} /><Field label="Stripe customer ID" value={subjectStripeCustomer} onChange={setSubjectStripeCustomer} placeholder="cus_..." /><Field label="Stripe subscription ID" value={subjectStripeSubscription} onChange={setSubjectStripeSubscription} placeholder="sub_..." /><Field label="Current period end" value={subjectPeriodEnd} onChange={setSubjectPeriodEnd} placeholder="2026-09-01T00:00:00Z" /><button disabled={!subject.trim() || !subjectStripeCustomer.trim() || !subjectStripeSubscription.trim() || result.state === 'working'} className="rounded bg-gray-900 px-3 py-1.5 text-sm text-white disabled:opacity-50">Save subject billing</button>{notice('subject-billing')}</form><form onSubmit={tenantBillingSubmit} className="space-y-2 rounded border border-gray-200 p-3"><h3 className="text-xs font-semibold uppercase tracking-wide text-gray-500">Institutional Stripe mapping</h3><Field label="Tenant ID" value={tenant} onChange={setTenant} /><Field label="Stripe customer ID" value={tenantStripeCustomer} onChange={setTenantStripeCustomer} placeholder="cus_..." /><Field label="Stripe subscription ID" value={tenantStripeSubscription} onChange={setTenantStripeSubscription} placeholder="sub_..." /><Field label="Current period end" value={tenantPeriodEnd} onChange={setTenantPeriodEnd} placeholder="2026-09-01T00:00:00Z" /><button disabled={!tenant.trim() || !tenantStripeCustomer.trim() || !tenantStripeSubscription.trim() || result.state === 'working'} className="rounded bg-gray-900 px-3 py-1.5 text-sm text-white disabled:opacity-50">Save tenant billing</button>{notice('tenant-billing')}</form></div></section> : null}
    {activeTab === 'refunds' ? <RefundRequestsView requests={snapshot?.refund_requests ?? []} search={refundSearch} onSearch={setRefundSearch} statusDrafts={refundStatusDrafts} noteDrafts={refundNoteDrafts} onStatusDraft={(id, value) => setRefundStatusDrafts((current) => ({ ...current, [id]: value }))} onNoteDraft={(id, value) => setRefundNoteDrafts((current) => ({ ...current, [id]: value }))} onSubmit={refundSubmit} workingKind={result.state === 'working' ? result.kind : ''} notice={notice} /> : null}
    {activeTab === 'users' ? <div className="space-y-4"><section className="grid gap-4 lg:grid-cols-2"><ProvisionSubject subjectTenant={subjectTenant} setSubjectTenant={setSubjectTenant} subject={subject} setSubject={setSubject} subjectPlan={subjectPlan} setSubjectPlan={setSubjectPlan} subjectStatus={subjectStatus} setSubjectStatus={setSubjectStatus} onSubmit={subjectSubmit} disabled={result.state === 'working'} notice={notice('subject')} /><ProvisionTenant tenant={tenant} setTenant={setTenant} tenantStatus={tenantStatus} setTenantStatus={setTenantStatus} onSubmit={tenantSubmit} disabled={result.state === 'working'} notice={notice('tenant')} /><ProvisionSeat seatTenant={seatTenant} setSeatTenant={setSeatTenant} seatSubject={seatSubject} setSeatSubject={setSeatSubject} seatRole={seatRole} setSeatRole={setSeatRole} seatStatus={seatStatus} setSeatStatus={setSeatStatus} onSubmit={seatSubmit} disabled={result.state === 'working'} notice={notice('seat')} /></section><SubjectTables snapshot={snapshot} /><UserActivityDrilldown snapshot={snapshot} activity={activitySnapshot} selectedSubject={selectedActivitySubject} onSelectSubject={setSelectedActivitySubject} /></div> : null}
    {activeTab === 'activity' ? <UserActivityView activity={activitySnapshot} search={activitySearch} onSearch={setActivitySearch} /> : null}
    {activeTab === 'funnel' ? <UpgradeFunnelView interactions={snapshot?.upgrade_interactions ?? []} search={funnelSearch} onSearch={setFunnelSearch} /> : null}
    {activeTab === 'audit' ? <AuditLogView snapshot={snapshot} search={auditSearch} onSearch={setAuditSearch} /> : null}
    {activeTab === 'webhooks' ? <WebhookLedgerView snapshot={snapshot} search={webhookSearch} onSearch={setWebhookSearch} /> : null}
  </div>;
}

function Summary({ snapshot }: { snapshot: SubscriberSubscriptionAdministrationResponse | null }) {
  const activeSubjects = snapshot?.subject_subscriptions.filter((record) => ['active', 'trialing'].includes(record.status)).length ?? 0;
  const activeSeats = snapshot?.seats.filter((record) => record.status === 'active').length ?? 0;
  const mappedStripeProducts = snapshot?.products.filter((product) => product.stripe_product_id || product.stripe_monthly_price_id || product.stripe_annual_price_id).length ?? 0;
  const openRefunds = snapshot?.refund_requests?.filter((record) => ['requested', 'reviewing', 'approved_for_manual_refund'].includes(record.status)).length ?? 0;
  return <div className="grid grid-cols-2 gap-2 md:grid-cols-3 xl:grid-cols-7"><MetricTile label="Products" value={snapshot?.products.length ?? 0} /><MetricTile label="Subject subscribers" value={activeSubjects} /><MetricTile label="Institutional seats" value={activeSeats} /><MetricTile label="Stripe product maps" value={mappedStripeProducts} /><MetricTile label="Refund queue" value={openRefunds} /><MetricTile label="Audit events" value={snapshot?.audit_events.length ?? 0} /><MetricTile label="Webhook events" value={snapshot?.billing_webhook_events.length ?? 0} /></div>;
}

function SectionTitle({ title, description }: { title: string; description: string }) { return <div><h2 className="text-sm font-semibold text-gray-900 dark:text-gray-100">{title}</h2><p className="text-xs text-gray-500 dark:text-gray-400">{description}</p></div>; }
function ProductBillingMap({ products, selectedProductKey, onSelectProduct }: { products: SubscriberSubscriptionProduct[]; selectedProductKey?: string; onSelectProduct?: (product: SubscriberSubscriptionProduct) => void }) {
  const interactive = Boolean(onSelectProduct);
  return <section className="space-y-2"><SectionTitle title="Stripe product map" description={interactive ? "Select a configured Stripe product row to control the Product mapping form below." : "Read-only billing reference for every configured tier."} /><section className="space-y-2"><h2 className="text-sm font-semibold text-gray-900 dark:text-gray-100">Configured Stripe products</h2><div className="overflow-x-auto rounded border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-900"><table className="min-w-full divide-y divide-gray-200 text-left text-xs dark:divide-gray-700"><thead className="bg-gray-50 text-gray-500 dark:bg-gray-800 dark:text-gray-300"><tr><th className="px-2 py-1 font-medium">Tier</th><th className="px-2 py-1 font-medium">Scope</th><th className="px-2 py-1 font-medium">Product</th><th className="px-2 py-1 font-medium">Monthly</th><th className="px-2 py-1 font-medium">Annual</th></tr></thead><tbody className="divide-y divide-gray-100 dark:divide-gray-800">{products.length ? products.map((product) => { const selected = selectedProductKey === product.product_key; return <tr key={product.product_key} onClick={onSelectProduct ? () => onSelectProduct(product) : undefined} className={interactive ? `cursor-pointer ${selected ? 'bg-brand-50 dark:bg-brand-950/30' : 'hover:bg-gray-50 dark:hover:bg-gray-800'}` : undefined}><td className="px-2 py-2 align-top">{interactive ? <button type="button" onClick={(event) => { event.stopPropagation(); onSelectProduct?.(product); }} className={`text-left font-medium ${selected ? 'text-brand-700 dark:text-brand-200' : 'text-gray-900 dark:text-gray-100'}`}>{product.display_name}<span className="ml-2">{selected ? <StatusBadge status="active" /> : null}</span></button> : <span className="font-medium text-gray-900 dark:text-gray-100">{product.display_name}</span>}</td><td className="px-2 py-2 align-top text-gray-700 dark:text-gray-300">{product.billing_scope}</td><td className="px-2 py-2 align-top">{stripeCode(product.stripe_product_id)}</td><td className="px-2 py-2 align-top"><PriceAdminCell display={product.monthly_display_price} id={product.stripe_monthly_price_id} /></td><td className="px-2 py-2 align-top"><PriceAdminCell display={product.annual_display_price} id={product.stripe_annual_price_id} /></td></tr>; }) : <tr><td colSpan={5} className="px-2 py-3 text-gray-500 dark:text-gray-400">No subscription products configured.</td></tr>}</tbody></table></div></section></section>;
}

function PriceAdminCell({ display, id }: { display?: string; id?: string }) { return <div className="space-y-1"><div className="font-medium text-gray-900 dark:text-gray-100">{display || 'Not set'}</div>{stripeCode(id)}</div>; }

function SubjectTables({ snapshot, compact = false }: { snapshot: SubscriberSubscriptionAdministrationResponse | null; compact?: boolean }) {
  const subjectRows = (snapshot?.subject_subscriptions ?? []).map((r) => [<IdentityCell subject={r.subject} displayName={r.subject_display_name} email={r.subject_email} />, r.display_name, <StatusBadge status={r.status} />, compactStripe(r.stripe_customer_id, r.stripe_subscription_id), formatDate(r.updated_at)]);
  const contractRows = (snapshot?.tenant_subscriptions ?? []).map((r) => [r.tenant_id, r.display_name, <StatusBadge status={r.status} />, compactStripe(r.stripe_customer_id, r.stripe_subscription_id), formatDate(r.updated_at)]);
  const seatRows = (snapshot?.seats ?? []).map((r) => [<IdentityCell subject={r.subject} displayName={r.subject_display_name} email={r.subject_email} />, <IdentityCell subject={r.assigned_by} displayName={r.assigned_by_display_name} email={r.assigned_by_email} />, r.seat_role, <StatusBadge status={r.status} />]);
  return <div className="grid gap-4 xl:grid-cols-2"><Table title="Subject subscriptions" headers={['Subject', 'Plan', 'Status', 'Stripe', 'Updated']} rows={subjectRows} empty="No subject subscriptions for this tenant." /><Table title="Institutional contracts" headers={['Tenant', 'Plan', 'Status', 'Stripe', 'Updated']} rows={contractRows} empty="No institutional contract for this tenant." />{compact ? null : <Table title="Institutional seats" headers={['Subject', 'Assigned by', 'Role', 'Status']} rows={seatRows} empty="No seats for this tenant." />}</div>;
}

function RefundRequestsView({ requests, search, onSearch, statusDrafts, noteDrafts, onStatusDraft, onNoteDraft, onSubmit, workingKind, notice }: { requests: SubscriberRefundRequestAdminRecord[]; search: string; onSearch: (value: string) => void; statusDrafts: Record<string, string>; noteDrafts: Record<string, string>; onStatusDraft: (id: string, value: string) => void; onNoteDraft: (id: string, value: string) => void; onSubmit: (request: SubscriberRefundRequestAdminRecord) => void; workingKind: string; notice: (kind: string) => ReactNode }) {
  const term = search.trim().toLowerCase();
  const filtered = requests.filter((record) => !term || [record.requested_at, record.subject_email, record.subject_display_name, record.subject, record.product_key, record.display_name, record.reason, record.status, record.admin_note, record.stripe_customer_id, record.stripe_subscription_id, record.stripe_session_id].join(' ').toLowerCase().includes(term));
  const openCount = requests.filter((record) => ['requested', 'reviewing', 'approved_for_manual_refund'].includes(record.status)).length;
  return <section className="space-y-4"><div className="flex flex-wrap items-end justify-between gap-3"><SectionTitle title="Refund request queue" description="Customers can request refunds; subscription admins triage here and execute any approved refund in Stripe Dashboard. This screen records disposition only." /><Field label="Search refunds" value={search} onChange={onSearch} placeholder="email, status, reason, Stripe ID…" /></div><div className="grid gap-2 sm:grid-cols-3"><MetricTile label="Open requests" value={openCount} /><MetricTile label="Total requests" value={requests.length} /><MetricTile label="Manual Stripe action" value="Admin only" /></div><div className="space-y-3">{filtered.length ? filtered.map((request) => { const kind = 'refund-' + request.refund_request_id; return <article key={request.refund_request_id} className="rounded border border-gray-200 bg-white p-3 text-xs dark:border-gray-700 dark:bg-gray-900"><div className="flex flex-wrap items-start justify-between gap-3"><div className="space-y-1"><IdentityCell subject={request.subject} displayName={request.subject_display_name} email={request.subject_email} /><div className="text-gray-600 dark:text-gray-400">{request.display_name || request.product_key || 'Subscription'} · requested {formatDate(request.requested_at)}</div><div className="text-gray-700 dark:text-gray-300">{request.reason || 'No reason supplied.'}</div></div><StatusBadge status={request.status} /></div><div className="mt-3 grid gap-2 md:grid-cols-3"><div><div className="font-medium text-gray-500 dark:text-gray-400">Stripe customer</div>{stripeCode(request.stripe_customer_id)}</div><div><div className="font-medium text-gray-500 dark:text-gray-400">Stripe subscription</div>{stripeCode(request.stripe_subscription_id)}</div><div><div className="font-medium text-gray-500 dark:text-gray-400">Stripe checkout session</div>{stripeCode(request.stripe_session_id)}</div></div><div className="mt-3 grid gap-2 md:grid-cols-[16rem_1fr_auto]"><Select label="Admin status" value={statusDrafts[request.refund_request_id] || request.status} onChange={(value) => onStatusDraft(request.refund_request_id, value)}><RefundStatusOptions /></Select><Field label="Admin note" value={noteDrafts[request.refund_request_id] ?? request.admin_note ?? ''} onChange={(value) => onNoteDraft(request.refund_request_id, value)} placeholder="Record Stripe refund reference, rejection reason, or follow-up." /><button type="button" disabled={workingKind === kind} onClick={() => onSubmit(request)} className="self-end rounded bg-gray-900 px-3 py-1.5 text-sm text-white disabled:opacity-50 dark:bg-gray-100 dark:text-gray-900">{workingKind === kind ? 'Saving…' : 'Save disposition'}</button></div>{notice(kind)}</article>; }) : <p className="rounded border border-dashed border-gray-300 p-3 text-xs text-gray-500 dark:border-gray-700 dark:text-gray-400">No refund requests match the current scope.</p>}</div></section>;
}

function RefundStatusOptions() { return <><option value="requested">Requested</option><option value="reviewing">Reviewing</option><option value="approved_for_manual_refund">Approved for manual Stripe refund</option><option value="rejected">Rejected</option><option value="manual_refund_completed">Manual refund completed</option><option value="closed">Closed</option></>; }

function UserActivityView({ activity, search, onSearch }: { activity: SubscriberUserActivityResponse | null; search: string; onSearch: (value: string) => void }) {
  const events = filterActivityEvents(activity?.events ?? [], search);
  const summaries = filterActivitySummaries(activity, search);
  return <section className="space-y-4"><div className="flex flex-wrap items-end justify-between gap-3"><SectionTitle title="User activity" description="Operational visibility into login, logout, MarketOps feature views, and POST/PUT/DELETE API mutations. Payloads, tokens, cookies, and response bodies are intentionally excluded." /><Field label="Search user activity" value={search} onChange={onSearch} placeholder="email, user, feature, route, status…" /></div><div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-4"><MetricTile label="Active users in ledger" value={activity?.summaries.length ?? 0} /><MetricTile label="Recent events" value={activity?.events.length ?? 0} /><MetricTile label="Feature views" value={activity?.events.filter((event) => event.event_type === 'feature_view').length ?? 0} /><MetricTile label="Mutations" value={activity?.events.filter((event) => event.event_type === 'api_mutation').length ?? 0} /></div><Table title={`User summaries (${summaries.length})`} headers={['User', 'Last activity', 'Last login', 'Feature views', 'Mutations', 'Top feature']} rows={summaries.map((r) => [<IdentityCell subject={r.subject} displayName={r.subject_display_name} email={r.subject_email} />, formatDate(r.last_activity_at), formatDate(r.last_login_at), String(r.feature_view_count), <span>{r.mutation_count}{r.failed_mutation_count ? <span className="ml-1 text-red-700 dark:text-red-300">({r.failed_mutation_count} failed)</span> : null}</span>, r.top_feature_key || '—'])} empty="No user activity summaries match the search." /><ActivityEventTable events={events} title={`Activity events (${events.length})`} /></section>;
}


function UpgradeFunnelView({ interactions, search, onSearch }: { interactions: SubscriberUpgradeInteractionRecord[]; search: string; onSearch: (value: string) => void }) {
  const rows = filterUpgradeInteractions(interactions, search);
  const promptShown = interactions.filter((item) => item.interaction_type === 'prompt_shown').length;
  const promptClicked = interactions.filter((item) => item.interaction_type === 'prompt_clicked').length;
  const clickRate = promptShown ? `${Math.round((promptClicked / promptShown) * 100)}%` : '—';
  return <section className="space-y-4"><div className="flex flex-wrap items-end justify-between gap-3"><SectionTitle title="Upgrade funnel" description="Contextual upgrade prompt evidence. This records intent only; it does not activate billing or entitlements." /><Field label="Search funnel" value={search} onChange={onSearch} placeholder="email, feature, tier, route, symbol…" /></div><div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-4"><MetricTile label="Prompt impressions" value={promptShown} /><MetricTile label="Prompt clicks" value={promptClicked} /><MetricTile label="Click-through" value={clickRate} /><MetricTile label="Unique users" value={new Set(interactions.map((item) => item.subject).filter(Boolean)).size} /></div><Table title={`Upgrade interactions (${rows.length})`} headers={['When', 'User', 'Interaction', 'Feature', 'Tier path', 'Asset', 'Route']} rows={rows.map((r) => [formatDate(r.occurred_at), <IdentityCell subject={r.subject} displayName={r.subject_display_name} email={r.subject_email} />, <span className="font-mono text-[11px] text-gray-700 dark:text-gray-300">{r.interaction_type}</span>, featureLabel(r.source_feature), `${r.current_tier || '—'} → ${r.required_tier}`, r.asset_symbol || '—', <span className="break-all font-mono text-[11px] text-gray-600 dark:text-gray-400">{r.source_route || '—'}</span>])} empty="No upgrade interactions match the current scope." /></section>;
}

function filterUpgradeInteractions(interactions: SubscriberUpgradeInteractionRecord[], search: string) {
  const term = search.trim().toLowerCase();
  return interactions.filter((record) => !term || [record.occurred_at, record.subject, record.subject_email, record.subject_display_name, record.interaction_type, record.source_feature, record.source_route, record.asset_symbol, record.current_tier, record.required_tier, record.correlation_id].join(' ').toLowerCase().includes(term));
}

function featureLabel(value: string) {
  return value ? value.replace(/_/g, ' ').replace(/\b\w/g, (match) => match.toUpperCase()) : '—';
}

function UserActivityDrilldown({ snapshot, activity, selectedSubject, onSelectSubject }: { snapshot: SubscriberSubscriptionAdministrationResponse | null; activity: SubscriberUserActivityResponse | null; selectedSubject: string; onSelectSubject: (value: string) => void }) {
  const subjects = useMemo(() => {
    const entries = new Map<string, { subject: string; displayName?: string; email?: string }>();
    for (const item of snapshot?.subject_subscriptions ?? []) entries.set(item.subject, { subject: item.subject, displayName: item.subject_display_name, email: item.subject_email });
    for (const item of snapshot?.seats ?? []) entries.set(item.subject, { subject: item.subject, displayName: item.subject_display_name, email: item.subject_email });
    for (const item of activity?.summaries ?? []) entries.set(item.subject, { subject: item.subject, displayName: item.subject_display_name, email: item.subject_email });
    return [...entries.values()].filter((entry) => entry.subject).sort((a, b) => ((a.email || a.displayName || a.subject).localeCompare(b.email || b.displayName || b.subject)));
  }, [activity?.summaries, snapshot?.seats, snapshot?.subject_subscriptions]);
  const effectiveSubject = selectedSubject || subjects[0]?.subject || '';
  const summary = activity?.summaries.find((item) => item.subject === effectiveSubject) ?? null;
  const events = (activity?.events ?? []).filter((event) => event.subject === effectiveSubject);
  return <section className="space-y-3 rounded border border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-900"><div className="flex flex-wrap items-end justify-between gap-3"><SectionTitle title="Selected user activity" description="Pick an enrolled user or seat to review high-level MarketOps activity without exposing request payloads." /><Select label="User" value={effectiveSubject} onChange={onSelectSubject}>{subjects.length ? subjects.map((entry) => <option key={entry.subject} value={entry.subject}>{entry.email || entry.displayName || entry.subject}</option>) : <option value="">No users available</option>}</Select></div>{summary ? <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-5"><MetricTile label="Last activity" value={formatDate(summary.last_activity_at)} /><MetricTile label="Last login" value={formatDate(summary.last_login_at)} /><MetricTile label="Feature views" value={summary.feature_view_count} /><MetricTile label="Mutations" value={summary.mutation_count} /><MetricTile label="Failed mutations" value={summary.failed_mutation_count} /></div> : <p className="rounded border border-dashed border-gray-300 p-3 text-xs text-gray-500 dark:border-gray-700 dark:text-gray-400">No captured activity for the selected user yet.</p>}<ActivityEventTable events={events} title={`Selected user events (${events.length})`} showUser={false} /></section>;
}

function ActivityEventTable({ events, title, showUser = true }: { events: SubscriberUserActivityEventRecord[]; title: string; showUser?: boolean }) {
  const headers = showUser ? ['When', 'User', 'Event', 'Feature', 'Method', 'Route', 'Status'] : ['When', 'Event', 'Feature', 'Method', 'Route', 'Status'];
  const rows = events.map((r) => {
    const cells = [formatDate(r.occurred_at), <span className="font-mono text-[11px] text-gray-700 dark:text-gray-300">{r.event_type}</span>, r.feature_key || '—', r.http_method || '—', <span className="break-all font-mono text-[11px] text-gray-600 dark:text-gray-400">{r.route_path || '—'}</span>, r.status_code ? <StatusBadge status={r.status_code >= 400 ? 'failed' : 'succeeded'} /> : '—'];
    return showUser ? [cells[0], <IdentityCell subject={r.subject} displayName={r.subject_display_name} email={r.subject_email} />, ...cells.slice(1)] : cells;
  });
  return <Table title={title} headers={headers} rows={rows} empty="No activity events match the current scope." />;
}

function filterActivitySummaries(activity: SubscriberUserActivityResponse | null, search: string) {
  const term = search.trim().toLowerCase();
  return (activity?.summaries ?? []).filter((record) => !term || [record.subject, record.subject_email, record.subject_display_name, record.top_feature_key, record.last_activity_at, record.last_login_at].join(' ').toLowerCase().includes(term));
}

function filterActivityEvents(events: SubscriberUserActivityEventRecord[], search: string) {
  const term = search.trim().toLowerCase();
  return events.filter((record) => !term || [record.occurred_at, record.subject, record.subject_email, record.subject_display_name, record.event_type, record.feature_key, record.http_method, record.route_path, String(record.status_code), record.correlation_id].join(' ').toLowerCase().includes(term));
}

function AuditLogView({ snapshot, search, onSearch }: { snapshot: SubscriberSubscriptionAdministrationResponse | null; search: string; onSearch: (value: string) => void }) {
  const term = search.trim().toLowerCase();
  const records = (snapshot?.audit_events ?? []).filter((record) => !term || [record.occurred_at, record.actor_email, record.actor_display_name, record.actor_subject, record.subject_email, record.subject_display_name, record.subject, record.event_type, record.correlation_id, JSON.stringify(record.after_state ?? {})].join(' ').toLowerCase().includes(term));
  return <section className="space-y-2"><div className="flex flex-wrap items-end justify-between gap-3"><SectionTitle title="Audit log" description="Log-proficient governance events. Search actor, subject, event, correlation, or after-state payload." /><Field label="Search audit log" value={search} onChange={onSearch} placeholder="actor, subject, event, correlation…" /></div><Table title={`Audit trail (${records.length})`} headers={['When', 'Actor', 'Subject', 'Event', 'Correlation']} rows={records.map((r) => [formatDate(r.occurred_at), <IdentityCell subject={r.actor_subject} displayName={r.actor_display_name} email={r.actor_email} />, <IdentityCell subject={r.subject} displayName={r.subject_display_name} email={r.subject_email} />, <span className="font-mono text-[11px] text-gray-700 dark:text-gray-300">{r.event_type}</span>, <span className="break-all font-mono text-[11px] text-gray-600 dark:text-gray-400">{r.correlation_id || '—'}</span>])} empty="No subscription audit events match the search." /></section>;
}

function WebhookLedgerView({ snapshot, search, onSearch }: { snapshot: SubscriberSubscriptionAdministrationResponse | null; search: string; onSearch: (value: string) => void }) {
  const term = search.trim().toLowerCase();
  const records = (snapshot?.billing_webhook_events ?? []).filter((record) => !term || [record.received_at, record.processed_at, record.event_type, record.processing_status, record.error_message, record.provider_event_id].join(' ').toLowerCase().includes(term));
  return <section className="space-y-2"><div className="flex flex-wrap items-end justify-between gap-3"><SectionTitle title="Stripe webhook ledger" description="Operational processing evidence for signed Stripe webhook events. Search event type, status, provider event ID, or error text." /><Field label="Search webhooks" value={search} onChange={onSearch} placeholder="event, status, provider ID…" /></div><Table title={`Stripe webhook ledger (${records.length})`} headers={['Received', 'Event', 'Status', 'Provider event', 'Error']} rows={records.map((r) => [formatDate(r.received_at), r.event_type, <StatusBadge status={r.processing_status} />, <span className="break-all font-mono text-[11px] text-gray-600 dark:text-gray-400">{r.provider_event_id}</span>, r.error_message || '—'])} empty="No Stripe webhook events match the search." /></section>;
}

function IdentityCell({ subject, displayName, email }: { subject?: string; displayName?: string; email?: string }) {
  const identifier = (subject ?? '').trim();
  const emailLabel = (email ?? '').trim();
  const displayLabel = (displayName ?? '').trim();
  const labeled = Boolean(emailLabel || displayLabel);
  const suffix = identifier ? identifier.slice(-8) : '';
  const primary = emailLabel || displayLabel || (identifier ? `Unlabeled user · ${suffix}` : '—');
  return <span className="block min-w-[12rem]" title={identifier || undefined}><span className="block font-medium text-gray-900 dark:text-gray-100">{primary}</span>{labeled && displayLabel && emailLabel && displayLabel !== emailLabel ? <span className="block text-[11px] text-gray-500 dark:text-gray-400">{displayLabel}</span> : null}</span>;
}

function Table({ title, headers, rows, empty }: { title: string; headers: string[]; rows: ReactNode[][]; empty: string }) {
  return <section className="space-y-2"><h2 className="text-sm font-semibold text-gray-900 dark:text-gray-100">{title}</h2><div className="overflow-x-auto rounded border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-900"><table className="min-w-full divide-y divide-gray-200 text-left text-xs dark:divide-gray-700"><thead className="bg-gray-50 text-gray-500 dark:bg-gray-800 dark:text-gray-300"><tr>{headers.map((h) => <th key={h} className="px-2 py-1 font-medium">{h}</th>)}</tr></thead><tbody className="divide-y divide-gray-100 dark:divide-gray-800">{rows.length ? rows.map((row, idx) => <tr key={idx}>{row.map((cell, cidx) => <td key={cidx} className="px-2 py-2 align-top text-gray-700 dark:text-gray-300">{cell}</td>)}</tr>) : <tr><td colSpan={headers.length} className="px-2 py-3 text-gray-500 dark:text-gray-400">{empty}</td></tr>}</tbody></table></div></section>;
}

function ProvisionSubject(props: { subjectTenant:string; setSubjectTenant:(v:string)=>void; subject:string; setSubject:(v:string)=>void; subjectPlan:SubjectPlan; setSubjectPlan:(v:SubjectPlan)=>void; subjectStatus:SubscriptionStatus; setSubjectStatus:(v:SubscriptionStatus)=>void; onSubmit:(e:FormEvent)=>void; disabled:boolean; notice:ReactNode }) { return <form onSubmit={props.onSubmit} className="space-y-3 rounded border border-gray-200 bg-white p-3 dark:border-gray-700 dark:bg-gray-900"><SectionTitle title="Explorer or Professional subject plan" description="Use for an individual subscriber within one tenant." /><div className="grid gap-2 sm:grid-cols-2"><Field label="Tenant ID" value={props.subjectTenant} onChange={props.setSubjectTenant} /><Field label="OIDC subject" value={props.subject} onChange={props.setSubject} placeholder="Keycloak subject UUID" required /></div><div className="grid gap-2 sm:grid-cols-2"><Select label="Plan" value={props.subjectPlan} onChange={(value) => props.setSubjectPlan(value as SubjectPlan)}><option value="explorer">Explorer</option><option value="professional">Professional</option></Select><Select label="Status" value={props.subjectStatus} onChange={(value) => props.setSubjectStatus(value as SubscriptionStatus)}><StatusOptions /></Select></div><button disabled={!props.subject.trim() || props.disabled} className="rounded bg-brand-700 px-3 py-1.5 text-sm text-white disabled:opacity-50">Provision subject plan</button>{props.notice}</form>; }
function ProvisionTenant(props: { tenant:string; setTenant:(v:string)=>void; tenantStatus:SubscriptionStatus; setTenantStatus:(v:SubscriptionStatus)=>void; onSubmit:(e:FormEvent)=>void; disabled:boolean; notice:ReactNode }) { return <form onSubmit={props.onSubmit} className="space-y-3 rounded border border-gray-200 bg-white p-3 dark:border-gray-700 dark:bg-gray-900"><SectionTitle title="Institutional tenant contract" description="A tenant contract has no analytical effect until seats are assigned." /><Field label="Tenant ID" value={props.tenant} onChange={props.setTenant} /><Select label="Status" value={props.tenantStatus} onChange={(value) => props.setTenantStatus(value as SubscriptionStatus)}><StatusOptions /></Select><button disabled={!props.tenant.trim() || props.disabled} className="rounded bg-brand-700 px-3 py-1.5 text-sm text-white disabled:opacity-50">Provision Institutional contract</button>{props.notice}</form>; }
function ProvisionSeat(props: { seatTenant:string; setSeatTenant:(v:string)=>void; seatSubject:string; setSeatSubject:(v:string)=>void; seatRole:SeatRole; setSeatRole:(v:SeatRole)=>void; seatStatus:SeatStatus; setSeatStatus:(v:SeatStatus)=>void; onSubmit:(e:FormEvent)=>void; disabled:boolean; notice:ReactNode }) { return <form onSubmit={props.onSubmit} className="space-y-3 rounded border border-gray-200 bg-white p-3 dark:border-gray-700 dark:bg-gray-900 lg:col-span-2"><SectionTitle title="Institutional seat" description="An active seat inherits its tenant’s active Institutional contract." /><div className="grid gap-2 sm:grid-cols-4"><Field label="Tenant ID" value={props.seatTenant} onChange={props.setSeatTenant} /><Field label="OIDC subject" value={props.seatSubject} onChange={props.setSeatSubject} placeholder="Keycloak subject UUID" required /><Select label="Seat role" value={props.seatRole} onChange={(value) => props.setSeatRole(value as SeatRole)}><option value="member">Member</option><option value="tenant_admin">Tenant administrator</option></Select><Select label="Status" value={props.seatStatus} onChange={(value) => props.setSeatStatus(value as SeatStatus)}><option value="active">Active</option><option value="revoked">Revoked</option></Select></div><button disabled={!props.seatTenant.trim() || !props.seatSubject.trim() || props.disabled} className="rounded bg-brand-700 px-3 py-1.5 text-sm text-white disabled:opacity-50">Save Institutional seat</button>{props.notice}</form>; }
function Field({ label, value, onChange, placeholder, required = false }: { label: string; value: string; onChange: (value: string) => void; placeholder?: string; required?: boolean }) { return <label className="text-xs font-medium text-gray-700 dark:text-gray-300">{label}<input required={required} value={value} onChange={(event) => onChange(event.target.value)} placeholder={placeholder} className="mt-1 block w-full rounded border border-gray-300 bg-white px-2 py-1.5 text-sm font-normal text-gray-900 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100" /></label>; }
function NumberField({ label, value, onChange }: { label: string; value: number; onChange: (value: number) => void }) { return <label className="text-xs font-medium text-gray-700 dark:text-gray-300">{label}<input type="number" min={0} max={31} value={value} onChange={(event) => onChange(Number(event.target.value))} className="mt-1 block w-full rounded border border-gray-300 bg-white px-2 py-1.5 text-sm font-normal text-gray-900 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100" /></label>; }
function Check({ label, checked, onChange }: { label: string; checked: boolean; onChange: (value: boolean) => void }) { return <label className="flex items-center gap-2 rounded border border-gray-200 bg-white px-2 py-1.5 text-xs text-gray-700 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-300"><input type="checkbox" checked={checked} onChange={(event) => onChange(event.target.checked)} />{label}</label>; }
function Select({ label, value, onChange, children }: { label: string; value: string; onChange: (value: string) => void; children: ReactNode }) { return <label className="text-xs font-medium text-gray-700 dark:text-gray-300">{label}<select value={value} onChange={(event) => onChange(event.target.value)} className="mt-1 block w-full rounded border border-gray-300 bg-white px-2 py-1.5 text-sm font-normal text-gray-900 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100">{children}</select></label>; }
function StatusOptions() { return <><option value="active">Active</option><option value="trialing">Trialing</option><option value="past_due">Past due</option><option value="suspended">Suspended</option><option value="canceled">Canceled</option></>; }
function StatusPill({ value }: { value: string }) { return <StatusBadge status={value} />; }
function stripeCode(value?: string) { return value ? <span className="break-all font-mono text-[11px] text-gray-700 dark:text-gray-300">{value}</span> : '—'; }
function compactStripe(customer?: string, subscription?: string) { return customer || subscription ? <span className="font-mono text-[11px] text-gray-700 dark:text-gray-300">{customer || '—'}<br />{subscription || '—'}</span> : '—'; }
function formatDate(value?: string) { return value ? new Date(value).toLocaleString() : '—'; }
