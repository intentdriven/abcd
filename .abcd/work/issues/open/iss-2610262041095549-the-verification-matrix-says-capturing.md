---
schema_version: 1
id: "iss-2610262041095549"
slug: "the-verification-matrix-says-capturing"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 980 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md:53"
remedy: "Re-base itd-30 on today's capture: bare quoted text writes a seeded draft with no interview, so a fiction format would change the seed skeleton and the planning interview, not a headline, body and quote interview behind /abcd:intent new."
related_intents: [itd-30]
finding_ref: cfk-1bcfd2c5fdf71a04
---

The verification matrix says capturing an intent runs no interview; itd-30 assumes capture runs a headline/body/quote interview that it reshapes.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/02-verification-matrix.md:53` — "Bare quoted text runs no interview"
- End B: `.abcd/development/intents/drafts/itd-30-design-fictions-as-intent-format.md:42` — "prompts shift from "headline + press-release body + customer quote""

The matrix's Intent capture row says bare quoted text runs no interview and writes a draft with seeded prompts. itd-30 is designed as a change to an existing capture interview, moving its prompts from headline, body and customer quote to fiction prompts. The interview it reshapes does not exist on the brief's account.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 980.
