---
schema_version: 1
id: "iss-2610262042473980"
slug: "ingest-means-importing-shared"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1093 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-126-a-team-shares-one-bibliography-without.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-126, itd-2609292108089653]
finding_ref: cfk-64b64d7d34d68ef0
---

Ingest means importing shared bibliography entries in itd-126 and storing a source's text in itd-2609292108089653

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-126-a-team-shares-one-bibliography-without.md:17` — "`abcd source ingest` imports the repo's shared entries into the local corpus."
- End B: `.abcd/development/intents/drafts/itd-2609292108089653-ingesting-a-source-and-consulting-the.md:21` — "Ingesting a source and consulting the sources corpus are abcd verbs"

itd-126's abcd source ingest imports the repo's shared citation entries into the local corpus. itd-2609292108089653 uses ingesting a source for adding a source with its stored text, refused when that text is a keyword stub. One verb word names two different operations on the same corpus.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1093.
