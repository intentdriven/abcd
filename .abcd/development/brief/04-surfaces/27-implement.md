# `/abcd:implement` — Share a Run Between Sessions

`/abcd:implement` is the run machinery an autonomous run calls. It holds a run's
shared state in the machine-scoped store, records the sessions that join it,
makes a record the unit of exclusion between them with a claim, keeps a second
session inside its bounds, and derives the comparison of the three ways of
dividing work from the run log (itd-2609221656373558, spc-2609221657588816).

It is the family the implement loop extends: `implement step` and
`implement receipt` (itd-2609201916151817, decision 8) are later sub-verbs of
the same verb, and the pacing intent (itd-2609201925079472) reads the same run
state. `build` is what a person types; `implement` is what a driving session
calls.

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
(`reading_corpus_lane`), or reaches the release step (`second_session_release`).
Checking a step that is not a claim — a lane, the release, a review, an audit or
a landing — asks before it; an allowed step writes nothing.

The second session's own agent ceiling (criterion 5) is recorded and reported,
not enforced. The session states it when it joins (1 to 64); the
record and the `session_open` line carry it, a resume cannot restate it, and
every check verdict reports it (`ceiling`). abcd runs no agent and counts none
— `agent_start` and `agent_end` are lines the session writes — so there is no
count here to hold it against; keeping it, and logging a `ceiling_wait` at it,
is the session's discipline, which the verdict puts in front of it at every
step. On a refused claim the second session
also logs a `backoff` with its reason and minutes.

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
no paths asks no corpus question. The release step is refused at
this verb, not inside the launch cut: the cut knows no session, and a gate
keyed on a flag the second session could omit would guard nothing.

## The log and the comparison

Logging appends one of the run's own events, with its key-value fields
(`backoff`, `lane_open`, `lane_close`, `agent_start`, `agent_end`,
`ceiling_wait`, `gate_run`, `review`, `fallback`, `stop`, `refusal`, `pr`,
`capture`, `context`). Every line carries `ts` (RFC 3339, UTC), `session` and `event`, then
the fields; it reaches the file in one `O_APPEND` write through
`fsutil.AppendLineIn`, so two writers each land whole lines. The session, window,
claim and load events are refused here: they are written by their own sub-verbs,
so the log cannot record a claim the run state does not hold, or a load warning
the check did not give.

The report derives, per mode, the windows, wall clock, lanes opened and
landed (a `lane_close` whose outcome is `merged` or `landed`), the second
session's lanes landed, collisions (`claim_denied`), lapsed claims, backoffs and
their minutes, agent minutes (`agent_end`'s `minutes`, `wall_minutes` or
`wall_min`, the key the run's hand-kept lines carry), ceiling wait and refusals,
with each session's share. An event belongs to the window open when it happened.
A `context` line is an orchestrator's context measurement (`used_pct`, `role`,
`note`); the report totals them per session across the whole run, not per mode,
with the last `used_pct` seen, because a session's context is carried across
windows. `leader`
is the mode with the most lanes landed per wall-clock hour — a figure the run's
report cites when it names the mode it would keep, not a verdict of the verb's.
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
outside the check's own parent chain older than the stray limit at a lifetime
CPU share of at least 0.9, or a one-minute load average strictly above the
extreme limit. Nothing is exempt by name: abcd's own lanes are exempt by time,
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

Sub-verbs: `abcd implement check`, `abcd implement claim`, `abcd implement join`, `abcd implement leave`, `abcd implement load`, `abcd implement log`, `abcd implement mode`, `abcd implement release`, `abcd implement report`.

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

<!-- surface-appendix:end -->
