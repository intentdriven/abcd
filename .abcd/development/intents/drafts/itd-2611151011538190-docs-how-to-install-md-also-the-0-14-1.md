---
id: itd-2611151011538190
slug: docs-how-to-install-md-also-the-0-14-1
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610141706063703]
origin: extracted-from-record
production_mode: hand-written
---

# docs/how-to/install.md (also the 0.14.1 changelog and README) says a gh that is not signed in installs on the checksum alone with a warning; in every path a gh with --source-digest first resolves the release commit through the unauthenticated GitHub API and refuses when that lookup fails, before sign-in is tested, so a rate limit or outage gives a refusal, not the checksum-only install (fails closed).

## Press Release

> _Seeded by promotion from iss-2610141706063703. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610141706063703`: docs/how-to/install.md (also the 0.14.1 changelog and README) says a gh that is not signed in installs on the checksum alone with a warning; in every path a gh with --source-digest first resolves the release commit through the unauthenticated GitHub API and refuses when that lookup fails, before sign-in is tested, so a rate limit or outage gives a refusal, not the checksum-only install (fails closed).. Read that issue record for the source observation.

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
