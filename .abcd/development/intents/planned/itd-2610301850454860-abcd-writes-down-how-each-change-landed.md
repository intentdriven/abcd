---
id: itd-2610301850454860
slug: abcd-writes-down-how-each-change-landed
spec_id: spc-2610301950480609
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-82, itd-2610150602547271]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# abcd records how each change landed

## Press Release

> abcd writes down how each change landed. When a change reaches the default branch, abcd records with it whether a person approved it, the drain landed it under its rule, or it merged automatically after its checks passed, and which rule let it through. Anyone can then ask which changes landed with no person involved, and a change that is later undone can be traced to the rule that admitted it.

## Why This Matters

Typed links: `builds_on`
[itd-82](itd-82-drain-ledger-triage.md) (the drain, whose lane
this record names by the rule record it took the issue under) and
[itd-2610150602547271](../shipped/itd-2610150602547271-a-repository-declares-whether-its.md)
(the facilitator mode, shipped: In `auto` a maintainer's pull request is armed
only when a review receipt names its head, in `human` only once an approval of
the head counts, and this record says which of the two admitted each merge).
Blocks
[itd-2610301750162657](itd-2610301750162657-a-person-spot-checks-what-abcd-s-agents.md)
(the spot-check, whose collection lane draws from this listing at each landing
once a window of time or of landed changes has passed, and whose review lane
reads an item's state when it was drawn and its state now).

Split out of itd-2610301750162657 on 2026-10-30 at the product thinker's
ruling in its planning interview: The spot-check of what abcd decided alone
waits for this record, and the record is useful on its own.

Today abcd cannot tell, after the fact, how a change reached the default
branch. Which issues a drain lane landed lives only in the drain's run state,
an uncommitted file in one worktree's local tier
(`internal/core/implement/loop/drain.go`); every merge commit carries the same
author whether a person, a lane, or the merge queue landed it; the facilitator
mode's two admitting facts, the technical facilitator's recorded approval and
the review receipt that arms a merge in `auto`, are files in the local tier
named by the head they vouch for (`internal/core/landing/facilitator.go`,
`internal/core/landing/review.go`); and whether a person approved a pull
request on the forge is known only to the forge. A change later reverted
cannot be traced to the rule that let it through, and nobody can list the
changes that landed with no person involved.

## Mechanism

- We expect every change that lands after this ships to be classifiable from
  history and the forge with no one's memory, because the lane's rule rides a
  trailer on a commit the merge carries into the default branch's history, and
  the facilitator mode sits in the committed configuration at the commit the
  change merged onto. Shown wrong if `abcd landings`, run over the landings
  since this shipped, lists a lane abcd opened whose rule reads `unknown`, or a
  reverted row whose revert cannot be traced to its landing's rule.

## Scope Conditions

- A repository whose every change reaches the default branch through a pull <!-- cond: cond-2610301950483000 -->
  request the forge merges (a merge queue or auto-merge) or through a
  person's direct push; abcd never pushes the default branch.
- The forge is GitHub, read through the `gh` client the landing already uses. <!-- cond: cond-2610301950485184 -->
- The default branch's merges keep the branch's commits as the merge's second <!-- cond: cond-2610301950486745 -->
  parent (the `MERGE` method, as abcd's own queue ruleset sets it); a squash or
  rebase merge keeps them only on the forge.
- The implement loop commits the lane's records on the lane's branch before <!-- cond: cond-2610301950486793 -->
  the lane's preflight runs, and the push is refused until a preflight receipt
  names the lane's head.
- The facilitator mode (itd-2610150602547271) decides the arming: `auto` by a <!-- cond: cond-2610301950487033 -->
  review receipt naming the head, `human` by a counted approval of the head;
  both are files in the local tier named by the head, written before the
  merge and read by any worktree of the checkout.
- Scale: abcd's own repository, about 500 merges on the default branch in the <!-- cond: cond-2610301950485439 -->
  30 days to 2026-10-30; the listing asks the forge once per pull request.

## What's Out of Scope

- **Freezing a cycle's rows into the repository at `launch ship`.** The
  consumer draws per window, not per release; the release commit is the wrong
  boundary (the tag is made after the ship pull request merges, so a file in
  the release commit cannot name the cycle's last landing); and it would add a
  refusal to the cut. If a trace is ever wanted after the forge no longer
  holds the repository, it is a later issue against this record.
- The drain's hand-backs and the automated facilitator's own decisions: They
  stay in the drain's run state and in their own records, and the spot-check
  reads those groups from there.

## Acceptance Criteria

Written by the technical facilitator under the delegation of
dcn-2610301913469955 (the acceptance criteria of an intent the product
thinker asked to plan); each is testable as stated.

- **Given** a drain lane whose issue the drain took under the repository's
  drain rule record, **when** the loop makes the lane's records commit,
  **then** that commit carries `Admitted-by: drain <rule record id>` beside
  its `Resolves:` and `Assisted-by:` trailers, and the attribution gate and
  the commit-msg hook accept it.
- **Given** a build lane for a spec, **when** the loop makes the lane's
  records commit, **then** it carries `Admitted-by: build <spec id>`; a lane
  started by `abcd build <iss-N>` carries `Admitted-by: build <iss-N>`.
- **Given** a lane whose landing marks no step, closes no spec, and resolves
  no capture, **when** the loop reaches its records step, **then** it still
  makes the records commit, changing no file, carrying the `Admitted-by:`
  trailer, and the lane's pushed head is that commit.
- **Given** a run started by the drain or by `abcd build`, **when** its state
  is written at the start, **then** it names the admitting rule (`admitted`:
  the lane kind and the record id, with the drain rule record's path for a
  drain's run); **given** a run state without it, **when** the records step
  runs, **then** it is refused naming the field, and no commit is made.
- **Given** a person's branch landed through `abcd land`, **when** land
  gates, pushes, and arms it, **then** no commit has been added to the branch
  and no trailer written: The preflight receipt, the review receipt, and the
  recorded approval all name the head the person made.
- **Given** the default branch with merges since the last release tag,
  **when** `abcd landings` runs with the forge reachable, **then** it prints
  one row per commit on the default branch's first-parent line, newest first,
  each with the commit, the pull request number where the message names one,
  the class (`person-approved`, `drain`, `automatic`, or `person-push`), the
  rule that admitted it, the lane kind, the approval it saw, and the issues
  and intents the branch's `Resolves:` and `Delivers:` trailers name.
- **Given** a merged pull request whose merged head a forge review by someone
  other than the author approves, not dismissed, **when** its row is classed,
  **then** it is `person-approved`, names the review and its time, and shows
  the lane kind whatever it is; the approver's login is read from the forge
  and printed, and no committed file carries it.
- **Given** a merged pull request whose merged head a recorded approval names
  in a worktree of the listing checkout, **when** its row is classed, **then**
  it is `person-approved` by that record, and the row says the approval was
  read from this checkout's local tier.
- **Given** a merged pull request with no counted approval seen whose first
  parent's committed `.abcd/config.json` sets `facilitator.mode: human`,
  **when** its row is classed, **then** it is `person-approved` by the mode's
  rule, with its approval shown as `none seen`.
- **Given** a merged pull request under `auto` with no counted approval seen
  whose branch carries `Admitted-by: drain <id>`, **when** its row is
  classed, **then** it is `drain` and its rule is that record, by id and path.
- **Given** a merged pull request under `auto` with no counted approval seen
  and no drain trailer, **when** its row is classed, **then** it is
  `automatic`, and its rule is `facilitator.mode: auto` as `.abcd/config.json`
  held it at the commit the change merged onto.
- **Given** a commit on the default branch's first-parent line that is no
  merge and whose message names no pull request, **when** its row is classed,
  **then** it is `person-push`, with no rule and no lane kind, and its
  committer is named from git.
- **Given** a pull request the forge merged by squash or rebase, **when** its
  row is composed, **then** the branch's commits and their trailers are read
  from the forge's listing of the pull request's commits, and with the forge
  unreachable the lane kind reads `unknown` and the row counts as unattended.
- **Given** `abcd landings --unattended`, **when** it runs, **then** it lists
  only the rows classed `drain` or `automatic`, and a row the evidence cannot
  place with a person is among them, never left out.
- **Given** a commit reachable from the default branch since the start of the
  listed range, on the first-parent line or on a merged branch, whose message
  says, in git's own words, `This reverts commit <sha>`, **when** `abcd
  landings <that sha>` or `abcd landings <the revert's sha>` runs, **then** it
  prints the row of the first-parent commit that brought the named commit
  onto the default branch, with its class and rule, or says the commit is not
  on the default branch.
- **Given** a revert the forge squashed, so that no commit says `This reverts
  commit`, **when** its pull request's body says `Reverts #N` (or `Reverts
  <owner>/<repo>#N`), **then** the row of pull request N is marked reverted
  by it; **given** neither line, **then** the revert is `untraceable`, and
  `--reverted` counts it in a group of its own.
- **Given** rows whose change a later commit on the default branch reverts,
  **when** `abcd landings --reverted` runs, **then** each is marked with the
  reverting commit and the rows are grouped by rule with a count per rule.
- **Given** the forge cannot be reached, **when** the listing runs, **then**
  the class and rule of every row still come from history, each row's
  approval reads `forge unread`, no row is classed `person-approved` by a
  forge review, and the command says why.
- **Given** `abcd landings --since <tag or date>`, **when** it runs, **then**
  the first-parent line is listed from that tag or from the first commit after
  that date, and without the flag from the last release tag.
- **Given** a landing from before this shipped, with no `Admitted-by:`
  trailer, **when** it is listed, **then** its lane kind reads `unknown`, its
  class comes from the approvals and the committed configuration alone, and
  no history is rewritten to add the trailer.
- **Given** `abcd landings --json`, **when** it runs, **then** it prints the
  same rows as JSON with stable keys, and `abcd landings` in any form writes
  nothing outside the local tier.

## Decisions

Each is the technical facilitator's, under dcn-2610301913469955, grounded in
how a change lands today (`internal/core/landing/land.go`,
`internal/core/landing/facilitator.go`,
`internal/core/implement/loop/land.go`,
`internal/core/implement/loop/pickcommit.go`,
`internal/core/implement/loop/drain.go`):

1. **The unit of record is the commit on the default branch's first-parent
   line**: The merge commit the forge makes, a squash or rebase commit, or a
   person's direct push, keyed by its sha. The merge queue makes the merge
   commit, not abcd (its committer is the forge, its message names the pull
   request), so nothing is ever written to that commit, and no row depends on
   it carrying anything but the pull request number and its parents.
2. **The durable source is written before the change merges, never after.** A
   commit added to the default branch after the merge would be a machine's,
   which the attribution gate refuses, so the admitting evidence that can be
   carried is carried on the branch, in history the merge brings in.
3. **One carrier for every loop lane: the `Admitted-by:` trailer on the
   landing's records commit, which is made on every lane.** The records
   commit is the commit the loop already makes on the lane's branch after the
   validators and before the preflight receipt and the push (`landRecords`;
   `landPush` refuses until a receipt names the lane's head), so the trailer
   costs no second gate run and rides the merge into history under the
   `MERGE` method. Today `landPrepare` skips the commit when the lane marks no
   step, closes no spec, and resolves no capture (`RecordsDone` set with
   nothing to do); that case goes: The commit is made anyway, changing no
   file, as the record-only commit it is. The trailer's values are `drain
   <rule record id>` (the record `drainrule.Load` names, whose `Record` is
   empty only for the bundled baseline, which a drain never runs under) and
   `build <spec id>`, or `build <iss-N>` for a lane a person started with
   `abcd build <iss-N>`. Chosen over the pick commit, which only a picked
   run's first lane makes (`lane.go`: The pick's entry is made when
   `State.Pick` names the lane), so a drain lane and a `build <id>` lane have
   none; over a committed file per landing, which would add a file to the
   tree for every merge; and over a trailer on the merge commit, which abcd
   cannot write.
4. **The run's state names the rule the lane was admitted under.** A new
   field on the run's `State`, `admitted`, with the lane kind (`drain` or
   `build`), the record id, and for a drain's run the rule record's path, is
   written at `Start` from what started the run: The drain passes the rule it
   loaded before opening the lane, `abcd build` the spec or issue it was
   given. `landRecords` writes the trailer from it and never guesses: A state
   without the field is refused at the records step, naming it. The drain's
   own state already keeps `Rule`, but the lane's landing reads only the
   run's state, which is why the field is there.
5. **abcd adds nothing to a person's branch.** The preflight receipt, the
   review receipt, and the recorded approval each name the head the person
   made, and a commit abcd added after them would void all three and send the
   branch back through the gate. Nor could such a commit name the approval or
   receipt that armed the merge: Both are of the head, and any commit that
   names them is a new head they do not name. So a person's branch is classed
   from history and the forge, and carries no `Admitted-by:`.
6. **What `abcd landings` reads, and from where.** History holds the
   first-parent merges, their messages (the pull request number), their
   second parents' commits and trailers (`Admitted-by:`, `Resolves:`,
   `Delivers:`, `Assisted-by:`), the committed `.abcd/config.json` at the
   first parent (the facilitator mode as the gate read it at the fetched tip,
   dcn-2610230058333177), and every `This reverts commit` line. The forge is
   needed for exactly three things history cannot hold: The pull request's
   reviews and author (a forge approval of the head by someone other than the
   author), the branch's commits of a pull request merged by squash or rebase,
   and the `Reverts #N` line of a squashed revert's pull request. Two facts
   neither holds, the recorded approval and the review receipt: Both live in
   the local tier, named by the head; the listing reads them from the
   worktrees of the checkout it runs in, as the merge gate does, and a
   checkout without them reads from history and the forge alone, so a row
   classed by a recorded approval says so. The forge unreachable: Class and
   rule still come from history, every row's approval reads `forge unread`,
   a squash or rebase merge's lane kind reads `unknown`, and the command says
   why. No committed file carries a person's login: A login appears only in a
   row read live from the forge.
7. **Four classes, decided in this order.** `person-push`: A first-parent
   commit that is no merge and names no pull request, since abcd never pushes
   the default branch. `person-approved`: A counted approval of the merged
   head is seen (a recorded approval in the listing checkout's worktrees, or
   a forge review by someone other than the author), or the committed mode at
   the first parent is `human`, under which abcd's gate arms nothing without
   a counted approval and a merge made in the forge's own page is a person's
   act; the row shows which it saw, or `none seen`. Then, under `auto`:
   `drain` when the branch carries `Admitted-by: drain`, else `automatic`,
   whose rule is `facilitator.mode: auto` at the first parent; the review
   receipt that armed it is not recoverable (decision 6), so the row names
   the mode's requirement, not the receipt. `person-approved` records an
   approval the rules keep, not a proof of who approved
   (dcn-2610300955154851): The recorded approval is a file anything under
   the person's account can write. The classes hold if the agents get a forge
   account of their own (itd-2610300955026531): The author becomes the
   agent's login, the person's forge review then counts as one by someone
   other than the author, and no class keys on whose login did what.
8. **A person's approval wins over the lane.** A drain lane a person approved
   is `person-approved`, with its lane kind still shown: The spot-check draws
   from what no person read, and an approval is a person reading. Chosen over
   ranking the drain first, which would count a read change as unread.
9. **When in doubt, unattended.** Any row under `auto` the evidence cannot
   place with a person (no approval seen, lane kind `unknown`, a machine-layer
   `human` abcd cannot see afterwards) is listed under `--unattended`. The
   listing feeds a spot-check, and the costly error is leaving an unread
   change out of the draw, not reading one too many.
10. **The listing is a read-only verb, `abcd landings`**, with `--since
    <tag|date>` (the last release tag by default), `--unattended`,
    `--reverted`, `--json`, and a positional sha to trace one commit. Chosen
    over a flag on `abcd land`, which pushes and arms and should not also be
    the thing a reader runs, and over folding it into `abcd status`, which
    lists announcements that lapse within the hour.
11. **The forge is read, never written.** The approvals come through the
    client the landing already uses, with the pull request query extended to
    the reviews it keeps. No comment, label, or commit status is posted to
    mark abcd's arming: The forge cannot tell abcd's arming from the person's,
    both made under the person's login, and the classes do not need it.
12. **A revert is traced through git's own words, over every commit in the
    range.** The trace walks every commit reachable from the default branch
    since the start of the listed range, second parents included, and matches
    `This reverts commit <sha>`, the line RS001's revert withdrawal already
    reads; the named commit is followed along the first-parent line to the
    merge that brought it in, then to that row and its rule, and a revert of a
    revert the same way. A revert the forge squashed carries no such line, so
    it is matched by `Reverts #N` in its pull request's body, the line the
    forge's own revert writes, and is otherwise `untraceable`, counted in a
    group of its own under `--reverted`, which groups the rest by rule.
13. **No backfill.** Landings before this ships carry no trailer and are
    classed from the approvals and the configuration alone, lane kind
    `unknown`; history is never rewritten.
14. **Naming the reviewers per change stays with itd-2610240922461281**,
    which records each review's findings per reviewer. A row carries the
    `Assisted-by:` values the branch's commits already name; it does not
    duplicate the review record.
15. **Impact: additive.** A new trailer, a new run-state field, and a new
    read-only verb; nothing existing users rely on changes.
16. **Ruled by the product thinker on 2026-10-30:** A person's own branch
    merged in `auto` mode with no approval seen counts as landing with no
    person involved, so `--unattended` lists it and the spot-check can draw
    it. Chosen over keeping `--unattended` to the drain's lanes and the
    branches no person's session directed: Such a branch is mostly written by
    an agent in a session the person directs, and nobody reads it before it
    merges. Every landing is recorded either way.

## Open Questions

Settled under dcn-2610301913469955 (2026-10-30): The unit of record, the
carrier and the run-state field, the classes and their evidence, what the
forge is asked for, the verb and its flags, and the revert trace, each in
`## Decisions`.

- resolved: whether a person's own branch, merged in `auto` mode with no
  approval seen, counts as landing with no person involved: It does, by the
  product thinker's ruling of 2026-10-30 (decision 16).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: abcd's agents merge changes no person reads, and nothing afterwards says which rule let each one through; we expect writing the admitting rule where the merge carries it, with the mode in the committed configuration at the merge, to make every landing classifiable from history and the forge without anyone's memory, and we are wrong if landings made after this ships still read unknown for a lane abcd opened, or a reverted one cannot be traced to its rule (taken by the technical facilitator under dcn-2610301913469955).
