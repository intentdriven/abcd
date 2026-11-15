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
[itd-2610150602547271](itd-2610150602547271-a-repository-declares-whether-its.md)
(the facilitator mode, shipped: In `auto` a maintainer's pull request is armed
only when a review receipt names its head, in `human` only once an approval of
the head counts, and this record says which of the two admitted each merge).
Blocks
[itd-2610301750162657](../planned/itd-2610301750162657-a-person-spot-checks-what-abcd-s-agents.md)
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

<!-- abcd-review: INGESTED receipt=rcp-6cf4f0f63ccf -->
Fidelity review — receipt rcp-6cf4f0f63ccf (verifier intent-auditor claude-fable-5-2).

Provenance: intent-auditor@claude-fable-5-2 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:f13efb3b0d6f34b4bbc6940f1b2707e4ab57b4895f40a7114cfba7baeaf28cd5
Input attestations: tree:227bc8c26030d2952e8f3d30dbbbec9d4bb2870b (frozen copy lane-al-227bc8c2; read only, no test or build run)@-; spec:.abcd/development/specs/closed/spc-2610301950480609-abcd-writes-down-how-each-change-landed.md@-;

Acceptance rollup: MET 12 · MET_WITH_CONCERNS 9 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: The records commit appends the admission after Resolves: and before Assisted-by:, and a test reads `Admitted-by: drain adr-N` back in that order. Concern: no test installs the repository's commit-msg hook and commits through it; the attribution gate and the hook's RS004 message form are run only over a `build spc-N` and a `build iss-N` message, never over a message carrying the drain form, so acceptance of the drain trailer by both gates is shown by the scripts naming no Admitted-by key, not by a run.
  evidence: internal/core/implement/loop/land.go:631 — "trailers = append(trailers, admittedTrailers(*st.Admitted, lane.Resolves)...)"
  evidence: internal/core/implement/loop/drain_admitted_test.go:85 — "func TestDrainLaneCommitsItsRule(t *testing.T) {"
  evidence: internal/core/implement/loop/land_attribution_test.go:158 — "func TestTheLandedRangePassesTheAttributionGate(t *testing.T) {"
  evidence: .abcd/development/specs/closed/spc-2610301950480609-abcd-writes-down-how-each-change-landed.md:53 — "No test installs the repository's hook in the fixture and makes the commit through it."
- ac-2 — MET_WITH_CONCERNS: A spec-keyed lane's records commit is asserted to carry exactly the build trailer naming its spec; an issue-keyed run's state takes `build <iss-N>` and the composer writes that trailer. Concern: the named test lands only the intent-keyed run; the issue form is shown by the start-state test and a unit test of the trailer composer, and the one issue-lane records commit test asserts only the prefix `Admitted-by: `, not the issue id.
  evidence: internal/core/implement/loop/land_admitted_test.go:23 — "func TestBuildLaneCommitsItsSpecOrIssue(t *testing.T) {"
  evidence: internal/core/implement/loop/loop.go:592 — "return &Admitted{Kind: landings.KindBuild, Record: chk.Key}"
  evidence: internal/core/implement/loop/land_admitted_test.go:54 — "if got := admittedTrailers(a, nil); !slices.Equal(got, []string{"
  evidence: internal/core/implement/loop/land_remedies_test.go:586 — "!strings.Contains(msg, "Admitted-by: ")"
- ac-3 — MET: A landing that moves no record commits with --allow-empty and the prose `records only the admission`; the test asserts an empty diff-tree, the trailer, the lane head equal to the commit, and the pushed head equal to it.
  evidence: internal/core/implement/loop/land.go:674 — "commit = append(commit, "--allow-empty")"
  evidence: internal/core/implement/loop/land_admitted_test.go:134 — "l2.Landing.Pushed != l.HeadSHA || f.remoteBranch(t, l.Branch) != l.HeadSHA"
