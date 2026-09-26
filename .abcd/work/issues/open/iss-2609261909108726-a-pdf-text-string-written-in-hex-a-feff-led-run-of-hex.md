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
---

A PDF text string written in hex, a FEFF-led run of hex digits between angle brackets, is not read by the payload byte scan: the hex spelling never puts the UTF-16 bytes in the file, so the UTF-16 view (utf16.go) cannot reach the caller's name in such an /Author or /Creator at any length. Reading it needs no escape grammar: the hex pairs decode to bytes with an offset map, and the decoded bytes feed the existing UTF-16 view. This is the hex half of iss-2609261831352258, split from its octal-literal half. Detector: the caller's name in a UTF-16 /Author written as a hex string is a real_name finding in the payload scan, and a short one is kept only where a person key stands within reach.
