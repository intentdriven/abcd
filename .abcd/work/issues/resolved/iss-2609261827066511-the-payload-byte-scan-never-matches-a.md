---
schema_version: 1
id: "iss-2609261827066511"
slug: "the-payload-byte-scan-never-matches-a"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/scanner.go"
resolution: "scanBytes decodes every run of UTF-16 text a byte-order mark opens, big- or little-endian, into one view scanned with the byte rules, and re-homes each finding onto the raw offset of its first code unit, so the caller's name, home path or email written as UTF-16 is a hard_fail finding, a short name behind a PDF /Author key is kept by the person-key rule, and a short name with no key in reach is still dropped. Tests: TestUTF16TextOnBytesIsRead, TestUTF16TextOnBytesKeepsTheShortNameRule, TestUTF16ViewWorkIsLinear. A run with no byte-order mark stays unread and belongs with the EXIF residue in iss-2609261659051539."
impact: fix
resolved_by:
  commit: "75ce1ac4"
---

The payload byte scan never matches a value written as UTF-16 text behind a byte-order mark, whatever its length: a PDF text string that opens with FE FF (the /Author, /Creator or /Title of a document with a non-ASCII field) or a little-endian string that opens with FF FE interleaves a zero byte with every letter, so the caller's real name, home path and email in it raise nothing and the file publishes. This is the UTF-16 half of iss-2609261659051539, split from its EXIF half, which needs an IFD reader; it reaches long values too, not only the short names that record names. Detector: the caller's name, home path or email written as byte-order-marked UTF-16 in a skip-listed file is a hard_fail finding in the payload scan, and a short name there is kept only where a person key stands within reach, as in plain bytes.

## Grounds

- pursued: a value written as byte-order-marked UTF-16 in a skip-listed file is found by the payload scan at linear cost without new false positives; a launch dry-run that reports a new finding on this tree's own assets, or a marked UTF-16 name the scan still passes, would show it wrong
