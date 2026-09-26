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
---

A declared lockstep path is contained lexically only, so a committed symlink inside the repository that points outside it is read through: launch reads a file out of the tree and surfaces its parse error or the version value at the pointer. The path must be resolved and a target outside the repository refused.
