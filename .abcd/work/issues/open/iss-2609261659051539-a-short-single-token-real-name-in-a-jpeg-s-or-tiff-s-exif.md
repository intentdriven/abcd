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
deferred_after: "v0.11.0"
deferral_reason: "needs a format reader the scanner does not have: the EXIF Artist (0x013B) and Copyright (0x8298) values are ASCII at an offset only a binary IFD entry names, so keeping a short name there means finding the TIFF header (a JPEG APP1 Exif segment, a PNG eXIf chunk, or a TIFF file's own header), reading its byte order, walking IFD0's entry count and twelve-byte entries with every offset bounds-checked against the segment, and scanning the Artist, Copyright and XPAuthor (UTF-16LE with no byte-order mark) values with the text rules; no parser-free reach tells those bytes from chance, and the UTF-16 half needed none, so it shipped on its own as iss-2609261827066511"
---

A short single-token real name in a JPEG's or TIFF's EXIF Artist tag (IFD0 tag 0x013B) is still dropped by the payload byte scan as chance noise. The byte scan keeps a short name only where a person metadata key stands as text within reach before it (metadataPersonKeys in internal/adapter/scanner/scanner.go: a PDF /Author, XMP dc:creator, a PNG text Author, an OOXML cp:lastModifiedBy), and EXIF stores the Artist tag as a binary IFD entry whose ASCII value sits at an offset with no key text beside it; a UTF-16 PDF string (/Author with a FEFF byte-order mark) is out of reach the same way. Closing it needs the IFD read structurally, the way container.go walks PNG chunks: find the Exif header, read the TIFF byte order, walk IFD0 and scan the Artist (and XPAuthor) value with the text rules. Residue of iss-2609090934372160. Detector: a short banned name in a camera-written EXIF Artist tag, with no XMP packet beside it, is a real_name finding in the payload scan.

**Split and deferred (2026-09-26, autonomous run A, lane drainS3).** The
UTF-16 half of this record needs no format parser: a byte-order-marked
UTF-16 run is decoded and scanned as a view of the bytes, which
iss-2609261827066511 shipped. What stays here is the EXIF half, and it needs
an IFD reader; the deferral names what that reader has to do. BOM-less
UTF-16 (EXIF XPAuthor, legacy binary documents) stays with it, since only
the structure around such a run tells it from chance bytes.
