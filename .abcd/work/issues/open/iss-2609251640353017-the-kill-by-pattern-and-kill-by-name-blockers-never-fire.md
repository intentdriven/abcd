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
---

The kill-by-pattern and kill-by-name blockers never fire when the pattern arrives through a command substitution: an unquoted substitution contributes no word under the vanish reading, so the operand count the min_operands constraint reads is zero and pkill or killall with its pattern substituted in is allowed. A substitution standing as its own word is one operand of unknown text for that count. Found by review-guard finding 3.
