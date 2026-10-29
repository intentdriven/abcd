---
schema_version: 1
id: "iss-2610262045183188"
slug: "itd-42-defers-full-body-comparison-of"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1265 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-42-coherence-aware-grill.md:66"
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): Who owns loading and comparing full intent bodies at scale? Answer 1: itd-39 widens its scope from memory pages to intent bodies. Answer 2: nobody for now, and both the itd-42 deferral and itd-39's Deferred-to-by line are dropped. Recommend 2, because itd-39's whole design is keyword recall of memory pages and a pre-pass that reads an index already ships."
related_intents: [itd-42, itd-39]
finding_ref: cfk-066f49d040b2cb15
---

itd-42 defers full-body comparison of intents to itd-39; itd-39 recalls keyword-matched memory pages and compares no intent bodies

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-42-coherence-aware-grill.md:66` — "Loading and comparing full intent bodies at scale is the scope-aware retrieval problem; deferred to"
- End B: `.abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:35` — "it loads keyword-matched, budget-bracketed pages only."

itd-42 says loading and comparing full intent bodies at scale is the scope-aware retrieval problem and defers it to itd-39. itd-39's scope is keyword recall of memory pages into prompts, loading keyword-matched, budget-bracketed pages only. Nothing in it loads or compares intent bodies, so the deferral points at an intent that does not cover it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1265.
