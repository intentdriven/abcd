---
id: itd-2611181402597555
slug: readme-md-40-introduces-the-principles
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610252211560027]
origin: extracted-from-record
production_mode: hand-written
---

# README.md:40 introduces the principles with a link to the record site's foundations page ('abcd is founded on several [principles](https://abcdev.app/record/foundations/), ...'). README.md:47, :57 and :60 then name three principles that page cannot carry: '**abcd builds abcd**', '**Work starts from an intent**' and '**The record is part of the product**'. The foundations page is built from the principle store alone (internal/core/site/foundations.go:16-18, explorer.go:721-746). .abcd/development/principles/ has no record for any of the three, and git grep finds those names only in README.md. The other five are in the store: prefer-the-experiment-to-the-inference.md, verifier-selects-gates-decide.md, less-but-better.md, the-record-lands-with-the-act.md and enforcement-claims-are-facts.md. So a reader who follows the link finds five of the eight principles the README names.

## Press Release

> _Seeded by promotion from iss-2610252211560027. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610252211560027`: README.md:40 introduces the principles with a link to the record site's foundations page ('abcd is founded on several [principles](https://abcdev.app/record/foundations/), ...'). README.md:47, :57 and :60 then name three principles that page cannot carry: '**abcd builds abcd**', '**Work starts from an intent**' and '**The record is part of the product**'. The foundations page is built from the principle store alone (internal/core/site/foundations.go:16-18, explorer.go:721-746). .abcd/development/principles/ has no record for any of the three, and git grep finds those names only in README.md. The other five are in the store: prefer-the-experiment-to-the-inference.md, verifier-selects-gates-decide.md, less-but-better.md, the-record-lands-with-the-act.md and enforcement-claims-are-facts.md. So a reader who follows the link finds five of the eight principles the README names.. Read that issue record for the source observation.

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
