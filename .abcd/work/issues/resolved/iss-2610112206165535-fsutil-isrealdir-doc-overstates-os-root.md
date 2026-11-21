---
schema_version: 1
id: "iss-2610112206165535"
slug: "fsutil-isrealdir-doc-overstates-os-root"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "security fix lane A, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/fsutil/fsutil.go"
remedy: "Correct the comment to what os.Root does, and check that no caller relies on the stronger claim (name any that do)"
resolution: "IsRealDir's doc comment now says os.Root refuses only a path that escapes the root and follows an in-root symlink, and names EnsureRealDirAll/ProbeRealDirAll for a walk that proves each level; no caller relies on the stronger claim (lifeboat/graveyard_lessons.go:292 repeats it in passing but does its own per-segment refusal)."
impact: internal
resolved_by:
  commit: "dd9b76e8ebe4cd6a0ea0566aa2c1622a1c3676b0"
---

The doc comment on fsutil.IsRealDir (internal/fsutil/fsutil.go, ~621) says os.Root refuses symlink traversal at every level, the same false claim iss-2610050259237102 corrected in internal/core/ideate/record.go: os.Root refuses only a path that escapes the root and follows a symlink that stays inside it.

## Grounds

- pursued: CreateExclusiveIn under os.Root follows an ancestor linked inside the root and refuses one linked outside, as the new test pins; a run where the in-root link is refused, or the outward link is followed, would show the corrected comment wrong.
