---
schema_version: 1
id: "iss-2609261133371466"
slug: "record-schema-skips-a-symlinked-declared-bucket-silently"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review2-lintA item 5"
origin: researcher-authored
production_mode: hand-written
resolution: "scanRecordStores emits one record_schema finding, declared bucket '<name>' is a link; nothing in it is checked, for a declared bucket whose DirEntry carries the symlink type bit, and reads nothing behind it. The release cut lists terminal folders with git ls-tree, which never descends a committed link, so it needed no change."
impact: fix
resolved_by:
  commit: "f6894de30f2318f878fba3891032d1e9ffff8ed4"
---

record_schema skips a symlinked declared bucket silently: scanRecordStores (internal/core/lint/schema.go) tests e.IsDir() on the store root's entries, which is false for a symlink DirEntry, so a declared bucket such as open/ or resolved/ that is a link falls to the .md suffix test and is dropped with no finding. A forged record behind the link is not read (nothing out of tree is trusted), but the gate reports nothing about a whole lifecycle state it never checked, the same silent-skip class the seam finding closed. The reading walk and capture's allocator already refuse a symlinked open/ loudly.

## Grounds

- pursued: a linked open/ or resolved/ draws exactly one finding on the bucket while the other bucket is still checked (TestRecordSchemaNamesASymlinkedBucket, red at base); a linked bucket drawing zero findings, or any record behind it surfacing, would show it wrong
