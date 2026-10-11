---
schema_version: 1
id: "iss-2608270500203019"
slug: "walklifeboatfiles-walks-an-untrusted"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "github-ledger-dedup-2026-08-27"
found_at: "internal/core/lifeboat/embark.go"
resolution: "the embark walk reads directories in bounded chunks against a wide untrusted directory (#343)"
impact: fix
---

walkLifeboatFiles walks an untrusted lifeboat with fs.WalkDir (unbounded ReadDir(-1) per directory), so one wide directory materialises and sorts its entire entry list before the maxEmbarkFiles cap can fire. GitHub mirror: #343