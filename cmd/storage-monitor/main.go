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
	"syscall"
	"time"
)

type store struct{ id, path string }

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(ctx context.Context) error {
	dsn := flag.String("database-url", os.Getenv("SIGNALOPS_DATABASE_URL"), "primary database URL")
	flag.Parse()
	if *dsn == "" {
		return fmt.Errorf("SIGNALOPS_DATABASE_URL is required")
	}
	db, err := sql.Open("pgx", *dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	for _, s := range []store{{"postgres", "/storage/postgres"}, {"timescaledb", "/storage/timescaledb"}, {"redpanda", "/storage/redpanda"}} {
		x := measure(s)
		raw, _ := json.Marshal(x.detail)
		_, err = db.ExecContext(ctx, `INSERT INTO storage_monitor_snapshots(store_id,used_bytes,capacity_bytes,free_bytes,status,detail) VALUES($1,$2,$3,$4,$5,$6)`, s.id, x.used, x.capacity, x.free, x.status, raw)
		if err != nil {
			return err
		}
	}
	return nil
}

type measurement struct {
	used, capacity, free int64
	status               string
	detail               map[string]any
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
	var used int64
	err := filepath.WalkDir(s.path, func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.Type().IsRegular() {
			i, e := d.Info()
			if e == nil {
				used += i.Size()
			}
		}
		return nil
	})
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
