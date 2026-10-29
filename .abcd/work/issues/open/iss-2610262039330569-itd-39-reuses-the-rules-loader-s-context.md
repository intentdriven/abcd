---
schema_version: 1
id: "iss-2610262039330569"
slug: "itd-39-reuses-the-rules-loader-s-context"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 868 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:75"
remedy: "In itd-39's decision of 2026-10-11 'Share the brackets' and the Context brackets bullet, say the bundle builds the context brackets once in the loader; itd-3's loader has none."
related_intents: [itd-39]
finding_ref: cfk-23fff70b6d6ce91c
---

itd-39 reuses the rules loader's context brackets (itd-3); the configuration chapter's loader has no context brackets and attributes the budget bracket to itd-39 itself

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:75` — "Memory recall uses the rules loader's context brackets (itd-3); there are no thresholds tuned separately for memory."
- End B: `.abcd/development/brief/05-internals/03-configuration.md:1174` — "but keyword recall with a budget bracket (itd-39)"

itd-39 rules that memory recall uses the rules loader's context brackets, with no thresholds of its own. The configuration chapter's account of the loader and the prompt router has a per-session dedup and a fixed 64 KiB injection budget, and no bracket tied to remaining context. Its one mention of a bracket credits it to itd-39 ('keyword recall with a budget bracket (itd-39)'). Each document assigns the brackets to the other, so neither one builds what itd-39 depends on.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 868.
