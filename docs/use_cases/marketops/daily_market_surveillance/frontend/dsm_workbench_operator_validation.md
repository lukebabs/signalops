# DSM Workbench Operator Validation

Status: current as of G078  
Route: `/marketops/dsm`

## Purpose

This checklist validates that the MarketOps DSM Workbench is rendering both persisted signals and first-class DSM artifact ledger records correctly under an authenticated operator session.

## Prerequisites

- Web has been rebuilt/deployed with the G078 bundle, for example with `make deploy-web`.
- Gateway auth is enabled.
- Operator can sign in and obtain a valid bearer token through the browser flow.
- At least one MarketOps DSM signal has a materialized artifact row in `marketops_dsm_artifacts`.

Known local smoke record from G077:

- Signal: `sig_marketops_dsm_taxonomy_v1_g077_artifact_live`
- Artifact: `artifact_marketops_dsm_v1_g077_live`
- Subject: `AAPL`
- Signal type: `marketops.dsm.pinning_risk`

## Validation Steps

1. Open the web app and sign in.
2. Select `MarketOps` in the app selector if it is not already active.
3. Open `/marketops/dsm`.
4. Confirm the network request to `/v1/marketops/dsm/artifacts` succeeds with `Authorization: Bearer ...`.
5. Confirm the `DSM Artifacts` metric tile renders a live count.
6. Find a row whose `Ledger` column says `persisted`.
7. Click that row.
8. Confirm the right-side detail panel shows `First-Class Artifact Ledger`.
9. Confirm the panel displays real ledger values, including artifact id, subject, artifact type, updated timestamp, event count, and quality issues.
10. Confirm rows without a matching artifact, if present, show `signal-only` instead of `persisted`.

## Expected Labels

`persisted` means the selected signal has a first-class artifact record in `marketops_dsm_artifacts`.

`signal-only` means the signal exists in `signal_ledger`, but no matching first-class artifact row was returned by the artifact API query.

Do not interpret `persisted` as the signal persistence state. Signals shown in this workbench are already persisted signal-ledger records.

## Failure Triage

If no `persisted` rows are visible:

- Set taxonomy filter to `all taxonomy`.
- Set severity to `any severity`.
- Set dataset to `any dataset`.
- Set limit to `100` or `200`.
- Refresh the page.
- Verify `GET /v1/marketops/dsm/artifacts` returns artifact rows for the same tenant/use case.

If the artifact API returns `401`, the browser session does not have a usable bearer token or the request is being made before login completes.

If the detail panel shows `signal-only` for a signal expected to be persisted, verify the artifact row exists in `marketops_dsm_artifacts` and that the artifact API response includes the matching `signal_id` or `artifact_id`.
