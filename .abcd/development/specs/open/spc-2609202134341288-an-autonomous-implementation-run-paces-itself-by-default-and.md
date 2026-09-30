---
id: spc-2609202134341288
slug: an-autonomous-implementation-run-paces-itself-by-default-and
intent: itd-2609201925079472
origin: researcher-authored
production_mode: hand-written
---
# an-autonomous-implementation-run-paces-itself-by-default-and

## Summary

The design record for itd-2609201925079472, from the six decisions on the
intent (2026-09-20).

## Scope

1. **The pace configuration**: `pace.work_minutes`, `pace.pause_minutes`,
   `pace.sub_agents` read through the layering the model-tier intent uses
   (flag, repository `.abcd/config.json`, machine `~/.abcd/config.json`,
   bundled 120/300/2); one resolver reports the value and the layer it
   came from (criteria 1 to 3, 9).
2. **The window clock** in the implement verb's state file: the window's
   start, `next_eligible_at`, the lanes alive; written by the loop at every
   step (criteria 4, 5).
3. **The ceiling**: the loop counts the agents alive from the state
   (implementers and validators together) and starts nothing above the
   ceiling, exiting with the lanes named and the wait accumulated
   (criterion 6). The ceiling is reachable because a run works in parallel
   up to it (piece 6).
4. **The budget check** (from `itd-29`): where the runner reports remaining
   quota, an estimate from the spec's size is compared before the run
   starts; a runner that reports none is named and the check is skipped
   out loud (criterion 7).
5. **The rate-limit checkpoint** (from `itd-29`): a runner's rate-limit
   response ends the window early with the lane checkpointed to its branch
   and `next_eligible_at` set (criterion 8).
6. **Concurrent lanes and validators** (ruling DR6, 2026-09-29): one run
   hands work to several agents at once, up to its ceiling: the validators
   of one round together, and the lanes of steps that do not need each
   other side by side. The section "Concurrent lanes and validators" below
   is the design; its criteria C1 to C12 are how criterion 6 is tested.

## Out of scope

