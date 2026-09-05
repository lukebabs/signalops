# S6 Options Capture Named Approval

Migration `000115` preserves a human approval statement separately from execution authority. An approval retains the named approver, exact provider, one-request limit, zero-retry limit, statement, correlation, and timestamp.

The only current approval state is `approved_pending_recovery`. It is accepted only while the review request remains pending and the capture gate remains disabled with its kill switch engaged. It cannot authorize or invoke the provider. The pending recovery state is authoritative until off-host backup/WAL and isolated-restore evidence has satisfied the production recovery runbook.
