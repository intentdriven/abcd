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
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (planning owed: the dredge_synthesis memory source class waits on itd-25 (draft)). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

Dredge synthesis output lands in the memory store as its own source class. itd-36 (closed as delivered on 2026-09-21) asked that the dredge synthesiser (itd-25, still a draft) write its synthesised entries to .abcd/memory/<type>_<domain>_<slug>.md with source.class dredge_synthesis and the per-run provenance the lint reads; dredge does not exist yet, so the memory store has no such class and the lint has no rule for it. Wanted, when itd-25 is planned: the class in the closed enum, the writer in the dredge path, and MS001/MS002 reading it as a cross-class input.
