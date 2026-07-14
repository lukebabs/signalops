# G098 Massive Credential Preflight

Status: implemented operator script and smoke integration.

## Purpose

G097 showed the configured `SIGNALOPS_MASSIVE_API_KEY` is present but rejected by Massive with HTTP `401`. G098 adds a credential preflight that fails before Compose services are started, keeping invalid-key validation separate from ingestion pipeline validation.

## Implemented Scope

- `scripts/marketops_massive_credential_preflight.sh` validates an explicit Massive key with a single bounded provider request.
- The preflight ignores generic `API_KEY` to avoid accidental use of unrelated credentials.
- `scripts/marketops_calibration_ingest_smoke.sh` invokes preflight before starting `normalizer`, `raw-worker`, `signal-persister`, or `massive-puller`.
- Operators can bypass preflight only with `MARKETOPS_INGEST_SKIP_PREFLIGHT=true`.

## Out Of Scope

- Credential provisioning.
- Synthetic data generation.
- Direct ledger inserts.
- Runtime policy deployment or detector mutation.

## Validation

- Shell syntax validation should pass for both scripts.
- With the current local credential, preflight is expected to fail with HTTP `401`; that means no ingestion services are started and no provider pull is attempted.
