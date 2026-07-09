import { describe, expect, it } from 'vitest';
import {
  canMutateLifecycle,
  canReadSignalOps,
  displayIdentity,
  hasRole,
  rolesFromClaims,
  tenantFromClaims,
  type AuthClaims,
} from './claims';

const claims = (o: Partial<AuthClaims>): AuthClaims => o as AuthClaims;

describe('auth claims', () => {
  it('extracts tenant_id', () => {
    expect(tenantFromClaims(claims({ tenant_id: 'tenant-local' }))).toBe('tenant-local');
    expect(tenantFromClaims(null)).toBeUndefined();
  });

  it('display identity precedence: preferred_username -> email -> sub', () => {
    expect(displayIdentity(claims({ preferred_username: 'lukeb', email: 'luke@x.io', sub: 's1' }))).toBe('lukeb');
    expect(displayIdentity(claims({ email: 'luke@x.io', sub: 's1' }))).toBe('luke@x.io');
    expect(displayIdentity(claims({ sub: 's1' }))).toBe('s1');
    expect(displayIdentity(null)).toBeUndefined();
  });

  it('merges roles from realm_access and resource_access.signalops-api', () => {
    const c = claims({
      realm_access: { roles: ['signalops:viewer', 'default-roles-syncratic'] },
      resource_access: { 'signalops-api': { roles: ['signalops:operator'] } },
    });
    expect(rolesFromClaims(c).sort()).toEqual([
      'default-roles-syncratic',
      'signalops:operator',
      'signalops:viewer',
    ]);
  });

  it('viewers can read but cannot mutate; operators/admins can mutate', () => {
    const viewer = claims({ realm_access: { roles: ['signalops:viewer'] } });
    const operator = claims({
      resource_access: { 'signalops-api': { roles: ['signalops:operator'] } },
    });
    const admin = claims({ realm_access: { roles: ['signalops:admin'] } });

    expect(canReadSignalOps(viewer)).toBe(true);
    expect(canMutateLifecycle(viewer)).toBe(false);
    expect(canMutateLifecycle(operator)).toBe(true);
    expect(canMutateLifecycle(admin)).toBe(true);
    expect(hasRole(admin, 'signalops:admin')).toBe(true);
    // A principal with no SignalOps role can neither read nor mutate.
    expect(canReadSignalOps(claims({ realm_access: { roles: ['unrelated'] } }))).toBe(false);
  });
});
