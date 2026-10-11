---
schema_version: 1
id: "iss-2610040639162928"
slug: "the-plain-terminal-interview-guard-reads"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25 (lane e8term4guard2, plain-Terminal step 4 guard items)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/interview/treewatch.go"
remedy: "In treewatch.go's hashIn, when the entry named is itself a link, record the link and also read what it leads to (fsutil.RealExistingPath, then the same mode/size/hash walk), keyed by its place in the tree when inside it and in full when not; grounds: git resolves these paths through the filesystem (it opens .git/hooks/<name>, .git/config and .git/info/* by path), so the watched state must be the target's, as N1 already does for core.hooksPath"
resolution: "An entry of git's own directory or of a core.hooksPath directory that is a link is now recorded and read where it leads, once; TestAGitEntryReachedThroughALinkIsReadWhereItLeads pins the hooks and info directories, the configuration, one hook and a submodule's hooks."
impact: fix
resolved_by:
  commit: "155e2b64e"
---

The plain-Terminal interview guard reads an entry of git's own directory that is a link (a dotfiles-managed .git/hooks, .git/info or .git/config, or a submodule's hooks under .git/modules) by its link text alone, so a role writing a hook or a configuration change where the link leads is not seen, though git follows the link and runs or reads it. Same class as the re-check's N1 for core.hooksPath, which resolves links before reading.

## Grounds

- pursued: a role writing through a linked .git/hooks, .git/info, .git/config, a linked hook or a linked submodule hooks directory stops the interview naming the real path; a link-target write that passes the interview would show it wrong
