---
schema_version: 1
id: "iss-2609270036259517"
slug: "a-stream-piped-into-a-brace-or-paren-group-reaches-only-the"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainG2"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
---

A stream piped into a brace or paren group reaches only the group's commands before its first separator as a stream, so a shell that reads its script from standard input placed after a separator inside a piped group allows under interpreter-reads-stream, while the same shell as the group's first command blocks. Same root as iss-2609270028388291 (the group's input is not handed to every command in it), found while fixing it.
