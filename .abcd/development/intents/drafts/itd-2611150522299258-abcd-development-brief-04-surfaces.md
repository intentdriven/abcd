---
id: itd-2611150522299258
slug: abcd-development-brief-04-surfaces
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2611040554494843]
origin: extracted-from-record
production_mode: hand-written
---

# .abcd/development/brief/04-surfaces/22-site.md:256-261 says one input reaches past the durable record, the issue-ledger switch, and .abcd/development/brief/04-surfaces/22-site.md:340-348 enumerates the record export without decisions. record.decisions (internal/core/site/manifest.go:313, set in .abcd/site.json) publishes .abcd/work/decisions/ as a decision log and a page per record, and record.json carries a `decisions` key. Its only brief mention is .abcd/development/brief/05-internals/10-site.md:105. Bare `abcd site` reports only the issue ledger.

## Press Release

> _Seeded by promotion from iss-2611040554494843. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2611040554494843`: .abcd/development/brief/04-surfaces/22-site.md:256-261 says one input reaches past the durable record, the issue-ledger switch, and .abcd/development/brief/04-surfaces/22-site.md:340-348 enumerates the record export without decisions. record.decisions (internal/core/site/manifest.go:313, set in .abcd/site.json) publishes .abcd/work/decisions/ as a decision log and a page per record, and record.json carries a `decisions` key. Its only brief mention is .abcd/development/brief/05-internals/10-site.md:105. Bare `abcd site` reports only the issue ledger.. Read that issue record for the source observation.

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
