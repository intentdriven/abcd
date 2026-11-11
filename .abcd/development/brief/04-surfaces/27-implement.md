# `/abcd:implement` — Share a Run Between Sessions

`/abcd:implement` is the run machinery an autonomous run calls. It holds a run's
shared state in the machine-scoped store, records the sessions that join it,
makes a record the unit of exclusion between them with a claim, keeps a second
session inside its bounds, and derives the comparison of the three ways of
dividing work from the run log (itd-2609221656373558, spc-2609221657588816).

It is also the family the implement loop is driven through (itd-2609201916151817,
decision 8): `build` is what a person types, and the loop's status, step,
receipt, widen and record are sub-verbs of this verb, which a driving session calls. The loop's
own state lives in the checkout's local tier, not in the shared run state below;
[`34-build.md`](34-build.md) is its chapter. The pacing intent
(itd-2609201925079472) reads the loop's window clock.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`). The existence
> fact is verified against the committed command-tree snapshot in both
> directions. The bucket cell is checked for membership of the closed adr-40
> vocabulary only._

| Verb | Bucket | Status |
|---|---|---|
| `join` | — | shipped |
| `leave` | — | shipped |
| `mode` | — | shipped |
| `claim` | — | shipped |
| `release` | — | shipped |
| `check` | gate | shipped |
| `log` | — | shipped |
| `report` | — | shipped |
| `load` | — | shipped |
| `status` | — | shipped |
| `step` | — | shipped |
| `receipt` | — | shipped |
| `widen` | — | shipped |
| `record` | — | shipped |
| `resume` | — | shipped |
| `stop` | — | shipped |
| `outage` | — | shipped |
| `outage record` | — | shipped |
| `outage probe` | — | shipped |
| `outage ack` | — | shipped |
| `outage clear` | — | shipped |

## Where the run lives

```text
~/.abcd.noindex/runs/<root-sha>/<YYYY-MM-DD>.jsonl    the run log, one event per line, per UTC day
~/.abcd.noindex/runs/<root-sha>/claims/<record>.json  one claim per record
~/.abcd.noindex/runs/<root-sha>/sessions/<id>.json    one record per joined session
~/.abcd.noindex/runs/<root-sha>/outage.json           the run's one outage, while it is open
~/.abcd.noindex/runs/<root-sha>/.lock                 the advisory lock every mutation takes
```

The key is the full root-commit SHA, the key the transcript, history and voyage
stores use, resolved from the checkout the caller stands in. Two sessions in two
worktrees of one repository share one directory and write no repository file.
Every level is created one directory at a time and proved real, never through a
symlink; outside a checkout, or in a repository with no commit, every sub-verb
refuses and nothing is created, with two exceptions that need no run: the load
check, its wait included, is a machine check with no checkout precondition and
runs anywhere, and the report given a log file to read reads that file and
nothing else. Bare `abcd implement` and the report create
nothing at all. Only joining brings a run into existence: every other writer acts
for a session that has joined, and one invoked for a session no run holds is
refused before anything is created — no directory, no lock, no log line.

## Joining

A session joins under its id, in the role `first` or `second`, optionally stating
an agent ceiling. Joining writes the session's record by an exclusive create and
logs `session_open` with the role. Nothing signals any other session. A session
that joins again with the role it holds is a resume, logged with `rejoin`; asking
for the other role is refused. Leaving releases every claim the session holds,
logs `session_close`, and removes the record.

