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
const MarketOpsDsmRoute = lazy(() =>
  import('./routes/MarketOpsDsmRoute').then((m) => ({ default: m.MarketOpsDsmRoute })),
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

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: DashboardRoute,
});

const runsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/runs',
  component: RunsRoute,
});

const rawEventsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/raw-events',
  component: RawEventsRoute,
});

const idempotencyRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/idempotency',
  component: IdempotencyRoute,
});

const systemRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/system',
  component: SystemRoute,
});

const sourcesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/sources',
  component: SourcesRoute,
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
const marketopsDashboardRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/dashboard', component: DashboardRoute });
const marketopsProvidersRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/providers', component: SourcesRoute });
const marketopsRawEventsRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/raw-events', component: RawEventsRoute });
const marketopsNormalizedRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/normalized', component: NormalizedEventsRoute });
const marketopsSignalsRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/signals', component: SignalsRoute });
const marketopsAlertsRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/alerts', component: AlertsRoute });
const marketopsInsightsRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/insights', component: InsightsRoute });
const marketopsReplayRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/replay', component: ReplayJobsRoute });
const marketopsPipelinesRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/pipelines', component: PipelinesRoute });
const marketopsHealthRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/health', component: SystemRoute });
const marketopsAssetsRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/assets', component: MarketOpsAssetsRoute });
const marketopsDsmRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/dsm', component: MarketOpsDsmRoute });
const marketopsBacktestsRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/backtests', component: MarketOpsBacktestsRoute });
const marketopsSyncraticRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/syncratic', component: MarketOpsSyncraticRoute });
const marketopsAlgorithmsRoute = createRoute({ getParentRoute: () => rootRoute, path: '/marketops/algorithms', component: AlgorithmsRoute });

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
  marketopsDsmRoute,
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
