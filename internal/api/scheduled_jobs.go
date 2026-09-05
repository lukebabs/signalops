package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

type scheduledJobDefinition struct {
	ID, Label, Schedule, Timezone string
	RunAction                     string
}

var scheduledJobDefinitions = []scheduledJobDefinition{
	{"marketops-warm-eod", "MarketOps warm EOD baseline", "Weekdays 18:00", "America/New_York", "scheduler-run-now:marketops-warm-eod"},
	{"marketops-daily-postclose", "MarketOps post-close", "Weekdays 18:01:55", "America/New_York", "scheduler-run-now:marketops-daily-postclose"},
	{"marketops-sri-refresh", "MarketOps SRI refresh", "Weekdays 20:07", "America/New_York", "scheduler-run-now:marketops-sri-refresh"},
	{"marketops-sri-holdings-refresh", "MarketOps SRI issuer holdings", "Weekdays 20:20", "America/New_York", "scheduler-run-now:marketops-sri-holdings-refresh"},
	{"marketops-intraday", "MarketOps intraday monitor", "Weekdays every 15 minutes, 09:30-20:00", "America/New_York", "scheduler-run-now:marketops-intraday"},
	{"marketops-fmp-continuation", "FMP continuation", "Saturday 02:00", "America/New_York", "scheduler-run-now:marketops-fmp-continuation"},
	{"marketops-fmp-annual-financial", "FMP annual financial capture", "Saturday 02:30", "America/New_York", "scheduler-run-now:marketops-fmp-annual-financial"},
	{"marketops-task-retry", "MarketOps governed task retry", "Weekdays every 15 minutes, 18:30-23:00", "America/New_York", "scheduler-run-now:marketops-task-retry"},
	{"marketops-postclose-recovery", "MarketOps post-close recovery guard", "Weekdays every 15 minutes, 18:30-23:00", "America/New_York", "scheduler-run-now:marketops-postclose-recovery"},
	{"marketops-risk-reward", "MarketOps Risk/Reward post-close", "Post-close completion stage", "America/New_York", "scheduler-run-now:marketops-risk-reward"},
	{"marketops-operations-monitor", "MarketOps operations monitor", "Hourly", "UTC", "scheduler-run-now:marketops-operations-monitor"},
	{"signalops-storage-monitor", "Persistent storage monitor", "Daily 02:00", "America/New_York", "scheduler-run-now:signalops-storage-monitor"},
	{"signalops-retention-governance", "Retention governance (dry run)", "Daily 02:30", "America/New_York", "scheduler-run-now:signalops-retention-governance"},
	{"marketops-retention-governance", "MarketOps subscriber activity retention (dry run)", "Manual dry run", "America/New_York", "scheduler-run-now:marketops-retention-governance"},
}

var errUnsupportedScheduledJob = errors.New("unsupported scheduled job")

type scheduledJobRunNowResult struct {
	JobID     string `json:"job_id"`
	Status    string `json:"status"`
	Action    string `json:"action"`
	Runner    string `json:"runner"`
	StartedAt string `json:"started_at"`
	Output    string `json:"output,omitempty"`
}

type scheduledJobRunnerRequest struct {
	Action string `json:"action"`
}

type scheduledJobRunnerResponse struct {
	Status string `json:"status"`
	Output string `json:"output"`
}

func scheduledJobStatuses(ctx context.Context, repo any) []map[string]any {
	jobs := scheduledJobDefinitions
	byID := make(map[string]map[string]any, len(jobs))
	out := make([]map[string]any, 0, len(jobs))
	for _, job := range jobs {
		row := map[string]any{
			"job_id":          job.ID,
			"label":           job.Label,
			"schedule":        job.Schedule,
			"timezone":        job.Timezone,
			"status":          "pending",
			"run_now_enabled": job.RunAction != "",
		}
		byID[job.ID] = row
		out = append(out, row)
	}

	if reader, ok := repo.(storage.MarketOpsScheduledJobStatusRepository); ok && reader != nil {
		if records, err := reader.ListMarketOpsScheduledJobStatuses(ctx); err == nil {
			for _, record := range records {
				if row, ok := byID[record.JobID]; ok {
					mergeScheduledJobDatabaseRecord(row, record)
				}
			}
			return out
		}
	}

	mergeScheduledJobFileStatuses(out, byID)
	return out
}

