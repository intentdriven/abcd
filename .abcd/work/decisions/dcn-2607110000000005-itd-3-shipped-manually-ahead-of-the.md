---
id: dcn-2607110000000005
slug: itd-3-shipped-manually-ahead-of-the
date: 2026-07-11
---
2026-07-11 — itd-3 **shipped manually** ahead of the intent-lifecycle pipeline.
Moved `planned/ → shipped/` by hand with `spec_id: spc-1` (reserved — the future
native spec store adopts spc-1 for itd-3, never re-mints it) and a hand-authored
`## Audit Notes` (the `intent-fidelity-reviewer` agent does not exist yet; judge
= Claude Opus 4.8). Rollup 3 MET / 1 MET_WITH_CONCERNS / 1 INCONCLUSIVE / 1
NOT_MET; every divergence is a signed-off D1–D4 delta, the one gap is the AC6
legacy-harvest completeness. Inbound links repointed planned→shipped by hand —
the link-drift-on-move the future reconcile pass automates.
