// Auth configuration parsed from Vite env (only VITE_* vars reach the browser).
// Auth is OFF by default; enabling it (VITE_SIGNALOPS_AUTH_ENABLED=true) switches the
// SPA to Syncratic IdP login + Bearer-token API calls. The deployed default stays off.

export interface AuthConfig {
  authEnabled: boolean;
  issuer: string;
  clientId: string;
  audience: string;
  realm: string;
  idleTimeoutMinutes: number;
  renewBeforeExpirySeconds: number;
}

const DEFAULT_ISSUER = 'https://auth.syncratic.co/realms/syncratic';

function parseEnabled(value: string | undefined): boolean {
  return String(value ?? '').trim().toLowerCase() === 'true';
}

function positiveInt(value: string | undefined, fallback: number): number {
  const parsed = Number.parseInt(String(value ?? ''), 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}

// Pure resolver so unit tests can feed arbitrary env maps without import.meta.env.
export function resolveAuthConfig(env: Record<string, string | undefined>): AuthConfig {
  return {
    authEnabled: parseEnabled(env.VITE_SIGNALOPS_AUTH_ENABLED),
    issuer: env.VITE_SIGNALOPS_AUTH_ISSUER ?? DEFAULT_ISSUER,
    clientId: env.VITE_SIGNALOPS_AUTH_CLIENT_ID ?? 'signalops-web',
    audience: env.VITE_SIGNALOPS_AUTH_AUDIENCE ?? 'signalops-api',
    realm: env.VITE_SIGNALOPS_AUTH_REALM ?? 'syncratic',
    idleTimeoutMinutes: positiveInt(env.VITE_SIGNALOPS_AUTH_IDLE_TIMEOUT_MINUTES, 30),
    renewBeforeExpirySeconds: positiveInt(env.VITE_SIGNALOPS_AUTH_RENEW_BEFORE_EXPIRY_SECONDS, 300),
  };
}

export const authConfig: AuthConfig = resolveAuthConfig(
  import.meta.env as unknown as Record<string, string | undefined>,
);
