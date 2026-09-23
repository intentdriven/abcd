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
refuses and nothing is created. Bare `abcd implement` and `report` create
nothing at all. Only `join` brings a run into existence: every other writer acts
for a session that has joined, and one invoked for a session no run holds is
refused before anything is created — no directory, no lock, no log line.

## Joining

`implement join --session <id> --role first|second` writes the session's record
by an exclusive create and logs `session_open` with the role. Nothing signals
any other session. A session that joins again with the role it holds is a
resume, logged with `rejoin`; asking for the other role is refused. `leave`
releases every claim the session holds, logs `session_close`, and removes the
record.

The role lives in that record and nowhere else: the environment is not a trust
input, because a variable a shell or a repository's configuration can set is not
a statement the session made. The record gives consistency, not authentication —
two sessions of one account can each write anything under that account's home —
so the bounds are a discipline two cooperating sessions keep, checked at every
sub-verb.

## The window's mode

`implement mode <single|claim|batch|split-roles>` logs `window_mode`; only the
first session sets it. The mode in force is the log's last `window_mode` line,
whoever wrote it, so a line the run writes by hand counts the same as one the
verb wrote.

## The claim

`implement claim <record> --lane <lane> [--lease <d>]` creates
`claims/<record>.json` exclusively through `fsutil.CreateExclusiveIn`; the create
is the exclusion, so of two sessions reaching for one record exactly one holds
it. A run-wide advisory lock orders the read-decide-write sequences around it (a
lapse and a re-claim, a cap count and a claim). The claim carries the session,
the lane, the time and the lease (default two hours, one minute to a day). The
holder claiming again renews its lease. A lease that has passed is claimable:
the lapse is logged as `claim_lapsed`, naming the previous holder, and the claim
is taken. A live claim held by another session is refused at exit 3 and logged
as `claim_denied` naming the holder. A granted claim is logged as `claim`; if
that line cannot be written the claim is removed again, so the directory never
holds a claim the log does not. `release` removes the holder's own claim and
logs `claim_released`; only the holder releases. A claim file nobody can parse
(a session killed between the exclusive create and the write leaves an empty
one) blocks nothing but its own record, and that only for a one-minute grace
from the file's modification time: the status lists it as unreadable, `leave`
and every other claim read past it, a claim on its record within the grace is
contention naming the file's full path, and after the grace the claim logs
`claim_lapsed` with reason `unparseable` and takes the record.

## The bounds

A session joined as `second` is refused at exit 2, with a `refusal` line naming
the condition, when it claims while holding another live claim
(`second_session_lane_cap`), claims or checks a lane in a `split-roles` window
(`split_roles_second_builds_nothing`), declares a path in the reading corpus
(`reading_corpus_lane`), or reaches the release step (`second_session_release`).
`implement check <lane|release|review|audit|land>` asks before a step that is not
a claim; an allowed step writes nothing. On a refused claim the second session
also logs a `backoff` with its reason and minutes.

The reading corpus is a stated list — `.abcd/config/reading-presets.json`,
`internal/core/{capture,grounds,intent,issueschema,lint,provenance}/` and
`commands/{capture,intent,reading}.md` — the set the run's routing found to move
the cold-reading windows (iss-2609211105023379). The release step is refused at
this verb, not inside `launch ship`: the ship verb knows no session, and a gate
keyed on a flag the second session could omit would guard nothing.

## The log and the comparison

`implement log <event> --field key=value …` appends one of the run's own events
(`backoff`, `lane_open`, `lane_close`, `agent_start`, `agent_end`,
`ceiling_wait`, `gate_run`, `review`, `fallback`, `stop`, `refusal`, `pr`,
`capture`, `context`). Every line carries `ts` (RFC 3339, UTC), `session` and `event`, then
the fields; it reaches the file in one `O_APPEND` write through
`fsutil.AppendLineIn`, so two writers each land whole lines. The session, window
and claim events are refused here: they are written by their own sub-verbs, so
the log cannot record a claim the run state does not hold.

`implement report` derives, per mode, the windows, wall clock, lanes opened and
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
`unparsed`, never dropped silently. `--date` reads one day, and `--log` reads one
file named directly.

## Exit codes

`0` done; `2` refused (an unrecognised input, a session that has not joined, a
bound the role does not permit, no checkout to key a run on), with nothing
written for the refused act; `3` contention (the record is claimed by another
session, or the run state is locked) — back off and take other work. `--json`
holds on every path: a refusal is the `{"abcd":"error",…}` envelope on stdout.
