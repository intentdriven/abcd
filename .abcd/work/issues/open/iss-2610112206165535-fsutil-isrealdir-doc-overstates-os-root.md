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
---

The doc comment on fsutil.IsRealDir (internal/fsutil/fsutil.go, ~621) says os.Root refuses symlink traversal at every level, the same false claim iss-2610050259237102 corrected in internal/core/ideate/record.go: os.Root refuses only a path that escapes the root and follows a symlink that stays inside it.
