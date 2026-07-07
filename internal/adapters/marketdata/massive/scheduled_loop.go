package massive

import (
	"context"
	"errors"
	"time"
)

type ScheduledPullFunc func(context.Context) (ScheduledPullReport, error)

type ScheduledLoopConfig struct {
	Interval           time.Duration
	MaxRuns            int
	RunImmediately     bool
	ContinueOnRunError bool
}

type ScheduledLoopReport struct {
	Runs       int                  `json:"runs"`
	Succeeded  int                  `json:"succeeded"`
	Failed     int                  `json:"failed"`
	LastReport *ScheduledPullReport `json:"last_report,omitempty"`
	LastError  string               `json:"last_error,omitempty"`
}

func RunScheduledLoop(ctx context.Context, cfg ScheduledLoopConfig, run ScheduledPullFunc) (ScheduledLoopReport, error) {
	if run == nil {
		return ScheduledLoopReport{}, errors.New("scheduled pull function is required")
	}
	if cfg.Interval <= 0 {
		return ScheduledLoopReport{}, errors.New("schedule interval must be greater than zero")
	}

	report := ScheduledLoopReport{}
	if cfg.RunImmediately {
		if err := runScheduledIteration(ctx, cfg, run, &report); err != nil {
			return report, err
		}
		if cfg.MaxRuns > 0 && report.Runs >= cfg.MaxRuns {
			return report, nil
		}
	}

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return report, ctx.Err()
		case <-ticker.C:
			if err := runScheduledIteration(ctx, cfg, run, &report); err != nil {
				return report, err
			}
			if cfg.MaxRuns > 0 && report.Runs >= cfg.MaxRuns {
				return report, nil
			}
		}
	}
}

func runScheduledIteration(ctx context.Context, cfg ScheduledLoopConfig, run ScheduledPullFunc, report *ScheduledLoopReport) error {
	pullReport, err := run(ctx)
	report.Runs++
	report.LastReport = &pullReport
	if err != nil {
		report.Failed++
		report.LastError = err.Error()
		if !cfg.ContinueOnRunError {
			return err
		}
		return nil
	}
	report.Succeeded++
	report.LastError = ""
	return nil
}
