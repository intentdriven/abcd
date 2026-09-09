---
schema_version: 1
id: "iss-2609090934372160"
slug: "a-short-single-token-real-name-is-dropped-on-bytes"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "adversarial-review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/scanner.go"
---

A real_name whose literal is a short single token is dropped by the byte scan as chance noise, so a banned name in binary metadata is not a finding there while the same name in text is. byteScanPolicy carries byteScanLongLiteral at eight, and the threshold counts bytes rather than runes, so a four-rune CJK name passes as long while a seven-byte Latin one does not. The consequence is a name in a PDF Author field or an EXIF artist tag reaching a published payload unreported. Decoding containers does not touch this: a decoded region is scanned with the same byte rules, so a short name inside a zip entry is dropped exactly as in raw bytes. The threshold exists to hold down a false-positive rate on binary bytes, where a short Latin token appears by chance, so simply lowering it trades one defect for another. Closing it needs an anchored context that makes a match meaningful rather than incidental, a PDF Author key or an EXIF artist tag read structurally, or an accepted false-positive rate stated as a decision. Detector: a banned short single-token name placed in a PDF Author field must be reported by the payload scan, and a byte-identical run over binary content carrying the same token incidentally must not flood the report. Split out of iss-2608291832160371, whose container half shipped separately; this half was never addressed by it.
