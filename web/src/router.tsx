import { lazy, useEffect } from 'react';
import { createRouter, createRoute, createRootRoute, useNavigate } from '@tanstack/react-router';
import { DashboardShell } from './components/DashboardShell';
import { AppProfileProvider } from './apps/AppProfileContext';
import { LoadingState } from './components/States';

// Route-level code splitting: AG Grid / ECharts only load when the Runs or
// Raw Events views are visited.
const DashboardRoute = lazy(() =>
  import('./routes/DashboardRoute').then((m) => ({ default: m.DashboardRoute })),
);
const RunsRoute = lazy(() =>
  import('./routes/RunsRoute').then((m) => ({ default: m.RunsRoute })),
);
const RawEventsRoute = lazy(() =>
  import('./routes/RawEventsRoute').then((m) => ({ default: m.RawEventsRoute })),
);
const IdempotencyRoute = lazy(() =>
  import('./routes/IdempotencyRoute').then((m) => ({ default: m.IdempotencyRoute })),
);
const SystemRoute = lazy(() =>
  import('./routes/SystemRoute').then((m) => ({ default: m.SystemRoute })),
);
const SourcesRoute = lazy(() =>
  import('./routes/SourcesRoute').then((m) => ({ default: m.SourcesRoute })),
);
const PipelinesRoute = lazy(() =>
  import('./routes/PipelinesRoute').then((m) => ({ default: m.PipelinesRoute })),
);
const RulesRoute = lazy(() =>
  import('./routes/RulesRoute').then((m) => ({ default: m.RulesRoute })),
);
const NormalizedEventsRoute = lazy(() =>
  import('./routes/NormalizedEventsRoute').then((m) => ({ default: m.NormalizedEventsRoute })),
);
const SignalsRoute = lazy(() =>
  import('./routes/SignalsRoute').then((m) => ({ default: m.SignalsRoute })),
);
const AlertsRoute = lazy(() =>
  import('./routes/AlertsRoute').then((m) => ({ default: m.AlertsRoute })),
);
const InsightsRoute = lazy(() =>
  import('./routes/InsightsRoute').then((m) => ({ default: m.InsightsRoute })),
);
const ReplayJobsRoute = lazy(() =>
  import('./routes/ReplayJobsRoute').then((m) => ({ default: m.ReplayJobsRoute })),
);
const MarketOpsAssetsRoute = lazy(() =>
  import('./routes/MarketOpsAssetsRoute').then((m) => ({ default: m.MarketOpsAssetsRoute })),
);
const MarketOpsDashboardRoute = lazy(() =>
  import('./routes/MarketOpsDashboardRoute').then((m) => ({ default: m.MarketOpsDashboardRoute })),
);
const MarketOpsDsmRoute = lazy(() =>
  import('./routes/MarketOpsDsmRoute').then((m) => ({ default: m.MarketOpsDsmRoute })),
);
const MarketOpsOpportunitiesRoute = lazy(() =>
  import('./routes/MarketOpsOpportunitiesRoute').then((m) => ({ default: m.MarketOpsOpportunitiesRoute })),
);
const MarketOpsStateRoute = lazy(() =>
  import('./routes/MarketOpsStateRoute').then((m) => ({ default: m.MarketOpsStateRoute })),
);
const MarketOpsBacktestsRoute = lazy(() =>
  import('./routes/MarketOpsBacktestsRoute').then((m) => ({ default: m.MarketOpsBacktestsRoute })),
);
const MarketOpsSyncraticRoute = lazy(() =>
  import('./routes/MarketOpsSyncraticRoute').then((m) => ({ default: m.MarketOpsSyncraticRoute })),
);
const AlgorithmsRoute = lazy(() =>
  import('./routes/AlgorithmsRoute').then((m) => ({ default: m.AlgorithmsRoute })),
);

// The root route hosts the app-profile provider so every route can read the
// active app (currentApp/metadataFilter/nav). The provider uses useLocation,
// so it must render inside <RouterProvider> (i.e. as the root route component).
function AppRoot() {
  return (
    <AppProfileProvider>
      <DashboardShell />
    </AppProfileProvider>
  );
}

const rootRoute = createRootRoute({ component: AppRoot });

