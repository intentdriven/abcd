---
schema_version: 1
id: "iss-2610262031230059"
slug: "04-naming-retires-the-logbook-name-and"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 123 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:84"
remedy: "none (filed automatically)"
finding_ref: cfk-bd989554b4cdbe62
---

04-naming retires the logbook name and forbids re-minting it; the lifeboat glossary entry lists a logbook summary among a lifeboat's contents.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:84` — "It is retired and must not be re-minted"
- End B: `.abcd/development/brief/glossary/core/lifeboat.md:21` — "logbook summary, and configuration into a self-contained directory elsewhere"

04-naming says .abcd/logbook/ was the maritime name for per-run logs and 'It is retired and must not be re-minted'. The glossary's lifeboat entry says disembark packs the grounded brief, a coverage report, a graveyard, key decisions, 'logbook summary, and configuration into a self-contained directory elsewhere'. The canonical definition of the artefact names one of its parts after the retired word.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 123.
