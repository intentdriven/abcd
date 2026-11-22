---
id: itd-2611110645205177
slug: the-exported-reach-caller-audit-iss-33
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2609252211487887]
origin: extracted-from-record
production_mode: hand-written
---

# The exported-reach caller audit iss-33 asked for exists only for internal/core/ahoy (TestEveryExportedAhoyFunctionHasAFrontDoor). A crude survey of the other internal/core packages (an exported top-level function with no 'pkg.Name' selector in non-test Go outside its package) lists about 140 names; many are reached only inside their own package, which is over-export rather than dead code, but some have no production caller anywhere, e.g. launch.Ship (grep for '.Ship(' outside tests finds none). Each hit needs sorting into dead scaffolding (delete or wire), in-package-only (unexport), or a declared test seam (name it ...ForTest), and the audit then generalised to every core package.

## Press Release

> _Seeded by promotion from iss-2609252211487887. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2609252211487887`: The exported-reach caller audit iss-33 asked for exists only for internal/core/ahoy (TestEveryExportedAhoyFunctionHasAFrontDoor). A crude survey of the other internal/core packages (an exported top-level function with no 'pkg.Name' selector in non-test Go outside its package) lists about 140 names; many are reached only inside their own package, which is over-export rather than dead code, but some have no production caller anywhere, e.g. launch.Ship (grep for '.Ship(' outside tests finds none). Each hit needs sorting into dead scaffolding (delete or wire), in-package-only (unexport), or a declared test seam (name it ...ForTest), and the audit then generalised to every core package.. Read that issue record for the source observation.

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
