---
schema_version: 1
id: "iss-2610262031042857"
slug: "the-scope-page-says-blocked-by-and"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 81 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/04-scope.md:40"
remedy: "Edit 04-scope.md line 40: `blocked_by` and the readiness gate decide when a lane can take an intent; `builds_on` is an advisory edge the lint and the graph page read, not the build."
finding_ref: cfk-7b47f4cb34d7488e
---

The scope page says blocked_by and builds_on decide when a lane can take an intent, but the build's checks read only blocked_by

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/04-scope.md:40` — "Its dependencies (`blocked_by`, `builds_on`) and the readiness gate (`/abcd:intent ready`) decide when a lane can take it"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:70` — "nothing the record names in `blocked_by` is unsettled"

The scope page says an intent's dependencies, blocked_by and builds_on, together with the readiness gate decide when a lane can take it. The build chapter creates no run until every check passes, and its dependency check reads only blocked_by. No check and no part of the pick's scoring reads builds_on. Under the build chapter, builds_on decides nothing about when a lane takes an intent.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 81.