A first session's join is the one act by which a session says it coordinates a
run, so a first session whose setting reads as off a profile is warned after each
join, in any mode, on the terms [`26-mode.md`](26-mode.md#a-session-off-a-profile) sets out
(dcn-2611041550129278).

The role lives in that record and nowhere else: the environment is not a trust
input, because a variable a shell or a repository's configuration can set is not
a statement the session made. The record gives consistency, not authentication —
two sessions of one account can each write anything under that account's home —
so the bounds are a discipline two cooperating sessions keep, checked at every
sub-verb.

## The window's mode

Setting the window's mode — single, claim, batch or split-roles — logs `window_mode`; only the
first session sets it. The mode in force is the log's last `window_mode` line,
whoever wrote it, so a line the run writes by hand counts the same as one the
verb wrote.

## The claim

Claiming a record for a named lane, with an optional lease, creates
`claims/<record>.json` exclusively through `fsutil.CreateExclusiveIn`; the create
is the exclusion, so of two sessions reaching for one record exactly one holds
it. A run-wide advisory lock orders the read-decide-write sequences around it (a
lapse and a re-claim, a cap count and a claim). The claim carries the session,
the lane, the time and the lease (default two hours, one minute to a day). The
holder claiming again renews its lease. A lease that has passed is claimable:
the lapse is logged as `claim_lapsed`, naming the previous holder, and the claim
is taken. A live claim held by another session is refused at exit 3 and logged
as `claim_denied` naming the holder. A granted claim is logged under the event
name claim; if that line cannot be written the claim is removed again, so the
directory never holds a claim the log does not. Releasing removes the holder's own claim and
logs `claim_released`; only the holder releases. A claim file nobody can parse
(a session killed between the exclusive create and the write leaves an empty
one) blocks nothing but its own record. The build's peers check counts it as a holder until a claim replaces the file; the claim verb
honours it only for a one-minute grace counted from the file's modification time: the status lists it as unreadable, leaving
and every other claim read past it, a claim on its record within the grace is
contention naming the file's full path, and after the grace the claim logs
`claim_lapsed` with reason `unparseable` and takes the record.

## The bounds

A session joined as `second` is refused at exit 2, with a `refusal` line naming
the condition, when it claims while holding another live claim
(`second_session_lane_cap`), claims or checks a lane in a `split-roles` window
(`split_roles_second_builds_nothing`), declares a path in the reading corpus
(`reading_corpus_lane`), or reaches the release stage (`second_session_release`).
Checking a stage that is not a claim — a lane, the release, a review, an audit
or a landing — asks before it; an allowed stage writes nothing. What the check
is asked about is a stage (ruling CM1, beside BU1's lane stages), in its verdict
and in its refusal line alike.

A session's own agent ceiling (criterion 5; for the second session, on top of
the first's) is held against the agents the session declares. The session
states it when it joins (1 to 64); the record and the `session_open` line carry
it, and a resume cannot restate it. abcd does not run a session's agents, so the count it holds
the ceiling against is the session's own lines: the agents its `agent_start`
lines since it joined name, less those an `agent_end` of the same `agent` has
ended. An `agent_start` that would take the count past the ceiling is refused at
exit 2 with a `refusal` line (`agent_ceiling`, naming the agent, the agents
alive and the ceiling); restating an agent already alive is not a new one. Every
check verdict reports the count (`agents_alive`) beside the ceiling
(`ceiling`). An agent the session never logs — a fork, one the host started
outside the log — is invisible to the count, which is why the run's no-fork
rule stays the discipline for that half (iss-2609240646542516); a run that went
over anyway says so with a `ceiling_overrun` line. On a refused claim the
second session also logs a `backoff` with its reason and minutes, the minutes
being what the attempt spent, measured from its start; a run state locked past
the lock's timeout by another session's change is contention too, and the second
session's `backoff` from it carries `on: run_state` and the minutes it waited. A
second session whose join meets the lock has no record yet, so the role it is
joining with places the line; a session that never joined has no role to place
it by, and the refusal says the backoff went unlogged. The append takes no lock,
so that line reaches the log while the lock is held.

Every bound keys on the role in the session's record, which is the session's
own statement: the release refusal, like the others, rests on a cooperative,
unauthenticated role, a discipline between cooperating sessions rather than a
wall against one that lies about its role.

The reading corpus is derived, never restated: the union of every position's
`object.paths` in the checkout's committed `.abcd/config/reading-presets.json`,
plus that file itself, read through the loader the `reading` verb uses — so the
two cannot disagree about what the corpus is. An entry names a file or a
directory, and a directory covers everything beneath it. A lane that edits one
of those paths moves what a cold reading is handed, and so the windows the first
session recalibrates (iss-2609211105023379). When the corpus cannot be derived —
no preset file, one that is untracked, symlinked or does not parse, or no
checkout to read it from — a second session's lane that declares paths is
refused (`reading_corpus_unknown`): the bound fails closed. A lane that declares
no paths asks no corpus question. The release stage is refused at
this verb, not inside the launch cut: the cut knows no session, and a gate
keyed on a flag the second session could omit would guard nothing.

## The log and the comparison

Logging appends one of the run's own events, with its key-value fields
(`backoff`, `lane_open`, `lane_close`, `agent_start`, `agent_end`,
`ceiling_wait`, `gate_run`, `review`, `fallback`, the stop event, `refusal`, `pr`,
`capture`, `context`, `ceiling_overrun`, `intervention`, `decision`). Every line carries `ts` (RFC 3339, UTC), `session` and `event`, then
the fields; it reaches the file in one `O_APPEND` write through
`fsutil.AppendLineIn`, which refuses a symlinked or non-regular leaf as its read
twin does, so two writers each land whole lines and a log leaf planted as a link
onto a claim file appends nothing. The session, window,
claim, load and outage events are refused here: they are written by their own sub-verbs,
so the log cannot record a claim the run state does not hold, or a load warning
the check did not give. A hand-logged `backoff` names its `reason` and the
`minutes` it spent (a number no smaller than zero), or it is refused with nothing
written: contention the verb cannot see, such as the merge queue, reaches the
comparison only this way, and a backoff with neither would count as one that
cost nothing for no reason.

An event missing a field the report reads is refused when it is written, naming
the field, rather than found missing afterwards (iss-2609240646555891): a
`lane_close` needs `lane` and `outcome`; an `agent_start` its `agent`; an
`agent_end` its `agent`, `role`, `model` and a number under `minutes`,
`wall_minutes` or `wall_min`; a `ceiling_overrun` (iss-2609240646549900) the
agents `alive`, the `ceiling`, the `lane` and the `minutes` over. The evidence
events an autonomous run keeps so a later run can need no person carry theirs:
an `intervention` its `kind` (`session_open`, `account`, `ruling`, `restart`,
`close_session`, `file_restore`, `permission` or `other`), `by`, `what`, `why`
and `autonomy_gap`; a stop event its `cause`; a `decision` its `what`, the
`alternative` not taken and `why`. An `at` or `last_productive` given is an
RFC 3339 time, and a `detected_after_min` or `noticed_after_min` a number.

The report derives, per mode, the windows, wall clock, lanes opened and
landed (a `lane_close` whose outcome is `merged` or `landed`), the second
session's lanes landed, collisions (`claim_denied`), lapsed claims, backoffs and
their minutes, agent minutes (`agent_end`'s `minutes`, `wall_minutes` or
`wall_min`, the key the run's hand-kept lines carry), ceiling wait, ceiling
overruns and refusals, with each session's share. An event belongs to the window
open when it happened, save a `session_open` logged at most a minute before the
next `window_mode`, which belongs to that window: a session joining a second
before the first sets the mode is joining that window (iss-2609240646544930).
A `context` line is an orchestrator's context measurement (`used_pct`, `role`,
`note`); the report totals them per session across the whole run, not per mode,
with the last `used_pct` seen, because a session's context is carried across
windows. `leader`
is the mode with the most lanes landed per wall-clock hour — a figure the run's
report cites when it names the mode it would keep, not a verdict of the verb's.
Over the whole run the report counts the evidence (interventions by kind with
the minutes they went undetected, stops with the minutes before each was
noticed, decisions), names per event the lines lacking a field logging requires
(`missing_fields`: lines written by hand or before the requirement, which every
figure above reads as absent), and names each of `lane_open`, `lane_close`,
`agent_start`, `agent_end` and `gate_run` whose last line falls more than six
hours before the run's last line, load samples aside (`coverage`). Every line
still parses in those cases, so without the two lists nothing would say that a
figure is short.
Lines that are not a JSON object with `ts`, `session` and `event` are listed as
`unparsed`, never dropped silently. The report can read one day, or one log file
named directly.

## The load check

The load check is what abcd's own test lanes run before they start
(itd-2609231434459890, spc-2609231542463113): once as the first stage
of `make preflight`, and once in the eval harness's
`TestMain`, never once per test package. It reads the machine through
`internal/core/machineload`, a standard-library-only leaf (macOS: the
`vm.loadavg` and `hw.activecpu` sysctls and one `/bin/ps` run; Linux: `/proc`
and `/sys/devices/system/cpu/online`), warns of low disk and memory pressure beside its status, and sets that status to a warning on two triggers: a process
outside the check's own parent chain older than the stray limit that uses
nearly all the CPU it could get, or a one-minute load average strictly above
the extreme limit. What a process could get is its fair share of the machine as
the one snapshot finds it loaded, the online cores divided by the one-minute
load and never more than one core (`machineload.FairShare`), and a lifetime CPU
share of at least 0.9 of it makes a stray: forty busy loops on 16 cores, each
at 0.4 of a core, are all strays, as one loop at a full core of an idle machine
is (the product thinker's ruling of 2026-09-25 on iss-2609231947544298). The
share test applies to the caller's own processes and to other accounts' alike.
Nothing is exempt by name: abcd's own lanes are exempt by time,
because everything they start lives for minutes. The caller's own strays are
named (name, pid, process group, age, share), masked through the private
banned-names layer's own engine, with commands that re-check each target before
a kill and never match by pattern; the group form is offered only for a group
whose every live member is named and which is neither the check's own group nor
an ancestor's. Other accounts' strays are a count and a CPU total, by type.

The limits live in the caller's machine tier, `~/.abcd.noindex/load-limits`
(`stray-minutes`, default 30; `extreme-load`, default four times the online core
count; `wait-load`, default twice it), read through the guarded declaration read
and never created; an unusable file is reported loudly and every default is used. On a CI runner
(`GITHUB_ACTIONS=true`, or a `CI` other than empty, `false` or `0`) nothing is
read and the line says why; each CI job that starts the harness runs the check
as a step first, so the reason reaches the job log. On another platform, or when
a read fails, the check says it could not check. Inside a live run (a run state
with a joined session) a warning is also written as a load event, attributed
to the first-role session, carrying the verdict's facts so it renders exactly
as it was printed; the run's hand-written load samples share the name and carry
no `triggers`. Without its wait flag the check exits 0 on every status: it never
refuses, waits or signals anything. With its wait flag it is the one waiter
concurrent sessions gate a preflight or a race run with, and `make preflight`
waits at it once, as one whole run (a records-only change takes a shorter
rule that waits at no gate): it blocks until the one-minute load is
below the wait limit, no other `go test -race` run is live, no other
`make preflight` is running and no other waiter is ahead of it in the queue (one
that started earlier, whatever its pid, or within two seconds of it with the
lower pid; iss-2610222125175640) or running its command, under any account; a waiter beneath a waiter
running its command takes that ancestor's place, ahead of every queued waiter. All of that is read from
the process table, which shows every account's processes, judged from each
candidate's arguments with the waiter's own ancestry excluded
(iss-2610140638074587): no file another account can write is read; the gate
reads only the process table. A waiter queued ahead still delays the
ones behind it, under any account, until it opens or reaches its own timeout,
even when only its own account's stricter wait limit keeps it shut; there is no
channel by which it could tell them so without a shared writable file. The one
exception is a crossed waiter, ahead by start with a higher pid, as after the
process ids wrap: an abcd built before the start order (v0.15.0 and earlier)
waits on every lower pid, so it may be waiting on this one while this one
waits on it, and nothing in the table tells it apart. Every cycle of waits
passes through a crossed wait, since every other wait from any abcd points at a
lower pid, so a wait stops counting crossed waiters once they alone have kept
it shut for five minutes on end (`WaitCrossedCap`), longer than a waiter that
orders by start shows queued while the machine is free to another account,
plus a step for each crossed waiter ahead beyond the first, so the earlier of
two waiters behind the same one opens before the later stops counting it; no
back-off breaks that cycle, since the back-off cap bounds only the last look,
and a stand-back starts the count again.
A waiter running its command holds every later first gate for the command's length at one preflight at a time, and a slot of it above one, by design. A look that finds the machine clear does not open the
gate: the waiter waits a random 15 to 60 seconds and checks everything again,
and opens only if that re-check is clear too (iss-2610102020442624). Within one
account it also takes a hold under the abcd home, which a waiter beneath the
holder ignores. Across accounts, which share no hold, two waiters can still open
in one moment, so the command form shows itself as running its command from the
moment it opens and looks once more two seconds later; it stands back from the
other unless it started earlier and has the lower pid too, gives up its hold,
re-draws its jitter and waits again, at most three times, while the other runs.
A command hold lasts until its command exits, starts its own race run or has run 60 minutes: on 2026-10-24 a long, slow
`go test` of the loop suite held the first gate for 45 minutes while nine
preflights queued, and they could not tell whether it was working
(iss-2610241419324875). So each waiter behind a command hold, and the holder
itself while its command runs, follows the CPU of the holder's whole process
tree between its looks, from the cumulative CPU times the process table
carries; a child seen at one look and gone by the next counts as work, except a
waiter's own errands in the tree (its reads of the table and the git and gh of
its reservation renewal), which are not the run's work. When the tree has used no
more than a tenth of a core for ten minutes (`QuietShare`, `QuietSpan`), the
waiters' `waiting for capacity:` line says so as a fact, with the holder's pid,
the checkout its hold record names and how long it has held the slot, and the
holder says it on its own standard error while others queue. That notice
speaks of a command hold, which a command that is not a race run takes only at
one preflight at a time; above one such a run holds its slot under the settle
hold, so no notice names it, and a stalled one is shown in the `holding:` part
below with its CPU and, once it has used no more than a tenth of a core for ten
minutes, marked quiet. It is not a
diagnosis: a slow run waiting on its subprocesses uses little CPU too. A tree
with a waiter of its own queued at a gate is not counted quiet. The gate only
says it: the hold stays in force and nothing is signalled, since a process is
stopped only by its owner through its own handle.
Every waiting line also names each holder ahead, the hold record's and the
work the process table shows in a slot of any account, with its age and its
tree's CPU over the last ten minutes of this waiter's looks, or `not yet
measured` before then, and a holder whose tree has used no more than a tenth
of a core over that span, with no waiter of its own queued at a gate, is marked
`quiet:` in place of its use, whatever hold it is under (`GateHolder.Quiet`,
`"quiet": true` in the JSON result); a preflight is named once whichever way its make
and the holding waiter nest, and a unit taking no slot (above one, a
preflight at its race gate) is not named. The line is said again each span
while the gate stays shut (iss-2610241457170255). A child that starts and exits between two
looks is never seen, so a tree of short git children reads low, and counts as
working for the quiet notice and the quiet mark alike; only the hold record's holder, of the waiter's own
account, is said quiet in the notice with its checkout.
The later start stands back whatever the pids; the earlier one stands back from
a lower pid as well, which an older abcd (v0.15.1 and earlier) runs past.
It runs a command given after `--` and exits with its code,
forwarding an interrupt to it, and gives up after its timeout at exit 1, naming
what it was still waiting on. Inside `make preflight` each test stage then
takes a `go test -p` derived from the online cores and the load as the stage
starts (`machineload.GoTestParallelism`: twice the cores less the load, at most
the cores, at least a quarter of them), and on a CI runner the gate opens at
once and go's own default stands.

Before its race stage `make preflight` waits again, at the race gate, which
has its own flag and never carries the first gate's wait flag (the two are
refused together): every abcd released before the race gate reads any load
check carrying the wait flag as a first gate's waiter, so it would read a
preflight at a race gate spelt with it as queued, not in its test stages, and
open beside it. At one preflight at a
time the race gate opens at once once the disk check passes, without reading
the load or the process table (read, inside a memory episode, only to ask whether the waiter admitted last is its own ancestor), taking a hold, or waiting a jitter or a settle,
so the race stage follows the tests as it did before the gate existed, unless
memory holds it (below), when it says it opened after its wait. Both
gates refuse, exiting 1 without waiting, while the volume holding the Go build
cache or the worktree store has less free space than the disk floor: 4 GiB for
each preflight the first gate admits at once, and 8 GiB at least; a disk that
falls below the floor while a gate waits is refused before the gate
opens. Both gates also wait, never refuse, while the system reports memory
pressure at or above the person's floor (itd-2611012048006057): macOS's own
pressure level against `load.memory_floor_macos` (`warn` by default, or
`critical`), or on Linux the share of the last ten seconds some task stalled
waiting for memory (`/proc/pressure/memory`, from 4.20) against
`load.memory_floor_linux` (10% by default), both read only from the machine
configuration, a repository's key named as ignored. Memory is read at every
look, at the re-check before a gate opens and before the one-slot race gate
opens; a wait that times out on it names `memory`. A reading that
cannot be taken shuts nothing and is said once: on stderr at a gate, and on
stdout as the load check's `memory check` line. For ten minutes after any
waiter of the account last saw the pressure at the floor, the gates admit one
run per two-minute settle through a record under `~/.abcd.noindex/`, reading
memory again before each, so a stage's memory use shows before the next
starts; the load check reports a reading at the floor as a warning and never
fails. Above one the race gate is
shut only by another live `go test -race` run, a race waiter queued ahead of
it by the same rule (started earlier, or within two seconds of it with the
lower pid) or running its command, the load limit, memory pressure at or
above the floor or its settle after the pressure falls, and the race gate's own hold, so race stages are kept apart among the preflights
whose checkouts have the race gate, one at a time within an account and across
accounts all but a narrow window: a race waiter exits as it opens, before its
preflight's race run shows in the table, and after a pid wrap two race waiters
whose start and pid disagree can both open: one of v0.15.1 and an earlier one
of the start order with a higher pid wait on neither, and two of the start
order disagree for a look where whole-second ages put their starts at the tie
tolerance's edge, or once the later has stopped counting the earlier past its
cap. A preflight run from a checkout
older than the gate runs its race stage ungated, so more than one at once is
safe only once no such checkout runs one. The setting is each account's own: an
account at one opens its race gate at once, so its race stage can start beside
the race stage of an account above one, and race stages are kept apart only
among accounts that all set more than one. And the plain form of the first gate
takes no last look, so across accounts two openers whose re-checks fall in one
moment can both open into the last free slot, one more running
than the setting. A first gate's waiter never queues
there, and a race waiter never queues at the first gate. A preflight whose
race waiter is queued runs nothing: at one it still takes its slot, and
above one it takes no slot until its race run starts (iss-2610282016250805),
since the race gate, not the slot count, keeps race runs to one at a time. Each gate keeps its
own hold record and is shut only by it: a first gate's command hold lasts
until its preflight starts its race run, so were it to shut the race gate, an
earlier preflight's race waiter would wait on it while the holder's own race
waiter queued behind that earlier one, and neither would move. That command
hold is taken at one slot; a look that reads more than one, for a waiter whose
own command is not a race run, judges it as the settle hold
(dcn-2610300618371321), so a raised count reaches a run already queued behind
it, and the running preflights are still counted against the slots. How many preflights
the first gate admits at once, each a whole run, is the person's own machine
setting (iss-2610161639025702): `load.preflights` in <!-- record-lint: forward-looking -->
`~/.abcd.noindex/config.json`, read through the layered configuration with the
machine layer alone, or `ABCD_LOAD_PREFLIGHTS` in the environment for a trial;
one by default, from 1 to 4 (dcn-2610161814279204 trialled two; the product
thinker asked on 2026-10-28 for the machine's capacity to be used, so four is
the most, with one race run at a time and the load limit as the bounds), a bad value refused naming where it was set, and no
repository file ever read for it, because a checkout must not raise the
machine's load. A machine `config.json` that cannot be read for the count
(writable by others, not valid JSON, a load that is not an object) is read
past, with one line on stderr naming the file, the cause and the fix, and one
preflight at a time applies, since the gate never read that file before the
setting existed; an unknown key in its load object is named on that line too.
Only a `load.preflights` that is there and unusable refuses the wait. At one the
first gate reads exactly as above. Above one it
admits a waiter while fewer than that many units of work are running
(a running preflight, or a waiter running its command, counted once
where one runs beneath the other), a waiter queued ahead still shuts it, a live
race run does not shut it by itself and the load limit still does, and an
opener still stands back from one that opened beside it, as above, whose mark
then counts it as a running unit; an opener stands back only from a
waiter at its own gate; a lone race run gated in the command form, its
command `go test -race`, is gated as a race run for both gates: it stays
shut while another race run is live, judges and takes the race gate's hold as
well as the first gate's, and counts as a race run to every race gate and every
other such command from the moment it opens (`machineload.RaceCommandWaiters`),
so a second one waits for the first rather than open beside it at its full
`-p`; each
stage's `go test -p` is at most the cores divided by the setting, since the
one-minute load a second stage starts under lags the first stage's demand.
Deriving the count from the machine's resources is a later step
(iss-2610161756384596). <!-- record-lint: forward-looking -->

A narrow run waits at a gate of its own (`internal/core/implement/load_narrow.go`,
`internal/core/machineload/narrow.go`), because on 2026-11-04 about thirty builders
queued one package's test each behind whole preflights, at a load of 12 on 16 cores
with 96% of memory free, until the 30-minute wait ran out and no test ran. It asks
for the gate with a mark in the waiter's argv, which is the one truth: every waiter
judges every other, and itself, by the mark and the command's words alone, and an
abcd built before the mark reads a marked waiter as a waiter like any other. The
command is go by that exact name with no shell in between, a gated `go test`, `go -C
<dir>` allowed: at most three packages named outright, judged cleaned as go cleans
them, no wildcard or bare word, and no flag beyond `-v`, `-short`, `-failfast`,
`-json`, `-trimpath`, `-run`, `-skip`, `-timeout` and `-count=1`. The reading fails closed, so its
one mistake is a slower wait: `make`, a shell, a wildcard and `-race` are whole runs,
the last so race runs stay where the rule keeping race stages apart can see them, and
so is the same command unmarked.

What the argv cannot show, the waiter proves: go's own answer for the effective
`GOFLAGS`, environment, `GOENV` file and `go env -w` alike, must be empty as the wait
starts and as it opens. Otherwise the waiter replaces itself without the mark, keeping
its pid and its start, so the argv stays the truth, and says why on stderr. The
command then runs with `GOFLAGS=-p=1` and `GOMAXPROCS=2`, or a lower one of the
person's, which the proof means overrides nothing.

A narrow run takes no whole-run slot and no place in the whole-run queue; a whole run
counts it through the load figure alone. It is shut by the load limit, the memory
floor, its account's 30-second hold (`load-wait-narrow.json`), its account's whole run
or race stage for that hold's two-minute settle, however long the hold itself lasts, and
its room. The room is the person's `load.narrow_runs` (1 to 32, read from the machine
layer alone, under the process's home as `load.preflights` is, with no environment variable
of its own; by default half the online cores divided by 2, so an account admits up to half
the machine, and each other account can add one run over that, since every account keeps a
place, two being a ceiling on what they took as measured on 16 cores, from about 0.2 of a core to
under 2). On a machine the load shows within two of the online cores, the two classes share
one budget: the room is what the whole runs in their slots leave of the cores, each at its
stage's `-p`, and one at the least. Running narrow runs and those queued ahead count against
it, and another account's never take an account's last place. Its jitter is 1 to 5
seconds, and it takes no open look: nothing waits on it, so two accounts opening together
overshoot by one for those runs and no more.

