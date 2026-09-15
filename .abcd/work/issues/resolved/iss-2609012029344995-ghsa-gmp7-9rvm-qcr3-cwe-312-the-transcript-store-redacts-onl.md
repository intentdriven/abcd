---
schema_version: 1
id: "iss-2609012029344995"
slug: "ghsa-gmp7-9rvm-qcr3-cwe-312-the-transcript-store-redacts-onl"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/history/history.go"
resolution: "Fixed in the shared scanner: the pem_private_key span now covers a same-line body through its END marker, is masked whole, and scanner.Redact consumes the following body-shaped lines through the END line into one placeholder, bounded at 4096 lines, so the stored record carries no body line in the renderings the shape rule covers and keeps the prose after the block: TestCaptureRedactsPEMBody. Residual stated in pem.go: stage two still cannot see a headerless base64 line (iss-96 tracks the entropy residue), so a body line the shape rule declines survives — a body rendered as log-prefixed or CSV lines, one element per XML tag or JSON array, with a trailing comment or double-escaped separators, the tail beyond the bound, and the RFC4716 armour the header pattern does not detect are the shapes that still store verbatim (iss-2609020127210042); the world-readable record mode is iss-2609012029343438."
impact: fix
---

GHSA-gmp7-9rvm-qcr3 (CWE-312): the transcript store redacts only the PEM BEGIN header and stores the key body verbatim. history.Capture stage one scans with the header-only pem_private_key pattern and scanner.Redact seals the header span alone; stage two rescans with the same detector, the fingerprinted header no longer matches, and the write proceeds with every body line and the END line in the record while the frontmatter counts one secret. Header and body on one line leak the same way because byte-span sealing stops at the header. Evidence: history.Capture, scanner.DefaultPatterns pem_private_key, scanner.Redact. The fix must consume the body through the END line in the shared primitive with a bound, mask the span whole, and keep the prose after the block; the record mode (0644) is a separate observation.

## Grounds

- pursued: the header is the only detectable part of a key block, so the redaction has to consume what the header announces; done once in scanner.Redact so the four committed stores cannot drift, bounded by line shape and by a line count so a truncated block never swallows a record
