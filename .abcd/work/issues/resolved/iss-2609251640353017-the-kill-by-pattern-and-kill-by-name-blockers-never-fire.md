---
schema_version: 1
id: "iss-2609251640353017"
slug: "the-kill-by-pattern-and-kill-by-name-blockers-never-fire"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/match.go"
resolution: "An unquoted substitution standing as its own word is recorded in segment.subWords and the min_operands count reads each one back as an operand, after value flags have had their pick; TestSubstitutedOperandCountsForMinOperands pins it."
impact: fix
resolved_by:
  commit: "d1011dfe93e4223888523ad7f8d8ec27b1431077"
---

The kill-by-pattern and kill-by-name blockers never fire when the pattern arrives through a command substitution: an unquoted substitution contributes no word under the vanish reading, so the operand count the min_operands constraint reads is zero and pkill or killall with its pattern substituted in is allowed. A substitution standing as its own word is one operand of unknown text for that count. Found by review-guard finding 3.

## Grounds

- pursued: pkill and killall with the pattern substituted in block, while the group and parent selectors fed by a substitution stay allowed and positional compares keep the vanish reading; an allow of a substituted pattern, or a block of a substituted group id, would show it wrong
