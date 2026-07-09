// Pure helpers over OIDC token claims. No React, no DOM — fully unit-testable.
// Roles are accepted from BOTH Keycloak locations for forward compatibility.

export interface AuthClaims {
  sub?: string;
  preferred_username?: string;
  email?: string;
  tenant_id?: string;
  realm_access?: { roles?: string[] };
  resource_access?: Record<string, { roles?: string[] }>;
}

export const ROLE_VIEWER = 'signalops:viewer';
export const ROLE_OPERATOR = 'signalops:operator';
export const ROLE_ADMIN = 'signalops:admin';

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

// Read access to protected /v1/* requires viewer, operator, or admin.
export function canReadSignalOps(claims: AuthClaims | null | undefined): boolean {
  const roles = rolesFromClaims(claims);
  return roles.includes(ROLE_VIEWER) || roles.includes(ROLE_OPERATOR) || roles.includes(ROLE_ADMIN);
}

// Lifecycle mutations (acknowledge/resolve/suppress, review/dismiss/archive) require operator or admin.
export function canMutateLifecycle(claims: AuthClaims | null | undefined): boolean {
  const roles = rolesFromClaims(claims);
  return roles.includes(ROLE_OPERATOR) || roles.includes(ROLE_ADMIN);
}

// Display identity precedence matches the backend actor resolution: preferred_username -> email -> sub.
export function displayIdentity(claims: AuthClaims | null | undefined): string | undefined {
  if (!claims) return undefined;
  return claims.preferred_username || claims.email || claims.sub;
}

export function tenantFromClaims(claims: AuthClaims | null | undefined): string | undefined {
  return claims?.tenant_id;
}
