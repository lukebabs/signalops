# K8S-2 OpenBao app staging role handoff

Status: script prepared; live OpenBao role/path execution pending admin-token handoff.

Recorded: 2026-09-12.

## Purpose

Before the staging `web` and `gateway` pods can be applied, OpenBao must prove that the app-plane service accounts can read only the staging app runtime path and that another plane cannot read that path.

This is required because the K8S-2 gateway manifest sources its runtime configuration from:

```text
signalops/data/k8s/app/signalops-gateway-runtime-staging
```

## Prepared automation

The prepared script is:

```bash
scripts/provision_openbao_signalops_app_staging.sh
```

It requires an OpenBao admin token through environment only:

```bash
export BAO_TOKEN='<openbao-admin-token>'
scripts/provision_openbao_signalops_app_staging.sh
unset BAO_TOKEN
```

Do not pass the token as a command-line argument.

## What the script does

The script is constrained to the staging path and placeholder values:

- verifies the OpenBao pod exists;
- verifies the `signalops-app` and `signalops-marketops` namespaces exist;
- verifies the relevant staging service accounts exist;
- creates short-lived Kubernetes service-account tokens for proof only;
- enables the `signalops` KV v2 mount only if missing;
- enables/configures Kubernetes auth only if needed;
- writes the app staging read policy;
- writes a marketops deny-proof policy that does not include the app path;
- writes the `signalops-app` Kubernetes auth role;
- writes the `signalops-marketops` deny-proof Kubernetes auth role;
- writes placeholder-only staging runtime values to the app staging path;
- proves the app role can read the app staging path;
- proves the marketops deny-proof role cannot read the app staging path.

## Expected success output

```text
openbao_signalops_app_staging_verified
mount=signalops
app_role=signalops-app
app_namespace=signalops-app
app_service_accounts=signalops-gateway,signalops-web,signalops-app-secret-reader
secret_path=signalops/k8s/app/signalops-gateway-runtime-staging
deny_role=signalops-marketops
deny_namespace=signalops-marketops
cross_plane_denied=true
secret_values=placeholder_only
production_cutover_allowed=false
```

## Evidence so far

- The host does not have `bao` or `vault` installed.
- The OpenBao pod has `/usr/bin/bao`.
- OpenBao status is initialized, unsealed, HA-enabled, and active as a single node.
- The required SignalOps app-plane service accounts exist.
- The script passes shell syntax validation.
- The script fails closed when no `BAO_TOKEN`, `VAULT_TOKEN`, or `OPENBAO_TOKEN` is present.
- No OpenBao mutation was performed during script preparation because no admin token was available in the execution environment.

## Boundary

This handoff does not authorize production workload cutover, production DNS changes, production provider polling, or production secret migration. It only prepares the staging app secret-role proof required before applying staging `web` and `gateway` pods.
