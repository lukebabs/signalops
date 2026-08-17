import { useMemo, useState, type FormEvent, type ReactNode } from 'react';
import { getAccessToken, useAuth, useTenant } from '../auth/session';
import { hasSubscriptionAdministrator } from '../auth/claims';

type SubjectPlan = 'explorer' | 'professional';
type SubscriptionStatus = 'trialing' | 'active' | 'past_due' | 'suspended' | 'canceled';
type SeatRole = 'member' | 'tenant_admin';
type SeatStatus = 'active' | 'revoked';

type MutationResult = { kind: string; state: 'idle' | 'working' | 'success' | 'error'; message: string };

const requestHeaders = () => ({
  'Content-Type': 'application/json',
  ...(getAccessToken() ? { Authorization: `Bearer ${getAccessToken()}` } : {}),
});

function correlation(prefix: string) {
  return prefix + "-" + Date.now();
}

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
  const [result, setResult] = useState<MutationResult>({ kind: '', state: 'idle', message: '' });
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

  if (!allowed) return <section className="rounded border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800"><h1 className="font-semibold">Subscription Administration</h1><p className="mt-1">The platform subscription administrator role is required. MarketOps tenant roles cannot create or alter analytical subscriptions.</p></section>;

  const run = async (kind: string, fn: () => Promise<void>) => {
    setResult({ kind, state: 'working', message: 'Submitting audited provisioning change…' });
    try {
      await fn();
      setResult({ kind, state: 'success', message: 'Provisioning change recorded successfully.' });
    } catch (error) {
      setResult({ kind, state: 'error', message: error instanceof Error ? error.message : 'Provisioning failed.' });
    }
  };

  const subjectSubmit = (event: FormEvent) => {
    event.preventDefault();
    void run('subject', () => provision('/v1/administration/subscriptions/subject', {
      tenant_id: subjectTenant.trim(), subject: subject.trim(), product_key: subjectPlan, status: subjectStatus,
      correlation_id: correlation('subject-plan'),
    }));
  };
  const tenantSubmit = (event: FormEvent) => {
    event.preventDefault();
    void run('tenant', () => provision('/v1/administration/subscriptions/tenant', {
      tenant_id: tenant.trim(), product_key: 'institutional', status: tenantStatus,
      correlation_id: correlation('institutional-contract'),
    }));
  };
  const seatSubmit = (event: FormEvent) => {
    event.preventDefault();
    void run('seat', () => provision('/v1/administration/subscriptions/seats', {
      tenant_id: seatTenant.trim(), subject: seatSubject.trim(), seat_role: seatRole, status: seatStatus,
      correlation_id: correlation('institutional-seat'),
    }));
  };
  const notice = (kind: string) => result.kind === kind && result.state !== 'idle'
    ? <p role="status" className={`mt-2 text-xs ${result.state === 'error' ? 'text-red-700' : result.state === 'success' ? 'text-green-700' : 'text-gray-600'}`}>{result.message}</p> : null;

  return <div className="max-w-5xl space-y-4">
    <div><h1 className="text-lg font-semibold">Subscription Administration</h1><p className="text-xs text-gray-500">Control analytical-depth access across tenants. This workspace never creates a tenant-specific market-data copy, triggers a provider request, or changes global coverage.</p></div>
    <section className="rounded border border-blue-200 bg-blue-50 p-3 text-xs text-blue-950"><p className="font-medium">Platform control plane</p><p className="mt-1">{roleDescription}</p><p className="mt-1">Use immutable OIDC subjects, not email addresses. You can locate a subject from Administration → Access when the operator also has directory-query permission.</p></section>
    <div className="grid gap-4 lg:grid-cols-2">
      <form onSubmit={subjectSubmit} className="space-y-3 rounded border border-gray-200 bg-white p-4">
        <div><h2 className="text-sm font-semibold">Explorer or Professional subject plan</h2><p className="text-xs text-gray-500">Use for an individual subscriber within one tenant. Professional trials are calculated by the service.</p></div>
        <div className="grid gap-2 sm:grid-cols-2"><Field label="Tenant ID" value={subjectTenant} onChange={setSubjectTenant} /><Field label="OIDC subject" value={subject} onChange={setSubject} placeholder="Keycloak subject UUID" required /></div>
        <div className="grid gap-2 sm:grid-cols-2"><Select label="Plan" value={subjectPlan} onChange={(value) => { const plan = value as SubjectPlan; setSubjectPlan(plan); if (plan === 'explorer' && subjectStatus === 'trialing') setSubjectStatus('active'); }}><option value="explorer">Explorer</option><option value="professional">Professional</option></Select><Select label="Status" value={subjectStatus} onChange={(value) => setSubjectStatus(value as SubscriptionStatus)}><option value="active">Active</option><option value="trialing" disabled={subjectPlan === 'explorer'}>Trialing</option><StatusOptions /></Select></div>
        <button disabled={!subject.trim() || result.state === 'working'} className="rounded bg-brand-700 px-3 py-1.5 text-sm text-white disabled:opacity-50">Provision subject plan</button>{notice('subject')}
      </form>
      <form onSubmit={tenantSubmit} className="space-y-3 rounded border border-gray-200 bg-white p-4">
        <div><h2 className="text-sm font-semibold">Institutional tenant contract</h2><p className="text-xs text-gray-500">A tenant contract has no analytical effect until seats are assigned below.</p></div>
        <Field label="Tenant ID" value={tenant} onChange={setTenant} />
        <Select label="Status" value={tenantStatus} onChange={(value) => setTenantStatus(value as SubscriptionStatus)}><StatusOptions /></Select>
        <button disabled={!tenant.trim() || result.state === 'working'} className="rounded bg-brand-700 px-3 py-1.5 text-sm text-white disabled:opacity-50">Provision Institutional contract</button>{notice('tenant')}
      </form>
      <form onSubmit={seatSubmit} className="space-y-3 rounded border border-gray-200 bg-white p-4 lg:col-span-2">
        <div><h2 className="text-sm font-semibold">Institutional seat</h2><p className="text-xs text-gray-500">An active seat inherits its tenant’s active Institutional contract. Revoking a seat does not delete the contract or audit history.</p></div>
        <div className="grid gap-2 sm:grid-cols-4"><Field label="Tenant ID" value={seatTenant} onChange={setSeatTenant} /><Field label="OIDC subject" value={seatSubject} onChange={setSeatSubject} placeholder="Keycloak subject UUID" required /><Select label="Seat role" value={seatRole} onChange={(value) => setSeatRole(value as SeatRole)}><option value="member">Member</option><option value="tenant_admin">Tenant administrator</option></Select><Select label="Status" value={seatStatus} onChange={(value) => setSeatStatus(value as SeatStatus)}><option value="active">Active</option><option value="revoked">Revoked</option></Select></div>
        <button disabled={!seatTenant.trim() || !seatSubject.trim() || result.state === 'working'} className="rounded bg-brand-700 px-3 py-1.5 text-sm text-white disabled:opacity-50">Save Institutional seat</button>{notice('seat')}
      </form>
    </div>
    <section className="rounded border border-gray-200 bg-white p-3 text-xs text-gray-600"><h2 className="font-semibold text-gray-800">Activation safeguard</h2><p className="mt-1">Provisioning records alone do not restrict current users. Commercial feature enforcement stays disabled until Explorer, Professional, and Institutional acceptance tests are retained, then the gateway feature flag is approved separately.</p></section>
  </div>;
}

function Field({ label, value, onChange, placeholder, required = false }: { label: string; value: string; onChange: (value: string) => void; placeholder?: string; required?: boolean }) {
  return <label className="text-xs font-medium text-gray-700">{label}<input required={required} value={value} onChange={(event) => onChange(event.target.value)} placeholder={placeholder} className="mt-1 block w-full rounded border border-gray-300 px-2 py-1.5 text-sm font-normal" /></label>;
}
function Select({ label, value, onChange, children }: { label: string; value: string; onChange: (value: string) => void; children: ReactNode }) {
  return <label className="text-xs font-medium text-gray-700">{label}<select value={value} onChange={(event) => onChange(event.target.value)} className="mt-1 block w-full rounded border border-gray-300 bg-white px-2 py-1.5 text-sm font-normal">{children}</select></label>;
}
function StatusOptions() {
  return <><option value="active">Active</option><option value="trialing">Trialing</option><option value="past_due">Past due</option><option value="suspended">Suspended</option><option value="canceled">Canceled</option></>;
}
