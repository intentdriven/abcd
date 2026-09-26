---
schema_version: 1
id: "iss-2609262218306589"
slug: "abcd-embark-writes-issue-records-into-the-target-repository"
severity: "nitpick"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25: fix2-drainL sibling sweep of iss-2609262143265180"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lifeboat/embark.go"
---

abcd embark writes issue records into the target repository's ledger outside the ledger lock (internal/core/lifeboat/embark.go: classifyEmbark judges each target path, then writeEmbark writes the set): an issue captured at a path the classification found absent, between the classify and the write, is replaced without a conflict. The ledger-side twin of iss-2609262143265180, which names the intent half.
