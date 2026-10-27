---
schema_version: 1
id: "iss-2610262034081412"
slug: "the-disposition-glossary-keeps"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 473 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/ledger/disposition.md:25"
remedy: "none (filed automatically)"
finding_ref: cfk-2b6fc77cc48c6c3e
---

the disposition glossary keeps disposition vocabulary apart from the issue ledger's wontfix; the capture surface records a wontfix under the token declined, a disposition state

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/ledger/disposition.md:25` — "Do not reuse the issue ledger's open, resolved and wontfix."
- End B: `.abcd/development/brief/04-surfaces/06-capture.md:344` — "the token stays `declined`, because a wontfix **is** that non-action"

The glossary gives a disposition four states, declined among them, and says not to reuse the issue ledger's open, resolved and wontfix. The capture surface writes every wontfix's grounds under the token declined, because a wontfix is that non-action. So declined names both a reading-item disposition state and an issue wontfix, the crossing of the two vocabularies the glossary rules out.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 473.
