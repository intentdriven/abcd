---
schema_version: 1
id: "iss-2609251640353993"
slug: "the-shell-guard-allows-every-blocker-when-a-double-quoted"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
resolution: "The line is read a second time with every followed double-quoted substitution removed, and each shadow that differs sits directly after the command it shadows, so a flag glued beside an empty quoted substitution is the flag; TestFollowedQuotedSubstitutionGluesNoText pins it."
impact: fix
resolved_by:
  commit: "d1011dfe93e4223888523ad7f8d8ec27b1431077"
---

The shell guard allows every blocker when a double-quoted command substitution sits in the same word as the flag: the quoted branch of the tokenizer keeps the substitution text in the word, so a force flag glued after an empty quoted substitution, or split around one, never matches, while bash joins the empty output onto the flag and runs it. The unquoted twin and an empty single-quoted pair already block. Found by review-guard finding 1.

## Grounds

- pursued: every blocker fires on a flag, subcommand or refspec glued beside or split around an empty quoted substitution, as its unquoted twin does, while quoted substitutions in ordinary arguments stay allowed; an allow of any case in TestFollowedQuotedSubstitutionGluesNoText, or a new block in the everyday-allow suites, would show it wrong
