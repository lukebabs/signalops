import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import type { User } from 'oidc-client-ts';
import { authConfig } from './config';
import { consumeRedirectPath, getUserManager, rememberRedirectPath } from './oidc';
import type { AuthClaims } from './claims';

export interface SessionState {
  authEnabled: boolean;
  loading: boolean; // initializing / processing
  authenticated: boolean;
  user: User | null;
  claims: AuthClaims | null;
  error: string | null;
  signIn: () => Promise<void>;
  finishCallback: () => Promise<string>;
  signOut: () => Promise<void>;
}

const SessionContext = createContext<SessionState | null>(null);

// Module-level access-token holder so the non-React api/client.ts can attach the
// current Bearer token without React context. The provider updates it on user changes.
let currentAccessToken: string | null = null;

export function getAccessToken(): string | null {
  return currentAccessToken;
}

// Test seam: set/clear the token holder without a provider.
export function setAccessTokenForTest(token: string | null): void {
  currentAccessToken = token;
}

function errMsg(e: unknown): string {
  return String((e as Error)?.message ?? e);
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(authConfig.authEnabled);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!authConfig.authEnabled) {
      setLoading(false);
      return;
    }
    const manager = getUserManager();
    let cancelled = false;
    void (async () => {
      try {
        const u = await manager.getUser();
        if (cancelled) return;
        setUser(u);
        currentAccessToken = u?.access_token ?? null;
      } catch (e) {
        if (!cancelled) setError(errMsg(e));
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    const onLoaded = (u: User) => {
      setUser(u);
      currentAccessToken = u.access_token;
    };
    const onUnloaded = () => {
      setUser(null);
      currentAccessToken = null;
    };
    const onSilentError = (e: Error) => setError(e.message);
    manager.events.addUserLoaded(onLoaded);
    manager.events.addUserUnloaded(onUnloaded);
    manager.events.addSilentRenewError(onSilentError);
    return () => {
      cancelled = true;
      manager.events.removeUserLoaded(onLoaded);
      manager.events.removeUserUnloaded(onUnloaded);
      manager.events.removeSilentRenewError(onSilentError);
    };
  }, []);

  const value = useMemo<SessionState>(
    () => ({
      authEnabled: authConfig.authEnabled,
      loading,
      authenticated: !!user && !user.expired,
      user,
      claims: (user?.profile as AuthClaims | undefined) ?? null,
      error,
      async signIn() {
        try {
          rememberRedirectPath(window.location.pathname + window.location.search);
          await getUserManager().signinRedirect();
        } catch (e) {
          setError(errMsg(e));
        }
      },
      async finishCallback() {
        try {
          const u = await getUserManager().signinRedirectCallback();
          setUser(u);
          currentAccessToken = u?.access_token ?? null;
          return consumeRedirectPath();
        } catch (e) {
          setError(errMsg(e));
          return '/';
        }
      },
      async signOut() {
        try {
          currentAccessToken = null;
          await getUserManager().signoutRedirect();
        } catch (e) {
          setError(errMsg(e));
        }
      },
    }),
    [user, loading, error],
  );

  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

export function useAuth(): SessionState {
  const ctx = useContext(SessionContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}

// Tenant used by route queries: token tenant_id when auth is on, else tenant-local (dev/disabled).
export function useTenant(): string {
  const { authEnabled, claims } = useAuth();
  if (!authEnabled) return 'tenant-local';
  return claims?.tenant_id ?? 'tenant-local';
}

// Lifecycle mutation permission: operator/admin when auth is on; allowed when auth is off (dev).
export function useCanMutateLifecycle(): boolean {
  const { authEnabled, claims } = useAuth();
  if (!authEnabled) return true;
  if (!claims) return false;
  const roles = [
    ...(claims.realm_access?.roles ?? []),
    ...(claims.resource_access?.['signalops-api']?.roles ?? []),
  ];
  return roles.includes('signalops:operator') || roles.includes('signalops:admin');
}
