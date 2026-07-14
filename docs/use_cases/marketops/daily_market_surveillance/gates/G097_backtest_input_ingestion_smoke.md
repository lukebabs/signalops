# G097 Back-Test Input Ingestion Smoke

Status: implemented operator script and Compose wiring; live provider execution blocked locally by missing Massive API key.

## Purpose

G096 proved that G095 campaigns are empty because no normalized MarketOps rows exist locally. G097 adds the bounded operator path for producing a small data-bearing input sample through the existing Massive puller and normalizer pipeline.

## Implemented Scope

- Pass database and temporal database URLs into the one-shot `massive-puller` Compose service so published raw events can be persisted when storage is configured.
- Add `scripts/marketops_calibration_ingest_smoke.sh`, a guarded smoke script that:
  - sources `.env`;
  - fails before provider access if no explicit Massive API key is present;
  - starts existing normalizer/worker/persister services;
  - runs the existing Massive puller with one-company/one-event publish bounds by default, independent of broader `SIGNALOPS_MASSIVE_*` defaults in `.env`.
- Document the runbook under MarketOps operations.

## Out Of Scope

- Synthetic data generation.
- Direct normalized ledger inserts.
- Runtime policy deployment.
- Detector mutation.
- Graph writes.
- Unbounded historical ingestion campaigns.

## Local Validation

- Shell syntax validation passed for the smoke script.
- Compose validation should pass with the `massive-pull` profile.
- Live provider execution requires `SIGNALOPS_MASSIVE_API_KEY` or `MASSIVE_API_KEY`; generic `API_KEY` is ignored by the smoke script. A provider `401` indicates the configured key is present but invalid for Massive.
