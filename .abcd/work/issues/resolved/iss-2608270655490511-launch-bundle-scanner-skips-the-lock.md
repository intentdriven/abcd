---
schema_version: 1
id: "iss-2608270655490511"
slug: "launch-bundle-scanner-skips-the-lock"
severity: "nitpick"
category: "security"
source: "agent-finding"
found_during: "security-cut-agent-flagged-siblings-2026-08-27"
found_at: "internal/adapter/scanner/scanner.go"
resolution: "the scanner no longer skips .lock as binary, so a committed text lockfile is scanned in a launch bundle"
impact: internal
resolved_by:
  commit: "e383293e"
---

launch bundle scanner skips the .lock extension as if binary, but lockfiles are text and can carry secrets, so an included .lock ships unscanned (same class as the .svg gap the launch-payload fix closed, but pre-existing and out of that scope). Flagged by the launch-payload fix agent.