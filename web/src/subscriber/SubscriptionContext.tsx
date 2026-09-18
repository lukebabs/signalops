import { createContext, useContext, useMemo, type ReactNode } from 'react';
import { useTenant } from '../auth/session';
import { useSubscriberSubscription } from '../api/queries';
import type { SubscriberEffectiveSubscription, SubscriberSubscriptionFeature } from '../types';

interface SubscriptionContextValue {
  subscription: SubscriberEffectiveSubscription | null | undefined;
  loading: boolean;
  // Only the gateway's explicit state can enable browser locking. This keeps a
  // pre-provisioning rollout non-disruptive and leaves the server authoritative.
  enforcementEnabled: boolean;
  known: boolean;
  allows: (feature: SubscriberSubscriptionFeature | undefined) => boolean;
}

const SubscriptionContext = createContext<SubscriptionContextValue | null>(null);

export function SubscriptionProvider({ children }: { children: ReactNode }) {
  const tenantId = useTenant();
  const query = useSubscriberSubscription(tenantId);
  const subscription = query.data?.subscription;
  const enforcementEnabled = query.data?.enforcement_enabled === true;
  const known = Boolean(query.data && !query.isError && enforcementEnabled);
  const value = useMemo<SubscriptionContextValue>(() => ({
    subscription,
    loading: query.isLoading,
    enforcementEnabled,
    known,
    allows: (feature) => {
      if (!feature || !enforcementEnabled) return true;
      return subscription?.feature_policy?.[feature] === true;
    },
  }), [enforcementEnabled, query.isLoading, subscription, known]);
  return <SubscriptionContext.Provider value={value}>{children}</SubscriptionContext.Provider>;
}

export function useSubscription(): SubscriptionContextValue {
  const value = useContext(SubscriptionContext);
  if (!value) throw new Error('useSubscription must be used within SubscriptionProvider');
  return value;
}
