---
schema_version: 1
id: "iss-2610262030401652"
slug: "phases-are-history-on-the-scope-page-yet"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 30 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/04-scope.md:3"
remedy: "Edit 05-intent.md line 12: define `planned/` as a committed capability whose spec was minted at planning, awaiting its build, with no roadmap phase and no Phase 4 scheduling."
finding_ref: cfk-8df5867543c6ec90
---

Phases are history on the scope page, yet the intent chapter defines planned/ as scoped into a roadmap phase

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/04-scope.md:3` — "The phase documents under `roadmap/phases/` are history."
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:12` — "a committed capability, scoped into a roadmap phase and awaiting its Go build"

04-scope states the phase documents under roadmap/phases/ are history (the phase was retired), while 05-intent defines a planned/ intent as one scoped into a roadmap phase whose spec_id is set once the spec layer schedules it in Phase 4, although its own plan step mints the spec at planning. A planned intent cannot be scoped into a unit the record has retired.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 30.
