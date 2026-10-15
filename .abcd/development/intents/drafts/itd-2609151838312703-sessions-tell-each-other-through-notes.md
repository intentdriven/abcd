---
id: itd-2609151838312703
slug: sessions-tell-each-other-through-notes
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609091034175565]
severity: minor
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
related_issues: [iss-2609091037191879, iss-2610142200372105, iss-2608220750029993]
related_intents: [itd-2609151838327688, itd-2610100704270254, itd-2609150819440345]
related_adrs: [adr-2610031352212697]
---

# Sessions under any account, on any machine or in the cloud tell each other what they are doing through short fixed-form notes on the shared remote, and what is really a hold stays a reservation

## Press Release

> **Sessions working on one project tell each other what they are doing, whoever runs them and wherever they run.** A session about to merge a change says so in a short note that every other session sees: on the same computer, under a colleague's account on another computer, or in a cloud session. A session cutting a release holds merges with a reservation everyone sees, and every session's merge step waits until the cut is done. A note says only what happened and which item it concerns, never free sentences, so it cannot slip instructions into another agent; a question points at the item, and people talk there.
>
> "Last night five of my sessions passed dozens of messages, and from this morning my colleague's sessions and my cloud sessions could see none of them," said Maya, an autonomous-development practitioner working with a second contributor. "Now when one of my sessions starts merging, my colleague's session sees 'merging the login fix' before it pushes, and when I cut a release, nobody's session merges into it. I can read every note my sessions sent in one list."

> _Agent-seeded on 2026-10-15 from the ruling (Decisions) and revised after two adversarial reviews the same day; confirmed as written by the product thinker at the planning interview of 2026-10-15 (Q4, in four parts)._

## Why This Matters

On 2026-10-14 five sessions coordinated through the host's cross-session messages: sub-agent slot loans, who owns a design, "I am merging this, rebase after", "I am cutting a release, hold your merge". Those messages reach only sessions under one account on one machine. From 2026-10-15 a second account works the ledger and hosted cloud sessions are planned (itd-2610120738075547, not yet on the default branch); neither can receive them, and the other person never sees them, so under the ruling of 2026-10-03 they are never the record of who works on what. A release cut in progress is invisible to every session outside the one cutting it (iss-2609091037191879), and the only defence today is a message the other account cannot receive. <!-- record-lint: forward-looking -->

## Mechanism

_Agent-seeded; for the product thinker's confirmation at the planning interview._

We expect sessions under different accounts, on different machines and in the cloud to stop merging into each other's release cuts and onto branches another session announced it was merging, because the hold and the announcement now live on the shared remote that each of them already fetches, and the merge step itself reads the hold. This would be shown wrong if, in the first two weeks of two-account work, a session lands a change into a release cut that held a live reservation, or merges onto a branch whose `merging` note it had fetched.

## Scope Conditions

- Holds for participants who can push to the shared remote: people with write access and the cloud sessions they start. An outside contributor can read the notes but cannot send one.
- Holds while senders are cooperative: anyone who can push can write a note under another person's name, and abcd shows a sender as claimed, not proven.
- Holds at coordination scale, on the order of the dozens of messages five sessions exchanged on 2026-10-14; a busier use is a re-decision.
- Holds while everything a note carries may be public for ever: a kind, an item id, a time and a username, on a public repository.
- Holds for warnings that may arrive late: a session sees a note only after a fetch, so nothing that must never be missed rests on a note alone; the release hold is enforced at the merge step instead (C7).
- Holds once the embargo check is on the default branch (adr-2610120558328615, on an unmerged branch on 2026-10-15); until then C5 cannot be built. <!-- record-lint: forward-looking -->
- Hypothesis until the cloud probe answers it: a hosted cloud session may push a branch outside the host's own prefix, and its git identity resolves to a forge username. If either fails, a cloud session cannot send notes under this design and the question goes back to the product thinker.

## Acceptance Criteria

> _Agent-seeded on 2026-10-15 and revised after two adversarial reviews (design and feasibility; record discipline) the same day; proposals for the product thinker to accept, edit or strike one by one at the planning interview._

