---
schema_version: 1
id: "iss-2608270735423546"
slug: "urlguard-blocks-the-provider-platform"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "security-release-2026-08-27"
resolution: "urlguard blocks the Azure WireServer platform IP, refusing an SSRF fetch where the metadata IP was already refused (#324)"
impact: fix
---

urlguard blocks the provider platform magic IP so an SSRF fetch to the host-platform endpoint is refused (GitHub mirror: #324).