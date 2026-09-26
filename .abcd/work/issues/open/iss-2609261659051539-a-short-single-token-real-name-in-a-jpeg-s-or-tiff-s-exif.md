---
schema_version: 1
id: "iss-2609261659051539"
slug: "a-short-single-token-real-name-in-a-jpeg-s-or-tiff-s-exif"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/scanner.go"
---

A short single-token real name in a JPEG's or TIFF's EXIF Artist tag (IFD0 tag 0x013B) is still dropped by the payload byte scan as chance noise. The byte scan keeps a short name only where a person metadata key stands as text within reach before it (metadataPersonKeys in internal/adapter/scanner/scanner.go: a PDF /Author, XMP dc:creator, a PNG text Author, an OOXML cp:lastModifiedBy), and EXIF stores the Artist tag as a binary IFD entry whose ASCII value sits at an offset with no key text beside it; a UTF-16 PDF string (/Author with a FEFF byte-order mark) is out of reach the same way. Closing it needs the IFD read structurally, the way container.go walks PNG chunks: find the Exif header, read the TIFF byte order, walk IFD0 and scan the Artist (and XPAuthor) value with the text rules. Residue of iss-2609090934372160. Detector: a short banned name in a camera-written EXIF Artist tag, with no XMP packet beside it, is a real_name finding in the payload scan.
