---
id: itd-2611251809580662
slug: a-drive-keeps-working-the-ledger-until
spec_id: spc-2611251810508418
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2611141610197753]
severity: minor
related_issues: [iss-2610231127555243, iss-2611050840353717, iss-2611251810162964]
refines: [itd-2610222215485265, itd-2610161016394967, itd-2610161016401115, itd-2611041656024409]
related_intents: [itd-2611062149308723, itd-2611141610201875]
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A drive keeps working the ledger until the time the person named

## Press Release

> **A drive keeps working the ledger until the time the person named.** A person starts a drive in any abcd-managed repository with `abcd drain --drive`. It runs until an end time, seven days on unless the person or the repository says otherwise, and it needs no restart in between: when one drain is finished it begins the next itself, and when nothing is eligible it waits, looks again and takes the next issue that another machine captures or a person puts back in the queue. Each drain works one issue at a time: it reserves an issue, fixes it, queues its pull request and reserves the next. A person who wants more done at once starts the same command in another terminal, on another machine or in another account. The repository sets how many issues one person may hold in all, so every machine reads the same number. Each drive publishes one line of state under a label the person chose, and `abcd status` on any of the machines shows all of that person's drives, the issues they hold and their pull requests in one place.

> "I started it on a Monday and answered one question. It told me it would end the next Monday, and that a reboot would end it sooner, with the command to carry on. It began a new drain eleven times that week and I typed nothing." — Iris, a product thinker (they/them)

> "Three drives work one ledger: two machines and a second account. Each takes one issue at a time, and the one view shows me nine held of the ten the repository allows, which three are being worked and which six wait in the merge queue." — Maya, an autonomous-development practitioner (they/them)

> "I am asked whether the drains are working. I run the status command at my desk and read three lines, one per drive. The laptop's says stopped, not heard from since 16:40, so I know which machine to walk to." — Nia, a technical facilitator (they/them)

## Why This Matters

A drive ends at its lane cap, eight unless the start says otherwise, and the cap cannot be changed while the drain runs. One drain on this repository's own ledger began on 7 November with a cap of 200 and stood at 189 lanes on 24 November, so the next start would have ended after eleven. A drive also ends when nothing is eligible, which on a ledger several machines write to is true only for a while. Every restart needs a person, and a start is also the moment the drive's checks against the shared remote can refuse (`.abcd/development/research/notes/drive-operations-2026-11.md`, `autonomous-run-2026-11-14.md`).

Sharing the queue between accounts is delivered (itd-2610161016394967). Three things about several machines are not.

- **The limit on live reservations is read from each machine's own file, and counted over the person.** One person may hold ten in all. Each lane holds one until its pull request merges, so several drives wait at the cap on each other, and two machines can read two different numbers for the same person (iss-2610231127555243).
- **Nothing says how much one drain may have in work.** The only gate on a drain's next lane is the person's total, which mixes the lane being worked with the lanes waiting to merge.
- **The one line that says whether a drive is working is shown only in that drive's checkout** (itd-2611141610197753). Answering for three machines means visiting three, and `abcd peers` shows another account's lanes at their last renewal and nothing of a drive that has stopped.

### Who it is for

- The product thinker (Iris), who wants the ledger worked for a week or more and decides once how long.
- The autonomous-development practitioner (Maya), who runs drives in more than one terminal, machine or account.
- The technical facilitator (Nia), who answers for all of them.

### What a person does

- Starts a drive once per terminal, machine or account, and answers one question. Names an end time or a lane cap only when the defaults do not suit.
- Gives each machine a label once, in that machine's own configuration.
- Stops a drive, or lets the end time come.
- Runs `abcd status` on any machine to see every drive that is theirs.

### What a person sees

- At the start: the end time, with the lane cap if one was given, before the yes; on `?` and in the full plan, how many lanes this drain works at once, how many reservations the repository allows one person, and that a reboot, a logout or a crash ends the drive, with the one command that carries on, which is printed once after the yes.
- One line each time the drive begins a new drain, and one when it waits on an empty queue.
- In `abcd peers` and in the drive's waits: the lanes this drain has in work against its limit, and the reservations the person holds against the repository's ceiling.
- In `abcd status` and on the board: one line per drive, with its label, its state, since when, and what it waits on.

### What a person never has to know

- That a drain has a cap, or that a new drain follows an old one.
- Which machine's file holds the reservation ceiling.
- A machine's host name, to tell its drive from another's.

## Mechanism

We expect a drive to need no restart for its cap or for an empty queue, because the person's yes already names a time bound and the drain's state already carries everything the next drain begins from. Shown wrong if a drive with time left ends for either reason, or if a drive is found running past the end time its start showed.

