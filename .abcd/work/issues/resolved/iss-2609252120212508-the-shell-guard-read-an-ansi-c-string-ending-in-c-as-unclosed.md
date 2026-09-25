---
schema_version: 1
id: "iss-2609252120212508"
slug: "the-shell-guard-read-an-ansi-c-string-ending-in-c-as-unclosed"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
resolution: "An ANSI-C string is closed before it is decoded, as bash reads it, so no escape reaches its closing quote; and the hook blocks a line the guard cannot split (command-unparsable) instead of failing open."
impact: fix
resolved_by:
  commit: "c72a73fa"
---

The shell guard decoded an ANSI-C \c escape as taking the next byte whatever it was, so a string ending in \c swallowed its own closing quote and the line did not parse, and the pre-tool-use hook maps a parse error to fail-open: a blocked command after such a string ran unchecked on every blocker. bash finds the closing quote first and decodes after, and reads \c followed by a backslash pair as one escape. Behind it, a parse error failing open on the hook is a bypass by construction wherever the tokenizer misreads bash (review4-guard finding 2).

## Grounds

- pursued: a blocked command after an ANSI-C string ending in any escape blocks, and no tokenizer error runs a line on the hook; a string bash closes that the guard reads as open, or a hook exit other than 2 on an unparsable line, would show it wrong
