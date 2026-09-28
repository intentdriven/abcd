---
schema_version: 1
id: "iss-2609261909108726"
slug: "a-pdf-text-string-written-in-hex-a-feff-led-run-of-hex"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainS3"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/utf16.go"
resolution: "pdfHexView decodes the hex pairs of every PDF hex string to bytes, each mapped to the raw offset of its first digit, and hands a byte-order-marked result to the existing UTF-16 view, so the caller's name in a hex-spelled /Author or /Creator is a real_name finding in the payload scan and a short one is kept only where a person key stands within reach. Tests: TestPDFHexTextStringIsRead, and TestUTF16ViewWorkIsLinear with the hex shapes. The octal-literal spelling stays deferred on iss-2609261831352258."
impact: fix
resolved_by:
  commit: "d010efe27"
---

A PDF text string written in hex, a FEFF-led run of hex digits between angle brackets, is not read by the payload byte scan: the hex spelling never puts the UTF-16 bytes in the file, so the UTF-16 view (utf16.go) cannot reach the caller's name in such an /Author or /Creator at any length. Reading it needs no escape grammar: the hex pairs decode to bytes with an offset map, and the decoded bytes feed the existing UTF-16 view. This is the hex half of iss-2609261831352258, split from its octal-literal half. Detector: the caller's name in a UTF-16 /Author written as a hex string is a real_name finding in the payload scan, and a short one is kept only where a person key stands within reach.

## Grounds

- pursued: a name in a hex-spelled PDF text string is found at linear cost without new findings on chance bytes; a launch dry-run that reports a new finding on this tree's assets, or a hex-spelled /Author name the scan still passes, would show it wrong
