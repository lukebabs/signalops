# Frontend Notes

Use this folder for MarketOps Daily Market Surveillance UI semantics and operator validation notes.

Current primary views:

- `/marketops/assets`: Top 50 mega-cap asset universe.
- `/marketops/dsm`: DSM Workbench for taxonomy signals and artifact visibility.

DSM Workbench Ledger column semantics:

- `persisted`: the signal has a first-class DSM artifact record in `marketops_dsm_artifacts`.
- `signal-only`: the signal may still include artifact proposal data in semantic evidence, but no first-class artifact ledger row was found by the frontend query.

Operator validation for G078 requires signing in, opening `/marketops/dsm`, selecting a row with `persisted`, and confirming the `First-Class Artifact Ledger` panel renders live artifact data.

Current notes:

- `dsm_workbench_operator_validation.md`: authenticated operator checklist and failure triage.
- `../../../../frontend/marketops_graph_proposals_readonly_spec.md`: frontend-agent handoff for G079 read-only graph proposal visibility.
- `../../../../frontend/marketops_backtests_ui_spec.md`: frontend-agent handoff for G081 isolated back-test run creation and review UI.
- `../../../../frontend/syncratic_context_windows_ui_spec.md`: frontend-agent handoff for G089 Syncratic context windows and synthesized insights UI.
- `../../../../frontend/syncratic_ask_quality_ui_spec.md`: frontend-agent handoff for G090 Ask metadata, data-quality warning, and operator-triggered Ask UI.
- `../../../../frontend/syncratic_materialization_preview_ui_spec.md`: frontend-agent handoff for G092 budgeted materialization preview and confirmed write UI.
- `../../../../frontend/marketops_asset_options_distribution_ui_spec.md`: frontend-agent handoff for G128 asset-level options coverage, distribution, and chain-row UI.
- `../../../../frontend/marketops_options_quality_visibility_ui_spec.md`: frontend-agent handoff for G132 options ratio/open-interest quality visibility in asset options and algorithm proposal views.
- `../../../../frontend/marketops_intelligence_readiness_ui_spec.md`: implemented G148 read-only aggregate intelligence readiness view under MarketOps Assets.
