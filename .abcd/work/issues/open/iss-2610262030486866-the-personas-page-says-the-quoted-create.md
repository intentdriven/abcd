---
schema_version: 1
id: "iss-2610262030486866"
slug: "the-personas-page-says-the-quoted-create"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 46 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/05-personas.md:7"
remedy: "none (filed automatically)"
finding_ref: cfk-3cc13edd1f7ac842
---

The personas page says the quoted create picks a persona for the customer quote; the intent chapter says the create writes the quoted text as the press release

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/05-personas.md:7` — "`/abcd:intent "<text>"` picks a persona for the customer quote in each press release"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:189` — "a body whose `## Press Release` IS the quoted text, as prose"

05-personas says /abcd:intent "< text>" picks a persona for the customer quote in each press release and points to 05-intent § 4 for it, while 05-intent says the create's press release is the quoted text as prose, and its § 4 describes only a lint resolving attributions against personas.json. Nothing in the create picks a persona.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 46.
