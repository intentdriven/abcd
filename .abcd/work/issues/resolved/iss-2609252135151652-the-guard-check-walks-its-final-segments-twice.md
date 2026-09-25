---
schema_version: 1
id: "iss-2609252135151652"
slug: "the-guard-check-walks-its-final-segments-twice"
severity: "nitpick"
category: "tech-debt"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/guard.go"
resolution: "The duplicated walk-and-refuse block in Check is one block."
impact: internal
resolved_by:
  commit: "7533febc"
---

Check in the shell guard walks the final segments and appends the unknown-sites block signal twice: the block that caches each segment's walk and refuses a capped one is duplicated verbatim (guard.go, from f38daa9a), so a capped walk adds the same signal two times and the second walk is dead work. Found while fixing review4-guard.

## Grounds

- pursued: Check walks its final segments once and adds the unknown-sites signal at most once; a second copy of the block would show it wrong
