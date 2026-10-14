import { UserManager } from 'oidc-client-ts';
import { authConfig } from './config';

// Current origin is used for redirect/logout return URIs so the flow works on any host:port.
function currentOrigin(): string {
  return typeof window !== 'undefined' ? window.location.origin : 'http://localhost:15173';
}

function userManagerSettings(registration = false) {
  const issuer = authConfig.issuer.replace(/\/$/, '');
  return {
    authority: authConfig.issuer,
    client_id: authConfig.clientId,
    redirect_uri: `${currentOrigin()}/auth/callback`,
    post_logout_redirect_uri: `${currentOrigin()}/auth/signed-out`,
    silent_redirect_uri: `${currentOrigin()}/auth/silent-renew`,
    response_type: 'code',
    scope: 'openid profile email',
    // Keycloak: request the signalops-api audience in the issued access token.
    extraQueryParams: { resource: authConfig.audience },
    automaticSilentRenew: false,
    loadUserInfo: true,
    ...(registration ? { metadataSeed: { authorization_endpoint: `${issuer}/protocol/openid-connect/registrations` } } : {}),
  };
}

export function createUserManager(): UserManager {
  return new UserManager(userManagerSettings(false));
}

export function createRegistrationUserManager(): UserManager {
  return new UserManager(userManagerSettings(true));
}

let singleton: UserManager | null = null;
let registrationSingleton: UserManager | null = null;

// Lazily create the UserManager (only when auth is enabled / a flow runs).
export function getUserManager(): UserManager {
  if (!singleton) singleton = createUserManager();
  return singleton;
}

export function getRegistrationUserManager(): UserManager {
  if (!registrationSingleton) registrationSingleton = createRegistrationUserManager();
  return registrationSingleton;
}

// Remember the requested path across the IdP redirect so the callback can restore it.
// Auth endpoints are flow machinery, never destinations: restoring one after a
// successful callback would render the application's not-found route.
const REDIRECT_PATH_KEY = 'signalops.auth.redirectPath';
const AUTH_CORRELATION_KEY = 'signalops.auth.correlation';
// The root route is the authenticated SignalOps landing page. It presents the
// available workspace widgets (MarketOps, NarrativeOps, Streams, etc.) and
// lets each user choose a domain rather than forcing a single product view.
export const DEFAULT_POST_LOGIN_PATH = '/';

export type AuthDiagnosticEvent = 'redirect_started' | 'callback_succeeded' | 'callback_failed';

function authCorrelation(): string {
  try {
    const existing = sessionStorage.getItem(AUTH_CORRELATION_KEY);
    if (existing) return existing;
    const correlation = globalThis.crypto?.randomUUID?.() ?? `auth-${Date.now()}-${Math.random().toString(36).slice(2)}`;
    sessionStorage.setItem(AUTH_CORRELATION_KEY, correlation);
    return correlation;
  } catch {
    return `auth-${Date.now()}`;
  }
}

export function recordAuthDiagnostic(eventType: AuthDiagnosticEvent, details: { destination?: string; error?: string } = {}): void {
  const payload = {
    event_type: eventType,
    correlation_id: authCorrelation(),
    destination: details.destination ?? '',
    oidc_issuer: authConfig.issuer,
    redirect_uri: `${currentOrigin()}/auth/callback`,
    ...(details.error ? { error: details.error.slice(0, 256) } : {}),
  };
  try {
    const body = new Blob([JSON.stringify(payload)], { type: 'application/json' });
    if (typeof navigator.sendBeacon === 'function' && navigator.sendBeacon('/v1/auth/diagnostics', body)) return;
    void fetch('/v1/auth/diagnostics', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload), keepalive: true });
  } catch {
    // Diagnostics must never interrupt authentication.
  }
}

export function clearAuthDiagnostic(): void {
  try { sessionStorage.removeItem(AUTH_CORRELATION_KEY); } catch { /* storage unavailable */ }
}

export function sanitizeRedirectPath(path: string | null | undefined): string {
  if (!path || !path.startsWith('/') || path.startsWith('//')) return DEFAULT_POST_LOGIN_PATH;
  if (path === '/auth' || path.startsWith('/auth/')) return DEFAULT_POST_LOGIN_PATH;
  return path;
}

export function rememberRedirectPath(path: string): void {
  try {
    sessionStorage.setItem(REDIRECT_PATH_KEY, sanitizeRedirectPath(path));
  } catch {
    /* sessionStorage unavailable */
  }
}

export function clearRedirectPath(): void {
  try {
    sessionStorage.removeItem(REDIRECT_PATH_KEY);
  } catch {
    /* sessionStorage unavailable */
  }
}

export function consumeRedirectPath(): string {
  try {
    const path = sanitizeRedirectPath(sessionStorage.getItem(REDIRECT_PATH_KEY));
    sessionStorage.removeItem(REDIRECT_PATH_KEY);
    return path;
  } catch {
    return DEFAULT_POST_LOGIN_PATH;
  }
}