We expect several drives of one person to stop waiting on each other for no reason, because each drain now holds one lane in work and takes the next as soon as that lane's pull request is queued, and the person's total is one number read from the repository. Shown wrong if two machines of one person are seen to apply different ceilings to the same repository, if a drain opens a second lane while its first is still being implemented under the default, or if adding a third drive adds no landings while the person stands below the ceiling and the merge queue has room.

We expect a person to answer "are the drains working?" from one machine, because each drive already knows its state as one value (itd-2611141610197753) and the status branch already carries fixed-form notices to every checkout. Shown wrong if people still log in to a machine to learn whether its drive runs, or if a published line is ever found to carry a host name, a path or an account name.

## Scope Conditions

- A repository with a remote named origin whose default branch only reviewed changes reach. The ceiling and the in-work limit are read from that branch; in a repository where anyone can push to it, anyone can set them. <!-- cond: cond-2611251810506368 -->
- One person is told as reservations already tell them: the account's holder id, the ids the person's home declares as theirs (`~/.abcd.noindex/holder-ids`), and their login on a hold made before the id. A second account or machine whose id is not declared is another holder to the ceiling and to the view, and the view says so. <!-- cond: cond-2611251810500921 -->
- Published lines are readable by everyone who can read the repository, as reservations are. The person accepted this on 2026-11-25. <!-- cond: cond-2611251810500312 -->
- The bound is a time, and optionally a count of lanes. It is not a limit on what the run spends; the per-drain limit and the subscription bound a day. <!-- cond: cond-2611251810507099 -->
- One drive works one lane at a time, whatever the in-work figure (decision 11). A figure above one for the in-work limit is honoured by a drain a host session steps; more than one lane at once inside one drive is iss-2610230724404433 and is not built here. <!-- cond: cond-2611251810501706 -->
- A reader on a release earlier than this one shows each published state line as an entry it cannot read, for a day, and never as text. Every machine of one person is expected to run this release or later before drives publish. <!-- cond: cond-2611251810505900 -->
- The state line, its `stalled` rule and the carry-on from one command are those of itd-2611141610197753, which this builds on. Nothing here is evidence for a drive across a reboot until a test or a live run covers it. <!-- cond: cond-2611251810509158 -->

## What's Out of Scope

- Starting a drive again without a person after a reboot, a logout or a crash. Ruled out on 2026-11-25 (decision 6); the draft itd-2611041656024409 takes the same answer.
- A daily number of lanes. Not asked for (decision 1).
- More than one lane at a time inside one drive (iss-2610230724404433).
- A stuck hand-back that is the machine's fault barring only that machine. It needs the count of hand-backs in a row that itd-2611141610197753's ninth criterion states, so it is captured as iss-2611251810162964 and is not a criterion here (decision 12).
- The drive's start fetching before it refuses, and a fast-forward at the start. The start fetches before it refuses (iss-2611251741261124, resolved); the fast-forward is itd-2611141610201875.
- A reservation whose landing has stopped moving, back-off traces, and clock skew between holders: iss-2611201429249531, iss-2611011010338815, iss-2610241424428420, iss-2610142200372105.
- Subscription windows, profiles and model fallback: itd-2611040344170206 and itd-2611150636483483.
- One machine starting or stopping a drive on another.
- A new command. The view is part of `abcd status` and of the board.

## Acceptance Criteria

