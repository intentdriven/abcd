---
id: itd-2609151838327688
slug: an-opt-in-adapter-to-a-local-message-broker-brings-push-deli
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609151838312703]
severity: minor
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
---

# An opt-in adapter to a local message broker brings push delivery and cross-machine reach to the session mailbox: with the adapter enabled a session is woken the moment a message arrives, machines on the local network exchange messages without a shared mount, and every delivered message is still mirrored to the mailbox as a record; without it the mailbox works as before

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

An opt-in adapter to a local message broker brings push delivery and cross-machine reach to the session mailbox: with the adapter enabled a session is woken the moment a message arrives, machines on the local network exchange messages without a shared mount, and every delivered message is still mirrored to the mailbox as a record; without it the mailbox works as before

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Prior Art

- [itd-2609151838312703](itd-2609151838312703-sessions-on-one-machine-or-one-local-network-leave-each-othe.md) — the basic this adapter extends; sequenced after it, and it shares the mailbox's record contract.
- [2026-09-15-agent-messaging-on-a-local-network](../../research/notes/2026-09-15-agent-messaging-on-a-local-network.md) — ranks an embedded NATS server with JetStream as the adapter, with an MCP mailbox server as the runner-up.
- [basics-built-in-adapters-bring-power](../../principles/basics-built-in-adapters-bring-power.md) — the stance; the adapter is opt-in and never a precondition.

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
