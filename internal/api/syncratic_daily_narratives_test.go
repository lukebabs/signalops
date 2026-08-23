package api

import (
	"strings"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
	"github.com/lukebabs/signalops/internal/syncratic/userapi"
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
	prompt, meta, err := buildSyncraticDailyNarrativeAskPrompt(ctx, syncraticAskRequest{MaxPromptBytes: 10000})
	if err != nil {
		t.Fatalf("prompt error: %v", err)
	}
	for _, required := range []string{"response_contract", "Return only a valid JSON object", "Write to the MarketOps analyst", "what_changed", "cited_artifacts", "data_quality_warnings", "rr-1", "sri-1", "opp-1"} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("prompt missing %s: %s", required, prompt)
		}
	}
	if meta.PromptBuilderVersion != dailyNarrativeAskPromptVersion || meta.ContextEvidenceDigest != "digest-1" || meta.PromptBytes != len(prompt) {
		t.Fatalf("unexpected meta: %+v", meta)
	}
	prompt2, meta2, err := buildSyncraticDailyNarrativeAskPrompt(ctx, syncraticAskRequest{MaxPromptBytes: 10000})
	if err != nil || prompt2 != prompt || meta2.PromptDigest != meta.PromptDigest {
		t.Fatalf("prompt digest is not stable: %v %+v %+v", err, meta, meta2)
	}
}

func TestBuildSyncraticDailyNarrativeAskPromptRejectsWrongVersion(t *testing.T) {
	ctx := storage.SyncraticContextWindowRecord{ContextStrategy: dailyNarrativeStrategyOverview, SummaryMetricsJSON: []byte(`{}`), LineageRefsJSON: []byte(`{}`), QualityWarningsJSON: []byte(`[]`)}
	_, _, err := buildSyncraticDailyNarrativeAskPrompt(ctx, syncraticAskRequest{PromptBuilderVersion: "wrong.version", MaxPromptBytes: 10000})
	if err == nil || !strings.Contains(err.Error(), dailyNarrativeAskPromptVersion) {
		t.Fatalf("expected version error, got %v", err)
	}
}

func TestBuildSyncraticDailyNarrativeAskPromptCompactsLargeOverview(t *testing.T) {
	manyRefs := strings.Builder{}
	manyRefs.WriteString(`{"strategy":"marketops_daily_overview_v1","artifacts":{"risk_reward_snapshots":[`)
	for i := 0; i < 140; i++ {
		if i > 0 {
			manyRefs.WriteString(",")
		}
		manyRefs.WriteString(`"rr-ref-`)
		manyRefs.WriteString(strings.Repeat("x", 24))
		manyRefs.WriteString(`"`)
	}
	manyRefs.WriteString(`],"sri_snapshots":[`)
	for i := 0; i < 180; i++ {
		if i > 0 {
			manyRefs.WriteString(",")
		}
		manyRefs.WriteString(`"sri-ref-`)
		manyRefs.WriteString(strings.Repeat("y", 24))
		manyRefs.WriteString(`"`)
	}
	manyRefs.WriteString(`]}}`)

	ctx := storage.SyncraticContextWindowRecord{
		ContextWindowID:       "synctx-daily-large",
		TenantID:              "tenant-local",
		SubjectSymbol:         dailyNarrativeSubjectSymbol,
		WindowStart:           time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC),
		ContextStrategy:       dailyNarrativeStrategyOverview,
		ContextBuilderVersion: dailyNarrativeBuilderVersion,
		SummaryMetricsJSON:    []byte(`{"strategy":"marketops_daily_overview_v1","sections":{"risk_reward":{"available":true,"breadth":{"bullish":10,"bearish":5},"top_examples":[{"symbol":"AAPL","score":90,"summary":"` + strings.Repeat("long ", 200) + `"},{"symbol":"NVDA","score":88}]},"sri":{"available":true,"leaders":[{"segment_id":"technology","rank":1},{"segment_id":"healthcare","rank":2},{"segment_id":"energy","rank":3},{"segment_id":"financials","rank":4}]},"review_queue":{"available":true,"active_examples":[{"symbol":"AAPL","summary":"` + strings.Repeat("active ", 200) + `"}]}}}`),
		LineageRefsJSON:       []byte(manyRefs.String()),
		QualityWarningsJSON:   []byte(`[]`),
		EvidenceDigest:        "digest-large",
	}
	prompt, meta, err := buildSyncraticDailyNarrativeAskPrompt(ctx, syncraticAskRequest{MaxPromptBytes: 10000})
	if err != nil {
		t.Fatalf("prompt should compact under budget: %v", err)
	}
	if len(prompt) > 10000 || meta.PromptBytes != len(prompt) {
		t.Fatalf("prompt did not respect compact budget: len=%d meta=%+v", len(prompt), meta)
	}
	if strings.Contains(prompt, "rr-ref-xxxxxxxxxxxxxxxxxxxxxxxx") && strings.Count(prompt, "rr-ref-") > 12 {
		t.Fatalf("prompt included too many risk/reward refs: %d", strings.Count(prompt, "rr-ref-"))
	}
	for _, required := range []string{"overview_synthesis_compact", "lineage_ref_summary", "total", "sample", "full_provenance_location"} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("compact prompt missing %s: %s", required, prompt)
		}
	}
}

func TestApplySyncraticAskResponseFallsBackWhenDailyAnswerIsMetaCommentary(t *testing.T) {
	ctx := storage.SyncraticContextWindowRecord{
		ContextWindowID:    "synctx-rr",
		TenantID:           "tenant-local",
		SubjectSymbol:      dailyNarrativeSubjectSymbol,
		WindowStart:        time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC),
		ContextStrategy:    dailyNarrativeStrategyRiskReward,
		SummaryMetricsJSON: []byte(`{"sections":{"risk_reward":{"breadth":{"bullish":13,"bearish":18,"neutral":101,"unavailable":0},"top_examples":[{"symbol":"WMT","direction":"bullish","score":50,"confidence":0.625,"risk_level":"medium"},{"symbol":"SO","direction":"bullish","score":45,"confidence":0.625,"risk_level":"medium"}]}}}`),
		EvidenceDigest:     "digest-rr",
	}
	insight := buildSyncraticDailyNarrativeInsight(ctx)
	updated := applySyncraticAskResponse(insight, ctx, syncraticAskPromptMeta{PromptBuilderVersion: dailyNarrativeAskPromptVersion, PromptDigest: "sha256:test", ContextEvidenceDigest: "digest-rr"}, userapi.AskResponse{QueryID: "ask-meta", Answer: "The context includes a JSON with instructions and summary_metrics."}, time.Now().UTC(), time.Now().UTC())
	if strings.Contains(strings.ToLower(updated.Explanation), "context includes a json") || !strings.Contains(updated.Explanation, "Risk/Reward breadth was 13 bullish, 18 bearish, 101 neutral") || !strings.Contains(updated.Explanation, "WMT was bullish") {
		t.Fatalf("fallback explanation not analyst-facing: %s", updated.Explanation)
	}
	if !strings.HasPrefix(updated.Title, "Risk/Reward daily evolution") {
		t.Fatalf("fallback title = %q", updated.Title)
	}
}
