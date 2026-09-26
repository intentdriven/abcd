---
schema_version: 1
id: "iss-2609261831352258"
slug: "a-pdf-text-string-written-in-hex-a-feff-led-run-of-hex"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/utf16.go"
deferred_after: "v0.11.0"
deferral_reason: "decoding PDF string syntax is a piece of a PDF reader, beyond the parser-free decodes lane drainS3 was scoped to: it needs each hex string's angle brackets and each literal string's balanced parentheses found, the hex pairs and the backslash escapes (octal, the named controls and line continuations) decoded with every byte mapped back to its raw offset, and the decoded bytes handed to the UTF-16 view, which then reaches the name"
---

A PDF text string written in hex (a FEFF-led run of hex digits between angle brackets) or with octal escapes (a backslash-376, backslash-377 opening inside parentheses), the forms several common PDF writers use for a non-ASCII document field, is read by neither the payload byte scan nor its UTF-16 view: neither spelling puts the UTF-16 bytes in the file, so the caller's name in such an /Author or /Creator raises nothing at any length. Reading them is a decode of the string syntax ahead of the UTF-16 view (hex pairs, and the PDF literal-string escapes), in the shape of the percent and JSON-escape views, not a format parser. Detector: the caller's name in a UTF-16 /Author written in hex or with octal escapes is a real_name finding in the payload scan.

**Deferred (2026-09-26, autonomous run A, lane drainS3).** Found by the
sibling sweep of iss-2609261827066511, whose UTF-16 view reads a
byte-order-marked run only where the file holds the UTF-16 bytes themselves.
