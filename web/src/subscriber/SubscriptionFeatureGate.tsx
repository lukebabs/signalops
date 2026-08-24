import { useEffect, useRef, type ReactNode } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { ArrowRight, LockKeyhole } from 'lucide-react';
import type { SubscriberSubscriptionFeature } from '../types';
import { api } from '../api/client';
import { useSubscription } from './SubscriptionContext';

export function SubscriptionFeatureGate({ feature, title, children }: { feature: SubscriberSubscriptionFeature; title: string; children: ReactNode }) {
  const { known, allows, subscription } = useSubscription();
  const navigate = useNavigate();
  const recordedShownRef = useRef('');
  const locked = known && !allows(feature);
  const requiredTier = requiredTierForFeature(feature);
  const currentTier = subscription?.product_key ?? 'unprovisioned';
  const routeContext = currentRouteContext();
  const copy = upgradeCopyForFeature(feature, title);

  useEffect(() => {
    if (!locked) return;
    const key = `${feature}:${routeContext.route}:${currentTier}:${requiredTier}`;
    if (recordedShownRef.current === key) return;
    recordedShownRef.current = key;
    void recordUpgradeInteraction('prompt_shown', feature, currentTier, requiredTier, title).catch(() => undefined);
  }, [currentTier, feature, locked, requiredTier, routeContext.route, title]);

  if (!locked) return <>{children}</>;

  const openPricing = () => {
    void recordUpgradeInteraction('prompt_clicked', feature, currentTier, requiredTier, title).catch(() => undefined);
    void navigate({ to: '/marketops/pricing', search: { source_feature: feature, return_url: routeContext.returnUrl } });
  };

  return <section className="mx-auto max-w-2xl space-y-4 rounded border border-amber-200 bg-amber-50 p-5 text-center dark:border-amber-800 dark:bg-amber-950/40">
    <LockKeyhole className="mx-auto text-amber-700 dark:text-amber-300" aria-hidden="true" />
    <div>
      <p className="text-xs font-semibold uppercase tracking-wide text-amber-700 dark:text-amber-300">Additional analytical depth available</p>
      <h1 className="mt-1 text-lg font-semibold text-gray-950 dark:text-gray-50">{copy.heading}</h1>
      <p className="mt-2 text-sm leading-6 text-gray-700 dark:text-gray-200">{copy.body}</p>
    </div>
    <div className="rounded border border-amber-200 bg-white p-3 text-left text-xs text-gray-700 dark:border-amber-800 dark:bg-gray-950 dark:text-gray-200">
      <div className="font-semibold">Current access: {subscription?.display_name ?? 'Unprovisioned'}</div>
      <div className="mt-1">Required depth: {requiredTier === 'institutional' ? 'Institutional' : 'Professional'}</div>
      <div className="mt-1 text-gray-500 dark:text-gray-400">Market data remains centrally governed; upgrading changes analytical access, not data ownership.</div>
    </div>
    <button type="button" onClick={openPricing} className="inline-flex items-center gap-2 rounded bg-brand-700 px-3 py-1.5 text-sm font-medium text-white hover:bg-brand-800">
      View upgrade options <ArrowRight size={15} />
    </button>
  </section>;
}

type UpgradeInteractionType = 'prompt_shown' | 'prompt_clicked';

async function recordUpgradeInteraction(interactionType: UpgradeInteractionType, feature: SubscriberSubscriptionFeature, currentTier: string, requiredTier: 'professional' | 'institutional', title: string): Promise<void> {
  const route = currentRouteContext();
  await api.recordSubscriberUpgradeInteraction({
    interaction_type: interactionType,
    app_id: 'marketops',
    source_feature: feature,
    source_route: route.route,
    source_url: route.url,
    asset_symbol: route.symbol,
    current_tier: currentTier,
    required_tier: requiredTier,
    prompt_variant: 'contextual_route_gate_v1',
    cta_label: interactionType === 'prompt_clicked' ? 'View upgrade options' : '',
    correlation_id: `upgrade-${interactionType}-${Date.now()}`,
    metadata: { title },
  });
}

function requiredTierForFeature(feature: SubscriberSubscriptionFeature): 'professional' | 'institutional' {
  if (['signal_assurance_analytics', 'portfolio_analysis', 'batch_screening', 'historical_replay', 'strategy_validation', 'custom_universes', 'api', 'white_label'].includes(feature)) return 'institutional';
  return 'professional';
}

function upgradeCopyForFeature(feature: SubscriberSubscriptionFeature, title: string): { heading: string; body: string } {
  switch (feature) {
    case 'value_intelligence':
      return { heading: 'Understand valuation before going deeper.', body: 'Professional adds valuation context, evidence, and research workflow depth for the assets you are already investigating.' };
    case 'distressed_opportunity_intelligence':
      return { heading: 'Evaluate whether the opportunity is distressed or improving.', body: 'Professional adds Distressed Opportunity Intelligence so you can separate ordinary weakness from evidence-backed reversal context.' };
    case 'earnings_opportunity_intelligence':
      return { heading: 'Analyze the event setup before earnings arrive.', body: 'Professional adds earnings-event context across technical setup, risk/reward, valuation, sector behavior, and options positioning.' };
    case 'signal_assurance_analytics':
      return { heading: 'Validate signal effectiveness at institutional depth.', body: 'Institutional adds Signal Assurance analytics, historical replay, portfolio-scale review, strategy validation, custom universes, and API workflows.' };
    default:
      return { heading: `${title} requires deeper MarketOps access.`, body: 'This workflow answers the next research question after discovery. Upgrade options preserve your current research context and watchlists.' };
  }
}

function currentRouteContext(): { route: string; url: string; returnUrl: string; symbol: string } {
  if (typeof window === 'undefined') return { route: '', url: '', returnUrl: '', symbol: '' };
  const route = window.location.pathname;
  const returnUrl = `${window.location.pathname}${window.location.search}${window.location.hash}`;
  const params = new URLSearchParams(window.location.search);
  const symbol = (params.get('symbol') || params.get('asset') || '').toUpperCase();
  return { route, url: window.location.href, returnUrl, symbol };
}
