---
schema_version: 1
id: "iss-2610262037018200"
slug: "the-loop-glossary-says-abcd-never-owns"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 691 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/loop.md:38"
remedy: "none (filed automatically)"
finding_ref: cfk-fa10e0c80a5b36e6
---

The loop glossary says abcd never owns the run loop; the build chapter ships abcd's own implement loop.

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/loop.md:38` — "abcd never owns the run loop."
- End B: `.abcd/development/brief/04-surfaces/34-build.md:5` — "It starts the implement loop: a loop over a state file, not a model, where every invocation reads the state, does at most one stage, writes the state and exits."

loop.md says abcd never owns the run loop (the iteration that picks up ready work and gates each step on a receipt): abcd defines the contract and a pluggable adapter supplies the loop. 34-build.md says `abcd build` starts abcd's own implement loop, a loop over a state file that the binary advances one stage per invocation. The glossary rule and the shipped loop cannot both hold.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 691.
