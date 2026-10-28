---
schema_version: 1
id: "iss-2610262034045777"
slug: "draft-itd-175-promises-the-product"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 468 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-175-the-product-thinker-writes-down-how-this.md:12"
remedy: "none (filed automatically)"
related_intents: [itd-175]
finding_ref: cfk-5064ad9e362bb5de
---

draft itd-175 promises the product thinker writes down how an intent could be wrong; the readiness gate's grounds already record what would show the conjecture wrong

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-175-the-product-thinker-writes-down-how-this.md:12` — "The product thinker writes down how this could be wrong, and what would show it"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:325` — "it appends the conjecture behind this decision — what is expected, and what would show it wrong — to the intent's `## Grounds` section"

itd-175 claims, as a draft, that the product thinker writes down how this could be wrong and what would show it. The intent surface says the readiness gate's grounds append the conjecture behind each decision, what is expected and what would show it wrong, to the intent's Grounds section. Both documents claim the same recorded falsifier, one as shipped and one as yet to be built.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 468.
