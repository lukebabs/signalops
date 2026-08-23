package api

import (
	"strings"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestBuildSyncraticDailyNarrativeAskPromptContracts(t *testing.T) {
	ctx := storage.SyncraticContextWindowRecord{
		ContextWindowID:       "synctx-daily",
		TenantID:              "tenant-local",
		AppID:                 "marketops",
		Domain:                "market_data",
		UseCase:               "daily_market_surveillance",
		SubjectType:           "market_scope",
		SubjectID:             dailyNarrativeStrategyOverview,
		SubjectSymbol:         dailyNarrativeSubjectSymbol,
		WindowStart:           time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC),
		WindowEnd:             time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC),
		ContextStrategy:       dailyNarrativeStrategyOverview,
		ContextBuilderVersion: dailyNarrativeBuilderVersion,
		SummaryMetricsJSON:    []byte(`{"strategy":"marketops_daily_overview_v1","sections":{"risk_reward":{"available":true}}}`),
		LineageRefsJSON:       []byte(`{"artifacts":{"risk_reward_snapshots":["rr-1"],"sri_snapshots":["sri-1"],"opportunities":["opp-1"]}}`),
		QualityWarningsJSON:   []byte(`[{"code":"sample_warning","message":"visible warning"}]`),
		EvidenceDigest:        "digest-1",
	}
	prompt, meta, err := buildSyncraticDailyNarrativeAskPrompt(ctx, syncraticAskRequest{MaxPromptBytes: 12000})
	if err != nil {
		t.Fatalf("prompt error: %v", err)
	}
	for _, required := range []string{"what_changed", "cited_artifacts", "data_quality_warnings", "rr-1", "sri-1", "opp-1"} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("prompt missing %s: %s", required, prompt)
		}
	}
	if meta.PromptBuilderVersion != dailyNarrativeAskPromptVersion || meta.ContextEvidenceDigest != "digest-1" || meta.PromptBytes != len(prompt) {
		t.Fatalf("unexpected meta: %+v", meta)
	}
	prompt2, meta2, err := buildSyncraticDailyNarrativeAskPrompt(ctx, syncraticAskRequest{MaxPromptBytes: 12000})
	if err != nil || prompt2 != prompt || meta2.PromptDigest != meta.PromptDigest {
		t.Fatalf("prompt digest is not stable: %v %+v %+v", err, meta, meta2)
	}
}

func TestBuildSyncraticDailyNarrativeAskPromptRejectsWrongVersion(t *testing.T) {
	ctx := storage.SyncraticContextWindowRecord{ContextStrategy: dailyNarrativeStrategyOverview, SummaryMetricsJSON: []byte(`{}`), LineageRefsJSON: []byte(`{}`), QualityWarningsJSON: []byte(`[]`)}
	_, _, err := buildSyncraticDailyNarrativeAskPrompt(ctx, syncraticAskRequest{PromptBuilderVersion: "wrong.version", MaxPromptBytes: 12000})
	if err == nil || !strings.Contains(err.Error(), dailyNarrativeAskPromptVersion) {
		t.Fatalf("expected version error, got %v", err)
	}
}