A whole run keeps its place honestly: sharing the load limit, narrow runs could take
every dip it waits for, and an older abcd counts them in its slots. So while a
whole-run waiter that started earlier has been queued five minutes or more and a whole slot is free
(`machineload.OverdueWhole`), a narrow run stands aside, spending at most five minutes
of one wait so. That gap in admissions lets the running ones finish and the load fall. The
waiter must have started earlier because an older abcd counts a narrow waiter that
started before it as ahead: standing aside for a later start would have each wait on
the other until a timeout. Every wait points at an earlier start or a command already
running, so no cycle forms across versions. An older abcd reads a narrow waiter as a
waiter like any other, the cautious direction, reads no narrow hold, and names
`load.narrow_runs` as a key nothing reads while `load.preflights` beside it counts;
this code counts its waiters in full, since none can carry the mark, so none passes its
limit. A narrow waiter queued ahead can be shut by its own account's stricter settings,
which no other account can see, so waiters ahead that alone keep a narrow run shut stop
counting after two minutes, flat whatever their number: a waiter free to go takes a free
place in under one, and each that leaves starts the count again, so a queue that is
moving never reaches it.

Both caps' spent time carries across retries of the same narrow wait in the same lane
(`internal/core/implement/load_narrow_carry.go`). A drive's harness
ends each command after about ten minutes, so a lane waits in short tries, each a new
waiter, and a try of two minutes never spends the five minutes of the stand-aside cap.
A try that gives up records what it has spent, under a digest of the gated command's
words, the lane's worktree and, on a drive's stage, the stage's write root, which names
the lane's directory under its run; the next try with the same key starts from it. The
record is the account's own `load-wait-narrow-carry.json`, owner-only and written
atomically, and no other account's waiter reads it; one that is not a regular file the
account owns, or does not parse, carries nothing and stops no wait. An entry lapses one
of its first try's timeouts after the last try gave up, a try that opens drops it, and
a whole-run waiter carries nothing, since its place is its start in the process table.

