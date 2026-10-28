---
schema_version: 1
id: "iss-2610110012045504"
slug: "a-validated-lane-cannot-take-main-in"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "abcd-60 drain run 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop"
remedy: "Let the land stage accept a head whose only commits past the judged head are merges of the default branch that change no file the validators judged (or re-run only the definition of done over them), so a lane held across a default-branch change can sync before its resolve commit instead of after its PR opens."
---

A validated lane cannot take the default branch in before it lands. On 2026-10-11 three drain lanes held through a peer's record-rename freeze were validated against a base that the rename then moved. Merging origin/main into one lane branch before its landing made implement step refuse at land ('is at <merge>, not at the head <sha> its validators judged'). So the only order that works is to land on the stale base (the resolve commit moves the record under its old long name), push, open the PR, and only then merge main and run the rename tool on top. Each lane then pays a third full preflight, and its PR is conflicting until that second push.

The same order is also what the land stage's own remedy prescribes, and that remedy does not work either. On 2026-10-27 a lane's pull request conflicted with main, and the land stage refused with the remedy: merge origin/main into the lane branch, resolve the conflict, run `make preflight`, push, then run `abcd implement step` again, "which arms it once the forge reports it mergeable". After those steps the land stage still armed auto-merge with `--match-head-commit` set to the judged head recorded in the run state (`internal/core/implement/loop/land.go:763`, `ld.Pushed`), not the preflighted merge head now on the branch. The forge refused every retry with "expected head oid does not match the current head oid (enablePullRequestAutoMerge)", and the pull request could be armed only by hand. The remedy text and the arming code must agree: either the arm takes the pushed head once a preflight receipt names it, or the conflict remedy routes the merge through a fresh validation round.