function LegacyRedirect({ to }: { to: string }) {
  const navigate = useNavigate();
  useEffect(() => { void navigate({ to: to as never, replace: true }); }, [navigate, to]);
  return <LoadingState label="Redirecting…" />;
}

const adminDashboardRoute = createRoute({ getParentRoute: () => rootRoute, path: '/admin/dashboard', component: DashboardRoute });
const adminRunsRoute = createRoute({ getParentRoute: () => rootRoute, path: '/admin/runs', component: RunsRoute });
const adminRawEventsRoute = createRoute({ getParentRoute: () => rootRoute, path: '/admin/raw-events', component: RawEventsRoute });
const adminNormalizedRoute = createRoute({ getParentRoute: () => rootRoute, path: '/admin/normalized-events', component: NormalizedEventsRoute });
const adminIdempotencyRoute = createRoute({ getParentRoute: () => rootRoute, path: '/admin/idempotency', component: IdempotencyRoute });
const adminSourcesRoute = createRoute({ getParentRoute: () => rootRoute, path: '/admin/sources', component: SourcesRoute });
const adminPipelinesRoute = createRoute({ getParentRoute: () => rootRoute, path: '/admin/pipelines', component: PipelinesRoute });
const adminRulesRoute = createRoute({ getParentRoute: () => rootRoute, path: '/admin/rules', component: RulesRoute });
const adminReplayRoute = createRoute({ getParentRoute: () => rootRoute, path: '/admin/replay', component: ReplayJobsRoute });
const adminSignalsRoute = createRoute({ getParentRoute: () => rootRoute, path: '/admin/signals', component: SignalsRoute });
const adminAlertsRoute = createRoute({ getParentRoute: () => rootRoute, path: '/admin/alerts', component: AlertsRoute });
const adminInsightsRoute = createRoute({ getParentRoute: () => rootRoute, path: '/admin/insights', component: InsightsRoute });
const adminAlgorithmsRoute = createRoute({ getParentRoute: () => rootRoute, path: '/admin/algorithms', component: AlgorithmsRoute });
const adminSystemRoute = createRoute({ getParentRoute: () => rootRoute, path: '/admin/system', component: SystemRoute });

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: () => <LegacyRedirect to="/admin/dashboard" />,
});

const runsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/runs',
  component: () => <LegacyRedirect to="/admin/runs" />,
});

const rawEventsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/raw-events',
  component: () => <LegacyRedirect to="/admin/raw-events" />,
});

const idempotencyRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/idempotency',
  component: () => <LegacyRedirect to="/admin/idempotency" />,
});

const systemRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/system',
  component: () => <LegacyRedirect to="/admin/system" />,
});

const sourcesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/sources',
  component: () => <LegacyRedirect to="/admin/sources" />,
});

const pipelinesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/pipelines',
  component: PipelinesRoute,
});

const rulesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/rules',
  component: RulesRoute,
});

const normalizedEventsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/normalized-events',
  component: NormalizedEventsRoute,
});

const signalsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/signals',
  component: SignalsRoute,
});

const alertsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/alerts',
  component: AlertsRoute,
});

const insightsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/insights',
  component: InsightsRoute,
});

const replayJobsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/replay',
  component: ReplayJobsRoute,
});

function MarketOpsIndexRouteComponent() {
  const navigate = useNavigate();
  useEffect(() => {
    navigate({ to: '/marketops/dashboard', replace: true });
  }, [navigate]);
  return <LoadingState label="Opening MarketOps..." />;
}

