import {
  createContext,
  useContext,
  useEffect,
  useCallback,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import type { User } from 'oidc-client-ts';
import { authConfig } from './config';
import { consumeRedirectPath, getUserManager, rememberRedirectPath } from './oidc';
import type { AuthClaims } from './claims';
import { displayIdentity } from './claims';

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

  // These handlers use only stable references (the UserManager singleton,
  // stable setState, and module functions), so they're memoized with empty
  // deps. A stable finishCallback identity matters: AuthCallbackProcessor's
  // effect depends on it, and if it changed when setUser() runs mid-callback
  // the effect would re-run and call signinRedirectCallback() a second time —
  // the PKCE state is already consumed on the first call, producing
  // "No matching state found in storage" and bouncing the user to login.
  const signIn = useCallback(async () => {
    try {
      rememberRedirectPath(window.location.pathname + window.location.search);
      await getUserManager().signinRedirect();
    } catch (e) {
      setError(errMsg(e));
    }
  }, []);

  // On success returns the path to restore; on failure it throws so the caller
  // (AuthCallbackProcessor) can surface the IdP/PKCE error.
  const finishCallback = useCallback(async () => {
    const u = await getUserManager().signinRedirectCallback();
    setUser(u);
    currentAccessToken = u?.access_token ?? null;
    return consumeRedirectPath();
  }, []);

  const signOut = useCallback(async () => {
    try {
      currentAccessToken = null;
      await getUserManager().signoutRedirect();
    } catch (e) {
      setError(errMsg(e));
    }
  }, []);

  const value = useMemo<SessionState>(
    () => ({
      authEnabled: authConfig.authEnabled,
      loading,
      authenticated: !!user && !user.expired,
      user,
      claims: (user?.profile as AuthClaims | undefined) ?? null,
      error,
      signIn,
      finishCallback,
      signOut,
    }),
    [user, loading, error, signIn, finishCallback, signOut],
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

// Actor name for replay job `requested_by`: token identity (preferred_username
// -> email -> sub) when auth is on, else operator-local. Unlike lifecycle
// mutations, the replay backend does not derive the actor from the token, so
// the identity is sent in the request body and falls back to operator-local.
export function useActor(): string {
  const { authEnabled, claims } = useAuth();
  if (!authEnabled) return 'operator-local';
  return displayIdentity(claims) ?? 'operator-local';
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
