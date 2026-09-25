---
schema_version: 1
id: "iss-2609252020505990"
slug: "the-shell-guard-s-stream-rule-missed-a-shell-handed-the"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/payload.go"
---

The shell guard's stream rule missed a shell handed the stdin device behind a pipe (/dev/stdin, /dev/fd/0) and a shell or source handed a process substitution as its script: each runs a downloaded stream as a script exactly as a pipe into a bare shell does, and each allowed. Found by review3-guard finding 4.
