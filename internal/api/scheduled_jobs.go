package api

import (
	"context"
	"encoding/json"
	"errors"
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
	{"marketops-daily-postclose", "MarketOps post-close", "Weekdays 18:01:55", "America/New_York", "scheduler-run-now:marketops-daily-postclose"},
	{"marketops-sri-refresh", "MarketOps SRI refresh", "Weekdays 20:07", "America/New_York", "scheduler-run-now:marketops-sri-refresh"},
	{"marketops-sri-holdings-refresh", "MarketOps SRI issuer holdings", "Weekdays 20:20", "America/New_York", "scheduler-run-now:marketops-sri-holdings-refresh"},
	{"marketops-intraday", "MarketOps intraday monitor", "Weekdays every 15 minutes, 09:30-20:00", "America/New_York", "scheduler-run-now:marketops-intraday"},
	{"marketops-fmp-continuation", "FMP continuation", "Saturday 02:00", "America/New_York", "scheduler-run-now:marketops-fmp-continuation"},
	{"marketops-fmp-annual-financial", "FMP annual financial capture", "Saturday 02:30", "America/New_York", "scheduler-run-now:marketops-fmp-annual-financial"},
	{"marketops-task-retry", "MarketOps governed task retry", "Weekdays every 15 minutes, 18:30-23:00", "America/New_York", "scheduler-run-now:marketops-task-retry"},
	{"marketops-postclose-recovery", "MarketOps post-close recovery guard", "Weekdays every 15 minutes, 18:30-23:00", "America/New_York", "scheduler-run-now:marketops-postclose-recovery"},
	{"marketops-risk-reward", "MarketOps Risk/Reward post-close", "Post-close completion stage", "America/New_York", "scheduler-run-now:marketops-risk-reward"},
	{"signalops-storage-monitor", "Persistent storage monitor", "Daily 02:00", "America/New_York", "scheduler-run-now:signalops-storage-monitor"},
	{"signalops-retention-governance", "Retention governance (dry run)", "Daily 02:30", "America/New_York", "scheduler-run-now:signalops-retention-governance"},
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

func scheduledJobStatuses() []map[string]any {
	jobs := scheduledJobDefinitions
	dir := os.Getenv("SIGNALOPS_SCHEDULE_STATUS_DIR")
	if dir == "" {
		dir = "/var/lib/signalops/scheduled-jobs"
	}
	out := make([]map[string]any, 0, len(jobs))
	for _, job := range jobs {
		row := map[string]any{"job_id": job.ID, "label": job.Label, "schedule": job.Schedule, "timezone": job.Timezone, "status": "pending"}
		if raw, err := os.ReadFile(filepath.Join(dir, job.ID+".json")); err == nil {
			var recorded map[string]any
			if json.Unmarshal(raw, &recorded) == nil {
				for k, v := range recorded {
					row[k] = v
				}
			}
		}
		out = append(out, row)
	}
	return out
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

const operationsHealthTaskStaleWindow = 2 * time.Hour

func marketOpsOperationsHealth(ctx context.Context, repo storage.QueryRepository, tenantID string, now time.Time) (map[string]any, error) {
	result := map[string]any{
		"generated_at":   now.Format(time.RFC3339),
		"tenant_id":      tenantID,
		"scheduled_jobs": scheduledJobStatuses(),
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