A wait gives up after six hours by default, the figure sessions came to
pass by hand once thirty minutes ended every gated test on a busy machine. It exits
1 as ever, and its last line starts `LOAD GATE TIMED OUT`, naming how long it waited
and that nothing ran, with the renewal stopped first so no message follows it, since
builders' pipes hid the exit code that day and they read the gate as still running;
under the JSON form, whose standard output stays one document, that line is the last
on standard error.

## The implement loop

Five sub-verbs drive the loop a build starts, the status, step, receipt, widen
and record verbs, each over the run's state file in the checkout's local tier ([`34-build.md`](34-build.md) states the file, the
checks and the step interface). The status render reads every run, or the one
named, and writes only the renewal of a due reservation of each run it shows,
logged in the run record as a step logs it: it names the slots in use out of the run's ceiling,
every lane alive with its stage and each agent it awaits, and each held lane with
its cause, the head judged, the landing step it stopped before and the two flags
that decide it. The step verb performs the run's next move and exits (a lane's
stages are worktree, brief, implement, validate and land; the lane as a whole
lands one of the spec's steps; the one lane of an auto-plan run takes
plan-review and plan instead, [`34-build.md`](34-build.md) says why). A run works in parallel up to its ceiling (ruling
DR6): a stage the binary owns moves on any lane first, then, while a slot is
free, the first waiting work takes it, an open lane's before a new lane's and
the lower spec step first; a lane opens for a ready spec step only when a helper
is free to take it, its worktree made just before its implementer takes the
slot, and never more step worktrees on disk than the ceiling (ruling DR6d-1); at
a stage that hands work to an agent it names the agent, the brief and the
receipt path, and a step that finds the ceiling reached hands out nothing and
names every agent out. A landing waiting on the forge's merge, or on the
preflight receipt its push needs, holds only its own lane, the latter shown as
waiting for its full check since the time the wait began (ruling DR6d-2); any
other refused stage the binary performs is the step's answer. The receipt verb looks the path up
among every outstanding await of the run, and the stage completes only when a
lane awaits that path and its verifier accepts it; a verified receipt frees its
slot. After a hand-back the siblings finish and a lane whose round passes is
held before it pushes or arms (ruling DR6c), an armed one disarmed, or, where
the forge refuses the withdrawal, the step refused naming the pull request; the
person's word on a held lane is given through the step verb, one lane per
invocation: release lands it as it is, and discard removes its worktree and
branch, then closes its pull request, and leaves its step unlanded, each refused, changing nothing, unless the lane is
held and no lane has work left. Discard also takes a lane handed back as stuck or
closed, whatever else the run is doing. A drain's issue run is then over
([`35-drain.md`](35-drain.md); spc-2610161044461448 step 5); an intent's run stays,
and a landing's hand-back says when to remove its directory. An issue run logs
each refused step on its lane, until its pull request is armed or left open, the stage, the check, the reason and a count,
for the drain to read; an intent's run refuses as before, writing nothing.
Without a named run, the step
and receipt verbs act on the one run in progress in the checkout and refuse naming
the runs when there are several. Their refusals name the stage, the reason and
the remedy, and a pause
before the run's next eligible time, a lock held by another invocation, or a
landing waiting for its pull request to merge, is contention at exit 3; a
pull request the forge reports conflicting is not such a wait but a refusal
at exit 2, since it never merges without a person, unless it is an issue lane's:
a conflict in declared generated files alone, or in no file when this checkout
merges, enters the lane's sync, an armed merge withdrawn first, and a head
already holding the default branch waits at exit 3 for the forge's stale verdict. The
record verb reads a run's record back at the end, and on a complete run captures
the run's transcripts into the history store, one capture per path; a
transcript stored without the scanner coverage the repository armed carries its
scan gap on the record, as the history verb reports it for a capture.

