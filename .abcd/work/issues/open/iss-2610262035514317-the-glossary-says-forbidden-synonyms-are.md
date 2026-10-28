---
schema_version: 1
id: "iss-2610262035514317"
slug: "the-glossary-says-forbidden-synonyms-are"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 603 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/README.md:7"
remedy: "none (filed automatically)"
finding_ref: cfk-07bad166e87eb26a
---

The glossary says forbidden synonyms are registered there and nowhere else; the docs chapter registers the retired role word as a docs-lint banned token

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/README.md:7` — "with their aliases and forbidden synonyms — is registered here and nowhere else"
- End B: `.abcd/development/brief/04-surfaces/10-docs.md:178` — "The `roles/retired-role-word` banned token refuses the one word that blurred them"

The glossary README says cross-cutting vocabulary, its aliases and its forbidden synonyms are registered in the glossary and nowhere else, and its product-thinker and technical-facilitator entries declare no forbidden synonyms. The docs chapter describes the word that blurred those two roles being refused by the `roles/retired-role-word` banned token in the docs-lint configuration, as a blocker, over roots that include the brief. So a forbidden vocabulary word is registered and enforced outside the glossary.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 603.
