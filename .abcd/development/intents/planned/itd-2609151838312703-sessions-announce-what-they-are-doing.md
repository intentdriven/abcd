---
id: itd-2609151838312703
slug: sessions-announce-what-they-are-doing
spec_id: spc-2610150728396006
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609091034175565]
severity: minor
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
related_issues: [iss-2609091037191879, iss-2610142200372105, iss-2608220750029993, iss-2610150726321422, iss-2610150726331622]
related_intents: [itd-2609151838327688, itd-2610100704270254, itd-2609150819440345]
related_adrs: [adr-2610031352212697, adr-2610150729333337]
---

# Sessions under any account, on any machine or in the cloud announce what they are doing through short fixed-form announcements on the shared remote, a person reads everyone's status with one command, and what is really a hold stays a reservation

## Press Release

> **Sessions working on one project tell each other what they are doing, whoever runs them and wherever they run.** A session about to merge a change says so in a short announcement that every other session sees: on the same computer, under a colleague's account on another computer, or in a cloud session. A session cutting a release holds merges with a reservation everyone sees, and every session's merge step waits until the cut is done. An announcement says only what happened and which item it concerns, never free sentences, so it cannot slip instructions into another agent; a question points at the item, and people talk there.
>
> "Last night five of my sessions passed dozens of messages, and from this morning my colleague's sessions and my cloud sessions could see none of them," said Maya, an autonomous-development practitioner working with a second contributor. "Now when one of my sessions starts merging, my colleague's session sees 'merging the login fix' before it pushes, and when I cut a release, nobody's session merges into it. I can read every announcement my sessions made in one list."

> _Agent-seeded on 2026-10-15 from the ruling (Decisions) and revised after two adversarial reviews the same day; confirmed as written by the product thinker at the planning interview of 2026-10-15 (Q4, in four parts)._

## Why This Matters

On 2026-10-14 five sessions coordinated through the host's cross-session messages: sub-agent slot loans, who owns a design, "I am merging this, rebase after", "I am cutting a release, hold your merge". Those messages reach only sessions under one account on one machine. From 2026-10-15 a second account works the ledger and hosted cloud sessions are planned (itd-2610120738075547, not yet on the default branch); neither can receive them, and the other person never sees them, so under the ruling of 2026-10-03 they are never the record of who works on what. A release cut in progress is invisible to every session outside the one cutting it (iss-2609091037191879), and the only defence today is a message the other account cannot receive. <!-- record-lint: forward-looking -->

## Mechanism

_Agent-seeded; the claim and its falsifier confirmed as written by the product thinker at the planning interview of 2026-10-15 (Q5)._

We expect sessions under different accounts, on different machines and in the cloud to stop merging into each other's release cuts and onto branches another session announced it was merging, because the hold and the announcement now live on the shared remote that each of them already fetches, and the merge step itself reads the hold. This would be shown wrong if, in the first two weeks of two-account work, a session lands a change into a release cut that held a live reservation, or merges onto a branch whose `merging` announcement it had fetched.

## Scope Conditions

- Holds for participants who can push to the shared remote: people with write access and the cloud sessions they start. An outside contributor can read the announcements but cannot make one. <!-- cond: cond-2610150728392489 -->
- Holds where the forge keeps a record of who pushed: abcd checks each announcement's sender against it and shows the sender as verified, or as doubtful when the record names someone else; where the forge cannot say (no login, a push made by an app), the sender is shown as claimed. <!-- cond: cond-2610150728396735 -->
- Holds up to hundreds of announcements a day per person; a session is shown the newest announcement per item and a count of the rest. Many announcements a second is beyond what pushes to the shared remote carry, and belongs to the faster carrier the broker draft keeps (decision 15). <!-- cond: cond-2610150728394771 -->
- Holds while everything an announcement carries may be public for ever: a kind, an item id, a time and the sender's GitHub username, on a public repository. Announcements are kept permanently (C15); a deleted branch would not be erased anyway, since the forge keeps its commits retrievable for a while and every clone keeps a copy. <!-- cond: cond-2610150728392907 -->
- Holds for warnings that may arrive late: a session sees an announcement only after a fetch, so nothing that must never be missed rests on an announcement alone; the release hold is enforced at the merge step instead (C7). <!-- cond: cond-2610150728397242 -->
- Holds once the embargo check is on the default branch (adr-2610120558328615, on an unmerged branch on 2026-10-15); until then C5 cannot be built; the session cutting v0.15.0 expects it on the default branch on 2026-10-15. <!-- record-lint: forward-looking --> <!-- cond: cond-2610150728395676 -->
- Hypothesis until the cloud probe answers it: a hosted cloud session may push a branch outside the host's own prefix, and its git identity resolves to a forge username. If either fails, a cloud session cannot make announcements under this design and the question goes back to the product thinker. <!-- cond: cond-2610150728394737 -->

