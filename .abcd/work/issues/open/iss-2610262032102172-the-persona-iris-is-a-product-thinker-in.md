---
schema_version: 1
id: "iss-2610262032102172"
slug: "the-persona-iris-is-a-product-thinker-in"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 231 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:7"
remedy: "In the shipped intent's customer quote replace 'Iris, a technical facilitator' with 'Nia, a facilitator' (the registered facilitator persona); the receipt digest covers only the Acceptance Criteria body, so the quote edit leaves the audit receipt valid."
related_intents: [itd-2609150819445595]
finding_ref: cfk-d7a3fa58a2d50a30
---

The persona Iris is a product thinker in the brief's press release and a technical facilitator in itd-2609150819445595.

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:7` — "said Iris, a product thinker"
- End B: `.abcd/development/intents/shipped/itd-2609150819445595-a-debt-nothing-lists-owed-fidelity.md:22` — "said Iris, a technical facilitator paying down a run's review debt"

The brief's press release voices Iris as a product thinker, and the persona rule ties each registered name to one role. itd-2609150819445595 attributes its quote to Iris as a technical facilitator. One name is used for two personas.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 231.
