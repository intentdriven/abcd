---
schema_version: 1
id: "iss-2609211905340006"
slug: "dredge-synthesis-output-lands-in-the-memory-store-as-its-own-source-class"
severity: "minor"
category: "future-work-seed"
source: "agent-finding"
found_during: "product thinker interview closing itd-36 as delivered, 2026-09-21"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/memory; itd-25 (dredge, draft)"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Plan the dredge_synthesis memory source class with draft itd-25?"
---

Dredge synthesis output lands in the memory store as its own source class. itd-36 (closed as delivered on 2026-09-21) asked that the dredge synthesiser (itd-25, still a draft) write its synthesised entries to .abcd/memory/<type>_<domain>_<slug>.md with source.class dredge_synthesis and the per-run provenance the lint reads; dredge does not exist yet, so the memory store has no such class and the lint has no rule for it. Wanted, when itd-25 is planned: the class in the closed enum, the writer in the dredge path, and MS001/MS002 reading it as a cross-class input.
