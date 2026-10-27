---
schema_version: 1
id: "iss-2610262042079651"
slug: "itd-42-extends-an-existing-abcd-intent"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1047 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-42-coherence-aware-grill.md:71"
remedy: "none (filed automatically)"
related_intents: [itd-42]
finding_ref: cfk-3108687adb78e47e
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262032113433. Same A line and same root cause; the build sequence states there is no grill sub-verb, as the other two chapters do."
duplicates: [iss-2610262032113433]
---

itd-42 extends an existing `/abcd:intent grill` sub-verb; the build sequence says intent has no grill sub-verb at all.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-42-coherence-aware-grill.md:71` — "this is a capability of the existing `/abcd:intent grill`, not a sibling verb"
- End B: `.abcd/development/brief/06-delivery/01-build-sequence.md:99` — "There is no `grill` sub-verb either: the grilling itd-27 asked for is the planning interview"

itd-42 refuses a new verb because its coherence tier is a capability of the existing `/abcd:intent grill`, and renames that sub-verb's flags (`--glossary`, `--coherence`, `--full`, `--light`). The build sequence states there is no grill sub-verb and that the grilling itd-27 asked for is the planning interview. A shipped intent cannot extend a sub-verb the brief says does not exist. itd-42's own Decisions moved the work into the planning interview's pre-pass, but its scope text still describes the grill sub-verb.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1047.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262032113433. Same A line and same root cause; the build sequence states there is no grill sub-verb, as the other two chapters do.
