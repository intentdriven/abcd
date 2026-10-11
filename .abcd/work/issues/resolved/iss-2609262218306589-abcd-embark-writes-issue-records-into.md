---
schema_version: 1
id: "iss-2609262218306589"
slug: "abcd-embark-writes-issue-records-into"
severity: "nitpick"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25: fix2-drainL sibling sweep of iss-2609262143265180"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lifeboat/embark.go"
resolution: "embark takes the target's ledger lock when its plan creates an issue record (and none otherwise, since taking it plants a ledger), and classifies every planned write again under it, so an issue captured at a planned target in the window is a conflict that refuses the whole write."
impact: fix
resolved_by:
  commit: "9c210450c"
---

abcd embark writes issue records into the target repository's ledger outside the ledger lock (internal/core/lifeboat/embark.go: classifyEmbark judges each target path, then writeEmbark writes the set): an issue captured at a path the classification found absent, between the classify and the write, is replaced without a conflict. The ledger-side twin of iss-2609262143265180, which names the intent half.

## Grounds

- pursued: an issue landing at a planned target between embark's plan and its write now refuses the embark and survives (TestEmbarkRejudgesARecordThatLandedAfterThePlan/issues), and a ledger writer during the write waits (TestEmbarkWritesUnderTheLedgerAndIntentLocks); a replaced issue would show it wrong
