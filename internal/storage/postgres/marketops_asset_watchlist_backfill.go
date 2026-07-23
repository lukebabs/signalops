package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) UpsertMarketOpsAsset(ctx context.Context, record storage.MarketOpsAssetRecord) (storage.MarketOpsAssetRecord, error) {
	if strings.TrimSpace(record.TenantID) == "" || strings.TrimSpace(record.Ticker) == "" {
		return storage.MarketOpsAssetRecord{}, fmt.Errorf("tenant id and ticker are required")
	}
	if record.UniverseGroup == "" {
		record.UniverseGroup = "analyst_watchlist"
	}
	if record.UniverseGroup == "analyst_watchlist" {
		ticker := strings.ToUpper(strings.TrimSpace(record.Ticker))
		var exists bool
		if err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM marketops_asset_universe WHERE tenant_id=$1 AND universe_group=$2 AND ticker=$3)`, record.TenantID, record.UniverseGroup, ticker).Scan(&exists); err != nil {
			return storage.MarketOpsAssetRecord{}, err
		}
		var inPrimaryUniverse bool
		if err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM marketops_asset_universe WHERE tenant_id=$1 AND universe_group<>$2 AND ticker=$3 AND is_active)`, record.TenantID, record.UniverseGroup, ticker).Scan(&inPrimaryUniverse); err != nil {
			return storage.MarketOpsAssetRecord{}, err
		}
		if inPrimaryUniverse {
			return storage.MarketOpsAssetRecord{}, fmt.Errorf("ticker already exists in the primary asset universe")
		}
		if !exists {
			var active int
			if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM marketops_asset_universe WHERE tenant_id=$1 AND universe_group=$2 AND is_active`, record.TenantID, record.UniverseGroup).Scan(&active); err != nil {
				return storage.MarketOpsAssetRecord{}, err
			}
			if active >= 50 {
				return storage.MarketOpsAssetRecord{}, fmt.Errorf("analyst watchlist capacity reached")
			}
		}
	}
	if record.AppID == "" {
		record.AppID = "marketops"
	}
	if record.Domain == "" {
		record.Domain = "market_data"
	}
	if record.UseCase == "" {
		record.UseCase = "daily_market_surveillance"
	}
	if record.SourceID == "" {
		record.SourceID = "src-massive"
	}
	if record.AssetType == "" {
		record.AssetType = "equity"
	}
	record.Ticker = strings.ToUpper(strings.TrimSpace(record.Ticker))
	record.TickerKey = strings.ToLower(record.Ticker)
	if record.Company == "" {
		record.Company = record.Ticker
	}
	if record.CompanyKey == "" {
		record.CompanyKey = strings.ToLower(strings.ReplaceAll(record.Company, " ", "_"))
	}
	if record.MetadataJSON == nil {
		record.MetadataJSON = []byte(`{}`)
	}
	if record.Rank <= 0 {
		if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(rank), 0) + 1 FROM marketops_asset_universe WHERE tenant_id=$1 AND universe_group=$2`, record.TenantID, record.UniverseGroup).Scan(&record.Rank); err != nil {
			return storage.MarketOpsAssetRecord{}, fmt.Errorf("allocate asset rank: %w", err)
		}
	}
	row := r.db.QueryRowContext(ctx, `INSERT INTO marketops_asset_universe (tenant_id,app_id,domain,use_case,source_id,universe_group,rank,ticker,ticker_key,company,company_key,asset_type,exchange,sector,sector_key,industry,industry_key,is_active,metadata) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,true,$18) ON CONFLICT (tenant_id,universe_group,ticker) DO UPDATE SET company=EXCLUDED.company, company_key=EXCLUDED.company_key, sector=EXCLUDED.sector, sector_key=EXCLUDED.sector_key, industry=EXCLUDED.industry, industry_key=EXCLUDED.industry_key, exchange=EXCLUDED.exchange, is_active=true, metadata=EXCLUDED.metadata, updated_at=now() RETURNING tenant_id,app_id,domain,use_case,source_id,universe_group,rank,ticker,ticker_key,company,company_key,asset_type,exchange,sector,sector_key,industry,industry_key,is_active,metadata,created_at,updated_at`, record.TenantID, record.AppID, record.Domain, record.UseCase, record.SourceID, record.UniverseGroup, record.Rank, record.Ticker, record.TickerKey, record.Company, record.CompanyKey, record.AssetType, record.Exchange, record.Sector, record.SectorKey, record.Industry, record.IndustryKey, record.MetadataJSON)
	return scanMarketOpsAsset(row)
}

