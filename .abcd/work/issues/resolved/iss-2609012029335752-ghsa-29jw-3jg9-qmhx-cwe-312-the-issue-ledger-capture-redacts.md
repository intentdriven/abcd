---
schema_version: 1
id: "iss-2609012029335752"
slug: "ghsa-29jw-3jg9-qmhx-cwe-312-the-issue-ledger-capture-redacts"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/redact.go"
resolution: "Fixed in the shared scanner: the pem_private_key pattern now reaches over a body on the header line through an END marker and the span is masked whole; scanner.Redact consumes the block body-shaped lines that follow through the END line (bounded at 4096 lines) into one redacted-pem-body placeholder. The slug is derived from the redacted text so the filename carries no body, the resolve note goes through the same primitive, and promote reuses the repaired slug. The consumer covers the common renderings, not every one: a body rendered as log-prefixed or CSV lines, one element per XML tag or JSON array, with a trailing comment or double-escaped separators, the tail beyond the bound, and the RFC4716 armour the header pattern does not detect all still reach the ledger verbatim — iss-2609020127210042. Proved by TestCaptureRedactsPEMBodyFromRecordFilenameNoteAndPromote and the scanner tests TestRedactPEMBlockConsumesBodyThroughEnd, TestRedactPEMOneLineBodyDoesNotSurvive, TestRedactPEMHeaderWithoutEndKeepsProse, TestPEMHeaderIsMaskedWhole, TestRedactPEMBodyConsumerIsBounded and TestRedactPEMBlockWithGutters."
impact: fix
---

GHSA-29jw-3jg9-qmhx (CWE-312): the issue-ledger capture redacts only the PEM private-key BEGIN header and commits the key body. The bundled pem_private_key pattern (scanner.DefaultPatterns) matches the header line alone and scanner.Redact seals matched spans per line, so a pasted key block reaches the committed record with its base64 body and END line verbatim while the JSON reports redacted 1; the slug is derived after redaction, so the body prefix becomes the filename, a one-line key in a capture resolve note leaks the same way, and capture promote inherits the body-prefixed slug into the minted intent. Evidence: capture.redactLedgerText, workflow.go deriveSlug, promote.go. The fix must extend redaction over the body through the END line in the shared primitive, bounded so a truncated block cannot swallow the record, mask the span whole, and leave body, END, filename, note and promoted intent free of key bytes.

## Grounds

- pursued: the header is the only detectable part of a key block, so the redaction has to consume what the header announces; done once in scanner.Redact so the four committed stores cannot drift, bounded by line shape and by a line count so a truncated block never swallows a record
