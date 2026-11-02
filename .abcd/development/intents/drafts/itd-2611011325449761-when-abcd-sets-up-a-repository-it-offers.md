---
id: itd-2611011325449761
slug: when-abcd-sets-up-a-repository-it-offers
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2611011325403127]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# New repos offer a choice of check profiles with their costs

## Press Release

> When abcd sets up a repository, it offers the product thinker a short choice of check profiles drawn from what developing abcd itself taught: local-first and free, self-attested, an independent checker box, or hosted CI. Each names the gates it keeps, why automation needs them, and what it is expected to cost.

## Why This Matters

> abcd sets up a repository's release and dependency workflows but leaves its checks to whatever the repository already had, and the product thinker is never told what a check costs or which checks automation actually depends on. Developing abcd itself has taught both: which gates catch the faults agents make (format, lint, the record gates, secret scanning, attribution, the unit tests, the race detector), which only cost minutes (macOS on every landing, the whole suite for a change to records), and what a merge queue doubles. A new repository should start from that experience as a choice the product thinker can understand, not a configuration a facilitator must design.

## Mechanism

> We expect the choice to be understandable because each profile is a fixed set of named gates with a one-line reason each and a cost estimate from itd-2611011325403127's measures, and the profiles differ along the one axis the product thinker cares about: who checks the agents' work, and what it costs. It would be shown wrong if product thinkers picked a profile and then were surprised by its cost or by a fault it was supposed to catch.

## Scope Conditions

- New repositories set up through abcd, and existing ones on request; a repository's own extra checks are kept, never removed.
- The gate sets are drawn from abcd's own repository's experience and are stated as such; a repository in another language gets the language-neutral gates and its own test command.
- The profiles available depend on the machine: self-attestation needs the conditions of itd-2611011325428560, and the checker box needs itd-2611011325430933's box.

## Acceptance Criteria

- Given a repository being set up, when abcd offers check profiles, then it offers at least local-first (free, the person's machine only), self-attested, independent checker box, and hosted CI, each naming its gates, why automation needs each, and its expected monthly cost at the repository's expected pace.
- Given a profile chosen, when abcd writes the repository's configuration, then the gates, their triggers and the required checks match the profile, and the choice is recorded with its date and who made it.
- Given a repository on a profile, when the measure of itd-2611011325403127 shows its cost differing from the estimate by more than a margin the person set, then abcd says so and offers the next cheaper profile.
- Given the essential gate set, when the product thinker asks why a gate is in it, then abcd names the class of fault it caught in abcd's own development, with a record to read.

## Open Questions

1. Which gates are essential in every profile, even local-first (secret scanning? attribution?), and which are optional?
2. Is the profile set per repository only, or also per change (a release always runs hosted CI)?
3. Does an existing managed repository get offered a profile on its next ahoy run, or only on request?

## Decomposition

Hand-run four-piece table, 2026-11-01, one of five drafts filed together from the investigation of hosted CI and local compute costs (a private ideate verdict, reframed). The product thinker ruled on 2026-11-01 that the work is filed as separate intents in this order: measure (itd-2611011325403127), fewer tests (itd-2611011325421304), self-attestation (itd-2611011325428560), an independent checker (itd-2611011325430933), and the check profiles new repositories are offered (itd-2611011325449761), the fifth at the product thinker's own request the same day.

| Part | Kind | Home |
|---|---|---|
| the profile choice at setup, the essential gate set with reasons, the cost estimate and the drift warning | capability | this intent |
| a repository's own checks are kept, never removed by a profile | trust rule | already standing in the ahoy surface (abcd never removes what a repository set); cited, no new ADR |
| writing the workflows and rulesets for a profile | plumbing | the ahoy and launch scaffold chapters |
| (none) | stance | none |

Verdict: FILE-AS-IS. Typed links: this intent `refines` itd-106 (abcd sets up the CI a repository requires) and `builds_on` itd-2611011325403127 for its estimates; it packages the other three drafts as choices.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
