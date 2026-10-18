---
id: itd-2610180912189125
slug: a-session-told-to-work-on-while-its
spec_id: spc-2610180914469489
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610170638184486]
severity: critical
impact: additive
origin: researcher-authored
production_mode: hand-written
related_issues: [iss-2610170619247185]
---

# A session left to work on never stops while it has work

## Press Release

> A session told to work on while its person is away keeps working until they are back. Alice, a solo founder who is also the product thinker, leaves for the evening and says the sessions should work on until 09:00. abcd records that once for the repository, with who set it and when it ends, and repeats it to every session at each start, after each compaction and with every prompt, so no summary and no relay between sessions can drop it. A session that tries to end its turn while it still has work it can do alone (a reservation whose work is not yet handed in, a pull request of its own that has fallen behind or is failing, a branch it has not pushed, a branch stacked on another change that is still open) is handed the next of those items instead. One with nothing left to do takes the next ready item, and may stop only with its next look scheduled before its reservations need renewing. A question it would have put to Alice is decided and recorded instead, and an act only Alice may approve (an advisory's approval, a publication, a new dependency, a secret, a browser step) is queued for their return. When Alice is back and says so, the mode ends and they see the queued steps and any stop the host forced; from then on nothing is refused, and a stop with work left is only pointed out.

## Why This Matters

On the night of 2026-10-16 every session had been told to work on until the
morning, and one sat idle for about ten of eleven and a half hours while it
held work it could do alone (iss-2610170619247185). The instruction lived only
in chat: relayed by a peer, overridden in the session's reading when the
product thinker came back for an hour, then dropped by a compaction summary.
The session ended its turn on a question nobody was there to answer, and a
Claude Code session runs only when something sends it input, so nothing ran
until a peer's message six hours later. The same night a coordinating session
held three finished steps behind a merge order when only their landing order
was fixed.

This is the smallest slice of the checklist draft (itd-2610170638184486) that
removes the stall's core cause, carved out so the security and bug-fix release
waits on it alone (dcn-2610180913379049, dcn-2610180913388445). It needs
nothing on the shared remote, so it does not wait on the coordination intent.

## Mechanism

We expect an autonomous run to stop stalling because the three ways the
session of 2026-10-16 went quiet are each closed where the host gives abcd a
hook: the instruction to work on becomes a record abcd repeats at every start,
compaction and prompt rather than context a summary can drop; ending a turn
with owed work is refused at the Stop hook, the one point every quiet spell
passes through; and a question to an absent person is refused before it can
leave the session waiting. Shown wrong if, in an away window of at least six
hours with owed work seeded at its start, a session following the slice is
idle (no tool call) for more than one hour while owed work remains, or ends
its turn on a question to the person.

## Scope Conditions

- A repository abcd manages, with the local tier and abcd's plugin hooks <!-- cond: cond-2610180914466989 -->
  installed in the host that runs its sessions.
- A host that runs a Stop hook and honours its block decision, a session-start <!-- cond: cond-2610180914464633 -->
  hook on compaction and a prompt hook, as Claude Code documents them; on a
  host without them the mode is still recorded and shown, and nothing is
  enforced.
- Sessions on one computer: the away record is held per repository on that <!-- cond: cond-2610180914460950 -->
  machine, so a session on another computer or in the cloud does not see it.
- The stop check's pull-request reading, made only while the person is away, <!-- cond: cond-2610180914462056 -->
  needs the forge client signed in; when it cannot read the forge in time it
  judges from a recent cached listing or from disk alone, and says so.
- A session that keeps working through the host: the check refuses a stop, it <!-- cond: cond-2610180914469321 -->
  cannot start a session that is not running. A turn the host ends on an API
  error, such as a rate limit, fires no Stop hook and cannot be continued: it
  is recorded, and only a wake the session had already scheduled resumes it.

## What's In Scope

- **The away mode.** `abcd mode autonomous [--until <time>]` records who set
  it, when, and when it ends, at most seven days on; `abcd mode back` ends it,
  prints the queue and the recorded stops, and records who ended it. One
  record per repository on the machine, shared by every working copy, changed
  only under a lock (dcn-2610180913399042, dcn-2610180913399499,
  dcn-2610180937181736).
- **Repeated, never summarised away.** While it holds, the session-start hook
  (every source, compaction included) and the prompt hook (every prompt) add
  one short block to the session's context (dcn-2610180913406972), and the
  prompt hook shows the person one line naming the away mode
  (dcn-2610180937181736).
- **The stop check.** A plugin Stop hook asks `abcd hook stop`. While the
  person is away a session is only working or waiting: a turn's end with owed
  work is refused, naming the next item, and a stop is allowed only with a
  wake that fires before its reservations need renewing; the host's cap of
  eight continuations is the outer bound, and a stop it forces is recorded.
  A StopFailure hook records a turn ended by an API error. While the person
  is present the check reads disk only and points the work out
  (dcn-2610180913413137, dcn-2610180913421109, dcn-2610180936577720,
  dcn-2610180936579498, dcn-2610180937175629).
- **Owed work**, in exactly these terms: a live reservation under this
  working copy's key whose renewal is due or whose work no open pull request
  carries yet; an open pull request of the session's own that is behind the
  default branch or failing its checks; a branch of its own with commits not
  pushed; and a branch of its own stacked on a change that is still open,
  while it is behind, failing or has no preflight receipt, since only its
  landing waits for the base. A branch is the session's own when this copy
  ever checked it out, a run in this copy cut it as a lane, or a reservation
  under this copy's key links it (dcn-2610180913416683,
  dcn-2610180936568358, dcn-2610180936571036).
