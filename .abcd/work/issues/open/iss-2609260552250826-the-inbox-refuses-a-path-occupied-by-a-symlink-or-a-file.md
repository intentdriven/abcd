---
schema_version: 1
id: "iss-2609260552250826"
slug: "the-inbox-refuses-a-path-occupied-by-a-symlink-or-a-file"
severity: "minor"
category: "bug"
source: "drift-detection"
found_during: "v0.11.0 release gate: brief-surface cross-check (autonomous run A, abcd-a2)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/report/inbox.go"
---

The inbox refuses a path occupied by a symlink or a file with an error not wrapped in ErrRefused (internal/core/report/inbox.go:176), so the refusal exits 1 against the documented exit-2 refusal contract in commands/inbox.md:66; a promotion can also exit 1 after its capture was already written. Found by the v0.11.0 brief-surface cross-check (x-065).
