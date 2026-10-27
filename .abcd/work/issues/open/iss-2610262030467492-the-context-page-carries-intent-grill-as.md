---
schema_version: 1
id: "iss-2610262030467492"
slug: "the-context-page-carries-intent-grill-as"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 43 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/02-context.md:15"
remedy: "none (filed automatically)"
finding_ref: cfk-24bf4c0612e5ed9c
---

The context page carries intent grill as designed-not-built per itd-27; the intent chapter says its intent is superseded

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/02-context.md:15` — "Designed, not built: the `intent grill` sub-verb"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:450` — "its intent itd-27 is in `intents/superseded/`, superseded by itd-94"

02-context sets out the intent grill sub-verb as designed and not built, citing itd-27, while 05-intent says itd-27 is in intents/superseded/, superseded by itd-94. The context page presents as pending a design the intent chapter records as retired.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 43.
