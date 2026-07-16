package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/appmeta"
	marketopsbacktest "github.com/lukebabs/signalops/internal/marketops/backtest"
	"github.com/lukebabs/signalops/internal/storage"
	"github.com/lukebabs/signalops/pkg/broker"
)

const (
	defaultMaxRawEventBytes = 1 << 20
	defaultPublishTimeout   = 5 * time.Second
	defaultStreamInterval   = 5 * time.Second
)

var supportedDashboardStreamChannels = map[string]struct{}{
	"health":         {},
	"scheduler_run":  {},
	"runs":           {},
	"raw_event":      {},
	"raw_events":     {},
	"provider_usage": {},
	"heartbeat":      {},
}

// RouterConfig contains process-local API wiring options.
type RouterConfig struct {
	ServiceName             string
	MarketOpsBacktestRunner func(context.Context, storage.MarketOpsBacktestRepository, marketopsbacktest.Config) (marketopsbacktest.Result, error)
	Auth                    AuthConfig
	Publisher               broker.Publisher
	RawTopic                string
	QueryRepository         storage.QueryRepository
	PublishRepository       storage.PublishRepository
	SyncraticAskClient      syncraticAskClient
}

// NewRouter creates the HTTP routes owned by the SignalOps gateway.
func NewRouter(cfg RouterConfig) http.Handler {
	mux := http.NewServeMux()
	serviceName := cfg.ServiceName
	if serviceName == "" {
		serviceName = "signalops"
	}
	rawTopic := cfg.RawTopic

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"service": serviceName,
			"time":    time.Now().UTC().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "ready",
			"service": serviceName,
			"time":    time.Now().UTC().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("GET /v1/scheduler/runs", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		runs, err := repo.ListSchedulerRuns(r.Context(), queryLimit(r, 50))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list scheduler runs")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"runs": schedulerRunResponses(runs)})
	})

	mux.HandleFunc("GET /v1/scheduler/runs/{run_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		record, err := repo.GetSchedulerRun(r.Context(), r.PathValue("run_id"))
		if err != nil {
			writeQueryError(w, err, "scheduler_run_not_found", "scheduler run not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"run": schedulerRunResponse(record)})
	})

	mux.HandleFunc("GET /v1/replay/jobs", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		records, err := repo.ListReplayJobs(r.Context(), storage.ReplayJobFilter{
			TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), SourceID: strings.TrimSpace(r.URL.Query().Get("source_id")),
			Dataset: strings.TrimSpace(r.URL.Query().Get("dataset")), SourceKind: strings.TrimSpace(r.URL.Query().Get("source_kind")),
			Status: strings.TrimSpace(r.URL.Query().Get("status")), Limit: queryLimit(r, 50),
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list replay jobs")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"replay_jobs": replayJobResponses(records)})
	})

	mux.HandleFunc("POST /v1/replay/jobs", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		req, err := readReplayJobRequest(w, r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		record, err := replayJobRecordFromRequest(req, replayActor(r, req.RequestedBy), time.Now().UTC())
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_replay_job", err.Error())
			return
		}
		if err := repo.UpsertReplayJob(r.Context(), record); err != nil {
			writeError(w, http.StatusServiceUnavailable, "persistence_failed", "failed to persist replay job")
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]any{"replay_job": replayJobResponse(record)})
	})

	mux.HandleFunc("GET /v1/replay/jobs/{replay_job_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		record, err := repo.GetReplayJob(r.Context(), r.PathValue("replay_job_id"))
		if err != nil {
			writeQueryError(w, err, "replay_job_not_found", "replay job not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"replay_job": replayJobResponse(record)})
	})

	mux.HandleFunc("POST /v1/replay/jobs/{replay_job_id}/cancel", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		req, err := readLifecycleMutationRequest(w, r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		record, err := repo.CancelReplayJob(r.Context(), r.PathValue("replay_job_id"), lifecycleActor(r, req.Actor), time.Now().UTC(), firstNonEmpty(req.Reason, req.Note), nil)
		if err != nil {
			writeQueryError(w, err, "replay_job_not_found", "replay job not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"replay_job": replayJobResponse(record)})
	})

	mux.HandleFunc("GET /v1/replay/status", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		now := time.Now().UTC()
		counts, err := repo.CountReplayJobsByStatus(r.Context(), tenantID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to count replay jobs")
			return
		}
		workers, err := repo.ListReplayWorkerHeartbeats(r.Context(), queryLimit(r, 20))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list replay worker heartbeats")
			return
		}
		latestJobs, err := repo.ListReplayJobs(r.Context(), storage.ReplayJobFilter{TenantID: tenantID, Limit: 5})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list replay jobs")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"replay_status": replayStatusResponse(now, counts, workers, latestJobs)})
	})

	mux.HandleFunc("GET /v1/app-profiles", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"app_profiles": appmeta.Profiles})
	})

	mux.HandleFunc("GET /v1/provider-usage", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		records, err := repo.ListProviderUsage(r.Context(), strings.TrimSpace(r.URL.Query().Get("run_id")), queryLimit(r, 50))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list provider usage")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"provider_usage": providerUsageResponses(records)})
	})

	mux.HandleFunc("GET /v1/raw-events", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		records, err := repo.ListRawEventLedger(r.Context(), storage.RawEventLedgerFilter{
			TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")),
			AppID:    strings.TrimSpace(r.URL.Query().Get("app_id")),
			Domain:   strings.TrimSpace(r.URL.Query().Get("domain")),
			UseCase:  strings.TrimSpace(r.URL.Query().Get("use_case")),
			SourceID: strings.TrimSpace(r.URL.Query().Get("source_id")),
			Dataset:  strings.TrimSpace(r.URL.Query().Get("dataset")),
			Limit:    queryLimit(r, 50),
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list raw events")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"raw_events": rawEventResponses(records)})
	})

	mux.HandleFunc("GET /v1/raw-events/{event_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		record, err := repo.GetRawEventLedger(r.Context(), r.PathValue("event_id"))
		if err != nil {
			writeQueryError(w, err, "raw_event_not_found", "raw event not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"raw_event": rawEventResponse(record)})
	})

	mux.HandleFunc("GET /v1/normalized-events", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		records, err := repo.ListNormalizedEventLedger(r.Context(), storage.RawEventLedgerFilter{
			TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), AppID: strings.TrimSpace(r.URL.Query().Get("app_id")), Domain: strings.TrimSpace(r.URL.Query().Get("domain")), UseCase: strings.TrimSpace(r.URL.Query().Get("use_case")), SourceID: strings.TrimSpace(r.URL.Query().Get("source_id")),
			Dataset: strings.TrimSpace(r.URL.Query().Get("dataset")), Limit: queryLimit(r, 50),
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list normalized events")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"normalized_events": normalizedEventResponses(records)})
	})

	mux.HandleFunc("GET /v1/normalized-events/{event_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		record, err := repo.GetNormalizedEventLedger(r.Context(), r.PathValue("event_id"))
		if err != nil {
			writeQueryError(w, err, "normalized_event_not_found", "normalized event not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"normalized_event": normalizedEventResponse(record)})
	})

	mux.HandleFunc("GET /v1/signals", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		records, err := repo.ListSignalLedger(r.Context(), storage.SignalLedgerFilter{
			TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), AppID: strings.TrimSpace(r.URL.Query().Get("app_id")), Domain: strings.TrimSpace(r.URL.Query().Get("domain")), UseCase: strings.TrimSpace(r.URL.Query().Get("use_case")), SourceID: strings.TrimSpace(r.URL.Query().Get("source_id")),
			Dataset: strings.TrimSpace(r.URL.Query().Get("dataset")), DetectorID: strings.TrimSpace(r.URL.Query().Get("detector_id")),
			Severity: strings.TrimSpace(r.URL.Query().Get("severity")), Limit: queryLimit(r, 50),
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list signals")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"signals": signalResponses(records)})
	})

	mux.HandleFunc("GET /v1/signals/{signal_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		record, err := repo.GetSignalLedger(r.Context(), r.PathValue("signal_id"))
		if err != nil {
			writeQueryError(w, err, "signal_not_found", "signal not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"signal": signalResponse(record)})
	})

	mux.HandleFunc("GET /v1/marketops/backtest-coverage", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		windowStart, err := optionalQueryTime(r, "window_start")
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_coverage_filter", "window_start must be an RFC3339 timestamp")
			return
		}
		windowEnd, err := optionalQueryTime(r, "window_end")
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_coverage_filter", "window_end must be an RFC3339 timestamp")
			return
		}
		if !windowStart.IsZero() && !windowEnd.IsZero() && !windowEnd.After(windowStart) {
			writeError(w, http.StatusBadRequest, "invalid_coverage_filter", "window_end must be after window_start")
			return
		}
		filter := storage.MarketOpsBacktestCoverageFilter{TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), AppID: firstNonEmptyBacktestValue(r.URL.Query().Get("app_id"), "marketops"), Domain: firstNonEmptyBacktestValue(r.URL.Query().Get("domain"), "market_data"), UseCase: firstNonEmptyBacktestValue(r.URL.Query().Get("use_case"), "daily_market_surveillance"), SourceID: strings.TrimSpace(r.URL.Query().Get("source_id")), SourceAdapter: strings.TrimSpace(r.URL.Query().Get("source_adapter")), Dataset: strings.TrimSpace(r.URL.Query().Get("dataset")), Symbols: querySymbols(r), WindowStart: windowStart, WindowEnd: windowEnd, Limit: queryLimit(r, 100)}
		if filter.TenantID == "" {
			writeError(w, http.StatusBadRequest, "invalid_coverage_filter", "tenant_id is required")
			return
		}
		records, err := repo.ListMarketOpsBacktestCoverage(r.Context(), filter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps backtest coverage")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"coverage": marketOpsBacktestCoverageResponses(records)})
	})

	mux.HandleFunc("POST /v1/marketops/backtest-campaigns", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		defer r.Body.Close()
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, defaultMaxRawEventBytes))
		decoder.DisallowUnknownFields()
		var req marketOpsBacktestCampaignCreateRequest
		if err := decoder.Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		runner := cfg.MarketOpsBacktestRunner
		if runner == nil {
			runner = marketopsbacktest.Run
		}
		campaignID := strings.TrimSpace(req.CampaignID)
		if campaignID == "" {
			campaignID = newID("btcamp_marketops")
		}
		record, childConfigs, err := marketOpsBacktestCampaignPlan(r.Context(), repo, req, campaignID, replayActor(r, req.RequestedBy))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_backtest_campaign", err.Error())
			return
		}
		if err := repo.UpsertMarketOpsBacktestCampaign(r.Context(), record); err != nil {
			writeError(w, http.StatusInternalServerError, "persist_failed", "failed to persist MarketOps backtest campaign")
			return
		}
		metrics := map[string]any{"planned_runs": len(childConfigs), "completed_runs": 0, "failed_runs": 0, "scanned": 0, "signals": 0, "artifacts": 0, "graph_proposals": 0, "policy_results": 0, "recommendation_counts": map[string]int{}, "datasets": record.DatasetScope, "symbols": record.Symbols, "window_step_days": record.WindowStepDays}
		childRunIDs := []string{}
		var runErr error
		for _, childCfg := range childConfigs {
			result, err := runner(r.Context(), repo, childCfg)
			childRunIDs = append(childRunIDs, childCfg.RunID)
			if err != nil {
				metrics["failed_runs"] = metrics["failed_runs"].(int) + 1
				runErr = err
				break
			}
			metrics["completed_runs"] = metrics["completed_runs"].(int) + 1
			metrics["scanned"] = metrics["scanned"].(int) + result.Metrics.Scanned
			metrics["signals"] = metrics["signals"].(int) + result.Metrics.Signals
			metrics["artifacts"] = metrics["artifacts"].(int) + result.Metrics.Artifacts
			metrics["graph_proposals"] = metrics["graph_proposals"].(int) + result.Metrics.GraphProposals
			metrics["policy_results"] = metrics["policy_results"].(int) + result.Metrics.PolicyResults
			recCounts := metrics["recommendation_counts"].(map[string]int)
			for key, count := range result.Metrics.RecommendationCounts {
				recCounts[key] += count
			}
		}
		completedAt := time.Now().UTC()
		record.ChildRunIDs = childRunIDs
		record.CompletedAt = &completedAt
		if runErr != nil {
			record.Status = storage.RunStatusFailed
			record.ErrorMessage = runErr.Error()
		} else {
			record.Status = storage.RunStatusSucceeded
		}
		metrics["child_run_ids"] = childRunIDs
		record.MetricsJSON = mustJSON(metrics)
		if err := repo.UpsertMarketOpsBacktestCampaign(r.Context(), record); err != nil {
			writeError(w, http.StatusInternalServerError, "persist_failed", "failed to persist completed MarketOps backtest campaign")
			return
		}
		stored, err := repo.GetMarketOpsBacktestCampaign(r.Context(), campaignID)
		if err != nil {
			writeQueryError(w, err, "backtest_campaign_not_found", "MarketOps backtest campaign not found")
			return
		}
		writeJSON(w, http.StatusCreated, marketOpsBacktestCampaignCreateResponse{Campaign: marketOpsBacktestCampaignResponse(stored)})
	})

	mux.HandleFunc("GET /v1/marketops/backtest-campaigns", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		records, err := repo.ListMarketOpsBacktestCampaigns(r.Context(), storage.MarketOpsBacktestCampaignFilter{TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), AppID: strings.TrimSpace(r.URL.Query().Get("app_id")), Domain: strings.TrimSpace(r.URL.Query().Get("domain")), UseCase: strings.TrimSpace(r.URL.Query().Get("use_case")), SourceID: strings.TrimSpace(r.URL.Query().Get("source_id")), DetectorID: strings.TrimSpace(r.URL.Query().Get("detector_id")), UniverseGroup: strings.TrimSpace(r.URL.Query().Get("universe_group")), Status: strings.TrimSpace(r.URL.Query().Get("status")), Limit: queryLimit(r, 50)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps backtest campaigns")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"backtest_campaigns": marketOpsBacktestCampaignResponses(records)})
	})

	mux.HandleFunc("GET /v1/marketops/backtest-campaigns/{campaign_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		record, err := repo.GetMarketOpsBacktestCampaign(r.Context(), r.PathValue("campaign_id"))
		if err != nil {
			writeQueryError(w, err, "backtest_campaign_not_found", "MarketOps backtest campaign not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"backtest_campaign": marketOpsBacktestCampaignResponse(record)})
	})

	mux.HandleFunc("POST /v1/marketops/backtests", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		req, err := readMarketOpsBacktestCreateRequest(w, r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		backtestCfg, err := marketOpsBacktestConfigFromRequest(req, replayActor(r, req.RequestedBy))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_backtest", err.Error())
			return
		}
		runner := cfg.MarketOpsBacktestRunner
		if runner == nil {
			runner = marketopsbacktest.Run
		}
		result, err := runner(r.Context(), repo, backtestCfg)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "backtest_failed", err.Error())
			return
		}
		metrics, err := json.Marshal(result.Metrics)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "metrics_failed", "failed to encode backtest metrics")
			return
		}
		writeJSON(w, http.StatusCreated, marketOpsBacktestCreateResponse{BacktestRun: marketOpsBacktestRunResponse(result.Run), Metrics: json.RawMessage(metrics)})
	})

	mux.HandleFunc("GET /v1/marketops/backtests", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		records, err := repo.ListMarketOpsBacktestRuns(r.Context(), storage.MarketOpsBacktestRunFilter{
			TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), AppID: strings.TrimSpace(r.URL.Query().Get("app_id")), Domain: strings.TrimSpace(r.URL.Query().Get("domain")), UseCase: strings.TrimSpace(r.URL.Query().Get("use_case")),
			SourceID: strings.TrimSpace(r.URL.Query().Get("source_id")), Dataset: strings.TrimSpace(r.URL.Query().Get("dataset")), DetectorID: strings.TrimSpace(r.URL.Query().Get("detector_id")), Status: strings.TrimSpace(r.URL.Query().Get("status")), Limit: queryLimit(r, 50),
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps backtests")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"backtest_runs": marketOpsBacktestRunResponses(records)})
	})

	mux.HandleFunc("POST /v1/marketops/backtest-calibration-summaries", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		var req marketOpsBacktestCalibrationSummaryCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		filter := marketOpsBacktestRunFilterFromCalibrationRequest(req)
		if strings.TrimSpace(filter.TenantID) == "" {
			writeError(w, http.StatusBadRequest, "invalid_calibration_summary", "tenant_id is required")
			return
		}
		runs, err := repo.ListMarketOpsBacktestRuns(r.Context(), filter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps backtests for calibration summary")
			return
		}
		summaryID := strings.TrimSpace(req.SummaryID)
		if summaryID == "" {
			summaryID = newID("btcal_marketops")
		}
		record, err := buildMarketOpsBacktestCalibrationSummary(summaryID, replayActor(r, req.RequestedBy), filter, runs)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_calibration_summary", err.Error())
			return
		}
		if err := repo.UpsertMarketOpsBacktestCalibrationSummary(r.Context(), record); err != nil {
			writeError(w, http.StatusInternalServerError, "persist_failed", "failed to persist MarketOps backtest calibration summary")
			return
		}
		stored, err := repo.GetMarketOpsBacktestCalibrationSummary(r.Context(), summaryID)
		if err != nil {
			writeQueryError(w, err, "calibration_summary_not_found", "MarketOps backtest calibration summary not found")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"calibration_summary": marketOpsBacktestCalibrationSummaryResponse(stored)})
	})

	mux.HandleFunc("GET /v1/marketops/backtest-calibration-summaries", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		records, err := repo.ListMarketOpsBacktestCalibrationSummaries(r.Context(), storage.MarketOpsBacktestCalibrationSummaryFilter{
			TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), AppID: strings.TrimSpace(r.URL.Query().Get("app_id")), Domain: strings.TrimSpace(r.URL.Query().Get("domain")), UseCase: strings.TrimSpace(r.URL.Query().Get("use_case")),
			SourceID: strings.TrimSpace(r.URL.Query().Get("source_id")), Dataset: strings.TrimSpace(r.URL.Query().Get("dataset")), DetectorID: strings.TrimSpace(r.URL.Query().Get("detector_id")), Limit: queryLimit(r, 50),
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps backtest calibration summaries")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"calibration_summaries": marketOpsBacktestCalibrationSummaryResponses(records)})
	})

	mux.HandleFunc("GET /v1/marketops/backtest-calibration-summaries/{summary_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		record, err := repo.GetMarketOpsBacktestCalibrationSummary(r.Context(), r.PathValue("summary_id"))
		if err != nil {
			writeQueryError(w, err, "calibration_summary_not_found", "MarketOps backtest calibration summary not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"calibration_summary": marketOpsBacktestCalibrationSummaryResponse(record)})
	})

	mux.HandleFunc("POST /v1/marketops/backtest-calibration-baselines", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		var req marketOpsBacktestCalibrationBaselineCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		if strings.TrimSpace(req.TenantID) == "" || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.SummaryID) == "" {
			writeError(w, http.StatusBadRequest, "invalid_calibration_baseline", "tenant_id, name, and summary_id are required")
			return
		}
		summary, err := repo.GetMarketOpsBacktestCalibrationSummary(r.Context(), req.SummaryID)
		if err != nil {
			writeQueryError(w, err, "calibration_summary_not_found", "MarketOps backtest calibration summary not found")
			return
		}
		if summary.TenantID != strings.TrimSpace(req.TenantID) {
			writeError(w, http.StatusBadRequest, "invalid_calibration_baseline", "summary tenant_id does not match request tenant_id")
			return
		}
		baselineID := strings.TrimSpace(req.BaselineID)
		if baselineID == "" {
			baselineID = newID("btbase_marketops")
		}
		req.BaselineID = baselineID
		record := marketOpsBacktestCalibrationBaselineFromRequest(req, replayActor(r, req.CreatedBy), summary)
		if err := repo.UpsertMarketOpsBacktestCalibrationBaseline(r.Context(), record); err != nil {
			writeError(w, http.StatusInternalServerError, "persist_failed", "failed to persist MarketOps backtest calibration baseline")
			return
		}
		stored, err := repo.GetMarketOpsBacktestCalibrationBaseline(r.Context(), baselineID)
		if err != nil {
			writeQueryError(w, err, "calibration_baseline_not_found", "MarketOps backtest calibration baseline not found")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"calibration_baseline": marketOpsBacktestCalibrationBaselineResponse(stored)})
	})

	mux.HandleFunc("GET /v1/marketops/backtest-calibration-baselines", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		records, err := repo.ListMarketOpsBacktestCalibrationBaselines(r.Context(), storage.MarketOpsBacktestCalibrationBaselineFilter{
			TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), AppID: strings.TrimSpace(r.URL.Query().Get("app_id")), Domain: strings.TrimSpace(r.URL.Query().Get("domain")), UseCase: strings.TrimSpace(r.URL.Query().Get("use_case")),
			DetectorID: strings.TrimSpace(r.URL.Query().Get("detector_id")), Dataset: strings.TrimSpace(r.URL.Query().Get("dataset")), Status: strings.TrimSpace(r.URL.Query().Get("status")), Limit: queryLimit(r, 50),
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps backtest calibration baselines")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"calibration_baselines": marketOpsBacktestCalibrationBaselineResponses(records)})
	})

	mux.HandleFunc("GET /v1/marketops/backtest-calibration-baselines/{baseline_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		record, err := repo.GetMarketOpsBacktestCalibrationBaseline(r.Context(), r.PathValue("baseline_id"))
		if err != nil {
			writeQueryError(w, err, "calibration_baseline_not_found", "MarketOps backtest calibration baseline not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"calibration_baseline": marketOpsBacktestCalibrationBaselineResponse(record)})
	})

	mux.HandleFunc("POST /v1/marketops/backtest-calibration-comparisons", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		var req marketOpsBacktestCalibrationComparisonCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		if strings.TrimSpace(req.TenantID) == "" || strings.TrimSpace(req.BaselineID) == "" || strings.TrimSpace(req.CandidateSummaryID) == "" {
			writeError(w, http.StatusBadRequest, "invalid_calibration_comparison", "tenant_id, baseline_id, and candidate_summary_id are required")
			return
		}
		baseline, err := repo.GetMarketOpsBacktestCalibrationBaseline(r.Context(), req.BaselineID)
		if err != nil {
			writeQueryError(w, err, "calibration_baseline_not_found", "MarketOps backtest calibration baseline not found")
			return
		}
		if baseline.TenantID != strings.TrimSpace(req.TenantID) {
			writeError(w, http.StatusBadRequest, "invalid_calibration_comparison", "baseline tenant_id does not match request tenant_id")
			return
		}
		baselineSummary, err := repo.GetMarketOpsBacktestCalibrationSummary(r.Context(), baseline.SummaryID)
		if err != nil {
			writeQueryError(w, err, "calibration_summary_not_found", "MarketOps baseline summary not found")
			return
		}
		candidateSummary, err := repo.GetMarketOpsBacktestCalibrationSummary(r.Context(), req.CandidateSummaryID)
		if err != nil {
			writeQueryError(w, err, "calibration_summary_not_found", "MarketOps candidate summary not found")
			return
		}
		if candidateSummary.TenantID != baseline.TenantID || baselineSummary.TenantID != baseline.TenantID {
			writeError(w, http.StatusBadRequest, "invalid_calibration_comparison", "summary tenant_id values must match baseline tenant_id")
			return
		}
		comparisonID := strings.TrimSpace(req.ComparisonID)
		if comparisonID == "" {
			comparisonID = newID("btcmp_marketops")
		}
		record, err := buildMarketOpsBacktestCalibrationComparison(comparisonID, replayActor(r, req.CreatedBy), baseline, baselineSummary, candidateSummary)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_calibration_comparison", err.Error())
			return
		}
		if err := repo.UpsertMarketOpsBacktestCalibrationComparison(r.Context(), record); err != nil {
			writeError(w, http.StatusInternalServerError, "persist_failed", "failed to persist MarketOps backtest calibration comparison")
			return
		}
		stored, err := repo.GetMarketOpsBacktestCalibrationComparison(r.Context(), comparisonID)
		if err != nil {
			writeQueryError(w, err, "calibration_comparison_not_found", "MarketOps backtest calibration comparison not found")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"calibration_comparison": marketOpsBacktestCalibrationComparisonResponse(stored)})
	})

	mux.HandleFunc("GET /v1/marketops/backtest-calibration-comparisons", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		records, err := repo.ListMarketOpsBacktestCalibrationComparisons(r.Context(), storage.MarketOpsBacktestCalibrationComparisonFilter{
			TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), BaselineID: strings.TrimSpace(r.URL.Query().Get("baseline_id")), DetectorID: strings.TrimSpace(r.URL.Query().Get("detector_id")), Dataset: strings.TrimSpace(r.URL.Query().Get("dataset")), Recommendation: strings.TrimSpace(r.URL.Query().Get("recommendation")), Limit: queryLimit(r, 50),
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps backtest calibration comparisons")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"calibration_comparisons": marketOpsBacktestCalibrationComparisonResponses(records)})
	})

	mux.HandleFunc("GET /v1/marketops/backtest-calibration-comparisons/{comparison_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		record, err := repo.GetMarketOpsBacktestCalibrationComparison(r.Context(), r.PathValue("comparison_id"))
		if err != nil {
			writeQueryError(w, err, "calibration_comparison_not_found", "MarketOps backtest calibration comparison not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"calibration_comparison": marketOpsBacktestCalibrationComparisonResponse(record)})
	})

	mux.HandleFunc("POST /v1/marketops/backtest-promotion-candidates", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		var req marketOpsBacktestPromotionCandidateCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		if strings.TrimSpace(req.TenantID) == "" || strings.TrimSpace(req.BaselineID) == "" || strings.TrimSpace(req.ComparisonID) == "" {
			writeError(w, http.StatusBadRequest, "invalid_promotion_candidate", "tenant_id, baseline_id, and comparison_id are required")
			return
		}
		baseline, err := repo.GetMarketOpsBacktestCalibrationBaseline(r.Context(), req.BaselineID)
		if err != nil {
			writeQueryError(w, err, "calibration_baseline_not_found", "MarketOps backtest calibration baseline not found")
			return
		}
		if baseline.TenantID != strings.TrimSpace(req.TenantID) {
			writeError(w, http.StatusBadRequest, "invalid_promotion_candidate", "baseline tenant_id does not match request tenant_id")
			return
		}
		comparison, err := repo.GetMarketOpsBacktestCalibrationComparison(r.Context(), req.ComparisonID)
		if err != nil {
			writeQueryError(w, err, "calibration_comparison_not_found", "MarketOps backtest calibration comparison not found")
			return
		}
		if comparison.TenantID != baseline.TenantID || comparison.BaselineID != baseline.BaselineID {
			writeError(w, http.StatusBadRequest, "invalid_promotion_candidate", "comparison must belong to the requested baseline and tenant")
			return
		}
		var evaluation *storage.MarketOpsBacktestEvaluationRecord
		if strings.TrimSpace(req.EvaluationID) != "" {
			storedEvaluation, err := repo.GetMarketOpsBacktestEvaluation(r.Context(), req.EvaluationID)
			if err != nil {
				writeQueryError(w, err, "backtest_evaluation_not_found", "MarketOps backtest evaluation not found")
				return
			}
			if storedEvaluation.TenantID != baseline.TenantID {
				writeError(w, http.StatusBadRequest, "invalid_promotion_candidate", "evaluation tenant_id does not match baseline tenant_id")
				return
			}
			evaluation = &storedEvaluation
		}
		var run *storage.MarketOpsBacktestRunRecord
		if evaluation != nil && strings.TrimSpace(evaluation.RunID) != "" {
			storedRun, err := repo.GetMarketOpsBacktestRun(r.Context(), evaluation.RunID)
			if err != nil {
				writeQueryError(w, err, "backtest_run_not_found", "MarketOps backtest run not found")
				return
			}
			if storedRun.TenantID != baseline.TenantID {
				writeError(w, http.StatusBadRequest, "invalid_promotion_candidate", "run tenant_id does not match baseline tenant_id")
				return
			}
			run = &storedRun
		}
		policyVersion := ""
		if run != nil {
			policy, err := repo.ListMarketOpsBacktestPolicyResults(r.Context(), storage.MarketOpsBacktestGraphProposalFilter{RunID: run.RunID, TenantID: run.TenantID, Limit: 1})
			if err != nil {
				writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps backtest policy results")
				return
			}
			if len(policy) > 0 {
				policyVersion = policy[0].PolicyVersion
			}
		}
		candidateID := strings.TrimSpace(req.CandidateID)
		if candidateID == "" {
			candidateID = newID("btpromo_marketops")
		}
		record, err := buildMarketOpsBacktestPromotionCandidate(candidateID, replayActor(r, req.RequestedBy), req.CandidateVersion, baseline, comparison, evaluation, run, policyVersion)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "promotion_candidate_failed", "failed to build MarketOps promotion candidate")
			return
		}
		if err := repo.UpsertMarketOpsBacktestPromotionCandidate(r.Context(), record); err != nil {
			writeError(w, http.StatusInternalServerError, "persist_failed", "failed to persist MarketOps backtest promotion candidate")
			return
		}
		stored, err := repo.GetMarketOpsBacktestPromotionCandidate(r.Context(), candidateID)
		if err != nil {
			writeQueryError(w, err, "promotion_candidate_not_found", "MarketOps backtest promotion candidate not found")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"promotion_candidate": marketOpsBacktestPromotionCandidateResponse(stored)})
	})

	mux.HandleFunc("GET /v1/marketops/backtest-promotion-candidates", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		records, err := repo.ListMarketOpsBacktestPromotionCandidates(r.Context(), storage.MarketOpsBacktestPromotionCandidateFilter{TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), AppID: strings.TrimSpace(r.URL.Query().Get("app_id")), Domain: strings.TrimSpace(r.URL.Query().Get("domain")), UseCase: strings.TrimSpace(r.URL.Query().Get("use_case")), BaselineID: strings.TrimSpace(r.URL.Query().Get("baseline_id")), ComparisonID: strings.TrimSpace(r.URL.Query().Get("comparison_id")), EvaluationID: strings.TrimSpace(r.URL.Query().Get("evaluation_id")), RunID: strings.TrimSpace(r.URL.Query().Get("run_id")), DetectorID: strings.TrimSpace(r.URL.Query().Get("detector_id")), Dataset: strings.TrimSpace(r.URL.Query().Get("dataset")), ReadinessStatus: strings.TrimSpace(r.URL.Query().Get("readiness_status")), Status: strings.TrimSpace(r.URL.Query().Get("status")), Limit: queryLimit(r, 50)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps backtest promotion candidates")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"promotion_candidates": marketOpsBacktestPromotionCandidateResponses(records)})
	})

	mux.HandleFunc("GET /v1/marketops/backtest-promotion-candidates/{candidate_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		record, err := repo.GetMarketOpsBacktestPromotionCandidate(r.Context(), r.PathValue("candidate_id"))
		if err != nil {
			writeQueryError(w, err, "promotion_candidate_not_found", "MarketOps backtest promotion candidate not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"promotion_candidate": marketOpsBacktestPromotionCandidateResponse(record)})
	})

	mux.HandleFunc("POST /v1/marketops/backtest-promotion-candidates/{candidate_id}/decision", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		var req marketOpsBacktestPromotionCandidateDecisionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		if !marketOpsBacktestPromotionCandidateDecisionStatusAllowed(req.Status) {
			writeError(w, http.StatusBadRequest, "invalid_status", "promotion candidate decision status is invalid")
			return
		}
		record, err := repo.MutateMarketOpsBacktestPromotionCandidateDecision(r.Context(), storage.MarketOpsBacktestPromotionCandidateDecisionMutation{CandidateID: r.PathValue("candidate_id"), Status: strings.TrimSpace(req.Status), ReviewedBy: replayActor(r, req.ReviewedBy), ReviewedAt: time.Now().UTC(), DecisionNote: strings.TrimSpace(req.DecisionNote)})
		if err != nil {
			if strings.Contains(err.Error(), "status") {
				writeError(w, http.StatusBadRequest, "invalid_status", "promotion candidate decision status is invalid")
				return
			}
			writeQueryError(w, err, "promotion_candidate_not_found", "MarketOps backtest promotion candidate not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"promotion_candidate": marketOpsBacktestPromotionCandidateResponse(record)})
	})

	mux.HandleFunc("POST /v1/marketops/backtest-calibration-readiness", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		var req marketOpsBacktestCalibrationReadinessCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		if strings.TrimSpace(req.TenantID) == "" || strings.TrimSpace(req.BaselineID) == "" || strings.TrimSpace(req.ComparisonID) == "" {
			writeError(w, http.StatusBadRequest, "invalid_calibration_readiness", "tenant_id, baseline_id, and comparison_id are required")
			return
		}
		baseline, err := repo.GetMarketOpsBacktestCalibrationBaseline(r.Context(), req.BaselineID)
		if err != nil {
			writeQueryError(w, err, "calibration_baseline_not_found", "MarketOps backtest calibration baseline not found")
			return
		}
		if baseline.TenantID != strings.TrimSpace(req.TenantID) {
			writeError(w, http.StatusBadRequest, "invalid_calibration_readiness", "baseline tenant_id does not match request tenant_id")
			return
		}
		comparison, err := repo.GetMarketOpsBacktestCalibrationComparison(r.Context(), req.ComparisonID)
		if err != nil {
			writeQueryError(w, err, "calibration_comparison_not_found", "MarketOps backtest calibration comparison not found")
			return
		}
		if comparison.TenantID != baseline.TenantID || comparison.BaselineID != baseline.BaselineID {
			writeError(w, http.StatusBadRequest, "invalid_calibration_readiness", "comparison must belong to the requested baseline and tenant")
			return
		}
		var evaluation *storage.MarketOpsBacktestEvaluationRecord
		if strings.TrimSpace(req.EvaluationID) != "" {
			storedEvaluation, err := repo.GetMarketOpsBacktestEvaluation(r.Context(), req.EvaluationID)
			if err != nil {
				writeQueryError(w, err, "backtest_evaluation_not_found", "MarketOps backtest evaluation not found")
				return
			}
			if storedEvaluation.TenantID != baseline.TenantID {
				writeError(w, http.StatusBadRequest, "invalid_calibration_readiness", "evaluation tenant_id does not match baseline tenant_id")
				return
			}
			evaluation = &storedEvaluation
		}
		var candidate *storage.MarketOpsBacktestPromotionCandidateRecord
		if strings.TrimSpace(req.CandidateID) != "" {
			storedCandidate, err := repo.GetMarketOpsBacktestPromotionCandidate(r.Context(), req.CandidateID)
			if err != nil {
				writeQueryError(w, err, "promotion_candidate_not_found", "MarketOps backtest promotion candidate not found")
				return
			}
			if storedCandidate.TenantID != baseline.TenantID {
				writeError(w, http.StatusBadRequest, "invalid_calibration_readiness", "promotion candidate tenant_id does not match baseline tenant_id")
				return
			}
			candidate = &storedCandidate
		}
		runs, err := repo.ListMarketOpsBacktestRuns(r.Context(), storage.MarketOpsBacktestRunFilter{TenantID: strings.TrimSpace(req.TenantID), AppID: baseline.AppID, Domain: baseline.Domain, UseCase: baseline.UseCase, DetectorID: firstNonEmptyBacktestValue(comparison.DetectorID, baseline.DetectorID), Status: storage.RunStatusSucceeded, Limit: 200})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps backtest runs for readiness")
			return
		}
		labels, err := repo.ListMarketOpsBacktestEvaluationLabels(r.Context(), storage.MarketOpsBacktestEvaluationLabelFilter{TenantID: strings.TrimSpace(req.TenantID), AppID: baseline.AppID, Domain: baseline.Domain, UseCase: baseline.UseCase, Limit: 200})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps evaluation labels for readiness")
			return
		}
		assets := []storage.MarketOpsAssetRecord{}
		if strings.TrimSpace(req.UniverseGroup) != "" {
			assets, err = repo.ListMarketOpsAssets(r.Context(), strings.TrimSpace(req.TenantID), strings.TrimSpace(req.UniverseGroup), true, 200)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps universe assets for readiness")
				return
			}
		}
		readinessID := strings.TrimSpace(req.ReadinessID)
		if readinessID == "" {
			readinessID = newID("btready_marketops")
		}
		record, err := buildMarketOpsBacktestCalibrationReadiness(readinessID, replayActor(r, req.RequestedBy), req, baseline, comparison, evaluation, candidate, runs, labels, assets)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_calibration_readiness", err.Error())
			return
		}
		if err := repo.UpsertMarketOpsBacktestCalibrationReadiness(r.Context(), record); err != nil {
			writeError(w, http.StatusInternalServerError, "persist_failed", "failed to persist MarketOps calibration readiness")
			return
		}
		stored, err := repo.GetMarketOpsBacktestCalibrationReadiness(r.Context(), readinessID)
		if err != nil {
			writeQueryError(w, err, "calibration_readiness_not_found", "MarketOps calibration readiness not found")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"calibration_readiness": marketOpsBacktestCalibrationReadinessResponse(stored)})
	})

	mux.HandleFunc("GET /v1/marketops/backtest-calibration-readiness", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		records, err := repo.ListMarketOpsBacktestCalibrationReadiness(r.Context(), storage.MarketOpsBacktestCalibrationReadinessFilter{TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), AppID: strings.TrimSpace(r.URL.Query().Get("app_id")), Domain: strings.TrimSpace(r.URL.Query().Get("domain")), UseCase: strings.TrimSpace(r.URL.Query().Get("use_case")), BaselineID: strings.TrimSpace(r.URL.Query().Get("baseline_id")), ComparisonID: strings.TrimSpace(r.URL.Query().Get("comparison_id")), EvaluationID: strings.TrimSpace(r.URL.Query().Get("evaluation_id")), CandidateID: strings.TrimSpace(r.URL.Query().Get("candidate_id")), DetectorID: strings.TrimSpace(r.URL.Query().Get("detector_id")), ReadinessStatus: strings.TrimSpace(r.URL.Query().Get("readiness_status")), Limit: queryLimit(r, 50)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps calibration readiness snapshots")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"calibration_readiness": marketOpsBacktestCalibrationReadinessResponses(records)})
	})

	mux.HandleFunc("GET /v1/marketops/backtest-calibration-readiness/{readiness_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		record, err := repo.GetMarketOpsBacktestCalibrationReadiness(r.Context(), r.PathValue("readiness_id"))
		if err != nil {
			writeQueryError(w, err, "calibration_readiness_not_found", "MarketOps calibration readiness not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"calibration_readiness": marketOpsBacktestCalibrationReadinessResponse(record)})
	})

	mux.HandleFunc("POST /v1/syncratic/context-windows/{context_window_id}/ask", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		if cfg.SyncraticAskClient == nil {
			writeError(w, http.StatusServiceUnavailable, "syncratic_ask_unavailable", "Syncratic Ask client is not configured")
			return
		}
		var req syncraticAskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		insight, result, err := enrichSyncraticInsightWithAsk(r.Context(), repo, cfg.SyncraticAskClient, r.PathValue("context_window_id"), req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				writeQueryError(w, err, "context_window_not_found", "Syncratic context window not found")
				return
			}
			if strings.Contains(err.Error(), "syncratic ask failed") {
				writeError(w, http.StatusBadGateway, "syncratic_ask_failed", "Syncratic Ask request failed")
				return
			}
			if strings.Contains(err.Error(), "tenant_id") || strings.Contains(err.Error(), "prompt") || strings.Contains(err.Error(), "max_prompt_bytes") {
				writeError(w, http.StatusBadRequest, "syncratic_ask_invalid", err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "syncratic_ask_failed", "failed to enrich Syncratic insight")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"syncratic_insight": syncraticInsightResponse(insight), "ask_result": result})
	})

	mux.HandleFunc("POST /v1/syncratic/context-windows", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		var req syncraticContextWindowCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		windowStart, err := parseRFC3339(req.WindowStart)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_context_window", "window_start must be RFC3339")
			return
		}
		windowEnd, err := parseRFC3339(req.WindowEnd)
		if err != nil || !windowEnd.After(windowStart) {
			writeError(w, http.StatusBadRequest, "invalid_context_window", "window_end must be RFC3339 and after window_start")
			return
		}
		record, err := buildSyncraticContextWindow(r.Context(), repo, req.TenantID, req.SubjectSymbol, req.ContextStrategy, windowStart, windowEnd, req.ContextBuilderVersion, req.SignalTypes, 1000, 1000)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_context_window", err.Error())
			return
		}
		if len(record.SignalIDs)+len(record.AlertIDs)+len(record.ArtifactIDs)+len(record.GraphProposalIDs)+len(record.LabelIDs) == 0 {
			writeError(w, http.StatusBadRequest, "empty_context_window", "context window requires at least one persisted evidence record")
			return
		}
		if err := repo.UpsertSyncraticContextWindow(r.Context(), record); err != nil {
			writeError(w, http.StatusInternalServerError, "persist_failed", "failed to persist Syncratic context window")
			return
		}
		stored, err := repo.GetSyncraticContextWindow(r.Context(), record.ContextWindowID)
		if err != nil {
			writeQueryError(w, err, "context_window_not_found", "Syncratic context window not found")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"context_window": syncraticContextWindowResponse(stored)})
	})

	mux.HandleFunc("GET /v1/syncratic/context-windows", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		records, err := repo.ListSyncraticContextWindows(r.Context(), storage.SyncraticContextWindowFilter{TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), AppID: strings.TrimSpace(r.URL.Query().Get("app_id")), Domain: strings.TrimSpace(r.URL.Query().Get("domain")), UseCase: strings.TrimSpace(r.URL.Query().Get("use_case")), SubjectSymbol: strings.TrimSpace(r.URL.Query().Get("subject_symbol")), ContextStrategy: strings.TrimSpace(r.URL.Query().Get("context_strategy")), Status: strings.TrimSpace(r.URL.Query().Get("status")), Limit: queryLimit(r, 50)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list Syncratic context windows")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"context_windows": syncraticContextWindowResponses(records)})
	})

	mux.HandleFunc("GET /v1/syncratic/context-windows/{context_window_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		record, err := repo.GetSyncraticContextWindow(r.Context(), r.PathValue("context_window_id"))
		if err != nil {
			writeQueryError(w, err, "context_window_not_found", "Syncratic context window not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"context_window": syncraticContextWindowResponse(record)})
	})

	mux.HandleFunc("POST /v1/syncratic/insights", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		var req syncraticInsightCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		contextWindow, err := repo.GetSyncraticContextWindow(r.Context(), req.ContextWindowID)
		if err != nil {
			writeQueryError(w, err, "context_window_not_found", "Syncratic context window not found")
			return
		}
		if strings.TrimSpace(req.TenantID) != "" && strings.TrimSpace(req.TenantID) != contextWindow.TenantID {
			writeError(w, http.StatusBadRequest, "invalid_insight", "tenant_id does not match context window")
			return
		}
		record := buildSyncraticInsight(contextWindow, req.InsightType, req.BuilderVersion)
		if err := repo.UpsertSyncraticInsight(r.Context(), record); err != nil {
			writeError(w, http.StatusInternalServerError, "persist_failed", "failed to persist Syncratic insight")
			return
		}
		stored, err := repo.GetSyncraticInsight(r.Context(), record.SyncraticInsightID)
		if err != nil {
			writeQueryError(w, err, "syncratic_insight_not_found", "Syncratic insight not found")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"syncratic_insight": syncraticInsightResponse(stored)})
	})

	mux.HandleFunc("GET /v1/syncratic/insights", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		filter := storage.SyncraticInsightFilter{TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), AppID: strings.TrimSpace(r.URL.Query().Get("app_id")), Domain: strings.TrimSpace(r.URL.Query().Get("domain")), UseCase: strings.TrimSpace(r.URL.Query().Get("use_case")), ContextWindowID: strings.TrimSpace(r.URL.Query().Get("context_window_id")), InsightType: strings.TrimSpace(r.URL.Query().Get("insight_type")), SubjectSymbol: strings.TrimSpace(r.URL.Query().Get("subject_symbol")), Status: strings.TrimSpace(r.URL.Query().Get("status")), Limit: queryLimit(r, 50)}
		records, err := repo.ListSyncraticInsights(r.Context(), filter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list Syncratic insights")
			return
		}
		contextWindows, err := repo.ListSyncraticContextWindows(r.Context(), storage.SyncraticContextWindowFilter{TenantID: filter.TenantID, AppID: filter.AppID, Domain: filter.Domain, UseCase: filter.UseCase, SubjectSymbol: filter.SubjectSymbol, Limit: 5000})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to derive Syncratic insight currentness")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"syncratic_insights": syncraticInsightResponsesWithContexts(records, syncraticContextWindowMap(contextWindows))})
	})

	mux.HandleFunc("GET /v1/syncratic/insights/{syncratic_insight_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		record, err := repo.GetSyncraticInsight(r.Context(), r.PathValue("syncratic_insight_id"))
		if err != nil {
			writeQueryError(w, err, "syncratic_insight_not_found", "Syncratic insight not found")
			return
		}
		relatedInsights, err := repo.ListSyncraticInsights(r.Context(), storage.SyncraticInsightFilter{TenantID: record.TenantID, AppID: record.AppID, Domain: record.Domain, UseCase: record.UseCase, SubjectSymbol: record.SubjectSymbol, Limit: 5000})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to derive Syncratic insight currentness")
			return
		}
		contextWindows, err := repo.ListSyncraticContextWindows(r.Context(), storage.SyncraticContextWindowFilter{TenantID: record.TenantID, AppID: record.AppID, Domain: record.Domain, UseCase: record.UseCase, SubjectSymbol: record.SubjectSymbol, Limit: 5000})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to derive Syncratic insight currentness")
			return
		}
		currentness := syncraticInsightCurrentness(relatedInsights, syncraticContextWindowMap(contextWindows))
		writeJSON(w, http.StatusOK, map[string]any{"syncratic_insight": syncraticInsightResponseWithCurrentness(record, currentness[record.SyncraticInsightID])})
	})

	mux.HandleFunc("POST /v1/syncratic/materialize", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		var req syncraticMaterializeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		result, err := materializeSyncraticContexts(r.Context(), repo, req)
		if err != nil {
			writeError(w, http.StatusBadRequest, "materialize_failed", err.Error())
			return
		}
		status := http.StatusCreated
		if req.DryRun {
			status = http.StatusOK
		}
		writeJSON(w, status, map[string]any{"materialization": result})
	})

	mux.HandleFunc("GET /v1/marketops/backtests/{run_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		record, err := repo.GetMarketOpsBacktestRun(r.Context(), r.PathValue("run_id"))
		if err != nil {
			writeQueryError(w, err, "backtest_run_not_found", "MarketOps backtest run not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"backtest_run": marketOpsBacktestRunResponse(record)})
	})

	mux.HandleFunc("GET /v1/marketops/backtests/{run_id}/signals", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		records, err := repo.ListMarketOpsBacktestSignals(r.Context(), storage.MarketOpsBacktestSignalFilter{RunID: r.PathValue("run_id"), TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), SignalType: strings.TrimSpace(r.URL.Query().Get("signal_type")), Limit: queryLimit(r, 50)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps backtest signals")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"backtest_signals": marketOpsBacktestSignalResponses(records)})
	})

	mux.HandleFunc("GET /v1/marketops/backtests/{run_id}/graph-proposals", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		filter := storage.MarketOpsBacktestGraphProposalFilter{RunID: r.PathValue("run_id"), TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), SignalType: strings.TrimSpace(r.URL.Query().Get("signal_type")), SubjectSymbol: strings.TrimSpace(r.URL.Query().Get("subject_symbol")), CandidateType: strings.TrimSpace(r.URL.Query().Get("candidate_type")), Recommendation: strings.TrimSpace(r.URL.Query().Get("recommendation")), Limit: queryLimit(r, 50)}
		records, err := repo.ListMarketOpsBacktestGraphProposals(r.Context(), filter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps backtest graph proposals")
			return
		}
		policy, err := repo.ListMarketOpsBacktestPolicyResults(r.Context(), filter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps backtest policy results")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"backtest_graph_proposals": marketOpsBacktestGraphProposalResponses(records), "policy_results": marketOpsBacktestPolicyResultResponses(policy)})
	})

	mux.HandleFunc("GET /v1/marketops/dsm/artifacts", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		records, err := repo.ListMarketOpsDSMArtifacts(r.Context(), storage.MarketOpsDSMArtifactFilter{
			TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), AppID: strings.TrimSpace(r.URL.Query().Get("app_id")), Domain: strings.TrimSpace(r.URL.Query().Get("domain")), UseCase: strings.TrimSpace(r.URL.Query().Get("use_case")),
			SignalType: strings.TrimSpace(r.URL.Query().Get("signal_type")), Severity: strings.TrimSpace(r.URL.Query().Get("severity")), SubjectSymbol: strings.TrimSpace(r.URL.Query().Get("subject_symbol")), Limit: queryLimit(r, 50),
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps DSM artifacts")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"artifacts": marketOpsDSMArtifactResponses(records)})
	})

	mux.HandleFunc("GET /v1/marketops/dsm/artifacts/{artifact_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		record, err := repo.GetMarketOpsDSMArtifact(r.Context(), r.PathValue("artifact_id"))
		if err != nil {
			writeQueryError(w, err, "artifact_not_found", "MarketOps DSM artifact not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"artifact": marketOpsDSMArtifactResponse(record)})
	})

	mux.HandleFunc("GET /v1/marketops/dsm/graph-proposals", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		records, err := repo.ListMarketOpsDSMGraphProposals(r.Context(), storage.MarketOpsDSMGraphProposalFilter{
			TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), AppID: strings.TrimSpace(r.URL.Query().Get("app_id")), Domain: strings.TrimSpace(r.URL.Query().Get("domain")), UseCase: strings.TrimSpace(r.URL.Query().Get("use_case")),
			ArtifactID: strings.TrimSpace(r.URL.Query().Get("artifact_id")), SignalID: strings.TrimSpace(r.URL.Query().Get("signal_id")), SignalType: strings.TrimSpace(r.URL.Query().Get("signal_type")), SubjectSymbol: strings.TrimSpace(r.URL.Query().Get("subject_symbol")),
			CandidateType: strings.TrimSpace(r.URL.Query().Get("candidate_type")), Status: strings.TrimSpace(r.URL.Query().Get("status")), Limit: queryLimit(r, 50),
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps DSM graph proposals")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"graph_proposals": marketOpsDSMGraphProposalResponses(records)})
	})

	mux.HandleFunc("GET /v1/marketops/dsm/graph-proposals/{proposal_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		record, err := repo.GetMarketOpsDSMGraphProposal(r.Context(), r.PathValue("proposal_id"))
		if err != nil {
			writeQueryError(w, err, "graph_proposal_not_found", "MarketOps DSM graph proposal not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"graph_proposal": marketOpsDSMGraphProposalResponse(record)})
	})

	mux.HandleFunc("POST /v1/marketops/dsm/graph-proposals/{proposal_id}/decision", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		var req graphProposalDecisionRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, defaultMaxRawEventBytes))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		status := strings.TrimSpace(req.Status)
		if status == "" {
			writeError(w, http.StatusBadRequest, "invalid_status", "status is required")
			return
		}
		record, err := repo.MutateMarketOpsDSMGraphProposal(r.Context(), storage.MarketOpsDSMGraphProposalMutation{
			ProposalID: r.PathValue("proposal_id"), Status: status, ReviewedBy: lifecycleActor(r, req.Actor), DecisionNote: strings.TrimSpace(req.Note), DecidedAt: time.Now().UTC(),
		})
		if err != nil {
			if strings.Contains(err.Error(), "status") {
				writeError(w, http.StatusBadRequest, "invalid_status", "graph proposal status is invalid")
				return
			}
			writeQueryError(w, err, "graph_proposal_not_found", "MarketOps DSM graph proposal not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"graph_proposal": marketOpsDSMGraphProposalResponse(record)})
	})

	mux.HandleFunc("POST /v1/marketops/backtest-evaluations", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		var req marketOpsBacktestEvaluationCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		if strings.TrimSpace(req.TenantID) == "" || strings.TrimSpace(req.RunID) == "" {
			writeError(w, http.StatusBadRequest, "invalid_backtest_evaluation", "tenant_id and run_id are required")
			return
		}
		run, err := repo.GetMarketOpsBacktestRun(r.Context(), req.RunID)
		if err != nil {
			writeQueryError(w, err, "backtest_run_not_found", "MarketOps backtest run not found")
			return
		}
		if run.TenantID != strings.TrimSpace(req.TenantID) {
			writeError(w, http.StatusBadRequest, "invalid_backtest_evaluation", "run tenant_id does not match request tenant_id")
			return
		}
		proposalFilter := storage.MarketOpsBacktestGraphProposalFilter{RunID: run.RunID, TenantID: run.TenantID, Limit: 1000}
		proposals, err := repo.ListMarketOpsBacktestGraphProposals(r.Context(), proposalFilter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps backtest graph proposals")
			return
		}
		policy, err := repo.ListMarketOpsBacktestPolicyResults(r.Context(), proposalFilter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps backtest policy results")
			return
		}
		labels, err := repo.ListMarketOpsBacktestEvaluationLabels(r.Context(), storage.MarketOpsBacktestEvaluationLabelFilter{TenantID: run.TenantID, LabelSource: firstNonEmptyBacktestValue(req.LabelSource, marketOpsEvaluationLabelSource), Limit: 1000})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps evaluation labels")
			return
		}
		evaluationID := strings.TrimSpace(req.EvaluationID)
		if evaluationID == "" {
			evaluationID = newID("bteval_marketops")
		}
		record, err := buildMarketOpsBacktestEvaluation(evaluationID, replayActor(r, req.RequestedBy), run, proposals, policy, labels)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "evaluation_failed", "failed to score MarketOps backtest evaluation")
			return
		}
		if err := repo.UpsertMarketOpsBacktestEvaluation(r.Context(), record); err != nil {
			writeError(w, http.StatusInternalServerError, "persist_failed", "failed to persist MarketOps backtest evaluation")
			return
		}
		stored, err := repo.GetMarketOpsBacktestEvaluation(r.Context(), evaluationID)
		if err != nil {
			writeQueryError(w, err, "backtest_evaluation_not_found", "MarketOps backtest evaluation not found")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"backtest_evaluation": marketOpsBacktestEvaluationResponse(stored)})
	})

	mux.HandleFunc("GET /v1/marketops/backtest-evaluations", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		records, err := repo.ListMarketOpsBacktestEvaluations(r.Context(), storage.MarketOpsBacktestEvaluationFilter{TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), AppID: strings.TrimSpace(r.URL.Query().Get("app_id")), Domain: strings.TrimSpace(r.URL.Query().Get("domain")), UseCase: strings.TrimSpace(r.URL.Query().Get("use_case")), RunID: strings.TrimSpace(r.URL.Query().Get("run_id")), DetectorID: strings.TrimSpace(r.URL.Query().Get("detector_id")), Dataset: strings.TrimSpace(r.URL.Query().Get("dataset")), Recommendation: strings.TrimSpace(r.URL.Query().Get("recommendation")), Limit: queryLimit(r, 50)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps backtest evaluations")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"backtest_evaluations": marketOpsBacktestEvaluationResponses(records)})
	})

	mux.HandleFunc("GET /v1/marketops/backtest-evaluations/{evaluation_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		record, err := repo.GetMarketOpsBacktestEvaluation(r.Context(), r.PathValue("evaluation_id"))
		if err != nil {
			writeQueryError(w, err, "backtest_evaluation_not_found", "MarketOps backtest evaluation not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"backtest_evaluation": marketOpsBacktestEvaluationResponse(record)})
	})

	mux.HandleFunc("POST /v1/marketops/backtest-evaluation-labels/sync", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		var req marketOpsBacktestEvaluationLabelSyncRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		if strings.TrimSpace(req.TenantID) == "" {
			writeError(w, http.StatusBadRequest, "invalid_label_sync", "tenant_id is required")
			return
		}
		limit := req.Limit
		if limit <= 0 {
			limit = 50
		}
		actor := replayActor(r, req.RequestedBy)
		synced := []storage.MarketOpsBacktestEvaluationLabelRecord{}
		for _, status := range marketOpsBacktestEvaluationLabelSyncStatuses(req) {
			proposals, err := repo.ListMarketOpsDSMGraphProposals(r.Context(), storage.MarketOpsDSMGraphProposalFilter{TenantID: strings.TrimSpace(req.TenantID), AppID: strings.TrimSpace(req.AppID), Domain: strings.TrimSpace(req.Domain), UseCase: strings.TrimSpace(req.UseCase), Status: status, Limit: limit})
			if err != nil {
				writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps graph proposals for label sync")
				return
			}
			labels, err := marketOpsBacktestEvaluationLabelsFromProposals(proposals, actor)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "label_sync_failed", "failed to build MarketOps evaluation labels")
				return
			}
			for _, label := range labels {
				if err := repo.UpsertMarketOpsBacktestEvaluationLabel(r.Context(), label); err != nil {
					writeError(w, http.StatusInternalServerError, "persist_failed", "failed to persist MarketOps evaluation label")
					return
				}
				stored, err := repo.GetMarketOpsBacktestEvaluationLabel(r.Context(), label.LabelID)
				if err != nil {
					writeQueryError(w, err, "evaluation_label_not_found", "MarketOps evaluation label not found")
					return
				}
				synced = append(synced, stored)
			}
		}
		writeJSON(w, http.StatusCreated, marketOpsBacktestEvaluationLabelSyncResponse{Synced: len(synced), Labels: marketOpsBacktestEvaluationLabelResponses(synced)})
	})

	mux.HandleFunc("GET /v1/marketops/backtest-evaluation-labels", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		records, err := repo.ListMarketOpsBacktestEvaluationLabels(r.Context(), storage.MarketOpsBacktestEvaluationLabelFilter{TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), AppID: strings.TrimSpace(r.URL.Query().Get("app_id")), Domain: strings.TrimSpace(r.URL.Query().Get("domain")), UseCase: strings.TrimSpace(r.URL.Query().Get("use_case")), SourceProposalID: strings.TrimSpace(r.URL.Query().Get("source_proposal_id")), ArtifactID: strings.TrimSpace(r.URL.Query().Get("artifact_id")), SignalID: strings.TrimSpace(r.URL.Query().Get("signal_id")), SubjectSymbol: strings.TrimSpace(r.URL.Query().Get("subject_symbol")), CandidateType: strings.TrimSpace(r.URL.Query().Get("candidate_type")), DecisionStatus: strings.TrimSpace(r.URL.Query().Get("decision_status")), Label: strings.TrimSpace(r.URL.Query().Get("label")), LabelSource: strings.TrimSpace(r.URL.Query().Get("label_source")), Limit: queryLimit(r, 50)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps evaluation labels")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"evaluation_labels": marketOpsBacktestEvaluationLabelResponses(records)})
	})

	mux.HandleFunc("GET /v1/marketops/backtest-evaluation-labels/{label_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		record, err := repo.GetMarketOpsBacktestEvaluationLabel(r.Context(), r.PathValue("label_id"))
		if err != nil {
			writeQueryError(w, err, "evaluation_label_not_found", "MarketOps evaluation label not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"evaluation_label": marketOpsBacktestEvaluationLabelResponse(record)})
	})

	mux.HandleFunc("GET /v1/alerts", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		records, err := repo.ListAlertLedger(r.Context(), storage.AlertLedgerFilter{
			TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), AppID: strings.TrimSpace(r.URL.Query().Get("app_id")), Domain: strings.TrimSpace(r.URL.Query().Get("domain")), UseCase: strings.TrimSpace(r.URL.Query().Get("use_case")), SourceID: strings.TrimSpace(r.URL.Query().Get("source_id")),
			Dataset: strings.TrimSpace(r.URL.Query().Get("dataset")), Severity: strings.TrimSpace(r.URL.Query().Get("severity")),
			Status: strings.TrimSpace(r.URL.Query().Get("status")), Limit: queryLimit(r, 50),
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list alerts")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"alerts": alertResponses(records)})
	})

	mux.HandleFunc("GET /v1/alerts/{alert_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		record, err := repo.GetAlertLedger(r.Context(), r.PathValue("alert_id"))
		if err != nil {
			writeQueryError(w, err, "alert_not_found", "alert not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"alert": alertResponse(record)})
	})

	mux.HandleFunc("POST /v1/alerts/{alert_id}/acknowledge", alertLifecycleHandler(cfg, storage.AlertStatusAcknowledged, "acknowledge"))
	mux.HandleFunc("POST /v1/alerts/{alert_id}/resolve", alertLifecycleHandler(cfg, storage.AlertStatusResolved, "resolve"))
	mux.HandleFunc("POST /v1/alerts/{alert_id}/suppress", alertLifecycleHandler(cfg, storage.AlertStatusSuppressed, "suppress"))

	mux.HandleFunc("GET /v1/insights", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		records, err := repo.ListInsightLedger(r.Context(), storage.InsightLedgerFilter{
			TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), AppID: strings.TrimSpace(r.URL.Query().Get("app_id")), Domain: strings.TrimSpace(r.URL.Query().Get("domain")), UseCase: strings.TrimSpace(r.URL.Query().Get("use_case")), SourceID: strings.TrimSpace(r.URL.Query().Get("source_id")),
			Dataset: strings.TrimSpace(r.URL.Query().Get("dataset")), InsightType: strings.TrimSpace(r.URL.Query().Get("insight_type")),
			Status: strings.TrimSpace(r.URL.Query().Get("status")), Limit: queryLimit(r, 50),
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list insights")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"insights": insightResponses(records)})
	})

	mux.HandleFunc("GET /v1/insights/{insight_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		record, err := repo.GetInsightLedger(r.Context(), r.PathValue("insight_id"))
		if err != nil {
			writeQueryError(w, err, "insight_not_found", "insight not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"insight": insightResponse(record)})
	})

	mux.HandleFunc("POST /v1/insights/{insight_id}/review", insightLifecycleHandler(cfg, storage.InsightStatusReviewed, "review"))
	mux.HandleFunc("POST /v1/insights/{insight_id}/dismiss", insightLifecycleHandler(cfg, storage.InsightStatusDismissed, "dismiss"))
	mux.HandleFunc("POST /v1/insights/{insight_id}/archive", insightLifecycleHandler(cfg, storage.InsightStatusArchived, "archive"))

	mux.HandleFunc("GET /v1/idempotency", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		sourceID := strings.TrimSpace(r.URL.Query().Get("source_id"))
		key := strings.TrimSpace(r.URL.Query().Get("idempotency_key"))
		if tenantID == "" || sourceID == "" || key == "" {
			writeError(w, http.StatusBadRequest, "missing_query", "tenant_id, source_id, and idempotency_key are required")
			return
		}
		record, err := repo.GetIdempotencyRecord(r.Context(), tenantID, sourceID, key)
		if err != nil {
			writeQueryError(w, err, "idempotency_not_found", "idempotency record not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"idempotency": idempotencyResponse(record)})
	})

	mux.HandleFunc("POST /v1/algorithms/definitions", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		var req algorithmDefinitionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		record := algorithmDefinitionRecord(req)
		if err := repo.UpsertAlgorithmDefinition(r.Context(), record); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_algorithm_definition", err.Error())
			return
		}
		stored, err := repo.GetAlgorithmDefinition(r.Context(), record.TenantID, record.AlgorithmID)
		if err != nil {
			writeQueryError(w, err, "algorithm_definition_not_found", "Algorithm definition not found")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"algorithm_definition": algorithmDefinitionResponse(stored)})
	})

	mux.HandleFunc("GET /v1/algorithms/definitions", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "invalid_algorithm_filter", "tenant_id is required")
			return
		}
		records, err := repo.ListAlgorithmDefinitions(r.Context(), storage.AlgorithmDefinitionFilter{TenantID: tenantID, AlgorithmType: strings.TrimSpace(r.URL.Query().Get("algorithm_type")), RuntimeType: strings.TrimSpace(r.URL.Query().Get("runtime_type")), Status: strings.TrimSpace(r.URL.Query().Get("status")), Limit: queryLimit(r, 50)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list algorithm definitions")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"algorithm_definitions": algorithmDefinitionResponses(records)})
	})

	mux.HandleFunc("GET /v1/algorithms/definitions/{algorithm_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "invalid_algorithm_filter", "tenant_id is required")
			return
		}
		record, err := repo.GetAlgorithmDefinition(r.Context(), tenantID, r.PathValue("algorithm_id"))
		if err != nil {
			writeQueryError(w, err, "algorithm_definition_not_found", "Algorithm definition not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"algorithm_definition": algorithmDefinitionResponse(record)})
	})

	mux.HandleFunc("POST /v1/algorithms/execution-requests", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		var req algorithmExecutionRequestCreate
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		record := algorithmExecutionRequestRecord(req, replayActor(r, req.RequestedBy))
		if err := repo.UpsertAlgorithmExecutionRequest(r.Context(), record); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_algorithm_execution_request", err.Error())
			return
		}
		stored, err := repo.GetAlgorithmExecutionRequest(r.Context(), record.TenantID, record.ExecutionRequestID)
		if err != nil {
			writeQueryError(w, err, "algorithm_execution_request_not_found", "Algorithm execution request not found")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"algorithm_execution_request": algorithmExecutionRequestResponse(stored)})
	})

	mux.HandleFunc("GET /v1/algorithms/execution-requests", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "invalid_algorithm_filter", "tenant_id is required")
			return
		}
		records, err := repo.ListAlgorithmExecutionRequests(r.Context(), storage.AlgorithmExecutionRequestFilter{TenantID: tenantID, AlgorithmID: strings.TrimSpace(r.URL.Query().Get("algorithm_id")), Status: strings.TrimSpace(r.URL.Query().Get("status")), CorrelationID: strings.TrimSpace(r.URL.Query().Get("correlation_id")), Limit: queryLimit(r, 50)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list algorithm execution requests")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"algorithm_execution_requests": algorithmExecutionRequestResponses(records)})
	})

	mux.HandleFunc("GET /v1/algorithms/execution-requests/{execution_request_id}/summary", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "invalid_algorithm_filter", "tenant_id is required")
			return
		}
		executionRequestID := r.PathValue("execution_request_id")
		request, err := repo.GetAlgorithmExecutionRequest(r.Context(), tenantID, executionRequestID)
		if err != nil {
			writeQueryError(w, err, "algorithm_execution_request_not_found", "Algorithm execution request not found")
			return
		}
		results, err := repo.ListAlgorithmResults(r.Context(), storage.AlgorithmResultFilter{TenantID: tenantID, ExecutionRequestID: executionRequestID, Limit: 200})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list algorithm results")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"algorithm_execution_summary": algorithmExecutionSummaryResponse(request, results, queryLimit(r, 10))})
	})

	mux.HandleFunc("GET /v1/algorithms/execution-requests/{execution_request_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "invalid_algorithm_filter", "tenant_id is required")
			return
		}
		record, err := repo.GetAlgorithmExecutionRequest(r.Context(), tenantID, r.PathValue("execution_request_id"))
		if err != nil {
			writeQueryError(w, err, "algorithm_execution_request_not_found", "Algorithm execution request not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"algorithm_execution_request": algorithmExecutionRequestResponse(record)})
	})

	mux.HandleFunc("GET /v1/algorithms/results", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "invalid_algorithm_filter", "tenant_id is required")
			return
		}
		records, err := repo.ListAlgorithmResults(r.Context(), storage.AlgorithmResultFilter{TenantID: tenantID, AlgorithmID: strings.TrimSpace(r.URL.Query().Get("algorithm_id")), ExecutionRequestID: strings.TrimSpace(r.URL.Query().Get("execution_request_id")), ResultType: strings.TrimSpace(r.URL.Query().Get("result_type")), Severity: strings.TrimSpace(r.URL.Query().Get("severity")), CorrelationID: strings.TrimSpace(r.URL.Query().Get("correlation_id")), Limit: queryLimit(r, 50)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list algorithm results")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"algorithm_results": algorithmResultResponses(records)})
	})

	mux.HandleFunc("GET /v1/algorithms/results/{algorithm_result_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "invalid_algorithm_filter", "tenant_id is required")
			return
		}
		record, err := repo.GetAlgorithmResult(r.Context(), tenantID, r.PathValue("algorithm_result_id"))
		if err != nil {
			writeQueryError(w, err, "algorithm_result_not_found", "Algorithm result not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"algorithm_result": algorithmResultResponse(record)})
	})

	mux.HandleFunc("GET /v1/algorithms/signal-proposals", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "invalid_algorithm_filter", "tenant_id is required")
			return
		}
		records, err := repo.ListAlgorithmSignalProposals(r.Context(), storage.AlgorithmSignalProposalFilter{TenantID: tenantID, AlgorithmID: strings.TrimSpace(r.URL.Query().Get("algorithm_id")), ExecutionRequestID: strings.TrimSpace(r.URL.Query().Get("execution_request_id")), AlgorithmResultID: strings.TrimSpace(r.URL.Query().Get("algorithm_result_id")), Status: strings.TrimSpace(r.URL.Query().Get("status")), Severity: strings.TrimSpace(r.URL.Query().Get("severity")), CorrelationID: strings.TrimSpace(r.URL.Query().Get("correlation_id")), Limit: queryLimit(r, 50)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list algorithm signal proposals")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"algorithm_signal_proposals": algorithmSignalProposalResponses(records)})
	})

	mux.HandleFunc("GET /v1/algorithms/signal-proposals/{proposal_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "invalid_algorithm_filter", "tenant_id is required")
			return
		}
		record, err := repo.GetAlgorithmSignalProposal(r.Context(), tenantID, r.PathValue("proposal_id"))
		if err != nil {
			writeQueryError(w, err, "algorithm_signal_proposal_not_found", "Algorithm signal proposal not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"algorithm_signal_proposal": algorithmSignalProposalResponse(record)})
	})

	mux.HandleFunc("POST /v1/algorithms/signal-proposals/{proposal_id}/decision", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		var req algorithmSignalProposalDecisionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		tenantID := strings.TrimSpace(req.TenantID)
		if tenantID == "" {
			tenantID = strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		}
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "invalid_algorithm_filter", "tenant_id is required")
			return
		}
		status := strings.TrimSpace(req.Status)
		if status != storage.AlgorithmSignalProposalStatusProposed && status != storage.AlgorithmSignalProposalStatusReviewed && status != storage.AlgorithmSignalProposalStatusRejected && status != storage.AlgorithmSignalProposalStatusSuperseded {
			writeError(w, http.StatusBadRequest, "invalid_status", "algorithm signal proposal decision status is invalid")
			return
		}
		metadata := algorithmJSONOrDefaultObject(req.Metadata)
		record, err := repo.MutateAlgorithmSignalProposal(r.Context(), storage.AlgorithmSignalProposalMutation{TenantID: tenantID, ProposalID: r.PathValue("proposal_id"), Status: status, ReviewedBy: replayActor(r, req.Actor), DecisionNote: strings.TrimSpace(req.Note), DecidedAt: time.Now().UTC(), MetadataJSON: metadata})
		if err != nil {
			writeQueryError(w, err, "algorithm_signal_proposal_not_found", "Algorithm signal proposal not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"algorithm_signal_proposal": algorithmSignalProposalResponse(record)})
	})

	mux.HandleFunc("GET /v1/tenants/{tenant_id}/catalog/sources", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		tenantID := strings.TrimSpace(r.PathValue("tenant_id"))
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "missing_path", "tenant_id is required")
			return
		}
		sources, err := repo.ListCatalogSources(r.Context(), tenantID, queryLimit(r, 50))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list catalog sources")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"sources": catalogSourceResponses(sources)})
	})

	mux.HandleFunc("GET /v1/tenants/{tenant_id}/catalog/pipelines", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		tenantID := strings.TrimSpace(r.PathValue("tenant_id"))
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "missing_path", "tenant_id is required")
			return
		}
		pipelines, err := repo.ListCatalogPipelines(r.Context(), tenantID, queryLimit(r, 50))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list catalog pipelines")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"pipelines": catalogPipelineResponses(pipelines)})
	})

	mux.HandleFunc("GET /v1/tenants/{tenant_id}/catalog/rules", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		tenantID := strings.TrimSpace(r.PathValue("tenant_id"))
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "missing_path", "tenant_id is required")
			return
		}
		rules, err := repo.ListCatalogRules(r.Context(), tenantID, queryLimit(r, 50))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list catalog rules")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"rules": catalogRuleResponses(rules)})
	})

	mux.HandleFunc("GET /v1/tenants/{tenant_id}/marketops/assets", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		tenantID := strings.TrimSpace(r.PathValue("tenant_id"))
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "missing_path", "tenant_id is required")
			return
		}
		universeGroup := strings.TrimSpace(r.URL.Query().Get("universe_group"))
		activeOnly := !strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("active_only")), "false")
		assets, err := repo.ListMarketOpsAssets(r.Context(), tenantID, universeGroup, activeOnly, queryLimit(r, 50))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps assets")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"assets": marketOpsAssetResponses(assets)})
	})

	mux.HandleFunc("GET /v1/streams/dashboard", func(w http.ResponseWriter, r *http.Request) {
		channels, err := dashboardStreamChannels(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_channel", err.Error())
			return
		}
		streamDashboard(w, r, serviceName, cfg.QueryRepository, channels, defaultStreamInterval)
	})

	mux.HandleFunc("POST /v1/events/raw", func(w http.ResponseWriter, r *http.Request) {
		if cfg.Publisher == nil || rawTopic == "" || cfg.PublishRepository == nil {
			writeError(w, http.StatusServiceUnavailable, "ingest_unavailable", "raw event ingestion is not fully configured")
			return
		}

		payload, fields, err := readJSONObject(w, r, defaultMaxRawEventBytes)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}

		eventID := firstNonEmpty(headerValue(r, "X-SignalOps-Event-ID"), jsonStringField(fields, "event_id"), newID("evt"))
		idempotencyKey := firstNonEmpty(headerValue(r, "X-Idempotency-Key"), jsonStringField(fields, "idempotency_key"), eventID)
		correlationID := firstNonEmpty(headerValue(r, "X-Correlation-ID"), jsonStringField(fields, "correlation_id"), newID("corr"))
		causationID := firstNonEmpty(headerValue(r, "X-Causation-ID"), jsonStringField(fields, "causation_id"))
		traceID := firstNonEmpty(headerValue(r, "X-Trace-ID"), jsonStringField(fields, "trace_id"))
		acceptedAt := time.Now().UTC()
		ingest, err := rawIngestPersistenceFields(fields, acceptedAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_event", err.Error())
			return
		}

		publishCtx, cancel := context.WithTimeout(r.Context(), defaultPublishTimeout)
		result, err := cfg.Publisher.Publish(publishCtx, broker.Message{
			Topic:         rawTopic,
			Key:           idempotencyKey,
			Value:         payload,
			Headers:       rawEventHeaders(eventID, idempotencyKey),
			CorrelationID: correlationID,
			CausationID:   causationID,
			TraceID:       traceID,
			PublishedAt:   acceptedAt,
		})
		cancel()
		if err != nil {
			writeError(w, http.StatusBadGateway, "publish_failed", "failed to publish raw event")
			return
		}
		ledger, idempotency, err := publishedRawEventRecords(payload, ingest, eventID, idempotencyKey, correlationID, causationID, traceID, result, acceptedAt)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "persistence_mapping_failed", "failed to map published raw event")
			return
		}
		persistCtx, persistCancel := context.WithTimeout(r.Context(), defaultPublishTimeout)
		err = cfg.PublishRepository.PersistPublishedRawEvent(persistCtx, ledger, idempotency)
		persistCancel()
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "persistence_failed", "raw event was published but its audit state could not be persisted")
			return
		}

		writeJSON(w, http.StatusAccepted, map[string]any{
			"status":          "accepted",
			"event_id":        eventID,
			"idempotency_key": idempotencyKey,
			"correlation_id":  correlationID,
			"topic":           result.Topic,
			"partition":       result.Partition,
			"offset":          result.Offset,
		})
	})

	return authMiddleware(mux, cfg.Auth)
}

