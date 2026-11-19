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
	for _, required := range []string{"response_contract", "Return only a valid JSON object", "Write to the MarketOps analyst", "relational natural language", "what_changed", "cited_artifacts", "data_quality_warnings", "rr-1", "sri-1", "opp-1"} {
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

func TestApplySyncraticAskResponseFallsBackWhenDailyAskReturnsNoText(t *testing.T) {
	ctx := storage.SyncraticContextWindowRecord{
		ContextWindowID:    "synctx-rr-empty-answer",
		TenantID:           "tenant-local",
		SubjectSymbol:      dailyNarrativeSubjectSymbol,
		WindowStart:        time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC),
		ContextStrategy:    dailyNarrativeStrategyRiskReward,
		SummaryMetricsJSON: []byte(`{"sections":{"risk_reward":{"breadth":{"bullish":13,"bearish":18,"neutral":101,"unavailable":0},"top_examples":[{"symbol":"WMT","direction":"bullish","score":50,"confidence":0.625,"risk_level":"medium"},{"symbol":"SO","direction":"bullish","score":45,"confidence":0.625,"risk_level":"medium"}]}}}`),
		EvidenceDigest:     "digest-rr-empty-answer",
	}
	insight := buildSyncraticDailyNarrativeInsight(ctx)
	updated := applySyncraticAskResponse(insight, ctx, syncraticAskPromptMeta{PromptBuilderVersion: dailyNarrativeAskPromptVersion, PromptDigest: "sha256:empty-answer", ContextEvidenceDigest: "digest-rr-empty-answer"}, userapi.AskResponse{QueryID: "ask-empty-answer"}, time.Now().UTC(), time.Now().UTC())
	if strings.Contains(updated.Explanation, "Syncratic Ask returned no textual explanation") {
		t.Fatalf("empty Ask answer should not be persisted as the Dashboard narrative: %s", updated.Explanation)
	}
	for _, required := range []string{"Executive summary", "Contextual read", "Risk/Reward breadth remains mostly neutral", "WMT is the clearest bullish exception", "Analyst follow-ups"} {
		if !strings.Contains(updated.Explanation, required) {
			t.Fatalf("fallback explanation missing %q: %s", required, updated.Explanation)
		}
	}
	if syncraticAskAlreadyApplied(updated, syncraticAskPromptMeta{PromptBuilderVersion: dailyNarrativeAskPromptVersion, PromptDigest: "sha256:empty-answer", ContextEvidenceDigest: "digest-rr-empty-answer"}) != true {
		t.Fatalf("usable deterministic fallback with completed Ask metadata should be reusable")
	}
	bad := updated
	bad.Explanation = "Syncratic Ask returned no textual explanation. Review deterministic evidence directly."
	if syncraticAskAlreadyApplied(bad, syncraticAskPromptMeta{PromptBuilderVersion: dailyNarrativeAskPromptVersion, PromptDigest: "sha256:empty-answer", ContextEvidenceDigest: "digest-rr-empty-answer"}) {
		t.Fatalf("no-text Ask fallback should not be treated as already applied")
	}
}

func TestApplySyncraticAskResponseFallsBackWhenDailyAnswerIsMetaCommentary(t *testing.T) {
	if !syncraticAskAnswerIsMetaCommentary("The main elements are the context metadata, summary metrics, and lineage references. Looking at the summary_metrics, the key pattern is mixed.") {
		t.Fatal("daily overview metadata commentary should force deterministic fallback")
	}

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
	if strings.Contains(strings.ToLower(updated.Explanation), "context includes a json") || !strings.Contains(updated.Explanation, "Risk/Reward breadth remains mostly neutral") || !strings.Contains(updated.Explanation, "WMT is the clearest bullish exception") {
		t.Fatalf("fallback explanation not analyst-facing: %s", updated.Explanation)
	}
	if strings.Contains(strings.ToLower(updated.Explanation), "score 50") || strings.Contains(strings.ToLower(updated.Explanation), "confidence 0.62") {
		t.Fatalf("fallback explanation should not recite raw risk/reward metrics: %s", updated.Explanation)
	}
	if !strings.HasPrefix(updated.Title, "Risk/Reward daily evolution") {
		t.Fatalf("fallback title = %q", updated.Title)
	}
}

