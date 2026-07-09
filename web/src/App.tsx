import { useEffect } from 'react';
import { QueryClient, QueryClientProvider, useQueryClient } from '@tanstack/react-query';
import { RouterProvider } from '@tanstack/react-router';
import { router } from './router';
import { DashboardStreamBridge } from './components/DashboardStreamBridge';
import { AuthProvider, useAuth } from './auth/session';
import { AuthCallbackProcessor, LoginScreen } from './auth/LoginScreen';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 10_000, refetchOnWindowFocus: false, retry: false },
  },
});

function isCallbackRoute(): boolean {
  return typeof window !== 'undefined' && window.location.pathname.startsWith('/auth/callback');
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
    return (
      <>
        <DashboardStreamBridge />
        <RouterProvider router={router} />
      </>
    );
  }
  if (isCallbackRoute()) {
    return <AuthCallbackProcessor />;
  }
  if (session.loading) {
    return <LoginScreen loading />;
  }
  if (!session.authenticated) {
    return <LoginScreen error={session.error} onSignIn={() => void session.signIn()} />;
  }
  return (
    <>
      <DashboardStreamBridge />
      <RouterProvider router={router} />
    </>
  );
}

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <RootGate />
      </AuthProvider>
    </QueryClientProvider>
  );
}
