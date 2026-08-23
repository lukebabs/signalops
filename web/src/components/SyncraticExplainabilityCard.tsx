import { Link } from '@tanstack/react-router';
import { Lock, Sparkles } from 'lucide-react';

import { useSubscription } from '../subscriber/SubscriptionContext';

const FEATURE = 'syncratic_explainability' as const;

interface SyncraticExplainabilityCardProps {
  surface: string;
  title?: string;
  description: string;
  mode?: 'compact' | 'full';
}

export function SyncraticExplainabilityCard({
  surface,
  title = 'Explain with Syncratic',
  description,
  mode = 'full',
}: SyncraticExplainabilityCardProps) {
  const subscription = useSubscription();
  const canInteract = subscription.allows(FEATURE);
  const tier = subscription.subscription?.display_name || subscription.subscription?.product_key || 'Explorer';
  const compact = mode === 'compact';

  return (
    <section
      aria-label={`${surface} Syncratic explainability`}
      className={`rounded border border-brand-200 bg-brand-50/70 text-brand-950 dark:border-brand-800 dark:bg-brand-950/30 dark:text-brand-100 ${compact ? 'p-2' : 'p-3'}`}
    >
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0">
          <div className="flex items-center gap-1 text-sm font-semibold">
            <Sparkles size={14} className="shrink-0 text-brand-700 dark:text-brand-300" />
            <span>{title}</span>
          </div>
          <p className="mt-1 max-w-3xl text-xs leading-5 text-brand-900 dark:text-brand-100">{description}</p>
          <p className="mt-1 text-[11px] text-brand-800 dark:text-brand-200">
            Syncratic uses persisted MarketOps context windows only. It does not poll providers or change official signals.
          </p>
        </div>
        <div className="flex shrink-0 flex-wrap items-center gap-2">
          {!canInteract && subscription.enforcementEnabled ? (
            <span className="inline-flex items-center gap-1 rounded border border-amber-300 bg-amber-50 px-2 py-1 text-[11px] font-medium text-amber-800 dark:border-amber-700 dark:bg-amber-950/50 dark:text-amber-200">
              <Lock size={12} /> Professional Ask required
            </span>
          ) : (
            <span className="rounded border border-emerald-300 bg-emerald-50 px-2 py-1 text-[11px] font-medium text-emerald-800 dark:border-emerald-800 dark:bg-emerald-950/50 dark:text-emerald-200">
              {subscription.enforcementEnabled ? `${tier} interactive` : 'Interactive controls available'}
            </span>
          )}
          <Link
            to="/marketops/syncratic"
            className="inline-flex items-center gap-1 rounded bg-brand-600 px-2.5 py-1 text-xs font-medium text-white hover:bg-brand-700"
          >
            Open Syncratic Intelligence
          </Link>
        </div>
      </div>
    </section>
  );
}
