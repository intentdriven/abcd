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
- The ceiling turned reviews into a queue: 40 minutes of a two-hour window with a lane waiting on the two-agent ceiling, 27 minutes for one review (iss-2609211105014235). That record proposed reviewers counted separately from implementers, or a ceiling set from the lane shape; its resolution ruled instead that reviewers take the same slots as implementers, and piece 6 ("The count") holds that ruling. What this spec takes from the record is the queue, not the separate count: the validators of one round run side by side, and an open lane's reviewers take a freed slot before any new lane (piece 6, "The order in which waiting work takes a freed slot", and C5).

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

### What this piece is built on

- **Piece 9 of `spc-2609202134338445`, the landing** (the pull request armed
  through the forge client, the repository's merge rule, the ancestor check,
  the cleanup), is a prerequisite. At the base of this amendment
  (`c5b4305f6`) it is not built: `internal/core/implement/loop` holds no body
  for the land stage. Every rule of "Two lanes that touch the same files"
  below, and criteria C8, C9 and C10, presuppose an armed pull request, so
  they are built after piece 9 lands. C1 to C7, C11 and C12 do not wait on it.

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
- **Open: DR6b.** What a step without the line needs is the product
  thinker's to rule, and the build of this section waits on it. The
  question, as the spec review put it: "When a spec lists steps and says
  nothing about what depends on what, should the build run the steps one
  after another (today's rule; a step must say `- needs: none` to run beside
  earlier ones) or run them side by side up to the limit (a step must say
  `- needs: 1` to wait for step 1)?" Options: "(a) serial default, opt in
  per step (the amendment; safest, never changes an existing spec's
  behaviour); (b) parallel default, opt out per step (DR6 as the new rule,
  itd-2609212103565953 AC2 rewritten); (c) serial unless the steps'
  `- packages:` lines are disjoint (no new line, but the loop guesses)."
  The draft this spec carries until the ruling is option (a): a step without
  the line needs every step before it, the order `itd-2609212103565953`
  rules (criterion 2: the next step's lane starts only after the previous
  step has merged), so every stepped spec written so far keeps it unchanged.
  C6's first case is written against option (a) and changes with the ruling.
- A lane opens for a step only when every step it needs has landed: its pull
  request is an ancestor of the default branch, or the spec at the default
  branch marks it `landed:`. Its branch is then cut from the default branch,
  so it holds what it needs.
- The parser refuses a `needs` that names the step itself, a later step, or
  a step the spec does not list, naming the line, and the readiness gate
  reports the refusal before a run starts.
- **A `needs` line across a remainder.** `spec close --remainder` carries the
  steps not marked landed, in document order, and renumbers them from one
  (`spec.Unlanded`, `spec.RenderSteps`, and the renumbering the close does
  before it mints; a step's number is its position, `steps.go`'s `Step`).
  Every other indented line is carried verbatim, so a `- needs: 1, 3` copied
  as written would name the wrong step, or one the remainder does not list,
  and the parser would refuse the remainder the close had just written. The
  copy therefore rewrites the `needs` line, the one indented line it does
  not carry verbatim:
  - a named step that landed is satisfied and leaves the list;
  - a named step that did not land is carried, and is renamed to its number
    in the remainder;
  - a list left empty is written `- needs: none`, never removed, because an
    absent line means the default (Open: DR6b), not "nothing";
  - a step without the line stays without it: the earlier steps the
    remainder lists are exactly the unlanded earlier steps, and the landed
    ones are satisfied, so the default reads the same over the remainder
    whichever way DR6b is ruled.

  The close's result names each `needs` line it rewrote, before and after.
  The rewrite is chosen over dropping the line because it is a total
  function of the parse the close already holds, with nothing guessed: a
  step is in the remainder exactly when it has no `landed:` (the predicate
  `Unlanded` cuts on), so every step a `needs` line names is either landed
  (satisfied) or carried, and the carried steps keep their order, so the map
  from old number to new is one-to-one. Dropping the line would turn an
  explicit list into the default, which runs the step later than declared
  under DR6b option (a) and earlier than its needs allow under option (b),
  so what a drop means would hang on the open ruling. Rewriting a satisfied
  need to its `landed:` marker instead would give the parser a second
  vocabulary for a need that constrains nothing. The steps spec
  (`spc-2609212138246060`, scope 4) names this exception to its verbatim
  copy.
- The validators of one round always run side by side, stepped spec or not:
  parallel reviewers need no declaration.

### The state file

The state goes to schema version 7:

- `lanes[].awaits` is a list of awaits, each with its role, brief, receipt
  path and `since`. A lane has one entry while its implementer works and one
  per validator while its round is out. It is a new key, not the version-6
  `awaiting` with its type changed: the state is decoded strictly (a field
  the version does not name is refused), and the version is peeked first to
  pick the shape the decode holds the file to, so a key that read an object
  in one version and a list in the next would be a second, silent branch
  inside one key. Version 7 does not write `awaiting`.
- `waiting` is a list of the work the ceiling holds back, each with its lane,
  its role and `since`, the time the ceiling first held it. It is written when
  a step finds the ceiling reached, and an item leaves it when it takes a
  slot, with a run-record entry naming the lane, the role and the whole
  minutes it waited.
- `lanes[].syncs` records each sync (below): the sibling lanes whose landing
  caused it, the default branch's sha merged in, whether it conflicted, and
  the head it produced.
- A version-6 state file reads as a run whose lanes each have zero or one
  await (its `awaiting` object becomes a one-entry `awaits`), and runs on
  unchanged; the writer writes version 7. A version-7 file is refused by an
  abcd that knows only version 6, naming the version, as every schema step
  already is.
