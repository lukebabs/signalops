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
import { clearRedirectPath, consumeRedirectPath, getUserManager, rememberRedirectPath } from './oidc';
import type { AuthClaims } from './claims';
import { displayIdentity, hasPlatformAdmin, mergeSessionClaims } from './claims';

export interface SessionState {
  authEnabled: boolean;
  loading: boolean; // initializing / processing
  authenticated: boolean;
  user: User | null;
  claims: AuthClaims | null;
  error: string | null;
  signIn: () => Promise<void>;
  signUp: () => Promise<void>;
  finishCallback: () => Promise<string>;
  signOut: () => Promise<void>;
}

const SessionContext = createContext<SessionState | null>(null);

// Module-level access-token holder so the non-React api/client.ts can attach the
// current Bearer token without React context. The provider updates it on user changes.
let currentAccessToken: string | null = null;
let authFailureRedirectInFlight = false;
const TOKEN_EXPIRY_SAFETY_SECONDS = 10;
const AUTH_FAILURE_RETRY_KEY = 'signalops.auth.failure.retry';
type AuthFailureOptions = { automaticRetry?: boolean };

function clearAuthFailureRetry(): void {
  try {
    window.sessionStorage.removeItem(AUTH_FAILURE_RETRY_KEY);
  } catch {
    // Storage may be unavailable in privacy-restricted browser contexts.
  }
}

function tokenExpiresWithin(token: string, windowSeconds: number): boolean {
  const [, payload] = token.split('.');
  if (!payload || typeof globalThis.atob !== 'function') return false;
  try {
    const normalized = payload.replace(/-/g, '+').replace(/_/g, '/');
    const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, '=');
    const parsed = JSON.parse(globalThis.atob(padded)) as { exp?: unknown };
    if (typeof parsed.exp !== 'number') return false;
    return parsed.exp <= Math.floor(Date.now() / 1000) + windowSeconds;
  } catch {
    return false;
  }
}

export function getAccessToken(): string | null {
  if (currentAccessToken && tokenExpiresWithin(currentAccessToken, TOKEN_EXPIRY_SAFETY_SECONDS)) {
    currentAccessToken = null;
    void redirectToSignInForAuthFailure();
    return null;
  }
  return currentAccessToken;
}

/**
 * Recover from a token that is expired, malformed, or bound to a different
 * tenant than the request. Clearing the oidc-client user before redirecting is
 * important: otherwise the SPA keeps restoring the rejected token and the
 * operator is trapped on the error screen until browser storage is cleared.
 */
export async function redirectToSignInForAuthFailure(options: AuthFailureOptions = {}): Promise<void> {
  if (!authConfig.authEnabled || authFailureRedirectInFlight) return;
  authFailureRedirectInFlight = true;
  currentAccessToken = null;
  try {
    const manager = getUserManager();
    if (options.automaticRetry === false) {
      await manager.removeUser();
      clearAuthFailureRetry();
      clearRedirectPath();
      authFailureRedirectInFlight = false;
      window.location.replace('/?auth_error=tenant_context');
      return;
    }
    // A service account (or a user with no tenant assignment) can receive the
    // same invalid token after a fresh OIDC login. Allow one clean retry, then
    // stop on the login screen instead of recursively starting OIDC forever.
    const alreadyRetried = window.sessionStorage.getItem(AUTH_FAILURE_RETRY_KEY) === '1';
    if (alreadyRetried) {
      await manager.removeUser();
      authFailureRedirectInFlight = false;
      return;
    }
    window.sessionStorage.setItem(AUTH_FAILURE_RETRY_KEY, '1');
    const path = `${window.location.pathname}${window.location.search}`;
    rememberRedirectPath(path);
    await manager.removeUser();
    await manager.signinRedirect();
  } catch {
    authFailureRedirectInFlight = false;
  }
}

// Kept as a compatibility export for existing callers and test seams.
export async function redirectToSignInForExpiredSession(): Promise<void> {
  return redirectToSignInForAuthFailure();
}

// Test seam: set/clear the token holder without a provider.
export function setAccessTokenForTest(token: string | null): void {
  currentAccessToken = token;
  authFailureRedirectInFlight = false;
}

function errMsg(e: unknown): string {
  return String((e as Error)?.message ?? e);
}

async function sendSessionActivity(eventType: 'login' | 'logout', token: string | null): Promise<void> {
  if (!authConfig.authEnabled || !token) return;
  try {
    await fetch('/v1/session/activity', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify({
        event_type: eventType,
        app_id: 'marketops',
        feature_key: 'session',
        route_path: window.location.pathname,
        correlation_id: `session-${eventType}-${Date.now()}`,
      }),
      keepalive: eventType === 'logout',
    });
  } catch {
    // Activity capture is best-effort and must never interrupt authentication.
  }
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
        if (!cancelled) {
          currentAccessToken = null;
          userRef.current = null;
          setUser(null);
          setError(`Session renewal failed: ${errMsg(e)}`);
          void redirectToSignInForAuthFailure();
        }
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
      clearAuthFailureRetry();
      const tenantContextFailure = new URLSearchParams(window.location.search).get('auth_error') === 'tenant_context';
      if (tenantContextFailure) clearRedirectPath();
      else rememberRedirectPath(window.location.pathname + window.location.search);
      await getUserManager().signinRedirect(
        tenantContextFailure ? { extraQueryParams: { prompt: 'login' } } : undefined,
      );
    } catch (e) {
      setError(errMsg(e));
    }
  }, []);

  const signUp = useCallback(async () => {
    try {
      clearAuthFailureRetry();
      rememberRedirectPath('/marketops/dashboard');
      const configuredURL = authConfig.signUpUrl.trim();
      if (configuredURL) {
        const url = new URL(configuredURL, window.location.origin);
        if (url.origin === window.location.origin && url.pathname === '/auth/login' && !url.searchParams.has('intent')) {
          url.searchParams.set('intent', 'register');
        }
        window.location.assign(url.toString());
        return;
      }
      setError('Account creation is not configured for this deployment. Use Sign in or contact support.');
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
    void sendSessionActivity('login', currentAccessToken);
    const restoredPath = consumeRedirectPath();
    return restoredPath;
  }, []);

  const signOut = useCallback(async () => {
    const manager = getUserManager();
    // Clear the app-held session before navigating away. Some IdPs complete their
    // logout redirect even when their browser SSO cookie remains, and without
    // this step oidc-client-ts restores the cached user at `/`.
    const token = currentAccessToken;
    void sendSessionActivity('logout', token);
    currentAccessToken = null;
    clearAuthFailureRetry();
    userRef.current = null;
    setUser(null);
    try {
      sessionStorage.removeItem("signalops.auth.redirectPath");
      await manager.removeUser();
      await manager.signoutRedirect();
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
      claims: mergeSessionClaims((user?.profile as AuthClaims | undefined) ?? null, user?.access_token),
      error,
      signIn,
      signUp,
      finishCallback,
      signOut,
    }),
    [user, loading, error, signIn, signUp, finishCallback, signOut],
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
  return roles.includes('signalops:operator') || hasPlatformAdmin(claims);
}
