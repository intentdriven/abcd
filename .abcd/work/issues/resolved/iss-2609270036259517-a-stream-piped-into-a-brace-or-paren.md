---
schema_version: 1
id: "iss-2609270036259517"
slug: "a-stream-piped-into-a-brace-or-paren"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainG2"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
resolution: "A shell after a separator inside a piped group reads the pipe as a stream (stdinStream from the group input); TestAPipeIntoAGroupFeedsEveryCommandInIt holds it."
impact: fix
resolved_by:
  commit: "a9f16767f"
---

A stream piped into a brace or paren group reaches only the group's commands before its first separator as a stream, so a shell that reads its script from standard input placed after a separator inside a piped group allows under interpreter-reads-stream, while the same shell as the group's first command blocks. Same root as iss-2609270028388291 (the group's input is not handed to every command in it), found while fixing it.

## Grounds

- pursued: a stream piped into a group reaches a stdin-reading shell anywhere in it as interpreter-reads-stream; a group spelling where it allows would show it wrong
