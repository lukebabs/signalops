import type { ReactNode } from 'react';
import { LockKeyhole } from 'lucide-react';
import type { SubscriberSubscriptionFeature } from '../types';
import { useSubscription } from './SubscriptionContext';

export function SubscriptionFeatureGate({ feature, title, children }: { feature: SubscriberSubscriptionFeature; title: string; children: ReactNode }) {
  const { known, allows, subscription } = useSubscription();
  if (!known || allows(feature)) return <>{children}</>;
  return <section className="mx-auto max-w-xl space-y-3 rounded border border-amber-200 bg-amber-50 p-5 text-center">
    <LockKeyhole className="mx-auto text-amber-700" aria-hidden="true" />
    <div><h1 className="text-lg font-semibold text-gray-900">{title} requires additional analytical depth</h1><p className="mt-1 text-sm text-gray-700">Your current {subscription?.display_name ?? 'unprovisioned'} access does not include this capability. Market data remains centrally governed; upgrading changes analysis access, not data ownership.</p></div>
    <p className="text-xs text-gray-600">Professional includes Value, Distressed Opportunity, Earnings Opportunity, detailed Sector Rotation, options and research. Institutional adds Signal Assurance, replay, portfolio and API capabilities.</p>
  </section>;
}
