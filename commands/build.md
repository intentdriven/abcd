---
name: build
description: "Start the loop that takes one READY intent to delivered: Writes the run's state file in the local tier; refuses an open question, a hold or a peer holding it."
argument-hint: "<itd-N> | <iss-N> | next"
block: people
---

# `/abcd:build`

Take one intent from READY towards delivered with the loop holding the run, not
this conversation. The run lives in a state file; every invocation reads it,
does at most one stage, writes it and exits, so a session that stops, is
compacted or is killed loses nothing, and the next invocation resumes where the
last one stopped.

Two words, two things. A **step** is a piece of the spec: the spec lists its
steps under `## Steps`, and each lands as one lane and one pull request. A
**stage** is what the loop does to a lane on the way: `worktree`, `brief`,
`implement`, `validate`, `land`. `implement step` performs one move of the
run; the payloads name the stage under `stage` and the spec's step under
`spec_step`.

## Start the run

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" build <itd-N> [--session <id>] [--pace <work-minutes>/<pause-minutes>] [--sub-agents <n>] [--fix-rounds <n>] --json
```

Pass `--session` with the host session's id when it has joined the shared run
(`implement join`): a new run then claims the intent there for that session,
with the run id as the lane, so a build of the same intent from any other
checkout of the repository is refused as held from the start, before this run's
lane has moved or claimed anything. The session's own live claim on the intent
is not counted as a peer's. A session that has not joined is refused at the
`claim` stage with nothing written. Without `--session` the run holds no claim,
and the result says so (`claim` is null): another checkout cannot see the run
until its lane shows.

An issue id builds the loop's issue-keyed lane instead: `build <iss-N>` (an
issue id by shape) checks that the repository's own drain rule takes the issue,
read as `/abcd:drain --dry-run` reads it, and that no peer holds it, then opens
one lane whose brief is the issue with its remedy as the work. The receipt must
name the issue in `resolves`, and the landing resolves it. `/abcd:drain` starts
these runs one at a time; see `/abcd:implement` for the lane's receipt and its
hand-back.

For an intent with no run in progress, the checks run first, and every one must
pass:

- `key` — the argument is an intent id or an issue id (`iss-N`, by shape), and
  anything else is refused; an issue id is checked as the issue-keyed lane
  above.
- `ready` — the intent is READY: planned, its criteria written, its spec linked
  and written (the same gate `/abcd:intent` reports).
- `open_questions` — no open question under `## Open Questions`: every list
  item there counts unless the section opens with an italic `_All resolved …_`
  line or the item is explicitly marked resolved or deferred (`**Deferred**`,
  `resolved:`, `**explicitly deferred**`).
- `claim_sections` — the `## Mechanism` prompt is answered (or the section
  absent) and the scope conditions are recorded.
- `hold` — the intent carries no `held:`.
- `blocked` — nothing the intent names in `blocked_by` is unsettled. An intent
  in `shipped/` or `disciplines/` is settled; one anywhere else, or one this
  checkout does not hold, blocks it. A superseded blocker is followed along
  `superseded_by` to the record that replaced it, transitively, and blocks only
  while that replacement is unsettled: an intent settles as above, and a
  decision (`adr-N`) settles when its status is `accepted`. A decision in any
  other status or missing from this checkout blocks, as does a chain that
  loops, ends at an intent this checkout does not hold, or stops at a
  superseded record naming no successor; the reason names the chain, and a
  settled chain is named in the passing row.
- `steps` — the spec's `## Steps` reads, and at least one step is not landed.
- `peers` — no peer holds the intent: no sibling worktree or local branch holds
  it in another bucket, and no session other than `--session` holds a live
  claim on it. A peer that
  cannot be read (a worktree git will not answer for, a ledger holding one id
  twice) and an unreadable claim count as holding it: what they hold is unknown.

A refusal writes nothing. It exits 2, or 3 when a peer holds the intent or the
run state is locked (back off and take other work). Under `--json` the refusal comes as its own document
before the error envelope: `refusal.stage`, `refusal.check`, `refusal.reason`,
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

## Pick the next intent