type rawIngestFields struct {
	TenantID        string
	SourceID        string
	SourceAdapter   string
	Dataset         string
	ObservationTime time.Time
	ProcessingTime  time.Time
	EntityHintsJSON []byte
}

func rawIngestPersistenceFields(fields map[string]json.RawMessage, acceptedAt time.Time) (rawIngestFields, error) {
	result := rawIngestFields{
		TenantID:        jsonStringField(fields, "tenant_id"),
		SourceID:        jsonStringField(fields, "source_id"),
		SourceAdapter:   jsonStringField(fields, "source_adapter"),
		Dataset:         jsonStringField(fields, "dataset"),
		ProcessingTime:  acceptedAt,
		EntityHintsJSON: []byte("[]"),
	}
	for name, value := range map[string]string{"tenant_id": result.TenantID, "source_id": result.SourceID, "source_adapter": result.SourceAdapter, "dataset": result.Dataset} {
		if value == "" {
			return rawIngestFields{}, fmt.Errorf("%s is required", name)
		}
	}
	observationTime, err := parseEventTime(fields, "observation_time")
	if err != nil {
		return rawIngestFields{}, err
	}
	result.ObservationTime = observationTime
	if jsonStringField(fields, "processing_time") != "" {
		result.ProcessingTime, err = parseEventTime(fields, "processing_time")
		if err != nil {
			return rawIngestFields{}, err
		}
	}
	if raw, ok := fields["entity_hints"]; ok {
		var hints []json.RawMessage
		if err := json.Unmarshal(raw, &hints); err != nil {
			return rawIngestFields{}, errors.New("entity_hints must be an array")
		}
		result.EntityHintsJSON = append([]byte(nil), raw...)
	}
	return result, nil
}

