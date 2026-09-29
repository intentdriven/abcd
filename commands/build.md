---
name: build
description: "Start the loop that takes one READY intent to delivered: Writes the run's state file in the local tier; refuses an open question, a hold or a peer holding it."
argument-hint: "<itd-N>"
block: people
---

# `/abcd:build`

Take one intent from READY towards delivered with the loop holding the run, not
this conversation. The run lives in a state file; every invocation reads it,
does at most one step, writes it and exits, so a session that stops, is
compacted or is killed loses nothing, and the next invocation resumes where the
last one stopped.

## Start the run

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" build <itd-N> [--session <id>] [--pace <work-minutes>/<pause-minutes>] [--sub-agents <n>] --json
```

Pass `--session` with the host session's id when it has joined the shared run
(`implement join`): a new run then claims the intent there for that session,
with the run id as the lane, so a build of the same intent from any other
checkout of the repository is refused as held from the start, before this run's
lane has moved or claimed anything. The session's own live claim on the intent
is not counted as a peer's. A session that has not joined is refused at the
`claim` step with nothing written. Without `--session` the run holds no claim,
and the result says so (`claim` is null): another checkout cannot see the run
until its lane shows.

For an intent with no run in progress, the checks run first, and every one must
pass:

- `key` — the argument is an intent id. An issue id is refused: the issue key
  is not built yet.
- `ready` — the intent is READY: planned, its criteria written, its spec linked
  and written (the same gate `/abcd:intent` reports).
- `open_questions` — no open question under `## Open Questions`: every list
  item there counts unless the section opens with an italic `_All resolved …_`
  line or the item is explicitly marked resolved or deferred (`**Deferred**`,
  `resolved:`, `**explicitly deferred**`).
- `claim_sections` — the `## Mechanism` prompt is answered (or the section
  absent) and the scope conditions are recorded.
- `hold` — the intent carries no `held:`.
- `steps` — the spec's `## Steps` reads, and at least one step is not landed.
- `peers` — no peer holds the intent: no sibling worktree or local branch holds
  it in another bucket, and no session other than `--session` holds a live
  claim on it. A peer that
  cannot be read (a worktree git will not answer for, a ledger holding one id
  twice) and an unreadable claim count as holding it: what they hold is unknown.

A refusal writes nothing. It exits 2, or 3 when a peer holds the intent (back
off and take other work). Under `--json` the refusal comes as its own document
before the error envelope: `refusal.step`, `refusal.check`, `refusal.reason`,
`refusal.remedy` and every check's row in `refusal.checks`. Tell the user the
check, the reason and the remedy, and do not work around it: an open question
goes back to the planning interview, a hold to the person who placed it.

When the checks pass, the payload names the `run_id`, the `state` file
(`.abcd/.work.local/run/<run-id>/state.json`), the first `lane` (the spec's first
unlanded step), the `pending` spec steps, `claim` (the claim `--session` took,
or null),
and `next`, the move to make. The
local tier is never created: in a repository abcd does not manage the verb
refuses. Starting again while the run is in progress creates nothing, runs no
check, and reports `resumed: true` with the same run and an empty `checks`: the
run's own lanes move and claim the intent, so judging it again would refuse the
run as its own peer.

## The pace

A new run is paced without being told: a working window, a pause after it, and
a ceiling on the run's lanes and validators alive at once. The three numbers are
read once, when the run starts, each from the highest layer that sets it:

1. `--pace <work-minutes>/<pause-minutes>` (for example `--pace 90/240`) and
   `--sub-agents <n>`, for this run only;
2. `pace.work_minutes`, `pace.pause_minutes` and `pace.sub_agents` in the
   repository's `.abcd/config.json`;
3. the same keys in `~/.abcd/config.json`, for every checkout on the machine;
4. the bundled 120/300 with 2 sub-agents.

The payload's `pace` carries each number as `value`, `layer` (`flag`, `repo`,
`machine` or `bundled`) and `origin` (the flag as typed, or the file), and the
run record's `pace` line names the same. Tell the user which layer set the pace.
A malformed pace or ceiling, typed or configured (`--pace 90`, a work window of
0, `--sub-agents two`, a misspelt key under `pace`), is refused at the `pace`
step with exit 2, naming the value and the accepted form, and nothing is
written. Starting again keeps the run's pace: a flag naming another pace is
refused, and one naming the same pace resumes.

The window and the pause bind through `implement step` (below). The ceiling is
recorded with the run; this build does not count lanes against it.

## Drive it

The host session drives the loop. Take one step at a time:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" implement step --json
```

`performed` names the step the call completed. When a step hands work to an
agent, `awaiting` names the `role` to start as a fresh agent, the `brief` to
hand it and the `receipt` path it writes to. Start that agent, and when it
returns hand the receipt back:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" implement receipt <path> --json
```

The lane advances only on a receipt that verifies. Asking for a step while the
lane awaits a receipt re-tells what it awaits and moves nothing. When a lane is
done, the spec's next pending step opens the next lane, and the run record gets
a line naming it, as the start line names the first.

The run's window opens when the run starts. Once its working minutes have
elapsed, `implement step` starts nothing: it writes `next_eligible_at` (now plus
the run's pause) into the state, records the pause, and exits 0 with
`next_eligible_at` in the payload and `next` naming the time. An agent already
started may still finish: hand its receipt back as usual. Before
`next_eligible_at`, `implement step` is refused at the `pause` step with exit 3,
naming the time, and nothing changes; stop driving the run and invoke it again
at or after that time, when a new window opens. The pause lives in the state
file, so no process waits through it.
`"${CLAUDE_PLUGIN_ROOT}/abcd" implement status --json` renders every run, its
lanes and its record, and writes nothing. A lane's `worktree` is home-relative,
or its directory name when it sits outside HOME; its `brief` and `receipt` keep
their full home-relative paths, because the agent acts on them.

A lane's steps run in order:

1. `worktree` — the loop makes the lane's worktree in the machine-scoped store,
   `~/.abcd/worktrees/<root-sha>/<run-id>-<lane-id>`, on a branch
   `build/<run-id>-<lane-id>` cut from the default branch. Nothing is made
   beside the checkout.
2. `brief` — the loop renders the lane's brief from that base: the intent, the
   spec, the conventions of `AGENTS.md` and the decisions the intent cites; the
   spec step the lane builds and each step before it, with what landed it (the
   spec's `landed:` line, or the earlier lane of the run that built it); and
   where the implementer's report, the definition of done's output and its
   receipt go, and the outbound policy: no session URL or tool attribution
   footer in public text, and a re-read-and-strip of every pull request, issue
   and comment the implementer creates. An intent the default branch does not carry as planned is
   refused here: land its planning first. So is a spec whose steps the base no
   longer lists as the run started from them: a run does not follow steps
   reordered mid-run.
3. `implement` — `awaiting` names an `implementer`. Start a fresh agent with
   nothing but the brief; it works in the lane's worktree, commits on the
   lane's branch and writes its receipt. Hand the receipt back unedited. A
   receipt that names no commit on the branch past its base, no passing
   definition of done's output or no report is refused naming what is missing;
   relay the refusal to a fresh implementer rather than completing the receipt
   yourself.
4. `validate` and `land` — not carried in this build: `step` refuses at
   `validate` naming the spec piece that delivers it, and the run stays ready
   to resume in an abcd that carries it. Report that refusal as it is; do not
   review, open the pull request or close the spec by hand on the run's behalf.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
