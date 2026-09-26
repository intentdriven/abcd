---
schema_version: 1
id: "iss-2609261848338673"
slug: "launch-render-json-and-launch-ship-s-payload-line-reports"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/render.go"
---

launch ship --json reports the release payload's destination as an absolute path in payload.dest, and the text report prints it on its payload line: PayloadRenderResult.Dest (internal/core/launch/render.go) is the symlink-resolved destination, so a destination under the home names the developer in machine output, against the iss-81 rule the site and lifeboat verbs are held to.
