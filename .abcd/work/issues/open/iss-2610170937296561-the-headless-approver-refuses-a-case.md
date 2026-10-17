---
schema_version: 1
id: "iss-2610170937296561"
slug: "the-headless-approver-refuses-a-case"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/approve.go"
remedy: "Read a case pattern, glob or not, as a pattern rather than a command name when the approver looks for programs the line does not name."
---

The headless approver refuses a case statement whose pattern is a glob as program-name-unread: case "$1" in a) echo a;; *) echo other;; esac is refused, with or without the leading ( before each pattern, while the same statement with a plain pattern (b) in place of *) is approved. The guard reads the glob pattern after ;; as a command name it cannot fix, so an ordinary default branch cannot run headless.
