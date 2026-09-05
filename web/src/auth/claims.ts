// Pure helpers over OIDC token claims. No React, no DOM — fully unit-testable.
// Roles are accepted from BOTH Keycloak locations for forward compatibility.

export interface AuthClaims {
  sub?: string;
  preferred_username?: string;
  email?: string;
  email_verified?: boolean;
  tenant_id?: string;
  realm_access?: { roles?: string[] };
  resource_access?: Record<string, { roles?: string[] }>;
}

export const ROLE_VIEWER = 'signalops:viewer';
export const ROLE_OPERATOR = 'signalops:operator';
export const ROLE_ADMIN = 'signalops:admin';
export const ROLE_SUBSCRIPTION_ADMIN = 'signalops:subscription_admin';
// The realm-level platform administrator role. ROLE_ADMIN remains a compatibility alias.
export const ROLE_SUPER_ADMIN = 'super_admin';

// Merge roles from realm_access.roles and resource_access["signalops-api"].roles.
export function rolesFromClaims(claims: AuthClaims | null | undefined): string[] {
  if (!claims) return [];
  const roles = new Set<string>();
  claims.realm_access?.roles?.forEach((r) => roles.add(r));
  claims.resource_access?.['signalops-api']?.roles?.forEach((r) => roles.add(r));
  return [...roles];
}

export function hasRole(claims: AuthClaims | null | undefined, role: string): boolean {
  return rolesFromClaims(claims).includes(role);
}

export function mergeSessionClaims(profile: AuthClaims | null | undefined, accessToken: string | null | undefined): AuthClaims | null {
  let accessClaims: AuthClaims = {};
  try {
    const payload = accessToken?.split(".")[1];
    if (payload) {
      const base64 = payload.replace(/-/g, "+").replace(/_/g, "/");
      accessClaims = JSON.parse(atob(base64.padEnd(Math.ceil(base64.length / 4) * 4, "="))) as AuthClaims;
    }
  } catch {
    // The gateway remains authorization authority; an unreadable browser token contributes no UI claims.
  }
  if (!profile && !accessToken) return null;
  return { ...profile, ...accessClaims, realm_access: accessClaims.realm_access ?? profile?.realm_access, resource_access: accessClaims.resource_access ?? profile?.resource_access };
}

export function hasPlatformAdmin(claims: AuthClaims | null | undefined): boolean {
  const roles = rolesFromClaims(claims);
  return roles.includes(ROLE_SUPER_ADMIN) || roles.includes(ROLE_ADMIN);
}

export function hasSubscriptionAdministrator(claims: AuthClaims | null | undefined): boolean {
  return hasPlatformAdmin(claims) || hasRole(claims, ROLE_SUBSCRIPTION_ADMIN);
}

// Read access to protected /v1/* requires viewer, operator, or admin.
export function canReadSignalOps(claims: AuthClaims | null | undefined): boolean {
  const roles = rolesFromClaims(claims);
  return roles.includes(ROLE_VIEWER) || roles.includes(ROLE_OPERATOR) || roles.includes(ROLE_ADMIN) || roles.includes(ROLE_SUPER_ADMIN);
}

// Lifecycle mutations (acknowledge/resolve/suppress, review/dismiss/archive) require operator or admin.
export function canMutateLifecycle(claims: AuthClaims | null | undefined): boolean {
  const roles = rolesFromClaims(claims);
  return roles.includes(ROLE_OPERATOR) || roles.includes(ROLE_ADMIN) || roles.includes(ROLE_SUPER_ADMIN);
}

// Display identity precedence matches the backend actor resolution: preferred_username -> email -> sub.
export function displayIdentity(claims: AuthClaims | null | undefined): string | undefined {
  if (!claims) return undefined;
  return claims.preferred_username || claims.email || claims.sub;
}

export function tenantFromClaims(claims: AuthClaims | null | undefined): string | undefined {
  return claims?.tenant_id;
}
