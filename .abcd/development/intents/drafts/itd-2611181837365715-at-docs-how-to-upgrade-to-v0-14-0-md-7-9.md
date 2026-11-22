---
id: itd-2611181837365715
slug: at-docs-how-to-upgrade-to-v0-14-0-md-7-9
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610151422157749]
origin: extracted-from-record
production_mode: hand-written
---

# At docs/how-to/upgrade-to-v0.14.0.md:7-9,32-35: The guide says a script that runs an old command name 'gets a refusal that names what replaced it'. The new paragraph covers bare `abcd memory` now running, but on this binary `abcd memory ingest`, `ask` and `lint` refuse as a plain cobra unknown sub-verb (`abcd: unknown command "ingest" for "abcd memory"`, exit 2) with no pointer to `library`. Verified: `abcd source` still names its successor. The paragraph also says only 'a release after v0.14.0' and does not name v0.14.2. / The note says 'A release after v0.14.0' reuses the name memory without naming the release, while the table above still lists `abcd memory` as retired. Verified true of this binary (`abcd memory` answers with a count, `abcd memory ingest` is still unknown command, exit 2). Naming v0.14.2 would be exact. (v0.14.2 release docs-currency review, raised by opus+sonnet.)

## Press Release

> _Seeded by promotion from iss-2610151422157749. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610151422157749`: At docs/how-to/upgrade-to-v0.14.0.md:7-9,32-35: The guide says a script that runs an old command name 'gets a refusal that names what replaced it'. The new paragraph covers bare `abcd memory` now running, but on this binary `abcd memory ingest`, `ask` and `lint` refuse as a plain cobra unknown sub-verb (`abcd: unknown command "ingest" for "abcd memory"`, exit 2) with no pointer to `library`. Verified: `abcd source` still names its successor. The paragraph also says only 'a release after v0.14.0' and does not name v0.14.2. / The note says 'A release after v0.14.0' reuses the name memory without naming the release, while the table above still lists `abcd memory` as retired. Verified true of this binary (`abcd memory` answers with a count, `abcd memory ingest` is still unknown command, exit 2). Naming v0.14.2 would be exact. (v0.14.2 release docs-currency review, raised by opus+sonnet.). Read that issue record for the source observation.

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
