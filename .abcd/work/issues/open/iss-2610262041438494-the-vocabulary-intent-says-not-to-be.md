---
schema_version: 1
id: "iss-2610262041438494"
slug: "the-vocabulary-intent-says-not-to-be"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1020 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/README.md:59"
remedy: "none (filed automatically)"
related_intents: [itd-2609211913453478]
finding_ref: cfk-c9dc76a86a92756c
---

The vocabulary intent says not_to_be_confused_with may name only families; the glossary README requires only that at least one member name a family.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/README.md:59` — "names at least one family on the record-families page, or the page itself, on every entry but that page."
- End B: `.abcd/development/intents/shipped/itd-2609211913453478-one-page-in-the-glossary-maps-abcd-s.md:39` — "every other glossary entry's `not_to_be_confused_with` may name only a family on the page"

The intent says every other glossary entry's not_to_be_confused_with 'may name only a family on the page'. The glossary README's constraint requires only that at least one member names a family row or the page itself. The live entries follow the README: oracle.md, for example, names core/transport, which is not a family, beside core/record-families. One rule is 'only families' and the other is 'at least one family', so both cannot be the rule.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1020.
