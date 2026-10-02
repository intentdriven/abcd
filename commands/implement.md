---
name: implement
description: "Share one autonomous run between sessions and drive the implement loop: Writes nothing bare, only the run state its sub-verbs name; refuses an unknown sub-verb."
argument-hint: "[join|leave|mode|claim|release|check|log|report|load|status|step|receipt|record] …"
block: agents
---

# `/abcd:implement` — share a run between sessions

An autonomous run is one session's by default. A second session may join it
for a window, and the run divides the work one of three ways — a claim per
record, whole batches per session, or the first building while the second
reviews and lands — and measures which way worked. This page is the run
machinery a driving session calls; the verb a person types to build an intent
is `/abcd:build`, and the implement loop it starts is driven from here (see
[Drive the implement loop](#drive-the-implement-loop)).

The shared run lives in the machine-scoped run state, `~/.abcd/runs/<root-sha>/`,
keyed on the repository's root commit, so sessions in different worktrees of
one repository share one run and no repository file. The shared run never
writes to the checkout; the implement loop writes only its state file, in the
checkout's gitignored local tier.

## See where the run stands

Bare invocation is read-only and creates nothing:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" implement --json
```

Report the `window` (the division mode in force, or none), the joined
`sessions` with their roles, and the `claims` — each with its holder, its lane,
when its lease ends, and whether it is still `live`.

## Join, and open a window

Every write acts for a joined session, named with `--session` on every call.
Join first, stating the role:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" implement join --session <id> --role first|second [--ceiling <n>] --json
```

Joining writes the session's record and a `session_open` line, and signals no
one: the first session learns of a second only by reading the run state, and
never waits on it. Joining again with the same role is a resume; asking for the
other role is refused.

A session states its own agent ceiling with `--ceiling`: the most agents it
runs at once (for a second session, kept on top of the first session's, never
instead of it). It is recorded and carried on the `session_open` line, and a
resume cannot restate it. abcd runs no agent: it counts the agents the
session's own `agent_start` and `agent_end` lines declare alive, since it
joined, matched by their `agent` field. An `agent_start` past the ceiling is
refused at exit 2 and the refusal logged (condition `agent_ceiling`); every
`check` reports `agents_alive` beside `ceiling`. At the ceiling, wait, log a
`ceiling_wait`, and log the `agent_end` of an agent that finished before
starting the next. An agent the session never logs — a fork, one started
outside the log — is invisible to the count, so never start one; if the run
went over the ceiling anyway, log a `ceiling_overrun`.

The first session opens each window by naming its mode — `single`, `claim`,
`batch` or `split-roles`:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" implement mode claim --session <id> --window <n> --json
```

The second session cannot set the mode.

## Claim a record before opening its lane

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" implement claim <record> --session <id> --lane <lane> [--lease 2h] [--path <file> …] --json
```

One claim file per record, created exclusively: of two sessions reaching for
one record, exactly one holds it. A claim is a lease (default two hours); a
session that dies holding one strands nothing, because a lapsed lease is
claimable again and the lapse is logged. A claim file nobody can parse (a session
killed mid-claim) holds its record for one minute from when it was written, then
lapses the same way, logged with reason `unparseable`. Claiming a record this
session already holds renews the lease. Release it when the lane is done, and leave when the
session stops:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" implement release <record> --session <id> --json
"${CLAUDE_PLUGIN_ROOT}/abcd" implement leave --session <id> --reason "<why>" --json
```

**Exit 3 means back off.** A record another session holds is refused at exit 3,
naming the holder, and logged as `claim_denied` (the second session also logs a
`backoff` naming the reason and the minutes the attempt spent). Take other work;
do not retry the same record in a loop. A locked run state is the same exit,
and the second session's backoff from it is logged the same way, with the
minutes it waited for the lock, a `join` that meets it included. A backoff that
cannot be logged (the session never joined) says so in the refusal.

## The second session's bounds

The second session is refused at exit 2, and the refusal is logged, when it:

- claims while it already holds a live claim — one lane at a time;
- claims in a `split-roles` window — there it reviews, audits and lands only;
- declares a `--path` in the reading corpus — every position's `object.paths` in
  the committed `.abcd/config/reading-presets.json`, plus that file — those
  lanes are the first's; when the preset file is absent or unreadable, any
  declared `--path` is refused, since nothing can say the lane is clear;
- reaches the release stage — only the first session cuts a release.

It also keeps its own agent ceiling (stated on joining, held against its logged
`agent_start` lines, reported by `check`).

The role these bounds key on is the session's own statement, not an
identity: the release refusal, like every bound here, rests on a cooperative,
unauthenticated role. Two sessions of one account can each write anything
under that account's home, so the bounds keep two cooperating sessions apart;
they are not a wall against a session that lies about its role.

Before a stage that is not a claim, ask:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" implement check release|lane|review|audit|land --session <id> [--path <file> …] --json
```

An allowed stage writes nothing, and the verdict names it in `stage`. On a
refusal, stop that stage and leave it to the first session; a stop condition the second session meets stops only itself.

## Log the run's events

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" implement log <event> --session <id> --field key=value … --json
```

One line per event, appended in a single write, so two sessions writing at once
each land whole lines. The events are `backoff`, `lane_open`, `lane_close`,
`agent_start`, `agent_end`, `ceiling_wait`, `gate_run`, `review`, `fallback`,
`stop`, `refusal`, `pr`, `capture`, `context`, `ceiling_overrun`,
`intervention` and `decision`. The session, window and claim events belong to
their own sub-verbs and are refused here.

An event missing a field the report reads is refused at exit 2, naming the
field, with nothing written:

| Event | Required fields | Checked when given |
|---|---|---|
| `backoff` | `reason`, `minutes` (a number) | |
| `lane_close` | `lane`, `outcome` | |
| `agent_start` | `agent` | |
| `agent_end` | `agent`, `role`, `model`, and `minutes` (or `wall_minutes`, `wall_min`), a number | |
| `ceiling_overrun` | `alive`, `ceiling`, `minutes` (numbers), `lane` | |
| `intervention` | `kind`, `by`, `what`, `why`, `autonomy_gap` | `at` (RFC 3339), `detected_after_min` (a number) |
| `stop` | `cause` | `last_productive` (RFC 3339), `noticed_after_min` (a number); `recovery` |
| `decision` | `what`, `alternative`, `why` | `at` (RFC 3339) |

An intervention's `kind` is one of `session_open`, `account`, `ruling`,
`restart`, `close_session`, `file_restore`, `permission` or `other`, and its
`autonomy_gap` says what abcd or the host would need so no person is needed. A
`decision` records a judgement call a person would normally make, with the
alternative not taken.

For the comparison to count them: a `lane_close` with `outcome=merged` (or
`landed`) is a lane landed; `backoff` and `ceiling_wait` carry `minutes` (a
`backoff` must also carry `reason`, or it is refused: contention the verb cannot
see, such as the merge queue, is logged this way with the minutes the
backed-off work cost); a `context` line carries `used_pct` (with `role` and `note`), the orchestrator's
share of its context window in use.

## Compare the modes

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" implement report [--date YYYY-MM-DD | --log <file>] --json
```

Read-only. Per mode: windows, wall clock, lanes opened and landed, the second
session's lanes landed, collisions, lapsed claims, backoffs and the minutes
backed off, agent minutes, ceiling wait, ceiling overruns and refusals, with
each session's share; and, per session across the run, its context lines and
the last `used_pct` seen. A join logged up to a minute before a window opens
counts in that window. Over the whole run, `evidence` counts the
interventions (by kind, with the minutes they went undetected), stops and
decisions; `missing_fields` names, per event, the lines lacking a field `log`
requires — lines written by hand or before the requirement, which the figures
read as absent; and `coverage` names each of `lane_open`, `lane_close`,
`agent_start`, `agent_end` and `gate_run` whose lines stop more than six hours
before the run's last line. Relay the last two whole: a figure they name is
short.
`leader` is the mode with the most lanes landed per wall-clock hour — a figure,
not a verdict: the run's own report names the mode it would keep and says why.
Relay any `unparsed` lines; they are counted nowhere.

## Drive the implement loop

`/abcd:build` starts a run of the implement loop in this checkout's local tier,
`.abcd/.work.local/run/<run-id>/state.json`, separate from the shared run state
above. Three sub-verbs drive it, each reading the state first and writing it
last, and a fourth reads its record at the end:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" implement status [--run <run-id>] --json
"${CLAUDE_PLUGIN_ROOT}/abcd" implement step [--run <run-id>] [--release <lane-id> | --discard <lane-id>] --json
"${CLAUDE_PLUGIN_ROOT}/abcd" implement receipt <path> [--run <run-id>] --json
"${CLAUDE_PLUGIN_ROOT}/abcd" implement record [--run <run-id>] [--transcript <path>]... --json
```

`status` renders every run (or the one `--run` names): its pace and the layer
each number came from, the slots in use out of its ceiling and the work the
ceiling holds back, whether it is paused and until when, its lanes, each lane's
spec step and next stage, each agent a lane awaits, a held lane with the lane
whose hand-back caused it, the head judged, the step it stopped before and the
two flags that decide it, the pending spec steps, the run's fallbacks from a
routed runner to the host (`fallbacks`, and in the text a count per runner and
per role) and the run record. It writes nothing.

A spec's **steps** and a lane's **stages** are two things: each spec step lands
as one lane, and the loop takes the lane through its stages. `step` performs the
run's next move and exits; the result names the lane, the stage it completed
under `performed_stage` and the lane's next one under `stage`. When a stage
hands work to an agent the result's `awaiting` names the `role` to start as a
fresh agent, the `brief` to hand it and the `receipt` path it writes; that work
moves only when `receipt` is called with that path and the receipt verifies. A
complete run says `complete: true`. A stage that fails leaves the state as it
was, so the next call performs it again, and a completed stage is never
repeated.

A run works in parallel up to its ceiling, the pace's `sub_agents`: each agent
handed work and not yet verified is a slot, implementers and validators alike,
and the result carries `slots`, `ceiling` and `alive` (every lane with anything
left, its stage and each await). Each `step` first performs a stage the binary
owns on any lane (the worktree, the brief, a round's close, a landing step, a
sync, a hold), which takes no slot and is never held by the ceiling. Then, while
a slot is free, it hands out the
first waiting work: a lane already open before a new one, the lower spec step
first, a round's validators in order, then the implementer of a new lane. A
`step` that finds the ceiling reached hands out nothing, exits 0 with
`ceiling_reached: true` naming every await, and records the held work under the
run's `waiting` with the time it was first held; the move that later serves it
records the minutes it waited. A lane opens for a spec step once every step it
needs has landed (the step's `- needs:` line, or by default every step before
it), whatever the ceiling: its worktree and brief are made, and only its
implementer waits for a slot. A landing waiting on the forge's merge holds only
its own lane: the call moves another lane, names the wait under `blocked` (a
`blocked:` line in the text form) and in `next`, and gives the wait (exit 3)
only when nothing else moves. Any other refusal of a stage the binary performs,
a missing preflight receipt included, is the call's answer, and no other lane
moves.

`receipt` looks the path up among every outstanding await of the run and
advances the lane it belongs to; a path no await names is refused, naming the
awaits there are, and frees nothing.

When the stage hands the lane to a role that `roles.<role>.runner` in
`~/.abcd/config.json` routes to a command-line runner (`claude` or `opencode`,
enabled under `runner.<name>` there), `step` starts it itself, in the lane's
worktree, with the same brief and receipt path; the claude runner runs in print
mode with `--bare`, so the repository's hooks, plugins and configured servers do
not run, and opencode runs with `--pure` and its project configuration, its
`CLAUDE.md` reading and its external skills switched off, so the repository's
instruction files, settings, agents, skills and plugins do not reach it. A route
the repository's `.abcd/config.json` sets to a runner is skipped with a warning
on stderr: the role is the host's, as if unrouted, and no fallback is recorded.
One it sets to `host` keeps the role on the host over the machine's route to a
runner, with a warning naming the repository's file, the role and the machine
route it displaced.
The runner's receipt is verified by the stage's own verifier: a verified one
completes the stage in the same call, and the result's `route` names the runner
that ran it. A runner that is absent, refuses, fails, runs past its time or
writes a receipt that does not verify leaves the lane awaiting, and the result
names `awaiting` as with no runner plus `fallback` (the role, the runner asked
for, the reason, the route that runs it); start the agent as for any await. A
`step` that re-tells an await starts no runner. The runner configuration is read
on every `step`; a fault is refused at the `runner` stage before anything runs.

`step` keeps the run's window clock, on the pace the run started with
(`/abcd:build`). Once the window's working minutes have elapsed, `step` starts
nothing, writes `next_eligible_at` (now plus the run's pause), records the
pause, and exits 0 with `next_eligible_at` in the result and `next` naming the
time; an agent already started may still hand its receipt back. Before the
run's `next_eligible_at` a `step` is refused as a pause (exit 3) naming the
time, and nothing changes; at or after it a new window opens and the stage
proceeds.

Without `--run`, both act on the one run in progress in this checkout, and are
refused naming the runs when there are several. A refusal exits 2 (3 on a pause
or a locked state), writes nothing, and under `--json` comes as its own document
before the error envelope, naming `refusal.stage`, `refusal.reason` and
`refusal.remedy`.

The lane's stages are `worktree` (the lane's worktree in
`~/.abcd/worktrees/<root-sha>/<run-id>-<lane-id>`, on a branch
`build/<run-id>-<lane-id>` cut from the default branch), `brief` (the lane's
brief, rendered from that base into
`.abcd/.work.local/run/<run-id>/<lane-id>/brief.md`, naming the spec step the
lane builds and each step before it with what landed it), `implement` (awaits an
`implementer`'s receipt at `.abcd/.work.local/run/<run-id>/<lane-id>/receipt.json`),
then `validate` and `land`. An implementer's receipt is one strict JSON object:
`schema_version`, `run_id`, `lane`, `branch`, `commits` (full object names),
`definition_of_done` (`command`, `exit_code`, `output`), `report`, an optional
`model`, and an optional `resolves` list naming each capture the lane fixed
(`issue`, the `commit` of the receipt's that fixed it, `note`, `impact`,
`grounds`), with `output` and `report` paths inside the lane's directory.
`receipt` refuses it, naming every gap, unless each commit is on the lane's
branch past its base, the definition of done's output exists with exit code 0,
the report exists, and each fixed capture names one of the receipt's commits and
an impact; any other field, a verdict included, refuses it.

An issue-keyed run (`build <iss-N>`, the run `/abcd:drain` starts for each
eligible issue) has one lane. Its brief is the issue's record read at the lane's
base, its remedy as the work, and the definition of done a detector watched to
fail before the fix and pass after. Its receipt must name the issue in
`resolves`, or `receipt` refuses naming it; its validators take no fidelity
audit, and `land` resolves the issue and opens one pull request. A receipt may
instead carry `handback` (`kind`: `user-visible`, `trust-rule`,
`design-finding` or `second-package`; `reason`; and `home`, required for the
last two) with no `resolves` and no definition of done: `receipt` then discards
the lane's worktree and branch, ends the lane at `handed-back` before the
validators, and the result's `hand_back` names the kind, the reason, the home
and the `discarded` head. `/abcd:drain` routes it by kind.
`validate` hands the lane's head to fresh validators, side by side up to the
ceiling, and writes a fix brief only once every validator of the round has
returned; it records each verdict from the validator's own return; the fidelity audit passes only
when every criterion is met, so an undecided (`INCONCLUSIVE`) criterion sends
the lane to a fresh implementer as a not-met one does. A lane that has taken
the run's fix rounds (`build --fix-rounds`, bundled 3) and still does not pass
is handed back: the result's `hand_back` names the verdict `unachievable` and
the last findings, and the loop starts nothing further for it. Its sibling
lanes finish under the same ceiling, window and fix rounds; no new lane opens,
pending steps stay pending, and no lane closes the spec. A sibling whose round
passes is **held**: its stage is `held` and its `hold` names the `cause` (the
handed-back lane), the `head` its round judged and `before`, the landing step it
stopped before (`push`, or `arm` once its pull request is open; an armed one is
disarmed with `gh pr merge <n> --disable-auto`). Where the forge refuses the
withdrawal, `step` refuses naming the pull request, moves no other lane, and the
person decides it on the forge; an armed pull request the forge reports merged
had landed before the hand-back and is recorded as landed. Once nothing is left to move,
every `step` refuses at the `handed-back` stage naming the hand-back and each
held lane. The person decides each held lane, one per invocation, once no lane
has work left:

- `step --release <lane-id>` lands it as it is: its stage returns to `land` and
  its landing resumes at the step it stopped before.
- `step --discard <lane-id>` does not land it: its worktree and branch are
  removed, then its pull request is closed if it opened one, its stage is
  `discarded`, and its spec step stays unlanded. A removal git refuses (a
  worktree with changes) leaves the pull request open and the lane held, so the
  retry closes it once.

Either is refused, changing nothing, for a lane that is not held or while a lane
still has work. The run stays in progress, so `build next` passes over its
intent; no verb clears it, and the refusal names the way out: once the intent
is replanned, remove the run's directory, `.abcd/.work.local/run/<run-id>`.

`land` takes one `step` per move, and the lane stays at `land` until the last.
Landing is one lane at a time: a lane waits at its landing, holding no slot,
while a sibling's landing is under way, the lower spec step landing first.
Before a lane's landing begins, a sibling of the run that landed since its base
is merged in (a **sync**): the default branch is merged into the lane's branch
with a merge commit in its worktree, never a rebase, and a fresh round judges
the merge head. A merge that conflicts is aborted with the branch unchanged, and
a fresh implementer is handed a sync brief naming each conflicting path and the
sibling lanes; its receipt must carry the merged sha as an ancestor of its head.
A sync counts no fix round. The closing lane, which reaches its landing with no
step pending, no other lane open and none handed back, takes the fidelity audit
over each of the run's lanes' own diff.

1. It checks the lane's worktree is clean and its branch is at the head the
   validators judged.
2. On the lane that closes the spec it runs `spec close` in the lane's worktree
   and ingests the verdict of the audit that lane took, and for each capture the
   lane's receipts declared fixed it runs `capture resolve` with that commit. It
   commits them on the lane's branch with `Delivers:` (when the close ships the
   intent) and `Resolves:` trailers, and an `Assisted-by:` naming the model the
   lane's receipts reported, since the records carry that model's prose (a lane
   whose receipt reports no model is refused). The commit runs the
   repository's hooks; one that refuses stops the landing, which resumes once
   what the hook names is settled.
3. It pushes the lane's branch only once the repository's preflight receipt
   (`.abcd/.work.local/preflight-receipts/<head>`, in any worktree) names the
   lane's head. Without one, `step` refuses naming it: run `make preflight` in
   the lane's worktree, then `step` again. The push runs the pre-push hook and
   never skips or forces anything.
4. It opens the pull request through `gh`, with a body written from the run's
   records and passed through the outbound scrub, then re-reads the body the
   forge holds and strips a session URL or tool footer the harness appended.
5. It reads the merge rule from the ruleset mirror (`.abcd/work/rulesets/`) at
   the lane's base: where a merge queue gates the default branch AND a ruleset
   requires a person's approval (an approving review count of one or more, or
   a code-owner review with a CODEOWNERS file naming an owner) it arms
   auto-merge with the queue's method. Elsewhere it leaves the pull request
   open for a person to merge, and `implement status` shows the landing as
   "left open for a person to merge: the ruleset requires no approval" where a
   queue exists but nothing requires approval; a later step never arms it, and
   a missing mirror requires nothing. Nothing is pushed to the lane after this.
6. It waits (exit 3) until the pushed head is an ancestor of the default branch
   on `origin`, then removes the lane's worktree and branch, and the lane is
   done. A pull request closed without merging, or merged in a way that rewrote
   the head, is refused and nothing is cleaned up.

Every landing step is recorded as it completes, so a killed `step` repeats the
move that did not complete and finds what it made rather than making it twice.

`record` renders a run's record: each lane with its receipts and the model each
runner reported, every verdict the loop recorded, the route that ran a receipt's
or a return's agent when a runner ran it, the captures it fixed, its pull
request and landing, every fallback with its count per runner and per role
(`fallbacks`, `fallback_counts`), the transcripts captured, and the record's
lines.
Without `--run` it reads the one run in progress, or else the latest run. With
`--transcript <path>` (repeatable) on a complete run it captures each transcript
into the history store as `history capture <path>` does, one capture per path,
and records it; on a run in progress it refuses at the `record` stage. A
transcript stored without the scanner coverage the repository armed (gitleaks
configured and not installed) carries `scan_gap` in `--json` and a `scan gap:`
block in the text, as `history capture` names it; relay it as printed. Report
every refusal as it is.

## Check the machine's load

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" implement load --site preflight|eval-harness [--json]
```

`make preflight` runs this first and the eval harness runs it once at its
start, so it rarely needs calling by hand. It reads the machine's load and
process table once and warns about two things: a program outside the running
work that has used nearly all the CPU it could get for longer than the stray
limit (30 minutes by default), and a one-minute load average above the extreme
limit (four times the online core count by default). What a program could get
is its fair share: the online cores divided by the one-minute load, never more
than one core. So on a loaded machine a stray can show well under 100% of a
core: forty busy loops on 16 cores each show about 40%, and all forty are
strays. The person's programs and other accounts' are judged alike. It never
refuses, never waits and never stops anything; it exits 0 on every `status`: `ok`, `warning`, `skipped` (in CI, with
the `reason`) and `unchecked` (a platform other than macOS and
Linux, or a read that failed, with the `reason`).

Relay a `warning` whole. The person's own strays (`own_strays`) are named with
pid, process group, age and CPU share, and `remedy` gives, per stray or per
wholly-stray group, a re-check to run before each kill: `pgrep -g <group>` must
list only the named pids before `kill -- -<group>`, and `ps -o pid=,comm= -p
<pid>` must still show the program before `kill <pid>`. Never stop anything
yourself, and never by pattern (`pkill -f`, `killall`): the choice to stop is
the person's. Other accounts' strays (`other_strays`) are only a count and a CPU
total; say nothing more about them. A name the private banned-names layer
matches reads `[private name]`.

The limits are per machine, in `~/.abcd/load-limits`, which the check reads and
never creates: `stray-minutes <1 to 10080>` and `extreme-load <load>`, one per
line, `#` for comments. An unusable file is reported (`limits.malformed`) and
both defaults are used. Inside an autonomous run, a warning is also written to
the run log as a `load` event (`run_log`).

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