- A file that claims a version older than what it carries is refused, in the
  shape of every earlier step's refusal (`internal/core/implement/loop/state.go:476-483`
  at `c5b4305f6`: a pace in a version-1 file, a pick in a version-1 or
  version-2 file, and below them a validation and a fix-round cap): a file
  of version 6 or lower that carries `awaits`, `waiting` or `syncs` is
  refused naming its version and what it carries that the version never
  wrote, with the remedy those refusals give (the loop is the file's only
  writer; restore it or remove the run directory). A version-7 file that
  carries `awaiting` is refused the same way, since version 7 never writes
  it.
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
  landed. A lane is the closing lane when it reaches its landing with no step
  pending, no other lane open, and no lane of the run handed back. If its
  last passing round had no auditor, the sync's fresh round carries one.
- After a hand-back, no lane closes the spec. A handed-back lane is neither
  done nor open, and the delivery lacks its step, so no auditor judges the
  delivery as whole and no lane runs the close; the spec stays open for the
  person's replan. What the siblings open at the hand-back do meanwhile is
  Open: DR6c (below).
- The intent-auditor reads the whole delivery (ruling AI) as the run's own
  lanes' changes, never as a sha range from the base of the run's first lane
  to the closing head: after a sync, that range also holds every commit the
  default branch gained since the base, work from outside the run included.
  The delivery is the diff of each of the run's pull requests, lane by lane:
  a lane's head against the default-branch sha it last merged in (the last
  entry of its `syncs`), or against its base when it never synced. Each of
  those shas is an ancestor of the lane's head, so each diff holds that
  lane's commits and its conflict resolutions and nothing a sibling or an
  outside change brought in; a landed lane's head stays reachable after its
  worktree is cleaned up, because piece 9's ancestor check holds it in the
  default branch. For a run of one lane that never synced, the delivery is
  the base-to-head range the implement verb's piece 8 reads.
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
- **Open: DR6c.** What a hand-back does to the lanes already open beside it
  is the product thinker's to rule, and the build of this bullet waits on it.
  The spec review's finding: the draft below "rewrites built behaviour"
  (`loop.go:556-557` refuses every step once `current()` is handed back, and
  `handback.go` promises "every later step refuses", DR1's shape, itd-50
  criterion 2); "The run then stops" is "ambiguous (at once, or once the last
  sibling lands?)", what `implement step` answers in between is not said, and
  "the intent goes back to the person to replan while siblings land pull
  requests from the plan being replanned". Options put to the product
  thinker: (a) stop all lanes at the hand-back; (b) siblings finish and
  merge, then the run stops; (c) siblings finish, and their merges are held.
  The draft this spec carries until the ruling: a lane handed back stops only
  itself; siblings already open run on to their landing or their own
  hand-back; no new lane opens after a hand-back, because the run's intent is
  the person's to replan; the run then stops and reports the hand-back.
  Whichever option is ruled, no new lane opens after a hand-back and no lane
  closes the spec (above).

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
  run and the clock moved on 14 minutes from the `implement step` call that
  first found the ceiling reached (the `since` C2 wrote, not the moment the
  work became ready), **when** one reviewer's receipt is verified and
  `implement step` is called, **then** the held work takes the freed slot
  and the run record carries an entry naming its lane, its role and 14
  minutes waited.
- **C4, implementers and reviewers together.** **Given** `--sub-agents 3`,
  lane 1 with both reviewers out and step 2 marked `- needs: none`, **when**
  `implement step` is called until lane 2's implementer is handed its brief
  (the worktree, the brief and the implementer are one move each), **then**
  lane 2 opens in its own worktree and its implementer takes the third slot;
  the next call finds the ceiling reached. A stage the binary performs itself proceeds at the ceiling.
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
  line. (The first case is written against DR6b option (a); Open: DR6b.)
  **Given** a spec of four steps where steps 1 and 3 are marked `landed:`,
  step 4 carries `- needs: 1, 2` and step 2 carries `- needs: 1`, **when**
  `spec close --remainder` runs, **then** the remainder lists old step 2 as
  step 1 with `- needs: none` and old step 4 as step 2 with `- needs: 1`,
  the close names both rewritten lines, and the remainder parses.
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
  its landing, and no new lane opens (the sibling's part is written against
  the draft; Open: DR6c); whatever DR6c rules, no lane closes the spec.
- **C12, the schema.** **Given** a version-6 state file with one lane
  awaiting its implementer, **when** it is read, **then** the lane has one
  await and the run advances on it; the next write is version 7 and carries
  `awaits`, never `awaiting`. **Given** a version-6 file carrying `awaits`,
  `waiting` or `syncs`, or a version-7 file carrying `awaiting`, **when** it
  is read, **then** it is refused naming the version and the key.

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
- `internal/core/implement/loop/loop.go:556-557` and
  `internal/core/implement/loop/handback.go:3-8`: once `current()` is handed
  back, every later step refuses, so the whole run stops at a hand-back
  (DR1's shape, itd-50 criterion 2). What replaces it waits on Open: DR6c.
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
  internal/core/intent (the remainder's `needs` rewrite),
  internal/surface/cli, internal/core/statusblock, internal/core/site
- a shape change, not only new content: `statusblock.Started` carries one
  `Lane` per run (`loop.go:948-957` at `c5b4305f6` fills it from `current()`
  alone), and the block reports one row per lane alive, so `Started` carries
  every lane alive and each reader of it changes with it, the site's status
  page (`internal/core/site/status.go`) included
- prerequisite: piece 9 of `spc-2609202134338445`, the landing, before C8 to
  C10 and the landing rules ("What this piece is built on")
- tests: C1 to C12 through the step interface with fake agents and a fake
  forge; the `needs` parser over a stepped spec; the remainder's `needs`
  rewrite; a version-6 state file read and advanced, and the version refusals
