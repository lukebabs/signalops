package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/config"
	"github.com/lukebabs/signalops/internal/storage"
	postgres "github.com/lukebabs/signalops/internal/storage/postgres"
)

var governedSuccessJobs = map[string]bool{
	"marketops-daily-postclose":      true,
	"marketops-fmp-continuation":     true,
	"marketops-fmp-annual-financial": true,
	"signalops-storage-monitor":      true,
	"signalops-retention-governance": true,
}

type input struct {
	TenantID, JobID, Status, Schedule, Timezone, StartedAt, CompletedAt string
	ExitCode                                                            int
}

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	var in input
	flag.StringVar(&in.TenantID, "tenant-id", "tenant-local", "tenant owning the scheduled job")
	flag.StringVar(&in.JobID, "job-id", "", "scheduled job id")
	flag.StringVar(&in.Status, "status", "", "succeeded or failed")
	flag.StringVar(&in.Schedule, "schedule", "", "human schedule")
	flag.StringVar(&in.Timezone, "timezone", "America/New_York", "schedule timezone")
	flag.StringVar(&in.StartedAt, "started-at", "", "RFC3339 start time")
	flag.StringVar(&in.CompletedAt, "completed-at", "", "RFC3339 completion time")
	flag.IntVar(&in.ExitCode, "exit-code", 0, "scheduled command exit code")
	flag.Parse()
	in.JobID = strings.TrimSpace(in.JobID)
	in.Status = strings.ToLower(strings.TrimSpace(in.Status))
	if in.JobID == "" || (in.Status != "succeeded" && in.Status != "failed") {
		return fmt.Errorf("job-id and status (succeeded or failed) are required")
	}
	if in.Status == "succeeded" && !governedSuccessJobs[in.JobID] {
		return nil
	}
	cfg := config.Load()
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		return fmt.Errorf("SIGNALOPS_DATABASE_URL is required")
	}
	repo, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	now := time.Now().UTC()
	started, completed := parseTime(in.StartedAt, now), parseTime(in.CompletedAt, now)
	category, severity, title, summary := notificationContent(in)
	evidence, _ := json.Marshal(map[string]any{
		"job_id": in.JobID, "status": in.Status, "schedule": in.Schedule, "timezone": in.Timezone,
		"started_at": started, "completed_at": completed, "exit_code": in.ExitCode,
	})
	_, err = repo.UpsertAdministrationNotification(ctx, storage.AdministrationNotificationRecord{
		NotificationID: newID(), TenantID: in.TenantID, Source: "scheduler", Category: category,
		Severity: severity, Title: title, Summary: summary, DedupeKey: "scheduler:" + in.JobID + ":" + in.Status,
		FirstOccurredAt: completed, LastOccurredAt: completed, EvidenceJSON: evidence,
	})
	return err
}

func notificationContent(in input) (category, severity, title, summary string) {
	label := strings.ReplaceAll(in.JobID, "-", " ")
	if in.Status == "failed" {
		return "routine_job_failure", "warning", "Scheduled job failed: " + label,
			fmt.Sprintf("%s exited with code %d at %s. Repeated failures are consolidated and escalate after the third occurrence.", in.JobID, in.ExitCode, in.CompletedAt)
	}
	return "governed_job_success", "info", "Scheduled job completed: " + label,
		fmt.Sprintf("%s completed successfully at %s.", in.JobID, in.CompletedAt)
}

func parseTime(raw string, fallback time.Time) time.Time {
	if value, err := time.Parse(time.RFC3339, raw); err == nil {
		return value.UTC()
	}
	return fallback
}

func newID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return fmt.Sprintf("notification-%d", time.Now().UTC().UnixNano())
	}
	return "notification-" + hex.EncodeToString(value[:])
}