// G067 MarketOps aliases: reuse existing route components under /marketops/*.
// App context (AppProfileProvider) scopes their data via metadataFilter; no
// business logic is duplicated.
const marketopsIndexRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops', component: MarketOpsIndexRouteComponent });
const marketopsDashboardRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/dashboard', component: MarketOpsDashboardRoute });
const marketopsProvidersRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/providers', component: () => <LegacyRedirect to="/admin/sources" /> });
const marketopsRawEventsRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/raw-events', component: () => <LegacyRedirect to="/admin/raw-events" /> });
const marketopsNormalizedRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/normalized', component: () => <LegacyRedirect to="/admin/normalized-events" /> });
const marketopsSignalsRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/signals', component: SignalsRoute });
const marketopsAlertsRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/alerts', component: AlertsRoute });
const marketopsInsightsRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/insights', component: InsightsRoute });
const marketopsReplayRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/replay', component: () => <LegacyRedirect to="/admin/replay" /> });
const marketopsPipelinesRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/pipelines', component: () => <LegacyRedirect to="/admin/pipelines" /> });
const marketopsHealthRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/health', component: () => <LegacyRedirect to="/admin/system" /> });
const marketopsAssetsRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/assets', component: MarketOpsAssetsRoute });
const marketopsStateRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/marketops/state',
  // Persist analyst context (symbol/session_date/tab/hypothesis_key/version) so
  // refresh/back/forward retains it. Unknown/invalid tab falls back to overview.
  validateSearch: (search: Record<string, unknown>): {
    symbol?: string;
    session_date?: string;
    tab?: string;
    hypothesis_key?: string;
    hypothesis_version?: string;
  } => ({
    symbol: typeof search.symbol === 'string' ? search.symbol : undefined,
    session_date: typeof search.session_date === 'string' ? search.session_date : undefined,
    tab: typeof search.tab === 'string' ? search.tab : undefined,
    hypothesis_key: typeof search.hypothesis_key === 'string' ? search.hypothesis_key : undefined,
    hypothesis_version: typeof search.hypothesis_version === 'string' ? search.hypothesis_version : undefined,
  }),
  component: MarketOpsStateRoute,
});
const marketopsDsmRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/dsm', component: MarketOpsDsmRoute });
const marketopsOpportunitiesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/marketops/opportunities',
  // Persist the selected opportunity in ?opportunity_id= so refresh/back/forward
  // retains analyst context (G139). Loose validation keeps the param optional.
  validateSearch: (search: Record<string, unknown>): { opportunity_id?: string } => ({
    opportunity_id: typeof search.opportunity_id === 'string' ? search.opportunity_id : undefined,
  }),
  component: MarketOpsOpportunitiesRoute,
});
const marketopsBacktestsRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/backtests', component: MarketOpsBacktestsRoute });
const marketopsSyncraticRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/syncratic', component: MarketOpsSyncraticRoute });
const marketopsAlgorithmsRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/algorithms', component: () => <LegacyRedirect to="/admin/algorithms" /> });

// /auth/callback is primarily handled by the auth gate in App.tsx (the router must not mount
// before authentication); this route is a fallback that returns the user to the dashboard.
function AuthCallbackRouteComponent() {
  const navigate = useNavigate();
  useEffect(() => {
    navigate({ to: '/', replace: true });
  }, [navigate]);
  return <LoadingState label="Redirecting…" />;
}

const authCallbackRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/auth/callback',
  component: AuthCallbackRouteComponent,
});

const routeTree = rootRoute.addChildren([
  adminDashboardRoute, adminRunsRoute, adminRawEventsRoute, adminNormalizedRoute, adminIdempotencyRoute, adminSourcesRoute, adminPipelinesRoute, adminRulesRoute, adminReplayRoute, adminSignalsRoute, adminAlertsRoute, adminInsightsRoute, adminAlgorithmsRoute, adminSystemRoute,
  indexRoute,
  runsRoute,
  rawEventsRoute,
  normalizedEventsRoute,
  idempotencyRoute,
  sourcesRoute,
  pipelinesRoute,
  rulesRoute,
  signalsRoute,
  alertsRoute,
  insightsRoute,
  replayJobsRoute,
  authCallbackRoute,
  systemRoute,
  // G067 MarketOps aliases (declared above) must be registered here or they 404.
  marketopsIndexRoute,
  marketopsDashboardRoute,
  marketopsProvidersRoute,
  marketopsRawEventsRoute,
  marketopsNormalizedRoute,
  marketopsSignalsRoute,
  marketopsAlertsRoute,
  marketopsInsightsRoute,
  marketopsReplayRoute,
  marketopsPipelinesRoute,
  marketopsHealthRoute,
  marketopsAssetsRoute,
  marketopsStateRoute,
  marketopsDsmRoute,
  marketopsOpportunitiesRoute,
  marketopsBacktestsRoute,
  marketopsSyncraticRoute,
  marketopsAlgorithmsRoute,
]);

export const router = createRouter({ routeTree });

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}
