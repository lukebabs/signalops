package main

import (
	"context"
	"flag"
	"github.com/lukebabs/signalops/internal/config"
	"github.com/lukebabs/signalops/internal/cyberops/detection"
	postgres "github.com/lukebabs/signalops/internal/storage/postgres"
	"log/slog"
	"time"
)

func main() {
	tenant := flag.String("tenant-id", "tenant-local", "")
	hours := flag.Int("hours", 192, "")
	flag.Parse()
	ctx := context.Background()
	cfg := config.Load()
	r, e := postgres.Open(ctx, cfg.DatabaseURL)
	if e != nil {
		panic(e)
	}
	defer r.Close()
	end := time.Now().UTC().Truncate(time.Hour)
	rows, e := r.DB().QueryContext(ctx, `SELECT observation_time,normalized_payload->>'message' FROM normalized_event_ledger WHERE tenant_id=$1 AND app_id='cyberops' AND dataset='cyberops.syslog.raw' AND observation_time >= $2 AND observation_time < $3`, *tenant, end.Add(-time.Duration(*hours)*time.Hour), end)
	if e != nil {
		panic(e)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var at time.Time
		var msg string
		rows.Scan(&at, &msg)
		f, ok := detection.ParseOPNsenseFilterlog(msg)
		if !ok || f.Action != "pass" {
			continue
		}
		_, e = r.DB().ExecContext(ctx, `INSERT INTO cyberops_iot_hourly_features(tenant_id,hour,device_ip,peer_ip,protocol,destination_port,allowed_log_count,first_seen,last_seen) VALUES($1,$2,$3::inet,$4::inet,$5,$6,1,$7,$7) ON CONFLICT(tenant_id,hour,device_ip,peer_ip,protocol,destination_port) DO UPDATE SET allowed_log_count=cyberops_iot_hourly_features.allowed_log_count+1,last_seen=EXCLUDED.last_seen`, *tenant, at.UTC().Truncate(time.Hour), f.SourceIP, f.DestinationIP, f.Protocol, f.DestinationPort, at.UTC())
		if e == nil {
			n++
		}
	}
	slog.Info("cyberops hourly feature materialization complete", "rows", n)
}