When the argument is `next`, let the run choose the intent:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" build next [--session <id>] [--pace <work-minutes>/<pause-minutes>] [--sub-agents <n>] [--fix-rounds <n>] --json
```

The candidates are the planned intents that pass every check above, judged by
the same function, less one this checkout already has a run in progress for.
Each is scored from its record, three parts at equal weight, each 0 to 100:

- criteria clarity: the share of its acceptance criteria written in
  Given-When-Then form, all three clauses present;
- test path: 100 when its spec's `## Footprint` section names the tests;
- footprint: 100 divided by the number of packages that section names, so
  fewer is readier.

A spec with no `## Footprint` section scores zero on the last two, and the
reason says the spec carries no footprint. The readiest is taken, the oldest
among equals; the reason then says the tie was broken by age.

The pick starts the run `build <itd-N>` would start for that intent, with the
same brief, worktree and receipt, and records the pick in the run's state. Its
reason is one grounds entry, `pursued: picked by run <run-id> on <date>; …`,
naming every candidate with its score, the rule, the runner-up and why it lost,
and the falsifier (fix rounds past the pace rule's count, or an unachievable
hand-back). A lane handed back after its fix rounds records the pick as
falsified in the run record, and the entry is left as written. The lane's `worktree` stage appends it to the intent in the lane's
own worktree and commits it there as the lane branch's first commit, a
record-only commit made before the brief. The receipt verifier does not count
it: a receipt naming it is refused, so the implementer names only its own
commits, and so is a receipt over a branch that no longer carries it after a
rebase or an amend. No existing entry changes, the checkout you run in is not written but
for the run state, and `intent ready` keeps reporting the person's entry as the
most recent conjecture.

The payload carries `candidates` (each with its `score` parts and `total`),
`excluded` (each planned intent a check excluded, with the `check` and the
`reason`), `pick` (`chosen`, `runner_up`, `tie_broken_by_age`, `rule`,
`falsifier`), `entry` (the reason's text) and `start`, the run as
`build <itd-N>` reports it. Tell the user which intent was picked and why, then
drive the run as below.

Refusals, each writing nothing:

- no candidate: refused at the `pick` stage, exit 2; `refusal.excluded` names
  every planned intent and the check that excluded it. Tell the user each one
  and do not work around it.
- `--max` above 1 or `--until-empty`: refused at the `pick` stage, exit 2.
  Picking again under the pace rule is not built in this abcd: run
  `build next` once per pick.
- the picked intent already has a run in progress: refused at the `pick` stage,
  exit 3; resume that run with `implement step`.
- a refusal of `build <itd-N>` itself (the pace, the claim stage) comes as that
  verb's refusal.

The pick commit is made with the git identity the repository's commits are
made with; with none configured the `worktree` stage is refused naming it.

## The pace

A new run is paced without being told: a working window, a pause after it, a
ceiling on the run's lanes and validators alive at once, and the fix rounds a
lane may take before it is handed back. The four numbers are read once, when the
run starts, each from the highest layer that sets it:

1. `--pace <work-minutes>/<pause-minutes>` (for example `--pace 90/240`),
   `--sub-agents <n>` and `--fix-rounds <n>` (0 to 64), for this run only;
2. `pace.work_minutes`, `pace.pause_minutes`, `pace.sub_agents` and
   `pace.fix_rounds` in the repository's `.abcd/config.json`;
3. the same keys in `~/.abcd/config.json`, for every checkout on the machine;
4. the bundled 120/300 with 2 sub-agents and 3 fix rounds.

The payload's `pace` carries each number as `value`, `layer` (`flag`, `repo`,
`machine` or `bundled`) and `origin` (the flag as typed, or the file), and the
run record's `pace` line names the same. Tell the user which layer set the pace.
A malformed pace or ceiling, typed or configured (`--pace 90`, a work window of
0, `--sub-agents two`, `--fix-rounds three`, a misspelt key under `pace`), is refused at the `pace`
stage with exit 2, naming the value and the accepted form, and nothing is
written. Starting again keeps the run's pace: a flag naming another pace is
refused, and one naming the same pace resumes.

