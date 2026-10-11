---
schema_version: 1
id: "iss-2608271804181894"
slug: "seven-intents-prd-path-points-at-a"
severity: "nitpick"
category: "observation"
source: "agent-finding"
found_during: "structural consistency review of .abcd/ and docs/ (2026-08-27)"
found_at: ".abcd/development/intents/planned/itd-65-launch-preflight-gate-suite.md"
resolution: "all seven prd_path pointers into the never-existent .abcd/intents/ set to null"
impact: internal
resolved_by:
  commit: "a032377d"
---

seven committed intents carry prd_path values pointing into .abcd/intents/, a directory that has never existed (intents live at .abcd/development/intents/), and no PRD exists at any path because the grill sub-verb that writes them is unshipped (itd-27). Set prd_path: null on all seven — the honest majority form — rather than repointing at a layout no record has adopted.