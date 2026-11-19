DELETE FROM signal_validation_contracts
WHERE contract_id IN (
  'saf_contract_live_change_point_candidate_v1_bullish',
  'saf_contract_live_change_point_candidate_v1_bearish'
);
DELETE FROM schema_migrations WHERE version='000173_marketops_signal_assurance_live_contracts';
