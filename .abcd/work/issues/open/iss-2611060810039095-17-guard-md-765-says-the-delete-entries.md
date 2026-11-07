---
schema_version: 1
id: "iss-2611060810039095"
slug: "17-guard-md-765-says-the-delete-entries"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/17-guard.md"
remedy: "Say the refusal also applies where the directory is not a readable checkout, and drop the implication that a delete there only warns."
---

`17-guard.md:765` says the delete entries only warn, and refuse in a repository. Outside any repository `git branch -D x` also blocks as `git-delete-unproven`, exit 1, with the reason `the directory the command runs in is not a git checkout abcd can read` (`internal/core/sweep/deleteproof.go:159`); `git worktree remove x` and `git push origin --delete x` do the same.
