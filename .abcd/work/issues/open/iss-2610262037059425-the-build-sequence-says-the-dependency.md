---
schema_version: 1
id: "iss-2610262037059425"
slug: "the-build-sequence-says-the-dependency"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 696 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/01-build-sequence.md:123"
remedy: "In section 4 narrow the design target to a dependency graph over specs, and say each spec step's needs line is shipped and sequences the build loop's lanes (34-build.md) and an intent carries blocked_by; apply the same narrowing to the Native spec store row of 02-verification-matrix.md."
finding_ref: cfk-e71c7ec81e46e25d
---

The build sequence says the dependency graph and the sequencing over it are a design target; the build loop sequences a spec's steps by their declared needs.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/01-build-sequence.md:123` — "The dependency graph and the sequencing over it are a **design target**; readiness today is per-intent, through `abcd intent ready`."
- End B: `.abcd/development/brief/04-surfaces/34-build.md:417` — "A lane opens for a spec step once every step it needs has landed (its `- needs:` line, or by default every step before it, ruling DR6b)"

01-build-sequence.md says the dependency graph over specs and tasks and the sequencing over it are a design target, with readiness today per-intent through `abcd intent ready`. 34-build.md says a lane opens for a spec step only once every step it needs has landed, read from each step's `- needs:` line: a dependency graph over a spec's steps, with sequencing over it, that ships. One of the two describes the shipped state wrongly.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 696.
