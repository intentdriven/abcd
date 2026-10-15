---
schema_version: 1
id: "iss-2610142321227822"
slug: "build-page-names-one-commit-failure"
severity: "nitpick"
category: "documentation"
source: "review-followup"
found_during: "fix/build-next-followups review"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/build.md"
remedy: "Name all three causes in commands/build.md's worktree-stage list: a hook refused the commit, no git identity is configured, or the commit stalled on a signing prompt or hook, each with its remedy as pickcommit.go words it."
---

The build page's worktree-stage refusal list gives only a missing git identity as the reason a pick commit fails, but the code also refuses when a repository hook rejects the commit, and separately when a signing prompt or hook stalls it (errCommitStalled), with different remedy wording.
