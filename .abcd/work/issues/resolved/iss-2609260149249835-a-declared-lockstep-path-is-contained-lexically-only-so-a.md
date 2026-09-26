---
schema_version: 1
id: "iss-2609260149249835"
slug: "a-declared-lockstep-path-is-contained-lexically-only-so-a"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25 (fix round, review of lane launchkind)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/lockstep.go"
resolution: "The lockstep checks read every repo-rooted document through an os.Root, so a symlink out of the repository is refused and one inside still reads."
impact: fix
resolved_by:
  commit: "c0688598"
---

A declared lockstep path is contained lexically only, so a committed symlink inside the repository that points outside it is read through: launch reads a file out of the tree and surfaces its parse error or the version value at the pointer. The path must be resolved and a target outside the repository refused.

## Grounds

- pursued: a declared lockstep file or primary manifest that is a symlink out of the repository makes the check unreadable without reading the outside value; a lockstep read that still opens a joined path would show it wrong
