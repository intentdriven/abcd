---
schema_version: 1
id: "iss-2609020529185438"
slug: "markerre-is-a-byte-pattern-not-a-grammar-the-intent-audit-s"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/audit.go"
---

markerRe is a byte pattern, not a grammar: the intent audit's review marker (an HTML comment naming a state and a receipt id) is matched by a regex over the record's raw bytes, so the ledger's notion of review state is whatever the bytes look like rather than what a markdown reader would parse. It is now line-anchored and whole-line, and termsafe breaks the comment delimiters in every field the ingest writes, so a marker can no longer be forged from a verdict payload. What remains is that the pattern still cannot tell a fenced code block, a quoted example in the brief, or an indented literal from a live marker: a marker-shaped line at column zero inside a fence counts. A durable fix parses the Audit Notes section as markdown, or moves the state out of a comment into frontmatter the record schema owns. Evidence symbol: markerRe in internal/core/intent/audit.go, read by existingMarker, markerState and upsertReviewBlock.
