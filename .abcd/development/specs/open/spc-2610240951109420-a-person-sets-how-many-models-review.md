---
id: spc-2610240951109420
slug: a-person-sets-how-many-models-review
intent: itd-2610240922461281
origin: researcher-authored
production_mode: hand-written
---
# a-person-sets-how-many-models-review

## Summary

The build loop's validate stage stops running one fixed reviewer per code-review role
and runs a **review set** instead: a list of reviewers per role, each at a model class,
and an optional judge. The bundled default is two reviewers per role, the middle class
and the class below it, with the stricter verdict deciding (the product thinker,
2026-10-24). Every round records, per reviewer, its class, the model asked for and the
model reported, its verdict and its located findings, with whether the fix round
applied or rejected each, so a later choice of review set is made from counts.

Approach, by acceptance criterion:

- **Configuration.** `review.reviewers` (a list of classes, one or two entries, per role
  or for both) and `review.judge` (a class or absent) resolve through the layers the
  pace uses (flag, repository, machine, bundled), read once when the run starts and kept
  for the run. The repository's list is clamped to `review.max_reviewers` from the
  machine's config (two when unset), with one stderr line naming both files. The class
  map `models.classes` (`top`, `middle`, `below`, `smallest`, each an admitted
  `<provider>/<model>` route) is read from the machine's config or the bundled default
  alone; a repository value there is skipped with a warning, as a repository runner
  route is. A class the map lacks, or a judge whose agent has no passing calibration
  record under itd-81, is refused at the existing `review` stage
  (`internal/core/implement/loop/validate.go`, `StageReview`) before anything is
  written. The run's `pace` payload gains a `review` sibling naming each value's layer.
- **Per-request model.** `runner.Request` gains a model, admitted against the provider
  allowlist as a route model is, so one runner serves two reviewers at two classes; the
  claude and opencode runners pass it on their command line. A host-run reviewer is
  handed the class and the resolved model in its `awaiting` payload; the record marks
  its model as asked, not confirmed.
- **Rounds keyed by reviewer.** A round's unit becomes (role, reviewer index): round
  directories, briefs, returns and the receipt match gain the index, so two reviewers of
  one role never collide; the briefs differ only in their return path. The run state's
  schema version is bumped, and an older state is migrated (its single reviewer becomes
  index 0) or refused by the loop's existing version rules. Reviewers are handed out in
  consecutive steps within `sub_agents`, each taking a slot, on the lane's head.
- **Verdict.** Disagreement is one reviewer's verdict passing and the other's failing,
  by the existing pass bit of each verdict word. The round fails when any reviewer fails
  it, as today: that is the stricter verdict.
- **Findings record.** Each `file:line` cited under a return's findings is one located
  finding of that reviewer; a return with none records zero located findings and is not
  refused. After the fix round, each finding carries applied or rejected from the fix
  report, or `unknown` when the report does not say. Two findings at the same file and
  line are marked as cited by both. `implement record --json` and `implement status`
  render them.
- **Judge.** On a disagreement with a judge set, the judge is handed both returns and
  the head and may only strike findings it rules false from the fix brief, each with its
  reason; it never passes a round, and a round whose failing findings are all struck
  goes to a fix round with an empty brief. Until a judge agent clears itd-81's gate the
  judge cannot be enabled, so the default runs without one.

Out of scope, recorded: the other top-class consult cases and the writer-class ruling
(iss-2610240946046901); reviews started by hand outside the loop (itd-83's seam); the
intent-auditor's fan-out.

## Footprint

- packages: internal/core/implement/loop, internal/core/runner, internal/core/layered, internal/surface/cli, commands
- tests: internal/core/implement/loop review-set resolution, clamp and refusal tests; per-reviewer round path and state-migration tests; disagreement and stricter-verdict tests; findings extraction, applied or rejected and overlap tests; judge strike-only tests; internal/core/runner per-request model admission tests

## Steps

1. The review set and the class map
   - packages: internal/core/implement/loop, internal/core/layered, internal/surface/cli, commands
   - tests: the bundled default resolves to two reviewers per role (middle, below); a repository list above the machine's ceiling is clamped with one stderr line naming both files; a repository class map is skipped with a warning; an unknown class and an uncalibrated judge are each refused at the `review` stage with nothing written; a running run keeps the set it started with; the payload names each value's layer
   - needs: none
2. A runner takes a model per request
   - packages: internal/core/runner
   - tests: a request's model is admitted against the provider allowlist and refused off it before the runner starts; the claude and opencode runners pass the request's model and record it as asked beside the model reported; a request without one keeps the route's model
   - needs: none
3. Rounds keyed by reviewer, and the stricter verdict across them
   - packages: internal/core/implement/loop, commands
   - tests: two reviewers of one role get distinct round paths, briefs differing only in the return path, and receipts matched by index; an older run state migrates its reviewer to index 0 or is refused by version; both reviewers are handed out within the sub-agent ceiling; a host-run reviewer's model is recorded as asked, not confirmed; pass against fail is a disagreement, two failing verdicts are not, and any failing reviewer fails the round
   - needs: 1, 2
   - rulings: dcn-2610292236161673 (a request for a model the machine pins is refused on claude with no fallback; a request model and a --route model that differ are refused before launch, naming both)
4. The per-reviewer findings record
   - packages: internal/core/implement/loop, internal/surface/cli, commands
   - tests: each `file:line` under a return's findings is one located finding of its reviewer; a return with none records zero and is not refused; the fix report's applied or rejected status is joined to each finding, `unknown` where it is silent; two findings at the same file and line are marked as cited by both; `implement record --json` carries them
   - needs: 3
5. The judge strikes false findings only
   - packages: internal/core/implement/loop, commands
   - tests: with a calibrated judge fixture, a disagreement hands the judge both returns and the head; a struck finding leaves the fix brief with its reason recorded; the judge never passes a round; a round whose failing findings are all struck still opens a fix round with an empty brief; agreeing reviewers start no judge
   - needs: 3
