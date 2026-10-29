---
schema_version: 1
id: "iss-2610262041388352"
slug: "the-persona-glossary-requires-a"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1014 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/persona.md:25"
remedy: "In persona.md 'When to use', scope the rule to the voice of a modelled user and say a real role abcd addresses (the product thinker) may be quoted by role, unnamed, as 19 intents already do."
related_intents: [itd-82]
finding_ref: cfk-d066b0479b25f2cc
---

The persona glossary requires a registered persona for any press-release quote; itd-82 quotes an unnamed product thinker instead

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/persona.md:25` — "Use a persona whenever a press release, intent document, or design fiction needs a human voice or"
- End B: `.abcd/development/intents/planned/itd-82-drain-ledger-triage.md:22` — "said a product thinker who ran the first ledger drains by hand"

The persona entry says to use a persona from the registry whenever a press release needs a human voice or quote, and never to invent one. The product-thinker entry says the product thinker is not a persona. itd-82's press release attributes its quote to "a product thinker who ran the first ledger drains by hand": a role, not a registered persona, used as exactly the voice the glossary reserves for personas.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1014.