func parseEventTime(fields map[string]json.RawMessage, name string) (time.Time, error) {
	value := jsonStringField(fields, name)
	if value == "" {
		return time.Time{}, fmt.Errorf("%s is required", name)
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be an RFC3339 timestamp", name)
	}
	return parsed.UTC(), nil
}

func publishedRawEventRecords(payload []byte, ingest rawIngestFields, eventID, idempotencyKey, correlationID, causationID, traceID string, result broker.PublishResult, publishedAt time.Time) (storage.RawEventLedgerRecord, storage.IdempotencyRecord, error) {
	partition, offset := result.Partition, result.Offset
	metadata, err := json.Marshal(map[string]any{
		"correlation_id": correlationID,
		"causation_id":   causationID,
		"trace_id":       traceID,
		"route":          "/v1/events/raw",
		"published_at":   publishedAt.Format(time.RFC3339Nano),
	})
	if err != nil {
		return storage.RawEventLedgerRecord{}, storage.IdempotencyRecord{}, err
	}
	hash := sha256.Sum256(payload)
	ledger := storage.RawEventLedgerRecord{
		EventID: eventID, TenantID: ingest.TenantID, SourceID: ingest.SourceID,
		AppID: recordAppIDFromPayload(payload), Domain: recordDomainFromPayload(payload, rawPayloadString(payload, "source_domain")), UseCase: recordUseCaseFromPayload(payload),
		SourceAdapter: ingest.SourceAdapter, Dataset: ingest.Dataset, IdempotencyKey: idempotencyKey,
		ObservationTime: ingest.ObservationTime, ProcessingTime: ingest.ProcessingTime,
		BrokerTopic: result.Topic, BrokerPartition: &partition, BrokerOffset: &offset,
		PayloadJSON: payload, EntityHintsJSON: ingest.EntityHintsJSON,
	}
	idempotency := storage.IdempotencyRecord{
		TenantID: ingest.TenantID, SourceID: ingest.SourceID, IdempotencyKey: idempotencyKey,
		EventID: eventID, SourceAdapter: ingest.SourceAdapter, Dataset: ingest.Dataset,
		Topic: result.Topic, Partition: &partition, Offset: &offset,
		PayloadHash: "sha256:" + hex.EncodeToString(hash[:]), Status: storage.IdempotencyStatusPublished,
		MetadataJSON: metadata,
	}
	return ledger, idempotency, nil
}

