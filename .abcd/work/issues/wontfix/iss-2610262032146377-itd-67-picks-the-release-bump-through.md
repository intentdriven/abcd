---
schema_version: 1
id: "iss-2610262032146377"
slug: "itd-67-picks-the-release-bump-through"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 241 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:47"
remedy: "none (filed automatically)"
related_intents: [itd-67]
finding_ref: cfk-6fcc67bbca7458b1
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. DeriveNext in internal/core/changelog derives the tier from record impact and no phase input exists; the later closed spec names the supersession."
---

itd-67 picks the release bump through brief §4 phase-completion detection; the mental model computes the version from the impact of what shipped

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:47` — "`/abcd:launch ship` auto-selects the bump tier (patch/minor/major) via the brief §4 phase-completion detection"
- End B: `.abcd/development/brief/01-product/03-mental-model.md:3` — "is the checkpoint: its version is computed from the impact of what shipped"

itd-67 has `/abcd:launch ship` choose patch, minor or major 'via the brief §4 phase-completion detection'. 03-mental-model says the derived release's version is computed from the impact of what shipped, and the same page retires the phase. The record carries two incompatible rules for choosing the version.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 241.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. DeriveNext in internal/core/changelog derives the tier from record impact and no phase input exists; the later closed spec names the supersession.
