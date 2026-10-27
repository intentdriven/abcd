---
schema_version: 1
id: "iss-2610262033072029"
slug: "the-embark-chapter-still-calls-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 363 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/03-embark.md:248"
remedy: "none (filed automatically)"
related_intents: [itd-125]
finding_ref: cfk-bc73b0ee06202284
---

The embark chapter still calls the judgement over a lifeboat an 'oracle product audit'; itd-125 renamed it a review everywhere

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/03-embark.md:248` — "would re-run the oracle product audit against the"
- End B: `.abcd/development/intents/shipped/itd-125-the-lifeboat-verdict-says-what-it-is.md:22` — "produced by an oracle, holding a review verdict — three words for one thing."

itd-125 says the judgement over a packed lifeboat is a review (`abcd disembark review`, returning SHIP, NEEDS_WORK or MAJOR_RETHINK). It says the old wording, an audit produced by an oracle holding a review verdict, was three words for one thing and is now one word in the verb, the flag, the artefact and the prose. The embark chapter's refresh-audit modifier would re-run the oracle product audit and report drift against the disembark-time audit, and its probe 'runs no product audit'. That is the retired wording, still used for the same judgement.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 363.