func rawPayloadString(payload []byte, name string) string {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		return ""
	}
	return jsonStringField(fields, name)
}

func recordAppIDFromPayload(payload []byte) string {
	return appMetadataFromPayload(payload, "").AppID
}

func recordDomainFromPayload(payload []byte, fallbackDomain string) string {
	return appMetadataFromPayload(payload, fallbackDomain).Domain
}

func recordUseCaseFromPayload(payload []byte) string {
	return appMetadataFromPayload(payload, "").UseCase
}

func appMetadataFromPayload(payload []byte, fallbackDomain string) appmeta.Metadata {
	return appmeta.Normalize(rawPayloadString(payload, "app_id"), rawPayloadString(payload, "domain"), rawPayloadString(payload, "use_case"), fallbackDomain)
}

type streamChannelSet map[string]bool

type sseEvent struct {
	Event string
	ID    string
	Data  any
}

func dashboardStreamChannels(r *http.Request) (streamChannelSet, error) {
	value := strings.TrimSpace(r.URL.Query().Get("channels"))
	if value == "" {
		return streamChannelSet{
			"health":         true,
			"scheduler_run":  true,
			"raw_event":      true,
			"provider_usage": true,
			"heartbeat":      true,
		}, nil
	}

	channels := streamChannelSet{}
	for _, part := range strings.Split(value, ",") {
		channel := strings.TrimSpace(part)
		if channel == "" {
			continue
		}
		if _, ok := supportedDashboardStreamChannels[channel]; !ok {
			return nil, fmt.Errorf("unsupported stream channel %q", channel)
		}
		switch channel {
		case "runs":
			channel = "scheduler_run"
		case "raw_events":
			channel = "raw_event"
		}
		channels[channel] = true
	}
	if len(channels) == 0 {
		return nil, errors.New("at least one stream channel is required")
	}
	return channels, nil
}