- **C1, a session sends a note.** **Given** a session in a checkout that can push to the shared remote, **when** it sends a `merging` note about pull request 1012, **then** one commit that changes no file lands on the sender's own notes branch, `abcd-notes/<username in lower case>`, carrying the kind, the item, the sender, the working copy, the time sent and an expiry, and the sender is told the note went out.
- **C2, a lost race is retried, then reported.** **Given** two sessions of one person sending notes at the same moment, **when** one push is rejected because the branch moved, **then** that note is rebuilt on the new tip and pushed again, up to three times, and if it still fails the sender is told "not sent" with the reason; no note is ever force-pushed.
- **C3, a note is a fixed form or nothing.** **Given** a note with a kind outside the fixed list (`merging`, `merged`, `asking`, `answered`, `handed-over`), an item that is not a record id or a pull-request number, any free text, or an expiry beyond 24 hours, **when** a session tries to send it, **then** abcd refuses, naming the field and the limit, and pushes nothing.
- **C4, a note about a private record is refused.** **Given** a note about a record that is absent from the default branch as last fetched (or with no default branch fetched at all), **when** a session tries to send it, **then** abcd refuses before pushing and says the record is not public yet.
- **C5, a note about an embargoed record is refused.** **Given** a note about a record the embargo store on this machine holds, **when** a session tries to send it, **then** abcd refuses before pushing; and where the store exists but its check cannot run, the note is refused, never let through.
- **C6, other sessions see it, once, as data.** **Given** a note pushed by another account or by another session of the same person, **when** a session's next prompt comes after a fetch that brought the note in, **then** that session is shown the note once, rendered by abcd from its fields alone (for example `note from @maya (claimed): merging #1012, for 30 min`), never the commit's raw text, with the line that a note informs and grants nothing.
- **C7, a release cut holds every merge.** **Given** a session that has reserved release v0.16.0 for its cut, **when** any other session's land step runs while that reservation is live, **then** the land step refuses and names the holder and the release, and the reservation ends when the cut's tag reaches the remote or its holder releases it.
- **C8, the hold is shown.** **Given** a live release reservation, **when** any session runs `abcd peers` or reaches its next prompt after a fetch, **then** it is shown "release v0.16.0 being cut by @maya: merges are held".
- **C9, expiry survives a wrong clock.** **Given** a sender whose clock runs fifteen minutes slow, **when** its thirty-minute `merging` note is fetched, **then** the note is shown for thirty minutes from when this checkout first fetched it, not judged against the sender's clock.
- **C10, a failed fetch is said, never silent.** **Given** a session whose last fetch of the notes failed (offline, an expired login, a refused fetch), **when** its next prompt comes, **then** it is shown "notes unread since 09:12: <reason>"; an empty list is shown only after a fetch that succeeded, and a checkout with no shared remote shows nothing.
- **C11, a doubtful note is flagged, and a bad branch never blocks the rest.** **Given** a note whose commit names a sender other than the branch's owner, a notes branch that was rewritten rather than added to, or a branch under `abcd-notes/` whose name is not a lower-case username, **when** the notes are read, **then** the doubtful note or branch is shown marked with the reason, and every other branch is still read.
- **C12, the prompt never waits on the network.** **Given** a shared remote that does not answer, **when** a prompt is submitted, **then** the prompt hook returns within one second, showing notes from the last completed fetch with its age.
- **C13, a cloud session can send.** **Given** a hosted cloud session with no preflight receipt, **when** it pushes a note, **then** the push hook admits the push because it is a fast-forward of commits that change no file, the same shape check reservations pass.
- **C14, only notes pass without a receipt.** **Given** a push under `abcd-notes/` that changes a file or rewrites the branch, **when** it is pushed, **then** the push hook refuses it as it refuses any branch without a receipt.
- **C15, the history cannot be erased or rewritten.** **Given** the repository's rulesets, **when** anyone tries to delete or force-push a branch under `abcd-notes/`, **then** the forge refuses it, and the ruleset is mirrored in the repository's rulesets folder and applied through the rulesets' own apply verb.
- **C16, a person reads what their sessions said.** **Given** notes from several senders, **when** a person runs `abcd notes`, **then** abcd fetches the notes once and lists them newest first with sender, kind, item, time and how long each still stands, marking claimed and doubtful senders as C6 and C11 do.

