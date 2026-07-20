# Historical Research Pipeline Operations

Use G141 in two explicit phases: acquire bounded equity history through the canonical Massive raw/normalized path, then run strict persisted-data preflight and research materialization.

## Prerequisites

- A configured SIGNALOPS_MASSIVE_API_KEY, MASSIVE_API_KEY, or API_KEY.
- Healthy Redpanda and normalizer services.
- Configured SignalOps relational and temporal database URLs.
- Applied G136-G140 migrations.
- Persisted option snapshots and distributions for the target sessions.

Do not replace an existing valid Massive key. The puller uses the existing credential precedence.

## Bounded AAPL Equity Acquisition

    docker compose --profile massive-pull run --rm massive-puller \
      --symbols AAPL \
      --datasets equity \
      --start-date 2026-01-02 \
      --end-date 2026-07-17 \
      --max-observation-days 150 \
      --max-companies 1 \
      --max-provider-requests 180 \
      --max-events-built 150 \
      --max-events-published 150 \
      --request-delay 250ms \
      --max-retries 1 \
      --continue-on-error=true \
      --dry-run=false

The range is inclusive and skips weekends. Exchange holidays remain visible as provider failures. Events still pass through raw topics, the normalizer, and the normalized ledger.

The Massive puller's options dataset is contract-reference history. It must not be treated as historical IV, Greeks, open-interest, or quote snapshots.

## Strict Preflight And Run

    signalops-marketops-history-runner \
      --tenant-id tenant-local \
      --symbol AAPL \
      --session-start 2026-01-02 \
      --session-end 2026-07-18 \
      --as-of 2026-07-20 \
      --max-sessions 150 \
      --minimum-equity-sessions 60 \
      --minimum-options-sessions 20 \
      --run-id g141-aapl

The session end is exclusive. The command writes only when all strict coverage checks pass. Review coverage readiness, blockers, hypothesis-specific warnings, equity/option session counts, eligible and triggered evaluations, opportunities, and outcome status counts.

## Diagnostic Partial Run

    signalops-marketops-history-runner \
      --tenant-id tenant-local \
      --symbol AAPL \
      --session-start 2026-01-02 \
      --session-end 2026-07-18 \
      --as-of 2026-07-20 \
      --max-sessions 150 \
      --minimum-equity-sessions 60 \
      --minimum-options-sessions 20 \
      --run-id g141-aapl-diagnostic \
      --allow-insufficient-coverage \
      --dry-run

The override is valid only with dry-run. It exists to inspect state counts and rejection reasons; it cannot write partial research rows.

## Options Coverage Closeout

Current historical option analytics cannot be reconstructed from contract-reference rows. Build valid coverage through one of these governed paths:

1. Retain bounded daily option-chain snapshots prospectively using the existing options coverage runner.
2. Add an approved historical analytics source that supplies point-in-time IV, Greeks, open interest, and quotes with source timestamps and licensing.
3. Use historical bars/quotes only for the fields they contain; keep unavailable OI and Greeks null.

Do not launch calibration or promotion based on the current zero-trigger live sample.
