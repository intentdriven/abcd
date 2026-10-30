---
id: itd-2610301850454860
slug: abcd-writes-down-how-each-change-landed
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-82, itd-2610150602547271]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# abcd records how each change landed

## Press Release

> abcd writes down how each change landed. When a change reaches the default branch, abcd records with it whether a person approved it, the drain landed it under its rule, or it merged automatically after its checks passed, and which rule let it through. Anyone can then ask which changes landed with no person involved, and a change that is later undone can be traced to the rule that admitted it.

## Why This Matters

Split out of itd-2610301750162657 on 2026-10-30 at the product thinker's
ruling in its planning interview: the spot-check of what abcd decided alone
waits for this record, and the record is useful on its own.

Today abcd cannot tell, after the fact, how a change reached the default
branch. Which issues a drain lane landed lives only in the drain's run state,
an uncommitted file in one worktree's local tier
(`internal/core/implement/loop/drain.go`); every merge commit carries the same
author whether a person, a lane or the merge queue landed it; and whether a
person approved a pull request is known only to the forge's review decision,
which abcd reads at landing (`internal/core/landing/land.go`) and keeps
nowhere. A change later reverted cannot be traced to the rule that let it
through, and nobody can list the changes that landed with no person involved.

## Mechanism

- We expect every change on the default branch to be classifiable after the
  fact with no one's memory, because each of the three facts the class needs
  is kept where it cannot be lost: the rule that admitted the lane in a
  trailer on the lane's own commit, which the merge carries into the default
  branch's history; the facilitator mode in the committed configuration at
  the commit the change merged onto; and the forge's approvals, frozen into
  the repository at each release cut. Shown wrong if, for any cycle cut after
  this ships, the frozen ledger holds a row for a lane abcd opened whose class
  or rule reads `unknown`, or two checkouts list the same cycle and disagree.

## Scope Conditions

- A repository whose every change reaches the default branch through a pull
  request the forge merges (a merge queue or auto-merge) or through a
  person's direct push. abcd never pushes the default branch, so a direct
  push is a person's act by construction.
- The forge is GitHub, read through the `gh` client the landing already
  uses. On any other forge, or with no forge reachable, the lane kind and its
  rule still read from git, and every merge with no person's push reads as
  unattended until the forge can be asked.
- The merge queue's method is `MERGE`, which keeps the branch's commits as the
  merge's second parent, as abcd's own default branch shows. A squash or
  rebase merge keeps them only in the message the forge folds; where it
  does not, the lane kind reads `unknown` and the row counts as unattended.
- The implement loop makes its records commit on the lane's branch before the
  lane's preflight runs, so the trailer costs no second gate run. A person's
  branch gets no commit from abcd and so no trailer.
- A `human` set in the machine layer over a repository that declares `auto`
  leaves no trace after the merge. Such a merge with no approving review
  reads as automatic, in the unattended direction, never the other way.
- Until the facilitator mode (itd-2610150602547271) ships, an automatic merge
  names ruling AM1 as its rule, read from the absence of `facilitator.mode`
  in the committed configuration at the merge.
- Scale: abcd's own repository, about 500 merges on the default branch in the
  30 days to 2026-10-30. The current cycle is read live, one forge query per
  pull request; a cycle already cut is read from the frozen ledger alone.

## Acceptance Criteria

Written by the technical facilitator and accepted under the product thinker's
delegation of 2026-10-30 for choices of this kind; each is testable as stated.

- **Given** a drain lane whose issue the drain took under the repository's
  drain rule record, **when** the loop makes the lane's records commit,
  **then** that commit carries `Admitted-by: drain <rule record id>` beside
  its `Resolves:` and `Assisted-by:` trailers, and the attribution gate and
  the commit-msg hook accept it.
- **Given** a build lane for a spec, **when** the loop makes the lane's
  records commit, **then** it carries `Admitted-by: build <spec id>`.
- **Given** a person's branch landed through `abcd land`, **when** land
  pushes it, **then** no commit has been added to the branch and no trailer
  written: abcd never rewrites or extends a person's commits.
- **Given** the default branch with merges since the last release tag,
  **when** `abcd landings` runs with the forge reachable, **then** it prints
  one row per commit on the default branch's first-parent line, newest first,
  each with the commit, the pull request number where the message names one,
  the class (`person-approved`, `drain`, `automatic`, or `person-push`), the
  rule that admitted it, the lane kind, and the issues and intents the
  branch's `Resolves:` and `Delivers:` trailers name.
