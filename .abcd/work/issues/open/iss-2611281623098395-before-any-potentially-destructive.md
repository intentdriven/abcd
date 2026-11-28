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
remedy: "Give every abcd verb that deletes, overwrites, force-moves or discards (sweep --apply, ahoy worktree prune, capture migrate --apply, the land and drain steps that delete branches or worktrees, and any other the guard's hazard registry names) a --dry-run that prints exactly what the real run would do and changes nothing; have the real run refuse unless a dry-run of the same command over the same tree state ran in this session within a bounded time (a receipt keyed on the command and the tree), naming the dry-run to run in the refusal; and add a guard rule that refuses the destructive form of a shell command (git worktree remove, git branch -D, rm -r of tracked paths) without a preceding dry-run or listing of the same target. Grounds: the plan-then-apply pattern of terraform plan/apply and kubectl --dry-run=server, where the apply is checked against the plan it follows."
---

Before any potentially destructive action, abcd must run that command with --dry-run first, with no exception (the person's rule, 2026-11-28). Today some destructive verbs offer a preview (capture migrate without --apply, sweep without --apply) but nothing requires one to have run, several verbs that delete, overwrite or move records or branches have no dry-run at all, and an agent can go straight to the destructive form.
