---
schema_version: 1
id: "iss-2609251640452031"
slug: "the-kill-by-pattern-entries-leave-four"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/defaults/guard.json"
resolution: "The guard reads a kill by the search that fed it and by user or terminal: kill-by-search (args_from over pgrep and pidof, through a substitution or a pipe into xargs), pkill-by-owner and killall-by-owner block the attached selector spellings, and the population selectors no longer consume their value in pkill-by-pattern and killall-by-name. The spellings left unseen are iss-2609262259360005."
impact: fix
resolved_by:
  commit: "c3f2e66af"
---

The kill-by-pattern entries leave four spellings of a kill by name or selector uncovered: kill handed the output of pgrep in a command substitution, pgrep piped into xargs kill, pkill selecting by user with -u, and pkill selecting by terminal with -t. The first two reach the pattern through a second command the entries do not read; the last two are value flags that consume the selector, so no operand remains, and a kill by user is every session of that user. The same holds when the selector is a command substitution, which fills the value flag's slot and leaves no operand: `pkill -u $(whoami)` (every session of the caller) and `pkill -g $(cat p)` are allowed by design, and review2-guard finding 8 names both. The commit that added the entries named the gap; no record did. Found by review-guard finding 5.

## Grounds

- pursued: each of the four spellings, and pkill -u $(whoami), now blocks while pgrep alone, kill of a literal or recorded pid, and pkill -g/-P stay allowed (killspellings_test.go); a kill spelling from the record that still allows, or a near-miss control that blocks, would show it wrong
