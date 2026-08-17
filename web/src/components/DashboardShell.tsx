import { Suspense } from 'react';
import { Link, Outlet, useLocation, useNavigate } from '@tanstack/react-router';
import {
  Activity,
  CircleDollarSign,
  ListTree,
  Database,
  KeyRound,
  Gauge,
  DatabaseZap,
  Workflow,
  ShieldCheck,
  LayoutDashboard,
  FileCheck2,
  Radar,
  TriangleAlert,
  Lightbulb,
  LogOut,
  History,
  Network,
  Sparkles,
  Monitor,
  Telescope,
  LineChart,
  LockKeyhole,
  type LucideIcon,
} from 'lucide-react';
import { HealthIndicator } from './HealthIndicator';
import { useAuth } from '../auth/session';
import { displayIdentity } from '../auth/claims';
import { useTheme, type ThemePreference } from '../theme/theme';
import { useAppProfile } from '../apps/AppProfileContext';
import { defaultRouteForApp } from '../apps/appRouting';
import syncraticPortalLogo from '../assets/syncratic-portal-logo.svg';
import type { AppProfile } from '../types';
import { useSubscription } from '../subscriber/SubscriptionContext';

const navItem =
  'inline-flex items-center gap-1 whitespace-nowrap border-b-2 border-transparent px-3 py-2 text-sm text-gray-600 hover:bg-gray-50';
const navItemActive = 'text-brand-700 border-brand-500';

// One icon per nav module. Keys are the `module` strings used by appRouting's
// navForApp maps (console + marketops), so both apps resolve an icon.
const MODULE_ICONS: Record<string, LucideIcon> = {
  dashboard: LayoutDashboard,
  runs: ListTree,
  raw_events: Database,
  normalized: FileCheck2,
  idempotency: KeyRound,
  sources: DatabaseZap,
  providers: DatabaseZap,
  symbols: CircleDollarSign,
  watchlists: ListTree,
  pipelines: Workflow,
  rules: ShieldCheck,
  replay: History,
  signals: Radar,
  alerts: TriangleAlert,
  insights: Lightbulb,
  health: Gauge,
  dsm: Network,
  valuation: LineChart,
  syncratic: Sparkles,
  opportunities: Telescope,
  market_state: LineChart,
  hypotheses: Lightbulb,
  indicator_reel: Radar,
  access: ShieldCheck,
  settings: ShieldCheck,
};

export function DashboardShell() {
  const { authEnabled, claims, signOut } = useAuth();
  const { preference, setPreference } = useTheme();
  const identity = authEnabled ? displayIdentity(claims) : undefined;
  const { profiles, currentApp, currentAppId, nav, superAdmin } = useAppProfile();
  const { subscription, known: subscriptionKnown, allows } = useSubscription();
  const location = useLocation();
  const isLanding = location.pathname === "/";
  const navigate = useNavigate();

  function selectApp(appId: string) {
    const profile = profiles.find((p: AppProfile) => p.app_id === appId);
    if (!profile || profile.app_id === currentAppId) return;
    void navigate({ to: defaultRouteForApp(profile) });
  }

  return (
    <div className="min-h-screen bg-gray-50 text-gray-900">
      <header className="flex flex-wrap items-center justify-between gap-2 border-b border-gray-200 bg-white px-4 py-2">
        <Link to="/" className="signalops-home-link flex items-center gap-2 rounded px-1 py-0.5 hover:bg-gray-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500" aria-label="SignalOps home">
          <img src={syncraticPortalLogo} alt="" className="h-7 w-auto" />
          <span className="text-sm font-semibold">SignalOps</span>
        </Link>
        <div className="flex flex-wrap items-center gap-3">
          {/* Active app label + selector. The select both displays the active app
              label and switches apps by navigating to the profile default route. */}
          {!isLanding && profiles.length > 0 ? <select
            value={currentAppId}
            onChange={(e) => selectApp(e.target.value)}
            aria-label="Active app"
            title="Switch app"
            className="rounded border border-gray-300 bg-white px-2 py-1 text-xs text-gray-700 hover:bg-gray-50"
          >
            {profiles.map((p) => (
              <option key={p.app_id} value={p.app_id}>
                {p.label}
              </option>
            ))}
          </select> : null}
          <label className="inline-flex items-center gap-1" title="Color theme">
            <Monitor size={14} className="text-gray-500" aria-hidden="true" />
            <span className="sr-only">Color theme</span>
            <select
              value={preference}
              onChange={(event) => setPreference(event.target.value as ThemePreference)}
              aria-label="Color theme"
              className="rounded border border-gray-300 bg-white px-2 py-1 text-xs text-gray-700 hover:bg-gray-50"
            >
              <option value="system">System</option>
              <option value="light">Light</option>
              <option value="dark">Dark</option>
            </select>
          </label>
          <HealthIndicator />
          {superAdmin && <Link to="/admin/dashboard" className="inline-flex items-center gap-1 rounded border border-gray-300 bg-white px-2 py-1 text-xs text-gray-700 hover:bg-gray-50"><ShieldCheck size={14} /> Administration</Link>}
          {currentAppId === 'marketops' && subscriptionKnown && (
            <span className="rounded bg-brand-50 px-2 py-1 text-xs font-medium text-brand-700" title="Current MarketOps analytical plan">
              {subscription?.display_name ?? "Subscription required"}
            </span>
          )}
          {identity && (
            <div className="flex items-center gap-2">
              <span className="text-xs text-gray-600">{identity}</span>
              <button
                type="button"
                onClick={() => void signOut()}
                aria-label="Sign out"
                title="Sign out"
                className="inline-flex items-center gap-1 rounded border border-gray-300 bg-white px-2 py-1 text-xs text-gray-700 hover:bg-gray-50"
              >
                <LogOut size={14} /> Sign out
              </button>
            </div>
          )}
        </div>
      </header>
      {!isLanding && <nav className="flex flex-wrap gap-1 border-b border-gray-200 bg-white px-2">
        {nav.map((item) => {
          const Icon = MODULE_ICONS[item.module] ?? Activity;
          const locked = !allows(item.subscriptionFeature);
          if (locked) return (
            <span key={item.to} title="Requires an included MarketOps subscription" className={` cursor-not-allowed text-gray-400`} aria-disabled="true">
              <LockKeyhole size={14} /> {item.label}
            </span>
          );
          return (
            <Link key={item.to} to={item.to} className={navItem} activeProps={{ className: navItemActive }}>
              <Icon size={14} /> {item.label}
            </Link>
          );
        })}
      </nav>}
      <main className="p-4">
        <Suspense
          fallback={<div className="p-4 text-sm text-gray-500">Loading view…</div>}
        >
          <Outlet />
        </Suspense>
      </main>
    </div>
  );
}
