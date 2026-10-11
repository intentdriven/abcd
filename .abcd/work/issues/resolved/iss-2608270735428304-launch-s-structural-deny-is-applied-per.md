---
schema_version: 1
id: "iss-2608270735428304"
slug: "launch-s-structural-deny-is-applied-per"
severity: "critical"
category: "security"
source: "agent-finding"
found_during: "security-release-2026-08-27"
resolution: "launch's structural deny runs per path segment and case-insensitively, so a nested or case-varied denied namespace cannot enter the payload (#335)"
impact: fix
---

launch's structural deny is applied per path segment and case-insensitively, so a nested or case-varied denied namespace cannot enter the payload (GitHub mirror: #335).