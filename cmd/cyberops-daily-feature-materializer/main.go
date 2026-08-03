package main

import (
	"context"
	"flag"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lukebabs/signalops/internal/config"
	postgres "github.com/lukebabs/signalops/internal/storage/postgres"
	"os"
	"time"
)

func main() {
	tenant := flag.String("tenant-id", "tenant-local", "")
	days := flag.Int("days", 2, "")
	flag.Parse()
	ctx := context.Background()
	cfg := config.Load()
	repo, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer repo.Close()
	end := time.Now().UTC().Truncate(24 * time.Hour)
	start := end.AddDate(0, 0, -*days)
	result, err := repo.DB().ExecContext(ctx, `INSERT INTO cyberops_iot_daily_features(tenant_id,feature_date,device_ip,peer_ip,protocol,destination_port,allowed_log_count,active_hours,first_seen,last_seen) SELECT tenant_id,hour::date,device_ip,peer_ip,protocol,destination_port,sum(allowed_log_count),count(DISTINCT hour),min(first_seen),max(last_seen) FROM cyberops_iot_hourly_features WHERE tenant_id=$1 AND hour >= $2 AND hour < $3 GROUP BY tenant_id,hour::date,device_ip,peer_ip,protocol,destination_port ON CONFLICT(tenant_id,feature_date,device_ip,peer_ip,protocol,destination_port) DO UPDATE SET allowed_log_count=EXCLUDED.allowed_log_count,active_hours=EXCLUDED.active_hours,first_seen=EXCLUDED.first_seen,last_seen=EXCLUDED.last_seen`, *tenant, start, end)
	if err != nil {
		panic(err)
	}
	n, _ := result.RowsAffected()
	fmt.Printf("materialized %d cyberops daily feature rows\n", n)
	_ = os.Stdout
}
