---
schema_version: 1
id: "iss-2610252215429343"
slug: "09-provenance-substrate-md-124-calls-the"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S13"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/09-provenance-substrate.md"
remedy: "Change the reference to say the registry is partially regenerable (ADR-13), as § 3 states."
---

09-provenance-substrate.md:124 calls the source-hash registry 'regenerable per the lifecycle taxonomy'; line 52 says it is only partially regenerable and overrides that classification
