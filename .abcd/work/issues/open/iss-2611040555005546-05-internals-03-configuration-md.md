---
schema_version: 1
id: "iss-2611040555005546"
slug: "05-internals-03-configuration-md"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Add the relay namespace and the two shared-work files to the inventory."
---

05-internals/03-configuration.md:1292-1302 and the shared-work row at :1310 list what .abcd/ and .abcd/work/ hold. The tree also holds .abcd/relay/ (github-app.json, written by abcd session setup, and the blueprint's files per 04-surfaces/37-session.md:118-121), .abcd/work/publish-surface.txt (the intake runbook's fence) and .abcd/work/brief-review-flags.json (internal/core/lint/briefflags.go). The chapter names none of them.
