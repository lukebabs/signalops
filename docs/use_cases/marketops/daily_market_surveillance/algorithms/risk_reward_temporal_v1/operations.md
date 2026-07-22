# Operations

Run after Market State materialization through `scripts/marketops_algorithm_corroboration.sh`. The schedule executes once per active asset using a 400-calendar-day feature window so the 252-session range can mature.

A failed asset is counted without preventing subsequent assets. No output is produced when no persisted usable state observations exist. Backfills use the same runner with explicit symbol and date window.
