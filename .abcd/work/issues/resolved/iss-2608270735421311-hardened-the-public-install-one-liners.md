---
schema_version: 1
id: "iss-2608270735421311"
slug: "hardened-the-public-install-one-liners"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "security-release-2026-08-27"
resolution: "the public install one-liners match bootstrap's curl lockdown"
impact: fix
---

Hardened the public install one-liners to match bootstrap's curl lockdown (-q first, proxy/CA scrub, proto pins).