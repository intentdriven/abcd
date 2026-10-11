---
schema_version: 1
id: "iss-2608261040378346"
slug: "issue-resolution-gate-cwd-errexit-scan"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "bughunt-a/round-7"
found_at: "scripts/check-issue-resolution.sh"
resolution: "cd to the repo root; non-fatal iss-id extraction; RS002 reads changed-record frontmatter via a helper shared with RS003. Three new cases pin it."
impact: fix
resolved_by:
  commit: "47efe8e82d3d45d3f8f968d55c57cc8655dcb350"
---

check-issue-resolution.sh scans cwd-relative and dies silently on a non-iss path, and RS002 reachability-checks a commit: line in a record body. Three sibling holes in the new issue-resolution gate: (1) it never cd's to the repo toplevel like check-reviews.sh does, so a git pathspec run from a subdirectory matches nothing and the gate reports a clean pass having scanned zero records — RS003, the drift detector with no other check, passes vacuously; (2) under set -euo pipefail the id extractors end a matched case with 'basename | grep ^iss-', which exits 1 on a non-iss filename, so when that is the last diff entry the while-loop and the closed= assignment inherit the failure and the script dies exit 1 with no diagnostic, RS001/RS002 never running; (3) RS002 greps the raw range diff for a '+  commit:' line with no frontmatter boundary, so a commit: example in a record's prose body is reachability-checked and raises a false violation — the exact boundary RS003 already draws. Fix: cd to toplevel; make the id extraction non-fatal; read the frontmatter per changed record via a helper RS003 shares.