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
remedy: "A committed .githooks/prepare-commit-msg writes Assisted-by: <value> from git config abcd.assistedBy, through git interpret-trailers --if-exists doNothing, on the sources whose commit the attribution gate judges (message, commit, squash, and merge without MERGE_HEAD: revert, cherry-pick, amend, a squash, a revert finished after a conflict), leaves a true merge, an editor-bound message and a template alone, and writes nothing when the key is unset or None. Grounds: githooks(5) (prepare-commit-msg runs for every commit git prepares a message for and is not suppressed by --no-verify) and git-interpret-trailers(1), with each source probed on git 2.52."
---

git revert writes no Assisted-by trailer, and the attribution commit gate (now also on the merge queue and on push to main, iss-280) refuses the message, so a plain revert can only be satisfied by rewriting history. Carried from the duplicate iss-2609100513521322's third acceptance: the trailer should be written when the message is written (a commit template or a prepare-commit-msg hook), not demanded after.
