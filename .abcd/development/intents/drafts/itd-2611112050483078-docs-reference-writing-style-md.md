---
id: itd-2611112050483078
slug: docs-reference-writing-style-md
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610251558557436]
origin: extracted-from-record
production_mode: hand-written
---

# docs/reference/writing-style.md describes its own standing in three ways the repository does not bear out: (a) lines 21-23 say a new rule enters as a warn and is promoted once the corpus is clean, but punctuation/em-dash-in-list-item (fb5ac85ea) and roles/retired-role-word (b4df4f25d) were armed at blocker in the change that cleaned the corpus, and only link_anchors entered warn-first (internal/core/lint/linkanchors.go:16-21); (b) lines 3-5 say other surfaces point at the guide rather than restate it, but the bundled DOCUMENTATION rules domain restates four of its rules (internal/core/rules/defaults/rules.json); (c) lines 87-90 list the rules without a line escape and omit persona_registry, which reads none (internal/core/lint/persona.go:65-85).

## Press Release

> _Seeded by promotion from iss-2610251558557436. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610251558557436`: docs/reference/writing-style.md describes its own standing in three ways the repository does not bear out: (a) lines 21-23 say a new rule enters as a warn and is promoted once the corpus is clean, but punctuation/em-dash-in-list-item (fb5ac85ea) and roles/retired-role-word (b4df4f25d) were armed at blocker in the change that cleaned the corpus, and only link_anchors entered warn-first (internal/core/lint/linkanchors.go:16-21); (b) lines 3-5 say other surfaces point at the guide rather than restate it, but the bundled DOCUMENTATION rules domain restates four of its rules (internal/core/rules/defaults/rules.json); (c) lines 87-90 list the rules without a line escape and omit persona_registry, which reads none (internal/core/lint/persona.go:65-85).. Read that issue record for the source observation.

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
