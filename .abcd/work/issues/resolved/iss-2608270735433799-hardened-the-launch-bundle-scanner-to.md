---
schema_version: 1
id: "iss-2608270735433799"
slug: "hardened-the-launch-bundle-scanner-to"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "security-release-2026-08-27"
resolution: "the launch bundle scanner fails closed on an include-selected file it did not scan"
impact: fix
---

Hardened the launch bundle scanner to fail closed on an include-selected file it did not scan.