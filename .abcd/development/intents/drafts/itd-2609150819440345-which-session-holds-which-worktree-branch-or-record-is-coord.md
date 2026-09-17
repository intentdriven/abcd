---
id: itd-2609150819440345
slug: which-session-holds-which-worktree-branch-or-record-is-coord
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
promoted_from: iss-2609100519122086
origin: extracted-from-record
production_mode: hand-written
---

# Which session holds which worktree, branch or record is coordinated entirely by conversation, so every new session repeats a handshake that nothing records. A session joining work in progress has no way to ask what is already claimed: it messages the peers it can see, waits for replies, and rebuilds a picture that the sessions before it had already built and did not write down. One measured encounter cost four messages and about fifteen minutes before any work began, and the picture it produced is not durable, so the session after that pays again. The convention that a diff you did not make is a peer's work depends on knowing who the peers are and what they hold, which is precisely the thing no artefact carries. The repository already records this gap for the narrow case of detecting a peer session before mutating git state; the wider case is claim rather than presence, and the two want the same substrate. Whatever holds it should be as cheap to write as it is to read, because a coordination record nobody updates is worse than the chat it replaced.

## Press Release

> _Seeded by promotion from iss-2609100519122086. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2609100519122086`: Which session holds which worktree, branch or record is coordinated entirely by conversation, so every new session repeats a handshake that nothing records. A session joining work in progress has no way to ask what is already claimed: it messages the peers it can see, waits for replies, and rebuilds a picture that the sessions before it had already built and did not write down. One measured encounter cost four messages and about fifteen minutes before any work began, and the picture it produced is not durable, so the session after that pays again. The convention that a diff you did not make is a peer's work depends on knowing who the peers are and what they hold, which is precisely the thing no artefact carries. The repository already records this gap for the narrow case of detecting a peer session before mutating git state; the wider case is claim rather than presence, and the two want the same substrate. Whatever holds it should be as cheap to write as it is to read, because a coordination record nobody updates is worse than the chat it replaced.. Read that issue record for the source observation.

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
