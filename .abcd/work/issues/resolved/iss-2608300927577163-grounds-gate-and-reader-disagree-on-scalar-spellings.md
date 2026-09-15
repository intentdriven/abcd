---
schema_version: 1
id: "iss-2608300927577163"
slug: "grounds-gate-and-reader-disagree-on-scalar-spellings"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "itd-179 adversarial security review, 2026-08-30"
found_at: "internal/core/lint/schema.go (grounds check)"
resolution: "The gate now reads the grounds scalar the way capture's reader reads it: absence by the null set alone, block-spelled through the blocks map, double-quote-only unquoting so a single-quoted value keeps the quote the reader sees, and an empty value accepted because the reader skips it. A parity table on each side pins the four spellings."
impact: fix
---

The record_schema grounds check passes three spellings capture's strict reader refuses — single-quoted, empty list, and block-spelled — so a committed record is lint-green and silently skipped by every capture surface, and an empty string is refused by lint while the reader accepts it; the lapsed_at block immediately below already handles every one of these shapes. Mirror the lapsed_at treatment: absence by the null set only, block-spelled via the blocks map, single-quoted as the malformed scalar the reader sees, and add the four spellings to the vocabulary test.

## Grounds

- pursued: we expect a gate that refuses exactly what the reader refuses to eliminate the lint-green-but-skipped record class, so a future divergence surfaces as a failing parity row rather than as a record nothing reads
