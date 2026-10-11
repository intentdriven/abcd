---
schema_version: 1
id: "iss-2608270735422516"
slug: "launch-scans-the-payload-on-the"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "security-release-2026-08-27"
resolution: "launch scans the payload on the materialising render path, so a secret in an included file cannot ship unscanned (#328)"
impact: fix
---

launch scans the payload on the materialising render path, so a secret in an included file cannot ship unscanned (GitHub mirror: #328).