func mergeScheduledJobDatabaseRecord(row map[string]any, record storage.MarketOpsScheduledJobStatusRecord) {
	row["schedule"] = record.Schedule
	row["timezone"] = record.Timezone
	row["status"] = record.Status
	row["status_source"] = "database"
	if record.Reason != "" {
		row["reason"] = record.Reason
	}
	if record.StartedAt != nil {
		row["started_at"] = record.StartedAt.UTC().Format(time.RFC3339)
	}
	if record.CompletedAt != nil {
		row["completed_at"] = record.CompletedAt.UTC().Format(time.RFC3339)
	}
	if record.ExitCode != nil {
		row["exit_code"] = *record.ExitCode
	}
	if record.Runner != "" {
		row["runner"] = record.Runner
	}
	if !record.UpdatedAt.IsZero() {
		row["updated_at"] = record.UpdatedAt.UTC().Format(time.RFC3339)
	}
	if len(record.DetailJSON) > 0 {
		var detail map[string]any
		if json.Unmarshal(record.DetailJSON, &detail) == nil {
			row["detail"] = detail
			for key, value := range detail {
				if _, exists := row[key]; !exists {
					row[key] = value
				}
			}
		}
	}
}

func mergeScheduledJobFileStatuses(out []map[string]any, byID map[string]map[string]any) {
	dir := os.Getenv("SIGNALOPS_SCHEDULE_STATUS_DIR")
	if dir == "" {
		dir = "/var/lib/signalops/scheduled-jobs"
	}
	for _, row := range out {
		jobID, _ := row["job_id"].(string)
		if raw, err := os.ReadFile(filepath.Join(dir, jobID+".json")); err == nil {
			var recorded map[string]any
			if json.Unmarshal(raw, &recorded) == nil {
				for k, v := range recorded {
					row[k] = v
				}
				row["status_source"] = "file"
			}
		}
	}
	_ = byID
}

func scheduledJobRunAction(jobID string) (string, bool) {
	jobID = strings.TrimSpace(jobID)
	for _, job := range scheduledJobDefinitions {
		if job.ID == jobID && job.RunAction != "" {
			return job.RunAction, true
		}
	}
	return "", false
}

func triggerScheduledJobRunNow(ctx context.Context, jobID string, now time.Time) (scheduledJobRunNowResult, error) {
	action, ok := scheduledJobRunAction(jobID)
	if !ok {
		return scheduledJobRunNowResult{}, errUnsupportedScheduledJob
	}
	if socketPath := strings.TrimSpace(os.Getenv("SIGNALOPS_SCHEDULE_RUNNER_SOCKET")); socketPath != "" {
		return triggerScheduledJobRunNowViaSocket(ctx, jobID, action, socketPath, now)
	}
	runner := strings.TrimSpace(os.Getenv("SIGNALOPS_SCHEDULE_RUNNER_BIN"))
	if runner == "" {
		runner = "/usr/local/sbin/signalops-deploy-agent"
	}
	cmd := exec.CommandContext(ctx, runner, action)
	output, err := cmd.CombinedOutput()
	result := scheduledJobRunNowResult{
		JobID:     jobID,
		Status:    "accepted",
		Action:    action,
		Runner:    runner,
		StartedAt: now.UTC().Format(time.RFC3339),
		Output:    strings.TrimSpace(string(output)),
	}
	if err != nil {
		if result.Output == "" {
			result.Output = err.Error()
		}
		return result, err
	}
	return result, nil
}

func triggerScheduledJobRunNowViaSocket(ctx context.Context, jobID, action, socketPath string, now time.Time) (scheduledJobRunNowResult, error) {
	payload, err := json.Marshal(scheduledJobRunnerRequest{Action: action})
	if err != nil {
		return scheduledJobRunNowResult{}, err
	}
	dialer := &net.Dialer{}
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return dialer.DialContext(ctx, "unix", socketPath)
			},
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://signalops-deployment-agent/run-now", bytes.NewReader(payload))
	if err != nil {
		return scheduledJobRunNowResult{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	result := scheduledJobRunNowResult{
		JobID:     jobID,
		Status:    "accepted",
		Action:    action,
		Runner:    "unix:" + socketPath,
		StartedAt: now.UTC().Format(time.RFC3339),
	}
	if err != nil {
		result.Output = err.Error()
		return result, err
	}
	defer resp.Body.Close()
	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if readErr != nil {
		result.Output = readErr.Error()
		return result, readErr
	}
	var runnerResponse scheduledJobRunnerResponse
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &runnerResponse)
	}
	if runnerResponse.Status != "" {
		result.Status = runnerResponse.Status
	}
	result.Output = strings.TrimSpace(runnerResponse.Output)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if result.Output == "" {
			result.Output = strings.TrimSpace(string(raw))
		}
		return result, fmt.Errorf("scheduled job runner returned HTTP %d", resp.StatusCode)
	}
	return result, nil
}

