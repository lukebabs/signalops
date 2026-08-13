# S6 Options-Demand Login Evidence — 2026-08-13

Status: dedicated workload login provisioned and preflight passed. Options demand remains default-deny; no snapshot, capture, provider call, or scheduler is enabled.

The database principal `signalops_subscriber_options_demand_runner` was created with `LOGIN NOINHERIT`, no administrative attributes, and membership only in `signalops_subscriber_options_demand`.

Its real password-authenticated preflight passed from an ephemeral PostgreSQL client container. The session used the explicit role boundary:

```text
PGOPTIONS=-c role=signalops_subscriber_options_demand
```

That role selection is required because the login is intentionally `NOINHERIT`. The deployment secret must retain both the runner DSN and this runtime setting. It must not share a login, credential, or role session with the gateway, global-EOD reconciler, or Options-capture worker.

The preflight verified aggregate-only input, append-only shadow-storage access, and absence of direct subscriber/list/Options-capture-table privileges. It made no provider request and wrote no snapshot.
