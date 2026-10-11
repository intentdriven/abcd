---
schema_version: 1
id: "iss-2608211432384091"
slug: "readme-overclaims-every-hook-self"
severity: "nitpick"
category: "documentation"
source: "user-observation"
found_during: "bughunt-b/round-5"
found_at: "README.md"
resolution: "Reword the README hook-bootstrap paragraph to state SessionEnd never downloads and why"
impact: fix
resolved_by:
  commit: "1c3ae21"
---

README claims every non-SessionStart hook self-bootstraps, but the SessionEnd hook deliberately never downloads the binary (pinned by TestSessionEndNeverBootstraps); the universal is false for SessionEnd