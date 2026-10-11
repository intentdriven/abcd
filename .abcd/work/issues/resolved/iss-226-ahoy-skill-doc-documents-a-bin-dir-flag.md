---
schema_version: 1
id: "iss-226"
slug: "ahoy-skill-doc-documents-a-bin-dir-flag"
severity: "major"
category: "observation"
source: "agent-observation"
found_during: "ahoy install dogfood"
found_at: "internal/surface"
resolution: "Not a doc drift: --bin-dir exists in current source (internal/surface/cli/cli.go install and uninstall flags); the stale pre-iss-171 repo-root binary rejected it. Rebuilt binary accepts the flag."
impact: internal
---

ahoy skill doc documents a --bin-dir flag for ahoy install/uninstall, but the binary rejects it (unknown flag); the documented escape hatch for an unwritable default bin dir does not exist