An issue lane widens its hold before it edits a path beyond it
(iss-2610311224444576; dcn-2610311251002713, dcn-2610311544325397). Its brief
names the widen verb, run from the run's checkout with the run, the lane and
each further path: the issue is reserved again with the footprint its
reservation holds and the paths it does not cover, judged for overlap and
against the cap like any reservation, and the run record names the paths
widened to; a path already covered adds nothing, and a run holding no
reservation has nothing to widen. Only the run's current lane widens, at its
implement or validate stage, and an intent's run, whose footprint is its
spec's, is refused. Another copy's live hold on one of the paths is
contention: nothing is pushed, the record line says the widening backed off,
and the implementer edits none of them and stops without a receipt, the lane
handed to an implementer again on a later step. As the backstop, an
implementer's receipt is refused while the lane's branch changes a path the
reservation does not cover, naming the paths and the widen verb, so a
widening forgotten or an edit made through the shell is caught before the
lane moves on. The landing's narrowing keeps a path the lane widened to and
changed, a resume at land renews that narrowing as it stands, a resume that
reserves again over this copy's live or lapsed hold keeps the paths that hold
names beside the found_at, and a lane leaving land for a fresh round reserves
its found_at again beside the paths its narrowed hold names. The run record
keeps each path widened to in a `reservation_widened_to` line of its own, so a
reservation made again names it even when the reservation branch was deleted.
The fence before a landing's push and its arming judges the lane's branch the
same way, since no receipt stands between a lane at its validators and its
landing: a path the reservation does not cover is reserved there, judged for
overlap and against the cap, and another's hold on it stops the lane at its
landing, naming that hold, with nothing pushed or armed. Only the records the
landing moves are left out of that reading, so anything else its records commit
holds is judged, and so is a reservation made by hand for a run that recorded
none. An origin it cannot reach is a wait, as at the fence's other reads.

