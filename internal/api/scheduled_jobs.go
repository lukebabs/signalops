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
	{"marketops-operations-monitor", "MarketOps operations monitor", "Hourly", "UTC", ""},
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

func marketOpsOperationsFreshnessResponses(records []storage.MarketOpsOperationsFreshnessRecord) []map[string]any {
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		row := map[string]any{
			"view_id":        record.ViewID,
			"label":          record.Label,
			"row_count":      record.RowCount,
			"expected_count": record.ExpectedCount,
			"status":         record.Status,
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
	result := map[string]any{
		"generated_at":   now.Format(time.RFC3339),
		"tenant_id":      tenantID,
		"scheduled_jobs": scheduledJobStatuses(ctx, repo),
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
		freshness = marketOpsOperationsFreshnessResponses(records)
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
