package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

type store struct{ id, path string }
type component struct {
	storeID, kind, name, appID, domain, attribution string
	physicalBytes, attributedBytes                  int64
	metadata                                        map[string]any
}
type measurement struct {
	used, capacity, free int64
	status               string
	detail               map[string]any
}

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(ctx context.Context) error {
	primaryDSN := flag.String("database-url", os.Getenv("SIGNALOPS_DATABASE_URL"), "primary database URL")
	temporalDSN := flag.String("temporal-database-url", os.Getenv("SIGNALOPS_TEMPORAL_DATABASE_URL"), "temporal database URL")
	flag.Parse()
	if *primaryDSN == "" {
		return fmt.Errorf("SIGNALOPS_DATABASE_URL is required")
	}
	db, err := sql.Open("pgx", *primaryDSN)
	if err != nil {
		return err
	}
	defer db.Close()
	now := time.Now().UTC()
	location, _ := time.LoadLocation("America/New_York")
	local := now.In(location)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location).UTC()
	detailed, err := needsDetailedSnapshot(ctx, db, start)
	if local.Hour() != 2 {
		detailed = false
	}
	if err != nil {
		return err
	}
	for _, s := range []store{{"postgres", "/storage/postgres"}, {"timescaledb", "/storage/timescaledb"}, {"redpanda", "/storage/redpanda"}} {
		x := measure(s)
		raw, _ := json.Marshal(x.detail)
		var snapshotID int64
		if err := db.QueryRowContext(ctx, `INSERT INTO storage_monitor_snapshots(store_id,used_bytes,capacity_bytes,free_bytes,status,detail) VALUES($1,$2,$3,$4,$5,$6) RETURNING snapshot_id`, s.id, x.used, x.capacity, x.free, x.status, raw).Scan(&snapshotID); err != nil {
			return err
		}
		if !detailed || x.status == "unavailable" {
			continue
		}
		var entries []component
		switch s.id {
		case "postgres":
			entries, err = postgresComponents(ctx, db)
		case "timescaledb":
			if *temporalDSN != "" {
				entries, err = temporalComponents(ctx, *temporalDSN)
			}
		case "redpanda":
			entries, err = redpandaComponents(s.path)
		}
		if err != nil {
			return fmt.Errorf("collect %s components: %w", s.id, err)
		}
		for _, entry := range entries {
			if err := insertComponent(ctx, db, snapshotID, entry); err != nil {
				return err
			}
		}
	}
	if detailed {
		_, err = db.ExecContext(ctx, `DELETE FROM storage_monitor_snapshots WHERE observed_at < $1`, now.AddDate(0, 0, -90))
		if err != nil {
			return err
		}
	}
	return nil
}
func needsDetailedSnapshot(ctx context.Context, db *sql.DB, start time.Time) (bool, error) {
	var exists bool
	err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM storage_component_snapshots c JOIN storage_monitor_snapshots s ON s.snapshot_id=c.snapshot_id WHERE s.observed_at >= $1)`, start).Scan(&exists)
	return !exists, err
}
func insertComponent(ctx context.Context, db *sql.DB, id int64, c component) error {
	raw, _ := json.Marshal(c.metadata)
	_, err := db.ExecContext(ctx, `INSERT INTO storage_component_snapshots(snapshot_id,store_id,component_kind,component_name,app_id,domain,attribution_method,physical_bytes,attributed_bytes,metadata) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, id, c.storeID, c.kind, c.name, c.appID, c.domain, c.attribution, c.physicalBytes, c.attributedBytes, raw)
	return err
}
func postgresComponents(ctx context.Context, db *sql.DB) ([]component, error) {
	rows, err := db.QueryContext(ctx, `SELECT c.relname,pg_total_relation_size(c.oid) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relkind IN ('r','m') ORDER BY c.relname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []component{}
	for rows.Next() {
		var name string
		var size int64
		if err = rows.Scan(&name, &size); err != nil {
			return nil, err
		}
		allocated, err := sharedAllocations(ctx, db, name, size)
		if err != nil {
			return nil, err
		}
		if len(allocated) > 0 {
			out = append(out, allocated...)
		} else {
			app, domain := classify(name)
			out = append(out, component{"postgres", "table", name, app, domain, "exact", size, size, map[string]any{}})
		}
	}
	return out, rows.Err()
}
func sharedAllocations(ctx context.Context, db *sql.DB, table string, size int64) ([]component, error) {
	if !isSharedLedger(table) {
		return nil, nil
	}
	var hasApp bool
	if err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name=$1 AND column_name='app_id')`, table).Scan(&hasApp); err != nil || !hasApp {
		return nil, err
	}
	query := fmt.Sprintf(`SELECT COALESCE(NULLIF(app_id,''),'platform'),COALESCE(NULLIF(domain,''),'platform'),count(*) FROM %s GROUP BY 1,2 ORDER BY 1,2`, table)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type group struct {
		app, domain string
		count       int64
	}
	groups := []group{}
	var total int64
	for rows.Next() {
		var g group
		if err = rows.Scan(&g.app, &g.domain, &g.count); err != nil {
			return nil, err
		}
		groups = append(groups, g)
		total += g.count
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if total == 0 {
		return nil, nil
	}
	out := make([]component, 0, len(groups))
	var assigned int64
	for i, g := range groups {
		bytes := size * g.count / total
		if i == len(groups)-1 {
			bytes = size - assigned
		}
		assigned += bytes
		out = append(out, component{"postgres", "table", table, g.app, g.domain, "estimated", size, bytes, map[string]any{"allocation_basis": "row_count", "row_count": g.count}})
	}
	return out, nil
}
func temporalComponents(ctx context.Context, dsn string) ([]component, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT hypertable_name,hypertable_size(format('%I.%I',hypertable_schema,hypertable_name)) FROM timescaledb_information.hypertables ORDER BY hypertable_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []component{}
	for rows.Next() {
		var name string
		var size int64
		if err = rows.Scan(&name, &size); err != nil {
			return nil, err
		}
		app, domain := classify(name)
		out = append(out, component{"timescaledb", "hypertable", name, app, domain, "exact", size, size, map[string]any{}})
	}
	return out, rows.Err()
}
func redpandaComponents(root string) ([]component, error) {
	entries, err := os.ReadDir(filepath.Join(root, "kafka"))
	if err != nil {
		return nil, err
	}
	out := []component{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		size, err := directorySize(filepath.Join(root, "kafka", entry.Name()))
		if err != nil {
			return nil, err
		}
		app, domain := classify(entry.Name())
		out = append(out, component{"redpanda", "topic", entry.Name(), app, domain, "exact", size, size, map[string]any{}})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out, nil
}
func directorySize(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(_ string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.Type().IsRegular() {
			i, e := d.Info()
			if e != nil {
				return e
			}
			total += i.Size()
		}
		return nil
	})
	return total, err
}
func isSharedLedger(name string) bool {
	switch name {
	case "normalized_event_ledger", "signal_ledger", "alert_ledger", "insight_ledger":
		return true
	}
	return false
}
func classify(name string) (string, string) {
	value := strings.ToLower(name)
	if strings.HasPrefix(value, "marketops_") || strings.Contains(value, "marketops") {
		return "marketops", "market_data"
	}
	if strings.HasPrefix(value, "cyberops_") || strings.Contains(value, "cyberops") {
		return "cyberops", "security"
	}
	return "platform", "platform"
}
func measure(s store) measurement {
	x := measurement{status: "unavailable", detail: map[string]any{"path": s.path}}
	var st syscall.Statfs_t
	if err := syscall.Statfs(s.path, &st); err != nil {
		x.detail["error"] = err.Error()
		return x
	}
	x.capacity = int64(st.Blocks) * int64(st.Bsize)
	x.free = int64(st.Bavail) * int64(st.Bsize)
	used, err := directorySize(s.path)
	if err != nil {
		x.detail["error"] = err.Error()
		return x
	}
	x.used = used
	pct := 0.0
	if x.capacity > 0 {
		pct = float64(x.used) * 100 / float64(x.capacity)
	}
	if pct >= 90 {
		x.status = "critical"
	} else if pct >= 75 {
		x.status = "warning"
	} else {
		x.status = "healthy"
	}
	x.detail["usage_percent"] = pct
	x.detail["collected_at"] = time.Now().UTC().Format(time.RFC3339)
	return x
}
