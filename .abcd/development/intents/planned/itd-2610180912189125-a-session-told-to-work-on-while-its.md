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

> A session told to work on while its person is away keeps working until they are back. Alice, a solo founder who is also the product thinker, leaves for the evening and says the sessions should work on until 09:00. abcd records that once for the repository, with who set it and when it ends, and repeats it to every session at each start, after each compaction and with every prompt, so no summary and no relay between sessions can drop it. A session that tries to end its turn while it still has work it can do alone (a reservation whose work is not yet handed in, a pull request of its own that has fallen behind or is failing, a branch it has not pushed, a branch stacked on another change that is still open) is handed the next of those items instead. One with nothing left to do may stop only with its next look scheduled. A question it would have put to Alice is decided and recorded instead, and a step only Alice's hands can do is queued for their return. In the morning Alice sees those steps and the decisions taken while they were away, and while they are at the terminal nothing is refused: a stop with work left is only pointed out.

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
- The stop check's pull-request reading needs the forge client signed in; when <!-- cond: cond-2610180914462056 -->
  it cannot read the forge it judges from disk alone and says so.
- A session that keeps working through the host: the check refuses a stop, it <!-- cond: cond-2610180914469321 -->
  cannot start a session that is not running.

## What's In Scope

- **The away mode.** `abcd mode autonomous [--until <time>]` records who set
  it, when, and when it ends; `abcd mode back` ends it. One record per
  repository on the machine, shared by every working copy
  (dcn-2610180913399042, dcn-2610180913399499).
- **Repeated, never summarised away.** While it holds, the session-start hook
  (every source, compaction included) and the prompt hook (every prompt) add
  one short block to the session's context (dcn-2610180913406972).
- **The stop check.** A plugin Stop hook asks `abcd hook stop`, which refuses
  to end a turn with owed work while the person is away, names the next item,
  and is bounded so it never spins; while the person is present it only points
  the work out (dcn-2610180913413137, dcn-2610180913421109).
- **Owed work**, in exactly these terms: a live reservation whose renewal is
  due or whose work no open pull request carries yet; an open pull request of
  the session's own that is behind the default branch or failing its checks; a
  branch of its own with commits not pushed; and a branch of its own stacked on
  a change that is still open, whose merge from the default branch, fixes,
  review and preflight are owed now while only its landing waits
  (dcn-2610180913416683).
- **The question reroute.** While the person is away, the guard refuses every
  question on the host's question tool: a decision is taken and recorded with
  `abcd decide`, and a step only the person's hands can do is queued with
  `abcd mode queue "<step>"` (dcn-2610180913431493).
- **The person's return.** The queue and the decisions recorded in the window,
  shown once on the next prompt after the mode ends (dcn-2610180913431985).
- **Shown and reachable.** The status line names the away mode and its end; a
  bare `abcd mode` prints the record, the queue and the owed items
  (dcn-2610180913445272).

## What's Out of Scope

- The rest of itd-2610170638184486: the per-session checklist, trigger
  promises, abcd's own timed wakes, the pre-run blocker sweep, the shared and
  published options and the phone.
- Idle time a coordinator sees across sessions, which comes with the
  coordination intent's central view in `abcd status`.
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
- **Given** the away mode holds and the session's own pull request is behind
  the default branch, **when** the session tries to end its turn, **then** the
  stop is blocked and the reason names that pull request as the next item and
  says how many items remain.
- **Given** the away mode holds and a branch of the session's own is stacked
  on a pull request that is still open, **when** it tries to end its turn,
  **then** the stop is blocked and the reason asks for the merge from the
  default branch, the fixes, the review and the preflight now, and says only
  the landing waits for the base.
- **Given** the away mode holds and the session holds a live reservation whose
  work no open pull request carries, or a branch with unpushed commits,
  **when** it tries to end its turn, **then** the stop is blocked naming that
  item.
- **Given** the stop was blocked once and the host reports the session is
  already continuing because of a stop hook, **when** it tries to stop again
  with the owed set unchanged, **then** the stop is allowed and recorded as
  stopped with work owed, and that record appears on Alice's return.
- **Given** the away mode holds and nothing is owed, **when** the session tries
  to stop with no scheduled wake and no background task, **then** the stop is
  blocked once asking it to schedule its next look in under an hour; **and
  when** it stops with a wake scheduled, **then** the stop is allowed and the
  session is waiting.
- **Given** no away mode holds (Alice is at the terminal), **when** a session
  with owed work ends its turn, **then** the stop is allowed and one line
  shown to Alice names the owed work.
- **Given** the away mode holds, **when** the agent puts a question to Alice on
  the host's question tool, **then** the guard refuses it, telling the agent to
  decide and record it with `abcd decide` and carry on, or, for a step only
  Alice's hands can do, to queue it with `abcd mode queue`.
- **Given** a window in which two decisions were recorded and one step was
  queued, **when** Alice runs `abcd mode back` or the window's time passes and
  they send the next prompt, **then** they see the queued step and both
  decisions with their text, once.
- **Given** the away mode holds, **when** Alice looks at the status line,
  **then** it names the away mode and its end; **and when** Nia, the technical
  facilitator, runs a bare `abcd mode` with no plugin host, **then** it prints
  the record, the queue and the owed items.
- **Given** `--until` names a time already past, or the repository has no local
  tier, **when** `abcd mode autonomous` runs, **then** it refuses, names the
  reason, and writes nothing.
- **Given** the forge cannot be read during a stop check, **when** the session
  tries to stop while away, **then** the check judges from the disk kinds alone
  and the reason says the pull requests were not read.
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
    (dcn-2610180913445272).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: we expect an autonomous session to stop going idle with owed work, because the instruction to work on becomes a record abcd repeats at start, compaction and prompt, and the Stop hook refuses a turn's end while owed work exists; shown wrong if, in a six-hour away window with seeded owed work, a session following the slice is idle over an hour while owed work remains, or ends its turn on a question to the person
