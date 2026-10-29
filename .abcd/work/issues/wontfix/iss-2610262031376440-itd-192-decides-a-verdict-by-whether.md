---
schema_version: 1
id: "iss-2610262031376440"
slug: "itd-192-decides-a-verdict-by-whether"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 154 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-192-an-acceptance-criterion-whose-producer.md:16"
remedy: "none (filed automatically)"
related_intents: [itd-192]
finding_ref: cfk-4257d7de1dabea94
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262042302710. Same unit problem in itd-192 as iss-2610262042302710, cited against the scope chapter."
duplicates: [iss-2610262042302710]
---

itd-192 decides a verdict by whether this phase wires a criterion, while the scope chapter makes phases history

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-192-an-acceptance-criterion-whose-producer.md:16` — "is judged by whether this phase wires it, not by whether the narrowing was disclosed"
- End B: `.abcd/development/brief/01-product/04-scope.md:3` — "The phase documents under `roadmap/phases/` are history."

itd-192's one test is whether this phase wires the criterion: wired in this phase is MET_WITH_CONCERNS, not wired is NOT_MET. The scope chapter says no stored unit sits above the intent for sequence and the phase documents are history. The rule's deciding unit is retired vocabulary, so an auditor cannot tell which delivery the question is asked of.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 154.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262042302710. Same unit problem in itd-192 as iss-2610262042302710, cited against the scope chapter.
