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
resolution: "The /abcd:ahoy row of 04-naming.md now lists worktree add and worktree prune among the write paths, and a test reads the row's write-paths clause against the command tree."
impact: internal
resolved_by:
  commit: "b13e61ed57fb80b6c7ae324fc2ff7978b8398355"
---

04-naming.md:14 says the bare form and the read-only sub-verbs report and that install, uninstall, connect, remote apply and credential (with --home) are the write paths. abcd ahoy worktree add writes the worktree, its start record and its local tier, and abcd ahoy worktree prune writes the notes archive and removes worktrees (their --help sentences; 01-ahoy.md:491-500).

## Grounds

- pursued: the naming row names every ahoy sub-verb listed as writing, worktree add and worktree prune included; TestNamingRowNamesAhoyWritePaths failing, or a write sub-verb of abcd ahoy the row's clause leaves out, would show it wrong
