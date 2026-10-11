---
schema_version: 1
id: "iss-2608221456599559"
slug: "scaffold-classify-a-symlinked-target"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "all-dimensions bug-hunt round 7"
found_at: "internal/core/launch/scaffold/scaffold.go"
resolution: "classify folds ELOOP into the non-regular branch and renders a path-free fallback reason."
impact: fix
resolved_by:
  commit: "dc4a22b"
---

scaffold classify: a symlinked target leaf fails ReadGuarded with ELOOP (not ErrNotRegular), so the symlink case named first in the function contract falls through to the generic branch and embeds a raw absolute developer-identity path in the dry-run report and its --json success envelope.