A ceiling across runs (the register's), which includes the further picks
`abcd build next --max` and `--until-empty` make: each pick is its own run,
and picks stay one after another. Telemetry and hand verbs. A time limit on
an agent that never hands back its receipt: its slot stays taken until the
receipt comes back or the person stops the run.

## Approach

One lane, test-first, on the implement verb's state file; the resolver is
the one canonical layering primitive shared with the model tier. The bundled
default lives in one constant the run record names.

## How the criteria are satisfied

1 to 3 and 9 by piece 1; 4 and 5 by piece 2; 6 by pieces 3 and 6, tested
through C1 to C12 below; 7 by piece 4; 8 by piece 5.

## Evidence the build must answer

- The pause has never fired. Three Dessau pilots on 2026-09-21 ran the scripted outer loop under a pacing window, and each finished inside its first window, so the gate that refuses an early start and the resume on `next_eligible_at` are untested by a real pause; the abcd pilot of 2026-09-20/21 paced by hand (idle wake-ups from the orchestrator's own scheduler) and broke its second pause on the facilitator's word. The build proves the pause with a test that sets the clock past the window and watches the refusal, and the first real run records whether the pause fired.
- The ceiling turned reviews into a queue: 40 minutes of a two-hour window with a lane waiting on the two-agent ceiling, 27 minutes for one review (iss-2609211105014235). Reviewers counted separately from implementers, or a ceiling set from the lane shape, is a criterion this spec takes from that record.

## Concurrent lanes and validators (ruling DR6, 2026-09-29)

The product thinker ruled, verbatim: "per-run agent limit: WORK IN PARALLEL
— a run may build several pieces and run reviewers concurrently up to its
limit (new build-loop work; then AC6 is testable)." Without this piece the
loop hands out one agent at a time, so a run never has more than one agent
alive and criterion 6's ceiling, which is at least 1, can never be reached.
This section is the design that makes it reachable. The pieces it
changes are the implement verb's (`itd-2609201916151817`: the state file, the
step interface, the validators and the landing) and the step parser of
`itd-2609212103565953`; they are specified here because the ceiling is this
spec's and nothing else needs them.

### The count

- A **slot** is one agent the run has handed work to and not yet taken a
  verified receipt from: an outstanding await on any lane of the run. The
  count is the number of outstanding awaits in the state file, and the state
  file is its only source; nothing else is counted and nothing is stored
  beside it that could disagree.
- Implementers and validators count alike, in any mix of roles: the first
  implementer of a lane, the fresh implementer of a fix round, the fresh
  implementer of a sync (below), the ruthless reviewer, the security
  reviewer and the intent-auditor. That reviewers take the same slots as
  implementers is already settled (`iss-2609211105014235`'s resolution and
  the decision log of 2026-09-23, 2026-09-24 and 2026-09-29) and is not
  reopened here.
- The ceiling is the run's `pace.sub_agents`, with the layer that supplied
  it, as piece 1 resolves it. The session that drives the loop is not a slot:
  it is handed no brief.
- A stage the binary performs itself takes no slot and is never held by the
  ceiling: the worktree, the brief, the landing's `gh` calls and the merge of
  a sync.

### Isolation

- Every lane has its own worktree in the machine-scoped store,
  `~/.abcd/worktrees/<root-sha>/<run-id>-<lane-id>`, on its own branch
  `build/<run-id>-<lane-id>`, exactly as the implement verb's piece 6 lays one
  out. Two lanes never share a worktree or a branch.
- The validators of one round read the lane's worktree at the head the round
  names, at the same time, and treat it as read-only: whatever a validator
  runs, it runs on a copy (`git archive <head>`), as `AGENTS.md` asks of a
  verifier. No implementer is handed the lane while its validators are out:
  a fix brief is written only once every validator of the round has returned.
- Every `implement step` and `implement receipt` holds the run tier's
  advisory lock for its read and its write of the state file, as it already
  does, so receipts that several agents hand back at once are applied one
  after another and none is lost.

### How a slot is freed

- A slot is freed when its receipt is verified: `implement receipt <path>`
  looks the path up among every outstanding await of the run, not only the
  first lane's, and advances the lane that await belongs to. A path no
  outstanding await names is refused, naming the awaits there are, and frees
  nothing.
- A receipt the verifier refuses frees nothing: the agent still owes it, and
  its slot stays taken.
- A lane handed back holds no slot, because a hand-back is decided only once
  every validator of its round has returned.
- A freed slot is filled by the next `implement step`, never by the receipt
  call itself: the receipt call verifies and advances one lane and exits, as
  every invocation does.

### The order in which waiting work takes a freed slot

Each `implement step` performs one move. It first performs any stage of any
lane the binary owns (those take no slot). When the move needs an agent, it
takes the first waiting item in this order:

1. Work on a lane already open, before any new lane: the round's validators
   and the fix or sync implementers. A lane already open is closer to
   landing, and its reviewers are what queued for 40 minutes in the record
   `iss-2609211105014235` carries.
2. Among open lanes, the lane of the lower-numbered spec step first.
3. Within one lane's round, the validators in the order the round lists them:
   the ruthless reviewer, the security reviewer, then the intent-auditor on
   the lane that closes the spec.
4. Then the first implementer of a new lane, for the lowest-numbered pending
   step whose needs have landed (below).

The order is a function of the state alone, so two runs of the same state
hand out the same work.

### Which steps may run side by side

- A spec step may carry a `- needs:` line beside its `- packages:` and
  `- tests:` lines: `- needs: none`, or `- needs: 1, 3`, naming earlier steps
  by number.
- A step without the line needs every step before it. That is the order
  `itd-2609212103565953` rules (criterion 2: the next step's lane starts only
  after the previous step has merged), and every stepped spec written so far
  keeps it unchanged.
- A lane opens for a step only when every step it needs has landed: its pull
  request is an ancestor of the default branch, or the spec at the default
  branch marks it `landed:`. Its branch is then cut from the default branch,
  so it holds what it needs.
- The parser refuses a `needs` that names the step itself, a later step, or
  a step the spec does not list, naming the line, and the readiness gate
  reports the refusal before a run starts.
- The validators of one round always run side by side, stepped spec or not:
  parallel reviewers need no declaration.

### The state file

The state goes to schema version 7:

- `lanes[].awaiting` is a list of awaits, each with its role, brief, receipt
  path and `since`. A lane has one entry while its implementer works and one
  per validator while its round is out.
- `waiting` is a list of the work the ceiling holds back, each with its lane,
  its role and `since`, the time the ceiling first held it. It is written when
  a step finds the ceiling reached, and an item leaves it when it takes a
  slot, with a run-record entry naming the lane, the role and the whole
  minutes it waited.
- `lanes[].syncs` records each sync (below): the sibling lanes whose landing
  caused it, the default branch's sha merged in, whether it conflicted, and
  the head it produced.
- A version-6 state file reads as a run whose lanes each have zero or one
  await, and runs on unchanged; the writer writes version 7. A version-7 file
  is refused by an abcd that knows only version 6, naming the version, as
  every schema step already is.
- `implement status` names the slots in use out of the ceiling, and every
  lane alive with its stage and each role it awaits. The status block reports
  one row per lane alive, not only the first.

### Two lanes that touch the same files

- Landing is one lane at a time within a run: at most one lane holds an armed
  pull request that has not merged. A lane whose round passes while a
  sibling's pull request is armed waits at its landing, holding no slot; when
  several wait, the lane of the lower-numbered spec step arms first.
- Before a lane arms its pull request, the loop checks whether a sibling lane
  of the same run has landed since this lane's base. If one has, the loop
  **syncs** the lane: it merges the default branch into the lane's branch
  with a merge commit in the lane's worktree. It never rebases, so no commit a
  validator judged is rewritten and every sha the record cites stays
  reachable.
- A clean merge moves the lane's head, so a fresh round judges the new head:
  no verdict stands over a head it did not read (the invariant of the
  implement verb's piece 8).
- A merge that conflicts is aborted, leaving the branch where it was, and the
  conflict goes to a fresh implementer with a sync brief naming each
  conflicting path and the sibling lane whose landing brought the other side.
  The receipt is verified as a fix receipt is, and must also carry the
  default branch's merged sha as an ancestor of the new head; then a fresh
  round judges it.
- A sync does not count against `--fix-rounds`. Each sync answers a sibling
  landing, and a run has a fixed number of lanes, so syncs are bounded by the
  lane count. A round after a sync that does not pass is a fix round like any
  other and counts.
- The lane that closes the spec is synced before it arms whatever it
  touches (a merge that finds the branch already holds the default branch
  changes nothing and needs no round), so its head holds every lane the run
  landed, and the
  intent-auditor reads the whole delivery from the base of the run's first
  lane to that head (ruling AI). A lane is the closing lane when it reaches
  its landing with no step pending and no other lane open. If its last
  passing round had no auditor, the sync's fresh round carries one.
- Movement of the default branch by anyone outside the run is left to the
  repository's merge rule, as it is for a single lane: the loop syncs only on
  its own siblings' landings.

### The pace and the fix rounds, per lane

- The window clock is the run's, one for every lane. When the window
  elapses, `implement step` starts no agent on any lane; every outstanding
  await may still hand back its receipt, and those receipts are verified and
  free their slots. `next_eligible_at` is written once for the run.
- A rate-limit response on any lane's agent ends the window for the whole
  run, because every lane spends the same budget; every lane with work in
  flight is checkpointed to its own branch, and the record names the lane the
  response came from.
- `--fix-rounds` is a per-run value that each lane spends on its own: a lane
  counts its own fix rounds against the run's cap, and a sibling's rounds
  never count against it.
- A lane handed back stops only itself. Siblings already open run on to their
  landing or their own hand-back. No new lane opens after a hand-back,
  because the run's intent is the person's to replan, and the run then stops
  and reports the hand-back.

### Criteria (Given, When, Then)

Each is a test through the step interface, with fake agents writing the
receipts and returns, on the clock `Options` already carries; the host-playing
end-to-end test of the implement verb is the pattern. C2 and C3 are criterion
6 made concrete.

- **C1, the count.** **Given** a run with `--sub-agents 2` whose lane reaches
  its validate stage, **when** `implement step` is called twice, **then** the
  ruthless and security reviewers both await at once, the state file carries
  two awaits on that lane, and `implement status` names 2 of 2 slots in use,
  each with its lane and role.
- **C2, the ceiling reached (criterion 6).** **Given** C1's run with both
  reviewers out, **when** `implement step` is called again, **then** it hands
  out no brief, exits 0 naming every lane alive with the role and receipt path
  it awaits, and writes the held work into `waiting` with the time it was
  first held; a second call before any receipt leaves that time unchanged.
- **C3, the slot filled and the wait counted (criterion 6).** **Given** C2's
  run and the clock moved on 14 minutes, **when** one reviewer's receipt is
  verified and `implement step` is called, **then** the held work takes the
  freed slot and the run record carries an entry naming its lane, its role
  and 14 minutes waited.
- **C4, implementers and reviewers together.** **Given** `--sub-agents 3`,
  lane 1 with both reviewers out and step 2 marked `- needs: none`, **when**
  `implement step` is called, **then** lane 2 opens in its own worktree and
  its implementer takes the third slot; the next call finds the ceiling
  reached. A stage the binary performs itself proceeds at the ceiling.
- **C5, the order.** **Given** a full run where lane 2's security reviewer,
  lane 1's fix implementer and a new lane for step 4 all wait, **when** one
  slot frees and `implement step` is called, **then** lane 1's fix implementer
  takes it; at the next freed slot lane 2's reviewer; step 4's lane opens
  last.
- **C6, needs.** **Given** a spec whose step 2 has no `- needs:` line, **when**
  lane 1 is open, **then** no lane opens for step 2 until lane 1's pull request
  is an ancestor of the default branch, even with a slot free. **Given**
  `- needs: none` on step 2, **then** its lane opens at the next free slot.
  **Given** `- needs: 3` on step 2, **then** the parser refuses naming the
  line.
- **C7, isolation and the receipt.** **Given** lanes 1 and 2 each awaiting an
  implementer, **when** lane 2's receipt is handed back, **then** lane 2
  advances and lane 1 is unchanged; the two worktrees and branches differ; a
  receipt path no await names is refused and the count is unchanged.
- **C8, a clean sync.** **Given** lane 1 has landed a change to one file and
  lane 2, branched before it, changed another file, **when** lane 2 reaches its
  landing, **then** the loop merges the default branch into lane 2 with a merge
  commit before arming, a fresh round judges the merge head, and no fix round
  is counted.
- **C9, a conflicting sync.** **Given** lane 1 has landed a change to a file
  lane 2 also changed, **when** lane 2 reaches its landing, **then** the merge
  is aborted with the branch unchanged, a fresh implementer is handed a sync
  brief naming the file and lane 1, a receipt whose head does not contain the
  merged sha is refused, and a verified one opens a fresh round; no fix round
  is counted.
- **C10, one landing at a time.** **Given** lanes 1 and 2 both waiting at
  their landing with passing rounds, **when** `implement step` is called,
  **then** lane 1 arms its pull request first, and lane 2 arms only after lane
  1's pull request has merged and lane 2 has synced and passed a fresh round.
- **C11, the pace across lanes.** **Given** two lanes each with an agent out
  and the window elapsed, **when** `implement step` is called, **then** it
  starts nothing on either lane, both receipts are still verified, and
  `next_eligible_at` is written once. **Given** `--fix-rounds 1` and lane 1
  failing its second round, **then** lane 1 is handed back, lane 2 runs on to
  its landing, and no new lane opens.
- **C12, the schema.** **Given** a version-6 state file with one lane
  awaiting its implementer, **when** it is read, **then** the lane has one
  await and the run advances on it; the next write is version 7.

### Where the loop assumes one lane at a time

At the base of this amendment (`c5b4305f6`), each of these holds the single
lane or the single agent in place, and the build replaces or reads past it:

- `internal/core/implement/loop/state.go:214`: "Lanes are the lanes opened so
  far, one at a time, in order."
- `internal/core/implement/loop/state.go:242-244`: one `Awaiting` per lane.
- `internal/core/implement/loop/state.go:410-420`: `current()` is the first
  lane not done, the only lane any call acts on.
- `internal/core/implement/loop/loop.go:550` and `:574`: `Advance` works on
  `current()` alone and returns while that lane awaits, so nothing else
  starts.
- `internal/core/implement/loop/loop.go:619` and `:726`: the next lane opens
  only when the current one is done, and `loop.go:510` opens it from
  `Pending[0]`.
- `internal/core/implement/loop/loop.go:683`: `Receipt` looks for the await on
  `current()` only.
- `internal/core/implement/loop/loop.go:493`, `:743` and `:947`: the start
  result, the next move and the status block each report one lane.
- `internal/core/implement/loop/validate.go:3-5` and `:145-154`: the round
  hands out "one fresh agent at a time", returning at the first validator
  without a verdict.
- `spc-2609202134338445`, `## Progress`, the entry for lane fidelityOnce
  (piece 8): the validators run "one at a time".
- `itd-2609212103565953`, `## What's In Scope` ("The loop") and criterion 2
  of its `## Acceptance Criteria`, and `spc-2609212138246060`, `## Scope`
  item 3: the next step's lane starts after the previous has merged. This holds
  unchanged for a step with no `- needs:` line; a step marked otherwise is the
  exception DR6 admits, and that intent's wording is amended when the needs
  line is built.

### Footprint of this piece

- packages: internal/core/implement/loop, internal/core/spec,
  internal/surface/cli, internal/core/statusblock
- tests: C1 to C12 through the step interface with fake agents and a fake
  forge; the `needs` parser over a stepped spec; a version-6 state file read
  and advanced
