---
schema_version: 1
id: "iss-2610262040034904"
slug: "the-surfaces-register-says-the-intent"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 905 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md:395"
remedy: "none (filed automatically)"
related_intents: [itd-42]
finding_ref: cfk-bb32a3bf8f9b2685
---

The surfaces register says the intent verb ships no grill sub-verb; shipped itd-42 scopes itself as a capability of an existing /abcd:intent grill

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/README.md:395` — "per-session output are command-shaped, and the `intent` parent ships no `grill`"
- End B: `.abcd/development/intents/shipped/itd-42-coherence-aware-grill.md:71` — "this is a capability of the existing `/abcd:intent grill`, not a sibling verb."

The README states that grill's design target is promotion to a sub-verb of /abcd:intent and that the intent parent ships no grill sub-verb. itd-42, in shipped/, scopes its coherence tier as a capability of the existing /abcd:intent grill, with renamed flags, and rules out a sibling verb. The grill sub-verb cannot both exist for itd-42 to extend and be unshipped.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 905.
