---
schema_version: 1
id: "iss-2608270500214438"
slug: "manifest-path-from-version-location-json"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "github-ledger-dedup-2026-08-27"
found_at: "internal/core/launch/lockstep.go"
resolution: "version-location.json manifest_path is path-contained, so the payload stamp cannot write outside the destination (#488)"
impact: fix
---

manifest_path from version-location.json is not path-contained: validateVersionLocation only checks non-empty (unlike launch include patterns which reject .. and absolute), so editManifest can stamp a payload outside the destination. GitHub mirror: #488