- **The question reroute.** While the person is away, the guard refuses every
  question on the host's question tool. A question about an act only the
  person may approve (a per-act approval such as an embargo close's digest,
  a publication or visibility change, a new dependency, a secret, credential
  or App, a website or browser step, a destructive act, a spend) is queued
  with `abcd mode queue "<step>"`; every other question is decided and
  recorded with `abcd decide` (dcn-2610180913431493, dcn-2610180937177502).

## What's Out of Scope

- The rest of itd-2610170638184486: the per-session checklist, trigger
  promises, abcd's own timed wakes, the pre-run blocker sweep, the shared and
  published options and the phone.
- Idle time a coordinator sees across sessions, which comes with the
  coordination intent's central view in `abcd status`.
- Moved back to itd-2610170638184486 by review round 1
  (dcn-2610180937181929): the report shown once on the person's return (the
  queue, the window's decisions gathered across working copies, the recorded
  stops and who ended the mode), the status line naming the away mode, and
  the bare `abcd mode` print of the record, the queue and the owed items.
- Starting or waking a session that is not running.
- Each autonomous session's own forge credential, owned by the coordination
  intent.

## Acceptance Criteria

- **Given** Alice, the product thinker, runs `abcd mode autonomous --until
  09:00` in one working copy at 20:00, **when** a session in a sibling working
  copy of the same repository starts, **then** its context carries the away
  block naming 09:00 and who set it, with no message from any peer.
- **Given** the away mode holds, **when** a session's context is compacted,
  **then** the session-start hook fired with source `compact` adds the away
  block again, and so does the prompt hook on the next prompt.
- **Given** the away mode holds and the session's own pull request, opened
  and checked out in its working copy before the mode was set, is behind the
  default branch, **when** the session tries to end its turn, **then** the
  stop is blocked and the reason names that pull request as the next item and
  says how many items remain.
- **Given** the away mode holds and a lane branch that a run in the
  session's working copy cut in its own worktree is stacked on a pull request
  that is still open and is behind the default branch, **when** the session
  tries to end its turn, **then** the stop is blocked and the reason asks for
  the merge from the default branch, the fixes, the review and the preflight
  now, and says only the landing waits for the base; **and when** the branch
  is up to date, green and has a preflight receipt, **then** it is waiting on
  its base and is not owed.
- **Given** the away mode holds and the session holds a live reservation whose
  work no open pull request carries, or a branch with unpushed commits,
  **when** it tries to end its turn, **then** the stop is blocked naming that
  item.
- **Given** the stop was blocked and the session replied "awaiting the
  person" with no tool call and no wake, **when** it tries to stop again with
  the owed set unchanged, **then** the stop is blocked again; **and when** the
  host's cap of continuations ends the turn, **then** it is recorded as
  stopped with work owed, shown when Alice runs `abcd mode back`.
- **Given** the away mode holds and nothing is owed, **when** the session tries
  to stop with no qualifying wake, **then** the stop is blocked, telling it to
  take the next ready item first and only then schedule its next look before
  its earliest reservation renewal is due; a wake set for 09:00 the next
  morning, or a background shell task, does not qualify; **and when** it stops
  with a wake that fires in time, **then** the stop is allowed and the
  session is waiting, with that time kept as its next look.
- **Given** no away mode holds (Alice is at the terminal), **when** a session
  with owed work ends its turn, **then** the stop is allowed, one line shown
  to Alice names the owed work read from disk, and the forge is not called.
- **Given** the away mode holds, **when** the agent puts a product question to
  Alice on the host's question tool, under any chip, **then** the guard
  refuses it, telling the agent to decide and record it with `abcd decide`
  and carry on; **and when** it asks Alice to approve closing a security
  advisory, **then** the same refusal tells it that an act only Alice may
  approve is queued with `abcd mode queue`, and it never supplies the
  approval itself.
- **Given** the away mode holds, **when** Alice sends any prompt, **then** one
  line shown to Alice names the away mode, its end, who set it and how to end
  it.
