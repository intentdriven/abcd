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
resolution: "The same-line PEM pattern's separator takes an escaped quote (one to three backslashes), the <br> element in its three spellings and the &#10;, &#13; and &#x0A; entities, and the armour markers fold case in both the pattern and the block consumer, which now share pemBegin and pemEnd. TestRedactPEMMoreOneLineRenderings (JSON array inside a string, escaped once and twice; <br>; <br />; &#10;; &#13;&#10;) and TestRedactPEMLowerCasedBlock (multi-line and one-line) were watched failing before the change and pass after; every earlier PEM test passes unchanged. Base64-of-PEM and a headerless second run remain iss-96's."
impact: fix
resolved_by:
  commit: "cd4c1f78"
---

Four one-line renderings of a private key leak its body past the PEM rules. The same-line pattern's separator (pemPrivateKeyPattern, internal/adapter/scanner/patterns.go) admits a blank, a comma, a quote and an escaped newline but not an escaped quote, so a JSON array serialised inside a JSON string (the elements joined by backslash-quote-comma-backslash-quote) masks the header and keeps the body verbatim on the header's line; an HTML rendering that joins the lines with <br> or the &#10; entity leaks the same way; and a header whose case a tool lowered is not detected at all, by the pattern or by the block consumer (pem.go pemBegin/pemEnd), so its whole body is stored. Found by the scanner-cluster review (items 4 and 5).

## Grounds

- pursued: a private key rendered on one line with escaped-quote, <br> or entity separators, or under lower-cased markers, is masked through its END marker; one of those renderings leaving a body chunk in the redacted text would show it wrong.
