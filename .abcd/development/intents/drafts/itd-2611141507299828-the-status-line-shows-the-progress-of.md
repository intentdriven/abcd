---
id: itd-2611141507299828
slug: the-status-line-shows-the-progress-of
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2611031342000215, itd-2610281253044600]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
blocked_by: [itd-2611141610197753]
---

# The status line shows the progress of work that runs without the person

## Press Release

> The status line shows the progress of work that runs without the person. With no command typed, the row at the bottom of a session says three things. It says what an unattended drive is doing, in one short phrase: working, waiting on something outside it, or stalled, with the time since when and whether it needs the person, followed by how many lanes have landed, are in flight, were set aside and were handed back. It says where this session's own pull requests stand: armed, in the merge queue, merged, or dropped out of the queue. And it says that a long check is running and how far along it is. A drive that stops making progress reads as stalled on the next refresh, in words as well as colour. On a narrow window the row drops detail in a declared order and keeps its badge. Each of the three parts switches off on its own, and a repository with no drive, no pull request and no check running shows exactly the row it shows today.

> "I came back after a day and the row said the drive had been stalled since Thursday afternoon and needed me. I did not have to ask anyone whether it was working." — Iris, a product thinker (they/them)

> "I stopped opening the log to find the last line that was not a wait. The row tells me the drive is waiting on the usage window, and my two pull requests are still in the queue." — Nia, a technical facilitator (they/them)

## What It Looks Like

