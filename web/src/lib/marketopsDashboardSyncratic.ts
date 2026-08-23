import type { SyncraticInsight } from "../types";

export const DASHBOARD_DAILY_NARRATIVE_INSIGHT_TYPE = "marketops.syncratic.daily_narrative.v1";

export const DASHBOARD_NARRATIVE_STRATEGIES = [
  { strategy: "marketops_daily_overview_v1", label: "Daily Overview" },
  { strategy: "marketops_sri_daily_v1", label: "Sector Rotation" },
  { strategy: "marketops_risk_reward_daily_v1", label: "Risk/Reward" },
  { strategy: "marketops_review_queue_daily_v1", label: "Review Queue" },
] as const;

export interface DashboardSyncraticNarrativeCard {
  strategy: typeof DASHBOARD_NARRATIVE_STRATEGIES[number]["strategy"];
  label: typeof DASHBOARD_NARRATIVE_STRATEGIES[number]["label"];
  insight: SyncraticInsight;
}

export function dashboardSyncraticNarrativeCards(
  narratives: SyncraticInsight[],
): DashboardSyncraticNarrativeCard[] {
  return DASHBOARD_NARRATIVE_STRATEGIES.flatMap((item) => {
    const insight = narratives
      .filter((candidate) => narrativeStrategy(candidate) === item.strategy)
      .sort((a, b) => String(b.updated_at).localeCompare(String(a.updated_at)))[0];
    return insight ? [{ ...item, insight }] : [];
  });
}

export function narrativeStrategy(insight: SyncraticInsight): string {
  const metrics = insight.metrics;
  if (!metrics || typeof metrics !== "object") return "";
  const strategy = (metrics as Record<string, unknown>).strategy;
  return typeof strategy === "string" ? strategy : "";
}

export function compactNarrative(value: string): string {
  return value.replace(/\s+/g, " ").trim();
}
