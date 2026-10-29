---
schema_version: 1
id: "iss-2610262035524537"
slug: "the-internals-index-says-the-check-gates"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 604 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/README.md:16"
remedy: "In the index row for 10-site.md say the abcd lint site gates and the explorer pages ship, not 'the check gates'."
finding_ref: cfk-b3ce36a04af0243c
---

The internals index says 'the `check` gates' ship; the lint chapter says the former check sub-verbs are unknown commands

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/README.md:16` — "The `check` gates and the explorer pages ship"
- End B: `.abcd/development/brief/04-surfaces/16-lint.md:44` — "The docs and site verbs' former check sub-verbs are unknown commands"

The 05-internals index row for 10-site.md says 'The `check` gates and the explorer pages ship', naming the gates by a `check` verb. The lint chapter says the docs and site verbs' former check sub-verbs are unknown commands, because the site gates are now the site target of `abcd lint`. The index still uses the retired name.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 604.
