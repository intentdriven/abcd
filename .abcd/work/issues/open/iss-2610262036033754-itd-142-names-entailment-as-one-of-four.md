---
schema_version: 1
id: "iss-2610262036033754"
slug: "itd-142-names-entailment-as-one-of-four"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 620 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-142-the-brief-creation-interview-abcd.md:60"
remedy: "In itd-142 rename 'question regimes' to 'question kinds' (or similar): `regime` is the glossary's term for a reading's licence (generative, explicative, evaluative, registrative), and 'entailment' there is a position, so the two uses collide."
related_intents: [itd-142]
finding_ref: cfk-6d91d5f979534594
---

itd-142 names entailment as one of four question regimes; 23-reading makes entailment a position whose regime is explicative.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-142-the-brief-creation-interview-abcd.md:60` — "Four question regimes — narrative / options / entailment / defaults"
- End B: `.abcd/development/brief/04-surfaces/23-reading.md:194` — "that its articulation does not state? | `explicative` |"

itd-142 lists four question regimes: narrative, options, entailment and defaults. In 23-reading a regime is the supply regime a reading's definition declares, and the entailment position's row gives its regime as explicative, so entailment is a position and not a regime. The two documents use regime for different things and put entailment on opposite sides of the position/regime distinction.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 620.