type marketOpsOperationsFreshnessContract struct {
	ExpectedFreshness string
	DependencyJobID   string
	RunNowJobID       string
	ActionLabel       string
	NextStepCurrent   string
	NextStepStale     string
}

func marketOpsOperationsFreshnessContractFor(record storage.MarketOpsOperationsFreshnessRecord) marketOpsOperationsFreshnessContract {
	switch strings.TrimSpace(record.ViewID) {
	case "dashboard":
		return marketOpsOperationsFreshnessContract{
			ExpectedFreshness: "After each completed trading day once post-close evidence is aligned.",
			DependencyJobID:   "marketops-daily-postclose",
			RunNowJobID:       "marketops-postclose-recovery",
			ActionLabel:       "Run recovery",
			NextStepCurrent:   "No action required. Dashboard evidence is aligned with the completed-session contract.",
			NextStepStale:     "Run the post-close recovery guard, then recheck Dashboard, Market State, Risk/Reward, and Signal Assurance.",
		}
	case "assets", "assets_coverage":
		return marketOpsOperationsFreshnessContract{
			ExpectedFreshness: "After each completed trading day for the selected/default watchlist coverage contract.",
			DependencyJobID:   "marketops-daily-postclose",
			RunNowJobID:       "marketops-postclose-recovery",
			ActionLabel:       "Run recovery",
			NextStepCurrent:   "No action required. Selected/default watchlist coverage has completed-session evidence.",
			NextStepStale:     "Run the post-close recovery guard and inspect incomplete MarketOps tasks for missed coverage activation.",
		}
	case "market_state":
		return marketOpsOperationsFreshnessContract{
			ExpectedFreshness: "After each completed trading day for every legacy-default asset.",
			DependencyJobID:   "marketops-daily-postclose",
			RunNowJobID:       "marketops-postclose-recovery",
			ActionLabel:       "Run recovery",
			NextStepCurrent:   "No action required. Market State is aligned with completed-session evidence.",
			NextStepStale:     "Run the post-close recovery guard and confirm Market State materialization completed.",
		}
	case "risk_reward":
		return marketOpsOperationsFreshnessContract{
			ExpectedFreshness: "After each completed trading day after EOD baseline and Market State are available.",
			DependencyJobID:   "marketops-risk-reward",
			RunNowJobID:       "marketops-risk-reward",
			ActionLabel:       "Run Risk/Reward",
			NextStepCurrent:   "No action required. Risk/Reward breadth is current for the completed session.",
			NextStepStale:     "Run the Risk/Reward post-close stage and verify breadth aligns to the legacy-default cohort.",
		}
	case "sri", "sector_rotation":
		return marketOpsOperationsFreshnessContract{
			ExpectedFreshness: "Weekday evening after issuer ETF evidence and sector ranking refresh complete.",
			DependencyJobID:   "marketops-sri-refresh",
			RunNowJobID:       "marketops-sri-refresh",
			ActionLabel:       "Run SRI",
			NextStepCurrent:   "No action required. Sector Rotation Intelligence has current ranking evidence.",
			NextStepStale:     "Run SRI refresh, then run issuer-holdings refresh if holdings coverage is also stale.",
		}
	case "signal_assurance":
		return marketOpsOperationsFreshnessContract{
			ExpectedFreshness: "After completed-session signal outcomes are projected for the legacy-default viability cohort.",
			DependencyJobID:   "marketops-daily-postclose",
			RunNowJobID:       "",
			ActionLabel:       "Status only",
			NextStepCurrent:   "No action required. Signal Assurance rows are aligned with current completed-session evidence.",
			NextStepStale:     "Use the constrained SAF projection refresh deployment-agent action under named approval; this action is intentionally not exposed as generic run-now.",
		}
	case "intraday", "intraday_conditions":
		return marketOpsOperationsFreshnessContract{
			ExpectedFreshness: "Every 15 minutes during the active MarketOps intraday window for hot watchlist assets.",
			DependencyJobID:   "marketops-intraday",
			RunNowJobID:       "marketops-intraday",
			ActionLabel:       "Run intraday",
			NextStepCurrent:   "No action required. Intraday snapshots are within the freshness window.",
			NextStepStale:     "Run the intraday monitor and verify the dedicated MarketOps scheduler timers remain active.",
		}
	case "fmp_annual", "fmp_annual_financials":
		return marketOpsOperationsFreshnessContract{
			ExpectedFreshness: "Weekly continuation until all eligible warm-catalog assets have annual financial evidence.",
			DependencyJobID:   "marketops-fmp-annual-financial",
			RunNowJobID:       "marketops-fmp-annual-financial",
			ActionLabel:       "Run FMP annual",
			NextStepCurrent:   "No action required. Annual financial capture is complete for the qualified cohort.",
			NextStepStale:     "Run FMP annual financial capture and inspect failed/ineligible symbols before expanding provider normalization.",
		}
	case "syncratic_ask":
		return marketOpsOperationsFreshnessContract{
			ExpectedFreshness: "After daily Syncratic narrative materialization has Ask-backed or deterministic explanation evidence.",
			DependencyJobID:   "marketops-daily-postclose",
			RunNowJobID:       "",
			ActionLabel:       "Status only",
			NextStepCurrent:   "No action required. Syncratic narrative evidence is available for current dashboard explainability.",
			NextStepStale:     "Review Syncratic narrative materialization and AI Gateway validation before exposing stale explanations.",
		}
	default:
		return marketOpsOperationsFreshnessContract{
			ExpectedFreshness: "Defined by the producing MarketOps workflow.",
			NextStepCurrent:   "No action required.",
			NextStepStale:     "Inspect the producing workflow and scheduled-job ledger for this view.",
		}
	}
}

