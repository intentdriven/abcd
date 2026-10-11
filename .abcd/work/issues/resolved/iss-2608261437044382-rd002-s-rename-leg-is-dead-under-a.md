---
schema_version: 1
id: "iss-2608261437044382"
slug: "rd002-s-rename-leg-is-dead-under-a"
severity: "major"
category: "bug"
source: "agent-observation"
found_during: "bughunt-b-round-9"
found_at: "scripts/check-reviews.sh"
resolution: "RD002 runs one DMRT history pass, fails closed on git errors, cases harness wired into gate and CI"
impact: fix
resolved_by:
  commit: "15b849d6"
---

RD002's rename leg is dead under a pathspec and a git failure reads as clean