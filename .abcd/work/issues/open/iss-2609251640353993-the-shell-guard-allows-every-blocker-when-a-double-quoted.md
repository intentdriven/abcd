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
---

The shell guard allows every blocker when a double-quoted command substitution sits in the same word as the flag: the quoted branch of the tokenizer keeps the substitution text in the word, so a force flag glued after an empty quoted substitution, or split around one, never matches, while bash joins the empty output onto the flag and runs it. The unquoted twin and an empty single-quoted pair already block. Found by review-guard finding 1.
