package api

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestScheduledJobRunActionIsAllowlisted(t *testing.T) {
	cases := map[string]string{
		"marketops-intraday":             "scheduler-run-now:marketops-intraday",
		"marketops-risk-reward":          "scheduler-run-now:marketops-risk-reward",
		"signalops-retention-governance": "scheduler-run-now:signalops-retention-governance",
	}
	for jobID, expected := range cases {
		action, ok := scheduledJobRunAction(jobID)
		if !ok {
			t.Fatalf("scheduled job %q is not allowlisted", jobID)
		}
		if action != expected {
			t.Fatalf("scheduled job %q action = %q, want %q", jobID, action, expected)
		}
	}
	if action, ok := scheduledJobRunAction("arbitrary-host-command"); ok || action != "" {
		t.Fatalf("unexpected action for unsupported job: %q", action)
	}
}

func TestTriggerScheduledJobRunNowViaSocket(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "runner.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/run-now" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var req scheduledJobRunnerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.Action != "scheduler-run-now:signalops-storage-monitor" {
			t.Fatalf("unexpected action: %s", req.Action)
		}
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(scheduledJobRunnerResponse{Status: "accepted", Output: "started"})
	})}
	defer server.Close()
	go func() { _ = server.Serve(listener) }()

	result, err := triggerScheduledJobRunNowViaSocket(context.Background(), "signalops-storage-monitor", "scheduler-run-now:signalops-storage-monitor", socketPath, time.Date(2026, 8, 19, 2, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if result.Runner != "unix:"+socketPath || result.Output != "started" || result.Status != "accepted" {
		t.Fatalf("unexpected result: %#v", result)
	}
}
