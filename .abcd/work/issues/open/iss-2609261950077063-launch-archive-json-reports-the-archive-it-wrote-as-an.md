---
schema_version: 1
id: "iss-2609261950077063"
slug: "launch-archive-json-reports-the-archive-it-wrote-as-an"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainSite"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/archive.go"
---

launch archive --json reports the archive it wrote as an absolute path: PluginArchive.Path (internal/core/launch/archive.go) is the --out directory made absolute by the front door and joined with the archive name, so archive.path names the developer's home whenever --out sits under it, against the iss-81 rule; the text report's written line prints the same value. Found in the drainPaths sweep of path-bearing --json fields.
