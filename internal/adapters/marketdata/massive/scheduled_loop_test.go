package massive

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRunScheduledLoopRunsImmediatelyToMaxRuns(t *testing.T) {
	runs := 0
	report, err := RunScheduledLoop(context.Background(), ScheduledLoopConfig{
		Interval:       time.Hour,
		MaxRuns:        1,
		RunImmediately: true,
	}, func(context.Context) (ScheduledPullReport, error) {
		runs++
		return ScheduledPullReport{EventsBuilt: 2}, nil
	})
	if err != nil {
		t.Fatalf("run scheduled loop: %v", err)
	}
	if runs != 1 || report.Runs != 1 || report.Succeeded != 1 || report.Failed != 0 {
		t.Fatalf("runs/report = %d/%+v", runs, report)
	}
	if report.LastReport == nil || report.LastReport.EventsBuilt != 2 {
		t.Fatalf("last report = %+v", report.LastReport)
	}
}

func TestRunScheduledLoopStopsOnRunError(t *testing.T) {
	report, err := RunScheduledLoop(context.Background(), ScheduledLoopConfig{
		Interval:       time.Hour,
		MaxRuns:        1,
		RunImmediately: true,
	}, func(context.Context) (ScheduledPullReport, error) {
		return ScheduledPullReport{Failures: 1}, errors.New("pull failed")
	})
	if err == nil {
		t.Fatal("expected scheduled loop error")
	}
	if report.Runs != 1 || report.Failed != 1 || report.LastError != "pull failed" {
		t.Fatalf("report = %+v", report)
	}
}

func TestRunScheduledLoopContinuesOnRunError(t *testing.T) {
	runs := 0
	report, err := RunScheduledLoop(context.Background(), ScheduledLoopConfig{
		Interval:           time.Millisecond,
		MaxRuns:            2,
		RunImmediately:     true,
		ContinueOnRunError: true,
	}, func(context.Context) (ScheduledPullReport, error) {
		runs++
		if runs == 1 {
			return ScheduledPullReport{Failures: 1}, errors.New("temporary pull failure")
		}
		return ScheduledPullReport{EventsBuilt: 1}, nil
	})
	if err != nil {
		t.Fatalf("run scheduled loop: %v", err)
	}
	if report.Runs != 2 || report.Failed != 1 || report.Succeeded != 1 {
		t.Fatalf("report = %+v", report)
	}
}

func TestRunScheduledLoopRequiresInterval(t *testing.T) {
	_, err := RunScheduledLoop(context.Background(), ScheduledLoopConfig{}, func(context.Context) (ScheduledPullReport, error) {
		return ScheduledPullReport{}, nil
	})
	if err == nil {
		t.Fatal("expected interval error")
	}
}
