---
schema_version: 1
id: "iss-2610262037021391"
slug: "the-step-glossary-says-the-build-loop"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 693 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/step.md:18"
remedy: "In step.md say a spec lands its steps one lane at a time by default, and that steps declaring '- needs:' may run in parallel up to the run's ceiling (rulings DR6 and DR6b); commands/build.md line about the next pending step opening the next lane gets the same qualifier."
finding_ref: cfk-9aede0d48387e65d
---

The step glossary says the build loop takes a spec's steps one lane at a time; the build chapter runs lanes in parallel up to a ceiling.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/step.md:18` — "the build loop (`abcd build`) takes them one lane at a time, the spec's next pending step opening the next lane when a lane is done"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:406` — "A run works in parallel up to its ceiling (ruling DR6, spc-2609202134341288)."

step.md says `abcd build` takes a spec's steps one lane at a time, the next pending step opening the next lane only when a lane is done. 34-build.md says a run works in parallel up to its ceiling (ruling DR6), so several of a spec's steps can be in lanes at once. A reader building from the glossary would serialise what the loop runs in parallel.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 693.
