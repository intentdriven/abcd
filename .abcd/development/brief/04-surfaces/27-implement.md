# `/abcd:implement` — Share a Run Between Sessions

`/abcd:implement` is the run machinery an autonomous run calls. It holds a run's
shared state in the machine-scoped store, records the sessions that join it,
makes a record the unit of exclusion between them with a claim, keeps a second
session inside its bounds, and derives the comparison of the three ways of
dividing work from the run log (itd-2609221656373558, spc-2609221657588816).

It is also the family the implement loop is driven through (itd-2609201916151817,
decision 8): `build` is what a person types, and the loop's status, step,
receipt and record are sub-verbs of this verb, which a driving session calls. The loop's
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
| `record` | — | shipped |

## Where the run lives

```text
~/.abcd/runs/<root-sha>/<YYYY-MM-DD>.jsonl    the run log, one event per line, per UTC day
~/.abcd/runs/<root-sha>/claims/<record>.json  one claim per record
~/.abcd/runs/<root-sha>/sessions/<id>.json    one record per joined session
```

The key is the full root-commit SHA, the key the transcript, history and voyage
stores use, resolved from the checkout the caller stands in. Two sessions in two
worktrees of one repository share one directory and write no repository file.
Every level is created one directory at a time and proved real, never through a
symlink; outside a checkout, or in a repository with no commit, every sub-verb
refuses and nothing is created. Bare `abcd implement` and the report create
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
one) blocks nothing but its own record, and that only for a one-minute grace
from the file's modification time: the status lists it as unreadable, leaving
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
it, and a resume cannot restate it. abcd runs no agent, so the count it holds
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
`ceiling_wait`, `gate_run`, `review`, `fallback`, `stop`, `refusal`, `pr`,
`capture`, `context`, `ceiling_overrun`, `intervention`, `decision`). Every line carries `ts` (RFC 3339, UTC), `session` and `event`, then
the fields; it reaches the file in one `O_APPEND` write through
`fsutil.AppendLineIn`, which refuses a symlinked or non-regular leaf as its read
twin does, so two writers each land whole lines and a log leaf planted as a link
onto a claim file appends nothing. The session, window,
claim and load events are refused here: they are written by their own sub-verbs,
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
and `autonomy_gap`; a `stop` its `cause`; a `decision` its `what`, the
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
(itd-2609231434459890, spc-2609231542463113): once as the first prerequisite
of `make preflight`, and once in the eval harness's
`TestMain`, never once per test package. It reads the machine through
`internal/core/machineload`, a standard-library-only leaf (macOS: the
`vm.loadavg` and `hw.activecpu` sysctls and one `/bin/ps` run; Linux: `/proc`
and `/sys/devices/system/cpu/online`), and warns on two triggers: a process
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

The limits live in the caller's machine tier, `~/.abcd/load-limits`
(`stray-minutes`, default 30; `extreme-load`, default four times the online core
count), read through the guarded declaration read and never created; an
unusable file is reported loudly and both defaults are used. On a CI runner
(`GITHUB_ACTIONS=true`, or a `CI` other than empty, `false` or `0`) nothing is
read and the line says why; each CI job that starts the harness runs the check
as a step first, so the reason reaches the job log. On another platform, or when
a read fails, the check says it could not check. Inside a live run (a run state
with a joined session) a warning is also written as a load event, attributed
to the first-role session, carrying the verdict's facts so it renders exactly
as it was printed; the run's hand-written load samples share the name and carry
no `triggers`. The check exits 0 on every status: it never refuses, waits or
signals anything.

## The implement loop

Three sub-verbs drive the loop a build starts, each over the run's state file in
the checkout's local tier ([`34-build.md`](34-build.md) states the file, the
checks and the step interface). The status render reads every run, or the one
named, and writes nothing: it names the slots in use out of the run's ceiling,
every lane alive with its stage and each agent it awaits, and each held lane with
its cause, the head judged, the landing step it stopped before and the two flags
that decide it. The step verb performs the run's next move and exits (a lane's
stages are worktree, brief, implement, validate and land; the lane as a whole
lands one of the spec's steps). A run works in parallel up to its ceiling (ruling
DR6): a stage the binary owns moves on any lane first, then, while a slot is
free, the first waiting work takes it, an open lane's before a new lane's and
the lower spec step first; a lane opens for a ready spec step whatever the
ceiling, and only its implementer waits for a slot; at a stage that hands work
to an agent it names the agent, the brief and the receipt path, and a step that
finds the ceiling reached hands out nothing and names every agent out. A landing
waiting on the forge's merge holds only its own lane; any other refused stage
the binary performs is the step's answer. The receipt verb looks the path up
among every outstanding await of the run, and the stage completes only when a
lane awaits that path and its verifier accepts it; a verified receipt frees its
slot. After a hand-back the siblings finish and a lane whose round passes is
held before it pushes or arms (ruling DR6c), an armed one disarmed, or, where
the forge refuses the withdrawal, the step refused naming the pull request; the
person's word on a held lane is given through the step verb, one lane per
invocation: release lands it as it is, and discard removes its worktree and
branch, then closes its pull request, and leaves its step unlanded, each refused, changing nothing, unless the lane is
held and no lane has work left.
Without a named run, the step
and receipt verbs act on the one run in progress in the checkout and refuse naming
the runs when there are several. Their refusals name the stage, the reason and
the remedy, and a pause
before the run's next eligible time, a lock held by another invocation, or a
landing waiting for its pull request to merge, is contention at exit 3. The
record verb reads a run's record back at the end, and on a complete run captures
the run's transcripts into the history store, one capture per path; a
transcript stored without the scanner coverage the repository armed carries its
scan gap on the record, as the history verb reports it for a capture.

A run keyed by an issue (decision 10 on itd-2609201916151817, the lane the
drain opens for each eligible issue) has one lane. Its brief is the issue's
record with its remedy as the work; its receipt must declare the issue fixed,
and the landing resolves it with the commit the receipt names. Its receipt may
instead hand the issue back, naming the kind of decision the lane found, the
reason and, for a design finding or a second package, the home: the receipt
verb then discards the lane's worktree and branch, records the discarded head,
and ends the lane before its validators, and the drain routes the hand-back by
kind ([`35-drain.md`](35-drain.md)).

## Exit codes

`0` done, and every status of the load check; `2` refused (an unrecognised input, a session that has not joined, a
bound the role does not permit, no checkout to key a run on), with nothing
written for the refused act; `3` contention (the record is claimed by another
session, or the run state is locked) — back off and take other work. The JSON
form holds on every path: a refusal is the `{"abcd":"error",…}` envelope on stdout.


<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd implement`

Sub-verbs: `abcd implement check`, `abcd implement claim`, `abcd implement join`, `abcd implement leave`, `abcd implement load`, `abcd implement log`, `abcd implement mode`, `abcd implement receipt`, `abcd implement record`, `abcd implement release`, `abcd implement report`, `abcd implement status`, `abcd implement step`.

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
| `--site` | string |

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

### `abcd implement status`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--run` | string |

### `abcd implement step`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--discard` | string |
| `--release` | string |
| `--run` | string |

<!-- surface-appendix:end -->
