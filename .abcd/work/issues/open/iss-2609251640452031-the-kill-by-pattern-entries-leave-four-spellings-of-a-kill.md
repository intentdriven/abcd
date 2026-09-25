---
schema_version: 1
id: "iss-2609251640452031"
slug: "the-kill-by-pattern-entries-leave-four-spellings-of-a-kill"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/defaults/guard.json"
---

The kill-by-pattern entries leave four spellings of a kill by name or selector uncovered: kill handed the output of pgrep in a command substitution, pgrep piped into xargs kill, pkill selecting by user with -u, and pkill selecting by terminal with -t. The first two reach the pattern through a second command the entries do not read; the last two are value flags that consume the selector, so no operand remains, and a kill by user is every session of that user. The commit that added the entries named the gap; no record did. Found by review-guard finding 5.
