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
resolution: "Past maxQuotedSubstitutionDepth, and for a quoted substitution whose text does not tokenize, the tokenizer raises substitutionUnread and Check blocks under the reserved id substitution-unread; TestQuotedSubstitutionPastTheDepthFailsClosed pins it."
impact: fix
resolved_by:
  commit: "d1011dfe93e4223888523ad7f8d8ec27b1431077"
---

The shell guard fails open past maxQuotedSubstitutionDepth: a command substitution nested inside double quotes nine levels deep is left as literal text rather than read, so a blocked command at that depth is allowed, while bash runs the innermost command at any depth. Past the depth the guard must refuse, as the brace-group, here-document and bang-alias budgets do. Found by review-guard finding 2.

## Grounds

- pursued: a command nested inside double-quoted substitutions past the depth budget is blocked whatever it is, and within the budget nothing changes; an allow at nine or more levels, or a block of a harmless nest at eight, would show it wrong
