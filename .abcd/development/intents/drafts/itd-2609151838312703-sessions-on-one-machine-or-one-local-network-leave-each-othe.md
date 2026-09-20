---
id: itd-2609151838312703
slug: sessions-on-one-machine-or-one-local-network-leave-each-othe
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
---

# Sessions on one machine or one local network leave each other messages in a shared mailbox abcd owns: a session under another account or on another machine writes a signed record into a shared directory, the recipient's prompt hook injects it and acknowledges it, and every message stays a readable record; no daemon, no dependency, nothing leaves the local network

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

Sessions on one machine or one local network leave each other messages in a shared mailbox abcd owns: a session under another account or on another machine writes a signed record into a shared directory, the recipient's prompt hook injects it and acknowledges it, and every message stays a readable record; no daemon, no dependency, nothing leaves the local network

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Prior Art

- [2026-09-15-agent-messaging-on-a-local-network](../../research/notes/2026-09-15-agent-messaging-on-a-local-network.md) — the survey this draft rests on: a Maildir-shaped shared-directory mailbox ranked first as the built-in basic under the local-only bound.
- [basics-built-in-adapters-bring-power](../../principles/basics-built-in-adapters-bring-power.md) — the stance that makes this the basic half; the broker adapter is itd-2609151838327688.
- The prompt-router hook (`abcd hook prompt-router`, itd-20's loader) — the injection path the mailbox reuses; no second hook.
- The hand protocol of 2026-09-15: two sessions under two accounts passing dated notes through one shared markdown file; this draft removes its two failure modes (colliding appends, no acknowledgement).

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
