---
schema_version: 1
id: "iss-2608270926030978"
slug: "foldpath-lower-cases-but-never"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "issue-sweep-review-2026-08-27"
found_at: "internal/fsutil/paths.go"
resolution: "FoldPath applies NFC after the fold, so NFC and NFD spellings of a name match on a normalisation-insensitive filesystem"
impact: fix
resolved_by:
  commit: "5f5a6b41"
---

FoldPath lower-cases but never normalises Unicode, while APFS is normalisation-insensitive as well as case-insensitive, so NFC and NFD spellings of one directory compute as non-overlapping in every folded gate — the payload-destination gate, the pack overlap gate and embark's claimed key close the case half of an equivalence class whose normalisation half stays open