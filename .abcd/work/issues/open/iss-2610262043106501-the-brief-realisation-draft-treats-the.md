---
schema_version: 1
id: "iss-2610262043106501"
slug: "the-brief-realisation-draft-treats-the"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1120 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-2610040817105016-a-product-thinker-reads-their-brief-as.md:19"
remedy: "none (filed automatically)"
related_intents: [itd-2610040817105016, itd-147]
finding_ref: cfk-a321c6e560e493e7
---

The brief-realisation draft treats the brief as the target design in the present tense; itd-147 sequences the brief as the shipped state.

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-2610040817105016-a-product-thinker-reads-their-brief-as.md:19` — "A product thinker reads their brief as the target design, in the present tense"
- End B: `.abcd/development/intents/shipped/itd-147-the-brief-s-surface-chapters-are-a.md:99` — "the brief is the shipped state, with itd-60 as the"

itd-2610040817105016 records the product thinker's ruling that a product thinker reads the brief 'as the target design, in the present tense', with intents shown as realising each section. itd-147 is a delivery rung of the phase whose rule is that 'the brief is the shipped state', generating each chapter's surface block from the shipped command tree and marking an unbuilt surface as unbuilt. A brief written as target design in the present tense is exactly the drift itd-147's generated blocks and currency rule exist to stop, so a builder cannot satisfy both definitions of what the brief is.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1120.
