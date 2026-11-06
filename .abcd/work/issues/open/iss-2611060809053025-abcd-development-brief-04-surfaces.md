---
schema_version: 1
id: "iss-2611060809053025"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md"
remedy: "Correct the passage to say what the code does."
---

`.abcd/development/brief/04-surfaces/01-ahoy.md:47`: The chapter says the slash command dispatches every ahoy sub-verb, and the command page gives no dispatch for worktree. commands/ahoy.md has an argument hint of install, uninstall, doctor, --dry-run, --remote, remote apply, --providers, connect and credential, and no worktree section (grep finds none), so `ahoy worktree` is not dispatched. commands/ahoy.md:1485-1487 says choosing a shared folder is the person's own act, which the guard refuses to an agent: the page hands over the line rather than dispatching it.
