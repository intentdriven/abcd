---
schema_version: 1
id: "iss-146"
slug: "guard-bash-gh-write-local-path-fp"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "2026-07-27 session"
found_at: "guard hook, agent shell usage"
remedy: "Waits on ruling G: if a fixture, add the compound gh-write-plus-local-path line, with a reserved example path, to the guard's false-positive corpus (internal/core/guard/testdata/corpus) so abcd's own guard is pinned to allow it, and note that the user-level regex hook stays outside the repository; if closed, move the record to wontfix with the reason that the blocking hook is user-level and not abcd's."
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed G): Give the user-level privacy regex hook a calibration-corpus fixture in this repo, or close since the hook lives outside it?"
---

guard-bash false positive class: a compound command chaining a gh write verb with a LOCAL command taking an absolute path argument (gh pr merge ... ; git worktree remove /abs/path) is blocked by the line-level privacy regex, which cannot see that the path never reaches GitHub. Second guard incident of 2026-07-27 (the first, blocking --no-verify, was a true positive). Both are fixture material for itd-103's calibration corpus: this one is a known-good case the TNR floor must protect; the workaround (split the compound) is recorded here so the friction is not silent.

## Remedy grounds (2026-09-29)

- Why: ruling G's two answers; it is unanswered and none is picked. The corpus exists at this base and already carries the documented-command class the line belongs to.
- Rejected: editing the user-level hook from this repository, which lives outside the tree the record governs.
