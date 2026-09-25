---
schema_version: 1
id: "iss-2608261133218490"
slug: "capture-accepts-quoted-enum-impact-lint-blocks"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "bughunt-round-8"
found_at: "internal/core/capture/validate.go:104"
resolution: "The capture parser keeps a quoted impact's raw token, so validateStrict refuses a quoted legal enum and a quoted empty impact exactly as record-lint's issue_impact_valid does; a differential test pins one verdict per spelling across both readings."
impact: fix
resolved_by:
  commit: "243414a3"
---

capture validateStrict accepts a quoted legal enum impact and a quoted empty impact that record-lint and the release derivation block; the quoted-scalar acceptance is wider than the nulls the round-8 fix closed

## Grounds

- pursued: no record the impact gate blocks loads cleanly through the capture reader; any spelling in the differential table where the two verdicts part would show it wrong
