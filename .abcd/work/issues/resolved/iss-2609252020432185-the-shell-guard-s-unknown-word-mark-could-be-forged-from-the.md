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
resolution: "An ANSI-C string now ends at its first decoded NUL, as bash ends it, so no decoded byte is the substitution mark and the mark is unforgeable by construction."
impact: fix
resolved_by:
  commit: "83b3de6d"
---

The shell guard's unknown-word mark could be forged from the command line: an ANSI-C escape that decodes to NUL (a hex, octal, unicode or control escape) put the tokenizer's substitution mark into a word after Check had stripped the line's own NUL bytes, so an ANSI-C NUL glued before a blocked command's name made that name an unknown word compared as text, and every blocker allowed. bash ends an ANSI-C string at its first NUL and runs the name that follows. Found by review3-guard finding 1.

## Grounds

- pursued: every escaped-NUL spelling ahead of a blocked command name blocks, and no ANSI-C escape form puts the mark into a word (TestForgedMarkIsTruncatedLikeBash, TestAnsiCEscapeNeverDecodesTheMark); a decoded NUL reaching a word would show it wrong.
