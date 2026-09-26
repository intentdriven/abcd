---
id: itd-126
slug: a-team-shares-one-bibliography-without-sharing-anyone-s-corp
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-76]
severity: minor
impact: additive
---

# A Team Shares One Bibliography Without Sharing Anyone's Corpus

## Press Release

> **Citation data travels through the repo; corpora never do.** Alice works from a personal sources corpus ([itd-76](../shipped/itd-76-source-provenance-ledger.md), which this intent `refines`); their teammate Bob has their own, or none. `abcd source share` writes a public source's *citation data* into the repo's committed CSL-JSON references store — `.abcd/development/research/references.csl.json`, the store the acknowledgements list already reads (maintainer ruling, 2026-08-16: one committed bibliography, not a second exchange file) — and refuses any `confidential: true` entry mechanically. `abcd source ingest` imports the repo's shared entries into the local corpus. Documents and influence ledgers never travel — only bibliography.
>
> "I just ingest the repo's shared references," said Bob, "and every public source Alice worked from is in my own local store, ready to consult."

## Why This Matters

A repo is a team surface even when every corpus is personal. Without a mechanical share path, the public slice of a bibliography is copied by hand and drifts; with one, a whole team builds on one reference list while each member's corpus — and everything confidential in it — stays private by construction. The refusal of confidential entries at the share boundary is the [adr-41](../../decisions/adrs/0041-corpus-trust-boundary.md) trust rule exercised at the team seam.

## Acceptance Criteria

> _Seeded by the itd-76 planning session, unconfirmed — the planning interview must walk these with the maintainer._

- Given a public entry in Alice's corpus, when they run `abcd source share`, then its citation data is written into `.abcd/development/research/references.csl.json` and nothing else — no document, no ledger line — enters the repo.
- Given a `confidential: true` entry, when share is attempted, then the refusal is mechanical and names the gate, not the source's identifying strings.
- Given shared entries in the repo, when Bob runs `abcd source ingest`, then they enter their local corpus as public entries.

## Open Questions

- Conflict shape when two teammates share the same source with divergent metadata — last-write, merge, or key-ownership. (Moved here from itd-76 at its planning; deliberately unresolved.)
- Whether ingest should mark provenance on imported entries (who shared, from which repo) so a team bibliography stays auditable. (Moved here from itd-76.)
- The references store carries hand-curated entries (the acknowledgements baseline) and will carry shared ones; whether share appends blindly or respects the store's admission criteria (verified metadata, resolver links) needs a ruling before planning.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
