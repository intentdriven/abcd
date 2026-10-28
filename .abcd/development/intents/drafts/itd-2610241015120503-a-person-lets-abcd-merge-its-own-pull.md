---
id: itd-2610241015120503
slug: a-person-lets-abcd-merge-its-own-pull
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610240850336695]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# The person and the repository agree when abcd may merge, and what a cloud session may do

Typed links: `builds_on`
[itd-2610240850336695](itd-2610240850336695-before-stepping-away-the-person-grants.md)
(the person's grant, from which this draft was split by its Decision 13). This
draft carries the reversal flags that split moved here, each for the product
thinker to confirm, never auto-classified: it may **reverse** ruling AM1 and
[adr-2610150603484802](../../decisions/adrs/2610150603484802-a-fresh-install-s-facilitator-is-a.md),
and it may **supersede** itd-2610150602547271 (the planned facilitator
setting, whose spec is open) and refine itd-97.

## Press Release

> A person lets abcd merge its own pull requests through the merge queue, and lets cloud sessions work without waiting, under rules the repository's owners and the person both agree to.

## Why This Matters

_Written by the agent on 2026-10-24 from the planning interview and the two
adversarial reviews of itd-2610240850336695; the planning interview confirms
or rewrites it._

The interview for the person's grant ruled two things the reviews found it
could not hold without decisions of their own, so the product thinker split
them out (itd-2610240850336695, Decision 13):

- **`queue-merge`, and the planned facilitator setting folded into it.** As
  drafted, `queue-merge` let a session queue "a pull request whose ruleset
  needs no approval". The shipped landing does the opposite under ruling AM1:
  it arms auto-merge only where the ruleset requires a person's approval, and
  otherwise leaves the merge to a person, so the draft would open the one case
  AM1 closes, a merge with no person anywhere in it. Folding
  `facilitator.mode: auto` into a person's home grant also reverses
  adr-2610150603484802, under which the repository's owners consent to abcd
  merging by a deliberate committed setting; a contributor's grant alone
  would bypass that.
- **Cloud sessions autonomous by default.** A cloud session cannot read the
  person's home, so its default would come from the binary or the repository,
  and the person's grant forbids the repository to grant. It also meets
  invariant 23 (cloud work reaches the default branch only through the
  person's own collect) and the grant's own rule that no enforcement means no
  grant, where a cloud host abcd's hooks cannot reach is exactly that case.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Prior Art

- itd-2610240850336695 (draft): the person's grant, split from here.
- adr-2610150603484802 and itd-2610150602547271: the repository's
  `facilitator.mode`, the standing answer on who consents to abcd merging.
- `internal/core/landing/land.go` (`mergeGate`, ruling AM1): the shipped rule
  on when abcd arms a merge.
- Invariant 23 and itd-2610120738075547: cloud work's route through the
  person's own collect.

## Open Questions

- Does a merge abcd queues need both the repository's consent and the person's
  grant, or does one supersede the other?
- Does `queue-merge` arm only what `mergeGate` arms today, or widen it, and if
  it widens, what ruling amends AM1?
- Which preset does a cloud session get by default, where does it read it, and
  how does it hold under invariant 23 and "no enforcement, no grant"?

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
