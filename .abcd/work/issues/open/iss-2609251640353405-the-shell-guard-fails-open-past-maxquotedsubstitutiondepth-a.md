---
schema_version: 1
id: "iss-2609251640353405"
slug: "the-shell-guard-fails-open-past-maxquotedsubstitutiondepth-a"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
---

The shell guard fails open past maxQuotedSubstitutionDepth: a command substitution nested inside double quotes nine levels deep is left as literal text rather than read, so a blocked command at that depth is allowed, while bash runs the innermost command at any depth. Past the depth the guard must refuse, as the brace-group, here-document and bang-alias budgets do. Found by review-guard finding 2.