## Acceptance Criteria

> _Agent-seeded on 2026-10-15 and revised after two adversarial reviews (design and feasibility; record discipline) the same day; walked one by one with the product thinker at the planning interview of 2026-10-15 (Q9 to Q15): C1 to C18 accepted, C17 and C18 added from the answers to Q6b and Q6c, and a criterion that an announcement names nobody struck by decision 19; renamed from notes to announcements by decision 21, with C8 and C16 reading through `abcd status`._

- **C1, a session announces.** **Given** a session in a checkout that can push to the shared remote, **when** it runs `abcd announce merging #1012`, **then** one commit that changes no file lands on the sender's own status branch, `abcd-status/<forge username in lower case>`, carrying the kind, the item, the sender, the working copy, the time sent and an expiry, and the sender is told the announcement went out.
- **C2, a lost race is retried, then reported.** **Given** two sessions of one person announcing at the same moment, **when** one push is rejected because the branch moved, **then** that announcement is rebuilt on the new tip and pushed again, up to three times, and if it still fails the sender is told "not sent" with the reason; no announcement is ever force-pushed.
- **C3, an announcement is a fixed form or nothing.** **Given** an announcement with a kind outside the fixed list (`merging`, `merged`, `asking`, `answered`, `handed-over`), an item that is not a record id or a pull-request number, any free text, or an expiry beyond 24 hours, **when** a session tries to send it, **then** abcd refuses, naming the field and the limit, and pushes nothing.
- **C4, an announcement about a private record is refused.** **Given** an announcement about a record that is absent from the default branch as last fetched (or with no default branch fetched at all), **when** a session tries to send it, **then** abcd refuses before pushing and says the record is not public yet.
- **C5, an announcement about an embargoed record is refused.** **Given** an announcement about a record the embargo store on this machine holds, **when** a session tries to send it, **then** abcd refuses before pushing; and where the store exists but its check cannot run, the announcement is refused, never let through.
- **C6, other sessions see it, once, as data.** **Given** an announcement pushed by another account or by another session of the same person, **when** a session's next prompt comes after a fetch that brought it in, **then** that session is shown it once, rendered by abcd from its fields alone (for example `@maya (verified): merging #1012, for 30 min`), never the commit's raw text, with the line that an announcement informs and grants nothing.
- **C7, a release cut holds every merge.** **Given** a session that has reserved release v0.16.0 for its cut, **when** any other session's land step runs while that reservation is live, **then** the land step refuses and names the holder and the release, and the reservation ends when the cut's tag reaches the remote or its holder releases it.
- **C8, the hold is shown.** **Given** a live release reservation, **when** any session or person runs `abcd status` or a session reaches its next prompt after a fetch, **then** it is shown "release v0.16.0 being cut by @maya: merges are held".
- **C9, expiry survives a wrong clock.** **Given** a sender whose clock runs fifteen minutes slow, **when** its thirty-minute `merging` announcement is fetched, **then** the announcement is shown for thirty minutes from when this checkout first fetched it, not judged against the sender's clock.
- **C10, a failed fetch is said, never silent.** **Given** a session whose last fetch of the announcements failed (offline, an expired login, a refused fetch), **when** its next prompt comes, **then** it is shown "status unread since 09:12: <reason>"; an empty list is shown only after a fetch that succeeded, and a checkout with no shared remote shows nothing.
- **C11, a doubtful announcement is flagged, and a bad branch never blocks the rest.** **Given** an announcement whose commit names a sender other than the branch's owner, a status branch that was rewritten rather than added to, or a branch under `abcd-status/` whose name is not a lower-case forge username, **when** the announcements are read, **then** the doubtful announcement or branch is shown marked with the reason, and every other branch is still read.
- **C12, the prompt never waits on the network.** **Given** a shared remote that does not answer, **when** a prompt is submitted, **then** the prompt hook returns within one second, showing announcements from the last completed fetch with its age.
- **C13, a cloud session can announce.** **Given** a hosted cloud session with no preflight receipt, **when** it pushes an announcement, **then** the push hook admits the push because it is a fast-forward of commits that change no file, the same shape check reservations pass.
- **C14, only announcements pass without a receipt.** **Given** a push under `abcd-status/` that changes a file or rewrites the branch, **when** it is pushed, **then** the push hook refuses it as it refuses any branch without a receipt.
- **C15, the history cannot be erased or rewritten.** **Given** the repository's rulesets, **when** anyone tries to delete or force-push a branch under `abcd-status/`, **then** the forge refuses it, and the ruleset is mirrored in the repository's rulesets folder and applied through the rulesets' own apply verb.
- **C16, a person reads everyone's status.** **Given** announcements from several senders, live reservations and a release hold, **when** a person runs `abcd status`, **then** abcd fetches once and lists the announcements newest first with sender, kind, item, time and how long each still stands, marking verified, claimed and doubtful senders as C17 and C11 do, together with who holds which reservation and any release being cut.
- **C17, a sender is checked against the forge's record.** **Given** an announcement on Maya's status branch, **when** a session runs `abcd status`, **then** abcd reads the forge's record of who pushed that update and shows the sender as verified when it names the branch's owner, as doubtful when it names someone else, and as claimed when the forge cannot say (no login, another forge, a push made by an app).
- **C18, a flood of announcements is summed up.** **Given** 200 announcements that arrived since a session's last fetch, **when** its next prompt comes, **then** it is shown the newest announcement for each item, at most ten lines, and a count of the rest with the command that lists them all.

