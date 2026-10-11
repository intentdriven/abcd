---
schema_version: 1
id: "iss-2608270655490198"
slug: "launch-installsurface-go-dirtree-resolve"
severity: "nitpick"
category: "security"
source: "agent-finding"
found_during: "security-cut-agent-flagged-siblings-2026-08-27"
found_at: "internal/core/launch/installsurface.go"
resolution: "installsurface dirTree.resolve delegates to fsutil.ValidRelPath, refusing an embedded a/../../b traversal"
impact: fix
resolved_by:
  commit: "286784c1"
---

launch installsurface.go dirTree.resolve carries a bespoke partial path guard (rejects absolute and a leading ../ but not an embedded a/../../b) instead of the canonical fsutil.ValidRelPath. Read-only payload smoke so lower severity, but it should use the one canonical containment. Flagged by the #488 fix agent.