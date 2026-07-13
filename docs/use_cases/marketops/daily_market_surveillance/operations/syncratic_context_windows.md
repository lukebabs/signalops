# Syncratic Context Window Operations

Status: proposed
Use case: MarketOps Daily Market Surveillance

## Validation Goals

When G088 is implemented, operators should validate that Syncratic context windows and synthesized insights are derived from existing ledgers only and do not mutate alert lifecycle, graph proposal, signal, alert, or insight rows unexpectedly.

## Smoke Shape

1. Ensure at least two related MarketOps alerts or signals exist for one symbol and strategy window.
2. Run the aggregate candidate scan for the MarketOps Top 50 universe.
3. Verify the related symbol crosses the materialization threshold.
4. Verify at least one quiet Top 50 symbol is scanned but not materialized.
5. Create or materialize a Syncratic context window for the threshold-crossing symbol and strategy.
6. Verify the context window includes supporting alert/signal/event ids.
7. Create a synthesized insight from the context window.
8. Rerun the same build and verify unchanged evidence digest skips a duplicate write.
9. Fetch list/detail APIs.
10. Confirm deterministic builder metadata, idempotency key, evidence digest, and materialization metrics are present.
11. Confirm no production graph writes or alert lifecycle mutations occurred.

## Safety Rules

- Do not add external source ingestion in the MVP smoke.
- Do not use LLM-generated narrative as acceptance evidence.
- Do not treat a single alert clone as a valid synthesized insight for default strategies.
- Do not suppress legacy one-signal insights until a separate migration gate is approved.
- Do not materialize context windows or insights for every Top 50 asset by default.
- Do not persist preview-only Syncratic output unless it passes the same threshold, digest, idempotency, and audit path.
