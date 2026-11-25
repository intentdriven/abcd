---
id: itd-2611251725209625
slug: abcd-build-runs-from-a-plain-terminal-as
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610222215485265, itd-2611251725203320]
refines: [itd-2609201916151817]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A build runs from a plain terminal, with no session open

## Press Release

> abcd build runs from a plain terminal, as abcd drain already does. Maya picks a planned idea, types one command, answers one question and leaves. abcd starts each agent itself on the separate login, lands what passes its checks and hands back what does not, with no session of Maya's holding it up.

## Why This Matters

`abcd drain` can make every move from a plain terminal; `abcd build` cannot, so building a planned idea still needs a session open and everything that session holds. adr-2610222222326104 lets a build adopt the drive's four conditions later. The product thinker ruled on 2026-11-25 that a build from the plain terminal follows the two parts that land first (dcn-2611251726008978).

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

- Holds under the four conditions of adr-2610222222326104, as the drive does.
- Holds for one planned intent a person names.

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- Whether the build takes the drive's loop as it is or a second caller of it: for the spec, once the drive's own changes of November 2026 have landed.

## Decisions

1. The product thinker's, on 2026-11-25 (dcn-2611251726008978): not before the six-day run starts. The reason is the writing session's: It changes the files the drive's own changes are in, so the run that holds those files plans it and builds it after them.

## Decomposition

Hand-run four-piece table, 2026-11-25. One capability, this intent; no new trust rule, since adr-2610222222326104 names the case. Typed links: `refines` itd-2609201916151817, whose opt-in process driver this makes the walled drive; `builds_on` itd-2610222215485265 and itd-2611251725203320. Flagged for the product thinker: This may duplicate in part that intent's process-driver criterion. Verdict: FILE-AS-IS.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
