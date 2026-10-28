---
schema_version: 1
id: "iss-2610262032131693"
slug: "itd-48-says-abcd-intent-shape-is-now-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 238 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:24"
remedy: "none (filed automatically)"
related_intents: [itd-48]
finding_ref: cfk-d3a75b2d2c5e4052
---

itd-48 says `/abcd:intent shape` is now a working command surface; the brief's press release says it is designed and not yet built

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:24` — "With all three roles live, `/abcd:intent consistency` and `/abcd:intent shape` move from documented command surface to working command surface."
- End B: `.abcd/development/brief/01-product/01-press-release.md:26` — "Two companions are designed and not yet built: `/abcd:intent grill` (per itd-27), a Socratic interview that stress-tests an intent, or a brief section, before it is planned; and `/abcd:intent shape` (Role 3), which keeps each intent's `kind` honest as the corpus grows."

itd-48's press release says that with all three roles live, `/abcd:intent consistency` and `/abcd:intent shape` become working command surface. The brief's press release lists `/abcd:intent shape` (Role 3) among the companions that are designed and not yet built. The shape surface cannot be both working and unbuilt.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 238.
