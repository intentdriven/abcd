---
schema_version: 1
id: "iss-2610262044561718"
slug: "itd-39-reuses-the-rules-loader-s-context"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1240 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:75"
remedy: "none (filed automatically)"
related_intents: [itd-39, itd-3]
finding_ref: cfk-eedf238a7825999e
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262039330569. Same End A (itd-39 line 75 credits context brackets to the rules loader); itd-3 never had them, as the other record shows from the configuration chapter."
duplicates: [iss-2610262039330569]
---

itd-39 reuses the rules loader's context brackets from itd-3, but itd-3's loader defines no brackets

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:75` — "Memory recall uses the rules loader's context brackets (itd-3)"
- End B: `.abcd/development/intents/shipped/itd-3-modular-rules-loader.md:30` — "keyword-matches the `prompt` against each domain's `recall` list, injects matching rules as a system message"

itd-39 rules that memory recall shares the rules loader's context brackets (itd-3), with no thresholds tuned separately for memory. itd-3's hook keyword-matches the prompt against each domain's recall list and injects every matching rule; its scope has no gating by remaining context budget and no FRESH, MODERATE or DEPLETED bracket. The dependency points at a mechanism the intent it names never provides, so either itd-39 must build the brackets itself or another record must supply them.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1240.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262039330569. Same End A (itd-39 line 75 credits context brackets to the rules loader); itd-3 never had them, as the other record shows from the configuration chapter.
