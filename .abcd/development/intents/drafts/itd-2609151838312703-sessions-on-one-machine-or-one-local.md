---
id: itd-2609151838312703
slug: sessions-on-one-machine-or-one-local
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609091034175565]
severity: minor
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
---

# Sessions under any account, on any machine or in the cloud tell each other what they are doing through short fixed-form notes on the shared remote, and what is really a hold stays a reservation

## Press Release

> _Re-seeded on 2026-10-15 from the product thinker's ruling (Decisions). Expand into the full press-release narrative before planning._

## Why This Matters

On 2026-10-14 five sessions coordinated through the host's cross-session messages: sub-agent slot loans, who owns a design, "I am merging this, rebase after", "I am cutting a release, hold your merge". Those messages reach only sessions under one account on one machine. From 2026-10-15 a second account works the ledger and hosted cloud sessions are planned (itd-2610120738075547, not yet on the default branch); neither can receive them, and the other person never sees them, which the 2026-10-03 ruling forbids as a channel. <!-- record-lint: forward-looking -->

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Decisions

Ruled by the product thinker on 2026-10-15 ("ok, agreed, option 1"), choosing between the three options of the design pass of 2026-10-14 ([2026-10-14-session-messages-across-accounts](../../research/notes/2026-10-14-session-messages-across-accounts.md)):

1. **The carrier is short fixed-form notes on the shared remote.** Each sending session owns one branch, and each note is one commit on it carrying a kind from a fixed list, the record or pull request it concerns, and an expiry. A note carries no free text; a question points at an item, and the conversation happens on that item. This replaces the shared directory and the local-network bound this draft first stated.
2. **What is really a hold stays a reservation** ([itd-2609091034175565](../planned/itd-2609091034175565-nothing-tells-an-agent-that-a-record-it.md)): who owns an item is its reservation, and "hold your merge during a release cut" becomes a reservation lease on the release rather than a note.

Settled without a question, each the one defensible answer on the record (the research note gives the grounds):

3. A shared machine folder is out: it needs a file another account can write, which abcd never reads (a ruling relayed on 2026-10-14, not yet in the decision log), and it never leaves the machine.
4. No note grants anything (no slot loans, no approvals): a note informs or asks, never authorises.
5. The sub-agent budget ([itd-2610100704270254](itd-2610100704270254-an-abcd-run-works-out-how-many-sub.md)) sits beside this intent, not in it: the machine cap is a property of one machine, and hosted sessions run on their own.
6. The host's same-account messages stay a wake-up accelerator only, never the record.
7. A note that cannot be fetched is reported as unread since a time and for a reason, never shown as silence; a sender is shown as unauthenticated until identity can be bound.
8. **Step one of planning is two probes from a real hosted session:** whether a push to a branch outside the host's own prefix is accepted, and which forge identity the push arrives as. A refused push returns the branch prefix to the product thinker.

## Prior Art

- [2026-10-14-session-messages-across-accounts](../../research/notes/2026-10-14-session-messages-across-accounts.md): the options, the adversarial review, the state-of-the-art pass and the recommendation the ruling adopted.
- [2026-10-03-multi-agent-coordination-sota-and-run-lessons](../../research/notes/2026-10-03-multi-agent-coordination-sota-and-run-lessons.md): the record inventory and the claim-as-branch evidence the reservations rest on.
- [2026-09-15-agent-messaging-on-a-local-network](../../research/notes/2026-09-15-agent-messaging-on-a-local-network.md): the survey this draft first rested on, under a local-only bound the 2026-10-03 and 2026-10-15 rulings replace.
- The prompt-router hook (`abcd hook prompt-router`, itd-20's loader): the injection path a received note reuses; no second hook.

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