func streamDashboard(w http.ResponseWriter, r *http.Request, serviceName string, repo storage.QueryRepository, channels streamChannelSet, interval time.Duration) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming_unsupported", "response streaming is not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	seen := map[string]struct{}{}
	emit := func(event sseEvent) bool {
		if err := writeSSE(w, event); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	if !emit(sseEvent{Event: "heartbeat", Data: heartbeatPayload(serviceName)}) {
		return
	}

	if repo == nil {
		if channels["health"] {
			if !emit(sseEvent{Event: "error", Data: map[string]string{
				"error":   "storage_unavailable",
				"message": "query storage is not configured",
			}}) {
				return
			}
		}
		streamHeartbeatsUntilCanceled(r, serviceName, interval, emit)
		return
	}

	if !emitDashboardSnapshot(r.Context(), repo, serviceName, channels, seen, emit) {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if channels["heartbeat"] && !emit(sseEvent{Event: "heartbeat", Data: heartbeatPayload(serviceName)}) {
				return
			}
			if !emitDashboardSnapshot(r.Context(), repo, serviceName, channels, seen, emit) {
				return
			}
		}
	}
}

func emitDashboardSnapshot(ctx context.Context, repo storage.QueryRepository, serviceName string, channels streamChannelSet, seen map[string]struct{}, emit func(sseEvent) bool) bool {
	if channels["health"] {
		if !emit(sseEvent{Event: "health", Data: map[string]string{
			"status":  "ok",
			"service": serviceName,
			"time":    time.Now().UTC().Format(time.RFC3339),
		}}) {
			return false
		}
	}
	if channels["scheduler_run"] {
		runs, err := repo.ListSchedulerRuns(ctx, 50)
		if err != nil {
			return emit(sseEvent{Event: "error", Data: map[string]string{"error": "query_failed", "message": "failed to list scheduler runs"}})
		}
		for _, run := range runs {
			key := "scheduler_run:" + run.RunID
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			if !emit(sseEvent{Event: "scheduler_run", ID: run.RunID, Data: schedulerRunResponse(run)}) {
				return false
			}
		}
	}
	if channels["raw_event"] {
		records, err := repo.ListRawEventLedger(ctx, storage.RawEventLedgerFilter{Limit: 50})
		if err != nil {
			return emit(sseEvent{Event: "error", Data: map[string]string{"error": "query_failed", "message": "failed to list raw events"}})
		}
		for _, record := range records {
			key := "raw_event:" + record.EventID
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			if !emit(sseEvent{Event: "raw_event", ID: record.EventID, Data: rawEventResponse(record)}) {
				return false
			}
		}
	}
	if channels["provider_usage"] {
		records, err := repo.ListProviderUsage(ctx, "", 50)
		if err != nil {
			return emit(sseEvent{Event: "error", Data: map[string]string{"error": "query_failed", "message": "failed to list provider usage"}})
		}
		for _, record := range records {
			key := "provider_usage:" + record.UsageID
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			if !emit(sseEvent{Event: "provider_usage", ID: record.UsageID, Data: providerUsageResponses([]storage.ProviderUsageRecord{record})[0]}) {
				return false
			}
		}
	}
	return true
}

func heartbeatPayload(serviceName string) map[string]string {
	return map[string]string{
		"status":  "alive",
		"service": serviceName,
		"time":    time.Now().UTC().Format(time.RFC3339),
	}
}

func streamHeartbeatsUntilCanceled(r *http.Request, serviceName string, interval time.Duration, emit func(sseEvent) bool) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if !emit(sseEvent{Event: "heartbeat", Data: heartbeatPayload(serviceName)}) {
				return
			}
		}
	}
}

