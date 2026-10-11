---
schema_version: 1
id: "iss-2608221456597173"
slug: "the-adopter-release-scaffold-hardcodes-a"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "all-dimensions bug-hunt round 7"
found_at: "internal/core/launch/scaffold/templates/release.yml.tmpl"
resolution: "The bare release scaffold runs the race leg over the whole module; abcd rendering unchanged."
impact: fix
resolved_by:
  commit: "dc4a22b"
---

The adopter release scaffold hardcodes a go test -race ./internal/... step outside the .Abcd guard, so a bare adopter module with no internal tree fails every release verify run with lstat ./internal/: no such file or directory, wedging the release.