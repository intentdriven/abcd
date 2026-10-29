---
schema_version: 1
id: "iss-2610262030417187"
slug: "invariant-8-requires-every-intent-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 32 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/03-invariants.md:153"
remedy: "In 03-invariants.md invariant 8, scope the press-release criteria rule to planned and shipped intents (the plan step's hasAcceptanceCriteria refuses a draft without them) and say a draft on the bench may carry none, as 05-intent.md section 6 records."
finding_ref: cfk-5eb0022c36a85741
---

Invariant 8 requires every intent's press release to carry Acceptance Criteria; the intent chapter says a bench draft may carry none

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/03-invariants.md:153` — "every intent's press release is followed by a `## Acceptance Criteria` block in Given-When-Then format (per itd-1)"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:506` — "a draft still on the bench may carry none"

03-invariants lists, as a non-negotiable invariant, that every intent's press release is followed by a Given-When-Then Acceptance Criteria block, while 05-intent records as accepted convention that a draft on the bench may carry none, and counts four drafts doing so. A draft with a press release and no criteria violates the invariant yet is admitted by the chapter.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 32.
