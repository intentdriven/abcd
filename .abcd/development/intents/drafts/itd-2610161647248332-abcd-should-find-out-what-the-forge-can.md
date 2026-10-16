---
id: itd-2610161647248332
slug: abcd-should-find-out-what-the-forge-can
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610161632245527]
origin: extracted-from-record
production_mode: hand-written
---

# abcd should find out what the forge can do instead of asking the person or assuming. On 2026-10-16, setting up a throwaway lab to test abcd land, the agent asked the product thinker whether the team organisation allowed private copies; the product thinker did not know and should not have to. A read-only query answered it (members_can_fork_private_repositories is false for the organisation's own repositories), and a real attempt showed a private fork of a repository the person owns works anyway. The general want: abcd tests the forge's functionality itself (can repositories be private, can they be forked privately, is a merge queue available) and goes with the route that delivers the wanted outcome; only when no route exists does it ask, offering the options. Likely an intent; it needs a state-of-the-art pass on forge capability detection.

## Press Release

> _Seeded by promotion from iss-2610161632245527. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610161632245527`: abcd should find out what the forge can do instead of asking the person or assuming. On 2026-10-16, setting up a throwaway lab to test abcd land, the agent asked the product thinker whether the team organisation allowed private copies; the product thinker did not know and should not have to. A read-only query answered it (members_can_fork_private_repositories is false for the organisation's own repositories), and a real attempt showed a private fork of a repository the person owns works anyway. The general want: abcd tests the forge's functionality itself (can repositories be private, can they be forked privately, is a merge queue available) and goes with the route that delivers the wanted outcome; only when no route exists does it ask, offering the options. Likely an intent; it needs a state-of-the-art pass on forge capability detection.. Read that issue record for the source observation.

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
