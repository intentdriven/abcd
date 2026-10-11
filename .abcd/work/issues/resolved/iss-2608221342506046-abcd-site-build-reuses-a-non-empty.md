---
schema_version: 1
id: "iss-2608221342506046"
slug: "abcd-site-build-reuses-a-non-empty"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "agent-finding"
found_at: "internal/core/site"
resolution: "the site build purges only directories carrying its own marker and refuses foreign non-empty directories instead of deleting them"
impact: fix
---

abcd site build reuses a non-empty output directory without purging, so files from an earlier build survive into the new tree; a purge needs a build-marker guard so it can never remove a directory the build did not write