---
schema_version: 1
id: "iss-2610262035075414"
slug: "itd-42-is-scoped-as-a-capability-of-an"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 548 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-42-coherence-aware-grill.md:71"
remedy: "none (filed automatically)"
related_intents: [itd-42]
finding_ref: cfk-ae0ebb823223711d
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262032113433. Same A line and same root cause; 05-intent.md names the pre-pass as what delivers itd-42."
duplicates: [iss-2610262032113433]
---

itd-42 is scoped as a capability of an existing /abcd:intent grill sub-verb; the intent chapter says no grill sub-verb ships

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-42-coherence-aware-grill.md:71` — "this is a capability of the existing `/abcd:intent grill`, not a sibling verb."
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:319` — "(per itd-27, `intents/superseded/`, superseded by itd-94; no grill sub-verb ships)"

itd-42 rules out a new sub-verb and builds the coherence tier into the existing /abcd:intent grill. The intent chapter says the grill's intent, itd-27, is superseded and that no grill sub-verb ships. The verb itd-42 depends on therefore does not exist, and the chapter instead lists a separate pre-pass sub-verb as the thing that delivers itd-42.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 548.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262032113433. Same A line and same root cause; 05-intent.md names the pre-pass as what delivers itd-42.
