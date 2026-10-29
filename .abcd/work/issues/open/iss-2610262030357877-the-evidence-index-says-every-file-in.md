---
schema_version: 1
id: "iss-2610262030357877"
slug: "the-evidence-index-says-every-file-in"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 20 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/03-evidence/README.md:4"
remedy: "Change the status line of 04-tradeoffs.md from PARTIAL to LIVE, as its three siblings and the chapter README say."
finding_ref: cfk-f85189e46ebe0d83
---

The evidence index says every file in the chapter is live; the tradeoffs page is marked PARTIAL

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/03-evidence/README.md:4` — "Every file is live — evidence is recorded as it accrues"
- End B: `.abcd/development/brief/03-evidence/04-tradeoffs.md:3` — "**Status: PARTIAL.** Tradeoff-shaped content (decisions made + alternatives weighed, with rationale) accumulates here as the project progresses."

The evidence chapter's README says every file is live and records evidence as it accrues. The tradeoffs file opens with a PARTIAL status, and its sibling pages are marked LIVE. The index and the page give two statuses for the same file.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 20.
