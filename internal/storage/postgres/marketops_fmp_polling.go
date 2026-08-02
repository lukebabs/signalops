package postgres

import (
 "context"
 "database/sql"
 "fmt"
 "strings"
 "time"
 "github.com/lukebabs/signalops/internal/storage"
)
func (r *Repository) ReserveMarketOpsFMPCalls(ctx context.Context, tenant string, day time.Time, max, calls int) (bool,error) {
 if strings.TrimSpace(tenant)=="" || day.IsZero() || calls<=0 || max<calls { return false,fmt.Errorf("invalid FMP budget reservation") }
 tx,err:=r.db.BeginTx(ctx,nil); if err!=nil{return false,err}; defer tx.Rollback()
 if _,err=tx.ExecContext(ctx,`INSERT INTO marketops_fmp_daily_budgets (tenant_id,provider_day,max_calls) VALUES ($1,$2,$3) ON CONFLICT (tenant_id,provider_day) DO UPDATE SET max_calls=LEAST(marketops_fmp_daily_budgets.max_calls,EXCLUDED.max_calls),updated_at=now()`,tenant,day.UTC(),max);err!=nil{return false,err}
 result,err:=tx.ExecContext(ctx,`UPDATE marketops_fmp_daily_budgets SET reserved_calls=reserved_calls+$3,updated_at=now() WHERE tenant_id=$1 AND provider_day=$2 AND reserved_calls+$3<=max_calls`,tenant,day.UTC(),calls);if err!=nil{return false,err}; n,err:=result.RowsAffected();if err!=nil{return false,err};if n==0{return false,nil};return true,tx.Commit()
}
func (r *Repository) CompleteMarketOpsFMPCalls(ctx context.Context, tenant string, day time.Time, calls int) error { _,err:=r.db.ExecContext(ctx,`UPDATE marketops_fmp_daily_budgets SET completed_calls=completed_calls+$3,updated_at=now() WHERE tenant_id=$1 AND provider_day=$2`,tenant,day.UTC(),calls);return err }
func (r *Repository) UpsertMarketOpsFMPPollState(ctx context.Context, x storage.MarketOpsFMPPollState) error { if strings.TrimSpace(x.TenantID)==""||strings.TrimSpace(x.Symbol)==""||strings.TrimSpace(x.Status)=="" {return fmt.Errorf("invalid FMP poll state")};_,err:=r.db.ExecContext(ctx,`INSERT INTO marketops_fmp_poll_states (tenant_id,symbol,status,last_success_at,next_eligible_at,attempt_count,last_provider_status,last_error,financial_snapshot_id,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,now()) ON CONFLICT (tenant_id,symbol) DO UPDATE SET status=EXCLUDED.status,last_success_at=EXCLUDED.last_success_at,next_eligible_at=EXCLUDED.next_eligible_at,attempt_count=EXCLUDED.attempt_count,last_provider_status=EXCLUDED.last_provider_status,last_error=EXCLUDED.last_error,financial_snapshot_id=EXCLUDED.financial_snapshot_id,updated_at=now()`,x.TenantID,strings.ToUpper(x.Symbol),x.Status,x.LastSuccessAt,x.NextEligibleAt,x.AttemptCount,x.LastProviderStatus,x.LastError,nullString(x.FinancialSnapshotID));return err }
func (r *Repository) GetMarketOpsFMPPollState(ctx context.Context, tenant,symbol string)(storage.MarketOpsFMPPollState,error){var x storage.MarketOpsFMPPollState;var success,next sql.NullTime;var status sql.NullInt32;var snapshot sql.NullString;err:=r.db.QueryRowContext(ctx,`SELECT tenant_id,symbol,status,last_success_at,next_eligible_at,attempt_count,last_provider_status,last_error,financial_snapshot_id,updated_at FROM marketops_fmp_poll_states WHERE tenant_id=$1 AND symbol=$2`,tenant,strings.ToUpper(symbol)).Scan(&x.TenantID,&x.Symbol,&x.Status,&success,&next,&x.AttemptCount,&status,&x.LastError,&snapshot,&x.UpdatedAt);if err!=nil{return x,err};if success.Valid{x.LastSuccessAt=&success.Time};if next.Valid{x.NextEligibleAt=&next.Time};if status.Valid{v:=int(status.Int32);x.LastProviderStatus=&v};if snapshot.Valid{x.FinancialSnapshotID=snapshot.String};return x,nil}
