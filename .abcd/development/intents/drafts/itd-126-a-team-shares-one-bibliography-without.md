---
id: itd-126
slug: a-team-shares-one-bibliography-without
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

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [sources-provenance SOTA note](../../research/notes/2026-10-23-sources-provenance-sota.md).

- **Provenance has a sanctioned home.** CSL-JSON forbids unknown top-level keys and reserves `custom` for "additional information that does not have a designated CSL JSON field" (CSL-JSON input schema), so who shared an entry, from which repository and when ride in `custom` and render in no style.
- **Conflicts are decided at item level in the field.** Zotero sync asks "which version you'd like to keep" when one item changed in two places and merges non-conflicting edits automatically; its duplicate merge is the field-level form, a person picking a master item and each surviving field (Zotero sync and duplicate-detection documentation).
- **Keys are owned by the store, not recomputed.** Better BibTeX generates keys deterministically against the keys already present and resolves a clash with a letter postfix; a changed pattern touches only later items.
- **A committed JSON array merges badly in git.** Sorting entries by `id` and refusing a share whose key exists with different content moves the conflict from git's textual merge to a refusal that names the key.
- **The team-library services share the wrong thing.** Zotero groups hold separate copies of items and attachments against one owner's quota; a corpus-free share needs the committed file, as the maintainer already ruled.
- **Path 2**: native `share` and `ingest` over `references.csl.json`, provenance in a `custom.shared` block (repository handle, source repo, date, corpus key; never a person's name or mail), first-sharer key ownership with a field-level diff on refusal and `--replace` as the deliberate overwrite. The entry stays pure CSL-JSON, so a later Zotero or CFF export is a projection. No Go dependency. The independent fit-challenge runs at plan time.

## Open Questions

- Conflict shape when two teammates share the same source with divergent metadata — last-write, merge, or key-ownership. (Moved here from itd-76 at its planning; deliberately unresolved.)
- Whether ingest should mark provenance on imported entries (who shared, from which repo) so a team bibliography stays auditable. (Moved here from itd-76.)
- The references store carries hand-curated entries (the acknowledgements baseline) and will carry shared ones; whether share appends blindly or respects the store's admission criteria (verified metadata, resolver links) needs a ruling before planning.
- (SOTA 2026-10-23) Does a shared entry that fails the baseline's admission criteria (no resolver link, unverified authors) enter marked `custom.shared.verified: false`, or is the share refused until the sharer completes it?
- (SOTA 2026-10-23) What identifies the sharer in `custom.shared.by` without a personal name or mail reaching a committed file: the repository's handle, the machine's `identity` block, or nothing beyond the source repository?

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
