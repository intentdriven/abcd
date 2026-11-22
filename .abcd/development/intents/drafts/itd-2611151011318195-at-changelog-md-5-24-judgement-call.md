---
id: itd-2611151011318195
slug: at-changelog-md-5-24-judgement-call
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610151422155612]
origin: extracted-from-record
production_mode: hand-written
---

# At CHANGELOG.md:5-24: Judgement call under the derived-changelog convention, which gives partly delivered intents no changelog line. This binary ships user-visible verbs that the 0.14.2 section never mentions: `abcd embargo` (listed in the default root help under Release, with a full command page), `abcd reserve` and `abcd memory` (in --help --agent), `rules --prompt`, drain reservations, and consistency chunking with --max-chunks. Line 5 says itd-2610112239022766 (embargo) is 'targeted and not shipped', but `abcd embargo` is in the binary. It also reuses the name `abcd memory`: in v0.14.0 and v0.14.1 bare `abcd memory` refused with exit 2; it now exits 0 with a different payload, and the section has no line saying so (the preamble says a break is stated as an Added line). Only docs/how-to/upgrade-to-v0.14.0.md mentions the reuse. / The pages shipped in this range document verbs the changelog and release note never announce: abcd reserve, abcd memory (add/list/remove), abcd embargo, `abcd rules --prompt`, reservation fields on `abcd peers` and the bare board. They exist in the binary and the pages match it, so nothing misleads; the changelog is record-derived, so this follows from which records were… (v0.14.2 release docs-currency review, raised by opus+sonnet.)

## Press Release

> _Seeded by promotion from iss-2610151422155612. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610151422155612`: At CHANGELOG.md:5-24: Judgement call under the derived-changelog convention, which gives partly delivered intents no changelog line. This binary ships user-visible verbs that the 0.14.2 section never mentions: `abcd embargo` (listed in the default root help under Release, with a full command page), `abcd reserve` and `abcd memory` (in --help --agent), `rules --prompt`, drain reservations, and consistency chunking with --max-chunks. Line 5 says itd-2610112239022766 (embargo) is 'targeted and not shipped', but `abcd embargo` is in the binary. It also reuses the name `abcd memory`: in v0.14.0 and v0.14.1 bare `abcd memory` refused with exit 2; it now exits 0 with a different payload, and the section has no line saying so (the preamble says a break is stated as an Added line). Only docs/how-to/upgrade-to-v0.14.0.md mentions the reuse. / The pages shipped in this range document verbs the changelog and release note never announce: abcd reserve, abcd memory (add/list/remove), abcd embargo, `abcd rules --prompt`, reservation fields on `abcd peers` and the bare board. They exist in the binary and the pages match it, so nothing misleads; the changelog is record-derived, so this follows from which records were… (v0.14.2 release docs-currency review, raised by opus+sonnet.). Read that issue record for the source observation.

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
