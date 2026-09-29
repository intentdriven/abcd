---
schema_version: 1
id: "iss-2609291653241690"
slug: "exifview-follows-only-the-exif-sub-ifd-pointer-0x8769"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/metaview.go"
remedy: "Follow the Exif sub-IFD's MakerNote (0x927c) through a per-vendor reader, Canon first: when IFD0's Make begins Canon, walk the MakerNote as a header-less IFD in the byte order its entry count fits, value offsets relative to the TIFF header and also at the base a matching Canon TIFF footer implies, and read OwnerName (0x0009) as a person tag; read a CR3 file's CMT3 box as the same directory; skip every other vendor's MakerNote with no finding. Every offset and count bounds-checked, the entry count capped, both charged to the walk's shared budgets."
deferred_after: "v0.11.1"
deferral_reason: "a vendor MakerNote parser is its own lane: each vendor lays out its MakerNote differently (Canon keeps TIFF byte order and header-relative offsets, others add their own header or base), so following 0x927c is a per-vendor reader, not a pointer the Exif walk can take; the raw ASCII scan still reads a long or multi-word owner name meanwhile"
resolution: "The EXIF walk reads a Canon MakerNote: when IFD0's Make begins Canon, the MakerNote (0x927c) is walked as a header-less IFD with value offsets relative to the TIFF header, in the byte order its entry count fits, and also at the base a matching Canon TIFF footer implies; OwnerName (0x0009) is read as a person tag, and a CR3 CMT3 box is read as the same directory. Every other vendor's MakerNote is skipped with no finding. Every offset and count is bounds-checked and charged to the walk's shared budgets. The sweep also follows each directory's next-IFD link (IFD1, later TIFF pages), each directory once."
impact: fix
resolved_by:
  commit: "248d6cac4"
---

exifView follows only the Exif sub-IFD pointer 0x8769 (internal/adapter/scanner/metaview.go:161-166) and never the MakerNote 0x927c, so a camera owner's name a vendor writes there (Canon's MakerNote tag 0x0009 OwnerName, in the TIFF byte order with header-relative offsets) is not read as a person tag: a short single-token owner name in a JPEG's or TIFF's Canon MakerNote is dropped as chance. A name of eight or more bytes, or one of several words, is still caught by the raw ASCII scan. Found by review-scanMeta (fixture 'Canon MakerNote OwnerName' scans clean at the integration tip).

## Grounds

- pursued: a short single-token OwnerName in a Canon MakerNote (either byte order, a moved block with its footer, a CR2-shaped TIFF, a CR3 CMT3 box) and a short Artist on a TIFF's second page are reported as real_name, while a short name in another Canon tag, another vendor's MakerNote or a malformed block is not; any of those outcomes differing in makernote_test.go or TestEXIFNextIFDIsRead, or a fuzz crash in FuzzCanonMakerNote, would show it wrong