func writeSSE(w io.Writer, event sseEvent) error {
	if event.Event != "" {
		if _, err := fmt.Fprintf(w, "event: %s\n", event.Event); err != nil {
			return err
		}
	}
	if event.ID != "" {
		if _, err := fmt.Fprintf(w, "id: %s\n", event.ID); err != nil {
			return err
		}
	}
	data, err := json.Marshal(event.Data)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
		return err
	}
	return nil
}

type schedulerRunDTO struct {
	RunID            string          `json:"run_id"`
	TenantID         string          `json:"tenant_id"`
	SourceID         string          `json:"source_id"`
	SourceAdapter    string          `json:"source_adapter"`
	Datasets         []string        `json:"datasets"`
	ObservationDate  time.Time       `json:"observation_date"`
	DryRun           bool            `json:"dry_run"`
	Status           string          `json:"status"`
	StartedAt        time.Time       `json:"started_at"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty"`
	EventsBuilt      int             `json:"events_built"`
	EventsPublished  int             `json:"events_published"`
	ProviderRequests int             `json:"provider_requests"`
	ProviderRetries  int             `json:"provider_retries"`
	Failures         int             `json:"failures"`
	Config           json.RawMessage `json:"config"`
	Report           json.RawMessage `json:"report"`
	ErrorMessage     string          `json:"error_message,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type replayJobDTO struct {
	ReplayJobID  string          `json:"replay_job_id"`
	TenantID     string          `json:"tenant_id"`
	SourceID     string          `json:"source_id,omitempty"`
	Dataset      string          `json:"dataset,omitempty"`
	SourceKind   string          `json:"source_kind"`
	ReplayMode   string          `json:"replay_mode"`
	Status       string          `json:"status"`
	RequestedBy  string          `json:"requested_by"`
	WindowStart  time.Time       `json:"window_start"`
	WindowEnd    time.Time       `json:"window_end"`
	StartedAt    *time.Time      `json:"started_at,omitempty"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`
	Filters      json.RawMessage `json:"filters"`
	Options      json.RawMessage `json:"options"`
	Result       json.RawMessage `json:"result"`
	ErrorMessage string          `json:"error_message,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type replayJobCreateRequest struct {
	TenantID    string          `json:"tenant_id"`
	SourceID    string          `json:"source_id"`
	Dataset     string          `json:"dataset"`
	SourceKind  string          `json:"source_kind"`
	ReplayMode  string          `json:"replay_mode"`
	RequestedBy string          `json:"requested_by"`
	WindowStart string          `json:"window_start"`
	WindowEnd   string          `json:"window_end"`
	Filters     json.RawMessage `json:"filters"`
	Options     json.RawMessage `json:"options"`
}

type replayStatusDTO struct {
	GeneratedAt time.Time               `json:"generated_at"`
	JobCounts   map[string]int          `json:"job_counts"`
	Workers     []replayWorkerStatusDTO `json:"workers"`
	LatestJobs  []replayJobDTO          `json:"latest_jobs"`
}

type replayWorkerStatusDTO struct {
	WorkerID                 string          `json:"worker_id"`
	Status                   string          `json:"status"`
	Health                   string          `json:"health"`
	ProcessStartedAt         time.Time       `json:"process_started_at"`
	LastSeenAt               time.Time       `json:"last_seen_at"`
	LastClaimedAt            *time.Time      `json:"last_claimed_at,omitempty"`
	LastClaimedReplayJobID   string          `json:"last_claimed_replay_job_id,omitempty"`
	LastCompletedAt          *time.Time      `json:"last_completed_at,omitempty"`
	LastCompletedReplayJobID string          `json:"last_completed_replay_job_id,omitempty"`
	LastErrorAt              *time.Time      `json:"last_error_at,omitempty"`
	LastErrorMessage         string          `json:"last_error_message,omitempty"`
	Metadata                 json.RawMessage `json:"metadata"`
	CreatedAt                time.Time       `json:"created_at"`
	UpdatedAt                time.Time       `json:"updated_at"`
}

type providerUsageDTO struct {
	UsageID      string          `json:"usage_id"`
	RunID        string          `json:"run_id"`
	Provider     string          `json:"provider"`
	Dataset      string          `json:"dataset"`
	RequestCount int             `json:"request_count"`
	RetryCount   int             `json:"retry_count"`
	EventCount   int             `json:"event_count"`
	Budget       json.RawMessage `json:"budget"`
	CreatedAt    time.Time       `json:"created_at"`
}

type rawEventDTO struct {
	EventID         string          `json:"event_id"`
	TenantID        string          `json:"tenant_id"`
	AppID           string          `json:"app_id"`
	Domain          string          `json:"domain"`
	UseCase         string          `json:"use_case"`
	SourceID        string          `json:"source_id"`
	SourceAdapter   string          `json:"source_adapter"`
	Dataset         string          `json:"dataset"`
	IdempotencyKey  string          `json:"idempotency_key"`
	ObservationTime time.Time       `json:"observation_time"`
	ProcessingTime  time.Time       `json:"processing_time"`
	BrokerTopic     string          `json:"broker_topic,omitempty"`
	BrokerPartition *int32          `json:"broker_partition,omitempty"`
	BrokerOffset    *int64          `json:"broker_offset,omitempty"`
	Payload         json.RawMessage `json:"payload"`
	EntityHints     json.RawMessage `json:"entity_hints"`
	CreatedAt       time.Time       `json:"created_at"`
}

type normalizedEventDTO struct {
	EventID             string          `json:"event_id"`
	TenantID            string          `json:"tenant_id"`
	AppID               string          `json:"app_id"`
	Domain              string          `json:"domain"`
	UseCase             string          `json:"use_case"`
	SourceID            string          `json:"source_id"`
	SourceAdapter       string          `json:"source_adapter"`
	Dataset             string          `json:"dataset"`
	IdempotencyKey      string          `json:"idempotency_key"`
	SchemaID            string          `json:"schema_id"`
	SchemaVersion       string          `json:"schema_version"`
	ObservationTime     time.Time       `json:"observation_time"`
	ProcessingTime      time.Time       `json:"processing_time"`
	Confidence          float64         `json:"confidence"`
	RawTopic            string          `json:"raw_topic"`
	RawPartition        int32           `json:"raw_partition"`
	RawOffset           int64           `json:"raw_offset"`
	NormalizedTopic     string          `json:"normalized_topic"`
	NormalizedPartition int32           `json:"normalized_partition"`
	NormalizedOffset    int64           `json:"normalized_offset"`
	NormalizedPayload   json.RawMessage `json:"normalized_payload"`
	Entities            json.RawMessage `json:"entities"`
	Evidence            json.RawMessage `json:"evidence"`
	Metadata            json.RawMessage `json:"metadata"`
	Event               json.RawMessage `json:"event"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

type signalDTO struct {
	SignalID          string          `json:"signal_id"`
	TenantID          string          `json:"tenant_id"`
	AppID             string          `json:"app_id"`
	Domain            string          `json:"domain"`
	UseCase           string          `json:"use_case"`
	SourceID          string          `json:"source_id"`
	SourceDomain      string          `json:"source_domain"`
	SourceAdapter     string          `json:"source_adapter"`
	IngestionMode     string          `json:"ingestion_mode"`
	Dataset           string          `json:"dataset"`
	EventIDs          []string        `json:"event_ids"`
	ArtifactIDs       []string        `json:"artifact_ids"`
	SignalType        string          `json:"signal_type"`
	DetectorID        string          `json:"detector_id"`
	DetectorVersion   string          `json:"detector_version"`
	ModelVersion      string          `json:"model_version"`
	SignalTime        time.Time       `json:"timestamp"`
	ObservationTime   time.Time       `json:"observation_time"`
	EffectiveTime     time.Time       `json:"effective_time"`
	ProcessingTime    time.Time       `json:"processing_time"`
	WindowStart       time.Time       `json:"window_start"`
	WindowEnd         time.Time       `json:"window_end"`
	Confidence        float64         `json:"confidence"`
	Severity          string          `json:"severity"`
	Entities          json.RawMessage `json:"entities"`
	SupportingMetrics json.RawMessage `json:"supporting_metrics"`
	GraphTargets      json.RawMessage `json:"graph_targets"`
	SemanticEvidence  json.RawMessage `json:"semantic_evidence"`
	Evidence          json.RawMessage `json:"evidence"`
	Recommendation    json.RawMessage `json:"recommendation"`
	CorrelationID     string          `json:"correlation_id"`
	TraceID           string          `json:"trace_id,omitempty"`
	CausationID       string          `json:"causation_id,omitempty"`
	ReplayJobID       string          `json:"replay_job_id,omitempty"`
	BrokerTopic       string          `json:"broker_topic"`
	BrokerPartition   int32           `json:"broker_partition"`
	BrokerOffset      int64           `json:"broker_offset"`
	Event             json.RawMessage `json:"event"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type alertDTO struct {
	AlertID         string          `json:"alert_id"`
	TenantID        string          `json:"tenant_id"`
	AppID           string          `json:"app_id"`
	Domain          string          `json:"domain"`
	UseCase         string          `json:"use_case"`
	SourceID        string          `json:"source_id"`
	SourceDomain    string          `json:"source_domain"`
	SourceAdapter   string          `json:"source_adapter"`
	Dataset         string          `json:"dataset"`
	SignalID        string          `json:"signal_id"`
	DetectorID      string          `json:"detector_id"`
	AlertType       string          `json:"alert_type"`
	Severity        string          `json:"severity"`
	Status          string          `json:"status"`
	Title           string          `json:"title"`
	Summary         string          `json:"summary"`
	Confidence      float64         `json:"confidence"`
	EventIDs        []string        `json:"event_ids"`
	Entities        json.RawMessage `json:"entities"`
	Evidence        json.RawMessage `json:"evidence"`
	Recommendation  json.RawMessage `json:"recommendation"`
	CorrelationID   string          `json:"correlation_id"`
	FirstObservedAt time.Time       `json:"first_observed_at"`
	LastObservedAt  time.Time       `json:"last_observed_at"`
	AcknowledgedAt  *time.Time      `json:"acknowledged_at,omitempty"`
	AcknowledgedBy  string          `json:"acknowledged_by,omitempty"`
	ResolvedAt      *time.Time      `json:"resolved_at,omitempty"`
	ResolvedBy      string          `json:"resolved_by,omitempty"`
	Metadata        json.RawMessage `json:"metadata"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type insightDTO struct {
	InsightID         string          `json:"insight_id"`
	TenantID          string          `json:"tenant_id"`
	AppID             string          `json:"app_id"`
	Domain            string          `json:"domain"`
	UseCase           string          `json:"use_case"`
	SourceID          string          `json:"source_id"`
	SourceDomain      string          `json:"source_domain"`
	SourceAdapter     string          `json:"source_adapter"`
	Dataset           string          `json:"dataset"`
	SignalID          string          `json:"signal_id"`
	DetectorID        string          `json:"detector_id"`
	InsightType       string          `json:"insight_type"`
	Status            string          `json:"status"`
	Title             string          `json:"title"`
	Summary           string          `json:"summary"`
	Confidence        float64         `json:"confidence"`
	Severity          string          `json:"severity"`
	EventIDs          []string        `json:"event_ids"`
	Entities          json.RawMessage `json:"entities"`
	SupportingMetrics json.RawMessage `json:"supporting_metrics"`
	SemanticEvidence  json.RawMessage `json:"semantic_evidence"`
	Recommendation    json.RawMessage `json:"recommendation"`
	CorrelationID     string          `json:"correlation_id"`
	ObservedAt        time.Time       `json:"observed_at"`
	ReviewedAt        *time.Time      `json:"reviewed_at,omitempty"`
	ReviewedBy        string          `json:"reviewed_by,omitempty"`
	Metadata          json.RawMessage `json:"metadata"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type marketOpsAssetDTO struct {
	TenantID      string          `json:"tenant_id"`
	AppID         string          `json:"app_id"`
	Domain        string          `json:"domain"`
	UseCase       string          `json:"use_case"`
	SourceID      string          `json:"source_id"`
	UniverseGroup string          `json:"universe_group"`
	Rank          int             `json:"rank"`
	Ticker        string          `json:"ticker"`
	TickerKey     string          `json:"ticker_key"`
	Company       string          `json:"company"`
	CompanyKey    string          `json:"company_key"`
	AssetType     string          `json:"asset_type"`
	Exchange      string          `json:"exchange"`
	Sector        string          `json:"sector"`
	SectorKey     string          `json:"sector_key"`
	Industry      string          `json:"industry"`
	IndustryKey   string          `json:"industry_key"`
	IsActive      bool            `json:"is_active"`
	Metadata      json.RawMessage `json:"metadata"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type catalogSourceDTO struct {
	TenantID       string          `json:"tenant_id"`
	SourceID       string          `json:"source_id"`
	SourceDomain   string          `json:"source_domain"`
	SourceAdapter  string          `json:"source_adapter"`
	DisplayName    string          `json:"display_name"`
	Description    string          `json:"description"`
	Status         string          `json:"status"`
	IngestionModes []string        `json:"ingestion_modes"`
	Datasets       []string        `json:"datasets"`
	Metadata       json.RawMessage `json:"metadata"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type catalogPipelineDTO struct {
	TenantID      string          `json:"tenant_id"`
	PipelineID    string          `json:"pipeline_id"`
	SourceID      string          `json:"source_id"`
	SourceDomain  string          `json:"source_domain"`
	PipelineName  string          `json:"pipeline_name"`
	Description   string          `json:"description"`
	Status        string          `json:"status"`
	Stages        []string        `json:"stages"`
	InputDatasets []string        `json:"input_datasets"`
	OutputTopics  []string        `json:"output_topics"`
	Metadata      json.RawMessage `json:"metadata"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type catalogRuleDTO struct {
	TenantID     string          `json:"tenant_id"`
	RuleID       string          `json:"rule_id"`
	RuleName     string          `json:"rule_name"`
	Description  string          `json:"description"`
	RuleType     string          `json:"rule_type"`
	Severity     string          `json:"severity"`
	Status       string          `json:"status"`
	Version      int             `json:"version"`
	SourceID     string          `json:"source_id,omitempty"`
	PipelineID   string          `json:"pipeline_id,omitempty"`
	DatasetScope []string        `json:"dataset_scope"`
	EntityScope  []string        `json:"entity_scope"`
	Expression   json.RawMessage `json:"expression"`
	Actions      []string        `json:"actions"`
	Metadata     json.RawMessage `json:"metadata"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type idempotencyDTO struct {
	TenantID       string          `json:"tenant_id"`
	SourceID       string          `json:"source_id"`
	IdempotencyKey string          `json:"idempotency_key"`
	EventID        string          `json:"event_id"`
	SourceAdapter  string          `json:"source_adapter"`
	Dataset        string          `json:"dataset"`
	Topic          string          `json:"topic,omitempty"`
	Partition      *int32          `json:"partition,omitempty"`
	Offset         *int64          `json:"offset,omitempty"`
	PayloadHash    string          `json:"payload_hash,omitempty"`
	Status         string          `json:"status"`
	Metadata       json.RawMessage `json:"metadata"`
	FirstSeenAt    time.Time       `json:"first_seen_at"`
	LastSeenAt     time.Time       `json:"last_seen_at"`
}

type lifecycleMutationRequest struct {
	Actor  string `json:"actor"`
	Note   string `json:"note"`
	Reason string `json:"reason"`
}

func alertLifecycleHandler(cfg RouterConfig, status string, action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		req, err := readLifecycleMutationRequest(w, r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		actor := lifecycleActor(r, req.Actor)
		metadata, err := lifecycleMetadata(action, actor, req)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "metadata_failed", "failed to encode lifecycle metadata")
			return
		}
		record, err := repo.MutateAlertLifecycle(r.Context(), storage.AlertLifecycleMutation{
			AlertID: r.PathValue("alert_id"), Status: status, Actor: actor, MutatedAt: time.Now().UTC(), MetadataJSON: metadata,
		})
		if err != nil {
			writeQueryError(w, err, "alert_not_found", "alert not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"alert": alertResponse(record)})
	}
}

func insightLifecycleHandler(cfg RouterConfig, status string, action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, cfg.QueryRepository)
		if !ok {
			return
		}
		req, err := readLifecycleMutationRequest(w, r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		actor := lifecycleActor(r, req.Actor)
		metadata, err := lifecycleMetadata(action, actor, req)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "metadata_failed", "failed to encode lifecycle metadata")
			return
		}
		record, err := repo.MutateInsightLifecycle(r.Context(), storage.InsightLifecycleMutation{
			InsightID: r.PathValue("insight_id"), Status: status, Actor: actor, MutatedAt: time.Now().UTC(), MetadataJSON: metadata,
		})
		if err != nil {
			writeQueryError(w, err, "insight_not_found", "insight not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"insight": insightResponse(record)})
	}
}

func readMarketOpsBacktestCreateRequest(w http.ResponseWriter, r *http.Request) (marketOpsBacktestCreateRequest, error) {
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, defaultMaxRawEventBytes))
	decoder.DisallowUnknownFields()
	var req marketOpsBacktestCreateRequest
	if err := decoder.Decode(&req); err != nil {
		return marketOpsBacktestCreateRequest{}, err
	}
	return req, nil
}

func marketOpsBacktestConfigFromRequest(req marketOpsBacktestCreateRequest, actor string) (marketopsbacktest.Config, error) {
	windowStart, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(req.WindowStart))
	if err != nil {
		return marketopsbacktest.Config{}, errors.New("window_start must be an RFC3339 timestamp")
	}
	windowEnd, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(req.WindowEnd))
	if err != nil {
		return marketopsbacktest.Config{}, errors.New("window_end must be an RFC3339 timestamp")
	}
	runID := strings.TrimSpace(req.RunID)
	if runID == "" {
		runID = newID("bt_marketops")
	}
	maxRecords := req.MaxRecords
	if maxRecords <= 0 {
		maxRecords = 50
	}
	batchSize := req.BatchSize
	if batchSize <= 0 || batchSize > maxRecords {
		batchSize = maxRecords
	}
	pythonBin := strings.TrimSpace(os.Getenv("SIGNALOPS_MARKETOPS_BACKTEST_PYTHON_BIN"))
	if pythonBin == "" {
		pythonBin = "python3"
	}
	cfg := marketopsbacktest.Config{
		RunID: runID, TenantID: strings.TrimSpace(req.TenantID), AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance",
		SourceID: strings.TrimSpace(req.SourceID), SourceAdapter: strings.TrimSpace(req.SourceAdapter), Dataset: strings.TrimSpace(req.Dataset),
		DetectorID: strings.TrimSpace(req.DetectorID), DetectorVersion: strings.TrimSpace(req.DetectorVersion), RequestedBy: actor,
		WindowStart: windowStart.UTC(), WindowEnd: windowEnd.UTC(), Symbols: cleanSymbols(req.Symbols), MaxRecords: maxRecords, BatchSize: batchSize,
		AutoAcceptConfidence: req.AutoAcceptConfidence, PythonBin: pythonBin,
	}
	if strings.TrimSpace(cfg.TenantID) == "" {
		return marketopsbacktest.Config{}, errors.New("tenant_id is required")
	}
	if !cfg.WindowEnd.After(cfg.WindowStart) {
		return marketopsbacktest.Config{}, errors.New("window_end must be after window_start")
	}
	if maxRecords > 1000 {
		return marketopsbacktest.Config{}, errors.New("max_records must be between 1 and 1000")
	}
	return cfg, nil
}

