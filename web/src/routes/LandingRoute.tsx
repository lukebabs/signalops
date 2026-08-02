import { useEffect } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { ArrowRight, BarChart3, Binary, ShieldCheck, Sparkles, type LucideIcon } from 'lucide-react';
import { useAppProfile } from '../apps/AppProfileContext';
import { useAuth } from '../auth/session';
import { displayIdentity } from '../auth/claims';
import syncraticPortalLogo from '../assets/syncratic-portal-logo.svg';

function iconFor(domains: string[]): LucideIcon {
  if (domains.includes('market_data')) return BarChart3;
  if (domains.includes('security')) return ShieldCheck;
  return Sparkles;
}

export function LandingRoute() {
  const { profiles, loading } = useAppProfile();
  const { claims } = useAuth();
  const navigate = useNavigate();
  const identity = displayIdentity(claims);
  useEffect(() => {
    if (loading || profiles.length !== 1) return;
    void navigate({ to: profiles[0].default_route as never, replace: true });
  }, [loading, profiles, navigate]);
  if (loading) return <div className="py-16 text-center text-sm text-gray-500">Preparing your SignalOps workspace…</div>;
  if (profiles.length === 1) return <div className="py-16 text-center text-sm text-gray-500">Opening your workspace…</div>;
  return <div className="mx-auto max-w-4xl space-y-7 py-8">
    <section className="max-w-2xl space-y-3">
      <img src={syncraticPortalLogo} alt="Syncratic" className="h-12 w-auto" />
      <div className="inline-flex items-center gap-1 rounded-full border border-brand-100 bg-brand-50 px-2.5 py-1 text-xs font-medium text-brand-700"><Binary size={13} /> Deterministic signals</div>
      <h1 className="text-3xl font-semibold tracking-tight text-gray-900">Welcome{identity ? `, ${identity}` : ''}.</h1>
      <p className="text-base leading-6 text-gray-600">SignalOps turns governed evidence into deterministic, explainable signals—so analysts can focus on what merits review.</p>
    </section>
    <section className="grid gap-3 sm:grid-cols-3" aria-label="SignalOps principles">
      <div className="rounded border border-gray-200 bg-white p-3"><div className="text-sm font-semibold text-gray-800">Deterministic</div><p className="mt-1 text-xs leading-5 text-gray-500">Versioned logic produces reproducible outcomes.</p></div>
      <div className="rounded border border-gray-200 bg-white p-3"><div className="text-sm font-semibold text-gray-800">Evidence-backed</div><p className="mt-1 text-xs leading-5 text-gray-500">Signals retain the facts and lineage behind them.</p></div>
      <div className="rounded border border-gray-200 bg-white p-3"><div className="text-sm font-semibold text-gray-800">Analyst-controlled</div><p className="mt-1 text-xs leading-5 text-gray-500">Review is explicit; insight is not a recommendation.</p></div>
    </section>
    <section className="space-y-3"><div><h2 className="text-base font-semibold text-gray-900">Your domains</h2><p className="text-sm text-gray-500">Choose a workspace available to your account.</p></div>{profiles.length === 0 ? <div className="rounded border border-gray-200 bg-white p-4 text-sm text-gray-600">No domain access has been assigned yet. Ask a SignalOps super-admin to grant MarketOps or CyberOps access.</div> : <div className="grid gap-3 md:grid-cols-2">{profiles.map((profile) => { const Icon = iconFor(profile.domains); return <button key={profile.app_id} type="button" onClick={() => void navigate({ to: profile.default_route as never })} className="group rounded border border-gray-200 bg-white p-4 text-left transition-all hover:-translate-y-0.5 hover:border-brand-300 hover:bg-brand-50 hover:shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"><div className="flex items-start justify-between gap-3"><div className="flex items-center gap-2"><span className="rounded bg-brand-50 p-2 text-brand-700 group-hover:bg-white"><Icon size={18} /></span><div><div className="font-semibold text-gray-900">{profile.label}</div><span className="text-xs text-gray-500">{profile.permission === 'write' ? 'Write access' : 'Read access'}</span></div></div><ArrowRight size={17} className="mt-1 text-gray-400 group-hover:text-brand-700" /></div><p className="mt-3 text-sm leading-5 text-gray-600">{profile.landing_summary}</p></button>; })}</div>}</section>
  </div>;
}
