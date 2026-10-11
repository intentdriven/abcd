---
schema_version: 1
id: "iss-2608261133091276"
slug: "resolution-gate-shallow-clone-false-red"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "bughunt-round-8"
found_at: "scripts/check-issue-resolution.sh:108"
resolution: "both history gates refuse a shallow checkout with exit 2 and name the remedy; the full-history prerequisite is stated in the Makefile and the lint chapter"
impact: fix
resolved_by:
  commit: "4cd7f40f"
---

RS002 and RS003 report an unfetched commit as names-no-commit on a shallow checkout, 85 false blockers on a clean tree, and check-reviews RD002 passes vacuously on the same input; neither script guards for shallow history