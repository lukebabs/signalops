package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/lukebabs/signalops/internal/config"
	"github.com/lukebabs/signalops/internal/signals"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
	"github.com/lukebabs/signalops/pkg/broker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger, os.Args[1:]); err != nil {
		logger.Error("signalops signal backfill failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger, args []string) error {
	var payloadPath string
	var topic string
	var partition int
	var startOffset int64
	var dryRun bool
	flags := flag.NewFlagSet("signal-backfill", flag.ContinueOnError)
	flags.StringVar(&payloadPath, "payload-jsonl", "", "path to newline-delimited signal.v1 payloads")
	flags.StringVar(&topic, "topic", "", "source topic to attach as broker lineage")
	flags.IntVar(&partition, "partition", -1, "source partition to attach as broker lineage")
	flags.Int64Var(&startOffset, "start-offset", 0, "broker offset for the first payload line")
	flags.BoolVar(&dryRun, "dry-run", false, "decode and validate payloads without persisting")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(payloadPath) == "" {
		return errors.New("-payload-jsonl is required")
	}
	if strings.TrimSpace(topic) == "" {
		return errors.New("-topic is required")
	}
	if partition < 0 {
		return errors.New("-partition must be non-negative")
	}

	cfg := config.Load()
	if strings.TrimSpace(cfg.DatabaseURL) == "" && !dryRun {
		return errors.New("SIGNALOPS_DATABASE_URL is required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	file, err := os.Open(payloadPath)
	if err != nil {
		return fmt.Errorf("open payload jsonl: %w", err)
	}
	defer file.Close()

	var processor signals.Processor
	var repository *postgresstorage.Repository
	if !dryRun {
		repository, err = postgresstorage.OpenWithTemporal(ctx, cfg.DatabaseURL, cfg.TemporalDatabaseURL)
		if err != nil {
			return err
		}
		defer repository.Close()
		processor = signals.Processor{Repository: repository}
	}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	count := 0
	for scanner.Scan() {
		value := strings.TrimSpace(scanner.Text())
		if value == "" {
			continue
		}
		offset := startOffset + int64(count)
		message := broker.ConsumedMessage{
			Message: broker.Message{
				Topic: topic,
				Value: []byte(value),
			},
			Partition: int32(partition),
			Offset:    offset,
		}
		if dryRun {
			event, err := signals.DecodeEvent(message.Value)
			if err != nil {
				return fmt.Errorf("decode payload at offset %d: %w", offset, err)
			}
			logger.Info("signal decoded", "signal_id", event.SignalID, "partition", partition, "offset", offset)
		} else {
			record, err := processor.Process(ctx, message)
			if err != nil {
				return fmt.Errorf("persist signal at offset %d: %w", offset, err)
			}
			logger.Info("signal backfilled", "signal_id", record.SignalID, "detector_id", record.DetectorID, "partition", partition, "offset", offset)
		}
		count++
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan payload jsonl: %w", err)
	}
	logger.Info("signal backfill complete", "count", count, "partition", partition, "start_offset", startOffset, "dry_run", dryRun)
	return nil
}
