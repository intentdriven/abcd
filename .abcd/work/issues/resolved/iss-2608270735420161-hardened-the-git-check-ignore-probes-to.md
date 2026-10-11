---
schema_version: 1
id: "iss-2608270735420161"
slug: "hardened-the-git-check-ignore-probes-to"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "security-release-2026-08-27"
resolution: "the git check-ignore probes run under the isolatedGit exec pins"
impact: fix
---

Hardened the git check-ignore probes to run under the isolatedGit exec pins (core.hooksPath, core.fsmonitor).