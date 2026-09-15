---
schema_version: 1
id: "iss-2608311230301576"
slug: "nothing-between-the-ingest-verb-s-4-mib-payload-cap-and-the"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "An item whose encoded record would exceed issueschema.RecordReadLimit is refused at item level."
impact: fix
resolved_by:
  intent: "itd-185"
  spec: "spc-63"
---

Nothing between the ingest verb's 4 MiB payload cap and the record write enforces issueschema.RecordReadLimit, so one item with a large body produces a committed reading record every reader in this repo then refuses, including the disposition path that makes the item answerable. The item is durable and permanently undispositionable, which is the split RecordReadLimit's own comment says the single constant exists to prevent.

## Grounds

- pursued: the finding is closed by a test that fails without the change; a later review or mutation run finding the same shape again would show this wrong
