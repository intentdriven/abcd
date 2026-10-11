---
schema_version: 1
id: "iss-2609231011136579"
slug: "abcd-intent-audit-ingest-writes-the"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run 2026-09-23 fidelity audit"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/audit.go"
resolution: "Every write of a review block returns a record ending in a newline (withFinalNewline in upsertReviewBlock and appendToAuditNotes), including a record that reached the ingest without one; the 39 intents the earlier writer left without one gained it in 3d866104. TestIngestedRecordEndsInANewline and TestEveryTreeReviewBlockRoundTrips pin it."
impact: fix
resolved_by:
  commit: "55964951091ab03bd9c07167ab6a26f0cb7df7c7"
---

abcd intent audit ingest writes the shipped intent back with no trailing newline: upsertReviewBlock (internal/core/intent/audit.go:956-980) splits the record on newline, and when the OWED marker's block runs to end of file it sets end = len(lines), so the trailing empty element that carried the file's final newline is dropped from lines[end:] and strings.Join returns a body ending at the last byte of the new block. The appendToAuditNotes path (audit.go:985-1041) re-adds the separator, so only the replace-in-place path (every OWED -> INGESTED transition) loses the newline. Observed on itd-157 and the batch-0 audits before it: the file ends in '- missing: (none)' with no newline. Every other record writer in the tree ends its file with a newline; the fix is to preserve the trailing empty element (or TrimRight and append one newline) in upsertReviewBlock, with a test asserting the ingested record ends in a newline

## Grounds

- pursued: every ingested, dead-lettered or re-ingested intent ends in exactly one newline; an ingest that leaves a record ending on its last block byte would show it wrong