- **Given** a merged pull request on whose merged head the forge holds an
  approving review by a person other than the author, not dismissed, **when**
  its row is classed, **then** it is `person-approved` and names the approving
  login and time, whatever its lane kind, and the lane kind is still shown.
- **Given** a merged pull request with no such approval whose branch carries
  `Admitted-by: drain <id>`, **when** its row is classed, **then** it is
  `drain` and its rule is that record, by id and path.
- **Given** a merged pull request with no such approval and no drain trailer,
  **when** its row is classed, **then** it is `automatic`, and its rule is
  `facilitator.mode: auto` as `.abcd/config.json` held it at the commit the
  change merged onto, or ruling AM1 where that file sets no mode.
- **Given** `abcd landings --unattended`, **when** it runs, **then** it lists
  only the rows classed `drain` or `automatic`, and a row the evidence cannot
  place with a person is among them, never left out.
- **Given** a commit on the default branch whose message says, in git's own
  words, `This reverts commit <sha>`, **when** `abcd landings <that sha>` or
  `abcd landings <the revert's sha>` runs, **then** it prints the row of the
  first-parent commit that brought the named commit onto the default branch,
  with its class and rule, or says the commit is not on the default branch.
- **Given** rows whose change a later commit on the default branch reverts,
  **when** `abcd landings --reverted` runs, **then** each is marked with the
  reverting commit and the rows are grouped by rule with a count per rule.
- **Given** the forge cannot be reached, **when** the current cycle is listed,
  **then** each row's approval reads `forge unread`, no row is classed
  `person-approved` on a guess, and the command says why.
- **Given** `launch ship` cuts a release, **when** it composes the changelog,
  **then** it also writes the cycle's rows to
  `.abcd/work/landings/<tag>.jsonl` in the release commit, one row per line,
  and refuses the cut, naming the rows, while any row's approval is still
  `forge unread`.
- **Given** a cycle already cut, **when** `abcd landings --since <that tag>`
  runs with no forge reachable, **then** its rows come from the frozen file
  and are complete.
- **Given** a landing from before this shipped, with no `Admitted-by:`
  trailer, **when** it is listed, **then** its lane kind reads `unknown`, its
  class comes from the forge and the committed configuration alone, and no
  history is rewritten to add the trailer.
- **Given** `abcd landings --json`, **when** it runs, **then** it prints the
  same rows as JSON with stable keys, and `abcd landings` writes nothing
  outside the local tier in any form.

## Decisions

Each is the technical facilitator's, under the product thinker's delegation
of 2026-10-30 for choices of this kind, grounded in how a change lands today
(`internal/core/landing/land.go`, `internal/core/implement/loop/land.go`,
`internal/core/implement/loop/drain.go`):

1. **The unit of record is the commit on the default branch's first-parent
   line**: the merge commit the forge makes, a squash or rebase commit, or a
   person's direct push, keyed by its sha. The merge queue makes the merge
   commit, not abcd (its committer is the forge, its message names the pull
   request), so nothing is ever written to that commit, and no row depends on
   it carrying anything but the pull request number and its parents.
2. **Three classes, each from evidence abcd does not have to remember.** A
   person's approval is the forge's review record on the merged head. The
   drain's admission is a trailer on the lane's own commit, carried into the
   default branch's history by the merge. An automatic merge is what is left
   when neither holds, and its rule is the facilitator mode in the committed
   `.abcd/config.json` at the commit the change merged onto (ruling AM1 until
   the mode ships). A fourth class, `person-push`, covers a first-parent
   commit that is no merge and names no pull request, since abcd never
   pushes the default branch.
3. **A person's approval wins.** A drain lane a person approved is
   `person-approved`, with its lane kind still shown: the spot-check draws
   from what no person read, and an approval is a person reading. Chosen over
   ranking the drain first, which would count a read change as unread.
