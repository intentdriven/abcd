---
schema_version: 1
id: "iss-2609090934372160"
slug: "a-short-single-token-real-name-is"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "adversarial-review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/scanner.go"
related_issues: ["iss-2608291832160371"]
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Read PDF Author/EXIF structurally, or accept a false-positive rate for short single-token names on raw bytes?"
resolution: "Fixed by 7babc8d7, by the anchored context the record names as closing it: the byte scan keeps a short single-token real_name where a metadata key that names a person (author, artist, creator, lastModifiedBy) ends within 96 bytes before it — a PDF /Author, XMP dc:creator, pdf:Author or tiff:Artist across pretty-printed lines, a PNG text Author, an OOXML cp:lastModifiedBy inside a decoded zip — at its hard_fail floor, and drops it elsewhere as before. No threshold was lowered and no false-positive rate accepted: the same token incidental to binary content, a hundred times over or past the key's reach, still raises nothing. The threshold keeps counting bytes, with the reason stated (a chance collision needs that many specific bytes). Pinned by TestShortRealNameOnBytesIsKeptInAMetadataField and the corrected TestRealNameOnBytesByLiteralShape (which had pinned the defect), both watched RED at 211b8853. Residue, captured as iss-2609261659051539: EXIF's binary Artist tag and a UTF-16 PDF string carry no key text and need a structural read."
impact: fix
resolved_by:
  commit: "7babc8d7"
---

A real_name whose literal is a short single token is dropped by the byte scan as chance noise, so a banned name in binary metadata is not a finding there while the same name in text is. byteScanPolicy carries byteScanLongLiteral at eight, and the threshold counts bytes rather than runes, so a four-rune CJK name passes as long while a seven-byte Latin one does not. The consequence is a name in a PDF Author field or an EXIF artist tag reaching a published payload unreported. Decoding containers does not touch this: a decoded region is scanned with the same byte rules, so a short name inside a zip entry is dropped exactly as in raw bytes. The threshold exists to hold down a false-positive rate on binary bytes, where a short Latin token appears by chance, so simply lowering it trades one defect for another. Closing it needs an anchored context that makes a match meaningful rather than incidental, a PDF Author key or an EXIF artist tag read structurally, or an accepted false-positive rate stated as a decision. Detector: a banned short single-token name placed in a PDF Author field must be reported by the payload scan, and a byte-identical run over binary content carrying the same token incidentally must not flood the report. Split out of iss-2608291832160371, whose container half shipped separately; this half was never addressed by it.

## Grounds

- pursued: a short banned name stamped into a document's author metadata is a payload finding while the same token as incidental binary noise is not; a /Author (NAME) PDF passing the payload scan, or a binary with the token scattered through it raising real_name, would show it wrong
