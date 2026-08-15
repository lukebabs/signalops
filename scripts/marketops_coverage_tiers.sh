#!/usr/bin/env bash
# Shared read-only selectors for the centrally governed MarketOps coverage tiers.
# Callers must source marketops_schedule_database.sh first.

marketops_warm_eod_symbols() {
  marketops_primary_psql -Atc "SELECT string_agg(canonical_symbol, ',' ORDER BY priority, global_asset_id) FROM subscriber_global_warm_eod_assets;" | tr -d '[:space:]'
}

marketops_hot_intraday_symbols() {
  marketops_primary_psql -Atc "SELECT string_agg(canonical_symbol, ',' ORDER BY canonical_symbol, global_asset_id) FROM subscriber_global_hot_intraday_assets;" | tr -d '[:space:]'
}