- **Given** a repository whose `.abcd/config.json` on origin's default branch sets `reservation.max_live`, **when** any machine or account of one person adds a reservation or a drain looks for its next lane, **then** that number is the person's ceiling on every one of them, ten where the repository sets none, and it counts every live reservation that is theirs, the lanes whose pull requests are queued included.
- **Given** a machine whose own `reservation.max_live` is lower than the repository's, **when** that machine's sessions reserve, **then** the lower number applies on that machine; **given** it is higher, **then** it is ignored and named, and the repository's applies.
- **Given** a `.abcd/config.json` changed in a working tree, a lane's worktree or any branch that is not origin's default branch, **when** the ceiling or a drain's in-work limit is read, **then** the change has no effect.
- **Given** a drain with one lane being implemented and the default in-work limit of one, **when** the drain moves, **then** it reserves nothing and opens no lane; **when** that lane's pull request is queued, **then** the lane keeps its reservation, leaves the count, and the drain reserves the next issue.
- **Given** `drain.in_work_max` set in the repository's configuration, or `--in-work <n>` given when a drain starts, **when** the drain begins, **then** it takes that number, the flag over the repository over the default of one, keeps it until the drain ends, and names where it came from; a value outside 1 to 10 is refused naming its origin.
- **Given** a person with reservations from several machines, **when** they run `abcd peers`, or a drive waits at either limit, **then** both counts are shown: the lanes this checkout's drain has in work against its limit, and the live reservations the person holds against the repository's ceiling.
- **Given** a drive whose drain reaches its cap or is otherwise finished before the drive's end time, **when** the drain ends, **then** the drive begins a new drain itself, says so in one line naming how many drains it has begun, and asks nothing.
- **Given** a drive started with `--max <n>`, **when** it has opened n lanes across all the drains it began, **then** it ends and says so; **given** no `--max`, **then** no count of lanes ends it.
- **Given** a drive with time left, **when** nothing is eligible, held or not, **then** it does not end: it waits, looks at origin again every fifteen minutes, takes the next issue that becomes eligible, and its state line reads `waiting` with the time of the next look.
- **Given** a drive started with no `--until`, **when** it starts, **then** its end time is seven days from the start, or the number of days the repository's `drive.until_days` on origin's default branch names, from 1 to 31, and the start shows the end time before the question; **given** `--until`, **then** that time is the end.
- **Given** a running drive, **when** a repository file, a machine file or a new release changes, or the drive re-executes itself for a release, **then** its end time and its lane cap are the ones its start showed: nothing but a start the person types sets either.
- **Given** a drive that meets a rate limit, an outage or an unreadable origin, **when** its end time has not come, **then** it waits and does not end, and ends only at the end time, at its lane cap or on a stop.
- **Given** a drain state left by an earlier start, **when** a drive starts, **then** the start says how old the state is and what the drive carries over from it.
- **Given** any drive, **when** it starts, **then** its full plan on `?`, which the drive also writes to its log, says that a reboot, a logout or a crash ends it and that nothing starts it again, and gives the one command that carries on, which names the same end time, and after the yes the drive prints that command once; **given** the drive was ended that way, **when** the person types that command, **then** the drive carries on from the drain's state and ends at the time first shown.
- **Given** a running drive in a repository with origin, **when** its state word or what it waits on changes, and at each renewal of what it holds, **then** it publishes its state line to the account's status branch on origin under the label the person chose for that machine, and at its end it publishes `stopped`.
- **Given** any published state line, **when** it is read as pushed, **then** it holds only the label, a state word, a reason from a fixed list, a record id, times, counts, how far the checkout is behind where a fast-forward was declined, the holder id and working-copy key every reservation already carries, and abcd's version, and never the machine's host name, a path, an account name or any sentence the drive composed; the commit's author and committer are those a reservation's commit has and hold none of the three either.
- **Given** a label, **when** the person sets it, **then** it is 1 to 24 lower-case letters, digits and hyphens, and one equal to the machine's host name or the account's name, in any case, is refused; **given** none is set, **then** the drive publishes under a label drawn from the working copy's random key and says how to set one.
- **Given** drives on two machines and a second account of one person, **when** that person runs `abcd status` on any of them, **then** a block of their own shows one line per drive (label, state, since when, on what, when it ends, and, where a fast-forward of its checkout was declined, how many commits it is behind and why in one fixed word), the reservations they hold with both counts, and their open pull requests on the repository by number and state; the board shows a one-line summary of the same; and `--json` carries the block.
- **Given** a published line whose drive has not been heard from past the time it said it would next write, **when** the view shows it, **then** it reads `stopped`, not heard from since that time, and never `working`.
- **Given** a drive or a reservation on the status branches or the reservations that carries a holder id the reader's home does not declare, **when** the view is shown, **then** it is not counted as the reader's, and where any such drive is present the view says how an id of the person's own is declared.

## Open Questions

All resolved by the product thinker in the planning interview of 2026-11-25, or settled by the technical facilitator as a consequence of those answers (decisions 1 to 13).

## Decisions

Recorded 2026-11-25. Decisions 1 to 6 are the product thinker's, given in a planning interview that day; each is restated here and none is quoted. Decisions 7 to 13 were settled by the technical facilitator the same day without a question, as consequences of those answers; each names what would reverse it where it is a default.

1. **A drive ends at its end time, at an optional lane cap, or on a stop, and at nothing else** (product thinker, 2026-11-25).
   - Question: what stops a drive that no longer stops at its drain's lane cap?
   - Answer: the end time, `--until`. It defaults to seven days from the start, and a repository may set another default. A person may also give `--max`, a cap on the lanes the drive opens in all. The drive begins the next drain itself when one ends and waits when the ledger is empty.
   - Not taken: a daily number of lanes, which the planning note recommended. The per-drain limit of decision 2 and the subscription bound a day.
   - What it costs: every drive now has an end time, so the endings a drive without one had (eight hours on any one wait, on an outage or on a limit) no longer end a drive; it waits until its end time. A drive on a forgotten machine holds reservations and may spend for up to seven days.
   - Reversed if: drives are seen to spend through a day in a way the person did not expect and a time bound did not prevent.