func scheduledJobRowsByID(rows []map[string]any) map[string]map[string]any {
	byID := make(map[string]map[string]any, len(rows))
	for _, row := range rows {
		if jobID, ok := row["job_id"].(string); ok && jobID != "" {
			byID[jobID] = row
		}
	}
	return byID
}

func rowString(row map[string]any, key string) string {
	if row == nil {
		return ""
	}
	switch value := row[key].(type) {
	case string:
		return value
	case fmt.Stringer:
		return value.String()
	default:
		return ""
	}
}

func rowBool(row map[string]any, key string) bool {
	if row == nil {
		return false
	}
	value, _ := row[key].(bool)
	return value
}

func rowIntString(row map[string]any, key string) string {
	if row == nil {
		return ""
	}
	switch value := row[key].(type) {
	case int:
		return fmt.Sprintf("%d", value)
	case int64:
		return fmt.Sprintf("%d", value)
	case float64:
		return fmt.Sprintf("%.0f", value)
	case string:
		return value
	default:
		return ""
	}
}

func marketOpsFreshnessStatusExplanation(record storage.MarketOpsOperationsFreshnessRecord, contract marketOpsOperationsFreshnessContract, dependency map[string]any) (string, string) {
	status := strings.ToLower(strings.TrimSpace(record.Status))
	reason := strings.TrimSpace(record.Reason)
	jobStatus := rowString(dependency, "status")
	jobReason := rowString(dependency, "reason")
	if jobReason == "" {
		jobReason = rowString(dependency, "error")
	}
	if jobReason == "" {
		jobReason = rowString(dependency, "error_message")
	}
	nextStep := contract.NextStepCurrent
	if status == "stale" || status == "partial" || status == "failed" || status == "missing" || status == "unavailable" || status == "pending" {
		nextStep = contract.NextStepStale
	}
	if nextStep == "" {
		nextStep = "Inspect the producing workflow and scheduled-job ledger for this view."
	}
	if reason != "" {
		return reason, nextStep
	}
	if jobStatus == "failed" {
		if jobReason != "" {
			return "Producing job failed: " + jobReason, nextStep
		}
		return "Producing job failed; inspect the scheduled-job ledger for details.", nextStep
	}
	if status == "current" {
		return "Latest evidence satisfies the configured MarketOps freshness contract.", nextStep
	}
	if status == "partial" {
		return "Freshness evidence exists but coverage is incomplete against the expected cohort.", nextStep
	}
	if status == "stale" {
		return "Latest evidence is outside the configured MarketOps freshness window.", nextStep
	}
	return "Freshness status is governed by the producing MarketOps workflow.", nextStep
}

