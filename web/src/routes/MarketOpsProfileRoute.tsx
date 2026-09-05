import { Link } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { CheckCircle2, CircleAlert, LogOut } from 'lucide-react';
import { api } from '../api/client';
import { displayIdentity, rolesFromClaims } from '../auth/claims';
import { useAuth, useTenant } from '../auth/session';
import { useSubscription } from '../subscriber/SubscriptionContext';
import { BillingSupportPanel, featureName } from '../subscriber/SubscriptionPlanCards';
import type { SubscriberSubscriptionFeature } from '../types';

const featureLabels: Partial<Record<SubscriberSubscriptionFeature, string>> = {
  market_dashboards: 'Market Dashboards',
  public_signals: 'Public Signals',
  sector_rotation_discovery: 'Sector Rotation Discovery',
  sector_rotation_detail: 'Sector Rotation Details',
  value_intelligence: 'Value Intelligence',
  distressed_opportunity_intelligence: 'Distressed Opportunity Intelligence',
  earnings_opportunity_intelligence: 'Earnings Opportunity Intelligence',
  options_signals: 'Options Signals',
  signal_assurance_analytics: 'Signal Assurance Analytics',
  syncratic_explainability: 'Syncratic Explainability',
  historical_replay: 'Historical Replay',
};

export function MarketOpsProfileRoute() {
  const auth = useAuth();
  const tenantId = useTenant();
  const subscription = useSubscription();
  const enrollmentQ = useQuery({ queryKey: ['session', 'enrollment', 'profile'], queryFn: api.getSessionEnrollment, staleTime: 30_000, retry: false });
  const watchlistQ = useQuery({ queryKey: ['subscriber-watchlist-context', tenantId, 'profile'], queryFn: () => api.getSubscriberWatchlistContext(tenantId), enabled: Boolean(tenantId), staleTime: 60_000, retry: false });
  const claims = auth.claims;
  const active = subscription.subscription;
  const featurePolicy = active?.feature_policy ?? {};
  const limitPolicy = active?.limit_policy ?? {};
  const subject = claims?.sub || enrollmentQ.data?.subject || '';
  const subjectSuffix = subject ? subject.slice(-8) : '—';

  return <div className="mx-auto max-w-5xl space-y-5">
    <section className="rounded border border-brand-100 bg-brand-50 p-5 dark:border-brand-900 dark:bg-brand-950/30">
      <p className="text-xs font-semibold uppercase tracking-wide text-brand-700 dark:text-brand-200">MarketOps profile</p>
      <div className="mt-2 flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold text-gray-950 dark:text-gray-50">MarketOps profile</h1>
          <p className="mt-1 text-sm text-gray-700 dark:text-gray-200">{displayIdentity(claims) || enrollmentQ.data?.display_name || 'MarketOps user'} · your identity, enrollment, and subscription access in one place.</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Link to="/marketops/settings" search={{ tab: 'subscription' }} className="rounded bg-brand-600 px-3 py-2 text-sm font-medium text-white hover:bg-brand-700">Upgrade package</Link>
          <button type="button" onClick={() => void auth.signOut()} className="inline-flex items-center gap-2 rounded border border-gray-300 bg-white px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-100"><LogOut size={15} /> Sign out</button>
        </div>
      </div>
    </section>

    <section className="grid gap-4 lg:grid-cols-3">
      <InfoCard title="Identity" rows={[
        ['Email', claims?.email || enrollmentQ.data?.email || '—'],
        ['Tenant', tenantId],
        ['Subject', `…${subjectSuffix}`],
        ['Email verified', enrollmentQ.data?.email_verified ? 'Yes' : 'No'],
      ]} />
      <InfoCard title="Enrollment" rows={[
        ['State', enrollmentQ.data?.state || (enrollmentQ.isLoading ? 'Loading…' : 'Unknown')],
        ['MarketOps access', enrollmentQ.data?.access?.marketops || '—'],
        ['Self-enrollment', enrollmentQ.data?.self_enrollment?.eligible ? 'Eligible' : 'Not active'],
        ['Roles', rolesFromClaims(claims).filter((role) => role.startsWith('signalops:') || role === 'super_admin').join(', ') || '—'],
      ]} />
      <InfoCard title="Subscription" rows={[
        ['Tier', active?.display_name || 'Unprovisioned'],
        ['Status', active?.status || '—'],
        ['Source', active?.source === 'tenant_seat' ? 'Institutional seat' : active?.source || '—'],
        ['Seat role', active?.seat_role || '—'],
      ]} />
    </section>

    <section className="grid gap-4 lg:grid-cols-2">
      <section className="rounded border border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-900">
        <h2 className="text-sm font-semibold text-gray-950 dark:text-gray-50">Feature access</h2>
        <div className="mt-3 grid gap-2 sm:grid-cols-2">{Object.entries(featureLabels).map(([key, label]) => {
          const allowed = (key in featurePolicy && featurePolicy[key as SubscriberSubscriptionFeature] === true) || (!subscription.enforcementEnabled && ['market_dashboards', 'public_signals', 'sector_rotation_discovery'].includes(key));
          return <div key={key} className="flex items-center gap-2 rounded border border-gray-200 px-3 py-2 text-xs text-gray-700 dark:border-gray-700 dark:text-gray-200">{allowed ? <CheckCircle2 size={15} className="text-emerald-600" /> : <CircleAlert size={15} className="text-gray-400" />}<span>{label}</span></div>;
        })}</div>
      </section>
      <section className="rounded border border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-900">
        <h2 className="text-sm font-semibold text-gray-950 dark:text-gray-50">Watchlist and limits</h2>
        <dl className="mt-3 space-y-2 text-xs text-gray-700 dark:text-gray-200">
          <Row label="Selected watchlist" value={watchlistQ.data?.list_name || (watchlistQ.isLoading ? 'Loading…' : 'Not selected')} />
          <Row label="Selection mode" value={watchlistQ.data?.selection_mode || '—'} />
          {Object.keys(limitPolicy).length ? Object.entries(limitPolicy).map(([key, value]) => <Row key={key} label={featureName(key)} value={String(value)} />) : <Row label="Limits" value="No explicit limits returned" />}
        </dl>
        <Link to="/marketops/watchlists" className="mt-4 inline-flex rounded border border-gray-300 px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-100">Manage watchlists</Link>
      </section>
    </section>

    <BillingSupportPanel tenantId={tenantId} subscription={active} />
  </div>;
}

function InfoCard({ title, rows }: { title: string; rows: Array<[string, string]> }) {
  return <section className="rounded border border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-900"><h2 className="text-sm font-semibold text-gray-950 dark:text-gray-50">{title}</h2><dl className="mt-3 space-y-2 text-xs text-gray-700 dark:text-gray-200">{rows.map(([label, value]) => <Row key={label} label={label} value={value} />)}</dl></section>;
}

function Row({ label, value }: { label: string; value: string }) {
  return <div className="flex justify-between gap-3"><dt className="text-gray-500 dark:text-gray-400">{label}</dt><dd className="text-right font-medium text-gray-900 dark:text-gray-100">{value}</dd></div>;
}
