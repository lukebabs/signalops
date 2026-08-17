import { createContext, useContext, useMemo, type ReactNode } from 'react';
import { useTenant } from '../auth/session';
import { useSubscriberSubscription } from '../api/queries';
import type { SubscriberEffectiveSubscription, SubscriberSubscriptionFeature } from '../types';

interface SubscriptionContextValue {
  subscription: SubscriberEffectiveSubscription | null | undefined;
  loading: boolean;
  // Unknown is deliberately permissive while the foundation is not deployed or
  // commercial enforcement is disabled. The gateway remains authoritative and
  // returns 402 once enforcement is enabled.
  known: boolean;
  allows: (feature: SubscriberSubscriptionFeature | undefined) => boolean;
}

const SubscriptionContext = createContext<SubscriptionContextValue | null>(null);

export function SubscriptionProvider({ children }: { children: ReactNode }) {
  const tenantId = useTenant();
  const query = useSubscriberSubscription(tenantId);
  const subscription = query.data?.subscription;
  const known = Boolean(query.data && !query.isError);
  const value = useMemo<SubscriptionContextValue>(() => ({
    subscription,
    loading: query.isLoading,
    known,
    allows: (feature) => {
      if (!feature || !known) return true;
      return subscription?.feature_policy?.[feature] === true;
    },
  }), [known, query.isLoading, subscription]);
  return <SubscriptionContext.Provider value={value}>{children}</SubscriptionContext.Provider>;
}

export function useSubscription(): SubscriptionContextValue {
  const value = useContext(SubscriptionContext);
  if (!value) throw new Error('useSubscription must be used within SubscriptionProvider');
  return value;
}