4. **One new trailer, `Admitted-by:`, written by the implement loop on the
   lane's records commit**, the commit it already makes before the lane's
   preflight with `Resolves:`, `Delivers:`, and `Assisted-by:`. Its values are
   `drain <rule record id>` (the record the drain rule loader names, never the
   bundled baseline, which is never a repository's rule) and `build <spec id>`.
   Chosen over a committed file per landing, which would add a file to the
   tree for every merge, and over a trailer on the merge commit, which abcd
   cannot write. The trailer rides the merge into history under the `MERGE`
   method and the announce step already reads trailers this way.
5. **abcd adds nothing to a person's branch.** `abcd land` writes no commit
   and no trailer: a commit added after the person's preflight would strip the
   receipt the push gate reads and send them back through the gate, and
   rewriting their commits is out of the question. A person's branch is
   classed from the forge and the committed configuration alone.
6. **When in doubt, unattended.** Any row the evidence cannot place with a
   person (no approval, lane kind `unknown`, a machine-layer `human` abcd
   cannot see afterwards) is listed under `--unattended`. The listing feeds
   a spot-check, and the costly error is leaving an unread change out of the
   draw, not reading one too many.
7. **The listing is a read-only verb, `abcd landings`**, with `--since
   <tag|date>` (the last release tag by default), `--unattended`,
   `--reverted`, `--json`, and a positional sha to trace one commit. Chosen
   over a flag on `abcd land`, which pushes and arms and should not also be
   the thing a reader runs, and over folding it into `abcd status`, which
   lists announcements that lapse within the hour.
8. **The forge is read, never written.** The approvals come through the
   client the landing already uses, with the pull request query extended to
   the reviews it keeps. No comment, label, or commit status is posted to
   mark abcd's arming: the forge cannot tell abcd's arming from the person's,
   both made under the person's login, and the class does not need it (see 2
   and 6).
9. **Each release cut freezes the cycle's rows into the repository**, as
   `.abcd/work/landings/<tag>.jsonl` in the release commit, beside the
   changelog `launch ship` already derives from records. A trace a year later
   must not depend on the forge still holding the repository; the cut is the
   moment the cycle's set is closed, and it refuses while any row is still
   `forge unread`, as it already refuses a changelog it cannot compose. The
   file is derived and never hand-edited. Between cuts the current cycle is
   read live.
10. **A revert is traced through git's own words.** `abcd landings <sha>`
    follows `This reverts commit <sha>`, the line RS001's revert withdrawal
    already reads, to the named commit, then along the first-parent line to
    the merge that brought it in, then to that row and its rule. A revert of
    a revert is followed the same way. The listing marks a reverted row with
    the reverting commit, and `--reverted` groups by rule, since the
    spot-check's own claim is judged by whether reverts keep tracing to rules
    no finding named.
11. **No backfill.** Landings before this ships carry no trailer and are
    classed from the forge and the configuration alone, lane kind `unknown`;
    history is never rewritten.
12. **Hand-backs and the facilitator's own decisions are out of scope.** This
    record covers changes that reached the default branch. The drain's
    hand-backs stay in its run state and the facilitator's decisions in their
    own records; the spot-check (itd-2610301750162657) reads those groups
    from there.
13. **Naming the reviewers per change stays with itd-2610240922461281**,
    which records each review's findings per reviewer. A row carries the
    `Assisted-by:` values the branch's commits already name; it does not
    duplicate the review record.
14. **Impact: additive.** A new trailer, a new read-only verb, and a derived
    file per cut; nothing existing users rely on changes.

15. Taken on 2026-10-30 under the product thinker's delegation, while they
    were away, and flagged for their review: A person's own branch merged in
    `auto` mode with no approving review is listed as landing with no person
    involved, so `--unattended` shows it. In this repository such a branch is
    mostly written by an agent in a session the person directs, nobody reads
    it before it merges, and the product thinker ruled that people do not
    read code, so it is exactly the work the spot-check exists to sample. The
    choice changes only which rows `--unattended` shows, and every row is
    recorded either way, so it is reversed by one line.

## Open Questions

Settled under the delegation (2026-10-30): the unit of record, the classes
and their evidence, the trailer and where it is written, the verb and its
flags, the freeze at the cut, and the revert trace, each in `## Decisions`.

- resolved: **A person's own branch, merged in `auto` mode with no approving
  review, counts as landing with no person involved** (decision 15).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: abcd's agents merge changes no person reads, and nothing afterwards says which rule let each one through; we expect writing the admitting rule where the merge carries it, with the forge's approvals frozen at each cut, to make every landing classifiable without anyone's memory, and we are wrong if cycles cut after this ships still hold landings whose class or rule reads unknown for a lane abcd opened.
