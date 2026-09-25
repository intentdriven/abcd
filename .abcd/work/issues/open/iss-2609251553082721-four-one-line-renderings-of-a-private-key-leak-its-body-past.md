---
schema_version: 1
id: "iss-2609251553082721"
slug: "four-one-line-renderings-of-a-private-key-leak-its-body-past"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/patterns.go"
---

Four one-line renderings of a private key leak its body past the PEM rules. The same-line pattern's separator (pemPrivateKeyPattern, internal/adapter/scanner/patterns.go) admits a blank, a comma, a quote and an escaped newline but not an escaped quote, so a JSON array serialised inside a JSON string (the elements joined by backslash-quote-comma-backslash-quote) masks the header and keeps the body verbatim on the header's line; an HTML rendering that joins the lines with <br> or the &#10; entity leaks the same way; and a header whose case a tool lowered is not detected at all, by the pattern or by the block consumer (pem.go pemBegin/pemEnd), so its whole body is stored. Found by the scanner-cluster review (items 4 and 5).
