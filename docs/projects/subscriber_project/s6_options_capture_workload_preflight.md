# S6 Options-Capture Workload Preflight

Run through a dedicated `LOGIN NOINHERIT` principal that is a member only of `signalops_subscriber_options_capture`:

```bash
PGOPTIONS='-c role=signalops_subscriber_options_capture' \
SIGNALOPS_SUBSCRIBER_WORKLOAD_IDENTITY=subscriber-options-capture \
SIGNALOPS_SUBSCRIBER_OPTIONS_CAPTURE_DATABASE_URL='<secret-managed dedicated DSN>' \
  ./scripts/subscriber_project_options_capture_workload_preflight.sh
```

The check is read-only. It requires aggregate snapshot/global-identity reads, append-only gate records, and no direct list, entitlement, or Options evidence access. A pass cannot enable a provider call or capture worker.