func TestDeterministicDailyNarrativeFallbackAddsContextualRead(t *testing.T) {
	now := time.Now().UTC()
	sriCtx := storage.SyncraticContextWindowRecord{
		ContextWindowID:    "synctx-sri-contextual",
		TenantID:           "tenant-local",
		SubjectSymbol:      dailyNarrativeSubjectSymbol,
		WindowStart:        time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC),
		ContextStrategy:    dailyNarrativeStrategySRI,
		SummaryMetricsJSON: []byte(`{"sections":{"sri":{"leaders":[{"segment_id":"sri_industry_biotech","rank":1,"state":"LEADING","composite_score":97.83},{"segment_id":"sri_sector_utilities","rank":16,"state":"LAGGING","composite_score":15.83}]}}}`),
		EvidenceDigest:     "digest-sri-contextual",
	}
	sri := applySyncraticAskResponse(buildSyncraticDailyNarrativeInsight(sriCtx), sriCtx, syncraticAskPromptMeta{PromptBuilderVersion: dailyNarrativeAskPromptVersion, PromptDigest: "sha256:sri", ContextEvidenceDigest: "digest-sri-contextual"}, userapi.AskResponse{QueryID: "ask-sri", Answer: "The prompt asks for a JSON narrative."}, now, now)
	for _, required := range []string{"Contextual read", "biotech is the primary leadership pocket", "utilities is the weakest sampled pocket", "leadership posture"} {
		if !strings.Contains(sri.Explanation, required) {
			t.Fatalf("SRI fallback missing %q: %s", required, sri.Explanation)
		}
	}

	rrCtx := storage.SyncraticContextWindowRecord{
		ContextWindowID:    "synctx-rr-contextual",
		TenantID:           "tenant-local",
		SubjectSymbol:      dailyNarrativeSubjectSymbol,
		WindowStart:        time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC),
		ContextStrategy:    dailyNarrativeStrategyRiskReward,
		SummaryMetricsJSON: []byte(`{"sections":{"risk_reward":{"breadth":{"bullish":13,"bearish":18,"neutral":101,"unavailable":0},"top_examples":[{"symbol":"WMT","direction":"bullish","score":50,"confidence":0.625,"risk_level":"medium"}]}}}`),
		EvidenceDigest:     "digest-rr-contextual",
	}
	rr := applySyncraticAskResponse(buildSyncraticDailyNarrativeInsight(rrCtx), rrCtx, syncraticAskPromptMeta{PromptBuilderVersion: dailyNarrativeAskPromptVersion, PromptDigest: "sha256:rr", ContextEvidenceDigest: "digest-rr-contextual"}, userapi.AskResponse{QueryID: "ask-rr", Answer: "The JSON provided contains breadth."}, now, now)
	for _, required := range []string{"Contextual read", "Most monitored assets are still neutral", "modest bearish tilt", "WMT is the clearest bullish exception"} {
		if !strings.Contains(rr.Explanation, required) {
			t.Fatalf("Risk/Reward fallback missing %q: %s", required, rr.Explanation)
		}
	}

	reviewCtx := storage.SyncraticContextWindowRecord{
		ContextWindowID:    "synctx-review-contextual",
		TenantID:           "tenant-local",
		SubjectSymbol:      dailyNarrativeSubjectSymbol,
		WindowStart:        time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC),
		ContextStrategy:    dailyNarrativeStrategyReviewQueue,
		SummaryMetricsJSON: []byte(`{"sections":{"review_queue":{"status_counts":{"active":15,"expired":285},"active_examples":[{"symbol":"BABA","direction":"downside","score":0.96,"confidence":0.93,"last_evaluated_date":"2026-08-21","summary":"BABA downside convergence"}]}}}`),
		EvidenceDigest:     "digest-review-contextual",
	}
	review := applySyncraticAskResponse(buildSyncraticDailyNarrativeInsight(reviewCtx), reviewCtx, syncraticAskPromptMeta{PromptBuilderVersion: dailyNarrativeAskPromptVersion, PromptDigest: "sha256:review", ContextEvidenceDigest: "digest-review-contextual"}, userapi.AskResponse{QueryID: "ask-review", Answer: "The instructions request a JSON object."}, now, now)
	for _, required := range []string{"Contextual read", "Active opportunities are a small minority", "last evaluated 2026-08-21", "expired rows out of primary triage"} {
		if !strings.Contains(review.Explanation, required) {
			t.Fatalf("Review Queue fallback missing %q: %s", required, review.Explanation)
		}
	}
	for _, forbidden := range []string{"composite score 97.83", "score 50", "confidence 0.62", "opportunity scored 0.96"} {
		if strings.Contains(strings.ToLower(sri.Explanation+rr.Explanation+review.Explanation), strings.ToLower(forbidden)) {
			t.Fatalf("fallback explanation should use relational prose, found %q", forbidden)
		}
	}
}
