// subscriber-global-ranking-import imports a user-provided, dated ranking
// snapshot. It never schedules collection or exposes a browser route.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "subscriber ranking import failed:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("subscriber-global-ranking-import", flag.ContinueOnError)
	databaseURL := flags.String("database-url", os.Getenv("SIGNALOPS_DATABASE_URL"), "catalog-sync PostgreSQL URL")
	input := flags.String("input", "", "ranked CSV input")
	asOf := flags.String("as-of", "", "required YYYY-MM-DD source snapshot date")
	actor := flags.String("actor", "subscriber-catalog-reference-sync", "catalog-sync identity")
	execute := flags.Bool("execute", false, "persist ranking snapshot")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !*execute {
		return errors.New("refusing to mutate: pass --execute after ranking-source review")
	}
	if strings.TrimSpace(*databaseURL) == "" || strings.TrimSpace(*input) == "" || strings.TrimSpace(*asOf) == "" {
		return errors.New("database-url, input, and as-of are required")
	}
	date, err := time.Parse("2006-01-02", *asOf)
	if err != nil {
		return fmt.Errorf("invalid as-of: %w", err)
	}
	bytes, err := os.ReadFile(*input)
	if err != nil {
		return err
	}
	entries, examined, duplicates, err := parse(bytes, 1000)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(bytes)
	provenance, _ := json.Marshal(map[string]any{"input_filename": *input, "selection_policy": "first_1000_distinct_symbols_by_source_rank", "source_as_of_date": date.Format("2006-01-02")})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	repo, err := postgresstorage.Open(ctx, *databaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	out, err := repo.ImportSubscriberGlobalRankingSnapshot(ctx, storage.SubscriberGlobalRankingSnapshotImport{SourceLabel: "companies.csv", SourceSHA256: hex.EncodeToString(sum[:]), AsOfDate: date, RequestedCapacity: 1000, SourceRowsExamined: examined, DuplicateSymbolsSkipped: duplicates, ImportedBy: *actor, ProvenanceJSON: provenance, Entries: entries})
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"ranking_snapshot_id": out.RankingSnapshotID, "as_of_date": date.Format("2006-01-02"), "selected": len(entries), "source_rows_examined": examined, "duplicate_symbols_skipped": duplicates, "source_sha256": out.SourceSHA256})
}

func parse(raw []byte, capacity int) ([]storage.SubscriberGlobalRankingSnapshotEntry, int, int, error) {
	reader := csv.NewReader(strings.NewReader(string(raw)))
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return nil, 0, 0, err
	}
	for i := range header {
		header[i] = strings.TrimPrefix(strings.TrimSpace(header[i]), "\ufeff")
	}
	if len(header) < 5 || header[0] != "No." || header[1] != "Symbol" || header[2] != "Company Name" || !strings.HasPrefix(header[3], "Market Cap") || header[4] != "Revenue" {
		return nil, 0, 0, errors.New("unexpected companies.csv header")
	}
	entries := []storage.SubscriberGlobalRankingSnapshotEntry{}
	seen := map[string]bool{}
	examined, duplicates := 0, 0
	previousRank := 0
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, examined, duplicates, err
		}
		if len(row) < 5 {
			return nil, examined, duplicates, errors.New("short companies.csv row")
		}
		rank, err := strconv.Atoi(strings.TrimSpace(row[0]))
		if err != nil || rank <= previousRank {
			return nil, examined, duplicates, errors.New("invalid non-contiguous source rank")
		}
		previousRank = rank
		examined++
		symbol := strings.ToUpper(strings.TrimSpace(row[1]))
		if symbol == "" {
			return nil, examined, duplicates, errors.New("blank symbol")
		}
		if seen[symbol] {
			duplicates++
			continue
		}
		seen[symbol] = true
		if len(entries) < capacity {
			sum := sha256.Sum256([]byte(strings.Join(row[:5], "\x00")))
			entries = append(entries, storage.SubscriberGlobalRankingSnapshotEntry{SelectionRank: len(entries) + 1, SourceRank: rank, ProviderSymbol: symbol, CompanyName: strings.TrimSpace(row[2]), MarketCapRaw: strings.TrimSpace(row[3]), RevenueRaw: strings.TrimSpace(row[4]), SourceRowSHA256: hex.EncodeToString(sum[:])})
		}
		if len(entries) == capacity {
			break
		}
	}
	if len(entries) != capacity {
		return nil, examined, duplicates, errors.New("ranking input has fewer than requested distinct symbols")
	}
	return entries, examined, duplicates, nil
}
