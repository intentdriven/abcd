---
schema_version: 1
id: "iss-2609251125591539"
slug: "git-revert-writes-no-assisted-by-trailer-and-the-attribution"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
deferred_after: "v0.11.1"
deferral_reason: "lane owed (lapsed-deferral triage, run A 2026-09-29): the githooks lane (a trust path, over an hour with its tests): a prepare-commit-msg hook in .githooks that appends the Assisted-by trailer from git config abcd.assistedBy when set and otherwise leaves the message for the commit gates, never inventing Assisted-by: None, with the commit-msg hook's environment pin and a test per commit source, plus the documented revert remedy. The lane must reconcile it with the scaffolded managed-repo hook (internal/core/ahoy/defaults/prepare-commit-msg), which by design never writes a value."
---

git revert writes no Assisted-by trailer, and the attribution commit gate (now also on the merge queue and on push to main, iss-280) refuses the message, so a plain revert can only be satisfied by rewriting history. Carried from the duplicate iss-2609100513521322's third acceptance: the trailer should be written when the message is written (a commit template or a prepare-commit-msg hook), not demanded after.