## Decomposition

Run by hand under [`decompose-before-filing`](../../principles/decompose-before-filing.md) on 2026-10-15; agent-seeded, revised after review, for the product thinker's confirmation.

| Part | Type | Home | Verdict |
| --- | --- | --- | --- |
| Sending, reading and listing fixed-form notes on the shared remote (C1 to C6, C9 to C12, C16) | capability | this intent, generalising the reservation primitive's write, fetch, trailer collection and render rather than adding a second copy ([`one-canonical-primitive`](../../principles/one-canonical-primitive.md)) | file here |
| A release cut held by a reservation the land step reads (C7, C8) | capability | this intent; it widens the reservation's subject from a record to a release and adds a reader to the land fence, so it `builds_on` [itd-2609091034175565](../planned/itd-2609091034175565-nothing-tells-an-agent-that-a-record-it.md); it would resolve iss-2609091037191879 | file here |
| A notes branch that moves without a preflight receipt, and the ruleset that keeps its history (C13 to C15) | trust rule | a new ADR beside [adr-2610031352212697](../../decisions/adrs/2610031352212697-claims-are-branches-on-the-shared-remote.md), plus clauses in brief invariants 7 and 10 naming the notes verbs and branch as the reservation's clauses do | mint at planning |
| When abcd fetches the notes (C12, C16): only through verbs whose meaning says so | trust rule | brief invariant 7 ([adr-38](../../decisions/adrs/0038-implicit-checks-are-disk-only.md)) as it stands, with a clause naming the notes verbs as the reservations' clause does (decision 14) | clause at planning |
| A message from a peer informs and never authorises | stance | already held for the host's own messages; stated as decision 4 and taught by C6's rendered line; no new principle | cite, no new record |
| The sub-agent budget | capability | [itd-2610100704270254](itd-2610100704270254-an-abcd-run-works-out-how-many-sub.md), beside this intent (decision 5) | not this intent |
| Session presence: which sessions are running now | capability | iss-2608220750029993 and the session register [itd-2609150819440345](itd-2609150819440345-which-session-holds-which-worktree.md); notes say what happened, not who is alive (decision 16) | keep both |
| An opt-in broker adapter for notes on a local network | capability | [itd-2609151838327688](itd-2609151838327688-an-opt-in-adapter-to-a-local-message.md), kept as a draft to be rewritten later as an optional faster carrier of these notes (decision 15) | keep both |

## Decisions

