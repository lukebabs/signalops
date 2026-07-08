import { lazy } from 'react';
import { createRouter, createRoute, createRootRoute } from '@tanstack/react-router';
import { DashboardShell } from './components/DashboardShell';

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

const rootRoute = createRootRoute({ component: DashboardShell });

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
  systemRoute,
]);

export const router = createRouter({ routeTree });

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}
