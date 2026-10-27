---
schema_version: 1
id: "iss-2610262045286894"
slug: "itd-3-names-a-roadmap-rule-domain-and"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1276 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-3-modular-rules-loader.md:34"
remedy: "none (filed automatically)"
related_intents: [itd-3, itd-2609211913453478]
finding_ref: cfk-a372e12fbb8d8e4a
---

itd-3 names a ROADMAP rule domain and teaches 'draft a milestone'; itd-2609211913453478 retires roadmap and milestone as superseded terms

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-3-modular-rules-loader.md:34` — "(e.g. `*ROADMAP draft a milestone`)"
- End B: `.abcd/development/intents/shipped/itd-2609211913453478-one-page-in-the-glossary-maps-abcd-s.md:40` — "`phase`, `milestone` and `roadmap` marked `status: superseded`"

itd-3's example star-command activates a ROADMAP domain to draft a milestone. itd-2609211913453478 marks phase, milestone and roadmap superseded, with successors named, so a rules domain and its taught example still use two words the glossary now says have left the vocabulary.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1276.
