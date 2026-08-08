package sri

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

type Repository interface {
	storage.MarketOpsSRIRepository
	ListMarketOpsEODEvents(context.Context, string, []string) ([]storage.NormalizedEventLedgerRecord, error)
}
type Config struct {
	TenantID, RunID string
	AsOf            time.Time
}
type Result struct {
	Segments  int
	Snapshots int
	Partial   int
}

func Run(ctx context.Context, repo Repository, cfg Config) (Result, error) {
	if strings.TrimSpace(cfg.TenantID) == "" {
		return Result{}, fmt.Errorf("tenant id is required")
	}
	if cfg.AsOf.IsZero() {
		cfg.AsOf = time.Now().UTC()
	}
	segs, etfs := FoundationRegistry(cfg.TenantID)
	for _, x := range segs {
		if err := repo.UpsertMarketOpsSRISegment(ctx, x); err != nil {
			return Result{}, err
		}
	}
	for _, x := range etfs {
		if err := repo.UpsertMarketOpsSRIETF(ctx, x); err != nil {
			return Result{}, err
		}
	}
	symbols := []string{}
	seen := map[string]bool{}
	for _, x := range etfs {
		if !seen[x.ETFSymbol] {
			seen[x.ETFSymbol] = true
			symbols = append(symbols, x.ETFSymbol)
		}
	}
	events, err := repo.ListMarketOpsEODEvents(ctx, cfg.TenantID, symbols)
	if err != nil {
		return Result{}, err
	}
	prices := priceHistory(events, cfg.AsOf)
	scored := Score(segs, etfs, prices)
	result := Result{Segments: len(scored)}
	for _, x := range scored {
		session := x.Session
		if session.IsZero() {
			session = cfg.AsOf.UTC()
		}
		components, _ := json.Marshal(x.Components)
		prov, _ := json.Marshal(x.Provenance)
		flags, _ := json.Marshal(x.Flags)
		key := strings.Join([]string{cfg.TenantID, x.Segment.SegmentID, session.Format("2006-01-02"), AlgorithmVersion}, "|")
		snapshot := storage.MarketOpsSRISnapshotRecord{SnapshotID: "sri_snapshot_" + stable(key), TenantID: cfg.TenantID, SegmentID: x.Segment.SegmentID, SessionDate: session, AsOfTime: cfg.AsOf, State: x.State, CompositeScore: x.Composite, RelativeStrengthScore: x.RelativeStrength, MomentumScore: x.Momentum, MomentumAcceleration: x.Acceleration, Rank: x.Rank, EvidenceQuality: x.EvidenceQuality, QualityState: x.QualityState, QualityFlagsJSON: flags, ComponentsJSON: components, InputProvenanceJSON: prov, AlgorithmVersion: AlgorithmVersion, ConfigurationVersion: RegistryVersion, CalculationRunID: cfg.RunID, DeterministicKey: key}
		if err := repo.UpsertMarketOpsSRISnapshot(ctx, snapshot); err != nil {
			return result, err
		}
		result.Snapshots++
		if x.QualityState != "usable" {
			result.Partial++
		}
	}
	return result, nil
}
func priceHistory(events []storage.NormalizedEventLedgerRecord, asOf time.Time) map[string][]PricePoint {
	by := map[string]map[string]PricePoint{}
	for _, e := range events {
		if e.ObservationTime.After(asOf) || e.Dataset != "equity_eod_prices" {
			continue
		}
		var p map[string]any
		if json.Unmarshal(e.NormalizedPayload, &p) != nil {
			continue
		}
		symbol := strings.ToUpper(str(p["symbol"]))
		closeValue, ok := num(p["close"])
		if symbol == "" || !ok || closeValue <= 0 {
			continue
		}
		if by[symbol] == nil {
			by[symbol] = map[string]PricePoint{}
		}
		key := e.ObservationTime.UTC().Format("2006-01-02")
		// The canonical event ledger is already idempotent per source event. For
		// duplicate observations on a session, the ordered ledger stream makes the
		// last normalized value authoritative.
		by[symbol][key] = PricePoint{Session: e.ObservationTime.UTC(), Close: closeValue, EventID: e.EventID}
	}
	out := map[string][]PricePoint{}
	for symbol, points := range by {
		for _, point := range points {
			out[symbol] = append(out[symbol], point)
		}
		sort.Slice(out[symbol], func(i, j int) bool { return out[symbol][i].Session.Before(out[symbol][j].Session) })
	}
	return out
}
func str(v any) string {
	if x, ok := v.(string); ok {
		return x
	}
	return ""
}
func num(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case json.Number:
		n, e := x.Float64()
		return n, e == nil
	default:
		return 0, false
	}
}
func stable(s string) string {
	var n uint64 = 1469598103934665603
	for _, c := range []byte(s) {
		n ^= uint64(c)
		n *= 1099511628211
	}
	return fmt.Sprintf("%x", n)
}