A run that holds a reservation on the shared remote (itd-2609091034175565;
the state's `reserved_at`) holds it for the copy the run lives in. The step
verb renews it before a stage once half its lease has passed, as last fetched,
keeping its own term (at least the default lease, at most a day), and logs
`reservation_renewed` in the run record; under
half it pushes nothing, and a renewal that fails is logged and named while the
stage still runs. A renewal renews only this person's own reservation from
this copy, never reserving a released record again or taking one over. A
renewal of a lapsed one is judged for overlap like a new reservation, but,
once no spec step is pending and every lane is past all editing (done,
discarded or handed back, or at land with no sync conflict pending), against
this person's own reservation from another copy only on the files the run's
lanes changed, read as the narrowing reads them; while any lane is in any
other state, held included, or at validate even awaiting its reviewers alone
(an implementer's edits are not on its branch until its receipt, and one
verdict hands a fix implementer any file of the footprint), or a step is
pending, or when the files cannot be read, the whole footprint is judged. An
issue run's revival at land narrows to its lane's changed files as the
landing's renewal does; an intent run's keeps its whole footprint, its record
line naming the other copy's reservation it went ahead past; and another
person's reservation still refuses either on any path of the footprint it
renews. A lane that later leaves land for a fresh round (a hold is not one)
has a reservation needing no widening (an intent run's, or an issue run's
still naming its whole found_at) judged again against this person's other
copy on its whole footprint, the reservations fetched first. An overlap, a
fetch that fails for any reason but a lost network, or a view that cannot be
read after the fetch refuses the step for a person, as a refused widening
does; a lost network is waited out on the shared network probe, as a lost
fetch of the default branch is; either way the lane is left at land with
nothing merged and nothing pushed (iss-2610261837062463, dcn-2610291352512058,
dcn-2610291925508286, dcn-2610292238191496, dcn-2610300422481486,
dcn-2610300615451659). Before
the landing's merge rule decides, the land fence fetches the reservation and
arms nothing unless it is still this person's, from this copy, and live,
naming the other person, the other copy or the release, as contention; when it
holds it is renewed for a day, with a ruleset mirror at the lane's base or
without one, since a landing can wait on a person (the approval the merge
gate needs, or a review receipt naming the head, [`38-land.md`](38-land.md)),
which is also past any queue's
`check_response_timeout_minutes`. Past the fence the hold rests on renewals at half the lease: each step
of the run, whatever its stage, each receipt handed back, each status render,
the load gate's waiter, while it waits and while its command runs, for each
run in progress that lives in its checkout or has a lane whose worktree is
that checkout, except an issue lane parked at land, which the drain's move
renews (a round skips a checkout's runs while another invocation holds its
run lock, and the next round renews them; when it exits, its command done or,
without one, its gate open, the waiter waits at most five seconds for a round
in flight to finish and be said, and a renewal still pushing then goes
unsaid, since a waiter lingering on a slow push reads as queued and holds
every other gate shut), and
each drain move for a drain lane parked at land or in progress, waiting on the pace or a judgement or not, unless it waits on origin or stops at a given-up outage
(spc-2610161044461448 step 7). The landing checks the reservation before its
push as the fence does before arming, renewing it with its own term, so a
reservation that moved while the lane worked pushes nothing and opens no
pull request. A run neither stepped, given a receipt, shown, gated
nor drained over for longer than its lease loses its reservation; one that finds
it held by another person or copy says so and logs `reservation_lost` once. Once the run's last lane has landed, the
step pushes a tombstone over the reservation while it is still this person's
from this copy, leaves another's as it stands and names the holder, and a
tombstone that fails is named without undoing the landing; the lane branch
removal touches only the lane's `build/` branch. A run keyed by an issue whose
lane is handed back pushes a tombstone over its reservation as the step (or
the receipt) hands it back, logging `reservation_released` with the outcome
(`reservation_release_failed` when the push fails, or the tip refuses the
release for a reason of its own, never `reservation_released`; a tombstone
the cap would refuse with its hand-back is pushed plain, saying why: the
drain's route tries
again for a lane a drain opened, and nothing retries one `abcd build <iss-N>`
started, whose reservation lapses with its lease),
whether a drain or `abcd build <iss-N>` started it (iss-2610142247197128); an
intent's run keeps its reservation through a hand-back. A run holding no reservation
is neither renewed nor tombstoned, and its own reservation is not fenced.

The land fence also reads the release hold (itd-2609151838312703 C7,
spc-2610150728396006 step 4). Every run's fence, one holding a reservation or
not, fetches the reservations from origin first (for a run holding one, the
fetch its renewal makes), then arms nothing, as contention, while a release
reservation (`rel-vX.Y.Z`, [`37-reserve.md`](37-reserve.md)) is live and not
this copy's own: the refusal names the release and its holder, or says it is
the person's own other copy, until when it holds, and that a session cutting a
release has the final say on what merges before its tag, and leaves the pull
request open and unarmed for the cutting session to merge or hold. The copy
cutting the release is not held, since its landing is its own ruling. A
release whose tag is on origin, or whose holder released it, holds nothing; an
unreadable release branch within its hold holds like a live one. A release
dated more than five minutes ahead holds from this copy's first sighting of
it, which the judgement keeps whichever fetch brought it; one whose sighting
cannot be kept is not counted, and the arming's note names it
([`37-reserve.md`](37-reserve.md)). An origin the fetch cannot read arms
nothing, since a hold it cannot see may be there; a checkout with no remote
named origin has no reservations and is not held.

A run keyed by an issue (decision 10 on itd-2609201916151817, the lane the
drain opens for each eligible issue) has one lane. Its brief is the issue's
record with its remedy as the work; its receipt must declare the issue fixed,
and the landing resolves it with the commit the receipt names. Its receipt may
instead hand the issue back, naming the kind of decision the lane found, the
reason and, for a design finding or a second package, the home: the receipt
verb then discards the lane's worktree and branch, records the discarded head,
and ends the lane before its validators, and the drain routes the hand-back by
kind ([`35-drain.md`](35-drain.md)).

## The stop and the take-over

A stop is a recorded state every run reads before each move, never a signal to a
terminal, because a stop can only take hold at a tool-call boundary
(dcn-2611031722512037). The person's own command, from their own account, writes it;
the same command lifts it, and the lift is recorded; the guard refuses both to an agent.
The account's home is the account database's, whatever the environment says. With a shared folder chosen at setup in each account
([`01-ahoy.md`](01-ahoy.md)) the stop is
`<folder>/abcd-stop-<number>/abcd-stop.json`, in a directory of the person's own, named by their account's number, on a machine no other person shares
(dcn-2611041550139264); without one it is `~/.abcd.noindex/stop.json`, which reaches
only that account's runs. The file in the folder is the one file outside abcd's
owner-only rule. It is a request to stop and nothing more: judged on one descriptor,
never followed through a symlink, at most 4 KiB, owned by the account named at setup,
and no byte of it is read. Anything else there is refused, named on each move, and is
no stop. A folder others can write needs the sticky bit and an owner that is root or the
person, and each parent above it is held to the same rule, so another account can remove
nothing and move nothing away.

One check stands before every move, and every entry to a move makes it. The release and
the discard of a held lane, a person's decisions over it, do not make it.
Under a stop no lane opens, no stage is handed out and no agent starts, and the run
record takes an entry naming who set it and when. A receipt is collected as
ever, and the lane goes no further. An agent is live by its launch's process, or for
a host's agent while its driver lives, or with no driver recorded for one reservation
lease. A step killed outright leaves its agent unknown until the claim's term ends. A take-over
names an unknown agent and leaves it to an explicit restart, which does not judge its state. While one is live its lane is left as it stands. Then the lane's uncommitted work is
saved aside as a restart saves it, an agent lost is marked kept, and with every lane
settled, a save that failed holding its lane, the run's reservation is released and the run is halted.
A run fixes the shared folder and the person at its start, in its control record, and
reads that folder, the one the setting names and the platform's shared folder before each
move, the last only for this account, the one the setting names or the one the run started
under: what any other account leaves there is not honoured and is said, on each step's answer, the
listing and the setup board, and a lift names a stop it cannot reach. A step under a stop claims no driver, so the agents are judged by the
driver they ran under. A setting that
cannot be read, that has gone or changed during the run, or a folder the run should read
whose stop cannot be looked at, holds the run: it moves
nothing and lets nothing go until the setting is put back, and only a read that saw
clearly and found no stop counts as a lift, or lets a run be taken over. A home that
cannot be found holds the run the same way. The chosen folder does not move while a stop
stands in it. A renewal that falls due under a stop settles the run first, unless the run's state
file states another version than the renewing abcd's, which leaves the run to its own driver's check;
its claim lapses at its lease if that check does not come first. With no agent left in its
call and its work saved, a run under a stop no longer renews its reservation; a run with work a save
could not take keeps it only while a gated waiter, a reading of the run's status or a drive still in an
agent's call or a preflight renews it, lapses at its lease otherwise, and the move's refusal says so. A take-over over a
record that has since finished on the default branch has nothing to reserve, and clears the
mark. A signal's stop has no file to lift, and
stays. A configuration changed during the run holds the save as it refuses a restart, and a
worktree that no longer stands holds nothing back. A stop lifted before the halt lets
the run go on, and a settled lane's agent is told again. Each account has its own page for the day,
`<folder>/abcd-digest-<uid>/<date>.md`, rewritten whole at each halting in that account, naming the stop,
every run of that account's it halted in this repository's worktrees and each checkout a driver worked in, what each left, and no account's name. A page that cannot be
written changes nothing about the stop: its answer and the listing say so, and a later check tries again.
A pull request armed before the stop can still merge under it: the stop's account holds its number, and the page and the text of the
listing name each with its disarm command, and nothing disarms it on its own. A halted run stays halted when
the stop lifts, until a take-over reserves it again. A stop that does not come from the file,
the terminal drive's first interrupt, takes the same order and writes no page.

A run records who drives it in `control.json`, beside its state: the live driver
(session, process, start time) and each in-flight launch with its route. One driver
steps a run: each step claims it, refused at exit 3 while another's process is alive.
The resume verb lists every unfinished run this account can read, in this repository's worktrees and each checkout a driver worked in, from those records and writes
nothing. A lane's own worktree is not read for runs, by the listing or the page, so a run in one is not listed. Taking a run over claims the driver, refused too while the process table cannot
say whether a recorded driver is gone, collects each receipt that has arrived before
anything else, reserves again what a stop released, saves aside and
names what a gone agent left, and re-tells its await; the next step starts the fresh
agent, so nothing is started twice.

## The lost connection

A lost connection to the network (git, gh, a download) or to the model service
is one outage for the run (iss-2610080620372731), kept in the shared run state
beside the claims, so every session and lane in every worktree reads the same
one. The outage sub-verb renders it, and its own sub-verbs report a loss (the
service, what noticed it: the host, an agent or a tool, the lane and what it was
doing), logging `outage_start` when the report opens the outage; run the one
shared probe when it falls due, under a lease so one session probes while the
others wait; acknowledge the give-up's notification; and close the outage by
hand, logged as an intervention. The probe falls due a minute after the outage
opens, then five and ten minutes after each failed probe, then hourly; the
failed probe eight hours into the hourly stage gives up, stops the run and
raises one notification for the product thinker, held until acknowledged. Each
probe tests only what failed: the network by the git remote, the model service
only by the verdict of a canary agent the lead runs, since the probe calls no model.
A usage or rate limit is not an outage. The report lists every outage with its
minutes, its outcome and what was retried, so a short one nobody was told of
still appears.

The loop's half is in [`34-build.md`](34-build.md): a step holds only the lanes
the outage reaches, refuses every move once the run gave up, and restarts a lane
whose implementer died or yielded from its last commit, the uncommitted edits
saved aside.

## Exit codes

`0` done, and every status of the load check but a wait that gives up; `2` refused (an unrecognised input, a session that has not joined, a
bound the role does not permit, no checkout to key a run on), with nothing
written for the refused act; `3` contention (the record is claimed by another
session, or the run state is locked) — back off and take other work. The
outage probe gives `3` a meaning of its own, wait: the probe is not due, another
session holds it, or a service is still down; and it exits `2` once the run gave
up. The load check's wait exits `1` when it gives up at its timeout or refuses
to wait at all: a disk below the floor, a setting it cannot use, or a hold under
the abcd home that cannot be placed, judged or written; with a command to run, it exits with that command's code,
`128+N` when signal N killed it, or `127` when the command could not be
started. The JSON form holds on every path: a refusal is the `{"abcd":"error",…}` envelope on stdout.


<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd implement`

Sub-verbs: `abcd implement check`, `abcd implement claim`, `abcd implement join`, `abcd implement leave`, `abcd implement load`, `abcd implement log`, `abcd implement mode`, `abcd implement outage`, `abcd implement receipt`, `abcd implement record`, `abcd implement release`, `abcd implement report`, `abcd implement resume`, `abcd implement status`, `abcd implement step`, `abcd implement stop`, `abcd implement widen`.

Flags: none.

### `abcd implement check`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--path` | stringArray |
| `--session` | string |

### `abcd implement claim`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--lane` | string |
| `--lease` | duration |
| `--path` | stringArray |
| `--session` | string |

### `abcd implement join`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--ceiling` | int |
| `--model` | string |
| `--reason` | string |
| `--role` | string |
| `--session` | string |

### `abcd implement leave`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--reason` | string |
| `--session` | string |

### `abcd implement load`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--narrow` | bool |
| `--race-gate` | bool |
| `--site` | string |
| `--timeout` | duration |
| `--wait` | bool |

### `abcd implement log`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--field` | stringArray |
| `--session` | string |

### `abcd implement mode`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--session` | string |
| `--window` | int |

### `abcd implement outage`

Sub-verbs: `abcd implement outage ack`, `abcd implement outage clear`, `abcd implement outage probe`, `abcd implement outage record`.

Flags: none.

### `abcd implement outage ack`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--session` | string |

### `abcd implement outage clear`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--reason` | string |
| `--session` | string |

### `abcd implement outage probe`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--model` | string |
| `--session` | string |

### `abcd implement outage record`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--kind` | string |
| `--lane` | string |
| `--service` | string |
| `--session` | string |
| `--what` | string |

### `abcd implement receipt`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--run` | string |

### `abcd implement record`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--run` | string |
| `--transcript` | stringArray |

### `abcd implement release`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--session` | string |

### `abcd implement report`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--date` | string |
| `--log` | string |

### `abcd implement resume`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--run` | string |
| `--take` | string |

### `abcd implement status`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--run` | string |

### `abcd implement step`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--accept-config-change` | bool |
| `--discard` | string |
| `--release` | string |
| `--restart` | string |
| `--route` | stringArray |
| `--run` | string |
| `--yielded` | string |

### `abcd implement stop`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--lift` | bool |

### `abcd implement widen`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--lane` | string |
| `--run` | string |

<!-- surface-appendix:end -->