- ac-4 — MET_WITH_CONCERNS: The start writes `admitted` (kind, record, and the rule's path for a drain) and a state without it is refused at the records step naming the field with the branch unmoved. Concerns: a run `abcd build` starts as an auto-plan run or a trial names no admission at its start; and the field landed in schema version 26 with version 25 read as unadmitted and admitted late on resume, not in version 16 as planned.
  evidence: internal/core/implement/loop/loop.go:870 — "Admitted: admission(chk, o),"
  evidence: internal/core/implement/loop/state_admitted_test.go:28 — "the drain's lane is admitted under the rule record, by id and path"
  evidence: internal/core/implement/loop/land.go:413 — "the run's state names no `admitted`, the rule its lanes were admitted under"
  evidence: internal/core/implement/loop/land_admitted_test.go:68 — "tip != judged || l.HeadSHA != judged || l.Landing.RecordsDone"
  evidence: internal/core/implement/loop/loop.go:593 — "case chk.autoPlan != nil || chk.Spec == "": return nil"
- ac-5 — MET: The test lands a person's branch and asserts the local branch, origin's branch and the head are the commit the person made, and that no commit in the range carries an Admitted-by line; the gate, push and arming therefore name that head.
  evidence: internal/core/landing/land_test.go:1407 — "if after := f.head(); after != before {"
  evidence: internal/core/landing/land_test.go:1425 — "strings.Contains(strings.ToLower(msgs), "admitted-by")"
- ac-6 — MET: The reader walks the first-parent line newest first and emits one row per commit; the text row prints commit, pull request, class, rule, lane, approval and the Resolves/Delivers ids, and tests assert the order and the trailers.
  evidence: internal/core/landings/landings.go:492 — "func (h *history) firstParent(to string) []string {"
  evidence: internal/surface/cli/landings.go:358 — "head := []string{landingsShort(r.Commit), pr, landingsOrDash(r.Class), rule, "lane " + landingsOrDash(r.Lane), "approval " + approval}"
  evidence: internal/core/landings/classes_test.go:442 — "if want := []string{m3, push, sq, m1}; strings.Join(got, " ") != strings.Join(want, " ")"
  evidence: internal/core/landings/classes_test.go:478 — "func TestRowCarriesResolvesAndDelivers(t *testing.T) {"
- ac-7 — MET: A latest submitted review approving the merged head by someone other than the author makes the row person-approved with the reviewer's login and the review's time; an earlier-commit, dismissed, own and pending approval count nothing; the lane kind stays on the row; the login is held only in the gitignored forge cache.
  evidence: internal/core/landings/forge.go:292 — "return Approval{Seen: true, By: lr.Login, At: at.UTC(), Source: SourceForge}, """
  evidence: internal/core/landings/classes_test.go:494 — "r.Class != ClassPersonApproved || r.Approval != (Approval{Seen: true, By: alice, At: time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC), Source: SourceForge})"
  evidence: internal/core/landings/classes_test.go:543 — "r.Class != ClassPersonApproved || r.Lane != KindDrain || r.Unattended"
  evidence: internal/core/landings/forge_test.go:177 — "t.Fatalf("the committed file %s carries the login", p)"
- ac-8 — MET_WITH_CONCERNS: A recorded approval of the merged head written in a sibling worktree's local tier classes the row person-approved with source `local tier`, which the row prints. Concern (signed off by dcn-2611120647286592): the record counts only where the forge's answer for the pull request was read and confirms the merge; with the forge unread the row is unattended whatever the local tier holds, a condition the criterion does not state.
  evidence: internal/core/landings/landings.go:727 — "if row.MergedHead != "" && a != nil { row.Approval = l.recorded(row.MergedHead, wts)"
  evidence: internal/core/landings/classes_test.go:557 — "r.Class != ClassPersonApproved || r.Approval != (Approval{Seen: true, Source: SourceLocalTier}) || r.Unattended"
- ac-9 — MET_WITH_CONCERNS: With the forge's answer read, no approval seen and the first parent's committed mode human, the row is person-approved by `facilitator.mode: human` with approval `none seen`. Concern (signed off by dcn-2611120632175235): with the forge's answer unread the same merge is classed `automatic` under the rule `forge unread` and is unattended, which the criterion's given does not exclude.
  evidence: internal/core/landings/landings.go:747 — "case human(row.Mode) && a != nil:"
  evidence: internal/core/landings/landings.go:762 — "row.Class, row.Rule = ClassAutomatic, Rule{Kind: RuleForgeUnread}"
  evidence: internal/core/landings/classes_test.go:580 — "r.Class != ClassPersonApproved || r.Mode != landing.ModeHuman || r.Approval.Source != SourceNoneSeen"
- ac-10 — MET: Under auto with no approval seen, a branch carrying the drain's admission is classed drain, its rule the ADR id with the record's path read from the first parent's tree.
  evidence: internal/core/landings/landings.go:751 — "case adm != nil && adm.Kind == KindDrain: row.Class = ClassDrain"
  evidence: internal/core/landings/classes_test.go:627 — "r.Rule != (Rule{Kind: KindDrain, ID: "adr-2609291342092738", Path: ruleRel})"
- ac-11 — MET: Under auto with no approval and no drain trailer the row is automatic, its rule `facilitator.mode: auto` with the config path and the first parent it was read at; a build lane and a trailer-less branch are both asserted.
  evidence: internal/core/landings/landings.go:764 — "row.Class, row.Rule = ClassAutomatic, f.modeRule()"
  evidence: internal/core/landings/classes_test.go:643 — "want := Rule{Kind: RuleMode, ID: landing.ModeAuto, Path: ".abcd/config.json", At: c.at}"
- ac-12 — MET: A commit whose subject names no pull request is classed person-push with an empty rule, an empty lane and git's committer name; the test asserts each and that the forge is not asked.
  evidence: internal/core/landings/landings.go:778 — "row.Class = ClassPersonPush"
  evidence: internal/core/landings/classes_test.go:677 — "r.Class != ClassPersonPush || r.Committer != "Pat ?[31mPusher" || r.Rule != (Rule{}) || r.Lane != """
- ac-13 — MET_WITH_CONCERNS: A one-parent row naming a pull request (a squash) takes its merged head, lane and trailers from the forge's listing of the pull request's commits, and with the forge unread its lane is unknown and it is unattended. Concern: only the squash half is delivered; a rebase merge's commits name no pull request in their subjects, so each is read as a person's push and no forge commit listing is read for them.
  evidence: internal/core/landings/landings.go:709 — "for _, c := range a.Commits.Nodes { messages = append(messages, c.Commit.Message)"
  evidence: internal/core/landings/classes_test.go:724 — "r.Class != ClassDrain || r.Lane != KindDrain || r.MergedHead != head"
  evidence: internal/core/landings/classes_test.go:754 — "r.Lane != LaneUnknown || !r.Unattended || r.Approval.Source != SourceForgeUnread"
  evidence: internal/core/landings/landings.go:53 — "the shape of a commit pushed to the default branch, which a // rebase merge's commits share"
- ac-14 — MET_WITH_CONCERNS: `--unattended` keeps exactly the rows with Unattended set, and every row the evidence cannot place with a person is among them. Concern (signed off by dcn-2611120418288645): the list is not only rows classed drain or automatic; a person-push row with no recorded approval of its commit is unattended and listed too.
  evidence: internal/surface/cli/landings.go:268 — "func unattendedRows(rows []landings.Row) []landings.Row {"
  evidence: internal/core/landings/landings.go:786 — "row.Unattended = !row.Approval.Seen"
  evidence: internal/core/landings/classes_test.go:766 — "for m, want := range map[string]bool{mu: true, mr: true, ma: false, push: true}"
- ac-15 — MET: The positional trace resolves a name, follows it to the first-parent row that brought it, and prints that row with its class and rule; for the reverting commit it prints its row and the rows it reverts; a commit the tip does not reach is said to be not on the default branch.
  evidence: internal/core/landings/revert.go:748 — "func TraceCommit(ctx context.Context, repoRoot, name string, o Options) (Trace, error) {"
  evidence: internal/core/landings/revert_test.go:40 — "tr.Row.Commit != m1 || tr.Row.Class != ClassAutomatic || tr.Row.Rule.String() != "facilitator.mode: auto""
  evidence: internal/core/landings/revert_test.go:122 — "tr.Row != nil || tr.Commit != off || tr.Says != SaysNotOnDefault"
  evidence: internal/surface/cli/landings.go:524 — "fmt.Fprintln(w, "it reverts:")"
- ac-16 — MET_WITH_CONCERNS: A squashed revert whose pull request body says `Reverts #N` or `Reverts <owner>/<repo>#N` marks pull request N's row with kind forge, and a revert with neither line is untraceable and grouped on its own. Concern (dcn-2611120948130053, dcn-2611121446194027): the body is asked only for a one-parent row whose subject is git revert's own, so a squashed revert whose title was reworded is not seen at all, and a body edited in the merge's second or after is not read.
  evidence: internal/core/landings/revert.go:117 — "forgeLineRe = regexp.MustCompile(`^Reverts ((?:([A-Za-z0-9] [A-Za-z0-9-]{0,38}/[A-Za-z0-9._-]{1,100}))?#([1-9] [0-9]{0,8}))\s*$`)"
  evidence: internal/core/landings/revert_test.go:205 — "*got != (Revert{By: s3, ByRow: s3, Kind: RevertForge})"
  evidence: internal/core/landings/revert_test.go:253 — "len(g) != 1 || g[0].Rule != GroupUntraceable || g[0].Count != 3"
  evidence: internal/core/landings/revert.go:26 — "title was reworded is not seen at all: nothing marks the row it reverts"
- ac-17 — MET: Each reverted row carries the reverting commit and its row; RevertedGroups groups them by the rule's name with a count, and the verb prints each group's rule and count before its rows.
  evidence: internal/core/landings/revert.go:640 — "func RevertedGroups(l Listing) []RevertGroup {"
  evidence: internal/core/landings/revert_test.go:278 — ""facilitator.mode: auto=2[" + mb[:7] + " " + ma[:7] + "]""
  evidence: internal/surface/cli/landings.go:475 — "fmt.Fprintf(w, "%s %d\n", landingsOrDash(g.Rule), g.Count)"
- ac-18 — MET_WITH_CONCERNS: With the forge unreachable every pull-request row's approval reads `forge unread`, none is person-approved by a review, the reason is on the listing once and the verb prints it; under auto, class and rule come from history (drain by its trailer, automatic by the mode). Concern (signed off by dcn-2611120632175235 and dcn-2611120647286592): under the human mode class and rule do not come from history: the row is classed automatic under the rule `forge unread`, not person-approved by `facilitator.mode: human`, and a recorded approval is not counted. The named test covers auto only.
  evidence: internal/core/landings/landings.go:732 — "case a == nil: row.Approval.Source = SourceForgeUnread"
  evidence: internal/core/landings/landings.go:759 — "case human(row.Mode): // An unattended merge never carries the human mode's rule"
  evidence: internal/core/landings/classes_test.go:792 — "r.Class != ClassDrain || r.Rule.ID != "adr-2609291342092738" || r.Approval.Source != SourceForgeUnread || !r.Unattended"
  evidence: internal/surface/cli/landings.go:490 — "{"range", out.SinceNote}, {"forge", out.ForgeUnread}, {"cache", out.CacheUnwritten}, {"mode", out.ModeUnread},"
- ac-19 — MET: The resolver takes a tag (looked up as that exact ref), a date (the range opens at the first first-parent commit at or after the day), or nothing (the last v* tag the tip reaches), and refuses naming --since when no release tag exists.
  evidence: internal/core/landings/since.go:84 — "func Resolve(ctx context.Context, repoRoot, since string) (Range, error) {"
  evidence: internal/core/landings/since.go:173 — ""describe", "--tags", "--match", ReleaseTagPattern, "--abbrev=0", to"
  evidence: internal/core/landings/since_test.go:45 — "func TestSinceTag(t *testing.T) {"
  evidence: internal/core/landings/since_test.go:93 — "func TestNoTagAndNoSinceRefuses(t *testing.T) {"
- ac-20 — MET: A branch with no Admitted-by trailer reads lane `unknown`; its class is decided by the approvals and the committed mode; the test compares every ref and reachable commit before and after the listing.
  evidence: internal/core/landings/landings.go:716 — "row.Lane = LaneUnknown"
  evidence: internal/core/landings/classes_test.go:838 — "if refs() != before { t.Fatal("the listing moved history")"
- ac-21 — MET: --json prints the listing's rows through one marshal of the same Listing the text renders; a test pins the full key set of every object against a held list and the command page; another hashes the git directory and the local tier outside the forge cache before and after the listing, the reverted listing and a trace, and finds only the cache new.
  evidence: internal/surface/cli/landings.go:235 — "b, err := json.MarshalIndent(v, "", " ")"
  evidence: internal/surface/cli/landings_surface_test.go:1180 — "t.Fatalf("%s carries %v, the test holds %v", obj, got[obj], want)"
  evidence: internal/surface/cli/landings_surface_test.go:510 — "t.Fatal("the verb changed the local tier outside its forge cache")"

Gap audit:
- honoured:
  - Every lane's records commit carries the rule that admitted it, read from the run's state, and a lane with nothing to record still commits
    evidence: internal/core/implement/loop/land.go:631 — "trailers = append(trailers, admittedTrailers(*st.Admitted, lane.Resolves)...)"
    evidence: internal/core/implement/loop/land.go:674 — "commit = append(commit, "--allow-empty")"
  - Nothing is added to a person's branch landed through abcd land
    evidence: internal/core/landing/land_test.go:1407 — "func TestLandMovesNoHeadAndWritesNoTrailer(t *testing.T) {"
  - A read-only verb, abcd landings, wired into the CLI, with its page, sentence, help group and brief chapter, lists one classed row per first-parent commit
    evidence: internal/surface/cli/cli.go:421 — "root.AddCommand(newLandingsCommand(&asJSON))"
    evidence: internal/surface/cli/helpgroups.go:126 — ""landings": {group: groupRecords},"
    evidence: internal/core/surface/sentences.go:416 — ""abcd landings": "List how each change landed on the default branch and what admitted it: ""
  - The four classes, the drain rule by id and path, and the mode as the first parent committed it
    evidence: internal/core/landings/landings.go:745 — "case row.Approval.Seen: row.Class, row.Rule = ClassPersonApproved, f.modeRule()"
  - The revert trace by git's own words, the positional trace, and the reverted rows grouped by rule with counts
    evidence: internal/core/landings/revert.go:115 — "revertLineRe = regexp.MustCompile(`^This reverts commit ([0-9a-f]{7,64})(?:[^0-9A-Za-z]|$)`)"
    evidence: internal/core/landings/revert.go:640 — "func RevertedGroups(l Listing) []RevertGroup {"
  - The listing fetches nothing and writes only its forge cache in the local tier; a login reaches no committed file
    evidence: internal/surface/cli/landings_surface_test.go:510 — "words := regexp.MustCompile(`(^| )(fetch|push|pull|ls-remote|remote|clone|send-pack|fetch-pack)( |$)`)"
    evidence: internal/core/landings/forge.go:68 — "const CacheRelDir = landing.TierRelDir + "/landings""
- diverged:
  - `--unattended` lists only rows classed drain or automatic: delivered, it also lists every person-push row with no recorded approval of its commit
    evidence: internal/core/landings/landings.go:786 — "row.Unattended = !row.Approval.Seen"
  - With the forge unreachable class and rule come from history: delivered, a merge under the human mode whose forge answer was not read is classed automatic under the rule `forge unread`, and a recorded approval of its merged head is not counted
    evidence: internal/core/landings/landings.go:762 — "row.Class, row.Rule = ClassAutomatic, Rule{Kind: RuleForgeUnread}"
    evidence: internal/core/landings/landings.go:727 — "if row.MergedHead != "" && a != nil {"
  - The attribution gate and the commit-msg hook accept the drain's records commit: delivered as the gates' judgements run over a build-form message, with no hook installed in a fixture and no run over the drain form
    evidence: .abcd/development/specs/closed/spc-2610301950480609-abcd-writes-down-how-each-change-landed.md:53 — "No test installs the repository's hook in the fixture and makes the commit through it."
  - A squashed revert is matched by the forge's `Reverts #N` line: delivered only for a row whose subject is git revert's own and whose body was not edited at or after the merge
    evidence: internal/core/landings/revert.go:395 — "if !accepted[r.Commit] || len(c.parents) > 1 || !revertSubject(c.message) ||"
  - The forge is asked once per pull request: delivered as batches of twenty asked again in halves within a bound, answers that can change asked on every listing, and a squashed revert's body asked apart on every listing
    evidence: .abcd/development/brief/02-constraints/03-invariants.md:182 — "in batches of up to twenty, asked again in halves, within a bound"
  - The admission joins state schema version 16: delivered as version 26, with a version-25 run admitted late when it is resumed
    evidence: internal/core/implement/loop/state.go:318 — "// Version 26 added the run's `admitted` (itd-2610301850454860)"
- missing:
  - A rebase-merged pull request's commits and trailers read from the forge's listing of its commits: a rebase merge's commits are read as person-push rows and no listing is asked for them
    evidence: internal/core/landings/landings.go:53 — "the shape of a commit pushed to the default branch, which a // rebase merge's commits share"
  - A test that lands an issue-keyed run and reads `Admitted-by: build <iss-N>` on its records commit, which the spec's criterion line says TestBuildLaneCommitsItsSpecOrIssue holds
    evidence: internal/core/implement/loop/land_admitted_test.go:23 — "func TestBuildLaneCommitsItsSpecOrIssue(t *testing.T) {"
  - A test that makes the records commit through the repository's own commit-msg hook installed in a fixture
    evidence: internal/core/implement/loop/land_attribution_test.go:158 — "RS004's message half, which the commit-msg hook runs, accepts its message"

Scope-condition dispositions:
- cond-2610301950483000 — survived: The reader places every first-parent commit as either a commit whose subject names a pull request the forge confirms merged as it, or a push, and the delivery adds no push of the default branch: the verb fetches nothing and pushes nothing.
  evidence: internal/core/landings/landings.go:677 — "if row.PR == 0 { return l.push(row, "", wts), "", nil"
  evidence: internal/surface/cli/landings_surface_test.go:510 — "t.Fatalf("the verb ran git %s", l)"
- cond-2610301950485184 — survived: The reader's one door to the forge is `gh api graphql` run through the landing's own client.
  evidence: internal/core/landings/forge.go:115 — "return []string{"api", "graphql", "-F", "o={owner}", "-F", "r={repo}", "-f", "query=" + batchQuery(asks)}"
  evidence: internal/core/landings/landings.go:352 — "forge = func(args ...string) (string, error) { return landing.GHAnswerContext(ctx, repoRoot, args...) }"
- cond-2610301950486745 — narrowed: A two-parent merge's branch is read from its second parent and a squash's commits from the forge, as assumed; a rebase merge's commits carry no pull request in their subjects, so the reader does not find them on the forge and reads each as a push.
  narrowing: holds for the MERGE method and for a squash whose subject ends in (#N); a rebase merge's commits are read as person-push rows, with nothing read from the forge for them
  evidence: internal/core/landings/landings.go:589 — "if f.twoParent { f.row.MergedHead = c.parents[1]"
  evidence: internal/core/landings/landings.go:53 — "the shape of a commit pushed to the default branch, which a // rebase merge's commits share"
- cond-2610301950486793 — survived: The records commit is made on the lane's branch before the push, and the push is still refused until a preflight receipt names the lane's head; the record-only test preflights the records commit and then pushes it.
  evidence: internal/core/implement/loop/land.go:1962 — "no preflight receipt names the lane's head "+lane.HeadSHA+", so the pre-push gate would refuse its push"
  evidence: internal/core/implement/loop/land_admitted_test.go:134 — "preflighted(t, l, l.HeadSHA) f.step(t) // push"
- cond-2610301950487033 — narrowed: The recorded approval is a local-tier file named by the head that the reader finds from a sibling worktree, as assumed; the delivery reads no review receipt at all, and it counts the recorded approval of a merged head only beside a forge answer read for that pull request.
  narrowing: holds for the recorded approval read across worktrees, and only where the forge's answer for the pull request was read; the review-receipt half is not exercised by the delivery
  evidence: internal/core/landings/classes_test.go:557 — "if _, err := landing.RecordApproval(wt, b); err != nil {"
  evidence: internal/core/landings/landings.go:724 — "// A recorded approval of the merged head counts only beside a forge answer // read for the pull request"
- cond-2610301950485439 — narrowed: A listing whose batches are answered whole asks about each pull request once and a second listing asks nothing; but a batch too large or carrying an unplaced error is asked again in halves, an answer that can change is asked on every listing, and a squashed revert's body is a further ask. No test lists a range at the stated scale of about 500 merges.
  narrowing: once per pull request holds only where each batch of up to twenty is answered whole and the answer is one the cache keeps; otherwise the forge is asked again within a bound of 64 calls a listing, and a squashed revert's body is asked apart on every listing
  evidence: internal/core/landings/forge_test.go:60 — "if ff.calls != 3 || len(ff.asked) != 2*MaxBatch+3 {"
  evidence: internal/core/landings/forge.go:72 — "var maxHalvingCalls = 64"
<!-- abcd-review-end receipt=rcp-6cf4f0f63ccf -->

## Grounds

- pursued: abcd's agents merge changes no person reads, and nothing afterwards says which rule let each one through; we expect writing the admitting rule where the merge carries it, with the mode in the committed configuration at the merge, to make every landing classifiable from history and the forge without anyone's memory, and we are wrong if landings made after this ships still read unknown for a lane abcd opened, or a reverted one cannot be traced to its rule (taken by the technical facilitator under dcn-2610301913469955).
