---
schema_version: 1
id: "iss-2608270735422415"
slug: "intent-redacts-caller-text-before"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "security-release-2026-08-27"
resolution: "intent redacts caller text before persisting a draft (#486)"
impact: fix
---

intent redacts caller text before persisting a draft, so a secret or home path is not committed (GitHub mirror: #486).