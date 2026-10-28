---
schema_version: 1
id: "iss-2610262034302701"
slug: "itd-61-reuses-the-existing-grill-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 501 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-61-brief-change-derivation.md:40"
remedy: "none (filed automatically)"
related_intents: [itd-61]
finding_ref: cfk-8aefee6c83b24da4
---

itd-61 reuses the existing grill's interrogation machinery; the intent chapter says the grill sub-verb is an unshipped design target.

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-61-brief-change-derivation.md:40` — "Reuse of the existing grill's interrogation machinery rather than a parallel re-implementation."
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:450` — "the grill sub-verb that writes these fields and the PRD at `.abcd/intents/<itd-N>/prd.md` are a design target"

itd-61's derivation pass depends on reusing the grill's interrogation machinery instead of building its own. 05-intent.md says the grill sub-verb is a design target that has not shipped, and that its intent was superseded. As written, the dependency names machinery the intent chapter says was never built.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 501.
