---
schema_version: 1
id: "iss-2610262037255566"
slug: "itd-57-says-nothing-holds-a-planned-item"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 722 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-57-manual-hold-sentinel.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-57]
finding_ref: cfk-59bfd06f3cbf4a37
---

itd-57 says nothing holds a planned item back from autonomous pickup today; the build brief's hold check already refuses any record carrying held:

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-57-manual-hold-sentinel.md:17` — "Today, holding a spec back from the run seam means either deleting it (loses the work), leaving `plan_review_status` open (fragile — a reviewer can flip it), or hand-rolling a never-`done` dependency each time."
- End B: `.abcd/development/brief/04-surfaces/34-build.md:68` — "the record carries no `held:`, well formed or not"

itd-57 assumes that today the only ways to keep work from an autonomous run are deleting it, leaving a review status open, or hand-rolling a never-done dependency, and that the selector skips specs by native spec dependency. The build chapter says no run starts, and the pick passes over, any record that carries `held:` in any form, and that blocking is read from the intent's `blocked_by`. The hold that itd-57 sets out to invent already exists on the record, so the two documents describe different pickup mechanisms for the same autonomous run.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 722.