## Decomposition

Run by hand under [`decompose-before-filing`](../../principles/decompose-before-filing.md) on 2026-10-15; agent-seeded, revised after review, for the product thinker's confirmation.

| Part | Type | Home | Verdict |
| --- | --- | --- | --- |
| Making, reading and listing fixed-form announcements on the shared remote (C1 to C6, C9 to C12, C16 to C18) | capability | this intent, generalising the reservation primitive's write, fetch, trailer collection and render rather than adding a second copy ([`one-canonical-primitive`](../../principles/one-canonical-primitive.md)) | file here |
| A release cut held by a reservation the land step reads (C7, C8) | capability | this intent; it widens the reservation's subject from a record to a release and adds a reader to the land fence, so it `builds_on` [itd-2609091034175565](../shipped/itd-2609091034175565-nothing-tells-an-agent-that-a-record-it.md); it would resolve iss-2609091037191879 | file here |
| A status branch that moves without a preflight receipt, and the ruleset that keeps its history (C13 to C15) | trust rule | a new ADR beside [adr-2610031352212697](../../decisions/adrs/2610031352212697-claims-are-branches-on-the-shared-remote.md), [adr-2610150729333337](../../decisions/adrs/2610150729333337-a-status-branch-moves-without-a.md), minted at planning; clauses in brief invariants 7 and 10 naming `abcd announce`, `abcd status` and the status branch, written in the change that builds them, as the reservation's clauses were | minted |
| When abcd fetches the announcements (C12, C16): only through verbs whose meaning says so | trust rule | brief invariant 7 ([adr-38](../../decisions/adrs/0038-implicit-checks-are-disk-only.md)) as it stands, with a clause naming `abcd announce` and `abcd status` as the reservations' clause does (decision 14) | clause at planning |
| A message from a peer informs and never authorises | stance | already held for the host's own messages; stated as decision 4 and taught by C6's rendered line; no new principle | cite, no new record |
| The sub-agent budget | capability | [itd-2610100704270254](../drafts/itd-2610100704270254-an-abcd-run-works-out-how-many-sub.md), beside this intent (decision 5) | not this intent |
| Session presence: which sessions are running now | capability | iss-2608220750029993 and the session register [itd-2609150819440345](../drafts/itd-2609150819440345-which-session-holds-which-worktree.md); announcements say what happened, not who is alive (decision 16) | keep both |
| An opt-in broker adapter for announcements on a local network | capability | [itd-2609151838327688](../drafts/itd-2609151838327688-an-opt-in-adapter-to-a-local-message.md), kept as a draft to be rewritten later as an optional faster carrier of these announcements (decision 15) | keep both |

