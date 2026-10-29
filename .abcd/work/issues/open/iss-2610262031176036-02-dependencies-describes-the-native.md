---
schema_version: 1
id: "iss-2610262031176036"
slug: "02-dependencies-describes-the-native"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 110 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/02-dependencies.md:39"
remedy: "In the Spec / task bullet of 02-dependencies.md, mark the dependency graph over specs and tasks as staged and keep directories-encode-status as the shipped default, as 06-delivery/02-verification-matrix.md states ('the spec core carries no graph')."
finding_ref: cfk-f764c997e226cc19
---

02-dependencies describes the native spec store as directories plus a dependency graph; the verification matrix stages the graph and says the spec core carries none.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/02-dependencies.md:39` — "directories whose location encodes status, plus a dependency graph over them."
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:33` — "the spec core carries no graph"

02-dependencies gives the spec/task native default as 'directories whose location encodes status, plus a dependency graph over them.' with no staged mark. The matrix's Native spec store row marks as staged the dependency graph over specs and tasks and the sequencing that resolves ready work from it, and says 'the spec core carries no graph'. The store either has a dependency graph or it does not.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 110.
