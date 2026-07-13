# Syncratic Context Window Operations

Status: proposed
Use case: MarketOps Daily Market Surveillance

## Validation Goals

When G088 is implemented, operators should validate that Syncratic context windows and synthesized insights are derived from existing ledgers only and do not mutate alert lifecycle, graph proposal, signal, alert, or insight rows unexpectedly.

## Smoke Shape

1. Ensure at least two related MarketOps alerts or signals exist for one symbol and strategy window.
2. Create a Syncratic context window for that symbol and strategy.
3. Verify the context window includes supporting alert/signal/event ids.
4. Create a synthesized insight from the context window.
5. Fetch list/detail APIs.
6. Confirm deterministic builder metadata and evidence digest are present.
7. Confirm no production graph writes or alert lifecycle mutations occurred.

## Safety Rules

- Do not add external source ingestion in the MVP smoke.
- Do not use LLM-generated narrative as acceptance evidence.
- Do not treat a single alert clone as a valid synthesized insight for default strategies.
- Do not suppress legacy one-signal insights until a separate migration gate is approved.
