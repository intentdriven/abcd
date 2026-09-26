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
---

scribe ingest's per-line state check splits the supplied dispositions on LF alone, so a text whose lines end in CR, U+2028 or U+2029 is one line and the check collapses to a whole-text test: '{0}: rejected ... CR {1}: accepted ... CR' wrote {0} accepted (probed by review2-scribe; CRLF is unaffected). lineCarries in internal/core/scribe/ingest.go must split on CRLF, CR, LF, U+2028 and U+2029.
