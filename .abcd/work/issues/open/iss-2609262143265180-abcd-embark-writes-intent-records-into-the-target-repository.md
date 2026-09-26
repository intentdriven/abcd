---
schema_version: 1
id: "iss-2609262143265180"
slug: "abcd-embark-writes-intent-records-into-the-target-repository"
severity: "nitpick"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25: drainL sibling sweep of iss-2609261941039204"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lifeboat/embark.go"
---

abcd embark writes intent records into the target repository outside the intent store's lock (internal/core/lifeboat/embark.go: classifyEmbark judges each target path, then writeEmbark writes the set): an intent created at a path the classification found absent, between the classify and the write, is replaced without a conflict. The window is a same-id create during an embark, so it is narrow, but it is the one intent-record writer the drainL sweep found outside intent.WithMintLock.
