---
schema_version: 1
id: "iss-2610262031524510"
slug: "the-personas-page-forbids-generic"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 189 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/05-personas.md:5"
remedy: "none (filed automatically)"
related_intents: [itd-2609221009495079]
finding_ref: cfk-01ae78f50d7e3022
---

The personas page forbids generic, unnamed quote attributions, but itd-2609221009495079's quote is credited to 'a product thinker'.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/05-personas.md:5` — "never use generic "a hypothetical user" phrasing (loses voice)"
- End B: `.abcd/development/intents/planned/itd-2609221009495079-abcd-s-typed-judgements-can-run-through.md:24` — "said a product thinker reading a run's cost"

05-personas codifies that press-release quotes use named placeholders from the registry, picked by role, and never generic phrasing, because that loses voice. itd-2609221009495079's press-release quote is credited to 'a product thinker' with no registered name.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 189.