Ruled by the product thinker on 2026-10-15 ("ok, agreed, option 1"; the decision log's entry of that day), choosing between the three options of the design pass of 2026-10-14 ([2026-10-14-session-messages-across-accounts](../../research/notes/2026-10-14-session-messages-across-accounts.md)):

1. **The carrier is short fixed-form notes on the shared remote.** Each note carries a kind from a fixed list, the record or pull request it concerns, and an expiry, and no free text; a question points at an item, and the conversation happens on that item. This replaces the shared directory and the local-network bound this draft first stated.
2. **What is really a hold stays a reservation** ([itd-2609091034175565](../planned/itd-2609091034175565-nothing-tells-an-agent-that-a-record-it.md)): who owns an item is its reservation, and "hold your merge during a release cut" becomes a reservation on the release rather than a note.

Settled without a question, each the one defensible answer on the record (the research note gives the grounds):

3. A shared machine folder is out: it needs a file another account can write, which abcd never reads (the ruling of 2026-10-14, in the decision log), and it never leaves the machine.
4. No note grants anything (no slot loans, no approvals): a note informs or asks, never authorises.
5. The sub-agent budget ([itd-2610100704270254](itd-2610100704270254-an-abcd-run-works-out-how-many-sub.md)) sits beside this intent, not in it: the machine cap is a property of one machine, and hosted sessions run on their own.
6. The host's same-account messages stay a wake-up accelerator only, never the record.
7. A note that cannot be fetched is reported as unread since a time and for a reason, never shown as silence; a sender is shown as claimed until identity can be bound.
8. **Two probes from a real hosted session come first in the build:** whether a push to a branch outside the host's own prefix is accepted, and which forge identity the push arrives as. The product thinker asked for the plan on 2026-10-15 before the probes ran, so both stand as a scope-condition hypothesis; a refused push returns the branch prefix to the product thinker.

Decided by the technical facilitator on 2026-10-15 on the two adversarial reviews, each grounded in a finding the reviews reproduced:

9. **One notes branch per person, not per session.** A person's sessions append to one branch, rebuilding on a lost race (C2). A branch per session could never be swept under a no-deletion rule, so their number would grow with every session for good; one per person grows with the number of people.
10. **Branch names are lower case.** A username resolved as `Maya` on one machine and `maya` on another would create two branches that differ only in case, which makes every fetch on a case-insensitive filesystem fail (reproduced with git 2.52 on macOS), and the no-deletion rule would keep the bad branch for ever.
11. **A release hold is read by the land step**, not only shown (C7): a hold that is only shown blocks nothing, which is the failure iss-2609091037191879 records.
12. **The push hook's shape check for notes is the reservation's** (a fast-forward of commits that change no file), not a second trailer parser in shell; a note the reader cannot parse is flagged by C11.
13. **Expiry is counted from when this checkout first fetched the note** (C9), the shape of the clock finding iss-2610142200372105 records for reservations.

Ruled by the product thinker at the planning interview of 2026-10-15:

14. **Notes are fetched only by a command whose job is that** ("Only when a command checks", Q1): `abcd notes`, `abcd peers` and the land step fetch them, and the prompt hook shows only the last fetch from disk, with its age. Brief invariant 7 stands without an exception; the alternative, an opt-in fetch from the prompt hook at most once a minute, was not chosen. Accepted cost: a session that runs none of those commands sees a new note only at its next merge.
15. **The broker draft stays, to be rewritten later** ([itd-2609151838327688](itd-2609151838327688-an-opt-in-adapter-to-a-local-message.md); "Keep it, rewrite later", Q2): a messaging service on the person's own network would wake an idle session the moment a note arrives, which notes on the shared remote cannot, but it never reaches another person's computer or a cloud session. It stays a separate draft, re-based on notes as an optional faster carrier when it is next worked on; until then it still describes the retired shared-folder mailbox.
16. **Notes are planned on their own, apart from the session register** ([itd-2609150819440345](itd-2609150819440345-which-session-holds-which-worktree.md); "Plan notes on their own", Q3): notes say what a session did, the register would say which sessions are running and what each holds. The register's way of sharing between computers is still undecided, and notes do not wait for it.

## Prior Art

- [2026-10-14-session-messages-across-accounts](../../research/notes/2026-10-14-session-messages-across-accounts.md): the options, the adversarial review, the state-of-the-art pass and the recommendation the ruling adopted.
- [2026-10-03-multi-agent-coordination-sota-and-run-lessons](../../research/notes/2026-10-03-multi-agent-coordination-sota-and-run-lessons.md): the record inventory and the claim-as-branch evidence the reservations rest on.
- [2026-09-15-agent-messaging-on-a-local-network](../../research/notes/2026-09-15-agent-messaging-on-a-local-network.md): the survey this draft first rested on, under a local-only bound the 2026-10-03 and 2026-10-15 rulings replace.
- Radicle's collaborative objects (each peer the sole writer of its own namespace) and Anthropic's sixteen-agent compiler run (coordination by locks alone), both cited in the research note: the per-writer branch and the hold-as-reservation.
- The prompt-router hook (`abcd hook prompt-router`, itd-20's loader): the path a fetched note is shown through; no second hook.

## Open Questions

_None open._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
