# S6 Pilot Options-Demand Shadow Evidence — 2026-08-13

Status: bounded pilot demand policy active; shadow planning passed. Options capture remains disabled.

`tenant-pilot-b` retains `subscriber-list-pilot-v1` and now has an explicit `options_demand` entitlement of ten distinct eligible global assets. The controlled provisioner retained catalog-search 50 and EOD-activation 10 unchanged.

The first non-zero planner run (`suboptdemand_1e5195e7ffc8338f2e03ace5`) saw twelve watcher contributions resolving to ten quota-eligible global assets. With a five-asset shadow capacity, it selected NVDA, AAPL, TSLA, META, and BRK.B; AVGO, GOOGL, SPCX, AMZN, and MSFT were deferred.

A repeat five-asset run (`suboptdemand_a35104af4805f1578bf3d697`) verified carry-forward fairness. NVDA and AAPL remained first because they each had two eligible watchers. The prior one-watcher deferred assets AVGO, GOOGL, and SPCX received deferred age one and moved ahead of AMZN/MSFT and the newly deferred TSLA/META/BRK.B.

The pilot tenant has zero rows in `marketops_options_capture_sessions`. Neither run contacted a provider, invoked an Options-capture worker, or started a scheduler. The entitlement permits demand contribution to the aggregate shadow plan only.
