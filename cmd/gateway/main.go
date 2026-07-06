package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lukebabs/signalops/internal/api"
	"github.com/lukebabs/signalops/internal/config"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg := config.Load()
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.NewRouter(api.RouterConfig{ServiceName: "signalops-gateway"}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("signalops gateway starting", "addr", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		logger.Error("signalops gateway failed", "error", err)
		os.Exit(1)
	case sig := <-stopCh:
		logger.Info("signalops gateway stopping", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("signalops gateway shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("signalops gateway stopped")
}

