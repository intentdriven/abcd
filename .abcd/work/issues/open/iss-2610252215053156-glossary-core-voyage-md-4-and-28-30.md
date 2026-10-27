---
schema_version: 1
id: "iss-2610252215053156"
slug: "glossary-core-voyage-md-4-and-28-30"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S11"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/voyage.md"
remedy: "Mark the embark paths staged, define the voyage as the disembark log until embark writes, and drop or rewrite the open question (spec.md:52-57 repeats it)."
---

glossary/core/voyage.md:4 and :28-30 present embark/provenance.json and embark/from/<timestamp>/ as part of what the voyage records ('every disembark and embark run'); 05-internals/03-configuration.md:497-500 says only the disembark log ships and the embark side is staged (03-embark.md:272 'not built yet'). Its Open question (:43-48) is also stale: oracle.md no longer says 'during a voyage', and phase.md:83 calls the sense retired.
