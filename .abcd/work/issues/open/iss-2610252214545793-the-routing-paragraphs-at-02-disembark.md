---
schema_version: 1
id: "iss-2610252214545793"
slug: "the-routing-paragraphs-at-02-disembark"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S10"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/02-disembark.md"
remedy: "Replace both paragraphs with a pointer to the commands page's Model-tier routing section (or add the two fields to both lists); the same boilerplate in 04-launch.md and 05-intent.md should get the same edit."
---

The routing paragraphs at 02-disembark.md:296-300 and 23-reading.md:352-356 list seven route-receipt fields; internal/core/oracle/receipt.go:79-84, commands/disembark.md:266-269 and commands/reading.md:291-294 also carry provider_call and unpinned.
