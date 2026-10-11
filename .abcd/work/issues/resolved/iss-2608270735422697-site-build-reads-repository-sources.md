---
schema_version: 1
id: "iss-2608270735422697"
slug: "site-build-reads-repository-sources"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "security-release-2026-08-27"
resolution: "site build reads repository sources through an os.Root, so a committed directory-symlink ancestor is not followed (#487)"
impact: fix
---

site build reads repository sources through an os.Root, so a committed directory-symlink ancestor is not followed (GitHub mirror: #487).