---
schema_version: 1
id: "iss-2610120527586962"
slug: "a-restart-and-a-rate-limit-checkpoint"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "security review of the loop small-fixes branch (fix/loop-quota-and-ratelimit), 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/restart.go"
remedy: "hold an await's receipt to a clean file of the lane's own directory outside its asides before a restart or checkpoint reads or removes it, refusing anything else with nothing changed, with a test per escape shape"
resolution: "proveAside, which both the restart and the rate-limit checkpoint run before anything is written, now holds the await's receipt to a clean file of the lane's own directory outside its asides (laneOwnReceipt) and refuses anything else with nothing changed"
impact: fix
---

A restart and a rate-limit checkpoint read, copy aside and then remove the file an await's receipt path names, and that path comes from the run's state file, which an archive of the checkout can carry hand-written, without being held to the lane's directory. A state naming `.env`, `go.mod` or `.git/config` (or a `..` path that cleans to one) had `abcd implement step --restart <lane>` move that file of the checkout into the lane's aside; one naming `changes.patch` or a file of the lane's asides could overwrite a saved patch before the worktree was reset, losing the uncommitted work. The worktree path already had this check (derivedLaneWorktree, iss-2610090821552801); the receipt path was the sibling that sweep missed. The route is abcd's: the removal is the loop's own.

## Grounds

- pursued: a state naming a checkout file, a .. path, an aside file or changes.patch is refused and the file stays put; a restart that moves any of them shows it wrong

## Route

abcd adds this route: the read, copy and removal are the loop's own, of a path from the loop's own state file. Nothing in git or the repository removes a file because a local-tier JSON file names it, so the fix is abcd's and changes nothing about how the repository works.
