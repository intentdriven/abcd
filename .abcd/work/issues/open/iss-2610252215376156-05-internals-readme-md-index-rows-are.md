---
schema_version: 1
id: "iss-2610252215376156"
slug: "05-internals-readme-md-index-rows-are"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S13"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/README.md"
remedy: "Rewrite the four descriptors to match their pages, or cut each row to its file and title."
---

05-internals/README.md index rows are stale: row 3 claims 'adapter-registry config' (03-configuration.md:404 says there is none), row 1 says every agent declares JSON I/O (01-agents.md:55-59 says four do not), row 7 credits principle-distiller with curating memory, and row 10 names 'The check gates', now lint site (7b579ca5e)
