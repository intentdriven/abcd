---
schema_version: 1
id: "iss-2609261205176571"
slug: "scribe-ingest-s-per-line-state-check-splits-the-supplied"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review2-scribe"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/scribe/ingest.go"
resolution: "lineCarries ends a line at LF, CR, CRLF, U+2028 and U+2029, so a CR- or separator-terminated dispositions text is judged line by line"
impact: fix
resolved_by:
  commit: "118b46a2"
---

scribe ingest's per-line state check splits the supplied dispositions on LF alone, so a text whose lines end in CR, U+2028 or U+2029 is one line and the check collapses to a whole-text test: '{0}: rejected ... CR {1}: accepted ... CR' wrote {0} accepted (probed by review2-scribe; CRLF is unaffected). lineCarries in internal/core/scribe/ingest.go must split on CRLF, CR, LF, U+2028 and U+2029.

## Grounds

- pursued: TestScribeIngestHoldsTheStateToTheItemsLineWhateverEndsIt refuses a state carried only by the next line under every terminator and lands the rulings the lines give; a CR, U+2028 or U+2029 subtest writing the next line's state would show it wrong
