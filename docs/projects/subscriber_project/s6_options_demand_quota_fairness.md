# S6 Options-Demand Quota and Fairness

Migration `000112` replaces the aggregate-only projection with a quota-aware version. It remains provider-free and does not reveal tenant, list, or subject data to the worker.

For each entitled tenant, the numeric `options_demand` quota is the maximum number of distinct eligible global assets that may contribute to one shadow plan. Candidate selection within that tenant is stable by global asset ID. Multiple lists or watchers of the same asset consume one tenant quota unit.

The projection carries forward an asset's deferred age only when its latest shadow-plan state was `deferred`; a later selected state resets that age. Cross-tenant rank remains tier, eligible-tenant count, eligible-watcher count, deferred age, then global asset ID.

The policy provides reproducibility and prevents an enabled capability with a quota of 1 from contributing an unlimited watchlist. It neither enables an entitlement nor authorizes Options capture.
