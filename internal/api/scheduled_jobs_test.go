package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestScheduledJobStatusesIncludePostCloseRecoveryStages(t *testing.T) {
	t.Setenv("SIGNALOPS_SCHEDULE_STATUS_DIR", t.TempDir())

	jobs := scheduledJobStatuses()
	byID := make(map[string]map[string]any, len(jobs))
	for _, job := range jobs {
		byID[job["job_id"].(string)] = job
	}

	for _, id := range []string{"marketops-postclose-recovery", "marketops-risk-reward"} {
		if _, ok := byID[id]; !ok {
			t.Fatalf("scheduled job %q is missing", id)
		}
	}
}

func TestScheduledJobStatusesReadRecoveryEvidence(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SIGNALOPS_SCHEDULE_STATUS_DIR", dir)

	recorded := map[string]any{
		"job_id":              "marketops-risk-reward",
		"status":              "succeeded",
		"session_date":        "2026-08-11",
		"risk_reward_results": float64(132),
	}
	raw, err := json.Marshal(recorded)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "marketops-risk-reward.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}

	for _, job := range scheduledJobStatuses() {
		if job["job_id"] != "marketops-risk-reward" {
			continue
		}
		if job["status"] != "succeeded" || job["session_date"] != "2026-08-11" {
			t.Fatalf("recorded recovery evidence was not merged: %#v", job)
		}
		return
	}
	t.Fatal("risk/reward job is missing")
}
