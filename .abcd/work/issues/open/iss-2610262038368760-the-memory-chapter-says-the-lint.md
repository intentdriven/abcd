---
schema_version: 1
id: "iss-2610262038368760"
slug: "the-memory-chapter-says-the-lint"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 806 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/07-memory.md:169"
remedy: "Edit 07-memory.md section 7 (library lint row): cite 07-library.md's exit contract (only blockers exit nonzero) instead of a divergence 06-lint.md section 2 records; 06-lint covers the two tree gates, not the memory lint."
finding_ref: cfk-85eb0d91ee3d9e03
---

The memory chapter says the lint contract §2 records library lint's warn-only exit-0 divergence; §2 records no divergence and says a warning raises the exit to 1.

A sequencing impossibility (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/07-memory.md:169` — "the recorded divergence from the standalone grammar"
- End B: `.abcd/development/brief/05-internals/06-lint.md:34` — "Raises the exit code to 1 on an otherwise-clean run"

07-memory gives `/abcd:library lint` a warn-only exit of 0 and calls it the recorded divergence from the standalone grammar, citing 06-lint §2. 06-lint §2 records no such divergence. Its severity model says a `warn` raises the exit code to 1 on an otherwise-clean run, so the record 07-memory depends on does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 806.
