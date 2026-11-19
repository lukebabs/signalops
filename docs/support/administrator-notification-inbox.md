# Administrator notification inbox

## Purpose

The Administrator Inbox is SignalOps operational control-plane telemetry. It is deliberately separate from analyst-facing Alerts, Signals, and domain workflows. It provides a durable, per-administrator work queue for platform events that need operational awareness.

## Initial event policy

- A successful **governed** job creates an informational event: MarketOps post-close, FMP continuation, storage monitoring, and retention governance.
- A failure from **any** scheduler-managed job creates a warning event.
- Routine failures deduplicate by tenant, job, and failure state. Repeated instances increment `occurrence_count`; the third consecutive recorded instance becomes critical.
- The scheduler’s primary exit code always wins. If notification persistence is unavailable, the wrapper logs that secondary failure but does not mask the scheduled job outcome.

Events include the job identifier, schedule, timezone, start/completion times, and exit code as evidence.

## Delivery and user state

Notifications persist in `administration_notifications`. Read/archive state is stored by authenticated administrator subject in `administration_notification_inbox_state`; one administrator’s archive action never hides an event for another.

The System workbench polls the inbox every 15 seconds. It displays unread count, severity, event context, repeat count, observed time, and archive/restore action.

## Scheduler wiring

`scripts/marketops_scheduled_job.sh` records scheduler state in the dedicated MarketOps database first, writes ignored local JSON only as fallback/debug output, then invokes the isolated `administration-notification-recorder` Docker service when policy requires an inbox event. The recorder is built by the scheduler-install path and is supplied the database URL only through Compose.

## Deferred delivery

The migration reserves SMTP settings and delivery-audit storage. SMTP configuration, recipient-resolution policy, encryption-key handling, and email sending are intentionally the next implementation layer. No secret is currently accepted or transmitted by this initial inbox release.

## SMTP settings

Super-admins configure SMTP in **Administration → System → Notification email**. Settings are tenant-scoped. The gateway encrypts a supplied password with AES-256-GCM before persistence; passwords are never returned to the UI. Set `SIGNALOPS_NOTIFICATION_ENCRYPTION_KEY` to a base64-encoded 32-byte key on the gateway before saving a password. The UI retains an existing password when its password field is blank.

The current delivery configuration is intentionally non-sending: it safely establishes stored configuration and the audit schema before recipient resolution and live email delivery are enabled. This prevents a partially configured SMTP endpoint from generating unintended mail.