2. **Each drain holds a set number of lanes in work, one by default** (product thinker, 2026-11-25).
   - Question: is the limit on live reservations one per person, or one per account and machine?
   - Answer: neither as asked. The limit that gates a drain's next lane is the drain's own: a number set per repository, one by default, with an override for one run. A drain reserves an issue, fixes it, and reserves the next; more at once means more drains, one per terminal, machine or account.
   - Settled by the technical facilitator from the person's description of that cycle: a lane is in work while it is being implemented, reviewed and prepared for landing. Once its pull request is queued it keeps its reservation, so no other drain takes the issue, and leaves the count. A lane that carries several issues (itd-2611201450052326) counts once.
   - What it costs: one drive does no more at once than it does today. The gain is that the gate is the drain's, so drives stop waiting on each other at a total.
3. **One person's ceiling is kept, and the repository sets it** (product thinker, 2026-11-25, refining decision 2).
   - Answer: a ceiling per person stays, ten by default. It counts every live reservation of the person across all machines and accounts, the lanes whose pull requests are queued included, because a queued lane's issue is still closed to every other drain. The ceiling is no longer each machine's `reservation.max_live`: the repository's configuration sets it, so every machine of one person applies the same number. A machine's own setting may lower it on that machine and never raise it.
   - What it replaces: until now the ceiling was the machine's alone, on the ground that a merged commit must not raise what every contributor may hold (the resolution of iss-2610170717515674; the comment in `internal/core/claim/maxlive.go`; the grounds of dcn-2611222342270127, which name it among the person's own settings). This decision is the product thinker's ruling and replaces that rule for the ceiling. It refines dcn-2611222342270127: that record's own ruling, on `drain.bundle_max`, stands, and the one clause of its grounds that names `reservation.max_live` no longer holds.
   - How the cost is bounded: the number is read from origin's default branch and never from a working tree, it is held to 1 to 100 as before, and a machine may still lower it.
   - This is the ruling iss-2610231127555243 waited on, recorded with decision 2 as dcn-2611251747513582.
4. **Each drive publishes its state line, and one view shows everything that is one person's** (product thinker, 2026-11-25).
   - Question: is each drive's one-line state published where every machine can read it?
   - Answer: yes, on every change, to the account's status branch on origin, under a label the person chooses for the machine. `abcd status` and the board gain a view of everything that is the person's: each drive with its state, their reservations and their pull requests.
   - Accepted cost, put to the person: anyone who can read the repository sees those lines, as with reservations today.
5. **A published line never carries a host name, a path or an account name** (product thinker, 2026-11-25, as part of decision 4).
   - What may be published about a machine: the chosen label, a state word, a reason from a fixed list, a record id, times and counts. The line also carries what every reservation already publishes and what names no machine: the opaque holder id, the working copy's random key and abcd's version.
6. **Nothing starts a drive again by itself** (product thinker, 2026-11-25).
   - Question: may abcd set a drive to start again after a reboot or a crash?
   - Answer: no. The drive says at its start that a reboot, a logout or a crash ends it, and prints the one command that carries on from the drain's state. Nothing that starts itself is installed. The draft itd-2611041656024409 takes this as its answer.
   - Not taken: an opt-in the person sets up once per account, which the planning note recommended.
7. **"One person" in the ceiling and the view is who reservations already say it is** (technical facilitator, 2026-11-25; accepted as the reversible default).
   - The rulings speak of a GitHub login. Nothing abcd pushes names a login: a holder is an opaque id per account (iss-2610290809526033). So the person is the account's holder id, the ids their home declares, and their login on a hold made before the id. The view is headed with the login, read on the machine and never pushed.
   - What it costs: a second account whose id is not declared is not counted or shown as the person's. The view says so and names the file.
   - Reversed if: the product thinker wants an undeclared account under the same login found and counted, which needs the forge's record of who pushed and is not built here.
8. **The repository's numbers are read from origin's default branch, as last fetched** (technical facilitator, 2026-11-25).
   - A lane's agent can write the working tree and its own branch. It cannot write origin's default branch. So the ceiling, the in-work limit and the default end are read from the commit origin's default branch stood at when the checkout last fetched it, and from no working tree.
