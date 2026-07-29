package main

import (
	"context"
	"github.com/lukebabs/signalops/internal/config"
	"github.com/lukebabs/signalops/internal/cyberops/anomaly"
	postgres "github.com/lukebabs/signalops/internal/storage/postgres"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	repo, err := postgres.OpenWithTemporal(ctx, cfg.DatabaseURL, cfg.TemporalDatabaseURL)
	if err != nil {
		logger.Error("iot anomaly worker failed", "error", err)
		return
	}
	defer repo.Close()
	run := func() {
		n, err := anomaly.Run(ctx, repo, time.Now())
		if err != nil {
			logger.Error("iot anomaly run failed", "error", err)
		} else {
			logger.Info("iot anomaly run complete", "results", n)
		}
	}
	run()
	wait := time.Until(time.Now().UTC().Truncate(time.Hour).Add(time.Hour))
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
	}
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		run()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
