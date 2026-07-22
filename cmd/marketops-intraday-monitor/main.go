package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/adapters/marketdata/massive"
	"github.com/lukebabs/signalops/internal/config"
	"github.com/lukebabs/signalops/internal/storage"
	postgresstorage "github.com/lukebabs/signalops/internal/storage/postgres"
)

type condition struct {
	Key             string  `json:"key"`
	Title           string  `json:"title"`
	Tone            string  `json:"tone"`
	Score           float64 `json:"score"`
	Evidence        string  `json:"evidence"`
	Interpretation  string  `json:"interpretation"`
	AnalystQuestion string  `json:"analyst_question"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(context.Background(), logger); err != nil {
		logger.Error("intraday monitor failed", "error", err)
		os.Exit(1)
	}
}
func run(ctx context.Context, logger *slog.Logger) error {
	app := config.Load()
	if strings.TrimSpace(app.DatabaseURL) == "" {
		return errors.New("SIGNALOPS_DATABASE_URL is required")
	}
	tenant := flag.String("tenant-id", "tenant-local", "tenant id")
	group := flag.String("universe-group", "top50_megacap", "asset universe")
	max := flag.Int("max-symbols", 50, "maximum active assets")
	dry := flag.Bool("dry-run", false, "calculate without persisting")
	flag.Parse()
	repo, err := postgresstorage.Open(ctx, app.DatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	client, err := massive.NewClient(massive.LoadClientConfigFromEnv())
	if err != nil {
		return err
	}
	assets, err := repo.ListMarketOpsAssets(ctx, *tenant, *group, true, *max)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	status, active := marketSession(now)
	if !active {
		logger.Info("intraday monitor skipped outside regular and extended sessions", "time", now)
		return nil
	}
	asOf := now.Truncate(15 * time.Minute)
	persisted := 0
	failed := 0
	for _, asset := range assets {
		q, err := client.GetEquityQuote(ctx, asset.Ticker)
		if err != nil {
			failed++
			logger.Warn("quote unavailable", "symbol", asset.Ticker, "error", err)
			continue
		}
		q.MarketStatus = status
		q.Stale = false
		if !*dry {
			if err := repo.UpsertMarketOpsAssetQuote(ctx, storage.MarketOpsAssetQuoteRecord{TenantID: *tenant, UniverseGroup: *group, Ticker: q.Symbol, Price: q.Price, QuoteTimestamp: q.Timestamp, MarketStatus: status, Stale: false, PreviousClose: q.PreviousClose, Change: q.Change, ChangePercent: q.ChangePercent, Week52Low: q.Week52Low, Week52High: q.Week52High, RefreshedAt: now, Provider: "massive"}); err != nil {
				return err
			}
		}
		conditions := derive(q)
		body, _ := json.Marshal(conditions)
		source, _ := json.Marshal(map[string]any{"price": q.Price, "previous_close": q.PreviousClose, "change_percent": q.ChangePercent, "week52_low": q.Week52Low, "week52_high": q.Week52High, "quote_timestamp": q.Timestamp, "provider": "massive"})
		key := fmt.Sprintf("%s|%s|%s|%s", *tenant, *group, q.Symbol, asOf.Format(time.RFC3339))
		sum := sha256.Sum256([]byte(key))
		record := storage.MarketOpsIntradayConditionSnapshotRecord{SnapshotID: hex.EncodeToString(sum[:]), TenantID: *tenant, UniverseGroup: *group, Symbol: q.Symbol, AsOfTime: asOf, MarketStatus: q.MarketStatus, Stale: q.Stale, ConditionsJSON: body, SourcePayloadJSON: source}
		if !*dry {
			if err := repo.UpsertMarketOpsIntradayConditionSnapshot(ctx, record); err != nil {
				return err
			}
		}
		persisted++
	}
	logger.Info("intraday monitor completed", "assets", len(assets), "snapshots", persisted, "quote_failures", failed, "dry_run", *dry)
	return nil
}
func marketSession(now time.Time) (string, bool) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		return "", false
	}
	t := now.In(loc)
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return "", false
	}
	minutes := t.Hour()*60 + t.Minute()
	if minutes >= 9*60+30 && minutes < 16*60 {
		return "regular", true
	}
	if minutes >= 16*60 && minutes <= 20*60 {
		return "extended", true
	}
	return "", false
}

func derive(q massive.EquityQuote) []condition {
	out := []condition{}
	if q.ChangePercent != nil {
		p := *q.ChangePercent
		if p >= 1 {
			out = append(out, condition{"session_move_up", "Session momentum up", "positive", p, fmt.Sprintf("Price is %+0.2f%% versus the prior completed close.", p), "The asset is extending higher during this session.", "Is the move holding as volume and the broader market evolve?"})
		}
		if p <= -1 {
			out = append(out, condition{"session_move_down", "Session momentum down", "negative", -p, fmt.Sprintf("Price is %+0.2f%% versus the prior completed close.", p), "The asset is extending lower during this session.", "Is the decline broad-market beta or asset-specific?"})
		}
	}
	if q.Week52Low != nil && q.Week52High != nil && *q.Week52High > *q.Week52Low {
		position := (q.Price - *q.Week52Low) / (*q.Week52High - *q.Week52Low) * 100
		if position >= 90 {
			out = append(out, condition{"near_52w_high", "Near 52-week high", "positive", position, fmt.Sprintf("Price is at %0.0f%% of its 52-week low-to-high range.", position), "The asset is trading near the upper end of its one-year range.", "Does price sustain near the high after the next catalyst or session?"})
		}
		if position <= 10 {
			out = append(out, condition{"near_52w_low", "Near 52-week low", "negative", 100 - position, fmt.Sprintf("Price is at %0.0f%% of its 52-week low-to-high range.", position), "The asset is trading near the lower end of its one-year range.", "Is selling pressure easing or is a new downside catalyst present?"})
		}
	}
	return out
}
