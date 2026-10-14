---
id: itd-2610141646463379
slug: a-reviewer-s-request-for-extra
spec_id: null
kind: standalone
bundle: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A reviewer's extra check must name what it protects

## Press Release

> **A reviewer's request for more validation or more complexity has to say what
> it protects.** Every abcd reviewer brief carries one rule: a finding that asks
> for new validation or complexity cites the acceptance criterion, the recorded
> invariant or the concrete risk it serves. A finding that cites none is recorded
> as a nit and does not force a fix.
>
> "The review asked for a third layer of input checks," said Carol, a tech lead.
> "It couldn't name a criterion or an attack it stopped, so it went down as a
> nit, and the change stayed small."

## Why This Matters

Survived, reframed, from an ideate run of 2026-10-14 (private verdict
review-against-intent) on the engineering write-up "Redesigning source control at Anthropic" (Anthropic, 2026-10-13, <https://www.anthropic.com/engineering/redesigning-source-control>), which reports that "one such review was overly
defensive about trusting another internal service, which resulted in Claude
adding excessive validation and complexity". Both reviewers of the run kept
line-level review (abcd's own review calibration shows small-delta reads finding
defects broad passes missed) and both converged on this rule. It applies abcd's
"less, but better" principle to reviews: an addition that only guards another
addition is a third copy of the problem. The security reviewer's "theoretical
unease is not a finding" is the precedent for one lens.

## Mechanism

We expect smaller, equally safe changes because a reviewer that must cite what a
check protects either finds a real criterion, invariant or risk (and the fix
stands) or cannot (and the check was guarding nothing).

## Scope Conditions

- Applies to abcd's own reviewer briefs (ruthless, security, adversarial
  ideate); line-level reading is unchanged.
- Judging whether a citation really supports a finding stays the reviewer's and
  the person's call; no lint decides it.

## Acceptance Criteria

- **Given** a reviewer finding that asks for new validation or complexity and
  cites no criterion, invariant or concrete risk, **when** the review is
  recorded, **then** it is recorded as a nit and does not block.
- **Given** such a finding that cites one, **when** the review is recorded,
  **then** it keeps its severity and the citation is kept with it.

## Open Questions

- Whether "a nit" or "rejected" is the right outcome for an uncited request (the
  two reviewers differed; product thinker at planning).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
