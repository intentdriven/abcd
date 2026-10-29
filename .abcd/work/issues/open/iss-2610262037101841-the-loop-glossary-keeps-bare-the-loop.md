---
schema_version: 1
id: "iss-2610262037101841"
slug: "the-loop-glossary-keeps-bare-the-loop"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 704 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/loop.md:50"
remedy: "In loop.md, after registering the implement loop, allow bare 'the loop' inside chapters whose subject is the implement loop once it has been named there (34-build.md, 27-implement.md, commands/build.md), and keep the full name everywhere else."
finding_ref: cfk-0cbe8fd62b187dc2
---

The loop glossary keeps bare "the loop" for the record loop; the build chapter uses it bare for its run iteration.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/loop.md:50` — "for the record loop, and only where no run loop or round-trip is in view"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:694` — "Only the loop writes a verdict (decision 9)"

loop.md allows bare "the loop" for the record loop only where no run loop or round-trip is in view, and requires the other senses to be named in full. 34-build.md, whose subject is the build run's own loop, writes "Only the loop writes a verdict", the bare form for the run iteration.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 704.
