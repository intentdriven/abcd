---
schema_version: 1
id: "iss-2608270735428527"
slug: "hardened-ahoy-install-to-contain-its"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "security-release-2026-08-27"
resolution: "ahoy install contains its writes within the repository root and refuses a non-real .abcd directory"
impact: fix
---

Hardened ahoy install to contain its writes within the repository root and to refuse a non-real .abcd directory.