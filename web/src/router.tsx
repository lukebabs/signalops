import { lazy } from 'react';
import { createRouter, createRoute, createRootRoute } from '@tanstack/react-router';
import { DashboardShell } from './components/DashboardShell';

// Route-level code splitting: AG Grid / ECharts only load when the Runs or
// Raw Events views are visited.
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

const rootRoute = createRootRoute({ component: DashboardShell });

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: RunsRoute,
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

const routeTree = rootRoute.addChildren([
  indexRoute,
  runsRoute,
  rawEventsRoute,
  idempotencyRoute,
  systemRoute,
]);

export const router = createRouter({ routeTree });

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}