const assetBackfillColumns = `backfill_job_id,tenant_id,symbol,universe_group,start_date,end_date,status,requested_by,requested_sessions,completed_sessions,failed_sessions,provider_requests,COALESCE(error_message,''),result,started_at,completed_at,created_at,updated_at`
const assetBackfillReturningColumns = `j.backfill_job_id,j.tenant_id,j.symbol,j.universe_group,j.start_date,j.end_date,j.status,j.requested_by,j.requested_sessions,j.completed_sessions,j.failed_sessions,j.provider_requests,COALESCE(j.error_message,''),j.result,j.started_at,j.completed_at,j.created_at,j.updated_at`

func (r *Repository) CreateMarketOpsAssetBackfillJob(ctx context.Context, x storage.MarketOpsAssetBackfillJobRecord) (storage.MarketOpsAssetBackfillJobRecord, error) {
	row := r.db.QueryRowContext(ctx, `INSERT INTO marketops_asset_backfill_jobs (backfill_job_id,tenant_id,symbol,universe_group,start_date,end_date,status,requested_by,requested_sessions,result) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING `+assetBackfillColumns, x.BackfillJobID, x.TenantID, x.Symbol, x.UniverseGroup, x.StartDate, x.EndDate, x.Status, x.RequestedBy, x.RequestedSessions, x.ResultJSON)
	return scanMarketOpsAssetBackfillJob(row)
}

func (r *Repository) ListMarketOpsAssetBackfillJobs(ctx context.Context, f storage.MarketOpsAssetBackfillJobFilter) ([]storage.MarketOpsAssetBackfillJobRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+assetBackfillColumns+` FROM marketops_asset_backfill_jobs WHERE tenant_id=$1 AND ($2='' OR symbol=$2) AND ($3='' OR status=$3) ORDER BY created_at DESC LIMIT $4`, f.TenantID, strings.ToUpper(strings.TrimSpace(f.Symbol)), strings.TrimSpace(f.Status), clampLimit(f.Limit))
	if err != nil {
		return nil, fmt.Errorf("list asset backfill jobs: %w", err)
	}
	defer rows.Close()
	out := []storage.MarketOpsAssetBackfillJobRecord{}
	for rows.Next() {
		x, err := scanMarketOpsAssetBackfillJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *Repository) GetMarketOpsAssetBackfillJob(ctx context.Context, tenantID, jobID string) (storage.MarketOpsAssetBackfillJobRecord, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+assetBackfillColumns+` FROM marketops_asset_backfill_jobs WHERE tenant_id=$1 AND backfill_job_id=$2`, tenantID, jobID)
	x, err := scanMarketOpsAssetBackfillJob(row)
	if err == sql.ErrNoRows {
		return x, storage.ErrNotFound
	}
	return x, err
}

func (r *Repository) ClaimNextMarketOpsAssetBackfillJob(ctx context.Context, workerID string, claimedAt time.Time) (storage.MarketOpsAssetBackfillJobRecord, error) {
	row := r.db.QueryRowContext(ctx, `WITH next AS (SELECT backfill_job_id FROM marketops_asset_backfill_jobs WHERE status='queued' ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1) UPDATE marketops_asset_backfill_jobs j SET status='running', started_at=COALESCE(started_at,$1), updated_at=$1, result=jsonb_set(result,'{worker_id}',to_jsonb($2::text),true) FROM next WHERE j.backfill_job_id=next.backfill_job_id RETURNING `+assetBackfillReturningColumns, claimedAt, workerID)
	x, err := scanMarketOpsAssetBackfillJob(row)
	if err == sql.ErrNoRows {
		return x, storage.ErrNotFound
	}
	return x, err
}

func (r *Repository) UpdateMarketOpsAssetBackfillJob(ctx context.Context, x storage.MarketOpsAssetBackfillJobRecord) error {
	_, err := r.db.ExecContext(ctx, `UPDATE marketops_asset_backfill_jobs SET status=$3,completed_sessions=$4,failed_sessions=$5,provider_requests=$6,error_message=NULLIF($7,''),result=$8,started_at=$9,completed_at=$10,updated_at=now() WHERE tenant_id=$1 AND backfill_job_id=$2`, x.TenantID, x.BackfillJobID, x.Status, x.CompletedSessions, x.FailedSessions, x.ProviderRequests, x.ErrorMessage, x.ResultJSON, x.StartedAt, x.CompletedAt)
	return err
}

func scanMarketOpsAssetBackfillJob(s interface{ Scan(...any) error }) (storage.MarketOpsAssetBackfillJobRecord, error) {
	var x storage.MarketOpsAssetBackfillJobRecord
	err := s.Scan(&x.BackfillJobID, &x.TenantID, &x.Symbol, &x.UniverseGroup, &x.StartDate, &x.EndDate, &x.Status, &x.RequestedBy, &x.RequestedSessions, &x.CompletedSessions, &x.FailedSessions, &x.ProviderRequests, &x.ErrorMessage, &x.ResultJSON, &x.StartedAt, &x.CompletedAt, &x.CreatedAt, &x.UpdatedAt)
	return x, err
}

var _ = json.Valid