The window and the pause bind through `implement step` (below), and so does the
ceiling: a run hands work to several agents at once, up to `sub_agents`, the
validators of one round side by side and the lanes of steps that do not need
each other beside one another. A step runs beside earlier steps only when the
spec's `- needs:` line under it says so (`- needs: none`, or `- needs: 1, 3`
naming the steps it waits for); without the line it needs every step before it.

## Drive it

The host session drives the loop, one move per call:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" implement step --json
```

`performed_stage` names the stage the call completed and `stage` the lane's
next one. When a stage hands work to an agent, `awaiting` names the `role` to start as a fresh agent, the `brief` to
hand it and the `receipt` path it writes to. Start that agent, and when it
returns hand the receipt back:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" implement receipt <path> --json
```

A lane advances only on a receipt that verifies, and the receipt path names the
lane it belongs to. While a slot is free, `implement step` hands out the next
waiting work (an open lane's validators or fix implementer before a new lane's
implementer, the lower spec step first); several agents may be out at once, so
start each as it is handed out. A step that finds the ceiling reached hands out
nothing and exits 0 with `ceiling_reached: true`, naming every agent out and its
receipt path: hand a receipt back, then step again. When a step's needs have
landed, its lane opens whatever the ceiling (its worktree and brief are made,
and its implementer takes the next free slot), and the run record gets a line
naming it, as the start line names the first.

