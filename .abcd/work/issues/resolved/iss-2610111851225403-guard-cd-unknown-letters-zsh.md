---
schema_version: 1
id: "iss-2610111851225403"
slug: "guard-cd-unknown-letters-zsh"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "security fix lane G round 4, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/script.go"
remedy: "Give an option word neither shell takes the zsh substitution candidate too (the option word replaced by the operand in the current directory, when that directory exists), keeping unknown only when no candidate is fixed; prove with a Check test over a planted layout whose path contains the option word"
resolution: "an option word neither shell takes now gives zsh's substitution reading, so the scripts after the cd are read in the directory it names"
impact: fix
---

After round 4 of the guard fixes (iss-2610111809142536), a cd whose option word has letters neither shell takes (`cd -x sub`) still leaves the directory unknown, so a following `bash s.sh` is allowed unread, while zsh reads the line as its substitution form (replacing `-x` in the current path with `sub`) when that string occurs in the path. Noted by guard round 4 on 2026-10-11; the same allow-unread held before round 3.
