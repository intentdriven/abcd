---
schema_version: 1
id: "iss-2611040554013625"
slug: "04-naming-md-14-says-the-bare-form-and"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md"
remedy: "Add worktree add and worktree prune to the row's write paths."
---

04-naming.md:14 says the bare form and the read-only sub-verbs report and that install, uninstall, connect, remote apply and credential (with --home) are the write paths. abcd ahoy worktree add writes the worktree, its start record and its local tier, and abcd ahoy worktree prune writes the notes archive and removes worktrees (their --help sentences; 01-ahoy.md:491-500).
