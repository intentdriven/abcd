---
schema_version: 1
id: "iss-2610120926008447"
slug: "commands-build-md-lists-only-one-of-the"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "build-next intent audit (run-2610110908001704)"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/build.md"
remedy: "Add the three missing worktree-stage refusals to commands/build.md's refusal list, each with what causes it and what to do, worded against pickcommit.go's refusal text and remedies."
---

commands/build.md lists only one of the four refusals the pick commit's worktree stage raises (internal/core/implement/loop/pickcommit.go): it names the missing git identity, but not the intent being no longer planned in the lane's worktree, a lane branch whose first commit is not the pick's, or a worktree dirty with other changes. A person whose 'abcd build next' run stops at the worktree stage on one of those three finds no description of it on the page. Noted by the intent auditor, run-2610110908001704 lane-1.
