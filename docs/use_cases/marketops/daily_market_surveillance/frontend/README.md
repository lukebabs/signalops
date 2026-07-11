# Frontend Notes

Use this folder for MarketOps Daily Market Surveillance UI semantics and operator validation notes.

Current primary views:

- `/marketops/assets`: Top 50 mega-cap asset universe.
- `/marketops/dsm`: DSM Workbench for taxonomy signals and artifact visibility.

DSM Workbench Ledger column semantics:

- `persisted`: the signal has a first-class DSM artifact record in `marketops_dsm_artifacts`.
- `signal-only`: the signal may still include artifact proposal data in semantic evidence, but no first-class artifact ledger row was found by the frontend query.

Operator validation for G078 requires signing in, opening `/marketops/dsm`, selecting a row with `persisted`, and confirming the `First-Class Artifact Ledger` panel renders live artifact data.
