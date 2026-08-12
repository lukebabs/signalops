package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

type subscriberCatalogSourceAsset struct {
	TenantID, SourceID, UniverseGroup, Ticker, Company, AssetType, Exchange, Sector, Industry string
	Rank                                                                                      int
	IsActive                                                                                  bool
	MetadataJSON                                                                              []byte
	CreatedAt, UpdatedAt                                                                      time.Time
}

// SeedSubscriberGlobalCatalogShadow imports only compatibility-universe metadata.
// It does not enable collection or modify existing tenant MarketOps records.
func (r *Repository) SeedSubscriberGlobalCatalogShadow(ctx context.Context, request storage.SubscriberGlobalCatalogSeedRequest) (storage.SubscriberGlobalCatalogSeedResult, error) {
	request.SourceTenantID, request.SeedRunID = strings.TrimSpace(request.SourceTenantID), strings.TrimSpace(request.SeedRunID)
	request.ActorIdentity, request.CorrelationID = strings.TrimSpace(request.ActorIdentity), strings.TrimSpace(request.CorrelationID)
	if request.SourceTenantID == "" || request.ActorIdentity == "" {
		return storage.SubscriberGlobalCatalogSeedResult{}, errors.New("source tenant id and actor identity are required")
	}
	if request.SeedRunID == "" {
		request.SeedRunID = newSubscriberID("subcatseed")
	}
	if request.ObservedAt.IsZero() {
		request.ObservedAt = time.Now().UTC()
	}
	result := storage.SubscriberGlobalCatalogSeedResult{SeedRunID: request.SeedRunID, SourceTenantID: request.SourceTenantID}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("begin global catalog shadow seed: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `INSERT INTO subscriber_global_catalog_seed_runs (seed_run_id,source_tenant_id,actor_identity,correlation_id) VALUES ($1,$2,$3,$4)`, request.SeedRunID, request.SourceTenantID, request.ActorIdentity, request.CorrelationID); err != nil {
		return result, fmt.Errorf("create global catalog seed run: %w", err)
	}
	assets, err := listSubscriberCatalogSourceAssets(ctx, tx, request.SourceTenantID)
	if err != nil {
		return result, err
	}
	identities := map[string]struct{}{}
	for _, asset := range assets {
		result.SourceRows++
		if asset.IsActive {
			result.ActiveSourceRows++
		}
		id := subscriberGlobalAssetID(asset.SourceID, asset.Ticker)
		identities[id] = struct{}{}
		inserted, err := upsertSubscriberGlobalAsset(ctx, tx, id, asset, request.ObservedAt)
		if err != nil {
			return result, err
		}
		if inserted {
			result.InsertedGlobalAssets++
		}
		if err := upsertSubscriberGlobalSourceLink(ctx, tx, id, asset, request.ObservedAt); err != nil {
			return result, err
		}
		observed, err := insertSubscriberGlobalReferenceObservation(ctx, tx, request, id, asset)
		if err != nil {
			return result, err
		}
		if observed {
			result.ObservedReferences++
		}
	}
	result.DistinctGlobalAssets = len(identities)
	if err := refreshSubscriberGlobalCoverageShadow(ctx, tx, request.ObservedAt); err != nil {
		return result, err
	}
	report, err := json.Marshal(result)
	if err != nil {
		return result, fmt.Errorf("encode global catalog seed report: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE subscriber_global_catalog_seed_runs SET source_rows=$2,active_source_rows=$3,distinct_global_assets=$4,inserted_global_assets=$5,observed_references=$6,completed_at=$7,report=$8::jsonb WHERE seed_run_id=$1`, result.SeedRunID, result.SourceRows, result.ActiveSourceRows, result.DistinctGlobalAssets, result.InsertedGlobalAssets, result.ObservedReferences, request.ObservedAt, string(report)); err != nil {
		return result, fmt.Errorf("complete global catalog seed run: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return result, fmt.Errorf("commit global catalog shadow seed: %w", err)
	}
	result.CompletedAt = request.ObservedAt
	return result, nil
}

func (r *Repository) ListSubscriberGlobalCatalogParity(ctx context.Context, sourceTenantID string, limit int) ([]storage.SubscriberGlobalCatalogParityRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT l.source_tenant_id,l.source_universe_group,l.source_ticker,l.source_is_active,l.global_asset_id,a.canonical_symbol,a.eligibility_status,COALESCE(c.coverage_state,''),COALESCE(c.execution_mode,''),COALESCE(c.active_source_rows,0) FROM subscriber_global_asset_source_links l JOIN subscriber_global_assets a ON a.global_asset_id=l.global_asset_id LEFT JOIN subscriber_global_asset_coverage c ON c.global_asset_id=l.global_asset_id AND c.coverage_product='eod_baseline' WHERE l.source_tenant_id=$1 ORDER BY l.source_is_active DESC,l.source_universe_group,l.source_ticker LIMIT $2`, strings.TrimSpace(sourceTenantID), clampLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list global catalog shadow parity: %w", err)
	}
	defer rows.Close()
	records := []storage.SubscriberGlobalCatalogParityRecord{}
	for rows.Next() {
		var x storage.SubscriberGlobalCatalogParityRecord
		if err := rows.Scan(&x.SourceTenantID, &x.SourceUniverseGroup, &x.SourceTicker, &x.SourceIsActive, &x.GlobalAssetID, &x.CanonicalSymbol, &x.EligibilityStatus, &x.CoverageState, &x.CoverageExecutionMode, &x.ActiveSourceRows); err != nil {
			return nil, fmt.Errorf("scan global catalog shadow parity: %w", err)
		}
		records = append(records, x)
	}
	return records, rows.Err()
}

func listSubscriberCatalogSourceAssets(ctx context.Context, tx *sql.Tx, tenantID string) ([]subscriberCatalogSourceAsset, error) {
	rows, err := tx.QueryContext(ctx, `SELECT tenant_id,source_id,universe_group,rank,ticker,company,asset_type,exchange,sector,industry,is_active,metadata,created_at,updated_at FROM marketops_asset_universe WHERE tenant_id=$1 ORDER BY universe_group,rank,ticker`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list compatibility MarketOps universe: %w", err)
	}
	defer rows.Close()
	assets := []subscriberCatalogSourceAsset{}
	for rows.Next() {
		var x subscriberCatalogSourceAsset
		if err := rows.Scan(&x.TenantID, &x.SourceID, &x.UniverseGroup, &x.Rank, &x.Ticker, &x.Company, &x.AssetType, &x.Exchange, &x.Sector, &x.Industry, &x.IsActive, &x.MetadataJSON, &x.CreatedAt, &x.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan compatibility MarketOps asset: %w", err)
		}
		assets = append(assets, x)
	}
	return assets, rows.Err()
}

func upsertSubscriberGlobalAsset(ctx context.Context, tx *sql.Tx, id string, x subscriberCatalogSourceAsset, observedAt time.Time) (bool, error) {
	provenance, _ := json.Marshal(map[string]any{"schema_version": storage.SubscriberGlobalCatalogShadowVersion, "source": "marketops_asset_universe", "source_tenant": x.TenantID, "source_group": x.UniverseGroup})
	var inserted bool
	err := tx.QueryRowContext(ctx, `INSERT INTO subscriber_global_assets (global_asset_id,source_id,provider_symbol,canonical_symbol,company_name,asset_type,exchange,sector,industry,eligibility_status,reference_effective_at,reference_provenance,first_seen_at,last_seen_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'discovered',$10,$11::jsonb,$12,$12) ON CONFLICT (source_id,provider_symbol) DO UPDATE SET canonical_symbol=EXCLUDED.canonical_symbol,company_name=EXCLUDED.company_name,asset_type=EXCLUDED.asset_type,exchange=EXCLUDED.exchange,sector=EXCLUDED.sector,industry=EXCLUDED.industry,reference_effective_at=EXCLUDED.reference_effective_at,reference_provenance=EXCLUDED.reference_provenance,last_seen_at=EXCLUDED.last_seen_at,updated_at=now() RETURNING (xmax=0)`, id, x.SourceID, x.Ticker, x.Ticker, x.Company, x.AssetType, x.Exchange, x.Sector, x.Industry, x.UpdatedAt, string(provenance), observedAt).Scan(&inserted)
	return inserted, func() error {
		if err != nil {
			return fmt.Errorf("upsert global catalog asset %s: %w", x.Ticker, err)
		}
		return nil
	}()
}

func upsertSubscriberGlobalSourceLink(ctx context.Context, tx *sql.Tx, id string, x subscriberCatalogSourceAsset, observedAt time.Time) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO subscriber_global_asset_source_links (source_tenant_id,source_universe_group,source_ticker,global_asset_id,source_rank,source_is_active,source_metadata,source_created_at,source_updated_at,last_observed_at) VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8,$9,$10) ON CONFLICT (source_tenant_id,source_universe_group,source_ticker) DO UPDATE SET global_asset_id=EXCLUDED.global_asset_id,source_rank=EXCLUDED.source_rank,source_is_active=EXCLUDED.source_is_active,source_metadata=EXCLUDED.source_metadata,source_created_at=EXCLUDED.source_created_at,source_updated_at=EXCLUDED.source_updated_at,last_observed_at=EXCLUDED.last_observed_at`, x.TenantID, x.UniverseGroup, x.Ticker, id, x.Rank, x.IsActive, string(x.MetadataJSON), x.CreatedAt, x.UpdatedAt, observedAt)
	if err != nil {
		return fmt.Errorf("upsert global catalog source link %s: %w", x.Ticker, err)
	}
	return nil
}

func insertSubscriberGlobalReferenceObservation(ctx context.Context, tx *sql.Tx, request storage.SubscriberGlobalCatalogSeedRequest, id string, x subscriberCatalogSourceAsset) (bool, error) {
	payload, _ := json.Marshal(x)
	provenance, _ := json.Marshal(map[string]any{"schema_version": storage.SubscriberGlobalCatalogShadowVersion, "actor_identity": request.ActorIdentity, "correlation_id": request.CorrelationID})
	res, err := tx.ExecContext(ctx, `INSERT INTO subscriber_global_asset_reference_observations (observation_id,global_asset_id,seed_run_id,source_fingerprint,observed_at,source_tenant_id,source_universe_group,source_ticker,reference_payload,provenance) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10::jsonb) ON CONFLICT (source_fingerprint) DO NOTHING`, newSubscriberID("subcatref"), id, request.SeedRunID, subscriberCatalogFingerprint(x), request.ObservedAt, x.TenantID, x.UniverseGroup, x.Ticker, string(payload), string(provenance))
	if err != nil {
		return false, fmt.Errorf("insert global reference observation %s: %w", x.Ticker, err)
	}
	count, err := res.RowsAffected()
	return count == 1, err
}

func refreshSubscriberGlobalCoverageShadow(ctx context.Context, tx *sql.Tx, observedAt time.Time) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO subscriber_global_asset_coverage (global_asset_id,coverage_product,coverage_state,execution_mode,active_source_rows,coverage_version,observed_at,reason_code,provenance) SELECT global_asset_id,'eod_baseline',CASE WHEN count(*) FILTER (WHERE source_is_active)>0 THEN 'active' ELSE 'not_requested' END,'shadow',count(*) FILTER (WHERE source_is_active),$1,$2,'compatibility_universe_observation',jsonb_build_object('schema_version',$1::text,'source','marketops_asset_universe') FROM subscriber_global_asset_source_links GROUP BY global_asset_id ON CONFLICT (global_asset_id,coverage_product) DO UPDATE SET coverage_state=EXCLUDED.coverage_state,execution_mode='shadow',active_source_rows=EXCLUDED.active_source_rows,coverage_version=EXCLUDED.coverage_version,observed_at=EXCLUDED.observed_at,reason_code=EXCLUDED.reason_code,provenance=EXCLUDED.provenance,updated_at=now()`, storage.SubscriberGlobalCatalogShadowVersion, observedAt)
	if err != nil {
		return fmt.Errorf("refresh global catalog coverage shadow: %w", err)
	}
	return nil
}

func subscriberGlobalAssetID(sourceID, ticker string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(sourceID)) + "\x00" + strings.ToUpper(strings.TrimSpace(ticker))))
	return "subglobal_" + hex.EncodeToString(sum[:16])
}
func subscriberCatalogFingerprint(x subscriberCatalogSourceAsset) string {
	payload, _ := json.Marshal(x)
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
