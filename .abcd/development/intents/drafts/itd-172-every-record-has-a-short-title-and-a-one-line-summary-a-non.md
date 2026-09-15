---
id: itd-172
slug: every-record-has-a-short-title-and-a-one-line-summary-a-non
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
---

# Every record has a short title and a one-line summary a non-engineer can read

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

A record's title is also its slug and therefore its filename, so a title written as a paragraph produces a filename ninety characters long and a listing nobody can scan. Several records in this repository are already in that state.

The fix is two fields rather than one. The title stays short and bounded, and carries the handle. A one-line summary carries what the record is about, written so that a reader who is not an engineer can tell whether it concerns them.

The summary belongs in frontmatter rather than in the prose. Frontmatter is what every listing surface already reads, so a summary there renders in the ledger listing, the machine-readable payloads, and the record export without any of them parsing prose and guessing which paragraph was meant. It is also checkable: presence, length, and register are all mechanical, where a prose convention drifts silently, which is how the long titles arrived. The agent prompts in this repository already carry exactly such a field, so the shape is not new here.

The summary is the natural home for product-thinker-addressed language, leaving the body free to address the facilitator. That makes this rule and the addressee register the same mechanism seen from two sides.

The migration is warn then promote, the pattern the writing-style guide already uses: required on new records, advisory on the corpus, swept opportunistically, and armed as a blocker once the corpus is clean. The duplication risk is real and is managed by making the field authoritative rather than repeating it as the body's opening line.

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
