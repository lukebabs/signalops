import { describe, expect, it } from "vitest";

import {
  compactNarrative,
  dashboardSyncraticNarrativeCards,
  fullExplainabilityNarrative,
  DASHBOARD_DAILY_NARRATIVE_INSIGHT_TYPE,
} from "./marketopsDashboardSyncratic";
import type { SyncraticInsight } from "../types";

function insight(strategy: string, updatedAt: string, id: string): SyncraticInsight {
  return {
    syncratic_insight_id: id,
    tenant_id: "tenant-local",
    app_id: "marketops",
    domain: "marketops",
    use_case: "dashboard",
    context_window_id: `ctx-${id}`,
    insight_type: DASHBOARD_DAILY_NARRATIVE_INSIGHT_TYPE,
    subject_type: "market",
    subject_id: "MARKETOPS",
    subject_symbol: "MARKETOPS",
    status: "active",
    severity: "low",
    confidence: 0.65,
    title: id,
    summary: "summary",
    explanation: "explanation",
    supporting_alert_ids: [],
    supporting_signal_ids: [],
    supporting_event_ids: [],
    supporting_artifact_ids: [],
    related_graph_proposal_ids: [],
    related_label_ids: [],
    metrics: { strategy },
    recommendation: {},
    builder_version: "test",
    created_at: updatedAt,
    updated_at: updatedAt,
  };
}

describe("dashboardSyncraticNarrativeCards", () => {
  it("selects the newest narrative for each dashboard strategy in display order", () => {
    const cards = dashboardSyncraticNarrativeCards([
      insight("marketops_sri_daily_v1", "2026-08-21T20:00:00Z", "sri-old"),
      insight("marketops_risk_reward_daily_v1", "2026-08-21T20:00:00Z", "risk"),
      insight("marketops_daily_overview_v1", "2026-08-21T20:00:00Z", "daily"),
      insight("marketops_sri_daily_v1", "2026-08-22T20:00:00Z", "sri-new"),
      insight("unsupported", "2026-08-22T20:00:00Z", "ignored"),
    ]);

    expect(cards.map((card) => card.label)).toEqual(["Daily Overview", "Sector Rotation", "Risk/Reward"]);
    expect(cards.map((card) => card.insight.syncratic_insight_id)).toEqual(["daily", "sri-new", "risk"]);
  });

  it("compacts narrative whitespace for dashboard excerpts", () => {
    expect(compactNarrative("  Sector\nrotation   improved.  ")).toBe("Sector rotation improved.");
  });

  it("preserves full explainability line breaks for the expanded dashboard panel", () => {
    const record = insight("marketops_risk_reward_daily_v1", "2026-08-21T20:00:00Z", "risk");
    record.explanation = "Executive summary:\nSession date: 2026-08-21.\n\nTop drivers:\n- WMT is constructive.";

    expect(fullExplainabilityNarrative(record)).toContain("Executive summary:\nSession date");
    expect(fullExplainabilityNarrative(record)).toContain("Top drivers:\n- WMT");
  });
});
