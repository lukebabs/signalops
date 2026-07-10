import { Suspense } from 'react';
import { Link, Outlet, useNavigate } from '@tanstack/react-router';
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
  type LucideIcon,
} from 'lucide-react';
import { HealthIndicator } from './HealthIndicator';
import { useAuth } from '../auth/session';
import { displayIdentity } from '../auth/claims';
import { useAppProfile } from '../apps/AppProfileContext';
import { defaultRouteForApp } from '../apps/appRouting';
import type { AppProfile } from '../types';

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
  pipelines: Workflow,
  rules: ShieldCheck,
  replay: History,
  signals: Radar,
  alerts: TriangleAlert,
  insights: Lightbulb,
  health: Gauge,
  dsm: Network,
};

export function DashboardShell() {
  const { authEnabled, claims, signOut } = useAuth();
  const identity = authEnabled ? displayIdentity(claims) : undefined;
  const { profiles, currentApp, currentAppId, nav } = useAppProfile();
  const navigate = useNavigate();

  function selectApp(appId: string) {
    const profile = profiles.find((p: AppProfile) => p.app_id === appId);
    if (!profile || profile.app_id === currentAppId) return;
    void navigate({ to: defaultRouteForApp(profile) });
  }

  return (
    <div className="min-h-screen bg-gray-50 text-gray-900">
      <header className="flex flex-wrap items-center justify-between gap-2 border-b border-gray-200 bg-white px-4 py-2">
        <div className="flex items-center gap-2">
          <Activity size={18} className="text-brand-700" />
          <span className="text-sm font-semibold">SignalOps</span>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          {/* Active app label + selector. The select both displays the active app
              label and switches apps by navigating to the profile default route. */}
          <select
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
          </select>
          <HealthIndicator />
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
      <nav className="flex flex-wrap gap-1 border-b border-gray-200 bg-white px-2">
        {nav.map((item) => {
          const Icon = MODULE_ICONS[item.module] ?? Activity;
          return (
            <Link key={item.to} to={item.to} className={navItem} activeProps={{ className: navItemActive }}>
              <Icon size={14} /> {item.label}
            </Link>
          );
        })}
      </nav>
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
