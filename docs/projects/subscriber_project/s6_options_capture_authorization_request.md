# S6 Options Capture Authorization Request

Migration `000114` adds a reviewable, append-only request for the disabled one-asset capture gate. The request is always `pending_approval`, retains a one-request Massive budget, and succeeds only while its source gate remains disabled with the kill switch engaged.

It is not an execution authorization. It creates no capture run, provider intent, worker, schedule, credential scope, or feature-flag change. A later separately reviewed release would need to consume this request through a named approver, create a one-time authorization, and remain subject to the captured recovery and rollback gates.