func marketOpsOperationsFreshnessResponses(records []storage.MarketOpsOperationsFreshnessRecord, jobs []map[string]any) []map[string]any {
	jobsByID := scheduledJobRowsByID(jobs)
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		contract := marketOpsOperationsFreshnessContractFor(record)
		dependency := jobsByID[contract.DependencyJobID]
		runNow := jobsByID[contract.RunNowJobID]
		runNowEnabled := contract.RunNowJobID != "" && rowBool(runNow, "run_now_enabled")
		explanation, nextStep := marketOpsFreshnessStatusExplanation(record, contract, dependency)
		row := map[string]any{
			"view_id":                 record.ViewID,
			"label":                   record.Label,
			"row_count":               record.RowCount,
			"expected_count":          record.ExpectedCount,
			"status":                  record.Status,
			"expected_freshness":      contract.ExpectedFreshness,
			"dependency_job_id":       contract.DependencyJobID,
			"dependency_label":        rowString(dependency, "label"),
			"dependency_status":       rowString(dependency, "status"),
			"dependency_schedule":     rowString(dependency, "schedule"),
			"dependency_timezone":     rowString(dependency, "timezone"),
			"dependency_started_at":   rowString(dependency, "started_at"),
			"dependency_completed_at": rowString(dependency, "completed_at"),
			"dependency_exit_code":    rowIntString(dependency, "exit_code"),
			"run_now_job_id":          contract.RunNowJobID,
			"run_now_enabled":         runNowEnabled,
			"action_label":            contract.ActionLabel,
			"staleness_explanation":   explanation,
			"next_step":               nextStep,
		}
		if record.LatestSessionDate != nil {
			row["latest_session_date"] = record.LatestSessionDate.UTC().Format("2006-01-02")
		}
		if record.LatestAsOf != nil {
			row["latest_as_of"] = record.LatestAsOf.UTC().Format(time.RFC3339)
		}
		if strings.TrimSpace(record.Reason) != "" {
			row["reason"] = record.Reason
		}
		out = append(out, row)
	}
	return out
}

const operationsHealthTaskStaleWindow = 2 * time.Hour

func marketOpsOperationsHealth(ctx context.Context, repo storage.QueryRepository, tenantID string, now time.Time) (map[string]any, error) {
	scheduledJobs := scheduledJobStatuses(ctx, repo)
	result := map[string]any{
		"generated_at":   now.Format(time.RFC3339),
		"tenant_id":      tenantID,
		"scheduled_jobs": scheduledJobs,
	}

	taskSummary := map[string]any{
		"tenant_id":               tenantID,
		"status_counts":           map[string]int{},
		"total_count":             0,
		"incomplete_count":        0,
		"stale_incomplete_count":  0,
		"stale_threshold_minutes": int(operationsHealthTaskStaleWindow.Minutes()),
		"available":               false,
	}

	freshness := []map[string]any{}
	if reader, ok := any(repo).(storage.MarketOpsOperationsFreshnessRepository); ok && reader != nil {
		records, err := reader.ListMarketOpsOperationsFreshness(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		freshness = marketOpsOperationsFreshnessResponses(records, scheduledJobs)
	}
	result["data_freshness"] = freshness

	if reader, ok := any(repo).(storage.MarketOpsTaskRepository); ok {
		tasks, err := reader.ListMarketOpsTaskItems(ctx, storage.MarketOpsTaskItemFilter{TenantID: tenantID, Limit: 200})
		if err != nil {
			return nil, err
		}
		taskSummary["available"] = true
		taskSummary["total_count"] = len(tasks)
		statusCounts := map[string]int{}
		incompleteCount := 0
		staleIncompleteCount := 0
		cutoff := now.Add(-operationsHealthTaskStaleWindow)
		if len(tasks) > 0 {
			taskSummary["latest_session_date"] = tasks[0].SessionDate.Format("2006-01-02")
		}
		var latestTaskUpdate time.Time
		for _, task := range tasks {
			statusCounts[task.Status] = statusCounts[task.Status] + 1
			if task.UpdatedAt.After(latestTaskUpdate) {
				latestTaskUpdate = task.UpdatedAt
			}
			switch strings.ToLower(task.Status) {
			case "succeeded", "failed", "canceled":
				continue
			}
			incompleteCount++
			if task.UpdatedAt.Before(cutoff) {
				staleIncompleteCount++
			}
		}
		if !latestTaskUpdate.IsZero() {
			taskSummary["latest_update"] = latestTaskUpdate.Format(time.RFC3339)
		}
		taskSummary["status_counts"] = statusCounts
		taskSummary["incomplete_count"] = incompleteCount
		taskSummary["stale_incomplete_count"] = staleIncompleteCount
		if len(tasks) == 0 {
			taskSummary["latest_session_date"] = ""
		}
	}
	taskSummary["error"] = ""
	result["marketops_tasks"] = taskSummary

	counts, err := repo.CountReplayJobsByStatus(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	workers, err := repo.ListReplayWorkerHeartbeats(ctx, 20)
	if err != nil {
		return nil, err
	}
	latestJobs, err := repo.ListReplayJobs(ctx, storage.ReplayJobFilter{TenantID: tenantID, Limit: 5})
	if err != nil {
		return nil, err
	}
	result["replay_status"] = replayStatusResponse(now, counts, workers, latestJobs)

	return result, nil
}
