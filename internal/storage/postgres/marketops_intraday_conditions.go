package postgres

import (
 "context"
 "fmt"
 "strings"
 "github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) UpsertMarketOpsIntradayConditionSnapshot(ctx context.Context, record storage.MarketOpsIntradayConditionSnapshotRecord) error {
 if strings.TrimSpace(record.SnapshotID)=="" || strings.TrimSpace(record.TenantID)=="" || strings.TrimSpace(record.Symbol)=="" || record.AsOfTime.IsZero() { return fmt.Errorf("intraday snapshot id, tenant_id, symbol, and as_of_time are required") }
 if strings.TrimSpace(record.UniverseGroup)=="" { record.UniverseGroup="top50_megacap" }; if strings.TrimSpace(record.MarketStatus)=="" { record.MarketStatus="intraday" }
 _,err:=r.db.ExecContext(ctx, `INSERT INTO marketops_intraday_condition_snapshots (snapshot_id,tenant_id,universe_group,symbol,as_of_time,market_status,stale,conditions,source_payload) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT (tenant_id,universe_group,symbol,as_of_time) DO UPDATE SET market_status=EXCLUDED.market_status,stale=EXCLUDED.stale,conditions=EXCLUDED.conditions,source_payload=EXCLUDED.source_payload`, strings.TrimSpace(record.SnapshotID),strings.TrimSpace(record.TenantID),strings.TrimSpace(record.UniverseGroup),strings.ToUpper(strings.TrimSpace(record.Symbol)),record.AsOfTime.UTC(),strings.TrimSpace(record.MarketStatus),record.Stale,jsonArrayOrEmpty(record.ConditionsJSON),jsonOrEmpty(record.SourcePayloadJSON))
 if err!=nil{return fmt.Errorf("upsert marketops intraday condition snapshot: %w",err)}; return nil
}
func (r *Repository) ListMarketOpsIntradayConditionSnapshots(ctx context.Context, filter storage.MarketOpsIntradayConditionSnapshotFilter) ([]storage.MarketOpsIntradayConditionSnapshotRecord,error) {
 rows,err:=r.db.QueryContext(ctx, `SELECT snapshot_id,tenant_id,universe_group,symbol,as_of_time,market_status,stale,conditions,source_payload,created_at FROM marketops_intraday_condition_snapshots WHERE ($1='' OR tenant_id=$1) AND ($2='' OR universe_group=$2) AND ($3='' OR symbol=$3) AND ($4::timestamptz IS NULL OR as_of_time >= $4::timestamptz) ORDER BY symbol ASC,as_of_time DESC LIMIT $5`,strings.TrimSpace(filter.TenantID),strings.TrimSpace(filter.UniverseGroup),strings.ToUpper(strings.TrimSpace(filter.Symbol)),nullTime(filter.Since),clampLimit(filter.Limit)); if err!=nil{return nil,fmt.Errorf("list marketops intraday condition snapshots: %w",err)}; defer rows.Close()
 out:=[]storage.MarketOpsIntradayConditionSnapshotRecord{}; for rows.Next(){var x storage.MarketOpsIntradayConditionSnapshotRecord; if err:=rows.Scan(&x.SnapshotID,&x.TenantID,&x.UniverseGroup,&x.Symbol,&x.AsOfTime,&x.MarketStatus,&x.Stale,&x.ConditionsJSON,&x.SourcePayloadJSON,&x.CreatedAt);err!=nil{return nil,mapScanError("scan marketops intraday condition snapshot",err)};out=append(out,x)}; return out,rows.Err()
}
