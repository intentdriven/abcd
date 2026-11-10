---
id: spc-2611101423523725
slug: a-person-sets-the-whole-model-set-a-run
intent: itd-2611101419006280
origin: researcher-authored
production_mode: hand-written
---
# a-person-sets-the-whole-model-set-a-run

## Summary

The four model classes take the names `research`, `top`, `default` and `small`, the
bundled review pair becomes `research` and `default` with `top` as a judge that waits
on itd-81's calibration gate, and each class carries an effort, `high` unless the
machine's configuration says otherwise. The class map, the effort and the review set
resolve through the one layered reader the review set already uses
(`internal/core/implement/loop/reviewset.go`), and the same reading rules hold: the
class map and the effort are the machine's or the bundled default's alone, and a
repository names only which classes review.

- **Names.** The class constants and the bundled map are renamed one for one. A
  configuration naming `middle`, `below` or `smallest` anywhere a class is read
  (`models.classes`, `models.effort`, `review.reviewers`, `review.judge`) is refused
  with the whole mapping. A machine class map that sets `top` without `research` is
  refused. A review list or judge naming `top`, while no configuration file names a new
  class, resolves with one stderr diagnostic saying which model `top` stands for.
- **State.** The run state's schema version moves by one. The migration renames every
  stored class label by the old-to-new mapping and gives a stored judge a status. A
  newer state is refused by an earlier abcd through the existing newer-version refusal.
- **Judge.** The bundled judge is stored with a status, `waiting` or `active`, fixed
  when the run starts from the calibration gate's answer. Only `active` is launched,
  awaited or counted. A judge named in a configuration file with no calibrated agent
  keeps the refusal at the `review` stage.
- **Effort.** `models.effort` is one level or an object keyed by class; levels are
  `low`, `medium`, `high`, `xhigh`, `max`. The resolved effort rides with each resolved
  class into the run state and the request a runner is handed. The harness runner
  passes it as its effort flag; a runner that has no way to pass it records that it was
  not passed. The launch record's existing field for the effort asked for carries the
  level, and `not configured` where a role is launched with no class.
- **Surfaces.** Every render of a class shows name, model and effort together
  (`classPhrase` and its callers), and a drive's start names the pair before it asks.

## Footprint

- packages: internal/core/implement/loop, internal/core/runner, internal/core/layered, internal/surface/cli, commands, docs/reference/cli, .abcd/development/brief/04-surfaces, .abcd/development/brief/05-internals, .abcd/development/intents/planned, .abcd/development/intents/drafts, .abcd/development/specs/open
- tests: internal/core/implement/loop review-set resolution, refusal, diagnostic and state-migration tests; internal/core/runner effort argument and launch-record tests; internal/surface/cli render and drive-start tests; a tagged live check that each bundled class launches at its effort

## Steps

1. The classes take their new names, and old names are refused
   - packages: internal/core/implement/loop, internal/core/layered, internal/surface/cli
   - tests: the bundled map resolves `research`, `top`, `default`, `small`; `middle`, `below` or `smallest` in `models.classes`, `review.reviewers` or `review.judge` is refused naming the key, the file and the mapping, with nothing written; a machine map setting `top` without `research` is refused; a review list naming `top` with no new name in any configuration file resolves with one diagnostic, and with a new name present resolves with none; a repository class map is still skipped with its warning
   - needs: none
   - landed: 6be8139b0
2. Run state moves one schema version and stored labels are migrated
   - packages: internal/core/implement/loop
   - tests: a state file of the earlier version with `top`, `middle`, `below` labels reads as `research`, `top`, `default` with each stored model unchanged and the run steppable; a stored judge gains a status; the earlier version's reader refuses the new version as a newer abcd's state (the existing refusal, asserted for this bump); a hand-edited label outside the four names is refused as before
   - needs: 1
   - landed: 71bbfc13c
3. The bundled pair is research and default, and the bundled judge waits
   - packages: internal/core/implement/loop, internal/surface/cli
   - tests: with nothing configured each role resolves to two reviewers at `research` and `default` and a judge at `top` with status `waiting`, the run starts, its record carries one line saying the judge waits on calibration, and no step hands out, awaits or estimates a judge; with a calibrated judge fixture the status is `active`; a status fixed at start does not change when the fixture changes mid-run; `review.judge` in a file with no calibrated agent is refused at the `review` stage
   - needs: 2
   - landed: aba860608
4. Each class carries an effort
   - packages: internal/core/implement/loop, internal/core/layered, internal/core/runner
   - tests: with nothing configured every class resolves `high` from the bundled layer; one level in the machine file applies to all four; an object applies per class and a class left out is `high`; a level outside the five, or an object keyed by anything but the four classes, is refused naming key, value and what is accepted; `models.effort` in a repository file is skipped with one line naming the machine's file; the resolved effort is stored with each class in the run state and survives a re-read
   - needs: 2
   - landed: e4a4b119b
5. A runner is handed the effort and records what was asked
   - packages: internal/core/runner, internal/core/implement/loop
   - tests: the harness runner's arguments carry the effort flag for a request at a class and none for a request with no class or for the runner's own configured model; the launch record's asked effort is the level, and `not configured` with no class; a runner with no way to pass an effort launches and records that it was not passed; a launch the harness fails naming the effort surfaces the effort and class in the failure; a host-run reviewer's await names the class's effort as asked, not confirmed
   - needs: 4
   - landed: 8fa474766
6. Every surface shows a class with its model and effort
   - packages: internal/surface/cli, internal/core/implement/loop
   - tests: `abcd build`'s and `abcd implement`'s text and JSON renders of a review set carry name, model and effort for each reviewer and the judge with its status; a drive's start text names the pair and their efforts before the question; no render shows a class name alone
   - needs: 3, 5
   - landed: 47f23997e
7. The pages and the open records use the new names
   - packages: commands, docs/reference/cli, .abcd/development/brief/04-surfaces, .abcd/development/brief/05-internals, .abcd/development/intents/planned, .abcd/development/intents/drafts, .abcd/development/specs/open
   - tests: the command page for build carries the mapping, a worked class map for a second provider and the one-line cheaper review; the generated command reference is regenerated and its drift check passes; record-lint and docs-lint pass; itd-2610240922461281's first criterion and its spec's first step name the new pair with a dated note pointing at this intent, and its judge step launches only an active judge; itd-2610100704270254 and spc-2610301925429250 use the new names; no shipped, closed or resolved record is changed
   - needs: 6
   - landed: f30035816
8. A live check that each bundled class launches at its effort
   - packages: internal/core/runner
   - tests: behind the existing live build tag, one short launch per bundled class through the harness runner at `high` returns an answer and reports a model; the check is skipped, saying so, where no harness is logged in
   - needs: 5
