---
schema_version: 1
id: "iss-146"
slug: "guard-bash-gh-write-local-path-fp"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "2026-07-27 session"
found_at: "guard hook, agent shell usage"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (ruling owed: the blocking line-level privacy regex is a user-level hook outside the repo; decide a calibration-corpus fixture or close). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

guard-bash false positive class: a compound command chaining a gh write verb with a LOCAL command taking an absolute path argument (gh pr merge ... ; git worktree remove /abs/path) is blocked by the line-level privacy regex, which cannot see that the path never reaches GitHub. Second guard incident of 2026-07-27 (the first, blocking --no-verify, was a true positive). Both are fixture material for itd-103's calibration corpus: this one is a known-good case the TNR floor must protect; the workaround (split the compound) is recorded here so the friction is not silent.