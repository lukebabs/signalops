-- Provider-backed technicals are independent of FMP financial polling.
UPDATE algorithm_definitions
SET description='Deterministic TTM valuation composite: cached FMP financials are refreshed only by explicit scheduled financial jobs; completed-session Massive SMA/RSI inputs are persisted separately for technical context.',
    input_features=ARRAY['marketops_fundamental_snapshot','massive_rsi_14','massive_sma_50','massive_sma_200'],
    metadata='{"research_only":true,"marketops_role":"valuation_composite","financial_refresh":"explicit_weekly_fmp","technical_provider":"massive"}'::jsonb,
    updated_at=now()
WHERE tenant_id='tenant-local' AND algorithm_id='signalops.algorithms.valuation_composite_v3';

UPDATE algorithm_definitions
SET description='Deterministic strategic opportunity composite using retained FMP TTM financials and provider-backed completed-session Massive RSI/SMA context; ordinary tactical refreshes do not poll FMP.',
    input_features=ARRAY['marketops_fundamental_snapshot','valuation_composite_v3','massive_rsi_14','massive_sma_50','massive_sma_200'],
    metadata='{"research_only":true,"marketops_role":"distressed_opportunity_scoring","financial_refresh":"explicit_weekly_fmp","technical_provider":"massive"}'::jsonb,
    updated_at=now()
WHERE tenant_id='tenant-local' AND algorithm_id='signalops.algorithms.distressed_opportunity_scoring_v3';

UPDATE algorithm_definitions
SET description='Daily deterministic EOD technical overlay using Massive provider-backed RSI-14, SMA-50, and SMA-200 plus canonical five-day price extension. It does not invoke FMP or alter strategic VC/DOSM scores.',
    input_features=ARRAY['massive_rsi_14','massive_sma_50','massive_sma_200','marketops_equity_eod'],
    metadata='{"research_only":true,"marketops_role":"tactical_posture","technical_provider":"massive","financial_polling":"none"}'::jsonb,
    updated_at=now()
WHERE tenant_id='tenant-local' AND algorithm_id='signalops.algorithms.tactical_market_posture_v1';
