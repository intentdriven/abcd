---
schema_version: 1
id: "iss-2610120640452939"
slug: "lane-creation-and-restart-still-switch"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "security fix lane C, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/repo.go"
remedy: "Decide per command whether the repository's own hooks and maintenance settings should run (worktree add runs post-checkout for the person's own git), then route those calls through the commit-path helper that keeps the repository's settings; prove with a post-checkout hook test"
---

Under the security scope rule (abcd never changes how a repository works on its own), abcd's lane creation (git worktree add, which runs a post-checkout hook) and a restart's reset and clean still run with core.hooksPath=/dev/null, gc.auto=0 and maintenance.auto=false through pickGit/WorkingTreeCommand, while the pick commit and sync merge were handed back to the repository's own settings by iss-2610111729399920. Found by lane C on 2026-10-12.
