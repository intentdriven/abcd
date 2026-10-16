---
schema_version: 1
id: "iss-2610152204573033"
slug: "when-the-clone-s-hooks-directory-is-a"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "outbound one-switch final review, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/outbound_shim.go"
remedy: "Name the hooks directory as the unreadable part when the directory itself cannot be opened inside the git directory"
---

When the clone's hooks directory is a symlink leading outside the git directory, the outbound install classifies the chained copy hooks/abcd-commit-msg as unreadable and its note advises moving that file aside, which does not help: the cause is the symlinked directory.
