---
schema_version: 1
id: "iss-2609252214137586"
slug: "the-shell-guard-ends-an-unquoted-here-document-body-at-a"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
resolution: "An unquoted here-document body is read by logical line: a physical line ending in an odd number of backslashes joins the next before the delimiter compare, as bash splices it; an even count and a quoted delimiter join nothing."
impact: fix
resolved_by:
  commit: "769fd3da"
---

The shell guard ends an unquoted here-document body at a line that bash joins onto the line before it. In a body whose delimiter is unquoted, bash joins a physical line ending in an odd number of backslashes with the next line before it compares the delimiter (read_secondary_line with its backslash-newline splice), so x-backslash then EOF is the one body line xEOF and the body goes on. skipHeredocBodies (internal/core/guard/tokenize.go) compared each physical line, ended the body at that EOF, and read what followed as command text, where a single-quoted span or a # comment hides a substitution the body runs: a blocked command in $( ) on the next line allowed, in the plain, <<- and three-backslash twins (review5-guard finding 1, verified on bash 3.2 and 5.3 with a neutral word). The fix report's claim that the guard can only over-read such a body is false.

## Grounds

- pursued: a body line ending in an odd count of backslashes followed by the delimiter line keeps the body open, so a substitution after it is read as body and blocks; it would be shown wrong by any shape where bash ends the body at a line the guard does not, or the reverse, with a neutral word on bash 3.2 and 5.3
