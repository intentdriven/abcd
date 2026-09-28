---
schema_version: 1
id: "iss-2609281229109140"
slug: "the-append-primitive-s-symlinked-leaf-refusal-has-a-race"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainI"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/fsutil/fsutil.go"
---

The append primitive's symlinked-leaf refusal has a race: fsutil.openAppendIn opens through os.Root, where O_NOFOLLOW is inert (os.Root follows an in-root leaf symlink on ELOOP), so the refusal rests on the pre-open Lstat and the SameFile check against the opened descriptor, and SameFile is skipped when the pre-open Lstat saw nothing. A symlink planted between an ENOENT Lstat and the open is followed and appended through; the comment claims every open carries O_NOFOLLOW as though that refused it. Needs a same-uid racer in a microsecond window.
