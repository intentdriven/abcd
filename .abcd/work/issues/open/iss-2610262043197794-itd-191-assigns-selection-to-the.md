---
schema_version: 1
id: "iss-2610262043197794"
slug: "itd-191-assigns-selection-to-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1130 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-191-the-selection-criteria-are-a-declared.md:20"
remedy: "none (filed automatically)"
related_intents: [itd-191, itd-2609212137129937]
finding_ref: cfk-509be2a63c2576a2
---

itd-191 assigns selection to 'the maintainer', a word itd-2609212137129937 removed in favour of naming the product thinker or the facilitator

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-191-the-selection-criteria-are-a-declared.md:20` — "combining characterisations into a choice is the maintainer's act"
- End B: `.abcd/development/intents/shipped/itd-2609212137129937-abcd-s-own-text-names-the-product.md:21` — "the word maintainer leaves the vocabulary"

The discipline says combining characterisations into a choice is the maintainer's act, while the shipped intent rules that the word maintainer leaves the vocabulary and that every use names the product thinker or the technical facilitator. The standing selection rule therefore leaves unstated which of the two people makes the choice, which is the ambiguity the shipped intent exists to remove.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1130.