9. **A drive's end time and lane cap are fixed by the start the person typed** (technical facilitator, 2026-11-25).
   - The repository's default end is read once, shown before the question, and covered by the yes. A re-execution for a release carries the end time and the lanes opened so far, and reads no default again. The command a drive prints for carrying on names the same end time, so carrying on after a reboot does not move the end.
10. **The published line is its own fixed form on the status branch, and the label is the machine's own setting** (technical facilitator, 2026-11-25).
    - The line is one empty commit of fixed fields on the person's status branch, validated field by field as an announcement is, pushed on a change of state and with the hourly renewal. It is not an announcement, whose kinds and day-long life do not fit it.
    - The label is `drive.label` in the machine's `~/.abcd.noindex/config.json`, never a repository's, so no repository names a person's machines.
    - What it costs: a reader on an earlier release shows each such commit as unreadable for a day.
11. **A drive works one lane at a time, whatever the in-work figure** (technical facilitator, 2026-11-25).
    - A drive runs its lanes one after another in one process, so a figure above one changes nothing inside a drive. A drain a host session steps honours it.
    - Reversed if: iss-2610230724404433 (several lanes at once inside one drive) is taken up, which is a build of its own.
12. **The hand-back that is a machine's fault is an issue of its own** (technical facilitator, 2026-11-25).
    - It needs the count of hand-backs in a row that itd-2611141610197753's ninth criterion states, so it is captured as iss-2611251810162964 and is no criterion of this intent.
13. **A declined fast-forward of a drive's checkout is carried in the view, as a count and one fixed word** (technical facilitator, 2026-11-25).
    - The ruling on itd-2611141610201875 puts a declined fast-forward on one event line, in `abcd implement status` and in this view, and keeps it off the state line. The published line carries how many commits the checkout is behind and one word from a fixed list for what declined it, and never the path or the sentence.

## Typed Links

- builds_on: itd-2611141610197753 (the stored state line, its `stalled` rule, and the carry-on from one command). The published line is that value, reduced to fixed fields.
- refines: itd-2610222215485265 (`abcd drain --drive` from a plain terminal): its endings at the cap, at an empty queue and after eight hours of waiting.
- refines: itd-2610161016394967 (several accounts share one queue): the gate on a drain's next lane, and where one person's ceiling is read.
- refines: itd-2610161016401115 (drains run unattended and visibly, where `abcd peers` shows another account's lanes): the two counts, and the view of one's own drives in `abcd status`.
- refines: adr-2610222222326104 (abcd drives agents with no host session): one yes covers many drains, bounded by an end time. adr-2611251746361324, filed with this intent, states it and carries decision 8's rule that the repository's numbers are read from origin's default branch.
- governed by: adr-2611251746369732 (a machine is published on the shared remote only as a chosen label, a state word, times and counts), which decisions 5, 10 and 13 apply.
- refines: dcn-2611222342270127 (the drain's bundle cap is set by the person's machine). Its ruling on `drain.bundle_max` stands. The clause of its grounds that names `reservation.max_live` among the person's own settings is replaced by decision 3, the product thinker's ruling of 2026-11-25: the repository sets the ceiling, and a machine may lower it and never raise it. The same rule in the resolution of iss-2610170717515674 and in the comment of `internal/core/claim/maxlive.go` is replaced with it.
- resolves: iss-2610231127555243 (the ceiling's unit), by decisions 2 and 3; iss-2611050840353717 (nothing restarts a long run), by decision 6, its opt-in ruled out.
- refines: itd-2611041656024409 (a run finishes after it dies, from the one command that resumes it), which takes decision 6. This intent's fourteenth criterion is that draft's first two, its spec's step 3 builds them and resolves iss-2611050840353717, and the draft keeps the two criteria that remain.
- context: itd-2611062149308723 (a drive takes each release itself), whose re-execution decision 9 binds; itd-2611141610201875 (the drive keeps its pull requests and checkout moving), whose declined fast-forward decision 13 carries in the view.
- evidence: `.abcd/development/research/notes/drive-operations-2026-11.md`, events 1, 2, 5 and 10; `.abcd/development/research/notes/autonomous-run-2026-11-14.md`, event 11; the drain state of 7 to 24 November (189 of 200 lanes).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: a person is expected to leave drives on several machines working one ledger for a week with no restart and to answer for all of them from one machine, because the yes already names a time bound, each drain can gate itself on one lane in work, and each drive already knows its state as one value; shown wrong if a drive with time left ends at a cap or an empty queue, if a drive runs past the end its start showed, if two machines apply different ceilings to one repository, if people still visit a machine to learn whether its drive runs, or if a published line carries a host name, a path or an account name