The examples show the row without colour. The words, the counts' labels and the order in which parts are dropped are illustrations: the order is an open question below, and the examples show one of its defensible answers (the drive first). `Henry` is the session's name. The examples are illustrative: 20 landed and 79 handed back are the figures of the drive described in the incident note, and every other figure (lanes in flight and set aside, the pull-request counts, the times and the preflight's steps) is made up to show the layout.

A drive that is working, at 120, 80 and 40 columns:

```text
abcd-managed · Henry · abcd:main · drive working · 20 landed · 2 in flight · 3 set aside · 79 handed back · PRs 2 queued
abcd-managed · Henry · drive working · 20 landed · 2 in flight · PRs 2 queued
abcd-managed · drive working · 20 landed
```

A drive that is waiting on something outside it:

```text
abcd-managed · Henry · abcd:main · drive waiting on the usage window since 14:02 · 20 landed · PRs 2 queued, 1 armed
abcd-managed · Henry · drive waiting on the usage window since 14:02 · 20 landed
abcd-managed · drive waiting since 14:02
```

A drive that is stalled:

```text
abcd-managed · Henry · abcd:main · ! drive stalled since Thu 16:40 on a missing record · needs you · 20 landed
abcd-managed · Henry · ! drive stalled since Thu 16:40 · needs you · 20 landed
abcd-managed · ! drive stalled Thu 16:40
```

No drive, with this session's pull requests and a preflight running:

```text
abcd-managed · Henry · abcd:main · PRs 1 armed, 2 queued, 20 merged, 1 dropped out · preflight 14 of 17, 31m
abcd-managed · Henry · PRs 1 armed, 2 queued, 1 dropped out · preflight 14 of 17
abcd-managed · PRs 1 dropped out
```

## Why This Matters

An unattended drive (`abcd drain --drive`, itd-2610222215485265) ran for three days on one machine. The person had to ask "is the drain working?", and answering took reading the log for the last line that was not a wait, comparing times and counting (event 5 of the incident note, `.abcd/development/research/notes/drive-operations-2026-11.md`). For 23 of its first 52 hours the drive did no new work and said so nowhere a person would notice (event 4). Its pull requests left the merge queue four times and stayed out until a second session noticed (event 3). The drive prints its state as a stream of similar lines, and there is no one place that says working, waiting or stalled.

abcd already writes a row into the host's status line, and the person already looks there: it leads with the badge that says whose answer is owed (itd-200, itd-2609212130146198) and names the session (itd-2610111707085098). It is the one surface a person sees without typing. Work that runs without the person belongs on it for that reason: the person is by definition not watching that work, so its state has to reach them where they already look.

### Who it is for

- The product thinker (Iris), who starts a drive or is told one is running and comes back to it hours or days later. They need one phrase: is it working, and does it need me.
- The technical facilitator (Nia), who works in a session while a drive, their own pull requests and a preflight run beside them. They need the counts, the queue state of their own pull requests and how long the check has left, without leaving the session.

### What it replaces

- Reading the drive's log to learn its state, and asking a session to read it.
- Opening the forge to see whether a pull request armed earlier is still in the merge queue.
- Switching to the terminal where a preflight runs to see whether it has finished.

It replaces none of the surfaces that hold the detail: the log, the board and the forge stay where they are. The row says that something needs a look and which thing.

### Why this is a new intent

Two drafts already extend the row. itd-2610281253044600 adds one element, the title of the work a session holds, and its owner's decision of 2026-10-28 is about where that title goes. itd-2611031342000215 sets the row's views, its dropping order by width and its configuration table, from two labs and its recorded decisions. This intent is about a different subject, the state of work the person is not attending, and it has its own sources of state, its own failure (a row that says "working" over a drive that died) and its own questions for the owner. Adding it to either draft would reopen decisions already recorded there. It is filed as a new draft that builds on both: it takes the row's width handling, per-element settings and warning style from itd-2611031342000215, and the match between a session and what it holds from itd-2610281253044600.

## Mechanism

- We expect the person to stop asking whether the drive is working, because the drive's own stored state is shown where they already look, in a phrase that separates working from waiting from stalled and gives the time. Shown wrong if people still open the log, or ask a session, to learn the drive's state after this ships.
- We expect a stalled drive to be noticed within the same working session rather than a day later, because the word and its time are on a row that redraws while the person works. Shown wrong if a drive stays stalled for hours while a session with the row is in use on the same machine.
- We expect a pull request that drops out of the merge queue to be armed again sooner, because the session that armed it shows the count. Shown wrong if dropped pull requests are still found by reading the forge.
- We expect the row to stay cheap, because every part reads a small state file that the work itself already writes, and the row derives nothing from logs, from git history or from the forge. Shown wrong if the added parts push ordinary refreshes towards the row's time ceiling, or if people switch the parts off for slowness.

## Scope Conditions

- A drive that keeps one stored line of state: working, waiting, stalled or stopped; since when; on what; and what a person can do. That line is defined by itd-2611141610197753 (a drive carries on from one command, and says in one line when it cannot); this intent reads the line and does not define it. Until the line exists the drive part is absent, never derived from the log.
- The stored line carries the time it was last written, so that a drive that died without writing a final state can be told from one that is working. The row judges age from that time alone; it does not read the process table. The spec of itd-2611141610197753 stores a next-write time for such a reader and says what it costs: a drive that died during a long wait can read as waiting for up to an hour.
- A host that runs the status command on a timer as well as on its own events (the refresh interval itd-2611031342000215 sets at install). On a host that refreshes only on events, "within one refresh" is as long as the gap between events, and a stall may sit unseen while the session is idle.
- A host that passes the terminal's width to the status command in `COLUMNS` (itd-2611031342000215). Without the width the row cannot choose what to drop and renders in full, with the badge-first order as its only protection, as today.
- A session open in a checkout of the repository on the machine where the drive runs. A drive on another machine or a hosted runner, and a person with no session open, are outside what a status line can reach.
- Pull-request state is as fresh as the last event that recorded it locally. The row never asks the forge, so a pull request the forge removed from its queue shows as dropped out only once something other than the status command has read that.
- A managed checkout: the row renders only there, and the person's previous status command runs everywhere else, unchanged.

## What's Out of Scope

- Defining the drive's state, its words or when a drive counts as stalled. That is itd-2611141610197753's.
- Acting on what the row shows: re-arming a pull request, restarting a drive, setting a lane aside. The row reports.
- The drive's own terminal output and its log.
- The product thinker's dashboard (itd-2610032150577708) and any view outside the terminal.
- The agents, spend and token figures of the launch ledger (itd-2611031342011261), which reach the row through that intent's remaining spec.
- A notification outside the status line. Whether a stalled drive may also raise one is an open question below; if the answer is yes it is a separate change.
- Work the session itself is doing in the foreground. The row's existing elements and itd-2610281253044600 cover it.

## Acceptance Criteria

- **Given** a drive whose stored state says working, **when** the status line refreshes in a session in that repository, **then** the row shows the word `working` for the drive and the counts of lanes landed, in flight, set aside and handed back, and no command was typed to produce them.
- **Given** a drive whose stored state says waiting, **when** the row renders, **then** it shows `waiting`, what the drive waits on and the time since when, each taken from the stored state.
- **Given** a drive whose stored state changes to stalled, **when** the status line next refreshes, **then** the row shows `stalled`, the time since when and what it stalled on, and, where the stored state says a person can act, that it needs the person; the change is visible within one refresh of the state being written.
- **Given** a drive whose stored state says stopped, **when** the row renders, **then** it shows `stopped` and the time since when, and, where the stored state names an act, that the drive needs the person.
- **Given** a stored drive state whose last-written time is older than the freshness bound the state declares, **when** the row renders, **then** it says the drive has not been heard from since that time and never shows `working`.
- **Given** a drive state file that is missing, unreadable, malformed or of an unknown schema version, **when** the row renders, **then** the drive part is absent, one line on stderr names why, the rest of the row renders, and the row never shows a drive state it could not read.
- **Given** pull requests this session armed, **when** the row renders, **then** it shows how many are armed, in the merge queue, merged and dropped out, counts only this session's own, and shows a state with a count of zero not at all.
- **Given** a pull request recorded as dropped out of the merge queue, **when** the row renders, **then** the dropped count carries the row's warning mark and the words `dropped out`.
- **Given** a preflight running in this checkout, **when** the row renders, **then** it shows that a preflight is running and how far along it is; **given** that it has ended, **when** the status line next refreshes, **then** the running indication is gone.
- **Given** no drive state, no pull request recorded for the session and no preflight running, **when** the row renders, **then** its output is byte for byte what the same payload, checkout and settings produce before this change.
- **Given** any refresh, **when** the status command runs, **then** it makes no network call, starts no process that makes one, starts no subprocess it does not start today, and asks git nothing it does not ask today; a test over the status path holds each of the four.
- **Given** any refresh, **when** the status command runs, **then** all of abcd's own work, the three parts included, finishes inside the existing ceiling of 500 ms, of which git may take 300 ms (`statuslineBudget` and `statuslineGitBudget` in `internal/surface/cli/statusline.go`); a part whose state is not read in time is dropped and named on stderr, and the rest of the row renders.
- **Given** a drive that has landed, set aside or handed back any number of lanes, and a session with any number of past pull requests, **when** the row renders, **then** the status command reads a bounded number of small files that does not grow with that history, and lists no directory whose size grows with it.
- **Given** any refresh, **when** the status command runs, **then** it writes no file it does not write today: the read-only guard on the status path passes with no new write admitted.
- **Given** text the row did not compose itself (what a drive waits on, an issue's title, a branch name, a check's name), **when** it is rendered, **then** it has passed the same terminal-safety pass the row's other untrusted text passes: an escape sequence, a control character, a bidirectional override or a hidden rune in it renders inert, and the text is cut at a declared length.
- **Given** the row rendered without colour (the text mode, or a terminal that shows none), **when** a person reads it, **then** working, waiting, stalled and not-heard-from are told apart by their words, a pull request that dropped out by its words, and every warning carries `!`; no state is carried by colour alone.
- **Given** `COLUMNS` in the environment and a row wider than it allows, **when** the row renders, **then** it fits, every prefix of it still begins with the badge, the parts drop in the order the table declares, and within the drive part the state word outlasts its counts and its detail.
- **Given** `COLUMNS` absent, **when** the row renders, **then** nothing is dropped and the row renders in full.
- **Given** one of the three parts switched off in the person's status-line setting, or in the repository's configuration, **when** the row renders, **then** that part is absent, its state is not read, and the other two parts and every existing element are unchanged.
- **Given** the same payload, checkout and settings, **when** the bare board and `abcd statusline --json` render, **then** they carry this session's pull requests and the running check in the same words as the row, from the one composition, each with its plain form. For the drive this draft yields to itd-2611141610197753: the board shows the drive's state as the one line that intent composes (its sixth criterion), and the row's shorter phrase is the row's own, drawn from the same stored value.
- **Given** a checkout abcd does not manage, **when** the status command runs, **then** the person's previous status command runs as it does today and none of the three parts is read or shown.

## Open Questions

Each question below is the owner's to decide and is not settled by any record. None is answered here.

1. "On a narrow line, which of the three parts wins: the drive, this session's pull requests, or the running check?"
   - The drive first. A stall is the costliest thing to miss. Cost: a facilitator on a narrow pane loses sight of their own pull requests while a healthy drive takes the space.
   - Whichever part carries a warning first, then a fixed order. Cost: the row's layout moves as states change, which makes it harder to read at a glance.
   - Per view, as itd-2611031342000215 does for its elements: the product thinker keeps the drive, the facilitator keeps their pull requests. Cost: three more entries in each view's dropping order to tune, and a stall can be the part a facilitator's view dropped.
   - A drive that is working collapses to one word and yields; only waiting and stalled hold their place. Cost: the counts are the first thing lost, and they are what shows progress.

2. "May a stalled drive also raise a desktop notification?"
   - No: the status line is the only surface. Cost: a person with no session open, or not looking at the terminal, learns of a stall only on return, which is the three-day case.
   - Yes, once per stall, when the state changes. Cost: a second surface with its own consent, its own settings and a platform-specific sender; it cannot be raised from the status path, so the drive raises it.
   - Yes, and again at intervals while the stall lasts. Cost: the above, plus a notifier people silence.

3. "Are the counts since the drive started, or for today?"
   - Since the drive started. They match the drive's own summary. Cost: on a drive that runs for weeks the figures grow large and stop showing whether anything moved recently.
   - For today, in the person's local day. Cost: a count that resets at midnight reads as lost work, and it differs from the drive's summary.
   - Since the person last looked, cleared by an acknowledgement. Cost: an acknowledgement is state the row cannot write, so a command or a hook has to.

4. "When a drive needs the person, does the badge change, or does only the drive's part say so?"
   - Only the drive's part says so; the badge keeps its meaning, whose answer the agent loop in this session is waiting on. Cost: the one element that survives every cut does not carry the stall.
   - The badge turns to waiting on the role the drive needs. Cost: the badge's rule (itd-2609212130146198: a question is refused until the mode is set, and the answer resets it) gains a second writer that no answer in this session resets, and the drive has to name a role.

5. "Which sessions show the drive: every session in the repository, or one?"
   - Every session in any checkout or worktree of the repository on that machine. Cost: the same phrase on every row, including sessions doing unrelated work.
   - Only sessions in the checkout the drive runs from. Cost: a person working in a worktree, which is where concurrent sessions are told to work, never sees it.
   - Every session, in full in one and as the state word alone in the rest. Cost: a rule for which session is the one.

6. "What keeps a pull request's state current, given that the status command never starts a network call?"
   - Only events that already read the forge write it: arming, the drive's own polling, a peers read. Cost: between such events the row can show a pull request as queued after the forge dropped it, for as long as nothing else looks.
   - A refresher that runs apart from the status command (the drive, a hook at the session's prompt, or a scheduled job) writes it at an interval. Cost: a new background reader of the forge with its own rate limit and credential use, and someone has to start it.
   - The detached refresh that decision 18 of itd-2611031342000215 describes, filled at most once a minute. Cost: something has to launch it; if the status command launches it, the status path starts a process that makes a network call, which the owner's words for this intent rule out, so that decision and this intent would need reconciling.

7. "How long do merged and dropped-out pull requests stay on the row?"
   - Merged for the rest of the session; dropped out until it is armed again or closed. Cost: a long session's merged count grows and takes width.
   - Merged for a fixed time after merging, an hour for example. Cost: a person away for longer never sees it.
   - Merged is never shown; only armed, queued and dropped out are. Cost: the row no longer shows that anything finished.

8. "What does 'how far along' mean for a running check, whose checks count, and is its result shown?"
   - Step k of n, from the check's own list of steps. Cost: steps differ in length by orders of magnitude, so 14 of 17 can mean most of the time is still ahead.
   - Elapsed time against the last complete run's duration on this machine. Cost: needs a stored duration, and reads wrong on a loaded machine or after the suite changes.
   - Whose: only a preflight this session started; or any preflight running in this checkout and its worktrees; or any on the machine, including one waiting at the load gate. Cost of the wider answers: more state files to find inside the time ceiling, and other sessions' work on this session's row.
   - Result: gone when it ends; or its pass or failure kept until the next prompt. Cost of keeping it: a failed check stays on the row with nothing to clear it but time or a write.

9. "Does the product thinker's view show the drive in the same words as the facilitator's?"
   - The same words in both. Cost: `in flight`, `set aside` and `handed back` are the drive's terms, and the product-thinker view of itd-2611031342000215 carries outcomes and no internals.
   - The product thinker sees the state phrase and one count of finished work. Cost: a second wording to keep in step with the first.

10. "With several drives on one machine for one repository, or a drive per checkout, which does the row show?"
    - The one for this checkout's repository that changed state most recently. Cost: a stalled older drive is hidden behind a working newer one.
    - The worst state among them, with a count. Cost: more than one state file read on every refresh.
    - One drive per repository is assumed, and a second is not shown. Cost: wrong the day two run.

## Typed Links

- builds_on: itd-2611031342000215 (the row's views, its dropping order by width, its per-element settings, its warning style and the rule that the row never waits on the network).
- builds_on: itd-2610281253044600 (the match between a session and the work it holds, which "this session's own pull requests" depends on; its second open question is the same question).
- related: itd-2610222215485265 (the drive whose state the row shows), itd-200, itd-2609212130146198 and itd-2610111707085098 (the row, its badge and the session's name, all shipped), itd-2611031342011261 (the launch ledger, whose remaining spec has an open question on whether the status path may gain a second written file; this intent adds no write, so it does not wait on that answer, and it must not be read as settling it).
- depends on: itd-2611141610197753, which defines the drive's one stored line of state (in `blocked_by`).
- evidence: `.abcd/development/research/notes/drive-operations-2026-11.md`, events 3, 4 and 5.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
