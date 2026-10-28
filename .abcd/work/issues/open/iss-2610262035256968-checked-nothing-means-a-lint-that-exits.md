---
schema_version: 1
id: "iss-2610262035256968"
slug: "checked-nothing-means-a-lint-that-exits"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 571 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/10-docs.md:39"
remedy: "none (filed automatically)"
finding_ref: cfk-e828c45a707bfded
---

'Checked nothing' means a lint that exits 0 in the docs chapter and a refused check that must never exit 0 in the lint chapter

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/10-docs.md:39` — "A lint that checked nothing (no rule armed,"
- End B: `.abcd/development/brief/04-surfaces/16-lint.md:35` — "exits 2 as the target alone does, never `0` over a check that never ran"

10-docs uses 'a lint that checked nothing' for a run with no rule armed or roots holding no document, and keeps its exit at 0. 16-lint uses 'checked nothing' for a target that refused, raises an error and exits 2, 'never 0 over a check that never ran'. The same phrase names two states with opposite exit codes, so a reader cannot tell whether a docs lint with nothing armed is a check that never ran.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 571.
