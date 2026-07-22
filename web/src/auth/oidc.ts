import { UserManager } from 'oidc-client-ts';
import { authConfig } from './config';

// Current origin is used for redirect/logout return URIs so the flow works on any host:port.
function currentOrigin(): string {
  return typeof window !== 'undefined' ? window.location.origin : 'http://localhost:15173';
}

export function createUserManager(): UserManager {
  return new UserManager({
    authority: authConfig.issuer,
    client_id: authConfig.clientId,
    redirect_uri: `${currentOrigin()}/auth/callback`,
    post_logout_redirect_uri: currentOrigin(),
    silent_redirect_uri: `${currentOrigin()}/auth/silent-renew`,
    response_type: 'code',
    scope: 'openid profile email',
    // Keycloak: request the signalops-api audience in the issued access token.
    extraQueryParams: { resource: authConfig.audience },
    automaticSilentRenew: false,
    loadUserInfo: true,
  });
}

let singleton: UserManager | null = null;

// Lazily create the UserManager (only when auth is enabled / a flow runs).
export function getUserManager(): UserManager {
  if (!singleton) singleton = createUserManager();
  return singleton;
}

// Remember the requested path across the IdP redirect so the callback can restore it.
const REDIRECT_PATH_KEY = 'signalops.auth.redirectPath';

export function rememberRedirectPath(path: string): void {
  try {
    sessionStorage.setItem(REDIRECT_PATH_KEY, path || '/');
  } catch {
    /* sessionStorage unavailable */
  }
}

export function consumeRedirectPath(): string {
  try {
    const path = sessionStorage.getItem(REDIRECT_PATH_KEY) || '/';
    sessionStorage.removeItem(REDIRECT_PATH_KEY);
    return path;
  } catch {
    return '/';
  }
}
