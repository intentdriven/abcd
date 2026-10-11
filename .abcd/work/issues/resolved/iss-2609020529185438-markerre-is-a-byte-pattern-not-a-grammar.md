---
schema_version: 1
id: "iss-2609020529185438"
slug: "markerre-is-a-byte-pattern-not-a-grammar"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/audit.go"
resolution: "The marker is matched on one line at a time, and only on live lines of the live Audit Notes section as condition.AuditNotes reads them through mdrecord.Mask, so a marker in a fence (backtick or tilde), an HTML comment span or another section is not state; the writer, the dead-letter reason reader and condition.ReadDispositions read through the same mask. TestOnlyALiveMarkerCounts, TestAFencedMarkerIsNotASolicitation, TestAppendLandsInTheLiveAuditNotes and TestReadDispositionsReadsOnlyLiveBlocks pin it; moving the state into frontmatter remains the durable option and is not attempted."
impact: fix
resolved_by:
  commit: "55964951091ab03bd9c07167ab6a26f0cb7df7c7"
---

markerRe is a byte pattern, not a grammar: the intent audit's review marker (an HTML comment naming a state and a receipt id) is matched by a regex over the record's raw bytes, so the ledger's notion of review state is whatever the bytes look like rather than what a markdown reader would parse. It is now line-anchored and whole-line, and termsafe breaks the comment delimiters in every field the ingest writes, so a marker can no longer be forged from a verdict payload. What remains is that the pattern still cannot tell a fenced code block, a quoted example in the brief, or an indented literal from a live marker: a marker-shaped line at column zero inside a fence counts. A durable fix parses the Audit Notes section as markdown, or moves the state out of a comment into frontmatter the record schema owns. Evidence symbol: markerRe in internal/core/intent/audit.go, read by existingMarker, markerState and upsertReviewBlock.

## Grounds

- pursued: only a marker a markdown reader parses as a live comment in Audit Notes carries review state; a fenced or commented marker that solicits, routes or silences a verdict would show it wrong
