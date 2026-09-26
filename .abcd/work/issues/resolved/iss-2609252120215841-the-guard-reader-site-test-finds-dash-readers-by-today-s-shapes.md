---
schema_version: 1
id: "iss-2609252120215841"
slug: "the-guard-reader-site-test-finds-dash-readers-by-today-s-shapes"
severity: "minor"
category: "tech-debt"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/unknownsites_test.go"
resolution: "The reader-site test finds a word's dash reader by any mention of a dash (a rune, a dash-led string, a named constant holding one), not by the shape of the test around it; five functions it had missed are listed exempt."
impact: internal
resolved_by:
  commit: "9124932d"
---

The reader-site test of the shell guard (unknownsites_test.go) detects a function that reads a word dash only by the shapes the package uses today (strings.HasPrefix with a dash literal, a comparison with a dash); a reader written as a switch on the first byte, strings.IndexByte, bytes.HasPrefix or a named dash constant would pass unlisted (review4-guard finding 5).

## Grounds

- pursued: a reader of a word's dash written in any spelling is found and must be listed; a spelling the meta-test feeds it that it misses would show it wrong
