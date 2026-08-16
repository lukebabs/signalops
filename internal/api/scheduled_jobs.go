package api

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type scheduledJobDefinition struct{ ID, Label, Schedule, Timezone string }

func scheduledJobStatuses() []map[string]any {
	jobs := []scheduledJobDefinition{
		{"marketops-daily-postclose", "MarketOps post-close", "Weekdays 18:01:55", "America/New_York"},
		{"marketops-sri-refresh", "MarketOps SRI refresh", "Weekdays 20:07", "America/New_York"},
		{"marketops-sri-holdings-refresh", "MarketOps SRI issuer holdings", "Weekdays 20:20", "America/New_York"},
		{"marketops-intraday", "MarketOps intraday monitor", "Weekdays every 15 minutes, 09:30–20:00", "America/New_York"},
		{"marketops-fmp-continuation", "FMP continuation", "Saturday 02:00", "America/New_York"},
		{"marketops-fmp-annual-financial", "FMP annual financial capture", "Saturday 02:30", "America/New_York"},
		{"marketops-task-retry", "MarketOps governed task retry", "Weekdays every 15 minutes, 18:30–23:00", "America/New_York"},
		{"marketops-postclose-recovery", "MarketOps post-close recovery guard", "Weekdays every 15 minutes, 18:30-23:00", "America/New_York"},
		{"marketops-risk-reward", "MarketOps Risk/Reward post-close", "Post-close completion stage", "America/New_York"},
		{"signalops-storage-monitor", "Persistent storage monitor", "Daily 02:00", "America/New_York"},
		{"signalops-retention-governance", "Retention governance (dry run)", "Daily 02:30", "America/New_York"},
	}
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