## Decisions

Ruled by the product thinker on 2026-10-15 ("ok, agreed, option 1"; the decision log's entry of that day), choosing between the three options of the design pass of 2026-10-14 ([2026-10-14-session-messages-across-accounts](../../research/notes/2026-10-14-session-messages-across-accounts.md)):

1. **The carrier is short fixed-form notes on the shared remote.** Each note carries a kind from a fixed list, the record or pull request it concerns, and an expiry, and no free text; a question points at an item, and the conversation happens on that item. This replaces the shared directory and the local-network bound this draft first stated.
2. **What is really a hold stays a reservation** ([itd-2609091034175565](../shipped/itd-2609091034175565-nothing-tells-an-agent-that-a-record-it.md)): who owns an item is its reservation, and "hold your merge during a release cut" becomes a reservation on the release rather than a note.

Settled without a question, each the one defensible answer on the record (the research note gives the grounds):

3. A shared machine folder is out: it needs a file another account can write, which abcd never reads (the ruling of 2026-10-14, in the decision log), and it never leaves the machine.
4. No note grants anything (no slot loans, no approvals): a note informs or asks, never authorises.
5. The sub-agent budget ([itd-2610100704270254](../drafts/itd-2610100704270254-an-abcd-run-works-out-how-many-sub.md)) sits beside this intent, not in it: the machine cap is a property of one machine, and hosted sessions run on their own.
6. The host's same-account messages stay a wake-up accelerator only, never the record.
7. A note that cannot be fetched is reported as unread since a time and for a reason, never shown as silence; a sender is shown as claimed until identity can be bound.
8. **Two probes from a real hosted session come first in the build:** whether a push to a branch outside the host's own prefix is accepted, and which forge identity the push arrives as. The product thinker asked for the plan on 2026-10-15 before the probes ran, so both stand as a scope-condition hypothesis; a refused push returns the branch prefix to the product thinker.

Decided by the technical facilitator on 2026-10-15 on the two adversarial reviews, each grounded in a finding the reviews reproduced:

9. **One notes branch per person, not per session.** A person's sessions append to one branch, rebuilding on a lost race (C2). A branch per session could never be swept under a no-deletion rule, so their number would grow with every session for good; one per person grows with the number of people.
10. **Branch names are lower case.** Two names that differ only in case make every fetch on a case-insensitive filesystem fail (reproduced with git 2.52 on macOS), and the no-deletion rule would keep the bad branch for ever; so a username is lower-cased before it names a branch, and the reader skips and names any other branch (C11).
11. **A release hold is read by the land step**, not only shown (C7): a hold that is only shown blocks nothing, which is the failure iss-2609091037191879 records.
12. **The push hook's shape check for notes is the reservation's** (a fast-forward of commits that change no file), not a second trailer parser in shell; a note the reader cannot parse is flagged by C11.
13. **Expiry is counted from when this checkout first fetched the note** (C9), the shape of the clock finding iss-2610142200372105 records for reservations.

Ruled by the product thinker at the planning interview of 2026-10-15:

14. **Notes are fetched only by a command whose job is that** ("Only when a command checks", Q1): `abcd notes`, `abcd peers` and the land step fetch them, and the prompt hook shows only the last fetch from disk, with its age. Brief invariant 7 stands without an exception; the alternative, an opt-in fetch from the prompt hook at most once a minute, was not chosen. Accepted cost: a session that runs none of those commands sees a new note only at its next merge.
15. **The broker draft stays, to be rewritten later** ([itd-2609151838327688](../drafts/itd-2609151838327688-an-opt-in-adapter-to-a-local-message.md); "Keep it, rewrite later", Q2): a messaging service on the person's own network would wake an idle session the moment a note arrives, which notes on the shared remote cannot, but it never reaches another person's computer or a cloud session. It stays a separate draft, re-based on notes as an optional faster carrier when it is next worked on; until then it still describes the retired shared-folder mailbox.
16. **Notes are planned on their own, apart from the session register** ([itd-2609150819440345](../drafts/itd-2609150819440345-which-session-holds-which-worktree.md); "Plan notes on their own", Q3): notes say what a session did, the register would say which sessions are running and what each holds. The register's way of sharing between computers is still undecided, and notes do not wait for it.
17. **Senders are checked against the forge's record of who pushed** ("Confirm as written" on the revised condition, Q6b, after the product thinker asked why it could not be added now): the forge's activity record names the account behind each push and was read successfully on 2026-10-15, so abcd checks it inside `abcd notes`, whose documented meaning includes that read (invariant 7); a forge that cannot say leaves the sender claimed, the native default under invariant 2, with the forge lookup the adapter.
18. **Notes cope with hundreds a day per person** (Q6c, after the product thinker asked whether the limit was necessary): nothing in git limits that rate; what a session can absorb is capped by showing the newest note per item and a count (C18), and the remaining limit, about one push a second per person, is the broker draft's to lift.
19. **Notes carry the sender's GitHub username, and are kept permanently** (the product thinker, 2026-10-15, after asking how abcd could know a sender without recording usernames). Chosen first was "Leave usernames out" (Q6d); on the explanation that the forge's own record of who pushed names the sender to anyone who looks, so a code or an anonymous note hides the name only from a casual reader while costing a pseudonymous commit author, the product thinker chose GitHub usernames ("agreed, github usernames"): they are already public on every commit and pull request, and they are the only form abcd can verify against the forge (C17). The criterion that a note names nobody was struck. Notes are permanent as C15 makes them; deletion would not erase them in any case.
20. **The branches are named `abcd-status/<forge username in lower case>`, one per person** (the product thinker, 2026-10-15, who did not like `abcd-notes/`): the name must stay a branch so cloud sessions can push it, and starts with `abcd-` so it sits beside `abcd-reserved/` in the forge's branch list; `abcd-notes` also invited confusion with git's own notes feature. Asked whether one shared `abcd-status` branch would do, the product thinker kept one branch per person ("ok, agreed keep") on the explanation that a shared branch makes every two sessions posting in the same second collide, most often around a release, and lets one bad push put everyone's notes in doubt. Hidden refs outside the branch list were weighed and not chosen: the cloud proxy refuses them and the forge's rulesets cannot protect them. What a session posts and the commands that post and read it are decision 21's.
21. **A session posts an announcement with `abcd announce`, and a person reads everyone's status with `abcd status`** (the product thinker, 2026-10-15). The word "note" clashed with git's notes feature and with abcd's own memory note, so what a session posts is an announcement; decisions 1 to 20 say "note" for it as they were ruled. `abcd announce <kind> <item>` is in the agents' command set beside `abcd reserve`. `abcd status` is the people's view of what everyone else is doing: announcements, who holds which reservation and any release being cut (C8, C16). `abcd peers` is left as it is for now: its reservation listing duplicates `abcd status`, and the product thinker ruled that `peers` keeps its name for its future meaning, the sessions running now, local first and later over a private network, with its local duplicate check moving to `abcd status --local`; that change and renaming the status board to the dashboard are captured as their own records. The status board's rename also frees the word "status" for this command.

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

## Grounds

- pursued: from today a second person's sessions and cloud sessions work on this project, and none of them can see the messages local sessions use to avoid merging into each other's work; we expect notes and the release hold to stop that, and it would be shown wrong by a session merging into a held release or onto an announced merge in the first two weeks.
