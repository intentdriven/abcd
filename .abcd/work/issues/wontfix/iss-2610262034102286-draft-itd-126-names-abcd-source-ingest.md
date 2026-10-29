---
schema_version: 1
id: "iss-2610262034102286"
slug: "draft-itd-126-names-abcd-source-ingest"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 475 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-126-a-team-shares-one-bibliography-without.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-126]
finding_ref: cfk-7935a787f95abfa1
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031350187. Same draft edit as iss-2610262031350187, set against a different statement of the same current state."
duplicates: [iss-2610262031350187]
---

draft itd-126 names `abcd source ingest` for importing shared references into the corpus; the library surface retired `source` and keeps ingest for the memory store, which registers nothing

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-126-a-team-shares-one-bibliography-without.md:17` — "`abcd source ingest` imports the repo's shared entries into the local corpus."
- End B: `.abcd/development/brief/04-surfaces/07-library.md:23` — "that the memory store's ingest registers nothing, because the retired page that"

itd-126 has an ingest verb under source import the repository's shared bibliography entries into the local sources corpus. The library surface merged source into library and says the memory store's ingest registers nothing, the corpus's add being what registers a document, because a retired page that registered a document carried the name ingest. itd-126 would reintroduce exactly that collision.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 475.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031350187. Same draft edit as iss-2610262031350187, set against a different statement of the same current state.
