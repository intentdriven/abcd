---
schema_version: 1
id: "iss-2611281623098395"
slug: "before-any-potentially-destructive"
severity: "major"
category: "process"
source: "user-observation"
found_during: "the person's ruling during the 2026-11-28 away run"
origin: researcher-authored
production_mode: hand-written
remedy: "Give every abcd verb that deletes, overwrites or moves a record, branch or file and has no preview today a --dry-run that prints exactly what the real run would do and changes nothing: history discard, capture resolve, capture wontfix and spec close, plus the drive's own land step that removes a worktree; keep the preview the report-only forms of sweep, capture migrate, reserve --sweep and ahoy worktree prune --dry-run already give; and make each real run refuse unless a dry-run of the same command over the same state ran within a bounded time, through a receipt abcd writes in the local tier keyed on the command and the tree (or, for a verb that acts on no checkout, such as history discard --root-sha or --pending, in abcd's home keyed on what it acts on), naming the dry-run in the refusal. The guard's existing delete-proof block on git worktree remove, git branch -D and git push --delete (internal/core/guard/deletes.go) stays as it is; the one shell gap is rm -r of tracked paths, which gets the same proof. Grounds: the plan-then-apply pattern of terraform plan/apply and kubectl --dry-run=server, where the apply is checked against the plan it follows."
---

Before any potentially destructive action, abcd must run that command with --dry-run first, with no exception (the person's rule, 2026-11-28). Today some destructive verbs offer a preview (capture migrate without --apply, sweep without --apply) but nothing requires one to have run, several verbs that delete, overwrite or move records or branches have no dry-run at all, and an agent can go straight to the destructive form.
