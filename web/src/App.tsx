import { useEffect, type ReactNode } from 'react';
import { QueryClient, QueryClientProvider, useQuery, useQueryClient } from '@tanstack/react-query';
import { RouterProvider } from '@tanstack/react-router';
import { router } from './router';
import { DashboardStreamBridge } from './components/DashboardStreamBridge';
import { AuthProvider, useAuth } from './auth/session';
import { AuthCallbackProcessor, AuthLoginRedirectProcessor, LoginScreen, SilentRenewProcessor } from './auth/LoginScreen';
import { authConfig } from './auth/config';
import { sanitizeRedirectPath } from './auth/oidc';
import { ThemeProvider } from './theme/theme';
import { MarketOpsWatchlistContextProvider } from "./components/MarketOpsWatchlistContext";
import { UserActivityBridge } from './components/UserActivityBridge';
import { LoadingState, ErrorState } from './components/States';
import { api } from './api/client';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 10_000, refetchOnWindowFocus: false, retry: false },
  },
});


function EnrollmentGate({ children }: { children: ReactNode }) {
  const enrollment = useQuery({ queryKey: ['session', 'enrollment'], queryFn: () => api.getSessionEnrollment(), retry: false });
  const state = enrollment.data?.state;
  const pathname = typeof window !== 'undefined' ? window.location.pathname : '';
  const pricingRoute = pathname.startsWith('/marketops/pricing') || pathname.startsWith('/marketops/subscription/return');
  const shouldRedirectToPricing = state === 'subscription_missing' && !pricingRoute;

  useEffect(() => {
    if (shouldRedirectToPricing && typeof window !== 'undefined') {
      window.location.replace('/marketops/pricing?source_feature=enrollment');
    }
  }, [shouldRedirectToPricing]);

  if (enrollment.isLoading) {
    return <div className="min-h-screen bg-gray-50 p-6"><LoadingState label="Preparing MarketOps access…" /></div>;
  }
  if (enrollment.isError) {
    return <div className="min-h-screen bg-gray-50 p-6"><ErrorState error={String((enrollment.error as Error)?.message ?? enrollment.error)} /></div>;
  }

  if (state === 'subscription_missing' && pricingRoute) {
    return <>{children}</>;
  }
  if (shouldRedirectToPricing) {
    return <div className="min-h-screen bg-gray-50 p-6 dark:bg-gray-950"><LoadingState label="Opening subscription options…" /></div>;
  }
  if (state && state !== 'marketops_ready') {
    const copy: Record<string, { title: string; message: string }> = {
      email_verification_required: { title: 'Verify your email to continue', message: 'Your Syncratic identity is authenticated, but MarketOps access is held until Keycloak confirms email verification.' },
      tenant_access_missing: { title: 'MarketOps access is pending', message: 'Your identity does not yet have an approved MarketOps tenant grant.' },
      subscription_missing: { title: 'Choose a subscription to continue', message: 'Your Syncratic identity is verified, but MarketOps access requires an active Explorer, Professional, or Institutional subscription.' },
      watchlist_context_missing: { title: 'Watchlist setup is pending', message: 'Your tenant does not yet have a default or private watchlist context.' },
    };
    const content = copy[state] ?? { title: 'Enrollment is pending', message: 'Your account is authenticated, but MarketOps enrollment is not complete.' };
    return (
      <div className="flex min-h-screen items-center justify-center bg-gray-50 p-4">
        <section className="w-full max-w-lg rounded border border-gray-200 bg-white p-6 shadow-sm">
          <p className="text-xs font-semibold uppercase tracking-wide text-brand-600">SignalOps enrollment</p>
          <h1 className="mt-2 text-xl font-semibold text-gray-900">{content.title}</h1>
          <p className="mt-2 text-sm text-gray-600">{content.message}</p>
          <dl className="mt-4 grid gap-2 text-xs text-gray-600">
            <div className="flex justify-between gap-3"><dt>Tenant</dt><dd className="font-mono">{enrollment.data?.tenant_id}</dd></div>
            <div className="flex justify-between gap-3"><dt>State</dt><dd className="font-mono">{state}</dd></div>
            {enrollment.data?.email && <div className="flex justify-between gap-3"><dt>Email</dt><dd>{enrollment.data.email}</dd></div>}
          </dl>
          <div className="mt-5 flex flex-wrap gap-2">
            {state === 'subscription_missing' ? <button type="button" onClick={() => window.location.assign('/marketops/pricing?source_feature=enrollment')} className="rounded bg-brand-600 px-3 py-2 text-sm text-white hover:bg-brand-700">View subscription options</button> : null}
            <button type="button" onClick={() => window.location.reload()} className="rounded border border-gray-300 px-3 py-2 text-sm text-gray-700 hover:bg-gray-50">Check again</button>
          </div>
        </section>
      </div>
    );
  }
  return <>{children}</>;
}

function isAuthLoginRoute(): boolean {
  return typeof window !== 'undefined' && window.location.pathname.startsWith('/auth/login');
}

function isCallbackRoute(): boolean {
  return typeof window !== 'undefined' && window.location.pathname.startsWith('/auth/callback');
}

function isSilentRenewRoute(): boolean {
  return typeof window !== 'undefined' && window.location.pathname.startsWith('/auth/silent-renew');
}

// App-level auth gate. When auth is enabled, no protected route — and therefore no protected
// /v1/* query — mounts before an access token exists. The IdP callback is processed here so the
// router never has to render while unauthenticated.
function RootGate() {
  const session = useAuth();
  const qc = useQueryClient();

  // On logout (or expired session) clear cached data so prior tenant/operator rows don't linger.
  useEffect(() => {
    if (session.authEnabled && !session.loading && !session.authenticated) {
      qc.clear();
    }
  }, [session.authEnabled, session.loading, session.authenticated, qc]);

  if (!session.authEnabled) {
    if (isAuthLoginRoute()) {
      const params = new URLSearchParams(window.location.search);
      window.location.replace(sanitizeRedirectPath(params.get('redirect') || '/marketops/dashboard'));
      return <LoginScreen loading />;
    }
    return (
      <>
        <DashboardStreamBridge />
        <MarketOpsWatchlistContextProvider><RouterProvider router={router} /></MarketOpsWatchlistContextProvider>
      </>
    );
  }
  if (isAuthLoginRoute()) {
    return <AuthLoginRedirectProcessor authenticated={session.authenticated} />;
  }
  if (isCallbackRoute()) {
    return <AuthCallbackProcessor />;
  }
  if (isSilentRenewRoute()) {
    return <SilentRenewProcessor />;
  }
  if (session.loading) {
    return <LoginScreen loading />;
  }
  if (!session.authenticated) {
    return <LoginScreen error={session.error} onSignIn={() => void session.signIn()} onSignUp={authConfig.signUpUrl ? () => void session.signUp() : undefined} />;
  }
  return (
    <EnrollmentGate>
      <DashboardStreamBridge />
      <UserActivityBridge />
      <MarketOpsWatchlistContextProvider><RouterProvider router={router} /></MarketOpsWatchlistContextProvider>
    </EnrollmentGate>
  );
}

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <AuthProvider>
          <RootGate />
        </AuthProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
}