func optionalQueryTime(r *http.Request, key string) (time.Time, error) {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

func querySymbols(r *http.Request) []string {
	values := []string{}
	for _, key := range []string{"symbol", "symbols", "subject_symbol"} {
		for _, raw := range r.URL.Query()[key] {
			for _, part := range strings.Split(raw, ",") {
				part = strings.TrimSpace(part)
				if part != "" {
					values = append(values, part)
				}
			}
		}
	}
	return cleanSymbols(values)
}

func marketOpsBacktestCampaignPlan(ctx context.Context, repo storage.QueryRepository, req marketOpsBacktestCampaignCreateRequest, campaignID string, actor string) (storage.MarketOpsBacktestCampaignRecord, []marketopsbacktest.Config, error) {
	windowStart, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(req.WindowStart))
	if err != nil {
		return storage.MarketOpsBacktestCampaignRecord{}, nil, errors.New("window_start must be an RFC3339 timestamp")
	}
	windowEnd, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(req.WindowEnd))
	if err != nil {
		return storage.MarketOpsBacktestCampaignRecord{}, nil, errors.New("window_end must be an RFC3339 timestamp")
	}
	if !windowEnd.After(windowStart) {
		return storage.MarketOpsBacktestCampaignRecord{}, nil, errors.New("window_end must be after window_start")
	}
	tenantID := strings.TrimSpace(req.TenantID)
	if tenantID == "" {
		return storage.MarketOpsBacktestCampaignRecord{}, nil, errors.New("tenant_id is required")
	}
	maxSymbols, err := boundedInt(req.MaxSymbols, 5, 1, 50, "max_symbols")
	if err != nil {
		return storage.MarketOpsBacktestCampaignRecord{}, nil, err
	}
	maxWindows, err := boundedInt(req.MaxWindows, 5, 1, 60, "max_windows")
	if err != nil {
		return storage.MarketOpsBacktestCampaignRecord{}, nil, err
	}
	maxRuns, err := boundedInt(req.MaxRuns, 25, 1, 250, "max_runs")
	if err != nil {
		return storage.MarketOpsBacktestCampaignRecord{}, nil, err
	}
	maxRecords, err := boundedInt(req.MaxRecords, 50, 1, 1000, "max_records")
	if err != nil {
		return storage.MarketOpsBacktestCampaignRecord{}, nil, err
	}
	batchSize := req.BatchSize
	if batchSize <= 0 || batchSize > maxRecords {
		batchSize = maxRecords
	}
	if batchSize > 1000 {
		return storage.MarketOpsBacktestCampaignRecord{}, nil, errors.New("batch_size must be between 1 and 1000")
	}
	stepDays := req.WindowStepDays
	if stepDays <= 0 {
		stepDays = 1
	}
	if stepDays > 31 {
		return storage.MarketOpsBacktestCampaignRecord{}, nil, errors.New("window_step_days must be between 1 and 31")
	}
	datasets := cleanDatasetScope(req.DatasetScope)
	if len(datasets) == 0 {
		datasets = []string{"equity_eod_prices"}
	}
	symbols := cleanSymbols(req.Symbols)
	universeGroup := strings.TrimSpace(req.UniverseGroup)
	if len(symbols) == 0 && universeGroup != "" {
		assets, err := repo.ListMarketOpsAssets(ctx, tenantID, universeGroup, true, maxSymbols)
		if err != nil {
			return storage.MarketOpsBacktestCampaignRecord{}, nil, err
		}
		for _, asset := range assets {
			if len(symbols) >= maxSymbols {
				break
			}
			ticker := strings.ToUpper(strings.TrimSpace(asset.Ticker))
			if ticker != "" {
				symbols = append(symbols, ticker)
			}
		}
	}
	if len(symbols) == 0 {
		return storage.MarketOpsBacktestCampaignRecord{}, nil, errors.New("symbols or universe_group is required")
	}
	if len(symbols) > maxSymbols {
		symbols = symbols[:maxSymbols]
	}
	windows := marketOpsCampaignWindows(windowStart.UTC(), windowEnd.UTC(), stepDays, maxWindows)
	if len(windows) == 0 {
		return storage.MarketOpsBacktestCampaignRecord{}, nil, errors.New("at least one campaign window is required")
	}
	pythonBin := strings.TrimSpace(os.Getenv("SIGNALOPS_MARKETOPS_BACKTEST_PYTHON_BIN"))
	if pythonBin == "" {
		pythonBin = "python3"
	}
	detectorID := firstNonEmptyBacktestValue(req.DetectorID, "marketops.dsm.taxonomy_v1")
	sourceAdapter := firstNonEmptyBacktestValue(req.SourceAdapter, "market_data.massive")
	configs := []marketopsbacktest.Config{}
	for _, dataset := range datasets {
		for _, symbol := range symbols {
			for _, window := range windows {
				if len(configs) >= maxRuns {
					break
				}
				configs = append(configs, marketopsbacktest.Config{RunID: marketOpsCampaignChildRunID(campaignID, dataset, symbol, window[0]), TenantID: tenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SourceID: strings.TrimSpace(req.SourceID), SourceAdapter: sourceAdapter, Dataset: dataset, DetectorID: detectorID, DetectorVersion: strings.TrimSpace(req.DetectorVersion), RequestedBy: actor, WindowStart: window[0], WindowEnd: window[1], Symbols: []string{symbol}, MaxRecords: maxRecords, BatchSize: batchSize, AutoAcceptConfidence: req.AutoAcceptConfidence, PythonBin: pythonBin})
			}
		}
	}
	if len(configs) == 0 {
		return storage.MarketOpsBacktestCampaignRecord{}, nil, errors.New("campaign produced no child runs")
	}
	now := time.Now().UTC()
	record := storage.MarketOpsBacktestCampaignRecord{CampaignID: strings.TrimSpace(campaignID), TenantID: tenantID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance", SourceID: strings.TrimSpace(req.SourceID), SourceAdapter: sourceAdapter, DetectorID: detectorID, DetectorVersion: strings.TrimSpace(req.DetectorVersion), RequestedBy: firstNonEmptyBacktestValue(actor, "operator-local"), UniverseGroup: universeGroup, DatasetScope: datasets, Symbols: symbols, WindowStart: windowStart.UTC(), WindowEnd: windowEnd.UTC(), WindowStepDays: stepDays, MaxSymbols: maxSymbols, MaxWindows: maxWindows, MaxRuns: maxRuns, MaxRecords: maxRecords, BatchSize: batchSize, Status: storage.RunStatusStarted, ChildRunIDs: []string{}, MetricsJSON: []byte(`{}`), StartedAt: now}
	return record, configs, nil
}

func boundedInt(value int, fallback int, min int, max int, name string) (int, error) {
	if value <= 0 {
		value = fallback
	}
	if value < min || value > max {
		return 0, fmt.Errorf("%s must be between %d and %d", name, min, max)
	}
	return value, nil
}

func marketOpsCampaignWindows(start time.Time, end time.Time, stepDays int, maxWindows int) [][2]time.Time {
	windows := [][2]time.Time{}
	cursor := start.UTC()
	for cursor.Before(end) && len(windows) < maxWindows {
		next := cursor.AddDate(0, 0, stepDays)
		if next.After(end) {
			next = end
		}
		windows = append(windows, [2]time.Time{cursor, next})
		cursor = next
	}
	return windows
}

func marketOpsCampaignChildRunID(campaignID string, dataset string, symbol string, windowStart time.Time) string {
	value := strings.ToLower(strings.TrimSpace(campaignID)) + "-" + sanitizeBacktestIDPart(dataset) + "-" + sanitizeBacktestIDPart(symbol) + "-" + windowStart.UTC().Format("20060102")
	if len(value) > 180 {
		return value[:180]
	}
	return value
}

func sanitizeBacktestIDPart(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "x"
	}
	return out
}

func cleanSymbols(values []string) []string {
	out := []string{}
	for _, value := range values {
		value = strings.ToUpper(strings.TrimSpace(value))
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func readReplayJobRequest(w http.ResponseWriter, r *http.Request) (replayJobCreateRequest, error) {
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, defaultMaxRawEventBytes))
	decoder.DisallowUnknownFields()
	var req replayJobCreateRequest
	if err := decoder.Decode(&req); err != nil {
		return replayJobCreateRequest{}, err
	}
	return req, nil
}

func replayJobRecordFromRequest(req replayJobCreateRequest, actor string, now time.Time) (storage.ReplayJobRecord, error) {
	windowStart, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(req.WindowStart))
	if err != nil {
		return storage.ReplayJobRecord{}, errors.New("window_start must be an RFC3339 timestamp")
	}
	windowEnd, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(req.WindowEnd))
	if err != nil {
		return storage.ReplayJobRecord{}, errors.New("window_end must be an RFC3339 timestamp")
	}
	sourceKind := firstNonEmpty(strings.TrimSpace(req.SourceKind), storage.ReplaySourceRaw)
	replayMode := firstNonEmpty(strings.TrimSpace(req.ReplayMode), storage.ReplayModeOriginal)
	filters := jsonOrDefaultObject(req.Filters)
	options := jsonOrDefaultObject(req.Options)
	record := storage.ReplayJobRecord{
		ReplayJobID: newID("replay"), TenantID: strings.TrimSpace(req.TenantID), SourceID: strings.TrimSpace(req.SourceID),
		Dataset: strings.TrimSpace(req.Dataset), SourceKind: sourceKind, ReplayMode: replayMode, Status: storage.ReplayJobStatusQueued,
		RequestedBy: actor, WindowStart: windowStart.UTC(), WindowEnd: windowEnd.UTC(), FiltersJSON: filters, OptionsJSON: options,
		ResultJSON: []byte("{}"), CreatedAt: now, UpdatedAt: now,
	}
	if strings.TrimSpace(record.TenantID) == "" {
		return storage.ReplayJobRecord{}, errors.New("tenant_id is required")
	}
	if !record.WindowEnd.After(record.WindowStart) {
		return storage.ReplayJobRecord{}, errors.New("window_end must be after window_start")
	}
	if !oneOf(sourceKind, storage.ReplaySourceRaw, storage.ReplaySourceNormalized, storage.ReplaySourceSignals) {
		return storage.ReplayJobRecord{}, errors.New("source_kind must be raw_events, normalized_events, or signals")
	}
	if !oneOf(replayMode, storage.ReplayModeOriginal, storage.ReplayModeLatestCompatible, storage.ReplayModeExplicit) {
		return storage.ReplayJobRecord{}, errors.New("replay_mode must be original, latest_compatible, or explicit")
	}
	return record, nil
}

func replayActor(r *http.Request, requestedBy string) string {
	return firstNonEmpty(strings.TrimSpace(r.Header.Get("X-SignalOps-Actor")), strings.TrimSpace(requestedBy), "operator-local")
}

func jsonOrDefaultObject(raw json.RawMessage) []byte {
	if len(raw) == 0 {
		return []byte("{}")
	}
	return append([]byte(nil), raw...)
}

func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func schedulerRunResponses(records []storage.SchedulerRunRecord) []schedulerRunDTO {
	items := make([]schedulerRunDTO, 0, len(records))
	for _, record := range records {
		items = append(items, schedulerRunResponse(record))
	}
	return items
}

func schedulerRunResponse(record storage.SchedulerRunRecord) schedulerRunDTO {
	return schedulerRunDTO{
		RunID:            record.RunID,
		TenantID:         record.TenantID,
		SourceID:         record.SourceID,
		SourceAdapter:    record.SourceAdapter,
		Datasets:         record.Datasets,
		ObservationDate:  record.ObservationDate,
		DryRun:           record.DryRun,
		Status:           record.Status,
		StartedAt:        record.StartedAt,
		CompletedAt:      record.CompletedAt,
		EventsBuilt:      record.EventsBuilt,
		EventsPublished:  record.EventsPublished,
		ProviderRequests: record.ProviderRequests,
		ProviderRetries:  record.ProviderRetries,
		Failures:         record.Failures,
		Config:           jsonRawOrEmptyObject(record.ConfigJSON),
		Report:           jsonRawOrEmptyObject(record.ReportJSON),
		ErrorMessage:     record.ErrorMessage,
		CreatedAt:        record.CreatedAt,
		UpdatedAt:        record.UpdatedAt,
	}
}

func replayJobResponses(records []storage.ReplayJobRecord) []replayJobDTO {
	items := make([]replayJobDTO, 0, len(records))
	for _, record := range records {
		items = append(items, replayJobResponse(record))
	}
	return items
}

func replayJobResponse(record storage.ReplayJobRecord) replayJobDTO {
	return replayJobDTO{
		ReplayJobID: record.ReplayJobID, TenantID: record.TenantID, SourceID: record.SourceID, Dataset: record.Dataset,
		SourceKind: record.SourceKind, ReplayMode: record.ReplayMode, Status: record.Status, RequestedBy: record.RequestedBy,
		WindowStart: record.WindowStart, WindowEnd: record.WindowEnd, StartedAt: record.StartedAt, CompletedAt: record.CompletedAt,
		Filters: jsonRawOrEmptyObject(record.FiltersJSON), Options: jsonRawOrEmptyObject(record.OptionsJSON), Result: jsonRawOrEmptyObject(record.ResultJSON),
		ErrorMessage: record.ErrorMessage, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt,
	}
}

func replayStatusResponse(now time.Time, counts []storage.ReplayJobStatusCount, workers []storage.ReplayWorkerHeartbeatRecord, latestJobs []storage.ReplayJobRecord) replayStatusDTO {
	jobCounts := map[string]int{
		storage.ReplayJobStatusQueued:    0,
		storage.ReplayJobStatusRunning:   0,
		storage.ReplayJobStatusSucceeded: 0,
		storage.ReplayJobStatusFailed:    0,
		storage.ReplayJobStatusCanceled:  0,
	}
	for _, count := range counts {
		jobCounts[count.Status] = count.Count
	}
	return replayStatusDTO{GeneratedAt: now, JobCounts: jobCounts, Workers: replayWorkerStatusResponses(now, workers), LatestJobs: replayJobResponses(latestJobs)}
}

func replayWorkerStatusResponses(now time.Time, records []storage.ReplayWorkerHeartbeatRecord) []replayWorkerStatusDTO {
	items := make([]replayWorkerStatusDTO, 0, len(records))
	for _, record := range records {
		items = append(items, replayWorkerStatusResponse(now, record))
	}
	return items
}

func replayWorkerStatusResponse(now time.Time, record storage.ReplayWorkerHeartbeatRecord) replayWorkerStatusDTO {
	health := "online"
	if now.Sub(record.LastSeenAt) > 30*time.Second || record.Status == "stopping" {
		health = "stale"
	}
	if record.Status == "error" {
		health = "error"
	}
	return replayWorkerStatusDTO{
		WorkerID: record.WorkerID, Status: record.Status, Health: health, ProcessStartedAt: record.ProcessStartedAt, LastSeenAt: record.LastSeenAt,
		LastClaimedAt: record.LastClaimedAt, LastClaimedReplayJobID: record.LastClaimedReplayJobID, LastCompletedAt: record.LastCompletedAt, LastCompletedReplayJobID: record.LastCompletedReplayJobID,
		LastErrorAt: record.LastErrorAt, LastErrorMessage: record.LastErrorMessage, Metadata: jsonRawOrEmptyObject(record.MetadataJSON), CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt,
	}
}

