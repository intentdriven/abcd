---
schema_version: 1
id: "iss-2609252020432185"
slug: "the-shell-guard-s-unknown-word-mark-could-be-forged-from-the"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
---

The shell guard's unknown-word mark could be forged from the command line: an ANSI-C escape that decodes to NUL (a hex, octal, unicode or control escape) put the tokenizer's substitution mark into a word after Check had stripped the line's own NUL bytes, so an ANSI-C NUL glued before a blocked command's name made that name an unknown word compared as text, and every blocker allowed. bash ends an ANSI-C string at its first NUL and runs the name that follows. Found by review3-guard finding 1.
