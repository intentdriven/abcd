---
schema_version: 1
id: "iss-2609012029335717"
slug: "ghsa-5qr6-f78x-g2cx-cwe-312-the-memory-store-redacts-only-th"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/memory/redact.go"
resolution: "Fixed in the shared scanner: the pem_private_key span now covers a same-line body through its END marker, is masked whole, and scanner.Redact consumes the following body-shaped lines through the END line into one placeholder, bounded at 4096 lines. The page body and the kept original go through that one primitive, so neither carries a body line in the renderings the shape rule covers: TestIngestRedactsPEMBodyFromPageAndKeptOriginal. It does not cover every one: a body rendered as log-prefixed or CSV lines, one element per XML tag or JSON array, with a trailing comment or double-escaped separators, the tail beyond the bound, and the RFC4716 armour the header pattern does not detect all still reach the page and the kept original verbatim — iss-2609020127210042. The kept original is redacted in place, as every other text-source secret already is, rather than refused; a refusal would be a new policy for one pattern and the redacted copy carries no key bytes."
impact: fix
---

GHSA-5qr6-f78x-g2cx (CWE-312): the memory store redacts only the PEM BEGIN header and stores the key body. memory.storeRedactor.redactText scans with the header-only pem_private_key pattern, scanner.Redact masks that one span, and the stage-two BlockingResidual rescan re-runs the same rule over a fingerprinted header and is clean by construction; a text source with a key block lands in the page body and in the --keep-original copy (redactOriginalBytes) with body and END line verbatim. Evidence: memory/redact.go redactText and redactOriginalBytes. The fix is the shared block consumer in scanner.Redact so memory, history and capture cannot drift; the page body and the kept original must carry no body line.

## Grounds

- pursued: the header is the only detectable part of a key block, so the redaction has to consume what the header announces; done once in scanner.Redact so the four committed stores cannot drift, bounded by line shape and by a line count so a truncated block never swallows a record