- **Given** a window in which one step was queued and the host forced one
  stop, **when** Alice runs `abcd mode back`, **then** they see the queued
  step and the stop, and the mode ends recording who ended it.
- **Given** a stop check reading the forge while Alice runs `abcd mode back`
  in another session, **when** the check finishes, **then** the mode stays
  ended.
- **Given** the away mode holds, **when** a turn ends on a rate limit, **then**
  the stop and its error type are recorded and shown when Alice runs `abcd
  mode back`.
- **Given** `--until` names a time already past or more than seven days
  ahead, or the repository has no local tier, **when** `abcd mode autonomous`
  runs, **then** it refuses, names the reason, and writes nothing.
- **Given** the forge cannot be read within ten seconds during a stop check,
  **when** the session tries to stop while away, **then** the check judges
  from a listing cached in the last half lease, naming its age, or from the
  disk kinds alone, saying the pull requests were not read.
- **Given** the slice is built, **when** a live Claude Code session with the
  plugin from the branch runs an away window with seeded owed work, a
  compaction and an attempted stop, **then** a captured transcript shows the
  away block after compaction, the blocked stop naming the next item, and the
  refused question.

## SOTA

Surveyed 2026-10-18 against the host's current hook reference
(https://code.claude.com/docs/en/hooks) and the record:

- **The host's Stop hook with a block decision:** documented and current.
  Adopted as the only point every quiet spell passes through; the host bounds
  it with `stop_hook_active` and a cap of eight consecutive continuations,
  reset by each tool call.
- **The host's session-scoped scheduled wakes** (CronCreate, ScheduleWakeup,
  `/loop`), reported to a Stop hook as `session_crons`: young. Adopted only as
  evidence that a waiting session will look again; abcd schedules nothing.
- **Re-injection through the session-start hook's compact source and the
  prompt hook:** documented. Adopted for the away block, as the checklist
  draft's format decision already chose for the checklist.
- **A supervisor process that restarts or prompts idle sessions:** a common
  pattern, not adopted: it needs a process that stays alive, which the
  coordination intent's on-demand pass rules out, and it cannot tell owed work
  from a finished session without the same reading this slice adds.
- **The host's agent-team idle hook (TeammateIdle):** young, bound to one agent
  team in one session, so it cannot reach sessions in separate working copies.
  Not adopted.

Path taken: 2. The host's hooks are the native floor; the away record, the
owed-work reading and the verdict are abcd's, behind a core seam that a
second host's hooks can call.

## Open Questions

_None. Every question of the slice was settled on delegation; the decisions
are listed below._

## Decisions

Each is the facilitator's, on delegation, and restates the product thinker's
ruling or the ground it rests on; the full text is in the decision record.

1. **Fixed first** (dcn-2610180913379049): the stall is fixed before the
   security and bug-fix release, not deferred.
2. **Its own intent** (dcn-2610180913388445): the slice builds on the
   checklist draft rather than planning it.
3. **The away record** (dcn-2610180913399042): set by a verb on the person's
   word, ended by its time or their word, never by a prompt.
4. **Per repository on the machine** (dcn-2610180913399499).
5. **Repeated at start, compaction and every prompt**
   (dcn-2610180913406972).
6. **Owed work** (dcn-2610180913416683): four kinds, read from disk and one
   forge listing; brief invariant 7 gains the clause.
7. **The stop check and its bound** (dcn-2610180913413137).
8. **Nothing owed means a scheduled look** (dcn-2610180913421109).
9. **No question waits while away** (dcn-2610180913431493).
10. **Two lists on return, derived** (dcn-2610180913431985).
11. **Host-agnostic core, reachable without a host**
    (dcn-2610180913445272); its status-line and bare-print halves moved by
    decision 19.

Review round 1 of 2026-10-18 refined them:

12. **"Own" without a window, this copy's reservations only**
    (dcn-2610180936568358).
13. **Stacked branches cleared; the forge read only while away**
    (dcn-2610180936571036).
14. **Away means working or waiting** (dcn-2610180936577720).
15. **A wake qualifies by its next fire time** (dcn-2610180936579498).
16. **API-error stops recorded; prose questions blocked once**
    (dcn-2610180937175629).
17. **Questions routed by the act** (dcn-2610180937177502).
18. **The record's lock, checks and seven-day cap** (dcn-2610180937181736).
19. **The return report, status line and bare print move to the draft**
    (dcn-2610180937181929).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: we expect an autonomous session to stop going idle with owed work, because the instruction to work on becomes a record abcd repeats at start, compaction and prompt, and the Stop hook refuses a turn's end while owed work exists; shown wrong if, in a six-hour away window with seeded owed work, a session following the slice is idle over an hour while owed work remains, or ends its turn on a question to the person
