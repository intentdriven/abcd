---
schema_version: 1
id: "iss-2610252215395382"
slug: "07-memory-md-42-says-source-class-is"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S13"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/07-memory.md"
remedy: "Say the backfill runs on the first memory verb that touches the store."
refines: [iss-2609091956001547]
---

07-memory.md:42 says source.class is backfilled 'on first /abcd:ahoy after itd-36 ships'; itd-36:163 records that ahoy has no memory step and that the backfill runs when a memory verb touches the store
