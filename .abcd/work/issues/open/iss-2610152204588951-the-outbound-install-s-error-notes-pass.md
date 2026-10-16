---
schema_version: 1
id: "iss-2610152204588951"
slug: "the-outbound-install-s-error-notes-pass"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "outbound one-switch final review, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/outbound_shim.go"
remedy: "Describe the failing location in words in every outbound shim error note, and keep the raw error out of the note"
---

The outbound install's error notes pass git and filesystem errors through fsutil.RedactHome only, so a failure to open a linked worktree's shared git directory prints a home-relative path outside the working tree, against the rule that notes describe such paths in words.
