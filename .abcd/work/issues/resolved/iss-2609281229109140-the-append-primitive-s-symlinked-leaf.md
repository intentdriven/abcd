---
schema_version: 1
id: "iss-2609281229109140"
slug: "the-append-primitive-s-symlinked-leaf"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainI"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/fsutil/fsutil.go"
resolution: "openAppendIn now re-Lstats the leaf after every open and refuses unless it is a regular file that SameFile-matches the opened descriptor, so a symlink planted between an empty pre-open Lstat and the open is refused and its target untouched; the comment names Lstat+SameFile as the mechanism, not O_NOFOLLOW."
impact: fix
resolved_by:
  commit: "35b18357f"
---

The append primitive's symlinked-leaf refusal has a race: fsutil.openAppendIn opens through os.Root, where O_NOFOLLOW is inert (os.Root follows an in-root leaf symlink on ELOOP), so the refusal rests on the pre-open Lstat and the SameFile check against the opened descriptor, and SameFile is skipped when the pre-open Lstat saw nothing. A symlink planted between an ENOENT Lstat and the open is followed and appended through; the comment claims every open carries O_NOFOLLOW as though that refused it. Needs a same-uid racer in a microsecond window.

## Grounds

- pursued: AppendLineIn refuses a leaf linked after its Lstat with ErrNotRegular and leaves the target unchanged (TestAppendLineInRefusesALeafLinkedAfterItsLstat, RED before the post-open Lstat, GREEN after); an append through a link planted in that window succeeding would show it wrong. A hard link still passes SameFile and is out of scope.
