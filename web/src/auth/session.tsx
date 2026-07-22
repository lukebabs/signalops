import {
  createContext,
  useContext,
  useEffect,
  useCallback,
  useMemo,
  useRef,
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
  const userRef = useRef<User | null>(null);
  const renewingRef = useRef(false);
  const lastActivityRef = useRef(Date.now());

  useEffect(() => {
    if (!authConfig.authEnabled) {
      setLoading(false);
      return;
    }
    const manager = getUserManager();
    const idleTimeoutMs = authConfig.idleTimeoutMinutes * 60_000;
    let cancelled = false;
    const applyUser = (u: User) => {
      userRef.current = u;
      setUser(u);
      currentAccessToken = u.access_token;
    };
    // Loading a renewed token must not itself count as user activity. Otherwise
    // a background renewal would keep an unattended session alive forever.
    const onLoaded = (u: User) => applyUser(u);
    const onUnloaded = () => {
      userRef.current = null;
      setUser(null);
      currentAccessToken = null;
    };
    const renew = async () => {
      if (renewingRef.current) return;
      renewingRef.current = true;
      try {
        const renewed = await manager.signinSilent();
        if (renewed) applyUser(renewed);
      } catch (e) {
        if (!cancelled) setError(`Session renewal failed: ${errMsg(e)}`);
      } finally {
        renewingRef.current = false;
      }
    };
    const maintainSession = () => {
      const active = userRef.current;
      if (!active) return;
      if (Date.now() - lastActivityRef.current >= idleTimeoutMs) {
        void manager.removeUser();
        return;
      }
      if (active.expires_in == null || active.expires_in <= authConfig.renewBeforeExpirySeconds) void renew();
    };
    const recordActivity = () => {
      lastActivityRef.current = Date.now();
      maintainSession();
    };
    void (async () => {
      try {
        const u = await manager.getUser();
        if (!cancelled) {
          if (u) {
            // Restoring the application is an active visit; subsequent token
            // renewals retain this timestamp until the user interacts again.
            lastActivityRef.current = Date.now();
            applyUser(u);
          }
          setLoading(false);
        }
      } catch (e) {
        if (!cancelled) {
          setError(errMsg(e));
          setLoading(false);
        }
      }
    })();
    const onSilentError = (e: Error) => setError(e.message);
    manager.events.addUserLoaded(onLoaded);
    manager.events.addUserUnloaded(onUnloaded);
    manager.events.addSilentRenewError(onSilentError);
    const activityEvents: Array<keyof DocumentEventMap> = ['pointerdown', 'keydown', 'scroll', 'touchstart'];
    activityEvents.forEach((event) => document.addEventListener(event, recordActivity, { passive: true }));
    window.addEventListener('focus', recordActivity);
    const sessionTimer = window.setInterval(maintainSession, 15_000);
    return () => {
      cancelled = true;
      window.clearInterval(sessionTimer);
      window.removeEventListener('focus', recordActivity);
      activityEvents.forEach((event) => document.removeEventListener(event, recordActivity));
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
    userRef.current = u;
    lastActivityRef.current = Date.now();
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
