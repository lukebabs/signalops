# G133 Bounded Top 50 Options Coverage Expansion

Status: implemented backend/CLI substrate

Date: 2026-07-18

## Purpose

G133 expands MarketOps options coverage beyond the original NVDA slice while keeping provider usage explicit, bounded, and operator-controlled.

The gate adds a single CLI that can process either an explicit symbol list or the active `top50_megacap` asset universe with hard caps. It fetches bounded Massive option-chain snapshots, persists chain rows, derives distribution snapshots, materializes `options_distribution_daily` normalized feature rows, and reports quality counts per symbol.

## Scope

In scope:

- Add `signalops-marketops-options-coverage-runner`.
- Resolve symbols from `--symbols` or `marketops_asset_universe` when symbols are omitted.
- Enforce `--max-symbols`, `--limit`, `--max-pages`, `--chain-scan-limit`, and `--distribution-limit` caps.
- Reuse the existing Massive credentials from env.
- Persist options chain rows, distribution snapshots, and normalized feature rows in write mode.
- Support dry-run mode that fetches and derives metrics without writing storage.
- Report aggregate and per-symbol quality counts.
- Fix derived options feature events so their synthetic raw offsets are stable per event instead of all using `-1`.

Out of scope:

- No scheduler.
- No automatic Top 50 fanout by default.
- No frontend controls.
- No algorithm execution or signal proposal generation.
- No runtime policy deployment.

## Command

Dry-run explicit symbols:

```bash
signalops-marketops-options-coverage-runner   --tenant-id tenant-local   --symbols AAPL,MSFT   --max-symbols 2   --limit 5   --max-pages 1   --window-days 10   --distribution-limit 10   --dry-run
```

Write explicit symbols:

```bash
signalops-marketops-options-coverage-runner   --tenant-id tenant-local   --symbols AAPL,MSFT   --max-symbols 2   --limit 5   --max-pages 1   --window-days 10   --distribution-limit 10
```

Bounded universe mode:

```bash
signalops-marketops-options-coverage-runner   --tenant-id tenant-local   --universe-group top50_megacap   --max-symbols 3   --limit 25   --max-pages 1   --window-days 10   --distribution-limit 25   --dry-run
```

## Validation

Automated validation:

- Focused Go tests passed for:
  - `./cmd/marketops-options-coverage-runner`
  - `./internal/marketops/options`
  - `./cmd/marketops-options-feature-materializer`
  - `./internal/storage/postgres`
  - `./internal/api`
- Docker target build passed for `marketops-options-coverage-runner`; the build ran `go test ./...`.
- JSON schema validation passed.

Live bounded validation:

- Dry-run AAPL/MSFT fetched 10 contracts total, converted 10, built 5 distribution snapshots, and wrote 0 rows.
- Write run AAPL/MSFT fetched 10 contracts total, converted 10, upserted 10 chain rows, upserted 5 distribution snapshots, and upserted 5 normalized feature rows.
- Persisted options chain coverage after the run:
  - `AAPL`: 3 trade days, 5 contracts, first trade date `2026-07-14`, last trade date `2026-07-18`.
  - `MSFT`: 2 trade days, 5 contracts, first trade date `2026-07-17`, last trade date `2026-07-18`.
  - `NVDA`: 27 trade days, 250 contracts, first trade date `2025-12-02`, last trade date `2026-07-17`.
- AAPL/MSFT distribution quality counts:
  - `AAPL`: `all_zero=1`, `denominator_zero=2`.
  - `MSFT`: `all_zero=1`, `denominator_zero=1`.
- AAPL/MSFT normalized feature rows:
  - `AAPL`: 3 rows across 3 trade days.
  - `MSFT`: 2 rows across 2 trade days.

## Result

MarketOps now has a bounded, repeatable operator path to expand options coverage from one symbol to selected Top 50 assets without introducing a scheduler or unbounded provider fanout. The path reports evidence quality up front, so low-quality open-interest data remains visible instead of silently feeding proposals.

## Deferred

- Broader Top 50 runs with larger per-symbol limits.
- Algorithm execution over expanded options feature rows.
- Quality-aware proposal generation across the expanded symbol set.
- Operator scheduling or queueing, if later approved.
