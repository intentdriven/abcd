---
schema_version: 1
id: "iss-2608300244489638"
slug: "block-spelled-valid-instant-splits-the-gates"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "itd-182 final security review, 2026-08-30"
found_at: "internal/core/lint/schema.go (checkIssueRecordShape, lapsed_at block branch)"
resolution: "A block-spelled lapsed_at is refused on its spelling, not its content: the look-ahead's value no longer reaches ValidLapsedAt, so an indented line that reads as a valid instant is refused exactly as a nested mapping is, matching a reader that builds a mapping from it either way."
impact: internal
---

The block-spelled lapsed_at value read by the new blocks map is fed to the RFC 3339 validator, so a single indented continuation line that spells a valid instant is lint-green while capture's reader builds a map and refuses the record as not a string — the exact gap the block-reader commit set out to close, and its own comment says what the block says is not parsed. A block-spelled scalar is never a string to the reader, so its presence is the finding regardless of content: emit the not-an-instant finding unconditionally when the value came from the block.
