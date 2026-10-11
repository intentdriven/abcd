---
schema_version: 1
id: "iss-2608270500202786"
slug: "walklifeboatfiles-counts-only-regular"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "github-ledger-dedup-2026-08-27"
found_at: "internal/core/lifeboat/embark.go"
resolution: "the embark walk is bounded, so a directory-only untrusted lifeboat cannot exhaust memory (#337)"
impact: fix
---

walkLifeboatFiles counts only regular files, so an untrusted lifeboat made purely of directories never trips maxEmbarkFiles: a deep directory chain walks unbounded and can OOM the CLI. GitHub mirror: #337