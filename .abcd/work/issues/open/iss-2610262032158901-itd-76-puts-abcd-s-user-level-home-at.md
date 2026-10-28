---
schema_version: 1
id: "iss-2610262032158901"
slug: "itd-76-puts-abcd-s-user-level-home-at"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 244 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-76-source-provenance-ledger.md:18"
remedy: "none (filed automatically)"
related_intents: [itd-76]
finding_ref: cfk-086b9bed04a91155
---

itd-76 puts abcd's user-level home at `~/.abcd/`; the invariants register gives the home as `~/.abcd.noindex`

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-76-source-provenance-ledger.md:18` — "abcd's user-level home, `~/.abcd/sources/` by default, itself a no-remote git repository"
- End B: `.abcd/development/brief/02-constraints/03-invariants.md:179` — "a name the indexer honours per folder: the `.noindex` suffix the home carries, `~/.abcd.noindex`"

itd-76 places the local-only corpus in 'abcd's user-level home, `~/.abcd/sources/` by default'. Invariant 21 says the home carries the `.noindex` suffix, `~/.abcd.noindex`. Each names a different directory as the single user-level home, so the corpus's default location is not settled.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 244.
