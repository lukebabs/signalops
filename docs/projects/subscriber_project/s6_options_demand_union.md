# Sprint S6 — Options Demand Union

Status: internal shadow-planner foundation. No Options entitlement has changed, and no planner, capture worker, scheduler, or provider call is enabled.

`internal/subscriber/optionsdemand` reduces an already-authorized immutable watcher snapshot to one non-identifying candidate per global asset. It retains only the global asset ID, highest applicable tier, eligible tenant count, eligible watcher count, and deferred-session age.

Ranking is deterministic: highest tier, tenant reach, watcher reach, deferred age, then global asset ID. A bounded plan returns selected and deferred global assets; it does not expose one tenant's memberships to another and it cannot create a provider request.

The next internal slice will add append-only snapshot and plan storage under the dedicated `subscriber-options-demand-planner` identity. Only after that shadow evidence, least-privilege preflight, entitlement approval, and a separate capture-worker gate can one selected asset cause one centrally owned capture.
