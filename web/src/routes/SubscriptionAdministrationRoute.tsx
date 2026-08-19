import { useEffect, useMemo, useState, type FormEvent, type ReactNode } from 'react';
import { api } from '../api/client';
import { getAccessToken, useAuth, useTenant } from '../auth/session';
import { hasSubscriptionAdministrator } from '../auth/claims';
import type { SubscriberSubscriptionAdministrationResponse, SubscriberSubscriptionFeature, SubscriberSubscriptionProduct } from '../types';

type SubjectPlan = 'explorer' | 'professional';
type SubscriptionStatus = 'trialing' | 'active' | 'past_due' | 'suspended' | 'canceled';
type SeatRole = 'member' | 'tenant_admin';
type SeatStatus = 'active' | 'revoked';
type MutationResult = { kind: string; state: 'idle' | 'working' | 'success' | 'error'; message: string };

const features: Array<{ key: SubscriberSubscriptionFeature; label: string; depth: 'Explorer' | 'Professional' | 'Institutional' }> = [
  { key: 'market_dashboards', label: 'Market dashboards', depth: 'Explorer' },
  { key: 'public_signals', label: 'Public signals', depth: 'Explorer' },
  { key: 'sector_rotation_discovery', label: 'Sector rotation discovery', depth: 'Explorer' },
  { key: 'value_intelligence', label: 'Value Intelligence', depth: 'Professional' },
  { key: 'distressed_opportunity_intelligence', label: 'Distressed Opportunity Intelligence', depth: 'Professional' },
  { key: 'earnings_opportunity_intelligence', label: 'Earnings Opportunity Intelligence', depth: 'Professional' },
  { key: 'sector_rotation_detail', label: 'Sector Rotation Intelligence detail', depth: 'Professional' },
  { key: 'options_signals', label: 'Options signals', depth: 'Professional' },
  { key: 'earnings_calendar', label: 'Earnings calendar', depth: 'Professional' },
  { key: 'research_reports', label: 'Research reports', depth: 'Professional' },
  { key: 'signal_assurance_analytics', label: 'Signal Assurance analytics', depth: 'Institutional' },
  { key: 'portfolio_analysis', label: 'Portfolio analysis', depth: 'Institutional' },
  { key: 'batch_screening', label: 'Batch screening', depth: 'Institutional' },
  { key: 'historical_replay', label: 'Historical replay', depth: 'Institutional' },
  { key: 'strategy_validation', label: 'Strategy validation', depth: 'Institutional' },
  { key: 'custom_universes', label: 'Custom universes', depth: 'Institutional' },
  { key: 'api', label: 'APIs', depth: 'Institutional' },
  { key: 'white_label', label: 'White-label deployment', depth: 'Institutional' },
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
  const [subjectPlan, setSubjectPlan] = useState<SubjectPlan>('explorer');
  const [subjectStatus, setSubjectStatus] = useState<SubscriptionStatus>('active');
  const [subjectTenant, setSubjectTenant] = useState(currentTenant);
  const [subject, setSubject] = useState('');
  const [tenant, setTenant] = useState(currentTenant);
  const [tenantStatus, setTenantStatus] = useState<SubscriptionStatus>('active');
  const [seatTenant, setSeatTenant] = useState(currentTenant);
  const [seatSubject, setSeatSubject] = useState('');
  const [seatRole, setSeatRole] = useState<SeatRole>('member');
  const [seatStatus, setSeatStatus] = useState<SeatStatus>('active');

  const roleDescription = useMemo(() => authEnabled
    ? 'The signed signalops:subscription_admin role is required. Every change is persisted with the signed actor and correlation identifier.'
    : 'Authentication is disabled in this environment; production requires the signed platform subscription-admin role.', [authEnabled]);

  const selectedProduct = snapshot?.products.find((product) => product.product_key === editingProductKey) ?? snapshot?.products[0] ?? null;

  const refresh = async (tenantId = tenantFilter) => {
    if (!allowed || !tenantId.trim()) return;
    setLoading(true); setLoadError('');
    try {
      const data = await api.getSubscriberSubscriptionAdministration(tenantId.trim());
      setSnapshot(data);
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
  const notice = (kind: string) => result.kind === kind && result.state !== 'idle' ? <p role="status" className={`mt-2 text-xs ${result.state === 'error' ? 'text-red-700' : result.state === 'success' ? 'text-green-700' : 'text-gray-600'}`}>{result.message}</p> : null;

  return <div className="max-w-7xl space-y-4">
    <div className="flex flex-wrap items-end justify-between gap-3"><div><h1 className="text-lg font-semibold">Subscription Administration</h1><p className="text-xs text-gray-500">Govern enrolled users, tenant contracts, seats, analytical-depth tiers, entitlement policy, limits, and audit evidence. This page never triggers provider polling or creates tenant-specific market-data copies.</p></div><div className="flex items-end gap-2"><Field label="Tenant filter" value={tenantFilter} onChange={setTenantFilter} /><button onClick={() => void refresh()} disabled={loading || !tenantFilter.trim()} className="rounded border border-gray-300 px-3 py-1.5 text-sm disabled:opacity-50">{loading ? 'Loading…' : 'Refresh'}</button></div></div>
    <section className="rounded border border-blue-200 bg-blue-50 p-3 text-xs text-blue-950"><p className="font-medium">Platform control plane</p><p className="mt-1">{roleDescription}</p><p className="mt-1">Use immutable OIDC subjects, not email addresses. Tenant-level MarketOps roles do not grant subscription administration authority.</p></section>
    {loadError ? <section className="rounded border border-red-200 bg-red-50 p-3 text-xs text-red-800">{loadError}</section> : null}
    <Summary snapshot={snapshot} />
    <section className="rounded border border-gray-200 bg-white p-4"><h2 className="text-sm font-semibold">Tier policy and entitlement alignment</h2><p className="mt-1 text-xs text-gray-500">Feature policy is the server-side entitlement contract used by subscription enforcement. Updating a product increments its revision and writes audit evidence.</p><div className="mt-3 grid gap-3 lg:grid-cols-3">{snapshot?.products.map((product) => <button key={product.product_key} onClick={() => seedProductEditor(product)} className={`rounded border p-3 text-left text-xs ${editingProductKey === product.product_key ? 'border-brand-600 bg-brand-50' : 'border-gray-200 bg-white'}`}><div className="flex items-center justify-between"><span className="text-sm font-semibold">{product.display_name}</span><StatusPill value={product.active === false ? 'inactive' : 'active'} /></div><p className="mt-1 text-gray-500">{product.billing_scope} · revision {product.revision} · trial {product.trial_days} days</p><p className="mt-2 text-gray-700">{Object.entries(product.feature_policy ?? {}).filter(([, enabled]) => enabled).length} enabled features · limits {Object.keys(product.limit_policy ?? {}).length}</p></button>)}</div>{selectedProduct ? <form onSubmit={productSubmit} className="mt-4 space-y-3 rounded border border-gray-200 p-3"><div className="grid gap-2 sm:grid-cols-4"><Field label="Display name" value={displayDraft} onChange={setDisplayDraft} required /><NumberField label="Trial days" value={trialDraft} onChange={setTrialDraft} /><Check label="Free tier" checked={freeDraft} onChange={setFreeDraft} /><Check label="Product active" checked={activeDraft} onChange={setActiveDraft} /></div><div className="grid gap-2 md:grid-cols-3">{features.map((feature) => <Check key={feature.key} label={`${feature.label} (${feature.depth})`} checked={Boolean(featureDraft[feature.key])} onChange={(checked) => setFeatureDraft((current) => ({ ...current, [feature.key]: checked }))} />)}</div><label className="block text-xs font-medium text-gray-700">Limit policy JSON<textarea value={limitDraft} onChange={(event) => setLimitDraft(event.target.value)} rows={4} className="mt-1 block w-full rounded border border-gray-300 px-2 py-1.5 font-mono text-xs" /></label><button disabled={!editingProductKey || result.state === 'working'} className="rounded bg-brand-700 px-3 py-1.5 text-sm text-white disabled:opacity-50">Save tier policy</button>{notice('product')}</form> : null}</section>
    <section className="grid gap-4 lg:grid-cols-2"><ProvisionSubject subjectTenant={subjectTenant} setSubjectTenant={setSubjectTenant} subject={subject} setSubject={setSubject} subjectPlan={subjectPlan} setSubjectPlan={setSubjectPlan} subjectStatus={subjectStatus} setSubjectStatus={setSubjectStatus} onSubmit={subjectSubmit} disabled={result.state === 'working'} notice={notice('subject')} /><ProvisionTenant tenant={tenant} setTenant={setTenant} tenantStatus={tenantStatus} setTenantStatus={setTenantStatus} onSubmit={tenantSubmit} disabled={result.state === 'working'} notice={notice('tenant')} /><ProvisionSeat seatTenant={seatTenant} setSeatTenant={setSeatTenant} seatSubject={seatSubject} setSeatSubject={setSeatSubject} seatRole={seatRole} setSeatRole={setSeatRole} seatStatus={seatStatus} setSeatStatus={setSeatStatus} onSubmit={seatSubmit} disabled={result.state === 'working'} notice={notice('seat')} /></section>
    <GovernanceTables snapshot={snapshot} />
  </div>;
}

function Summary({ snapshot }: { snapshot: SubscriberSubscriptionAdministrationResponse | null }) {
  const activeSubjects = snapshot?.subject_subscriptions.filter((record) => ['active', 'trialing'].includes(record.status)).length ?? 0;
  const activeSeats = snapshot?.seats.filter((record) => record.status === 'active').length ?? 0;
  return <div className="grid gap-3 sm:grid-cols-4"><Metric label="Products" value={snapshot?.products.length ?? 0} /><Metric label="Subject subscribers" value={activeSubjects} /><Metric label="Institutional seats" value={activeSeats} /><Metric label="Audit events" value={snapshot?.audit_events.length ?? 0} /></div>;
}
function Metric({ label, value }: { label: string; value: number }) { return <div className="rounded border border-gray-200 bg-white p-3"><p className="text-xs text-gray-500">{label}</p><p className="text-2xl font-semibold">{value}</p></div>; }
function GovernanceTables({ snapshot }: { snapshot: SubscriberSubscriptionAdministrationResponse | null }) {
  return <div className="grid gap-4 xl:grid-cols-2"><Table title="Subject subscriptions" headers={['Subject', 'Plan', 'Status', 'Updated']} rows={(snapshot?.subject_subscriptions ?? []).map((r) => [r.subject, r.display_name, <StatusPill value={r.status} />, formatDate(r.updated_at)])} empty="No subject subscriptions for this tenant." /><Table title="Institutional contracts" headers={['Tenant', 'Plan', 'Status', 'Updated']} rows={(snapshot?.tenant_subscriptions ?? []).map((r) => [r.tenant_id, r.display_name, <StatusPill value={r.status} />, formatDate(r.updated_at)])} empty="No institutional contract for this tenant." /><Table title="Institutional seats" headers={['Subject', 'Role', 'Status', 'Assigned']} rows={(snapshot?.seats ?? []).map((r) => [r.subject, r.seat_role, <StatusPill value={r.status} />, formatDate(r.assigned_at)])} empty="No seats for this tenant." /><Table title="Audit trail" headers={['When', 'Actor', 'Event', 'Correlation']} rows={(snapshot?.audit_events ?? []).map((r) => [formatDate(r.occurred_at), r.actor_subject, r.event_type, r.correlation_id])} empty="No subscription audit events for this tenant." /></div>;
}
function Table({ title, headers, rows, empty }: { title: string; headers: string[]; rows: ReactNode[][]; empty: string }) { return <section className="rounded border border-gray-200 bg-white p-4"><h2 className="text-sm font-semibold">{title}</h2><div className="mt-2 overflow-x-auto"><table className="min-w-full text-left text-xs"><thead><tr>{headers.map((h) => <th key={h} className="border-b px-2 py-1 text-gray-500">{h}</th>)}</tr></thead><tbody>{rows.length ? rows.map((row, idx) => <tr key={idx}>{row.map((cell, cidx) => <td key={cidx} className="border-b px-2 py-1 align-top">{cell}</td>)}</tr>) : <tr><td colSpan={headers.length} className="px-2 py-3 text-gray-500">{empty}</td></tr>}</tbody></table></div></section>; }
function ProvisionSubject(props: { subjectTenant:string; setSubjectTenant:(v:string)=>void; subject:string; setSubject:(v:string)=>void; subjectPlan:SubjectPlan; setSubjectPlan:(v:SubjectPlan)=>void; subjectStatus:SubscriptionStatus; setSubjectStatus:(v:SubscriptionStatus)=>void; onSubmit:(e:FormEvent)=>void; disabled:boolean; notice:ReactNode }) { return <form onSubmit={props.onSubmit} className="space-y-3 rounded border border-gray-200 bg-white p-4"><div><h2 className="text-sm font-semibold">Explorer or Professional subject plan</h2><p className="text-xs text-gray-500">Use for an individual subscriber within one tenant.</p></div><div className="grid gap-2 sm:grid-cols-2"><Field label="Tenant ID" value={props.subjectTenant} onChange={props.setSubjectTenant} /><Field label="OIDC subject" value={props.subject} onChange={props.setSubject} placeholder="Keycloak subject UUID" required /></div><div className="grid gap-2 sm:grid-cols-2"><Select label="Plan" value={props.subjectPlan} onChange={(value) => props.setSubjectPlan(value as SubjectPlan)}><option value="explorer">Explorer</option><option value="professional">Professional</option></Select><Select label="Status" value={props.subjectStatus} onChange={(value) => props.setSubjectStatus(value as SubscriptionStatus)}><StatusOptions /></Select></div><button disabled={!props.subject.trim() || props.disabled} className="rounded bg-brand-700 px-3 py-1.5 text-sm text-white disabled:opacity-50">Provision subject plan</button>{props.notice}</form>; }
function ProvisionTenant(props: { tenant:string; setTenant:(v:string)=>void; tenantStatus:SubscriptionStatus; setTenantStatus:(v:SubscriptionStatus)=>void; onSubmit:(e:FormEvent)=>void; disabled:boolean; notice:ReactNode }) { return <form onSubmit={props.onSubmit} className="space-y-3 rounded border border-gray-200 bg-white p-4"><div><h2 className="text-sm font-semibold">Institutional tenant contract</h2><p className="text-xs text-gray-500">A tenant contract has no analytical effect until seats are assigned.</p></div><Field label="Tenant ID" value={props.tenant} onChange={props.setTenant} /><Select label="Status" value={props.tenantStatus} onChange={(value) => props.setTenantStatus(value as SubscriptionStatus)}><StatusOptions /></Select><button disabled={!props.tenant.trim() || props.disabled} className="rounded bg-brand-700 px-3 py-1.5 text-sm text-white disabled:opacity-50">Provision Institutional contract</button>{props.notice}</form>; }
function ProvisionSeat(props: { seatTenant:string; setSeatTenant:(v:string)=>void; seatSubject:string; setSeatSubject:(v:string)=>void; seatRole:SeatRole; setSeatRole:(v:SeatRole)=>void; seatStatus:SeatStatus; setSeatStatus:(v:SeatStatus)=>void; onSubmit:(e:FormEvent)=>void; disabled:boolean; notice:ReactNode }) { return <form onSubmit={props.onSubmit} className="space-y-3 rounded border border-gray-200 bg-white p-4 lg:col-span-2"><div><h2 className="text-sm font-semibold">Institutional seat</h2><p className="text-xs text-gray-500">An active seat inherits its tenant’s active Institutional contract.</p></div><div className="grid gap-2 sm:grid-cols-4"><Field label="Tenant ID" value={props.seatTenant} onChange={props.setSeatTenant} /><Field label="OIDC subject" value={props.seatSubject} onChange={props.setSeatSubject} placeholder="Keycloak subject UUID" required /><Select label="Seat role" value={props.seatRole} onChange={(value) => props.setSeatRole(value as SeatRole)}><option value="member">Member</option><option value="tenant_admin">Tenant administrator</option></Select><Select label="Status" value={props.seatStatus} onChange={(value) => props.setSeatStatus(value as SeatStatus)}><option value="active">Active</option><option value="revoked">Revoked</option></Select></div><button disabled={!props.seatTenant.trim() || !props.seatSubject.trim() || props.disabled} className="rounded bg-brand-700 px-3 py-1.5 text-sm text-white disabled:opacity-50">Save Institutional seat</button>{props.notice}</form>; }
function Field({ label, value, onChange, placeholder, required = false }: { label: string; value: string; onChange: (value: string) => void; placeholder?: string; required?: boolean }) { return <label className="text-xs font-medium text-gray-700">{label}<input required={required} value={value} onChange={(event) => onChange(event.target.value)} placeholder={placeholder} className="mt-1 block w-full rounded border border-gray-300 px-2 py-1.5 text-sm font-normal" /></label>; }
function NumberField({ label, value, onChange }: { label: string; value: number; onChange: (value: number) => void }) { return <label className="text-xs font-medium text-gray-700">{label}<input type="number" min={0} max={31} value={value} onChange={(event) => onChange(Number(event.target.value))} className="mt-1 block w-full rounded border border-gray-300 px-2 py-1.5 text-sm font-normal" /></label>; }
function Check({ label, checked, onChange }: { label: string; checked: boolean; onChange: (value: boolean) => void }) { return <label className="flex items-center gap-2 rounded border border-gray-200 px-2 py-1.5 text-xs text-gray-700"><input type="checkbox" checked={checked} onChange={(event) => onChange(event.target.checked)} />{label}</label>; }
function Select({ label, value, onChange, children }: { label: string; value: string; onChange: (value: string) => void; children: ReactNode }) { return <label className="text-xs font-medium text-gray-700">{label}<select value={value} onChange={(event) => onChange(event.target.value)} className="mt-1 block w-full rounded border border-gray-300 bg-white px-2 py-1.5 text-sm font-normal">{children}</select></label>; }
function StatusOptions() { return <><option value="active">Active</option><option value="trialing">Trialing</option><option value="past_due">Past due</option><option value="suspended">Suspended</option><option value="canceled">Canceled</option></>; }
function StatusPill({ value }: { value: string }) { return <span className={`inline-flex rounded-full px-2 py-0.5 text-[11px] font-medium ${value === 'active' || value === 'trialing' ? 'bg-green-100 text-green-800' : value === 'revoked' || value === 'canceled' || value === 'inactive' ? 'bg-gray-100 text-gray-700' : 'bg-amber-100 text-amber-800'}`}>{value}</span>; }
function formatDate(value?: string) { return value ? new Date(value).toLocaleString() : '—'; }
