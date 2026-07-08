import { Suspense } from 'react';
import { Link, Outlet } from '@tanstack/react-router';
import { Activity, ListTree, Database, KeyRound, Gauge } from 'lucide-react';
import { HealthIndicator } from './HealthIndicator';

const navItem =
  'inline-flex items-center gap-1 border-b-2 border-transparent px-3 py-2 text-sm text-gray-600 hover:bg-gray-50';
const navItemActive = 'text-brand-700 border-brand-500';

export function DashboardShell() {
  return (
    <div className="min-h-screen bg-gray-50 text-gray-900">
      <header className="flex items-center justify-between border-b border-gray-200 bg-white px-4 py-2">
        <div className="flex items-center gap-2">
          <Activity size={18} className="text-brand-700" />
          <span className="text-sm font-semibold">SignalOps</span>
        </div>
        <HealthIndicator />
      </header>
      <nav className="flex gap-1 border-b border-gray-200 bg-white px-2">
        <Link to="/runs" className={navItem} activeProps={{ className: navItemActive }}>
          <ListTree size={14} /> Runs
        </Link>
        <Link to="/raw-events" className={navItem} activeProps={{ className: navItemActive }}>
          <Database size={14} /> Raw Events
        </Link>
        <Link to="/idempotency" className={navItem} activeProps={{ className: navItemActive }}>
          <KeyRound size={14} /> Idempotency
        </Link>
        <Link to="/system" className={navItem} activeProps={{ className: navItemActive }}>
          <Gauge size={14} /> System
        </Link>
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