A role can run through a command-line runner instead of an agent you start.
`roles.<role>.runner` in `~/.abcd/config.json` names `host` (the default) or a
runner, `claude` or `opencode`, that the machine enables under `runner.<name>`
in the same file (with an optional `model` route, `<provider>/<model>`,
admitted against that provider's allowlist). Only the person's own file hands a
role to a runner, because a runner spends their key: the repository's
`.abcd/config.json` may set a role to `host`, and a route there to a runner is
skipped with a warning on stderr, so the role runs on the host as if unrouted
(tell the user, naming the file the warning names). A repository route to
`host` stands over the person's own route to a runner, because it spends
nothing of theirs, and says so with a warning on stderr naming the repository's
file, the role and the machine route it displaced (tell the user that too). A person whose Claude
account is a subscription, with no API key, runs the role in their own session:
the claude runner runs bare, on an API key only. `build` reads this configuration before it creates the run, and a
fault, a model route off the allowlist included, is refused at the `runner`
stage with nothing created. When a step hands work to a routed role,
`implement step` starts the runner itself with the brief and the receipt path
you would be handed, in the lane's worktree; its transcript goes to abcd's
history store and its receipt is verified exactly as yours would be, so a
verified one completes the stage in the same call, and the payload's `route`
names the runner that ran it (`asked`, `ran`, `model`). Do not start an agent
for it. When the runner is absent, refuses, fails, runs past its time or writes
a receipt that does not verify, the payload still names `awaiting` as usual and
adds `fallback` (the `role`, the runner `asked` for, the `reason` and the route
that runs it): start the agent yourself as above. Every fallback is recorded;
`implement status` and `implement record` count them per runner and per role.
Tell the user each fallback's reason.

The run's window opens when the run starts. Once its working minutes have
elapsed, `implement step` starts nothing: it writes `next_eligible_at` (now plus
the run's pause) into the state, records the pause, and exits 0 with
`next_eligible_at` in the payload and `next` naming the time. An agent already
started may still finish: hand its receipt back as usual. Before
`next_eligible_at`, `implement step` is refused at the `pause` stage with exit 3,
naming the time, and nothing changes; stop driving the run and invoke it again
at or after that time, when a new window opens. The pause lives in the state
file, so no process waits through it.
`"${CLAUDE_PLUGIN_ROOT}/abcd" implement status --json` renders every run, its
lanes and its record, and writes nothing. A lane's `worktree` is home-relative,
or its directory name when it sits outside HOME; its `brief` and `receipt` keep
their full home-relative paths, because the agent acts on them.

A lane's stages run in order:

1. `worktree` — the loop makes the lane's worktree in the machine-scoped store,
   `~/.abcd/worktrees/<root-sha>/<run-id>-<lane-id>`, on a branch
   `build/<run-id>-<lane-id>` cut from the default branch. Nothing is made
   beside the checkout. In a run `build next` started, the first lane's
   worktree stage also commits the pick's reason as the branch's first commit.
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
4. `validate` — `awaiting` names each validator in turn: a `ruthless-reviewer`,
   a `security-reviewer` and, on the lane whose landing ships the intent, an
   `intent-auditor`. Start each as a fresh agent with its brief and hand its
   return back unedited; the loop records the verdict itself. A round one of
   them did not pass goes to a fresh `implementer` with the findings. The audit
   passes only when every criterion is met: an undecided (`INCONCLUSIVE`)
   criterion fails the round as a not-met one does. Once the lane has taken the
   run's fix rounds, a round that still does not pass hands the lane back: the
   result carries `hand_back` (`verdict` `unachievable`, the last `round`, the
   `fix_rounds` cap, the `findings` returns and the criteria `not_met` or
   `undecided`), and the run starts nothing further for it. The lanes beside it
   finish; no new lane opens and no lane closes the spec, and a lane whose round
   passes is held before it pushes or arms (`/abcd:implement` names the hold).
   Once nothing is left to move, every step is refused at the `handed-back`
   stage naming the hand-back and each held lane. Tell the user the intent is
   handed back to them with those findings, and ask, for each held lane,
   whether it lands as it is (`implement step --release <lane-id>`) or is
   discarded (`implement step --discard <lane-id>`); do not start another fix
   round. The run stays
   in progress until its directory, `.abcd/.work.local/run/<run-id>`, is
   removed, which the refusal names as the way to build the intent afresh once
   it is replanned.
5. `land` — one `implement step` per move, the lane staying at `land` until
   the last. The loop checks the lane's worktree is clean at the judged head;
   on the lane that closes the spec it runs `spec close` in the lane's worktree
   and ingests the audit that lane took, and it runs `capture resolve` for each
   capture the receipts named in `resolves`, with that commit, committing them
   on the lane's branch with `Delivers:` and `Resolves:` trailers and an
   `Assisted-by:` naming the model the lane's receipts reported, the
   repository's hooks running: a lane whose receipt reports no model is
   refused, and a hook that refuses the commit stops the landing until what it
   names is settled. It pushes the
   branch only once the repository's preflight receipt names the lane's head:
   when `step` refuses for want of one, run `make preflight` in the lane's
   worktree, then `step` again; never push, skip a hook or mint a receipt by
   hand. It opens the pull request through `gh`, with a body built from the
   run's records and passed through the outbound scrub, ending with
   `Assisted-by: abcd:<version>` and then one `Assisted-by:` per model the
   lane's receipts reported, re-reads the body the forge holds and strips a
   session URL or tool footer. It arms auto-merge with
   the merge-queue method the ruleset mirror (`.abcd/work/rulesets/`) names
   only where that mirror also requires a person's approval (an approving
   review count of one or more, or a code-owner review with a CODEOWNERS file
   naming an owner). Otherwise it leaves the pull request open for a person to
   merge, says so in the step's note and the run record ("left open for a
   person to merge: the ruleset requires no approval"), and never arms it on a
   later step; a missing mirror requires nothing. It pushes nothing to the lane
   afterwards. Then `step` exits 3 until the
   pushed head is an ancestor of the default branch on `origin`; stop driving
   the run and come back later. Once it is, the loop removes the lane's
   worktree and branch, the lane is done, and the next pending step opens the
   next lane. A pull request closed without merging, or merged in a way that
   rewrote its head, is refused and nothing is cleaned up: report it as it is.

When the run is complete, read its record and capture its transcripts:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" implement record --transcript <path> [--transcript <path>]... --json
```

The record names every lane, the receipts with the model each runner
reported, every verdict the loop recorded, the captures fixed, the pull
requests and what each landing did, and the transcripts captured into the
history store. Name the transcript of this session and of every agent it
started; each is captured as `history capture <path>` captures it, one capture
per path.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
