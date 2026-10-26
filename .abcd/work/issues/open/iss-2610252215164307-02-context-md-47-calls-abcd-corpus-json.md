---
schema_version: 1
id: "iss-2610252215164307"
slug: "02-context-md-47-calls-abcd-corpus-json"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S12"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/02-context.md"
remedy: "Mark the corpus list at 02-context.md:47 as staged, with the three repos as its intended seed, so the SSOT the platform page names agrees with naming.md:80."
---

02-context.md:47 calls .abcd/corpus.json the 'user-maintained' validation-corpus list, and 01-platform.md:41 makes that page its SSOT. The file is absent, and 04-naming.md:80 says the name is reserved and 'not yet in the tree'.
