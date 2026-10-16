---
id: dcn-2607140000000003
slug: the-brief-lifeboat-mapping-table-now
date: 2026-07-14
---
2026-07-14 — The brief↔lifeboat mapping table now EXISTS. `00-meta.md` has always
called it "the contract" while no such table existed anywhere (found by the
2026-07-06 plan-consistency review). It lands as Go — `internal/core/lifeboat/
mapping.go` is the single source of truth — and is rendered into `00-meta.md`
between generated markers, with a test asserting the two agree so the document
cannot drift from the code. It is framed as the experiment's HYPOTHESIS, stating
the best status each brief section could reach at each source tier, in the SAME
three-valued vocabulary the probe reports (`grounded`/`partial`/`blank`) so
prediction and evidence are directly comparable. M2 is expected to revise it.
A monotonicity test (a richer tier can never ground a section worse than a poorer
one — tiers are CUMULATIVE) caught a real error in the first draft of the table.
