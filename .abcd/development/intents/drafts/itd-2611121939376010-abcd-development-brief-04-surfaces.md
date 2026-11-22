---
id: itd-2611121939376010
slug: abcd-development-brief-04-surfaces
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2611040554256499]
origin: extracted-from-record
production_mode: hand-written
---

# .abcd/development/brief/04-surfaces/30-inbox.md:30-32 says a waiting file this abcd cannot read is listed as unreadable, never dropped, and that a file that is not a report says so. The list reads only names matching fileNameRe (internal/core/report/inbox.go:89, 373). A file named notes.md in the inbox is absent from `abcd inbox`, its tally and its unreadable rows; only a report-shaped name is listed as unreadable.

## Press Release

> _Seeded by promotion from iss-2611040554256499. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2611040554256499`: .abcd/development/brief/04-surfaces/30-inbox.md:30-32 says a waiting file this abcd cannot read is listed as unreadable, never dropped, and that a file that is not a report says so. The list reads only names matching fileNameRe (internal/core/report/inbox.go:89, 373). A file named notes.md in the inbox is absent from `abcd inbox`, its tally and its unreadable rows; only a report-shaped name is listed as unreadable.. Read that issue record for the source observation.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