func providerUsageResponses(records []storage.ProviderUsageRecord) []providerUsageDTO {
	items := make([]providerUsageDTO, 0, len(records))
	for _, record := range records {
		items = append(items, providerUsageDTO{
			UsageID:      record.UsageID,
			RunID:        record.RunID,
			Provider:     record.Provider,
			Dataset:      record.Dataset,
			RequestCount: record.RequestCount,
			RetryCount:   record.RetryCount,
			EventCount:   record.EventCount,
			Budget:       jsonRawOrEmptyObject(record.BudgetJSON),
			CreatedAt:    record.CreatedAt,
		})
	}
	return items
}

func rawEventResponses(records []storage.RawEventLedgerRecord) []rawEventDTO {
	items := make([]rawEventDTO, 0, len(records))
	for _, record := range records {
		items = append(items, rawEventResponse(record))
	}
	return items
}

func rawEventResponse(record storage.RawEventLedgerRecord) rawEventDTO {
	return rawEventDTO{
		EventID:         record.EventID,
		AppID:           record.AppID,
		Domain:          record.Domain,
		UseCase:         record.UseCase,
		TenantID:        record.TenantID,
		SourceID:        record.SourceID,
		SourceAdapter:   record.SourceAdapter,
		Dataset:         record.Dataset,
		IdempotencyKey:  record.IdempotencyKey,
		ObservationTime: record.ObservationTime,
		ProcessingTime:  record.ProcessingTime,
		BrokerTopic:     record.BrokerTopic,
		BrokerPartition: record.BrokerPartition,
		BrokerOffset:    record.BrokerOffset,
		Payload:         jsonRawOrEmptyObject(record.PayloadJSON),
		EntityHints:     jsonRawOrEmptyArray(record.EntityHintsJSON),
		CreatedAt:       record.CreatedAt,
	}
}

func normalizedEventResponses(records []storage.NormalizedEventLedgerRecord) []normalizedEventDTO {
	items := make([]normalizedEventDTO, 0, len(records))
	for _, record := range records {
		items = append(items, normalizedEventResponse(record))
	}
	return items
}

func normalizedEventResponse(record storage.NormalizedEventLedgerRecord) normalizedEventDTO {
	return normalizedEventDTO{EventID: record.EventID, TenantID: record.TenantID, AppID: record.AppID, Domain: record.Domain, UseCase: record.UseCase, SourceID: record.SourceID,
		SourceAdapter: record.SourceAdapter, Dataset: record.Dataset, IdempotencyKey: record.IdempotencyKey,
		SchemaID: record.SchemaID, SchemaVersion: record.SchemaVersion, ObservationTime: record.ObservationTime,
		ProcessingTime: record.ProcessingTime, Confidence: record.Confidence, RawTopic: record.RawTopic,
		RawPartition: record.RawPartition, RawOffset: record.RawOffset, NormalizedTopic: record.NormalizedTopic,
		NormalizedPartition: record.NormalizedPartition, NormalizedOffset: record.NormalizedOffset,
		NormalizedPayload: jsonRawOrEmptyObject(record.NormalizedPayload), Entities: jsonRawOrEmptyArray(record.EntitiesJSON),
		Evidence: jsonRawOrEmptyArray(record.EvidenceJSON), Metadata: jsonRawOrEmptyObject(record.MetadataJSON),
		Event: jsonRawOrEmptyObject(record.EventJSON), CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}

func signalResponses(records []storage.SignalLedgerRecord) []signalDTO {
	items := make([]signalDTO, 0, len(records))
	for _, record := range records {
		items = append(items, signalResponse(record))
	}
	return items
}

func signalResponse(record storage.SignalLedgerRecord) signalDTO {
	recommendation := json.RawMessage(record.RecommendationJSON)
	if len(recommendation) == 0 {
		recommendation = json.RawMessage("null")
	}
	return signalDTO{SignalID: record.SignalID, TenantID: record.TenantID, AppID: record.AppID, Domain: record.Domain, UseCase: record.UseCase, SourceID: record.SourceID,
		SourceDomain: record.SourceDomain, SourceAdapter: record.SourceAdapter, IngestionMode: record.IngestionMode,
		Dataset: record.Dataset, EventIDs: record.EventIDs, ArtifactIDs: record.ArtifactIDs, SignalType: record.SignalType,
		DetectorID: record.DetectorID, DetectorVersion: record.DetectorVersion, ModelVersion: record.ModelVersion,
		SignalTime: record.SignalTime, ObservationTime: record.ObservationTime, EffectiveTime: record.EffectiveTime,
		ProcessingTime: record.ProcessingTime, WindowStart: record.WindowStart, WindowEnd: record.WindowEnd,
		Confidence: record.Confidence, Severity: record.Severity, Entities: jsonRawOrEmptyArray(record.EntitiesJSON),
		SupportingMetrics: jsonRawOrEmptyObject(record.SupportingMetrics), GraphTargets: jsonRawOrEmptyArray(record.GraphTargetsJSON),
		SemanticEvidence: jsonRawOrEmptyArray(record.SemanticEvidenceJSON), Evidence: jsonRawOrEmptyArray(record.EvidenceJSON),
		Recommendation: recommendation, CorrelationID: record.CorrelationID, TraceID: record.TraceID,
		CausationID: record.CausationID, ReplayJobID: record.ReplayJobID, BrokerTopic: record.BrokerTopic,
		BrokerPartition: record.BrokerPartition, BrokerOffset: record.BrokerOffset,
		Event: jsonRawOrEmptyObject(record.EventJSON), CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}

func alertResponses(records []storage.AlertLedgerRecord) []alertDTO {
	items := make([]alertDTO, 0, len(records))
	for _, record := range records {
		items = append(items, alertResponse(record))
	}
	return items
}

func alertResponse(record storage.AlertLedgerRecord) alertDTO {
	recommendation := json.RawMessage(record.RecommendationJSON)
	if len(recommendation) == 0 {
		recommendation = json.RawMessage("null")
	}
	return alertDTO{AlertID: record.AlertID, TenantID: record.TenantID, AppID: record.AppID, Domain: record.Domain, UseCase: record.UseCase, SourceID: record.SourceID,
		SourceDomain: record.SourceDomain, SourceAdapter: record.SourceAdapter, Dataset: record.Dataset,
		SignalID: record.SignalID, DetectorID: record.DetectorID, AlertType: record.AlertType,
		Severity: record.Severity, Status: record.Status, Title: record.Title, Summary: record.Summary,
		Confidence: record.Confidence, EventIDs: record.EventIDs, Entities: jsonRawOrEmptyArray(record.EntitiesJSON),
		Evidence: jsonRawOrEmptyArray(record.EvidenceJSON), Recommendation: recommendation,
		CorrelationID: record.CorrelationID, FirstObservedAt: record.FirstObservedAt, LastObservedAt: record.LastObservedAt,
		AcknowledgedAt: record.AcknowledgedAt, AcknowledgedBy: record.AcknowledgedBy, ResolvedAt: record.ResolvedAt,
		ResolvedBy: record.ResolvedBy, Metadata: jsonRawOrEmptyObject(record.MetadataJSON), CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt}
}

func insightResponses(records []storage.InsightLedgerRecord) []insightDTO {
	items := make([]insightDTO, 0, len(records))
	for _, record := range records {
		items = append(items, insightResponse(record))
	}
	return items
}

func insightResponse(record storage.InsightLedgerRecord) insightDTO {
	recommendation := json.RawMessage(record.RecommendationJSON)
	if len(recommendation) == 0 {
		recommendation = json.RawMessage("null")
	}
	return insightDTO{InsightID: record.InsightID, TenantID: record.TenantID, AppID: record.AppID, Domain: record.Domain, UseCase: record.UseCase, SourceID: record.SourceID,
		SourceDomain: record.SourceDomain, SourceAdapter: record.SourceAdapter, Dataset: record.Dataset,
		SignalID: record.SignalID, DetectorID: record.DetectorID, InsightType: record.InsightType,
		Status: record.Status, Title: record.Title, Summary: record.Summary, Confidence: record.Confidence,
		Severity: record.Severity, EventIDs: record.EventIDs, Entities: jsonRawOrEmptyArray(record.EntitiesJSON),
		SupportingMetrics: jsonRawOrEmptyObject(record.SupportingMetrics), SemanticEvidence: jsonRawOrEmptyArray(record.SemanticEvidenceJSON),
		Recommendation: recommendation, CorrelationID: record.CorrelationID, ObservedAt: record.ObservedAt,
		ReviewedAt: record.ReviewedAt, ReviewedBy: record.ReviewedBy, Metadata: jsonRawOrEmptyObject(record.MetadataJSON),
		CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}

func marketOpsAssetResponses(records []storage.MarketOpsAssetRecord) []marketOpsAssetDTO {
	items := make([]marketOpsAssetDTO, 0, len(records))
	for _, record := range records {
		items = append(items, marketOpsAssetDTO{
			TenantID:      record.TenantID,
			AppID:         record.AppID,
			Domain:        record.Domain,
			UseCase:       record.UseCase,
			SourceID:      record.SourceID,
			UniverseGroup: record.UniverseGroup,
			Rank:          record.Rank,
			Ticker:        record.Ticker,
			TickerKey:     record.TickerKey,
			Company:       record.Company,
			CompanyKey:    record.CompanyKey,
			AssetType:     record.AssetType,
			Exchange:      record.Exchange,
			Sector:        record.Sector,
			SectorKey:     record.SectorKey,
			Industry:      record.Industry,
			IndustryKey:   record.IndustryKey,
			IsActive:      record.IsActive,
			Metadata:      jsonRawOrEmptyObject(record.MetadataJSON),
			CreatedAt:     record.CreatedAt,
			UpdatedAt:     record.UpdatedAt,
		})
	}
	return items
}

func catalogSourceResponses(records []storage.CatalogSourceRecord) []catalogSourceDTO {
	items := make([]catalogSourceDTO, 0, len(records))
	for _, record := range records {
		items = append(items, catalogSourceDTO{
			TenantID:       record.TenantID,
			SourceID:       record.SourceID,
			SourceDomain:   record.SourceDomain,
			SourceAdapter:  record.SourceAdapter,
			DisplayName:    record.DisplayName,
			Description:    record.Description,
			Status:         record.Status,
			IngestionModes: record.IngestionModes,
			Datasets:       record.Datasets,
			Metadata:       jsonRawOrEmptyObject(record.MetadataJSON),
			CreatedAt:      record.CreatedAt,
			UpdatedAt:      record.UpdatedAt,
		})
	}
	return items
}

func catalogPipelineResponses(records []storage.CatalogPipelineRecord) []catalogPipelineDTO {
	items := make([]catalogPipelineDTO, 0, len(records))
	for _, record := range records {
		items = append(items, catalogPipelineDTO{
			TenantID:      record.TenantID,
			PipelineID:    record.PipelineID,
			SourceID:      record.SourceID,
			SourceDomain:  record.SourceDomain,
			PipelineName:  record.PipelineName,
			Description:   record.Description,
			Status:        record.Status,
			Stages:        record.Stages,
			InputDatasets: record.InputDatasets,
			OutputTopics:  record.OutputTopics,
			Metadata:      jsonRawOrEmptyObject(record.MetadataJSON),
			CreatedAt:     record.CreatedAt,
			UpdatedAt:     record.UpdatedAt,
		})
	}
	return items
}

func catalogRuleResponses(records []storage.CatalogRuleRecord) []catalogRuleDTO {
	items := make([]catalogRuleDTO, 0, len(records))
	for _, record := range records {
		items = append(items, catalogRuleDTO{
			TenantID:     record.TenantID,
			RuleID:       record.RuleID,
			RuleName:     record.RuleName,
			Description:  record.Description,
			RuleType:     record.RuleType,
			Severity:     record.Severity,
			Status:       record.Status,
			Version:      record.Version,
			SourceID:     record.SourceID,
			PipelineID:   record.PipelineID,
			DatasetScope: record.DatasetScope,
			EntityScope:  record.EntityScope,
			Expression:   jsonRawOrEmptyObject(record.ExpressionJSON),
			Actions:      record.Actions,
			Metadata:     jsonRawOrEmptyObject(record.MetadataJSON),
			CreatedAt:    record.CreatedAt,
			UpdatedAt:    record.UpdatedAt,
		})
	}
	return items
}

func idempotencyResponse(record storage.IdempotencyRecord) idempotencyDTO {
	return idempotencyDTO{
		TenantID:       record.TenantID,
		SourceID:       record.SourceID,
		IdempotencyKey: record.IdempotencyKey,
		EventID:        record.EventID,
		SourceAdapter:  record.SourceAdapter,
		Dataset:        record.Dataset,
		Topic:          record.Topic,
		Partition:      record.Partition,
		Offset:         record.Offset,
		PayloadHash:    record.PayloadHash,
		Status:         record.Status,
		Metadata:       jsonRawOrEmptyObject(record.MetadataJSON),
		FirstSeenAt:    record.FirstSeenAt,
		LastSeenAt:     record.LastSeenAt,
	}
}

func jsonRawOrEmptyObject(value []byte) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(value)
}

func jsonRawOrEmptyArray(value []byte) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(`[]`)
	}
	return json.RawMessage(value)
}

func requireQueryRepository(w http.ResponseWriter, repo storage.QueryRepository) (storage.QueryRepository, bool) {
	if repo == nil {
		writeError(w, http.StatusServiceUnavailable, "storage_unavailable", "query storage is not configured")
		return nil, false
	}
	return repo, true
}

func writeQueryError(w http.ResponseWriter, err error, notFoundCode string, notFoundMessage string) {
	if errors.Is(err, storage.ErrNotFound) {
		writeError(w, http.StatusNotFound, notFoundCode, notFoundMessage)
		return
	}
	writeError(w, http.StatusInternalServerError, "query_failed", "query failed")
}

func queryLimit(r *http.Request, fallback int) int {
	value := strings.TrimSpace(r.URL.Query().Get("limit"))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	if parsed > 200 {
		return 200
	}
	return parsed
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{
		"error":   code,
		"message": message,
	})
}

func readLifecycleMutationRequest(w http.ResponseWriter, r *http.Request) (lifecycleMutationRequest, error) {
	if r.Body == nil || r.ContentLength == 0 {
		return lifecycleMutationRequest{}, nil
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
	if err != nil {
		return lifecycleMutationRequest{}, errors.New("request body exceeds 65536 bytes or cannot be read")
	}
	defer r.Body.Close()
	if len(strings.TrimSpace(string(body))) == 0 {
		return lifecycleMutationRequest{}, nil
	}
	var req lifecycleMutationRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return lifecycleMutationRequest{}, errors.New("request body must be a valid JSON object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil || fields == nil {
		return lifecycleMutationRequest{}, errors.New("request body must be a JSON object")
	}
	return req, nil
}

func lifecycleActor(r *http.Request, bodyActor string) string {
	if principal, ok := principalFromContext(r.Context()); ok {
		return principal.Actor
	}
	if actor := strings.TrimSpace(r.Header.Get("X-SignalOps-Actor")); actor != "" {
		return actor
	}
	if actor := strings.TrimSpace(bodyActor); actor != "" {
		return actor
	}
	return "operator-local"
}

func lifecycleMetadata(action string, actor string, req lifecycleMutationRequest) ([]byte, error) {
	entry := map[string]any{
		"action":     action,
		"actor":      actor,
		"mutated_at": time.Now().UTC().Format(time.RFC3339Nano),
	}
	if note := strings.TrimSpace(req.Note); note != "" {
		entry["note"] = note
	}
	if reason := strings.TrimSpace(req.Reason); reason != "" {
		entry["reason"] = reason
	}
	return json.Marshal(map[string]any{"lifecycle": entry})
}

func readJSONObject(w http.ResponseWriter, r *http.Request, maxBytes int64) ([]byte, map[string]json.RawMessage, error) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBytes))
	if err != nil {
		return nil, nil, fmt.Errorf("request body exceeds %d bytes or cannot be read", maxBytes)
	}
	defer r.Body.Close()

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil, nil, errors.New("request body must be a valid JSON object")
	}
	if fields == nil {
		return nil, nil, errors.New("request body must be a JSON object")
	}

	return body, fields, nil
}

func rawEventHeaders(eventID, idempotencyKey string) map[string]string {
	return map[string]string{
		"content_type":            "application/json",
		"signalops_event_id":      eventID,
		"signalops_idempotency":   idempotencyKey,
		"signalops_ingest_route":  "/v1/events/raw",
		"signalops_ingest_format": "raw_signal_event.v1",
	}
}

func jsonStringField(fields map[string]json.RawMessage, key string) string {
	value, ok := fields[key]
	if !ok {
		return ""
	}

	var decoded string
	if err := json.Unmarshal(value, &decoded); err != nil {
		return ""
	}
	return strings.TrimSpace(decoded)
}

func headerValue(r *http.Request, key string) string {
	return strings.TrimSpace(r.Header.Get(key))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func newID(prefix string) string {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UTC().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(buf[:])
}
